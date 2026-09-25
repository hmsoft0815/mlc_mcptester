// Package mcptasks adds the Tasks extension (io.modelcontextprotocol/tasks,
// spec 2026-07-28) to a server built on the official go-sdk, which does not
// support it yet.
//
// A tool registered as task-capable answers tools/call with a task handle
// (resultType "task") when the client declares the extension on that request,
// runs in the background and is polled with tasks/get. Clients without the
// extension get the normal synchronous result. The tool can ask the client for
// input while it runs (status input_required) with RequestInput.
//
//	caps := &mcp.ServerCapabilities{}
//	mcptasks.Declare(caps)
//	s := mcp.NewServer(impl, &mcp.ServerOptions{Capabilities: caps})
//	mcp.AddTool(s, tool, handler)
//	mcptasks.Enable(s, mcptasks.NewStore(), "generate_image")
package mcptasks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Extension is the identifier of the Tasks extension.
const Extension = "io.modelcontextprotocol/tasks"

// Task statuses.
const (
	Working       = "working"
	InputRequired = "input_required"
	Completed     = "completed"
	Failed        = "failed"
	Cancelled     = "cancelled"
)

// Error codes the extension uses.
const (
	codeInvalidParams              = -32602
	codeInternalError              = -32603
	codeMissingRequiredCapability  = -32021
	defaultTTL                     = time.Hour
	defaultPollInterval            = time.Second
	metaKeyClientCapabilities      = "io.modelcontextprotocol/clientCapabilities"
	methodToolsCall                = "tools/call"
	methodTasksGet                 = "tasks/get"
	methodTasksUpdate              = "tasks/update"
	methodTasksCancel              = "tasks/cancel"
	resultTypeTask, resultComplete = "task", "complete"
)

// Declare adds the extension to the server capabilities (for server/discover).
func Declare(caps *mcp.ServerCapabilities) {
	caps.AddExtension(Extension, map[string]any{})
}

// Store keeps the tasks of one server. The zero value is ready to use.
type Store struct {
	// TTL is reported as ttlMs; finished tasks are dropped after it.
	TTL time.Duration
	// PollInterval is suggested to clients as pollIntervalMs.
	PollInterval time.Duration

	mu    sync.Mutex
	tasks map[string]*task
}

// NewStore returns a store with a TTL of one hour and a poll interval of one second.
func NewStore() *Store {
	return &Store{TTL: defaultTTL, PollInterval: defaultPollInterval, tasks: map[string]*task{}}
}

type task struct {
	id        string
	created   time.Time
	cancel    context.CancelFunc
	mu        sync.Mutex
	status    string
	message   string
	updated   time.Time
	result    json.RawMessage
	err       *jsonrpc.Error
	requests  mcp.InputRequestMap
	responses chan mcp.InputResponseMap
	answered  map[string]bool
}

// Enable registers the middleware and the tasks/* methods; toolNames are the
// tools that may run as a task.
func Enable(s *mcp.Server, store *Store, toolNames ...string) error {
	s.AddReceivingMiddleware(store.middleware(toolNames))
	if err := mcp.AddReceivingCustomMethod(s, methodTasksGet, store.get); err != nil {
		return err
	}
	if err := mcp.AddReceivingCustomMethod(s, methodTasksUpdate, store.update); err != nil {
		return err
	}
	return mcp.AddReceivingCustomMethod(s, methodTasksCancel, store.cancelTask)
}

// wire types

// TaskParams are the params of tasks/get and tasks/cancel.
type TaskParams struct {
	mcp.ParamsBase
	TaskID string `json:"taskId"`
}

// UpdateParams are the params of tasks/update.
type UpdateParams struct {
	mcp.ParamsBase
	TaskID         string          `json:"taskId"`
	InputResponses json.RawMessage `json:"inputResponses"`
}

// TaskResult is a Task as returned by tools/call (resultType "task") and
// tasks/get (resultType "complete", with the status-specific fields).
type TaskResult struct {
	mcp.ResultBase
	ResultType     string              `json:"resultType"`
	TaskID         string              `json:"taskId"`
	Status         string              `json:"status"`
	StatusMessage  string              `json:"statusMessage,omitempty"`
	CreatedAt      string              `json:"createdAt"`
	LastUpdatedAt  string              `json:"lastUpdatedAt"`
	TTLMs          *int64              `json:"ttlMs"`
	PollIntervalMs int64               `json:"pollIntervalMs,omitempty"`
	Result         json.RawMessage     `json:"result,omitempty"`
	Error          *jsonrpc.Error      `json:"error,omitempty"`
	InputRequests  mcp.InputRequestMap `json:"inputRequests,omitempty"`
}

