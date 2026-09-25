// Package httpcheck probes a Streamable HTTP MCP endpoint with hand-built
// requests and compares the answers with the transport rules of spec
// revision 2026-07-28: request metadata headers, error codes and HTTP status,
// Origin validation and the removal of sessions and GET streams.
package httpcheck

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

// Run executes all checks.
func (c *Checker) Run(ctx context.Context) *Report {
	rep := &Report{Endpoint: c.Endpoint}
	add := func(name string, status Status, format string, args ...any) {
		rep.Results = append(rep.Results, Result{Name: name, Status: status, Detail: fmt.Sprintf(format, args...)})
	}

	// Baseline: a correct server/discover must work, otherwise nothing else is meaningful
	base, err := c.send(ctx, request{rpcMethod: "server/discover"})
	switch {
	case err != nil:
		add("server/discover", Fail, "request failed: %v", err)
		return rep
	case base.status != http.StatusOK || base.rpc == nil || base.rpc.Error != nil:
		add("server/discover", Fail, "a correct request got %s; the server does not speak %s (checks stopped)", describe(base), Revision)
		return rep
	}
	var discover struct {
		SupportedVersions []string `json:"supportedVersions"`
	}
	_ = json.Unmarshal(base.rpc.Result, &discover)
	rep.SupportedVersions = discover.SupportedVersions
	add("server/discover", Pass, "supportedVersions %v", discover.SupportedVersions)

	ct := base.header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/event-stream") {
		add("response Content-Type", Pass, "%s", ct)
	} else {
		add("response Content-Type", Fail, "%q, must be application/json or text/event-stream", ct)
	}

	c.expectError(ctx, add, "missing MCP-Protocol-Version header", Fail,
		request{rpcMethod: "server/discover", headers: map[string]string{"MCP-Protocol-Version": ""}},
		http.StatusBadRequest, CodeHeaderMismatch)
	c.expectError(ctx, add, "MCP-Protocol-Version differs from _meta", Fail,
		request{rpcMethod: "server/discover", headers: map[string]string{"MCP-Protocol-Version": "2025-11-25"}},
		http.StatusBadRequest, CodeHeaderMismatch)
	c.expectError(ctx, add, "missing Mcp-Method header", Fail,
		request{rpcMethod: "server/discover", headers: map[string]string{"Mcp-Method": ""}},
		http.StatusBadRequest, CodeHeaderMismatch)
	c.expectError(ctx, add, "Mcp-Method differs from body", Fail,
		request{rpcMethod: "server/discover", headers: map[string]string{"Mcp-Method": "tools/list"}},
		http.StatusBadRequest, CodeHeaderMismatch)
	// A future version is unambiguous; for versions older than the server's
	// oldest, dual-era servers may answer on the legacy path instead
	c.expectError(ctx, add, "unsupported protocol version", Fail,
		request{rpcMethod: "server/discover", params: map[string]any{}, headers: map[string]string{"MCP-Protocol-Version": "2099-12-31"}},
		http.StatusBadRequest, CodeUnsupportedProtocolVersion)
	c.expectError(ctx, add, "unknown method", Fail,
		request{rpcMethod: "mcp-tester/no-such-method"},
		http.StatusNotFound, CodeMethodNotFound)
	c.expectError(ctx, add, "missing _meta request metadata", Fail,
		request{rpcMethod: "tools/list", noMeta: true},
		http.StatusBadRequest, CodeInvalidParams)

	if len(c.Tools) == 0 {
		add("tools/call header checks", Skip, "the server lists no tools")
	} else {
		toolName := c.Tools[0].Name
		call := map[string]any{"name": toolName, "arguments": placeholderArgs(c.Tools[0].InputSchema)}
		c.expectError(ctx, add, "tools/call without Mcp-Name", Fail,
			request{rpcMethod: "tools/call", params: call, headers: map[string]string{"Mcp-Name": ""}},
			http.StatusBadRequest, CodeHeaderMismatch)
		c.expectError(ctx, add, "Mcp-Name differs from params.name", Fail,
			request{rpcMethod: "tools/call", params: call, headers: map[string]string{"Mcp-Name": toolName + "-other"}},
			http.StatusBadRequest, CodeHeaderMismatch)

		// A Base64 sentinel value must be decoded before comparing; any
		// outcome but HeaderMismatch (even a tool error) is fine
		encoded := "=?base64?" + base64.StdEncoding.EncodeToString([]byte(toolName)) + "?="
		res, err := c.send(ctx, request{rpcMethod: "tools/call", params: call, headers: map[string]string{"Mcp-Name": encoded}})
		switch {
		case err != nil:
			add("Base64-encoded Mcp-Name", Fail, "request failed: %v", err)
		case res.rpc != nil && res.rpc.Error != nil && res.rpc.Error.Code == CodeHeaderMismatch:
			add("Base64-encoded Mcp-Name", Fail, "%s: the server does not decode =?base64?…?= values", describe(res))
		default:
			add("Base64-encoded Mcp-Name", Pass, "accepted (%d)", res.status)
		}
	}

	c.checkParamHeaders(ctx, add)

	// Origin: a foreign origin must be refused; which origins count as valid is up to the server
	if res, err := c.send(ctx, request{rpcMethod: "server/discover", headers: map[string]string{"Origin": "http://mcp-tester.invalid"}}); err != nil {
		add("foreign Origin header", Fail, "request failed: %v", err)
	} else if res.status == http.StatusForbidden {
		add("foreign Origin header", Pass, "403")
	} else {
		add("foreign Origin header", Warn, "Origin http://mcp-tester.invalid got %d; servers MUST answer an invalid Origin with 403 (unless every origin is allowed on purpose)", res.status)
	}

	for _, m := range []string{http.MethodGet, http.MethodDelete} {
		if res, err := c.send(ctx, request{httpMethod: m}); err != nil {
			add(m+" on the MCP endpoint", Warn, "request failed: %v", err)
		} else if res.status == http.StatusMethodNotAllowed {
			add(m+" on the MCP endpoint", Pass, "405")
		} else {
			add(m+" on the MCP endpoint", Warn, "got %d; a %s-only server SHOULD answer 405", res.status, Revision)
		}
	}

	if res, err := c.send(ctx, request{rpcMethod: "server/discover", headers: map[string]string{"Mcp-Session-Id": "mcp-tester-session"}}); err != nil {
		add("Mcp-Session-Id is ignored", Warn, "request failed: %v", err)
	} else if sid := res.header.Get("Mcp-Session-Id"); res.status != http.StatusOK || sid != "" {
		add("Mcp-Session-Id is ignored", Warn, "got %d with Mcp-Session-Id %q; sessions are removed in %s", res.status, sid, Revision)
	} else {
		add("Mcp-Session-Id is ignored", Pass, "no session id minted or echoed")
	}

	return rep
}

