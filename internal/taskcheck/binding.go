package taskcheck

import (
	"context"
	"time"
)

// bindingSettle is how long a cancel by another identity gets to show.
const bindingSettle = 300 * time.Millisecond

// checkBinding: servers MUST check on every task request that the caller
// may access the task. Another identity (c.Other) must not read, answer or
// cancel it; ideally it gets -32602 as for an unknown task, so the task's
// existence does not leak.
func (c *Checker) checkBinding(ctx context.Context, rep *Report, id string) {
	if c.Other == nil {
		rep.add("auth binding", Skip, "needs a second identity (--other-bearer)")
		return
	}
	other := &Checker{Session: c.Other}
	probes := []struct {
		method string
		params map[string]any
	}{
		{"tasks/get", map[string]any{"taskId": id}},
		{"tasks/update", map[string]any{"taskId": id, "inputResponses": map[string]any{}}},
		{"tasks/cancel", map[string]any{"taskId": id}},
	}
	for _, p := range probes {
		name := p.method + " by another identity"
		res, err := other.call(ctx, p.method, p.params, true)
		code, rpcErr := errorCode(err)
		switch {
		case err == nil && p.method == "tasks/get":
			rep.add(name, Fail, "returned the task (status %v) to another identity", res["status"])
		case err == nil:
			rep.add(name, Warn, "acknowledged for another identity's task, expected error %d", codeInvalidParams)
		case !rpcErr:
			rep.add(name, Fail, "%v", err)
		case code != codeInvalidParams:
			rep.add(name, Warn, "refused with %d; %d (as for an unknown task) would not reveal that the task exists", code, codeInvalidParams)
		default:
			rep.add(name, Pass, "%d", code)
		}
	}
	time.Sleep(bindingSettle)
	if res, err := c.get(ctx, id); err == nil && res["status"] == "cancelled" && !c.Cancel {
		rep.add("auth binding", Fail, "another identity cancelled the task")
	}
}
