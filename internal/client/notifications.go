package client

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Notification methods a client receives via subscriptions/listen.
const (
	ToolsListChanged     = "notifications/tools/list_changed"
	PromptsListChanged   = "notifications/prompts/list_changed"
	ResourcesListChanged = "notifications/resources/list_changed"
	ResourceUpdated      = "notifications/resources/updated"
)

// NotificationMethods are the methods Notifications can wait for.
var NotificationMethods = []string{ToolsListChanged, PromptsListChanged, ResourcesListChanged, ResourceUpdated}

// Notification is one received server notification.
type Notification struct {
	Method string
	URI    string // only for resources/updated
}

// Notifications records list-changed and resource-updated notifications.
// Installing its handlers makes the SDK open a subscriptions/listen stream
// for every list the server marks as listChanged.
type Notifications struct {
	// Out receives one line per notification.
	Out io.Writer

	mu       sync.Mutex
	received []Notification
}

// Install sets the notification handlers on opts.
func (n *Notifications) Install(opts *mcp.ClientOptions) {
	opts.ToolListChangedHandler = func(context.Context, *mcp.ToolListChangedRequest) { n.add(Notification{Method: ToolsListChanged}) }
	opts.PromptListChangedHandler = func(context.Context, *mcp.PromptListChangedRequest) { n.add(Notification{Method: PromptsListChanged}) }
	opts.ResourceListChangedHandler = func(context.Context, *mcp.ResourceListChangedRequest) {
		n.add(Notification{Method: ResourcesListChanged})
	}
	opts.ResourceUpdatedHandler = func(ctx context.Context, req *mcp.ResourceUpdatedNotificationRequest) {
		n.add(Notification{Method: ResourceUpdated, URI: req.Params.URI})
	}
}

func (n *Notifications) add(note Notification) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.received = append(n.received, note)
	if n.Out != nil {
		if note.URI != "" {
			fmt.Fprintf(n.Out, "[NOTIFY] %s %s\n", note.Method, note.URI)
		} else {
			fmt.Fprintf(n.Out, "[NOTIFY] %s\n", note.Method)
		}
	}
}

// Wait blocks until a notification with method (and uri, if not empty)
// arrives or timeout passes, and consumes it. Notifications received before
// the call count, so a wait after the triggering call cannot miss one.
func (n *Notifications) Wait(ctx context.Context, method, uri string, timeout time.Duration) error {
	if !slices.Contains(NotificationMethods, method) {
		return fmt.Errorf("unknown notification %q, expected one of %v", method, NotificationMethods)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		if n.take(method, uri) {
			return nil
		}
		select {
		case <-ctx.Done():
			if uri != "" {
				return fmt.Errorf("no %s for %s within %s", method, uri, timeout)
			}
			return fmt.Errorf("no %s within %s", method, timeout)
		case <-tick.C:
		}
	}
}

func (n *Notifications) take(method, uri string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i, note := range n.received {
		if note.Method == method && (uri == "" || note.URI == uri) {
			n.received = slices.Delete(n.received, i, i+1)
			return true
		}
	}
	return false
}
