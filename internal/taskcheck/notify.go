package taskcheck

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
)

// Notifications of the extension, recorded by a client.NotificationTap.
const (
	NotificationTasks = "notifications/tasks"
	NotificationAcked = "notifications/subscriptions/acknowledged"
)

const (
	// listenWait bounds the wait for a refusal or an acknowledgement; both
	// are the first answer on a listen stream.
	listenWait = 2 * time.Second
	// notifyGrace is how long a terminal notification may trail tasks/get.
	notifyGrace = time.Second
)

// listenParams asks for notifications/tasks of the tasks.
func listenParams(ids ...string) map[string]any {
	return map[string]any{"notifications": map[string]any{"taskIds": ids}}
}

// checkListenWithoutCapability: asking for task notifications without the
// extension MUST be refused with -32021.
func (c *Checker) checkListenWithoutCapability(ctx context.Context, rep *Report) {
	const name = "subscriptions/listen for tasks without the extension"
	ctx, cancel := context.WithTimeout(ctx, listenWait)
	defer cancel()
	_, err := c.call(ctx, "subscriptions/listen", listenParams(unknownTaskID), false)
	code, rpcErr := errorCode(err)
	switch {
	case err == nil:
		rep.add(name, Fail, "accepted (the stream ended without error), must fail with %d", codeMissingRequiredCapability)
	case errors.Is(err, context.DeadlineExceeded):
		rep.add(name, Fail, "the stream stayed open, must fail with %d", codeMissingRequiredCapability)
	case !rpcErr:
		rep.add(name, Fail, "%v", err)
	case code != codeMissingRequiredCapability:
		rep.add(name, Fail, "error %d, must be %d (Missing Required Client Capability)", code, codeMissingRequiredCapability)
	case !namesExtension(err):
		rep.add(name, Warn, "%d without data.requiredCapabilities.extensions naming %s", code, client.TasksExtension)
	default:
		rep.add(name, Pass, "%d", code)
	}
}

// listener follows the notifications/tasks of one task.
type listener struct {
	c      *Checker
	id     string
	cancel context.CancelFunc
	agreed bool
}

// listen subscribes to the task's notifications. Without a tap, or if the
// server does not agree to send them (MAY), it reports a skip and returns nil.
func (c *Checker) listen(ctx context.Context, rep *Report, id string) *listener {
	const name = "task notifications"
	if c.Tap == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		_, err := c.call(ctx, "subscriptions/listen", listenParams(id), true)
		done <- err
	}()
	l := &listener{c: c, id: id, cancel: cancel}
	deadline := time.After(listenWait)
	for !l.agreed {
		select {
		case err := <-done:
			if l.agreed = l.acked(); !l.agreed {
				cancel()
				rep.add(name, Skip, "not offered (MAY): the listen stream ended without acknowledging the task (%v)", errOrNone(err))
				return nil
			}
		case <-deadline:
			cancel()
			rep.add(name, Skip, "not offered (MAY): no acknowledgement naming the task within %s", listenWait)
			return nil
		case <-time.After(20 * time.Millisecond):
			l.agreed = l.acked()
		}
	}
	rep.add(name, Pass, "subscribed, the acknowledgement names the task")
	return l
}

// acked reports whether an acknowledgement lists the task id.
func (l *listener) acked() bool {
	for _, ack := range l.c.Tap.Received(NotificationAcked) {
		notes, _ := ack["notifications"].(map[string]any)
		ids, _ := notes["taskIds"].([]any)
		if slices.Contains(ids, any(l.id)) {
			return true
		}
	}
	return false
}

// finish checks the notifications received against the final status from
// tasks/get, then closes the stream.
func (l *listener) finish(rep *Report, final string) {
	defer l.cancel()
	var notes []map[string]any
	deadline := time.Now().Add(notifyGrace)
	for {
		notes = l.notifications()
		if final == "" || hasStatus(notes, final) || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	bad := false
	for _, n := range notes {
		must, should := checkTask(n, "")
		for _, p := range must {
			bad = true
			rep.add("notifications/tasks", Fail, "%s", p)
		}
		for _, p := range should {
			rep.add("notifications/tasks", Warn, "%s", p)
		}
	}
	switch {
	case len(notes) == 0:
		rep.add("notifications/tasks", Warn, "the server agreed to notify, but sent none")
	case final != "" && !hasStatus(notes, final):
		rep.add("notifications/tasks", Warn, "%d notification(s), none for the final status %s", len(notes), final)
	case !bad:
		rep.add("notifications/tasks", Pass, "%d notification(s), well-formed, the last status matches tasks/get", len(notes))
	}
}

// notifications returns the recorded notifications/tasks of the task.
func (l *listener) notifications() []map[string]any {
	var out []map[string]any
	for _, n := range l.c.Tap.Received(NotificationTasks) {
		if n["taskId"] == l.id {
			out = append(out, n)
		}
	}
	return out
}

func hasStatus(notes []map[string]any, status string) bool {
	return slices.ContainsFunc(notes, func(n map[string]any) bool { return n["status"] == status })
}

func errOrNone(err error) any {
	if err == nil {
		return "no error"
	}
	return err
}
