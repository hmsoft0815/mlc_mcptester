package scripting

import (
	"context"
	"fmt"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
)

// Commands for the Tasks extension (io.modelcontextprotocol/tasks).

type taskMode int

const (
	taskOff   taskMode = iota
	taskCall           // call_task: start and wait for the result
	taskStart          // start_task: only the handle
)

const defaultTaskTimeout = 2 * time.Minute

func (r *Runner) taskClient() *client.TaskClient {
	return &client.TaskClient{Session: r.session, Responder: r.Responder, Roots: r.roots, Out: r.w()}
}

// handleTaskCallCommand runs "call_task <tool> [args]" and "start_task <tool> [args]"
// with the argument handling of call_tool.
func (r *Runner) handleTaskCallCommand(ctx context.Context, i int, parts []string, mode taskMode) error {
	if len(parts) < 2 {
		return fmt.Errorf("line %d: %s expects at least a tool name", i+1, parts[0])
	}
	if r.Responder == nil {
		return fmt.Errorf("line %d: this runner cannot run tasks", i+1)
	}
	r.taskMode = mode
	defer func() { r.taskMode = taskOff }()
	if mode == taskCall {
		// Waiting may take long; timeout <d> call_task ... sets a tighter limit
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, defaultTaskTimeout)
			defer cancel()
		}
	}
	// Unwrapped like call_tool, so expect_error can tell tool from RPC errors
	return r.callToolPositional(ctx, parts[1], parts[2:])
}

// executeTaskCall starts the tool as a task, waits for it and returns the
// final tool result. A server may answer synchronously instead.
func (r *Runner) executeTaskCall(ctx context.Context, name string, args map[string]any) (map[string]any, string, error) {
	tc := r.taskClient()
	res, done, err := tc.Start(ctx, name, args)
	if err != nil {
		return nil, "", err
	}
	if done {
		fmt.Fprintln(r.w(), "[TASK] the server answered synchronously")
		r.lastTask = nil
		return res, extractTextFromRaw(res), nil
	}
	task, err := tc.Wait(ctx, res["taskId"].(string))
	if err != nil {
		return nil, "", err
	}
	r.lastTask = task
	result, err := client.TaskResult(task)
	if err != nil {
		return nil, "", err
	}
	text := extractTextFromRaw(result)
	fmt.Fprintf(r.w(), "Response: %s\n", text)
	return result, text, nil
}

// startTask stores the task handle, so set_var can take taskId from it.
func (r *Runner) startTask(ctx context.Context, name string, args map[string]any) error {
	res, done, err := r.taskClient().Start(ctx, name, args)
	if err != nil {
		return err
	}
	if done {
		return fmt.Errorf("the server answered %s synchronously instead of creating a task", name)
	}
	r.lastTask = res
	r.updateState(res, fmt.Sprint(res["taskId"]))
	return nil
}

// handleWaitTaskCommand runs "wait_task <taskId> [timeout]" and stores the
// final task state (status, result, error).
func (r *Runner) handleWaitTaskCommand(ctx context.Context, i int, parts []string) error {
	if len(parts) < 2 || len(parts) > 3 {
		return fmt.Errorf("line %d: usage: wait_task <taskId> [timeout]", i+1)
	}
	timeout := defaultTaskTimeout
	if len(parts) == 3 {
		d, err := time.ParseDuration(parts[2])
		if err != nil {
			return fmt.Errorf("line %d: invalid timeout %q", i+1, parts[2])
		}
		timeout = d
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	task, err := r.taskClient().Wait(ctx, parts[1])
	if err != nil {
		return fmt.Errorf("line %d: %w", i+1, err)
	}
	r.storeTask(task)
	return nil
}

// handleGetTaskCommand runs "get_task <taskId>" (one tasks/get).
func (r *Runner) handleGetTaskCommand(ctx context.Context, i int, parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("line %d: usage: get_task <taskId>", i+1)
	}
	task, err := r.taskClient().Get(ctx, parts[1])
	if err != nil {
		return fmt.Errorf("line %d: %w", i+1, err)
	}
	r.storeTask(task)
	return nil
}

// handleCancelTaskCommand runs "cancel_task <taskId>".
func (r *Runner) handleCancelTaskCommand(ctx context.Context, i int, parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("line %d: usage: cancel_task <taskId>", i+1)
	}
	if err := r.taskClient().Cancel(ctx, parts[1]); err != nil {
		return fmt.Errorf("line %d: %w", i+1, err)
	}
	return nil
}

// handleAssertTaskStatusCommand runs "assert_task_status <status>".
func (r *Runner) handleAssertTaskStatusCommand(i int, parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("line %d: usage: assert_task_status <status>", i+1)
	}
	if r.lastTask == nil {
		return fmt.Errorf("line %d: assertion failed: no task state (the server may have answered synchronously)", i+1)
	}
	if got := fmt.Sprint(r.lastTask["status"]); got != parts[1] {
		return fmt.Errorf("line %d: assertion failed: task status is %q, want %q", i+1, got, parts[1])
	}
	fmt.Fprint(r.w(), i18n.T(i18n.MsgAssertionPassed, "task status "+parts[1]))
	return nil
}

// storeTask keeps a task state: set_var sees status, result.*, error.*; the
// text is the result text of a completed task, else the status message.
func (r *Runner) storeTask(task map[string]any) {
	r.lastTask = task
	text := fmt.Sprint(task["statusMessage"])
	if result, err := client.TaskResult(task); err == nil {
		text = extractTextFromRaw(result)
	}
	r.updateState(task, text)
}