// expectError sends req and checks the HTTP status and JSON-RPC error code.
func (c *Checker) expectError(ctx context.Context, add func(string, Status, string, ...any), name string, severity Status, req request, status, code int) {
	res, err := c.send(ctx, req)
	if err != nil {
		add(name, severity, "request failed: %v", err)
		return
	}
	gotCode := 0
	if res.rpc != nil && res.rpc.Error != nil {
		gotCode = res.rpc.Error.Code
	}
	if res.status == status && gotCode == code {
		add(name, Pass, "%d, %d", status, code)
		return
	}
	add(name, severity, "got %s, want HTTP %d with error %d", describe(res), status, code)
}

// send builds a request that is valid except for what req changes.
func (c *Checker) send(ctx context.Context, req request) (*response, error) {
	httpMethod := req.httpMethod
	if httpMethod == "" {
		httpMethod = http.MethodPost
	}

	var body io.Reader
	headers := map[string]string{}
	if httpMethod == http.MethodPost {
		params := map[string]any{}
		for k, v := range req.params {
			params[k] = v
		}
		version := Revision
		if v, ok := req.headers["MCP-Protocol-Version"]; ok && v == "2099-12-31" {
			version = v // unsupported version: header and body agree
		}
		if !req.noMeta {
			params["_meta"] = map[string]any{
				"io.modelcontextprotocol/protocolVersion":    version,
				"io.modelcontextprotocol/clientInfo":         map[string]any{"name": "mcp-tester", "version": "http-check"},
				"io.modelcontextprotocol/clientCapabilities": map[string]any{},
			}
		}
		data, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": req.rpcMethod, "params": params})
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
		headers["Content-Type"] = "application/json"
		headers["Accept"] = "application/json, text/event-stream"
		headers["MCP-Protocol-Version"] = version
		headers["Mcp-Method"] = req.rpcMethod
		if name, ok := req.params["name"].(string); ok {
			headers["Mcp-Name"] = name
		}
	}
	for k, v := range req.headers {
		headers[k] = v
	}

	httpReq, err := http.NewRequestWithContext(ctx, httpMethod, c.Endpoint, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		if v != "" {
			httpReq.Header.Set(k, v)
		}
	}
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	res := &response{status: httpResp.StatusCode, header: httpResp.Header, body: data}
	res.rpc = parseRPC(httpResp.Header.Get("Content-Type"), data)
	return res, nil
}

