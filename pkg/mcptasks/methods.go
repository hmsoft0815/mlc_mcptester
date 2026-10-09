package mcptasks

// The tasks/* methods: get, update, cancel.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Store) lookup(meta mcp.Meta, taskID string) (*task, error) {
	if !declaresTasks(meta) {
		return nil, missingCapability()
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
	// Answers to keys that are not outstanding are ignored; a partial set is
	// kept, and the task stays input_required until the rest arrives
	for key, resp := range responses {
		if _, pending := t.requests[key]; pending && !t.answered[key] {
			t.collected[key] = resp
			t.answered[key] = true
			delete(t.requests, key)
		}
	}
	if t.status == InputRequired && len(t.requests) == 0 {
		t.setStatus(Working, "input received, the operation continues")
		t.responses <- t.collected
		t.collected = nil
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
