package taskcheck

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
)

const (
	defaultTimeout = 2 * time.Minute
	// maxPoll caps the server's pollIntervalMs, so a slow suggestion does not
	// stretch a check run
	maxPoll = 5 * time.Second
	// A task handle comes at once; a call still running after this answers
	// synchronously, and is cancelled instead of awaited.
	defaultPlainWait = 5 * time.Second
)

// checkPlainCall: a server MUST NOT return a task to a client that did not
// declare the extension on the request; -32021 is the allowed refusal.
func (c *Checker) checkPlainCall(ctx context.Context, rep *Report) {
	const name = "tools/call without the extension"
	wait := durationOr(c.PlainWait, defaultPlainWait)
	callCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	res, err := c.call(callCtx, "tools/call", c.toolParams(), false)
	code, rpcErr := errorCode(err)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		rep.add(name, Pass, "no task handle within %s, the call runs synchronously (cancelled)", wait)
	case err == nil && res["resultType"] == "task":
		rep.add(name, Fail, "returned a task to a client that did not declare %s", client.TasksExtension)
	case err == nil:
		rep.add(name, Pass, "answered synchronously")
	case rpcErr && code == codeMissingRequiredCapability:
		rep.add(name, Pass, "refused with %d: the tool needs tasks", code)
	default:
		rep.add(name, Skip, "the call failed, nothing to check: %v", err)
	}
}

func (c *Checker) toolParams() map[string]any {
	args := c.Args
	if args == nil {
		args = map[string]any{}
	}
	return map[string]any{"name": c.Tool, "arguments": args}
}

// checkLifecycle calls the tool as a task and follows it to a terminal status.
func (c *Checker) checkLifecycle(ctx context.Context, rep *Report) {
	ctx, cancel := context.WithTimeout(ctx, durationOr(c.Timeout, defaultTimeout))
	defer cancel()

	res, err := c.call(ctx, "tools/call", c.toolParams(), true)
	if err != nil {
		rep.add("tools/call with the extension", Skip, "the call failed, nothing to check: %v", err)
		return
	}
	if res["resultType"] != "task" {
		rep.add("task created", Skip, "the server answered synchronously (allowed: it decides per request)")
		return
	}
	addShape(rep, "task handle", res, "task")
	id, _ := res["taskId"].(string)
	if id == "" {
		return
	}
	rep.TaskID = id
	if why := guessableID(id); why != "" {
		rep.add("task id entropy", Warn, "task id %q %s; ids must not be guessable", id, why)
	} else {
		rep.add("task id entropy", Pass, "%d characters", len(id))
	}

	f := &follower{c: c, rep: rep, id: id, seen: map[string]bool{}}
	f.observe(res)
	first, err := c.get(ctx, id)
	if err != nil {
		rep.add("durable creation", Fail, "tasks/get right after the handle failed: %v", err)
		return
	}
	rep.add("durable creation", Pass, "tasks/get resolves the new task at once")
	c.checkBinding(ctx, rep, id)
	l := c.listen(ctx, rep, id)
	if c.Cancel {
		c.checkCancel(ctx, rep, id)
	}
	final := f.follow(ctx, first)
	if final != "" {
		c.checkTerminalStays(ctx, rep, id, final)
	}
	if l != nil {
		l.finish(rep, final)
	}
}

func (c *Checker) get(ctx context.Context, id string) (map[string]any, error) {
	return c.call(ctx, "tasks/get", map[string]any{"taskId": id}, true)
}

// checkCancel: tasks/cancel MUST be acknowledged with an empty result
// (resultType "complete"); honouring it is up to the server.
func (c *Checker) checkCancel(ctx context.Context, rep *Report, id string) {
	res, err := c.call(ctx, "tasks/cancel", map[string]any{"taskId": id}, true)
	switch {
	case err != nil:
		rep.add("tasks/cancel", Fail, "%v", err)
	case res["resultType"] != "complete":
		rep.add("tasks/cancel", Fail, "resultType %v, must be \"complete\"", res["resultType"])
	default:
		rep.add("tasks/cancel", Pass, "acknowledged")
	}
}

