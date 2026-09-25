package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TasksExtension identifies the Tasks extension (spec 2026-07-28).
const TasksExtension = "io.modelcontextprotocol/tasks"

// Terminal task statuses.
var terminalStatus = map[string]bool{"completed": true, "failed": true, "cancelled": true}

// TaskClient runs tool calls as tasks. The go-sdk client does not know the
// extension (it drops the task fields of a CreateTaskResult), so requests go
// over the raw connection with the per-request metadata built here.
type TaskClient struct {
	Session   *mcp.ClientSession
	Responder *Responder // answers the task's inputRequests
	Roots     []*mcp.Root
	Out       io.Writer // one line per status change; optional
	// MaxPoll caps the server's pollIntervalMs, to keep tests fast.
	MaxPoll time.Duration

	logged map[any]any // last logged status per task
}

// meta is the per-request metadata, declaring the extension and the input
// kinds the tester can answer.
func (c *TaskClient) meta() map[string]any {
	return map[string]any{
		mcp.MetaKeyProtocolVersion: c.Session.InitializeResult().ProtocolVersion,
		mcp.MetaKeyClientInfo:      map[string]any{"name": version.AppName, "version": version.Version},
		mcp.MetaKeyClientCapabilities: map[string]any{
			"extensions":  map[string]any{TasksExtension: map[string]any{}},
			"elicitation": map[string]any{"form": map[string]any{}, "url": map[string]any{}},
			"sampling":    map[string]any{},
			"roots":       map[string]any{"listChanged": true},
		},
	}
}

func (c *TaskClient) call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	if !IsStateless(c.Session) {
		return nil, fmt.Errorf("tasks need protocol %s or later, the server speaks %s", StatelessRevision, c.Session.InitializeResult().ProtocolVersion)
	}
	params["_meta"] = c.meta()
	return CallRaw(ctx, c.Session, method, params)
}

// Start calls the tool declaring the extension. It returns the task handle,
// or with done set the synchronous CallToolResult if the server chose not to
// create a task.
func (c *TaskClient) Start(ctx context.Context, name string, args map[string]any) (res map[string]any, done bool, err error) {
	res, err = c.call(ctx, "tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return nil, false, err
	}
	if res["resultType"] != "task" {
		return res, true, nil
	}
	if id, _ := res["taskId"].(string); id == "" {
		return nil, false, fmt.Errorf("task handle without taskId")
	}
	c.logStatus(res)
	return res, false, nil
}

// Get returns the current state of a task (tasks/get).
func (c *TaskClient) Get(ctx context.Context, taskID string) (map[string]any, error) {
	return c.call(ctx, "tasks/get", map[string]any{"taskId": taskID})
}

// Cancel asks the server to cancel a task (tasks/cancel).
func (c *TaskClient) Cancel(ctx context.Context, taskID string) error {
	_, err := c.call(ctx, "tasks/cancel", map[string]any{"taskId": taskID})
	return err
}

// Wait polls the task until it is completed, failed or cancelled, answering
// input requests on the way, and returns its final state.
func (c *TaskClient) Wait(ctx context.Context, taskID string) (map[string]any, error) {
	answered := map[string]bool{}
	for {
		task, err := c.Get(ctx, taskID)
		if err != nil {
			return nil, err
		}
		status, _ := task["status"].(string)
		c.logStatus(task)
		if terminalStatus[status] {
			return task, nil
		}
		if status == "input_required" {
			if err := c.answer(ctx, taskID, task["inputRequests"], answered); err != nil {
				return nil, err
			}
		}

		interval := time.Second
		if ms, ok := task["pollIntervalMs"].(float64); ok && ms > 0 {
			interval = time.Duration(ms) * time.Millisecond
		}
		if c.MaxPoll > 0 && interval > c.MaxPoll {
			interval = c.MaxPoll
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("task %s still %s: %w", taskID, status, ctx.Err())
		case <-time.After(interval):
		}
	}
}

// answer fulfills the not yet answered input requests of a task via tasks/update.
func (c *TaskClient) answer(ctx context.Context, taskID string, raw any, answered map[string]bool) error {
	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	var requests mcp.InputRequestMap
	if err := json.Unmarshal(data, &requests); err != nil {
		return fmt.Errorf("task %s: invalid inputRequests: %w", taskID, err)
	}
	responses := map[string]any{}
	for key, ir := range requests {
		if answered[key] {
			continue // keys are unique over a task's lifetime
		}
		resp, err := c.Responder.Answer(ctx, ir, c.Roots)
		if err != nil {
			return fmt.Errorf("task %s: input request %q: %w", taskID, key, err)
		}
		responses[key] = resp
		answered[key] = true
	}
	if len(responses) == 0 {
		return nil
	}
	_, err = c.call(ctx, "tasks/update", map[string]any{"taskId": taskID, "inputResponses": responses})
	return err
}

// logStatus prints a line when a task's status changes.
func (c *TaskClient) logStatus(task map[string]any) {
	if c.Out == nil || c.logged[task["taskId"]] == task["status"] {
		return
	}
	if c.logged == nil {
		c.logged = map[any]any{}
	}
	c.logged[task["taskId"]] = task["status"]
	msg := ""
	if m, ok := task["statusMessage"].(string); ok && m != "" {
		msg = ": " + m
	}
	fmt.Fprintf(c.Out, "[TASK] %v %v%s\n", task["taskId"], task["status"], msg)
}

// TaskResult returns the final tool result of a completed task, or an
// error for a failed or cancelled one.
func TaskResult(task map[string]any) (map[string]any, error) {
	switch task["status"] {
	case "completed":
		result, _ := task["result"].(map[string]any)
		if result == nil {
			return nil, fmt.Errorf("completed task without result")
		}
		return result, nil
	case "failed":
		e, _ := task["error"].(map[string]any)
		code, _ := e["code"].(float64)
		msg, _ := e["message"].(string)
		return nil, &RPCError{Code: int64(code), Message: msg}
	}
	return nil, fmt.Errorf("task %v", task["status"])
}
