package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// taskServer is a server with task-capable tools, connected in memory.
func taskServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	caps := &mcp.ServerCapabilities{}
	mcptasks.Declare(caps)
	s := mcp.NewServer(&mcp.Implementation{Name: "tasks", Version: "1"}, &mcp.ServerOptions{Capabilities: caps})

	text := func(s string) *mcp.CallToolResult {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
	}
	mcp.AddTool(s, &mcp.Tool{Name: "slow"}, func(ctx context.Context, r *mcp.CallToolRequest, a struct{}) (*mcp.CallToolResult, any, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return text("done"), nil, nil
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	})
	mcp.AddTool(s, &mcp.Tool{Name: "ask"}, func(ctx context.Context, r *mcp.CallToolRequest, a struct{}) (*mcp.CallToolResult, any, error) {
		resp, err := mcptasks.RequestInput(ctx, mcp.InputRequestMap{
			"name": &mcp.ElicitParams{Mode: "form", Message: "Your name?", RequestedSchema: map[string]any{"type": "object"}},
		})
		if err != nil {
			return nil, nil, err
		}
		er, _ := resp["name"].(*mcp.ElicitResult)
		return text("Hello " + er.Content["name"].(string)), nil, nil
	})
	mcp.AddTool(s, &mcp.Tool{Name: "broken"}, func(ctx context.Context, r *mcp.CallToolRequest, a struct{}) (*mcp.CallToolResult, any, error) {
		return nil, nil, &jsonrpc.Error{Code: -32603, Message: "backend down"}
	})
	if err := mcptasks.Enable(s, &mcptasks.Store{TTL: time.Minute, PollInterval: 10 * time.Millisecond}, "slow", "ask", "broken"); err != nil {
		t.Fatal(err)
	}

	ct, st := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func TestTaskLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cs := taskServer(t)
	responder := &Responder{Strict: true}
	tc := &TaskClient{Session: cs, Responder: responder}

	t.Run("completes", func(t *testing.T) {
		handle, done, err := tc.Start(ctx, "slow", nil)
		if err != nil || done {
			t.Fatalf("Start: done=%v err=%v; want a task handle", done, err)
		}
		if handle["status"] != "working" {
			t.Errorf("seed status %v, want working", handle["status"])
		}
		task, err := tc.Wait(ctx, handle["taskId"].(string))
		if err != nil {
			t.Fatal(err)
		}
		if task["resultType"] != "complete" {
			t.Errorf("tasks/get resultType %v, want complete", task["resultType"])
		}
		result, err := TaskResult(task)
		if err != nil || resultText(result) != "done" {
			t.Errorf("result %v, err %v", result, err)
		}
	})

	t.Run("input required", func(t *testing.T) {
		responder.QueueElicit(&mcp.ElicitResult{Action: "accept", Content: map[string]any{"name": "Ada"}})
		handle, _, err := tc.Start(ctx, "ask", nil)
		if err != nil {
			t.Fatal(err)
		}
		task, err := tc.Wait(ctx, handle["taskId"].(string))
		if err != nil {
			t.Fatal(err)
		}
		result, err := TaskResult(task)
		if err != nil || resultText(result) != "Hello Ada" {
			t.Errorf("result %v, err %v", result, err)
		}
		if responder.LastElicitMessage() != "Your name?" {
			t.Errorf("elicitation %q not seen", responder.LastElicitMessage())
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		handle, _, err := tc.Start(ctx, "slow", nil)
		if err != nil {
			t.Fatal(err)
		}
		id := handle["taskId"].(string)
		if err := tc.Cancel(ctx, id); err != nil {
			t.Fatal(err)
		}
		task, err := tc.Wait(ctx, id)
		if err != nil || task["status"] != "cancelled" {
			t.Errorf("status %v, err %v; want cancelled", task["status"], err)
		}
	})

	t.Run("failed", func(t *testing.T) {
		handle, _, err := tc.Start(ctx, "broken", nil)
		if err != nil {
			t.Fatal(err)
		}
		task, err := tc.Wait(ctx, handle["taskId"].(string))
		if err != nil {
			t.Fatal(err)
		}
		_, err = TaskResult(task)
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32603 {
			t.Errorf("failed task error %v, want RPC -32603", err)
		}
	})

	t.Run("unknown task", func(t *testing.T) {
		_, err := tc.Get(ctx, "no-such-task")
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32602 {
			t.Errorf("error %v, want RPC -32602", err)
		}
	})

	t.Run("without the extension", func(t *testing.T) {
		// The SDK client does not declare tasks: the server answers synchronously
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "slow"})
		if err != nil || len(res.Content) == 0 {
			t.Fatalf("synchronous call: %v, %v", res, err)
		}
		// and tasks/get without the capability is -32021
		_, err = CallRaw(ctx, cs, "tasks/get", map[string]any{"taskId": "x", "_meta": map[string]any{
			mcp.MetaKeyProtocolVersion: cs.InitializeResult().ProtocolVersion, mcp.MetaKeyClientCapabilities: map[string]any{},
		}})
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32021 {
			t.Errorf("error %v, want RPC -32021", err)
		}
	})
}

// resultText joins the text content of a raw CallToolResult.
func resultText(result map[string]any) string {
	text := ""
	content, _ := result["content"].([]any)
	for _, c := range content {
		if m, ok := c.(map[string]any); ok {
			s, _ := m["text"].(string)
			text += s
		}
	}
	return text
}
