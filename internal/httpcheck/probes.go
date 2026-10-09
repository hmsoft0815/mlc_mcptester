package httpcheck

import (
	"context"
	"encoding/base64"
	"net/http"
)

// expectError sends req and checks the HTTP status and JSON-RPC error code.
func (c *Checker) expectError(ctx context.Context, rep *Report, name string, severity Status, req request, status, code int) {
	res, err := c.send(ctx, req)
	if err != nil {
		rep.add(name, severity, "request failed: %v", err)
		return
	}
	gotCode := 0
	if res.rpc != nil && res.rpc.Error != nil {
		gotCode = res.rpc.Error.Code
	}
	if res.status == status && gotCode == code {
		rep.add(name, Pass, "%d, %d", status, code)
		return
	}
	rep.add(name, severity, "got %s, want HTTP %d with error %d", describe(res), status, code)
}

// checkHeaderErrors: wrong or missing request metadata headers, an
// unsupported version and an unknown method get their status and code.
func (c *Checker) checkHeaderErrors(ctx context.Context, rep *Report) {
	discover := func(headers map[string]string) request {
		return request{rpcMethod: "server/discover", headers: headers}
	}
	c.expectError(ctx, rep, "missing MCP-Protocol-Version header", Fail,
		discover(map[string]string{"MCP-Protocol-Version": ""}), http.StatusBadRequest, CodeHeaderMismatch)
	c.expectError(ctx, rep, "MCP-Protocol-Version differs from _meta", Fail,
		discover(map[string]string{"MCP-Protocol-Version": "2025-11-25"}), http.StatusBadRequest, CodeHeaderMismatch)
	c.expectError(ctx, rep, "missing Mcp-Method header", Fail,
		discover(map[string]string{"Mcp-Method": ""}), http.StatusBadRequest, CodeHeaderMismatch)
	c.expectError(ctx, rep, "Mcp-Method differs from body", Fail,
		discover(map[string]string{"Mcp-Method": "tools/list"}), http.StatusBadRequest, CodeHeaderMismatch)
	// A future version is unambiguous; for versions older than the server's
	// oldest, dual-era servers may answer on the legacy path instead
	c.expectError(ctx, rep, "unsupported protocol version", Fail,
		request{rpcMethod: "server/discover", params: map[string]any{}, headers: map[string]string{"MCP-Protocol-Version": unsupportedVersion}},
		http.StatusBadRequest, CodeUnsupportedProtocolVersion)
	c.expectError(ctx, rep, "unknown method", Fail,
		request{rpcMethod: "mcp-tester/no-such-method"}, http.StatusNotFound, CodeMethodNotFound)
	c.expectError(ctx, rep, "missing _meta request metadata", Fail,
		request{rpcMethod: "tools/list", noMeta: true}, http.StatusBadRequest, CodeInvalidParams)
}

// checkNameHeader: tools/call needs an Mcp-Name that matches params.name,
// also when it arrives Base64-encoded.
func (c *Checker) checkNameHeader(ctx context.Context, rep *Report) {
	if len(c.Tools) == 0 {
		rep.add("tools/call header checks", Skip, "the server lists no tools")
		return
	}
	toolName := c.Tools[0].Name
	call := map[string]any{"name": toolName, "arguments": placeholderArgs(c.Tools[0].InputSchema)}
	c.expectError(ctx, rep, "tools/call without Mcp-Name", Fail,
		request{rpcMethod: "tools/call", params: call, headers: map[string]string{"Mcp-Name": ""}},
		http.StatusBadRequest, CodeHeaderMismatch)
	c.expectError(ctx, rep, "Mcp-Name differs from params.name", Fail,
		request{rpcMethod: "tools/call", params: call, headers: map[string]string{"Mcp-Name": toolName + "-other"}},
		http.StatusBadRequest, CodeHeaderMismatch)

	// A Base64 sentinel value must be decoded before comparing; any
	// outcome but HeaderMismatch (even a tool error) is fine
	encoded := "=?base64?" + base64.StdEncoding.EncodeToString([]byte(toolName)) + "?="
	res, err := c.send(ctx, request{rpcMethod: "tools/call", params: call, headers: map[string]string{"Mcp-Name": encoded}})
	switch {
	case err != nil:
		rep.add("Base64-encoded Mcp-Name", Fail, "request failed: %v", err)
	case res.rpc != nil && res.rpc.Error != nil && res.rpc.Error.Code == CodeHeaderMismatch:
		rep.add("Base64-encoded Mcp-Name", Fail, "%s: the server does not decode =?base64?…?= values", describe(res))
	default:
		rep.add("Base64-encoded Mcp-Name", Pass, "accepted (%d)", res.status)
	}
}

