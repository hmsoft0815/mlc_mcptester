package mcptasks_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func question(msg string) *mcp.ElicitParams {
	return &mcp.ElicitParams{Mode: "form", Message: msg, RequestedSchema: map[string]any{"type": "object"}}
}

// connect serves tools "slow" (ms) and "two" (asks two questions at once)
// as tasks and returns a client session.
func connect(t *testing.T, store *mcptasks.Store) *mcp.ClientSession {
	t.Helper()
	caps := &mcp.ServerCapabilities{}
	mcptasks.Declare(caps)
	s := mcp.NewServer(&mcp.Implementation{Name: "tasks", Version: "1"}, &mcp.ServerOptions{Capabilities: caps})
	mcp.AddTool(s, &mcp.Tool{Name: "slow"}, func(ctx context.Context, r *mcp.CallToolRequest, a struct {
		MS int `json:"ms"`
	}) (*mcp.CallToolResult, any, error) {
		select {
		case <-time.After(time.Duration(a.MS) * time.Millisecond):
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "done"}}}, nil, nil
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	})
	mcp.AddTool(s, &mcp.Tool{Name: "two"}, func(ctx context.Context, r *mcp.CallToolRequest, a struct{}) (*mcp.CallToolResult, any, error) {
		resp, err := mcptasks.RequestInput(ctx, mcp.InputRequestMap{"a": question("A?"), "b": question("B?")})
		if err != nil {
			return nil, nil, err
		}
		ra, _ := resp["a"].(*mcp.ElicitResult)
		rb, _ := resp["b"].(*mcp.ElicitResult)
		if ra == nil || rb == nil {
			return nil, nil, fmt.Errorf("answers missing: %v", resp)
		}
		text := fmt.Sprintf("%v %v", ra.Content["v"], rb.Content["v"])
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil, nil
	})
	if err := mcptasks.Enable(s, store, "slow", "two"); err != nil {
		t.Fatal(err)
	}
	ct, st := mcp.NewInMemoryTransports()
	if _, err := s.Connect(context.Background(), st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// call sends a request; declare puts the extension into the per-request capabilities.
func call(t *testing.T, cs *mcp.ClientSession, method string, params map[string]any, declare bool) (map[string]any, error) {
	t.Helper()
	caps := map[string]any{}
	if declare {
		caps["extensions"] = map[string]any{mcptasks.Extension: map[string]any{}}
		caps["elicitation"] = map[string]any{"form": map[string]any{}}
	}
	params["_meta"] = client.RequestMeta(cs, caps)
	return client.CallRaw(context.Background(), cs, method, params)
}

func start(t *testing.T, cs *mcp.ClientSession, tool string, args map[string]any) string {
	t.Helper()
	res, err := call(t, cs, "tools/call", map[string]any{"name": tool, "arguments": args}, true)
	if err != nil || res["resultType"] != "task" {
		t.Fatalf("no task: %v %v", res, err)
	}
	return res["taskId"].(string)
}

// waitFor polls until the task has the status.
func waitFor(t *testing.T, cs *mcp.ClientSession, id, status string) map[string]any {
	t.Helper()
	for range 200 {
		res, err := call(t, cs, "tasks/get", map[string]any{"taskId": id}, true)
		if err != nil {
			t.Fatal(err)
		}
		if res["status"] == status {
			return res
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("task %s never reached %s", id, status)
	return nil
}

func answer(v string) map[string]any {
	return map[string]any{"action": "accept", "content": map[string]any{"v": v}}
}

func update(t *testing.T, cs *mcp.ClientSession, id string, responses map[string]any) {
	t.Helper()
	res, err := call(t, cs, "tasks/update", map[string]any{"taskId": id, "inputResponses": responses}, true)
	if err != nil || res["resultType"] != "complete" {
		t.Fatalf("tasks/update: %v %v", res, err)
	}
}

func TestPartialInputResponses(t *testing.T) {
	cs := connect(t, mcptasks.NewStore())
	id := start(t, cs, "two", nil)
	waitFor(t, cs, id, mcptasks.InputRequired)

	update(t, cs, id, map[string]any{"a": answer("x"), "unknown": answer("ignored")})
	res := waitFor(t, cs, id, mcptasks.InputRequired)
	reqs, _ := res["inputRequests"].(map[string]any)
	if _, ok := reqs["a"]; ok || len(reqs) != 1 {
		t.Fatalf("after a partial answer only b may be outstanding: %v", reqs)
	}

	// a again is already answered and must be ignored
	update(t, cs, id, map[string]any{"a": answer("again"), "b": answer("y")})
	res = waitFor(t, cs, id, mcptasks.Completed)
	got, err := client.TaskResult(res)
	if err != nil {
		t.Fatal(err)
	}
	if text := got["content"].([]any)[0].(map[string]any)["text"]; text != "x y" {
		t.Errorf("result %q, want \"x y\"", text)
	}
}

func TestExpiredTaskIsGone(t *testing.T) {
	cs := connect(t, &mcptasks.Store{TTL: 50 * time.Millisecond, PollInterval: time.Millisecond})
	id := start(t, cs, "slow", map[string]any{"ms": 1})
	waitFor(t, cs, id, mcptasks.Completed)
	time.Sleep(80 * time.Millisecond)
	_, err := call(t, cs, "tasks/get", map[string]any{"taskId": id}, true)
	var rpc *client.RPCError
	if !errors.As(err, &rpc) || rpc.Code != -32602 {
		t.Errorf("expired task: %v, want -32602", err)
	}
}

func TestMissingCapability(t *testing.T) {
	cs := connect(t, mcptasks.NewStore())
	res, err := call(t, cs, "tools/call", map[string]any{"name": "slow", "arguments": map[string]any{"ms": 1}}, false)
	if err != nil || res["resultType"] == "task" {
		t.Errorf("a client without the extension got %v %v, want a plain result", res, err)
	}
	for _, method := range []string{"tasks/get", "tasks/update", "tasks/cancel"} {
		_, err := call(t, cs, method, map[string]any{"taskId": "x", "inputResponses": map[string]any{}}, false)
		var rpc *client.RPCError
		if !errors.As(err, &rpc) || rpc.Code != -32021 {
			t.Errorf("%s without the extension: %v, want -32021", method, err)
		}
	}
}

func TestCancelledStaysCancelled(t *testing.T) {
	cs := connect(t, mcptasks.NewStore())
	id := start(t, cs, "slow", map[string]any{"ms": 10000})
	if _, err := call(t, cs, "tasks/cancel", map[string]any{"taskId": id}, true); err != nil {
		t.Fatal(err)
	}
	waitFor(t, cs, id, mcptasks.Cancelled)
	time.Sleep(20 * time.Millisecond) // the tool has returned ctx.Err() by now
	waitFor(t, cs, id, mcptasks.Cancelled)
}
