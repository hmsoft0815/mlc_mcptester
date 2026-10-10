package client

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNotificationTapRecordsOnlyItsNotifications(t *testing.T) {
	tap := NewNotificationTap("notifications/tasks")
	id, _ := jsonrpc.MakeID(float64(1))
	for _, msg := range []jsonrpc.Message{
		&jsonrpc.Request{Method: "notifications/tasks", Params: json.RawMessage(`{"taskId":"a","status":"working"}`)},
		&jsonrpc.Request{Method: "notifications/progress", Params: json.RawMessage(`{}`)},      // other method
		&jsonrpc.Request{ID: id, Method: "notifications/tasks", Params: json.RawMessage(`{}`)}, // a call, not a notification
		&jsonrpc.Request{Method: "notifications/tasks", Params: json.RawMessage(`not json`)},   // unreadable params
		&jsonrpc.Response{ID: id, Result: json.RawMessage(`{}`)},                               // a response
		&jsonrpc.Request{Method: "notifications/tasks", Params: json.RawMessage(`{"taskId":"a","status":"completed"}`)},
	} {
		tap.record(msg)
	}
	got := tap.Received("notifications/tasks")
	if len(got) != 2 || got[0]["status"] != "working" || got[1]["status"] != "completed" {
		t.Fatalf("recorded %v", got)
	}
	if len(tap.Received("notifications/progress")) != 0 {
		t.Error("recorded a method the tap was not asked for")
	}
}

// A raw request abandoned on timeout must reach the server as
// notifications/cancelled, so the server stops the work.
func TestCallRawCancelReachesServer(t *testing.T) {
	cancelled := make(chan struct{})
	s := mcp.NewServer(&mcp.Implementation{Name: "slow", Version: "1"}, nil)
	mcp.AddTool(s, &mcp.Tool{Name: "wait", InputSchema: map[string]any{"type": "object"}}, func(ctx context.Context, r *mcp.CallToolRequest, a map[string]any) (*mcp.CallToolResult, any, error) {
		<-ctx.Done()
		close(cancelled)
		return nil, nil, ctx.Err()
	})
	ct, st := mcp.NewInMemoryTransports()
	if _, err := s.Connect(context.Background(), st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = CallRaw(ctx, cs, "tools/call", map[string]any{"name": "wait", "arguments": map[string]any{}, "_meta": RequestMeta(cs, nil)})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CallRaw: %v, want deadline exceeded", err)
	}
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("the server kept working: notifications/cancelled did not arrive")
	}
}
