package scripting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/internal/conformance"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (r *Runner) parseArgs(line string) ([]string, error) {
	var parts []string
	var current strings.Builder
	inQuotes := false
	var quoteChar rune
	runes := []rune(line)

	for i := 0; i < len(runes); i++ {
		char := runes[i]
		switch {
		// Inside double quotes \" and \\ are escapes; any other backslash stays
		// literal so paths like "C:\temp" survive. Single quotes are verbatim.
		case char == '\\' && inQuotes && quoteChar == '"' && i+1 < len(runes) &&
			(runes[i+1] == '"' || runes[i+1] == '\\'):
			i++
			current.WriteRune(runes[i])
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
	if inQuotes {
		return nil, fmt.Errorf("unterminated %c quote", quoteChar)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts, nil
}

// callToolPositional calls the tool with the given name and arguments.
func (r *Runner) callToolPositional(ctx context.Context, name string, args []string) error {
	tools, err := client.ListAllTools(ctx, r.session)
	if err != nil {
		return err
	}

	var targetTool *mcp.Tool
	for _, t := range tools {
		if t.Name == name {
			targetTool = t
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

	var outputSchema any
	if targetTool != nil {
		outputSchema = targetTool.OutputSchema
	}
	return r.call(ctx, name, toolArgs, outputSchema)
}

// call calls the tool with the given name and arguments. A result that a strict
// client would reject against outputSchema fails the call, even though this
// tester's own SDK would accept it.
func (r *Runner) call(ctx context.Context, name string, args map[string]any, outputSchema any) error {
	var rawResponse map[string]any
	var text string
	var err error

	switch {
	case r.taskMode == taskStart:
		// Only the handle: nothing to check against the output schema yet
		return r.startTask(ctx, name, args)
	case r.taskMode == taskCall:
		rawResponse, text, err = r.executeTaskCall(ctx, name, args)
	case r.Raw:
		rawResponse, text, err = r.executeRawCall(ctx, name, args)
	default:
		rawResponse, text, err = r.executeSDKCall(ctx, name, args)
	}

	if err != nil {
		return err
	}

	r.updateState(rawResponse, text)

	if isErr, ok := rawResponse["isError"].(bool); ok && isErr {
		return &client.ToolError{
			Message: text,
		}
	}

	if err := conformance.CheckToolResult(outputSchema, rawResponse); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

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
	// The SDK path sends the per-request metadata 2026-07-28 requires and runs
	// the multi round-trip middleware, which answers input requests through
	// the client's handlers (see Runner.Responder). --raw bypasses both.
	params := &mcp.CallToolParams{Name: name, Arguments: args}
	if r.logLevel != "" {
		params.Meta = client.WithLogLevel(params.Meta, r.logLevel)
	}
	params.SetProgressToken(fmt.Sprintf("script-progress-%s", name))
	result, err := r.session.CallTool(ctx, params)
	if err != nil {
		var wireErr *jsonrpc.Error
		if errors.As(err, &wireErr) {
			return nil, "", &client.RPCError{Code: wireErr.Code, Message: wireErr.Message, Data: wireErr.Data}
		}
		return nil, "", err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal result: %w", err)
	}
	var rawResponse map[string]any
	if err := json.Unmarshal(data, &rawResponse); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal result: %w", err)
	}

	text := r.processSDKResult(result)
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
			fmt.Fprintf(r.w(), "Response: %s\n", c.Text)
			textBuilder.WriteString(c.Text)
		case *mcp.ImageContent:
			fmt.Fprintf(r.w(), "Response: [Image data, size %d]\n", len(c.Data))
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
		fmt.Fprintf(r.w(), "Raw Response:\n%s\n", r.lastResponse)
	}
}
