package scripting

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// handleCallTool calls a tool with arguments
// expected format: call_tool <tool_name> <arg1> <arg2> ...
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

// handleInputVar prompts the user for a value and stores it in a variable
// expected format: input_var <name> [prompt]
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

// handleSetVar sets a variable to a value extracted from the last response
// expected format: set_var <name> <path>
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
