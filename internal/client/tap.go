package client

import (
	"context"
	"encoding/json"
	"slices"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NotificationTap records server notifications the go-sdk does not know
// (notifications/tasks) or keeps to itself (notifications/subscriptions/
// acknowledged), by reading the raw messages below the SDK. The messages are
// passed on unchanged.
type NotificationTap struct {
	methods []string

	mu       sync.Mutex
	received []TappedNotification
}

// TappedNotification is one recorded notification.
type TappedNotification struct {
	Method string
	Params map[string]any
}

// NewNotificationTap records notifications with one of the methods.
func NewNotificationTap(methods ...string) *NotificationTap {
	return &NotificationTap{methods: methods}
}

// Wrap returns t with the tap installed on its connections.
func (n *NotificationTap) Wrap(t mcp.Transport) mcp.Transport { return tapTransport{t, n} }

// Received returns the notifications recorded so far with the method.
func (n *NotificationTap) Received(method string) []map[string]any {
	n.mu.Lock()
	defer n.mu.Unlock()
	var out []map[string]any
	for _, r := range n.received {
		if r.Method == method {
			out = append(out, r.Params)
		}
	}
	return out
}

func (n *NotificationTap) record(msg jsonrpc.Message) {
	req, ok := msg.(*jsonrpc.Request)
	if !ok || req.IsCall() || !slices.Contains(n.methods, req.Method) {
		return
	}
	var params map[string]any
	if json.Unmarshal(req.Params, &params) != nil {
		return
	}
	n.mu.Lock()
	n.received = append(n.received, TappedNotification{Method: req.Method, Params: params})
	n.mu.Unlock()
}

type tapTransport struct {
	mcp.Transport
	n *NotificationTap
}

func (t tapTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	conn, err := t.Transport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return tapConn{conn, t.n}, nil
}

type tapConn struct {
	mcp.Connection
	n *NotificationTap
}

func (c tapConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	msg, err := c.Connection.Read(ctx)
	if err == nil {
		c.n.record(msg)
	}
	return msg, err
}