// checkTerminalStays: once terminal, a task's state does not change.
func (c *Checker) checkTerminalStays(ctx context.Context, rep *Report, id, final string) {
	res, err := c.get(ctx, id)
	switch {
	case err != nil:
		rep.add("terminal status stays", Warn, "tasks/get after the end failed: %v", err)
	case res["status"] != final:
		rep.add("terminal status stays", Fail, "was %s, now %v", final, res["status"])
	default:
		rep.add("terminal status stays", Pass, "%s", final)
	}
}

// follower polls one task and checks every state it reports.
type follower struct {
	c        *Checker
	rep      *Report
	id       string
	last     string
	seen     map[string]bool // reported problems, each once
	answered map[string]bool
	bad      bool
}

// observe records a status and checks the transition from the last one.
func (f *follower) observe(t map[string]any) {
	status, _ := t["status"].(string)
	if f.last != "" && terminal[f.last] && status != f.last {
		f.problem(Fail, "status transition", "left terminal status "+f.last+" for "+status)
	}
	if status != f.last {
		f.rep.Statuses = append(f.rep.Statuses, status)
	}
	f.last = status
}

func (f *follower) problem(status Status, name, detail string) {
	if f.seen[name+detail] {
		return
	}
	f.seen[name+detail] = true
	if status == Fail {
		f.bad = true
	}
	f.rep.add(name, status, "%s", detail)
}

// follow polls until a terminal status and returns it, or "" if none came.
func (f *follower) follow(ctx context.Context, t map[string]any) string {
	for {
		f.check(t)
		status, _ := t["status"].(string)
		if terminal[status] {
			f.done(status)
			return status
		}
		if status == "input_required" && !f.answer(ctx, t) {
			return ""
		}
		select {
		case <-ctx.Done():
			f.rep.add("terminal status", Warn, "still %s after the timeout", status)
			return ""
		case <-time.After(f.interval(t)):
		}
		var err error
		if t, err = f.c.get(ctx, f.id); err != nil {
			f.rep.add("tasks/get", Fail, "%v", err)
			return ""
		}
	}
}

func (f *follower) check(t map[string]any) {
	must, should := checkTask(t, "complete")
	for _, p := range must {
		f.problem(Fail, "tasks/get result", p)
	}
	for _, p := range should {
		f.problem(Warn, "tasks/get result", p)
	}
	if t["taskId"] != f.id {
		f.problem(Fail, "tasks/get result", "taskId changed")
	}
	f.observe(t)
}

func (f *follower) done(status string) {
	if !f.bad {
		f.rep.add("tasks/get results", Pass, "every poll well-formed")
	}
	f.rep.add("terminal status", Pass, "%s (%s)", status, strings.Join(f.rep.Statuses, " → "))
}

// answer fulfills input requests, or cancels the task without a responder.
func (f *follower) answer(ctx context.Context, t map[string]any) bool {
	if f.c.Responder == nil {
		f.rep.add("input_required", Skip, "no answers configured, the task is cancelled")
		f.c.checkCancel(ctx, f.rep, f.id)
		return false
	}
	if f.answered == nil {
		f.answered = map[string]bool{}
	}
	tc := &client.TaskClient{Session: f.c.Session, Responder: f.c.Responder}
	if err := tc.Answer(ctx, f.id, t["inputRequests"], f.answered); err != nil {
		f.rep.add("tasks/update", Fail, "%v", err)
		return false
	}
	return true
}

func (f *follower) interval(t map[string]any) time.Duration {
	d := time.Second
	if ms, ok := t["pollIntervalMs"].(float64); ok && ms > 0 {
		d = time.Duration(ms) * time.Millisecond
	}
	return min(d, maxPoll)
}

// addShape reports the MUST and SHOULD violations of one task object.
func addShape(rep *Report, name string, t map[string]any, resultType string) {
	must, should := checkTask(t, resultType)
	for _, p := range must {
		rep.add(name, Fail, "%s", p)
	}
	for _, p := range should {
		rep.add(name, Warn, "%s", p)
	}
	if len(must)+len(should) == 0 {
		rep.add(name, Pass, "taskId, status, timestamps, ttlMs, pollIntervalMs")
	}
}

func durationOr(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}
