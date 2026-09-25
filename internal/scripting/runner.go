// Package scripting provides a simple script runner for testing MCP tools.
package scripting

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Runner manages the execution of MCP test scripts.
type Runner struct {
	session         *mcp.ClientSession
	lastResponse    string // The full JSON response
	lastText        string // Just the text content
	lastRawMap      map[string]any
	Raw             bool
	variables       map[string]string
	lastErrorCode   int64
	lastIsToolError bool
	// out receives per-command progress. It is stderr in JSON mode so that
	// stdout carries nothing but the summary document.
	out io.Writer
}

// TestResult holds numeric summary of test execution
type TestResult struct {
	Executed int           `json:"executed"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Failures []TestFailure `json:"failures,omitempty"`
}

// TestFailure records one failed script line.
type TestFailure struct {
	Line  int    `json:"line"`
	Error string `json:"error"`
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
	failed         int
}

// w returns the writer for per-command progress output.
func (r *Runner) w() io.Writer {
	if r.out == nil {
		return os.Stdout
	}
	return r.out
}

// Run executes a script string against the established MCP session.
func (r *Runner) Run(ctx context.Context, script string, outputFormat string) (*TestResult, error) {
	lines := strings.Split(script, "\n")
	state := &runState{}
	var failures []TestFailure

	if outputFormat == "json" && r.out == nil {
		r.out = os.Stderr
	}

	for i, line := range lines {
		if err := r.processLine(ctx, i, line, state); err != nil {
			state.failed++
			failures = append(failures, TestFailure{Line: i + 1, Error: err.Error()})
			fmt.Fprintf(r.w(), "Error: %v\n", err)
		}
	}

	if state.accumulating {
		return nil, fmt.Errorf("error: heredoc marker %q not found", state.heredocMarker)
	}

	result := &TestResult{
		Executed: state.executed,
		Passed:   state.passed,
		Failed:   state.failed,
		Failures: failures,
	}

	if outputFormat == "text" {
		fmt.Print(i18n.T(i18n.MsgTestSummary, result.Executed, result.Passed, result.Failed))
	} else if outputFormat == "json" {
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
	}

	return result, nil
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
	finalCmd, err := r.replaceVariables(processedLine)
	if err != nil {
		return fmt.Errorf("line %d: %w", i+1, err)
	}
	if err := r.dispatchCommand(ctx, i, finalCmd); err != nil {
		return err
	}
	state.passed++
	return nil
}

func (r *Runner) finalizeHeredoc(ctx context.Context, i int, state *runState) error {
	content := strings.TrimSuffix(state.heredocContent.String(), "\n")

	var err error
	state.currentCommand, err = r.replaceVariables(state.currentCommand)
	if err != nil {
		return fmt.Errorf("line %d: %w", i+1, err)
	}
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
	case "echo":
		return r.handleEchoCommand(parts)
	case "call_tool":
		return r.handleCallToolParts(ctx, i, parts)
	case "set_var":
		return r.handleSetVarCommand(i, parts)
	case "input_var":
		return r.handleInputVarParts(i, parts)
	case "assert_contains":
		return r.handleAssertContainsParts(i, parts)
	case "assert_equals":
		return r.handleAssertEqualsParts(i, parts)
	case "assert_number":
		return r.handleAssertNumberCommand(i, parts)
	case "assert_gt":
		return r.handleAssertGreaterThanCommand(i, parts)
	case "assert_string_length":
		return r.handleAssertStringLengthCommand(i, parts)
	case "assert_error_code":
		return r.handleAssertErrorCodeCommand(i, parts)
	case "assert_tool_error":
		return r.handleAssertToolErrorCommand(i, parts)
	case "timeout":
		return r.handleTimeoutCommand(ctx, i, parts)
	case "expect_error":
		return r.handleExpectErrorCommand(ctx, i, parts)
	case "ping":
		return r.handlePingCommand(ctx, i)
	case "logging":
		return r.handleLoggingCommand(ctx, i, parts)
	default:
		return fmt.Errorf("line %d: unknown command: %s", i+1, cmd)
	}
}
