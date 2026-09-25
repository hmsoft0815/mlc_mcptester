// Package scripting provides a simple script runner for testing MCP tools.
package scripting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
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
	// Responder answers elicitation and sampling requests with the answers
	// queued by elicit_response / sample_response. Optional.
	Responder *client.Responder
	// Client receives roots added by add_root. Optional.
	Client *mcp.Client
	// Notifications records server notifications for wait_notification. Optional.
	Notifications *client.Notifications
	// logLevel is sent with each tool call on protocol 2026-07-28 and later.
	logLevel string
	// roots added by add_root, also offered to tasks that ask for them
	roots []*mcp.Root
	// taskMode makes call_tool's machinery start or run a task (call_task, start_task)
	taskMode taskMode
	// lastTask is the last task state seen, for assert_task_status
	lastTask map[string]any
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
	if r.Responder != nil && r.Responder.Out == nil {
		r.Responder.Out = r.w()
	}
	if r.Notifications != nil && r.Notifications.Out == nil {
		r.Notifications.Out = r.w()
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

	if idx := indexOutsideQuotes(processedLine, "<<"); idx != -1 {
		state.accumulating = true
		state.heredocMarker = strings.TrimSpace(processedLine[idx+2:])
		state.currentCommand = strings.TrimSpace(processedLine[:idx])
		return nil
	}

	state.executed++
	parts, err := r.parseArgs(processedLine)
	if err != nil {
		return &scriptError{fmt.Errorf("line %d: failed to parse command: %w", i+1, err)}
	}
	if parts, err = r.replaceInParts(parts); err != nil {
		return &scriptError{fmt.Errorf("line %d: %w", i+1, err)}
	}
	if err := r.dispatchParts(ctx, i, parts); err != nil {
		var se *scriptError
		if errors.As(err, &se) && !strings.HasPrefix(err.Error(), "line ") {
			return &scriptError{fmt.Errorf("line %d: %w", i+1, err)}
		}
		return err
	}
	state.passed++
	return nil
}

func (r *Runner) finalizeHeredoc(ctx context.Context, i int, state *runState) error {
	content := strings.TrimSuffix(state.heredocContent.String(), "\n")

	parts, err := r.parseArgs(state.currentCommand)
	if err != nil {
		return &scriptError{fmt.Errorf("line %d: failed to parse command prefix: %w", i+1, err)}
	}
	if parts, err = r.replaceInParts(parts); err != nil {
		return &scriptError{fmt.Errorf("line %d: %w", i+1, err)}
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
	case "complete":
		return r.handleCompleteCommand(ctx, i, parts)
	case "elicit_response":
		return r.handleElicitResponseCommand(i, parts)
	case "sample_response":
		return r.handleSampleResponseCommand(i, parts)
	case "add_root":
		return r.handleAddRootCommand(i, parts)
	case "assert_elicited":
		return r.handleAssertElicitedCommand(i, parts)
	case "assert_sampled":
		return r.handleAssertSampledCommand(i, parts)
	case "call_tool_raw":
		if len(parts) != 3 {
			return &scriptError{fmt.Errorf("line %d: usage: call_tool_raw <tool> '<json object>'", i+1)}
		}
		return r.callToolRaw(ctx, parts[1], parts[2])
	case "call_task":
		return r.handleTaskCallCommand(ctx, i, parts, taskCall)
	case "start_task":
		return r.handleTaskCallCommand(ctx, i, parts, taskStart)
	case "wait_task":
		return r.handleWaitTaskCommand(ctx, i, parts)
	case "get_task":
		return r.handleGetTaskCommand(ctx, i, parts)
	case "cancel_task":
		return r.handleCancelTaskCommand(ctx, i, parts)
	case "assert_task_status":
		return r.handleAssertTaskStatusCommand(i, parts)
	case "list_skills":
		return r.handleListSkillsCommand(ctx, i)
	case "verify_skills":
		return r.handleVerifySkillsCommand(ctx, i)
	case "subscribe":
		return r.handleSubscribeCommand(ctx, i, parts)
	case "wait_notification":
		return r.handleWaitNotificationCommand(ctx, i, parts)
	case "logging":
		return r.handleLoggingCommand(ctx, i, parts)
	default:
		return &scriptError{fmt.Errorf("line %d: unknown command: %s", i+1, cmd)}
	}
}

// scriptError is a mistake in the script itself. expect_error must not
// accept it as the error it expects from the server.
type scriptError struct{ err error }

func (e *scriptError) Error() string { return e.err.Error() }
func (e *scriptError) Unwrap() error { return e.err }
