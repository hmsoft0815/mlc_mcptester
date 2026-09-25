package scripting

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (r *Runner) handleCallTool(ctx context.Context, lineIdx int, line string) error {
	parts, err := r.parseArgs(line)
	if err != nil || len(parts) < 2 {
		return fmt.Errorf("line %d: invalid call_tool command", lineIdx+1)
	}
	toolName := parts[1]
	args := parts[2:]
	fmt.Fprint(r.w(), i18n.T(i18n.MsgExecuting, toolName, args))
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
	fmt.Fprint(r.w(), prompt)
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
	fmt.Fprint(r.w(), i18n.T(i18n.MsgVariableSet, varName, r.variables[varName]))
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
	r.lastErrorCode = 0
	r.lastIsToolError = false
	// errors.As: commands may wrap the error with the script line
	var rpcErr *client.RPCError
	var toolErr *client.ToolError
	if errors.As(err, &rpcErr) {
		r.lastErrorCode = rpcErr.Code
	} else if errors.As(err, &toolErr) {
		r.lastIsToolError = true
	}
	r.updateState(map[string]any{"error": err.Error(), "code": r.lastErrorCode, "isToolError": r.lastIsToolError}, err.Error())
	fmt.Fprint(r.w(), i18n.T(i18n.MsgExpectedError, err, r.lastErrorCode))
	return nil
}

func (r *Runner) handlePingCommand(ctx context.Context, i int) error {
	fmt.Fprintln(r.w(), "Ping...")
	method, err := client.Ping(ctx, r.session)
	if err != nil {
		return fmt.Errorf("line %d: ping failed: %w", i+1, err)
	}
	fmt.Fprintf(r.w(), "Pong! (%s)\n", method)
	return nil
}

func (r *Runner) handleLoggingCommand(ctx context.Context, i int, parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: logging expects a level", i+1)
	}
	level := parts[1]
	if client.IsStateless(r.session) {
		// No logging/setLevel since 2026-07-28: the level goes with every later call
		r.logLevel = level
		fmt.Fprintf(r.w(), "Server logging level %s is sent with each following call\n", level)
		return nil
	}
	fmt.Fprintf(r.w(), "Setting server logging level to %s...\n", level)
	if err := r.session.SetLoggingLevel(ctx, &mcp.SetLoggingLevelParams{Level: mcp.LoggingLevel(level)}); err != nil {
		return fmt.Errorf("line %d: failed to set logging level: %w", i+1, err)
	}
	return nil
}

func (r *Runner) handleEchoCommand(parts []string) error {
	if len(parts) > 1 {
		fmt.Fprintln(r.w(), strings.Join(parts[1:], " "))
	} else {
		fmt.Fprintln(r.w())
	}
	return nil
}

// handleCompleteCommand runs "complete <prompt:name|resource:uri> <argument> [value]".
// The result is kept as {values, total, hasMore}, so set_var can address
// values.0 or total, and assert_contains sees the values one per line.
func (r *Runner) handleCompleteCommand(ctx context.Context, i int, parts []string) error {
	if len(parts) < 3 || len(parts) > 4 {
		return fmt.Errorf("line %d: usage: complete <prompt:name|resource:uri> <argument> [value]", i+1)
	}
	value := ""
	if len(parts) == 4 {
		value = parts[3]
	}
	res, err := client.Complete(ctx, r.session, parts[1], parts[2], value, nil)
	if err != nil {
		return fmt.Errorf("line %d: complete: %w", i+1, err)
	}

	values := make([]any, len(res.Completion.Values))
	for j, v := range res.Completion.Values {
		values[j] = v
	}
	r.updateState(map[string]any{
		"values":  values,
		"total":   res.Completion.Total,
		"hasMore": res.Completion.HasMore,
	}, strings.Join(res.Completion.Values, "\n"))
	fmt.Fprintf(r.w(), "Completions: %s\n", strings.Join(res.Completion.Values, ", "))
	return nil
}
