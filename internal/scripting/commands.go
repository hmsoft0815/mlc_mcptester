package scripting

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
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
	if err != nil {
		return err
	}
	return r.handleSetVarCommand(lineIdx, parts)
}

func (r *Runner) handleSetVarCommand(lineIdx int, parts []string) error {
	if len(parts) != 3 {
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

func (r *Runner) handleCallToolParts(ctx context.Context, lineIdx int, parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: call_tool expects at least a tool name", lineIdx+1)
	}
	return r.callToolPositional(ctx, parts[1], parts[2:])
}

func (r *Runner) handleTimeoutCommand(ctx context.Context, i int, parts []string) error {
	if len(parts) < 3 {
		return fmt.Errorf("line %d: timeout expects <ms> <command>", i+1)
	}
	ms, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("line %d: invalid timeout value: %s", i+1, parts[1])
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(ms)*time.Millisecond)
	defer cancel()
	return r.dispatchParts(timeoutCtx, i, parts[2:])
}

func (r *Runner) handleExpectErrorCommand(ctx context.Context, i int, parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: expect_error expects a command", i+1)
	}
	err := r.dispatchParts(ctx, i, parts[1:])
	if err == nil {
		return fmt.Errorf("line %d: expected error but command succeeded", i+1)
	}
	// Capture error as response for assertions
	r.updateState(map[string]any{"error": err.Error()}, err.Error())
	fmt.Printf("Expected error caught: %v\n", err)
	return nil
}
