// Package taskcheck checks a server against the Tasks extension
// (io.modelcontextprotocol/tasks, spec 2026-07-28): error codes of the
// tasks/* methods without calling a tool, and with a tool given the whole
// life of one task — handle, durable creation, polling, statuses, results.
package taskcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Status of one check.
type Status string

const (
	Pass Status = "PASS"
	Fail Status = "FAIL" // a MUST is violated
	Warn Status = "WARN" // a SHOULD is violated
	Skip Status = "SKIP"
)

// Error codes of the extension.
const (
	codeMethodNotFound            = -32601
	codeInvalidParams             = -32602
	codeMissingRequiredCapability = -32021
)

// unknownTaskID is a task id no server hands out.
const unknownTaskID = "mcp-tester-no-such-task"

// Result is the outcome of one check.
type Result struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Report is the outcome of a check run.
type Report struct {
	Declared bool     `json:"declared"`
	Tool     string   `json:"tool,omitempty"`
	TaskID   string   `json:"taskId,omitempty"`
	Statuses []string `json:"statuses,omitempty"` // observed status sequence, without repeats
	Results  []Result `json:"results"`
}

// Failed reports whether any check failed.
func (r *Report) Failed() bool {
	for _, res := range r.Results {
		if res.Status == Fail {
			return true
		}
	}
	return false
}

func (r *Report) add(name string, status Status, format string, args ...any) {
	r.Results = append(r.Results, Result{Name: name, Status: status, Detail: fmt.Sprintf(format, args...)})
}

// Checker runs the checks over a session.
type Checker struct {
	Session *mcp.ClientSession
	// Tool, if set, is called as a task to check a task's life; without it
	// only the tasks/* methods are probed. The tool is called twice: once
	// without and once with the extension declared.
	Tool string
	Args map[string]any
	// Cancel cancels the task right after it was created.
	Cancel bool
	// Responder answers input requests; without one an input_required task
	// is cancelled.
	Responder *client.Responder
	// Timeout bounds the wait for a terminal status (default 2 minutes).
	Timeout time.Duration
	// PlainWait bounds the call without the extension (default 5 seconds).
	PlainWait time.Duration
	// MaxPoll caps the server's pollIntervalMs (default 5 seconds).
	MaxPoll time.Duration
}

// Run probes the tasks/* methods and, with a tool, the life of one task.
func (c *Checker) Run(ctx context.Context) *Report {
	rep := &Report{Tool: c.Tool}
	_, rep.Declared = c.Session.InitializeResult().Capabilities.Extensions[client.TasksExtension]
	if !client.IsStateless(c.Session) {
		rep.add("protocol revision", Skip, "tasks need %s or later, the server speaks %s", client.StatelessRevision, c.Session.InitializeResult().ProtocolVersion)
		return rep
	}
	if !rep.Declared {
		rep.add("extension declared", Skip, "the server does not declare %s", client.TasksExtension)
		if c.Tool != "" {
			c.checkPlainCall(ctx, rep)
		}
		return rep
	}
	rep.add("extension declared", Pass, "%s", client.TasksExtension)
	c.checkMissingCapability(ctx, rep)
	c.checkUnknownTask(ctx, rep)
	if c.Tool != "" {
		c.checkPlainCall(ctx, rep)
		c.checkLifecycle(ctx, rep)
	}
	return rep
}

// call sends a request; declare puts the extension into the per-request
// client capabilities.
func (c *Checker) call(ctx context.Context, method string, params map[string]any, declare bool) (map[string]any, error) {
	caps := map[string]any{}
	if declare {
		caps = map[string]any{
			"extensions":  map[string]any{client.TasksExtension: map[string]any{}},
			"elicitation": map[string]any{"form": map[string]any{}, "url": map[string]any{}},
			"sampling":    map[string]any{},
		}
	}
	params["_meta"] = client.RequestMeta(c.Session, caps)
	return client.CallRaw(ctx, c.Session, method, params)
}

// taskMethods are the methods of the extension with params naming unknownTaskID.
func taskMethods() []struct {
	method string
	params func() map[string]any
} {
	return []struct {
		method string
		params func() map[string]any
	}{
		{"tasks/get", func() map[string]any { return map[string]any{"taskId": unknownTaskID} }},
		{"tasks/update", func() map[string]any {
			return map[string]any{"taskId": unknownTaskID, "inputResponses": map[string]any{}}
		}},
		{"tasks/cancel", func() map[string]any { return map[string]any{"taskId": unknownTaskID} }},
	}
}

// checkMissingCapability: tasks/* from a client that does not declare the
// extension MUST fail with -32021.
func (c *Checker) checkMissingCapability(ctx context.Context, rep *Report) {
	for _, m := range taskMethods() {
		name := m.method + " without the extension"
		_, err := c.call(ctx, m.method, m.params(), false)
		code, rpcErr := errorCode(err)
		switch {
		case err == nil:
			rep.add(name, Fail, "answered with a result, must fail with %d", codeMissingRequiredCapability)
		case !rpcErr:
			rep.add(name, Fail, "%v", err)
		case code != codeMissingRequiredCapability:
			rep.add(name, Fail, "error %d, must be %d (Missing Required Client Capability)", code, codeMissingRequiredCapability)
		case !namesExtension(err):
			rep.add(name, Warn, "%d without data.requiredCapabilities.extensions naming %s", code, client.TasksExtension)
		default:
			rep.add(name, Pass, "%d", code)
		}
	}
}

// checkUnknownTask: an unknown task id MUST give -32602 on tasks/get and
// SHOULD on tasks/update and tasks/cancel.
func (c *Checker) checkUnknownTask(ctx context.Context, rep *Report) {
	for _, m := range taskMethods() {
		name := m.method + " of an unknown task"
		miss := Warn
		if m.method == "tasks/get" {
			miss = Fail
		}
		_, err := c.call(ctx, m.method, m.params(), true)
		code, rpcErr := errorCode(err)
		switch {
		case err == nil:
			rep.add(name, miss, "answered with a result, expected error %d", codeInvalidParams)
		case rpcErr && code == codeMethodNotFound:
			rep.add(name, Fail, "%s is not implemented although the extension is declared", m.method)
		case !rpcErr:
			rep.add(name, Fail, "%v", err)
		case code != codeInvalidParams:
			rep.add(name, miss, "error %d, expected %d (Invalid params)", code, codeInvalidParams)
		default:
			rep.add(name, Pass, "%d", code)
		}
	}
}

// errorCode returns the JSON-RPC error code of err, if it is one.
func errorCode(err error) (int64, bool) {
	var rpc *client.RPCError
	if errors.As(err, &rpc) {
		return rpc.Code, true
	}
	return 0, false
}

// namesExtension reports whether a -32021 error names the Tasks extension in
// data.requiredCapabilities.extensions.
func namesExtension(err error) bool {
	var rpc *client.RPCError
	if !errors.As(err, &rpc) || rpc.Data == nil {
		return false
	}
	raw, ok := rpc.Data.(json.RawMessage)
	if !ok {
		var merr error
		if raw, merr = json.Marshal(rpc.Data); merr != nil {
			return false
		}
	}
	var data struct {
		RequiredCapabilities struct {
			Extensions map[string]any `json:"extensions"`
		} `json:"requiredCapabilities"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return false
	}
	_, ok = data.RequiredCapabilities.Extensions[client.TasksExtension]
	return ok
}