// AckResult is the empty acknowledgement of tasks/update and tasks/cancel.
type AckResult struct {
	mcp.ResultBase
	ResultType string `json:"resultType"`
}

// ctxKey carries the running task into the tool handler for RequestInput.
type ctxKey struct{}

// RequestInput asks the client for input while a task runs: the task turns
// input_required with requests until the client answers via tasks/update.
// Outside a task it returns an error, so the tool can fall back (e.g. to a
// multi round-trip InputRequiredResult). Keys must be unique over the task's
// lifetime.
func RequestInput(ctx context.Context, requests mcp.InputRequestMap) (mcp.InputResponseMap, error) {
	t, ok := ctx.Value(ctxKey{}).(*task)
	if !ok {
		return nil, errors.New("mcptasks: RequestInput outside a task")
	}
	t.mu.Lock()
	for key := range requests {
		if t.answered[key] {
			t.mu.Unlock()
			return nil, fmt.Errorf("mcptasks: input request key %q was already used", key)
		}
	}
	t.requests = requests
	t.setStatus(InputRequired, "waiting for client input")
	t.mu.Unlock()

	select {
	case responses := <-t.responses:
		return responses, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsTask reports whether ctx belongs to a tool call running as a task.
func IsTask(ctx context.Context) bool {
	_, ok := ctx.Value(ctxKey{}).(*task)
	return ok
}

func (s *Store) middleware(toolNames []string) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != methodToolsCall {
				return next(ctx, method, req)
			}
			params, ok := req.GetParams().(*mcp.CallToolParamsRaw)
			// The client must declare the extension on this very request, and the
			// multi round-trip retry of a call is answered synchronously
			if !ok || !slices.Contains(toolNames, params.Name) || !declaresTasks(params.Meta) || params.InputResponses != nil {
				return next(ctx, method, req)
			}
			return s.start(ctx, method, req, next), nil
		}
	}
}

// start runs the tool call in the background and returns its task handle.
func (s *Store) start(ctx context.Context, method string, req mcp.Request, next mcp.MethodHandler) *TaskResult {
	now := time.Now()
	t := &task{
		id:        newTaskID(),
		created:   now,
		updated:   now,
		status:    Working,
		message:   "the operation is in progress",
		responses: make(chan mcp.InputResponseMap, 1),
		answered:  map[string]bool{},
	}
	// The task outlives the request that created it
	runCtx, cancel := context.WithCancel(context.WithValue(context.WithoutCancel(ctx), ctxKey{}, t))
	t.cancel = cancel

	s.mu.Lock()
	if s.tasks == nil {
		s.tasks = map[string]*task{}
	}
	s.dropExpired(now)
	s.tasks[t.id] = t // durably created before the handle is returned
	s.mu.Unlock()

	go func() {
		defer cancel()
		res, err := next(runCtx, method, req)
		t.mu.Lock()
		defer t.mu.Unlock()
		if t.status == Cancelled {
			return
		}
		if err != nil {
			// Only JSON-RPC errors make a task failed; anything else is internal
			var wire *jsonrpc.Error
			if !errors.As(err, &wire) {
				wire = &jsonrpc.Error{Code: codeInternalError, Message: err.Error()}
			}
			t.err = wire
			t.setStatus(Failed, wire.Message)
			return
		}
		data, merr := json.Marshal(res)
		if merr != nil {
			t.err = &jsonrpc.Error{Code: codeInternalError, Message: "marshaling result: " + merr.Error()}
			t.setStatus(Failed, t.err.Message)
			return
		}
		t.result = data
		t.setStatus(Completed, "the operation completed")
	}()

	return s.view(t, resultTypeTask)
}

// setStatus must be called with t.mu held.
func (t *task) setStatus(status, message string) {
	t.status, t.message, t.updated = status, message, time.Now()
	if status != InputRequired {
		t.requests = nil
	}
}

func (s *Store) ttl() time.Duration {
	if s.TTL <= 0 {
		return defaultTTL
	}
	return s.TTL
}

