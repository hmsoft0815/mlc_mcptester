// Package scripting provides a simple script runner for testing MCP tools.
package scripting

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/mlc-mcptester/mcp-tester/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Runner manages the execution of MCP test scripts.
type Runner struct {
	session      *mcp.ClientSession
	lastResponse string // The full JSON response
	lastText     string // Just the text content
	lastRawMap   map[string]any
	Raw          bool
	variables    map[string]string
}

// NewRunner creates a new Runner with the given MCP client session.
func NewRunner(session *mcp.ClientSession, raw bool) *Runner {
	return &Runner{
		session:   session,
		Raw:       raw,
		variables: make(map[string]string),
	}
}

// Run executes a script string against the established MCP session.
func (r *Runner) Run(ctx context.Context, script string) error {
	lines := strings.Split(script, "\n")
	executed := 0
	passed := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Remove trailing comments
		if idx := strings.Index(line, " #"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		if idx := strings.Index(line, " //"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		executed++
		// 1. Variable Substitution ($VAR_NAME)
		line = r.replaceVariables(line)

		// 2. Command Parsing
		if strings.HasPrefix(line, "call_tool ") {
			if err := r.handleCallTool(ctx, i, line); err != nil {
				return err
			}
			passed++
		} else if strings.HasPrefix(line, "set_var ") {
			if err := r.handleSetVar(i, line); err != nil {
				return err
			}
			passed++
		} else if strings.HasPrefix(line, "input_var ") {
			if err := r.handleInputVar(i, line); err != nil {
				return err
			}
			passed++
		} else if strings.HasPrefix(line, "assert_contains ") {
			expected := strings.TrimPrefix(line, "assert_contains ")
			expected = strings.Trim(expected, "\"")
			if !strings.Contains(r.lastText, expected) && !strings.Contains(r.lastResponse, expected) {
				return fmt.Errorf("line %d: assertion failed: expected to contain %q", i+1, expected)
			}
			fmt.Printf("Assertion passed: contains %q\n", expected)
			passed++
		} else if strings.HasPrefix(line, "assert_equals ") {
			expected := strings.TrimPrefix(line, "assert_equals ")
			expected = strings.Trim(expected, "\"")
			if r.lastText != expected && r.lastResponse != expected {
				return fmt.Errorf("line %d: assertion failed: expected exactly %q, but got %q", i+1, expected, r.lastText)
			}
			fmt.Printf("Assertion passed: equals %q\n", expected)
			passed++
		} else if strings.HasPrefix(line, "assert_number ") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "assert_number "))
			if _, err := strconv.ParseFloat(val, 64); err != nil {
				return fmt.Errorf("line %d: assertion failed: %q is not a number", i+1, val)
			}
			fmt.Printf("Assertion passed: %q is a number\n", val)
			passed++
		} else if strings.HasPrefix(line, "assert_gt ") {
			parts := strings.Fields(strings.TrimPrefix(line, "assert_gt "))
			if len(parts) != 2 {
				return fmt.Errorf("line %d: assert_gt expects 2 arguments", i+1)
			}
			v1, err1 := strconv.ParseFloat(parts[0], 64)
			v2, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 != nil || err2 != nil {
				return fmt.Errorf("line %d: assert_gt arguments must be numbers", i+1)
			}
			if !(v1 > v2) {
				return fmt.Errorf("line %d: assertion failed: %f is not greater than %f", i+1, v1, v2)
			}
			fmt.Printf("Assertion passed: %f > %f\n", v1, v2)
			passed++
		} else {
			return fmt.Errorf("line %d: unknown command: %s", i+1, line)
		}
	}
	fmt.Printf("\nTest Summary: %d commands executed, %d passed, 0 failed\n", executed, passed)
	return nil
}

func (r *Runner) replaceVariables(line string) string {
	for name, val := range r.variables {
		line = strings.ReplaceAll(line, "$"+name, val)
	}
	return line
}

func (r *Runner) handleCallTool(ctx context.Context, lineIdx int, line string) error {
	rdr := csv.NewReader(strings.NewReader(line))
	rdr.Comma = ' '
	rdr.TrimLeadingSpace = true
	parts, err := rdr.Read()
	if err != nil || len(parts) < 2 {
		return fmt.Errorf("line %d: invalid call_tool command", lineIdx+1)
	}
	toolName := parts[1]
	args := parts[2:]

	fmt.Printf("Executing: %s %v\n", toolName, args)
	return r.callToolPositional(ctx, toolName, args)
}

