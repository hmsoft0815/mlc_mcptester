// Package httpcheck probes a Streamable HTTP MCP endpoint with hand-built
// requests and compares the answers with the transport rules of spec
// revision 2026-07-28: request metadata headers, error codes and HTTP status,
// Origin validation and the removal of sessions and GET streams.
package httpcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Revision is the protocol revision whose transport rules are checked.
const Revision = "2026-07-28"

// JSON-RPC error codes the checks expect.
const (
	CodeMethodNotFound             = -32601
	CodeInvalidParams              = -32602
	CodeHeaderMismatch             = -32020
	CodeUnsupportedProtocolVersion = -32022
)

// Status of one check.
type Status string

const (
	Pass Status = "PASS"
	Fail Status = "FAIL" // a MUST is violated
	Warn Status = "WARN" // a SHOULD is violated
	Skip Status = "SKIP"
)

// Result is the outcome of one check.
type Result struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Report is the outcome of all checks.
type Report struct {
	Endpoint          string   `json:"endpoint"`
	SupportedVersions []string `json:"supportedVersions,omitempty"`
	Results           []Result `json:"results"`
}

func (r *Report) add(name string, status Status, format string, args ...any) {
	r.Results = append(r.Results, Result{Name: name, Status: status, Detail: fmt.Sprintf(format, args...)})
}

// Failed reports whether any check failed.
func (r *Report) Failed() bool {
	for _, res := range r.Results {
		if res.Status == Fail {
			return true
		}
	}
	return false
}

// Checker sends the probe requests.
type Checker struct {
	Endpoint string
	Client   *http.Client // carries authentication; http.DefaultClient if nil
	// Tools from tools/list; the tools/call header checks are skipped without them.
	Tools []Tool
}

// Tool is what the checks need to know about a listed tool.
type Tool struct {
	Name        string
	InputSchema any
}

// response is what a probe got back.
type response struct {
	status int
	header http.Header
	body   []byte
	rpc    *rpcMessage // the JSON-RPC message in the body, if any
}

type rpcMessage struct {
	ID     any             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	} `json:"error"`
}

// request describes one probe; zero fields fall back to a valid request.
type request struct {
	httpMethod string
	rpcMethod  string
	params     map[string]any
	noMeta     bool
	headers    map[string]string // "" value removes the header
}

// discoverResult is what the checks read from server/discover.
type discoverResult struct {
	SupportedVersions []string `json:"supportedVersions"`
	Capabilities      struct {
		Resources json.RawMessage `json:"resources"`
	} `json:"capabilities"`
}

// Run executes all checks.
func (c *Checker) Run(ctx context.Context) *Report {
	rep := &Report{Endpoint: c.Endpoint}
	base, discover, ok := c.checkDiscover(ctx, rep)
	if !ok {
		return rep
	}
	c.checkServerInfo(ctx, rep, base)
	checkContentType(rep, base)
	c.checkHeaderErrors(ctx, rep)
	c.checkNameHeader(ctx, rep)
	c.checkParamHeaders(ctx, rep)
	c.checkResourceNotFound(ctx, rep, discover.Capabilities.Resources != nil)
	c.checkOrigin(ctx, rep)
	c.checkLegacyMethods(ctx, rep)
	c.checkSessionIgnored(ctx, rep)
	return rep
}

// checkDiscover is the baseline: a correct server/discover must work,
// otherwise nothing else is meaningful.
func (c *Checker) checkDiscover(ctx context.Context, rep *Report) (*response, discoverResult, bool) {
	var discover discoverResult
	base, err := c.send(ctx, request{rpcMethod: "server/discover"})
	switch {
	case err != nil:
		rep.add("server/discover", Fail, "request failed: %v", err)
		return nil, discover, false
	case base.status != http.StatusOK || base.rpc == nil || base.rpc.Error != nil:
		rep.add("server/discover", Fail, "a correct request got %s; the server does not speak %s (checks stopped)", describe(base), Revision)
		return nil, discover, false
	}
	_ = json.Unmarshal(base.rpc.Result, &discover)
	rep.SupportedVersions = discover.SupportedVersions
	rep.add("server/discover", Pass, "supportedVersions %v", discover.SupportedVersions)
	return base, discover, true
}

// checkServerInfo: servers SHOULD identify themselves in every result's _meta.
func (c *Checker) checkServerInfo(ctx context.Context, rep *Report, base *response) {
	for _, probe := range []struct {
		method string
		res    *response
	}{{"server/discover", base}, {"tools/list", nil}} {
		res := probe.res
		if res == nil {
			var err error
			if res, err = c.send(ctx, request{rpcMethod: probe.method}); err != nil || res.rpc == nil || res.rpc.Error != nil {
				continue
			}
		}
		var result struct {
			Meta map[string]json.RawMessage `json:"_meta"`
		}
		_ = json.Unmarshal(res.rpc.Result, &result)
		if _, ok := result.Meta["io.modelcontextprotocol/serverInfo"]; ok {
			rep.add("serverInfo in "+probe.method+" _meta", Pass, "present")
		} else {
			rep.add("serverInfo in "+probe.method+" _meta", Warn, "missing; servers SHOULD send io.modelcontextprotocol/serverInfo with every result")
		}
	}
}

func checkContentType(rep *Report, base *response) {
	ct := base.header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/event-stream") {
		rep.add("response Content-Type", Pass, "%s", ct)
	} else {
		rep.add("response Content-Type", Fail, "%q, must be application/json or text/event-stream", ct)
	}
}