// checkResourceNotFound: resource not found is -32602 since 2026-07-28; the
// old -32002 is retired.
func (c *Checker) checkResourceNotFound(ctx context.Context, rep *Report, hasResources bool) {
	const uri = "mcp-tester://does-not-exist"
	if !hasResources {
		rep.add("resource not found", Skip, "the server declares no resources")
		return
	}
	res, err := c.send(ctx, request{rpcMethod: "resources/read", params: map[string]any{"uri": uri}, headers: map[string]string{"Mcp-Name": uri}})
	switch {
	case err != nil:
		rep.add("resource not found", Fail, "request failed: %v", err)
	case res.rpc == nil || res.rpc.Error == nil:
		rep.add("resource not found", Fail, "got %s, want error %d", describe(res), CodeInvalidParams)
	case res.rpc.Error.Code == CodeInvalidParams:
		rep.add("resource not found", Pass, "%d", CodeInvalidParams)
	default:
		rep.add("resource not found", Fail, "error %d, want %d (-32002 is retired and must not be sent)", res.rpc.Error.Code, CodeInvalidParams)
	}
}

// checkOrigin: a foreign origin must be refused; which origins count as valid
// is up to the server.
func (c *Checker) checkOrigin(ctx context.Context, rep *Report) {
	res, err := c.send(ctx, request{rpcMethod: "server/discover", headers: map[string]string{"Origin": "http://mcp-tester.invalid"}})
	switch {
	case err != nil:
		rep.add("foreign Origin header", Fail, "request failed: %v", err)
	case res.status == http.StatusForbidden:
		rep.add("foreign Origin header", Pass, "403")
	default:
		rep.add("foreign Origin header", Warn, "Origin http://mcp-tester.invalid got %d; servers MUST answer an invalid Origin with 403 (unless every origin is allowed on purpose)", res.status)
	}
}

// checkLegacyMethods: GET streams and DELETE of sessions are gone.
func (c *Checker) checkLegacyMethods(ctx context.Context, rep *Report) {
	for _, m := range []string{http.MethodGet, http.MethodDelete} {
		res, err := c.send(ctx, request{httpMethod: m})
		switch {
		case err != nil:
			rep.add(m+" on the MCP endpoint", Warn, "request failed: %v", err)
		case res.status == http.StatusMethodNotAllowed:
			rep.add(m+" on the MCP endpoint", Pass, "405")
		default:
			rep.add(m+" on the MCP endpoint", Warn, "got %d; a %s-only server SHOULD answer 405", res.status, Revision)
		}
	}
}

// checkSessionIgnored: sessions are removed, an Mcp-Session-Id must neither
// be minted nor echoed.
func (c *Checker) checkSessionIgnored(ctx context.Context, rep *Report) {
	res, err := c.send(ctx, request{rpcMethod: "server/discover", headers: map[string]string{"Mcp-Session-Id": "mcp-tester-session"}})
	if err != nil {
		rep.add("Mcp-Session-Id is ignored", Warn, "request failed: %v", err)
		return
	}
	if sid := res.header.Get("Mcp-Session-Id"); res.status != http.StatusOK || sid != "" {
		rep.add("Mcp-Session-Id is ignored", Warn, "got %d with Mcp-Session-Id %q; sessions are removed in %s", res.status, sid, Revision)
		return
	}
	rep.add("Mcp-Session-Id is ignored", Pass, "no session id minted or echoed")
}
