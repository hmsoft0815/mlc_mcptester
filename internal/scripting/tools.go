package scripting

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (r *Runner) parseArgs(line string) ([]string, error) {
	var parts []string
	var current strings.Builder
	inQuotes := false
	var quoteChar rune

	for _, char := range line {
		switch {
		case (char == '"' || char == '\'') && !inQuotes:
			inQuotes = true
			quoteChar = char
		case char == quoteChar && inQuotes:
			inQuotes = false
		case char == ' ' && !inQuotes:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts, nil
}

// callToolPositional calls the tool with the given name and arguments.
func (r *Runner) callToolPositional(ctx context.Context, name string, args []string) error {
	tools, err := r.session.ListTools(ctx, nil)
	if err != nil {
		return err
	}

	var targetTool *mcp.Tool
	for i := range tools.Tools {
		if tools.Tools[i].Name == name {
			targetTool = tools.Tools[i]
			break
		}
	}

	var properties map[string]any
	if targetTool != nil && targetTool.InputSchema != nil {
		if schema, ok := targetTool.InputSchema.(map[string]any); ok {
			if props, ok := schema["properties"].(map[string]any); ok {
				properties = props
			}
		}
	}

	if properties == nil {
		properties = make(map[string]any)
	}

	var propNames []string
	for k := range properties {
		propNames = append(propNames, k)
	}
	sort.Strings(propNames)

	toolArgs := make(map[string]any)
	var positionalArgs []string

	// First pass: extract named arguments and collect positional ones
	for _, arg := range args {
		if strings.Contains(arg, ":") {
			parts := strings.SplitN(arg, ":", 2)
			key := parts[0]
			val := parts[1]
			if propSchema, ok := properties[key].(map[string]any); ok {
				toolArgs[key] = convertValue(val, propSchema)
				continue
			}
		}
		// If not a named arg OR the key doesn't exist, treat as positional
		positionalArgs = append(positionalArgs, arg)
	}

	// Second pass: fill remaining properties with positional arguments
	posIdx := 0
	for _, propName := range propNames {
		if _, alreadySet := toolArgs[propName]; alreadySet {
			continue
		}
		if posIdx < len(positionalArgs) {
			propSchema, _ := properties[propName].(map[string]any)
			toolArgs[propName] = convertValue(positionalArgs[posIdx], propSchema)
			posIdx++
		}
	}

	return r.call(ctx, name, toolArgs)
}

// convertValue converts a string value to the type specified in the schema.
func convertValue(val string, schema map[string]any) any {
	if schema == nil {
		return val
	}
	typeName, _ := schema["type"].(string)
	switch typeName {
	case "integer":
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	case "number":
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	case "boolean":
		if strings.ToLower(val) == "true" || val == "1" {
			return true
		}
		if strings.ToLower(val) == "false" || val == "0" {
			return false
		}
	case "object", "array":
		var result any
		if err := json.Unmarshal([]byte(val), &result); err == nil {
			return result
		}
	}
	return val
}

// call calls the tool with the given name and arguments.
func (r *Runner) call(ctx context.Context, name string, args map[string]any) error {
	var rawResponse map[string]any
	var text string
	var err error

	if r.Raw {
		rawResponse, text, err = r.executeRawCall(ctx, name, args)
	} else {
		rawResponse, text, err = r.executeSDKCall(ctx, name, args)
	}

	if err != nil {
		return err
	}

	r.updateState(rawResponse, text)
	return nil
}

// executeRawCall calls the tool with the given name and arguments using the raw call method.
func (r *Runner) executeRawCall(ctx context.Context, name string, args map[string]any) (map[string]any, string, error) {
	meta := map[string]any{"progressToken": fmt.Sprintf("script-progress-%s", name)}
	rawResponse, err := client.CallToolRaw(ctx, r.session, name, args, meta)
	if err != nil {
		return nil, "", err
	}

	text := extractTextFromRaw(rawResponse)
	return rawResponse, text, nil
}

// executeSDKCall calls the tool with the given name and arguments using the SDK call method.
func (r *Runner) executeSDKCall(ctx context.Context, name string, args map[string]any) (map[string]any, string, error) {
	meta := map[string]any{"progressToken": fmt.Sprintf("script-progress-%s", name)}
	rawResponse, err := client.CallToolRaw(ctx, r.session, name, args, meta)
	if err != nil {
		return nil, "", err
	}

	// Try to unmarshal into SDK result for backward compatibility if needed,
	// but we mainly need the text for the runner's state.
	var sdkResult mcp.CallToolResult
	data, _ := json.Marshal(rawResponse)
	_ = json.Unmarshal(data, &sdkResult)

	text := r.processSDKResult(&sdkResult)
	return rawResponse, text, nil
}

// extractTextFromRaw extracts the text from the raw response. (legacy)
func extractTextFromRaw(rawResponse map[string]any) string {
	var textBuilder strings.Builder
	if contents, ok := rawResponse["content"].([]any); ok {
		for _, c := range contents {
			if cm, ok := c.(map[string]any); ok {
				if t, ok := cm["text"].(string); ok {
					textBuilder.WriteString(t)
				}
			}
		}
	}
	return textBuilder.String()
}

// processSDKResult processes the SDK result and returns the text.
func (r *Runner) processSDKResult(result *mcp.CallToolResult) string {
	var textBuilder strings.Builder
	for _, content := range result.Content {
		switch c := content.(type) {
		case *mcp.TextContent:
			fmt.Printf("Response: %s\n", c.Text)
			textBuilder.WriteString(c.Text)
		case *mcp.ImageContent:
			fmt.Printf("Response: [Image data, size %d]\n", len(c.Data))
		}
	}
	return textBuilder.String()
}

// updateState updates the runner's state with the given raw response and text.
func (r *Runner) updateState(rawResponse map[string]any, text string) {
	r.lastText = text
	r.lastRawMap = rawResponse
	respData, _ := json.MarshalIndent(rawResponse, "", "  ")
	r.lastResponse = string(respData)
	if r.Raw {
		fmt.Printf("Raw Response:\n%s\n", r.lastResponse)
	}
}
