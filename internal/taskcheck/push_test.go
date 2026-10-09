package taskcheck

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// pushTransport makes a server send notifications/tasks: it answers a
// declared subscriptions/listen for task ids itself (acknowledgement, stream
// held open) and mirrors every tasks/get result of a subscribed task as a
// notification.
type pushTransport struct{ mcp.Transport }

func (t pushTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.Transport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &pushConn{Connection: conn, gets: map[any]bool{}, subscribed: map[string]bool{}}, nil
}

type pushConn struct {
	mcp.Connection
	mu         sync.Mutex
	gets       map[any]bool // ids of pending tasks/get requests
	subscribed map[string]bool
}

func (c *pushConn) notify(ctx context.Context, method string, params any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return c.Connection.Write(ctx, &jsonrpc.Request{Method: method, Params: raw})
}

func (c *pushConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	for {
		msg, err := c.Connection.Read(ctx)
		if err != nil {
			return nil, err
		}
		req, ok := msg.(*jsonrpc.Request)
		if !ok || !req.IsCall() {
			return msg, nil
		}
		if req.Method == "tasks/get" {
			c.mu.Lock()
			c.gets[req.ID.Raw()] = true
			c.mu.Unlock()
		}
		ids := listenTaskIDs(req)
		if len(ids) == 0 {
			return msg, nil
		}
		c.mu.Lock()
		for _, id := range ids {
			c.subscribed[id] = true
		}
		c.mu.Unlock()
		ack := map[string]any{"notifications": map[string]any{"taskIds": ids}}
		if err := c.notify(ctx, NotificationAcked, ack); err != nil {
			return nil, err
		}
	}
}

func (c *pushConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	if err := c.Connection.Write(ctx, msg); err != nil {
		return err
	}
	resp, ok := msg.(*jsonrpc.Response)
	if !ok || resp.Error != nil {
		return nil
	}
	c.mu.Lock()
	isGet := c.gets[resp.ID.Raw()]
	delete(c.gets, resp.ID.Raw())
	c.mu.Unlock()
	var task map[string]any
	if !isGet || json.Unmarshal(resp.Result, &task) != nil {
		return nil
	}
	c.mu.Lock()
	subscribed := c.subscribed[task["taskId"].(string)]
	c.mu.Unlock()
	if !subscribed {
		return nil
	}
	delete(task, "resultType")
	return c.notify(ctx, NotificationTasks, task)
}

// listenTaskIDs returns the task ids of a declared subscriptions/listen.
func listenTaskIDs(req *jsonrpc.Request) []string {
	if req.Method != "subscriptions/listen" {
		return nil
	}
	var p struct {
		Meta          mcp.Meta `json:"_meta"`
		Notifications struct {
			TaskIDs []string `json:"taskIds"`
		} `json:"notifications"`
	}
	if json.Unmarshal(req.Params, &p) != nil {
		return nil
	}
	caps, _ := p.Meta["io.modelcontextprotocol/clientCapabilities"].(map[string]any)
	ext, _ := caps["extensions"].(map[string]any)
	if _, ok := ext[mcptasks.Extension]; !ok {
		return nil
	}
	return p.Notifications.TaskIDs
}

func TestTaskNotifications(t *testing.T) {
	tests := []struct {
		name string
		wrap func(mcp.Transport) mcp.Transport
		want Status
	}{
		{"pushed", func(t mcp.Transport) mcp.Transport { return mcptasks.GuardListen(pushTransport{t}) }, Pass},
		{"not offered", mcptasks.GuardListen, Skip},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tap := client.NewNotificationTap(NotificationTasks, NotificationAcked)
			cs := connectVia(t, conformingServer(t), tt.wrap, tap)
			rep := (&Checker{Session: cs, Tap: tap, Tool: "slow", Args: map[string]any{"ms": 100}, PlainWait: 300 * time.Millisecond, Timeout: 10 * time.Second}).Run(context.Background())
			noFailures(t, rep)
			if r := result(rep, "task notifications"); r == nil || r.Status != tt.want {
				t.Fatalf("task notifications: %+v, want %s", r, tt.want)
			}
			if tt.want != Pass {
				return
			}
			if r := result(rep, "notifications/tasks"); r == nil || r.Status != Pass || !strings.Contains(r.Detail, "matches") {
				t.Errorf("notifications/tasks: %+v", r)
			}
			if !slices.ContainsFunc(tap.Received(NotificationTasks), func(n map[string]any) bool { return n["status"] == "completed" }) {
				t.Error("no completed notification recorded")
			}
		})
	}
}

func TestListenWithoutCapabilityUnguarded(t *testing.T) {
	// A plain go-sdk server drops taskIds and acknowledges: a MUST violation
	cs := connectVia(t, conformingServer(t), func(t mcp.Transport) mcp.Transport { return t }, nil)
	rep := (&Checker{Session: cs}).Run(context.Background())
	if r := result(rep, "subscriptions/listen for tasks without the extension"); r == nil || r.Status != Fail {
		t.Errorf("unguarded server: %+v, want FAIL", r)
	}
}
