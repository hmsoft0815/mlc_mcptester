package httpcheck

import (
	"context"
	"net/http"
)

// checkParamHeaders checks that the server validates Mcp-Param-{Name} headers
// against the body, using the first tool with a string x-mcp-header.
func (c *Checker) checkParamHeaders(ctx context.Context, rep *Report) {
	tool, header, ok := firstStringParamHeader(c.Tools)
	if !ok {
		rep.add("Mcp-Param-* header checks", Skip, "no tool with a string x-mcp-header")
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
		rep.add(name+" matching the body", Fail, "request failed: %v", err)
	case res.rpc != nil && res.rpc.Error != nil && res.rpc.Error.Code == CodeHeaderMismatch:
		rep.add(name+" matching the body", Fail, "%s", describe(res))
	default:
		rep.add(name+" matching the body", Pass, "accepted (%d)", res.status)
	}

	c.expectError(ctx, rep, name+" differs from the body", Fail,
		request{rpcMethod: "tools/call", params: call, headers: map[string]string{name: "other-value"}},
		http.StatusBadRequest, CodeHeaderMismatch)
	c.expectError(ctx, rep, name+" missing, value in the body", Fail,
		request{rpcMethod: "tools/call", params: call},
		http.StatusBadRequest, CodeHeaderMismatch)
}

// firstStringParamHeader returns the first tool with a valid string
// x-mcp-header, and that header.
func firstStringParamHeader(tools []Tool) (Tool, ParamHeader, bool) {
	for _, t := range tools {
		valid, _ := XMCPHeaders(t.InputSchema)
		for _, h := range valid {
			if h.Type == "string" {
				return t, h, true
			}
		}
	}
	return Tool{}, ParamHeader{}, false
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