// parseRPC extracts the JSON-RPC response from a JSON body or the last data
// event of an SSE body.
func parseRPC(contentType string, body []byte) *rpcMessage {
	candidates := [][]byte{body}
	if strings.HasPrefix(contentType, "text/event-stream") {
		candidates = nil
		sc := bufio.NewScanner(bytes.NewReader(body))
		sc.Buffer(make([]byte, 64*1024), 1<<20)
		for sc.Scan() {
			if data, ok := strings.CutPrefix(sc.Text(), "data:"); ok {
				candidates = append(candidates, []byte(strings.TrimSpace(data)))
			}
		}
	}
	for i := len(candidates) - 1; i >= 0; i-- {
		var msg rpcMessage
		if json.Unmarshal(candidates[i], &msg) == nil && (msg.Result != nil || msg.Error != nil) {
			return &msg
		}
	}
	return nil
}

func describe(res *response) string {
	if res.rpc != nil && res.rpc.Error != nil {
		return fmt.Sprintf("HTTP %d with error %d (%s)", res.status, res.rpc.Error.Code, res.rpc.Error.Message)
	}
	if res.rpc != nil {
		return fmt.Sprintf("HTTP %d with a result", res.status)
	}
	text := strings.TrimSpace(string(res.body))
	if len(text) > 80 {
		text = text[:80] + "…"
	}
	return fmt.Sprintf("HTTP %d %q", res.status, text)
}

// checkParamHeaders checks that the server validates Mcp-Param-{Name} headers
// against the body, using the first tool with a string x-mcp-header.
func (c *Checker) checkParamHeaders(ctx context.Context, add func(string, Status, string, ...any)) {
	var tool Tool
	var header ParamHeader
	for _, t := range c.Tools {
		valid, _ := XMCPHeaders(t.InputSchema)
		for _, h := range valid {
			if h.Type == "string" {
				tool, header = t, h
				break
			}
		}
		if tool.Name != "" {
			break
		}
	}
	if tool.Name == "" {
		add("Mcp-Param-* header checks", Skip, "no tool with a string x-mcp-header")
		return
	}

	args := placeholderArgs(tool.InputSchema)
	setPath(args, header.Path, "mcp-tester")
	call := map[string]any{"name": tool.Name, "arguments": args}
	name := "Mcp-Param-" + header.Header

	// The correct header must pass header validation (a tool error is fine)
	res, err := c.send(ctx, request{rpcMethod: "tools/call", params: call, headers: map[string]string{name: "mcp-tester"}})
	switch {
	case err != nil:
		add(name+" matching the body", Fail, "request failed: %v", err)
	case res.rpc != nil && res.rpc.Error != nil && res.rpc.Error.Code == CodeHeaderMismatch:
		add(name+" matching the body", Fail, "%s", describe(res))
	default:
		add(name+" matching the body", Pass, "accepted (%d)", res.status)
	}

	c.expectError(ctx, add, name+" differs from the body", Fail,
		request{rpcMethod: "tools/call", params: call, headers: map[string]string{name: "other-value"}},
		http.StatusBadRequest, CodeHeaderMismatch)
	c.expectError(ctx, add, name+" missing, value in the body", Fail,
		request{rpcMethod: "tools/call", params: call},
		http.StatusBadRequest, CodeHeaderMismatch)
}

// placeholderArgs fills the required top-level properties of a schema with
// values of the right type, so that argument validation does not stop a probe.
func placeholderArgs(schema any) map[string]any {
	args := map[string]any{}
	root, _ := schema.(map[string]any)
	props, _ := root["properties"].(map[string]any)
	required, _ := root["required"].([]any)
	for _, r := range required {
		name, _ := r.(string)
		prop, _ := props[name].(map[string]any)
		switch prop["type"] {
		case "integer", "number":
			args[name] = 1
		case "boolean":
			args[name] = true
		case "array":
			args[name] = []any{}
		case "object":
			args[name] = map[string]any{}
		default:
			args[name] = "x"
		}
	}
	return args
}

// setPath sets value at a property path, creating intermediate objects.
func setPath(args map[string]any, path []string, value any) {
	for _, key := range path[:len(path)-1] {
		next, ok := args[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			args[key] = next
		}
		args = next
	}
	args[path[len(path)-1]] = value
}
