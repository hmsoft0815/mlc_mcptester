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

// Run executes a script string against the established MCP session.
func (r *Runner) Run(ctx context.Context, script string) error {
	lines := strings.Split(script, "\n")
	executed := 0
	passed := 0

	for i, line := range lines {
		line = r.preprocessLine(line)
		if line == "" {
			continue
		}

		executed++
		line = r.replaceVariables(line)

		if err := r.dispatchCommand(ctx, i, line); err != nil {
			return err
		}
		passed++
	}
	fmt.Printf("\nTest Summary: %d commands executed, %d passed, 0 failed\n", executed, passed)
	return nil
}

func (r *Runner) dispatchCommand(ctx context.Context, i int, line string) error {
	switch {
	case strings.HasPrefix(line, "call_tool "):
		return r.handleCallTool(ctx, i, line)
	case strings.HasPrefix(line, "set_var "):
		return r.handleSetVar(i, line)
	case strings.HasPrefix(line, "input_var "):
		return r.handleInputVar(i, line)
	case strings.HasPrefix(line, "assert_contains "):
		return r.handleAssertContains(i, line)
	case strings.HasPrefix(line, "assert_equals "):
		return r.handleAssertEquals(i, line)
	case strings.HasPrefix(line, "assert_number "):
		return r.handleAssertNumber(i, line)
	case strings.HasPrefix(line, "assert_gt "):
		return r.handleAssertGreaterThan(i, line)
	default:
		return fmt.Errorf("line %d: unknown command: %s", i+1, line)
	}
}
