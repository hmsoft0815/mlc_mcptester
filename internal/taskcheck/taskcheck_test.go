package taskcheck

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func text(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

// conformingServer has task-capable tools served by pkg/mcptasks.
func conformingServer(t *testing.T) *mcp.Server {
	t.Helper()
	caps := &mcp.ServerCapabilities{}
	mcptasks.Declare(caps)
	s := mcp.NewServer(&mcp.Implementation{Name: "tasks", Version: "1"}, &mcp.ServerOptions{Capabilities: caps})
	mcp.AddTool(s, &mcp.Tool{Name: "slow"}, func(ctx context.Context, r *mcp.CallToolRequest, a struct {
		MS int `json:"ms"`
	}) (*mcp.CallToolResult, any, error) {
		select {
		case <-time.After(time.Duration(a.MS) * time.Millisecond):
			return text("done"), nil, nil
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	})
	mcp.AddTool(s, &mcp.Tool{Name: "ask"}, func(ctx context.Context, r *mcp.CallToolRequest, a struct{}) (*mcp.CallToolResult, any, error) {
		if !mcptasks.IsTask(ctx) {
			return text("needs a task"), nil, nil
		}
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
	return s
}

func connect(t *testing.T, s *mcp.Server) *mcp.ClientSession {
	t.Helper()
	return connectVia(t, s, mcptasks.GuardListen, nil)
}

// connectVia wraps the server side of the connection with wrap and, with a
// tap, the client side too.
func connectVia(t *testing.T, s *mcp.Server, wrap func(mcp.Transport) mcp.Transport, tap *client.NotificationTap) *mcp.ClientSession {
	t.Helper()
	var ct, st mcp.Transport
	ct, st = mcp.NewInMemoryTransports()
	if tap != nil {
		ct = tap.Wrap(ct)
	}
	if _, err := s.Connect(context.Background(), wrap(st), nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func noFailures(t *testing.T, rep *Report) {
	t.Helper()
	for _, r := range rep.Results {
		if r.Status == Fail {
			t.Errorf("FAIL %s: %s", r.Name, r.Detail)
		}
	}
}

func result(rep *Report, name string) *Result {
	for i := range rep.Results {
		if rep.Results[i].Name == name {
			return &rep.Results[i]
		}
	}
	return nil
}

func TestConformingServer(t *testing.T) {
	answers := &client.Responder{}
	answers.QueueElicit(&mcp.ElicitResult{Action: "accept", Content: map[string]any{"name": "Ada"}})
	tests := []struct {
		name   string
		c      Checker
		final  string
		passes int // checks that must pass at least
	}{
		{"passive only", Checker{}, "", 7},
		{"completed", Checker{Tool: "slow", Args: map[string]any{"ms": 50}}, "completed", 13},
		{"failed", Checker{Tool: "broken"}, "failed", 12},
		{"cancelled", Checker{Tool: "slow", Args: map[string]any{"ms": 30000}, Cancel: true}, "cancelled", 13},
		{"input answered", Checker{Tool: "ask", Responder: answers}, "completed", 13},
		{"input without answers", Checker{Tool: "ask"}, "", 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.c
			c.Session, c.Timeout, c.PlainWait = connect(t, conformingServer(t)), 10*time.Second, 300*time.Millisecond
			rep := c.Run(context.Background())
			noFailures(t, rep)
			if !rep.Declared {
				t.Error("extension not seen as declared")
			}
			passes := 0
			for _, r := range rep.Results {
				if r.Status == Pass {
					passes++
				}
			}
			if passes < tt.passes {
				t.Errorf("%d checks passed, want at least %d: %+v", passes, tt.passes, rep.Results)
			}
			if tt.final == "" {
				return
			}
			if r := result(rep, "terminal status"); r == nil || r.Status != Pass || !strings.HasPrefix(r.Detail, tt.final) {
				t.Errorf("terminal status: %+v, want %s", r, tt.final)
			}
		})
	}
}

// brokenServer declares the extension, but answers every tasks/* request
// with an acknowledgement and hands out counter ids.
func brokenServer(t *testing.T) *mcp.Server {
	t.Helper()
	caps := &mcp.ServerCapabilities{}
	mcptasks.Declare(caps)
	s := mcp.NewServer(&mcp.Implementation{Name: "broken", Version: "1"}, &mcp.ServerOptions{Capabilities: caps})
	ack := func(ctx context.Context, _ *mcp.ServerSession, p *mcptasks.TaskParams) (*mcptasks.AckResult, error) {
		return &mcptasks.AckResult{ResultType: "complete"}, nil
	}
	for _, m := range []string{"tasks/get", "tasks/cancel"} {
		if err := mcp.AddReceivingCustomMethod(s, m, ack); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func TestBrokenServer(t *testing.T) {
	rep := (&Checker{Session: connect(t, brokenServer(t))}).Run(context.Background())
	if !rep.Failed() {
		t.Fatal("broken server passed")
	}
	if failed := strings.Join(rep.FailedNames(), ", "); !strings.Contains(failed, "tasks/get of an unknown task") || strings.Contains(failed, "tasks/cancel of an unknown task") {
		t.Errorf("FailedNames: %s", failed)
	}
	want := map[string]Status{
		"tasks/get without the extension":    Fail,
		"tasks/get of an unknown task":       Fail,
		"tasks/cancel of an unknown task":    Warn,
		"tasks/update of an unknown task":    Fail, // not implemented at all
		"tasks/update without the extension": Fail,
	}
	for name, status := range want {
		if r := result(rep, name); r == nil || r.Status != status {
			t.Errorf("%s: %+v, want %s", name, r, status)
		}
	}
}

func TestCheckTask(t *testing.T) {
	good := map[string]any{
		"resultType": "complete", "taskId": "786512e2-9e0d-44bd-8f29-789f320fe840", "status": "failed",
		"createdAt": "2025-11-25T10:30:00Z", "lastUpdatedAt": "2025-11-25T10:40:00.123Z",
		"ttlMs": nil, "pollIntervalMs": 5000.0, "statusMessage": "rate limit",
		"error": map[string]any{"code": -32603.0, "message": "rate limit"},
	}
	if must, should := checkTask(good, "complete"); len(must)+len(should) > 0 {
		t.Errorf("good task: %v %v", must, should)
	}
	bad := map[string]any{
		"resultType": "task", "taskId": "", "status": "running", "createdAt": "yesterday",
		"pollIntervalMs": 1.5,
	}
	must, _ := checkTask(bad, "complete")
	for _, want := range []string{"resultType", "taskId", "status", "createdAt", "lastUpdatedAt", "ttlMs", "pollIntervalMs"} {
		if !strings.Contains(strings.Join(must, "\n"), want) {
			t.Errorf("no problem reported for %s: %v", want, must)
		}
	}
	failed := map[string]any{"resultType": "complete", "taskId": "x", "status": "failed", "createdAt": "2025-11-25T10:30:00Z", "lastUpdatedAt": "2025-11-25T10:30:00Z", "ttlMs": 1.0}
	if must, should := checkTask(failed, "complete"); len(must) != 1 || len(should) != 1 {
		t.Errorf("failed without error/statusMessage: %v %v", must, should)
	}
}

func TestGuessableID(t *testing.T) {
	for id, guessable := range map[string]bool{"17": true, "123456789012345678": true, "abc": true, "786512e2-9e0d-44bd": false} {
		if got := guessableID(id) != ""; got != guessable {
			t.Errorf("%q: guessable %v, want %v", id, got, guessable)
		}
	}
}