func (s *Store) pollInterval() time.Duration {
	if s.PollInterval <= 0 {
		return defaultPollInterval
	}
	return s.PollInterval
}

func (s *Store) view(t *task, resultType string) *TaskResult {
	ttl := s.ttl().Milliseconds()
	t.mu.Lock()
	defer t.mu.Unlock()
	res := &TaskResult{
		ResultType:     resultType,
		TaskID:         t.id,
		Status:         t.status,
		StatusMessage:  t.message,
		CreatedAt:      t.created.UTC().Format(time.RFC3339),
		LastUpdatedAt:  t.updated.UTC().Format(time.RFC3339),
		TTLMs:          &ttl,
		PollIntervalMs: s.pollInterval().Milliseconds(),
	}
	if resultType == resultComplete {
		// tasks/get carries the status-specific payload
		switch t.status {
		case Completed:
			res.Result = t.result
		case Failed:
			res.Error = t.err
		case InputRequired:
			res.InputRequests = t.requests
		}
	}
	return res
}

func (s *Store) lookup(meta mcp.Meta, taskID string) (*task, error) {
	if !declaresTasks(meta) {
		return nil, &jsonrpc.Error{
			Code:    codeMissingRequiredCapability,
			Message: "Missing required client capability",
			Data:    json.RawMessage(`{"requiredCapabilities":{"extensions":{"` + Extension + `":{}}}}`),
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropExpired(time.Now())
	t, ok := s.tasks[taskID]
	if !ok {
		return nil, &jsonrpc.Error{Code: codeInvalidParams, Message: fmt.Sprintf("Failed to retrieve task: task %q not found", taskID)}
	}
	return t, nil
}

func (s *Store) get(ctx context.Context, _ *mcp.ServerSession, p *TaskParams) (*TaskResult, error) {
	t, err := s.lookup(p.Meta, p.TaskID)
	if err != nil {
		return nil, err
	}
	return s.view(t, resultComplete), nil
}

func (s *Store) update(ctx context.Context, _ *mcp.ServerSession, p *UpdateParams) (*AckResult, error) {
	t, err := s.lookup(p.Meta, p.TaskID)
	if err != nil {
		return nil, err
	}
	var responses mcp.InputResponseMap
	if err := json.Unmarshal(p.InputResponses, &responses); err != nil {
		return nil, &jsonrpc.Error{Code: codeInvalidParams, Message: "invalid inputResponses: " + err.Error()}
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	// Answers to keys that are not outstanding are ignored
	accepted := mcp.InputResponseMap{}
	for key, resp := range responses {
		if _, pending := t.requests[key]; pending && !t.answered[key] {
			accepted[key] = resp
		}
	}
	if len(accepted) > 0 && len(accepted) == len(t.requests) {
		for key := range accepted {
			t.answered[key] = true
		}
		t.setStatus(Working, "input received, the operation continues")
		t.responses <- accepted
	}
	return &AckResult{ResultType: resultComplete}, nil
}

func (s *Store) cancelTask(ctx context.Context, _ *mcp.ServerSession, p *TaskParams) (*AckResult, error) {
	t, err := s.lookup(p.Meta, p.TaskID)
	if err != nil {
		return nil, err
	}
	t.mu.Lock()
	if t.status == Working || t.status == InputRequired {
		t.setStatus(Cancelled, "cancelled by the client")
	}
	t.mu.Unlock()
	t.cancel()
	return &AckResult{ResultType: resultComplete}, nil
}

// dropExpired must be called with s.mu held.
func (s *Store) dropExpired(now time.Time) {
	for id, t := range s.tasks {
		if now.Sub(t.created) > s.ttl() {
			t.cancel()
			delete(s.tasks, id)
		}
	}
}

// declaresTasks reports whether request metadata declares the extension in
// the per-request client capabilities.
func declaresTasks(meta mcp.Meta) bool {
	caps, _ := meta[metaKeyClientCapabilities].(map[string]any)
	if caps == nil {
		// Typed capabilities, depending on how the params were decoded
		data, err := json.Marshal(meta[metaKeyClientCapabilities])
		if err != nil || json.Unmarshal(data, &caps) != nil {
			return false
		}
	}
	ext, _ := caps["extensions"].(map[string]any)
	_, ok := ext[Extension]
	return ok
}

// newTaskID returns an unguessable id (task ids may act as bearer tokens).
func newTaskID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return hex.EncodeToString(b)
}
