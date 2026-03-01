package scripting

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (r *Runner) parseArgs(line string) ([]string, error) {
	rdr := csv.NewReader(strings.NewReader(line))
	rdr.Comma = ' '
	rdr.TrimLeadingSpace = true
	parts, err := rdr.Read()
	if err != nil {
		return nil, err
	}
	return parts, nil
}

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

	if targetTool == nil {
		return fmt.Errorf("tool not found: %s", name)
	}

	if targetTool.InputSchema == nil {
		return r.call(ctx, name, nil)
	}

	schema, ok := targetTool.InputSchema.(map[string]any)
	if !ok {
		return r.call(ctx, name, nil)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return r.call(ctx, name, nil)
	}

	var propNames []string
	for k := range properties {
		propNames = append(propNames, k)
	}
	sort.Strings(propNames)

	toolArgs := make(map[string]any)
	for i, val := range args {
		if i >= len(propNames) {
			break
		}
		propName := propNames[i]
		propSchema, _ := properties[propName].(map[string]any)
		toolArgs[propName] = convertValue(val, propSchema)
	}

	return r.call(ctx, name, toolArgs)
}

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

func (r *Runner) executeRawCall(ctx context.Context, name string, args map[string]any) (map[string]any, string, error) {
	rawResponse, err := client.CallToolRaw(ctx, r.session, name, args)
	if err != nil {
		return nil, "", err
	}

	text := extractTextFromRaw(rawResponse)
	return rawResponse, text, nil
}

func (r *Runner) executeSDKCall(ctx context.Context, name string, args map[string]any) (map[string]any, string, error) {
	rawResponse, err := client.CallToolRaw(ctx, r.session, name, args)
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

func (r *Runner) updateState(rawResponse map[string]any, text string) {
	r.lastText = text
	r.lastRawMap = rawResponse
	respData, _ := json.MarshalIndent(rawResponse, "", "  ")
	r.lastResponse = string(respData)
	if r.Raw {
		fmt.Printf("Raw Response:\n%s\n", r.lastResponse)
	}
}
