package scripting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultNotificationTimeout = 5 * time.Second

// handleSubscribeCommand runs "subscribe <uri>": resource updates for uri are
// delivered via subscriptions/listen.
func (r *Runner) handleSubscribeCommand(ctx context.Context, i int, parts []string) error {
	if len(parts) != 2 {
		return fmt.Errorf("line %d: usage: subscribe <uri>", i+1)
	}
	if err := r.session.Subscribe(ctx, &mcp.SubscribeParams{URI: parts[1]}); err != nil {
		return fmt.Errorf("line %d: subscribe: %w", i+1, err)
	}
	return nil
}

// handleWaitNotificationCommand runs "wait_notification <method> [uri] [timeout]",
// e.g. "wait_notification tools/list_changed" or
// "wait_notification resources/updated mcp://time 2s".
func (r *Runner) handleWaitNotificationCommand(ctx context.Context, i int, parts []string) error {
	if len(parts) < 2 || len(parts) > 4 {
		return fmt.Errorf("line %d: usage: wait_notification <method> [uri] [timeout]", i+1)
	}
	if r.Notifications == nil {
		return fmt.Errorf("line %d: this runner does not record notifications", i+1)
	}
	method := parts[1]
	if !strings.HasPrefix(method, "notifications/") {
		method = "notifications/" + method
	}
	uri, timeout := "", defaultNotificationTimeout
	for _, arg := range parts[2:] {
		if d, err := time.ParseDuration(arg); err == nil {
			timeout = d
		} else {
			uri = arg
		}
	}
	if err := r.Notifications.Wait(ctx, method, uri, timeout); err != nil {
		return fmt.Errorf("line %d: assertion failed: %w", i+1, err)
	}
	fmt.Fprint(r.w(), i18n.T(i18n.MsgAssertionPassed, "received "+method))
	return nil
}