func (r *Runner) handleInputVar(lineIdx int, line string) error {
	parts := strings.Fields(strings.TrimPrefix(line, "input_var "))
	if len(parts) < 1 {
		return fmt.Errorf("line %d: input_var expects a variable name", lineIdx+1)
	}
	varName := parts[0]
	prompt := "Enter value for " + varName + ": "
	if len(parts) > 1 {
		prompt = strings.Join(parts[1:], " ")
		prompt = strings.Trim(prompt, "\"")
	}

	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		r.variables[varName] = scanner.Text()
	}
	return nil
}

func (r *Runner) handleSetVar(lineIdx int, line string) error {
	parts := strings.Fields(strings.TrimPrefix(line, "set_var "))
	if len(parts) != 2 {
		return fmt.Errorf("line %d: set_var expects <name> <path>", lineIdx+1)
	}
	varName := parts[0]
	path := parts[1]

	val, err := r.extractValue(path)
	if err != nil {
		return fmt.Errorf("line %d: failed to extract %q: %w", lineIdx+1, path, err)
	}
	r.variables[varName] = fmt.Sprintf("%v", val)
	fmt.Printf("Variable set: %s = %s\n", varName, r.variables[varName])
	return nil
}

// extractValue finds a value in the last response using dot notation (e.g., "structuredContent.0.id")
func (r *Runner) extractValue(path string) (any, error) {
	if r.lastRawMap == nil {
		return nil, fmt.Errorf("no previous response available")
	}

	parts := strings.Split(strings.TrimPrefix(path, "$."), ".")
	var current any = r.lastRawMap

	for _, part := range parts {
		if part == "" { continue }
		
		switch v := current.(type) {
		case map[string]any:
			current = v[part]
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("invalid array index %q", part)
			}
			current = v[idx]
		default:
			return nil, fmt.Errorf("cannot navigate path at %q", part)
		}
	}
	
	if current == nil {
		return nil, fmt.Errorf("path %q not found", path)
	}
	return current, nil
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
	if schema == nil { return val }
	typeName, _ := schema["type"].(string)
	switch typeName {
	case "integer":
		if i, err := strconv.Atoi(val); err == nil { return i }
	case "number":
		if f, err := strconv.ParseFloat(val, 64); err == nil { return f }
	case "boolean":
		if strings.ToLower(val) == "true" || val == "1" { return true }
		if strings.ToLower(val) == "false" || val == "0" { return false }
	}
	return val
}

func (r *Runner) call(ctx context.Context, name string, args map[string]any) error {
	var rawResponse map[string]any
	var err error
	var textBuilder strings.Builder

	if r.Raw {
		rawResponse, err = client.CallToolRaw(ctx, r.session, name, args)
		// Extract text for raw mode too
		if err == nil {
			if contents, ok := rawResponse["content"].([]any); ok {
				for _, c := range contents {
					if cm, ok := c.(map[string]any); ok {
						if t, ok := cm["text"].(string); ok {
							textBuilder.WriteString(t)
						}
					}
				}
			}
		}
	} else {
		// Fallback to SDK but wrap it into map[string]any for the variable extractor
		result, callErr := r.session.CallTool(ctx, &mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		})
		if callErr == nil {
			data, _ := json.Marshal(result)
			_ = json.Unmarshal(data, &rawResponse)
			
			// Display for user
			for _, content := range result.Content {
				switch c := content.(type) {
				case *mcp.TextContent:
					fmt.Printf("Response: %s\n", c.Text)
					textBuilder.WriteString(c.Text)
				case *mcp.ImageContent:
					fmt.Printf("Response: [Image data, size %d]\n", len(c.Data))
				}
			}
		}
		err = callErr
	}

	if err != nil { return err }

	r.lastText = textBuilder.String()
	r.lastRawMap = rawResponse
	respData, _ := json.MarshalIndent(rawResponse, "", "  ")
	r.lastResponse = string(respData)
	if r.Raw {
		fmt.Printf("Raw Response:\n%s\n", r.lastResponse)
	}
	return nil
}
