package scripting

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

// handleCallTool calls a tool with arguments
// expected format: call_tool <tool_name> <arg1> <arg2> ...
func (r *Runner) handleCallTool(ctx context.Context, lineIdx int, line string) error {
	parts, err := r.parseArgs(line)
	if err != nil || len(parts) < 2 {
		return fmt.Errorf("line %d: invalid call_tool command", lineIdx+1)
	}
	toolName := parts[1]
	args := parts[2:]

	fmt.Printf("Executing: %s %v\n", toolName, args)
	return r.callToolPositional(ctx, toolName, args)
}

func (r *Runner) handleInputVar(lineIdx int, line string) error {
	parts, err := r.parseArgs(line)
	if err != nil {
		return err
	}
	return r.handleInputVarParts(lineIdx, parts)
}

func (r *Runner) handleInputVarParts(lineIdx int, parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: input_var expects a variable name", lineIdx+1)
	}
	varName := parts[1]
	prompt := "Enter value for " + varName + ": "
	if len(parts) > 2 {
		prompt = strings.Join(parts[2:], " ")
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
	parts, err := r.parseArgs(line)
	if err != nil || len(parts) != 3 {
		return fmt.Errorf("line %d: set_var expects <name> <path>", lineIdx+1)
	}
	return r.handleSetVarParts(lineIdx, parts[1], parts[2])
}

func (r *Runner) handleSetVarParts(lineIdx int, varName, path string) error {
	val, err := r.extractValue(path)
	if err != nil {
		return fmt.Errorf("line %d: failed to extract %q: %w", lineIdx+1, path, err)
	}
	r.variables[varName] = fmt.Sprintf("%v", val)
	fmt.Printf("Variable set: %s = %s\n", varName, r.variables[varName])
	return nil
}
