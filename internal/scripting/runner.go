// Package scripting provides a simple script runner for testing MCP tools.
package scripting

import (
	"context"
	"fmt"
	"strings"

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

type runState struct {
	accumulating   bool
	heredocMarker  string
	heredocContent strings.Builder
	currentCommand string
	executed       int
	passed         int
}

// Run executes a script string against the established MCP session.
func (r *Runner) Run(ctx context.Context, script string) error {
	lines := strings.Split(script, "\n")
	state := &runState{}

	for i, line := range lines {
		if err := r.processLine(ctx, i, line, state); err != nil {
			return err
		}
	}

	if state.accumulating {
		return fmt.Errorf("error: heredoc marker %q not found", state.heredocMarker)
	}

	fmt.Printf("\nTest Summary: %d commands executed, %d passed, 0 failed\n", state.executed, state.passed)
	return nil
}

func (r *Runner) processLine(ctx context.Context, i int, line string, state *runState) error {
	if state.accumulating {
		if strings.TrimSpace(line) == state.heredocMarker {
			state.accumulating = false
			return r.finalizeHeredoc(ctx, i, state)
		}
		state.heredocContent.WriteString(line + "\n")
		return nil
	}

	processedLine := r.preprocessLine(line)
	if processedLine == "" {
		return nil
	}

	if idx := strings.Index(processedLine, "<<"); idx != -1 {
		state.accumulating = true
		state.heredocMarker = strings.TrimSpace(processedLine[idx+2:])
		state.currentCommand = strings.TrimSpace(processedLine[:idx])
		return nil
	}

	state.executed++
	finalCmd := r.replaceVariables(processedLine)
	if err := r.dispatchCommand(ctx, i, finalCmd); err != nil {
		return err
	}
	state.passed++
	return nil
}

func (r *Runner) finalizeHeredoc(ctx context.Context, i int, state *runState) error {
	content := strings.TrimSuffix(state.heredocContent.String(), "\n")

	state.currentCommand = r.replaceVariables(state.currentCommand)
	parts, err := r.parseArgs(state.currentCommand)
	if err != nil {
		return fmt.Errorf("line %d: failed to parse command prefix: %w", i+1, err)
	}
	parts = append(parts, content)

	state.executed++
	if err := r.dispatchParts(ctx, i, parts); err != nil {
		return err
	}
	state.passed++
	state.currentCommand = ""
	state.heredocContent.Reset()
	return nil
}

func (r *Runner) dispatchCommand(ctx context.Context, i int, line string) error {
	parts, err := r.parseArgs(line)
	if err != nil {
		return fmt.Errorf("line %d: failed to parse command: %w", i+1, err)
	}
	return r.dispatchParts(ctx, i, parts)
}

func (r *Runner) dispatchParts(ctx context.Context, i int, parts []string) error {
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	switch cmd {
	case "call_tool":
		if len(parts) < 2 {
			return fmt.Errorf("line %d: call_tool expects at least a tool name", i+1)
		}
		return r.callToolPositional(ctx, parts[1], parts[2:])
	case "set_var":
		if len(parts) != 3 {
			return fmt.Errorf("line %d: set_var expects <name> <path>", i+1)
		}
		return r.handleSetVarParts(i, parts[1], parts[2])
	case "input_var":
		return r.handleInputVarParts(i, parts)
	case "assert_contains":
		return r.handleAssertContainsParts(i, parts)
	case "assert_equals":
		return r.handleAssertEqualsParts(i, parts)
	case "assert_number":
		if len(parts) != 2 {
			return fmt.Errorf("line %d: assert_number expects 1 argument", i+1)
		}
		return r.handleAssertNumber(i, parts[1])
	case "assert_gt":
		if len(parts) != 3 {
			return fmt.Errorf("line %d: assert_gt expects 2 arguments", i+1)
		}
		return r.handleAssertGreaterThan(i, parts[1], parts[2])
	default:
		return fmt.Errorf("line %d: unknown command: %s", i+1, cmd)
	}
}
