package httpcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// conformingEndpoint serves a stateless go-sdk server with a tool whose
// region parameter is mirrored into Mcp-Param-Region, and one resource.
func conformingEndpoint(t *testing.T) (string, []Tool) {
	t.Helper()
	s := mcp.NewServer(&mcp.Implementation{Name: "http", Version: "1"}, nil)
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"region": map[string]any{"type": "string", "x-mcp-header": "Region"},
		},
		"required": []any{"region"},
	}
	mcp.AddTool(s, &mcp.Tool{Name: "deploy", InputSchema: schema}, func(ctx context.Context, r *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}}, nil, nil
	})
	s.AddResource(&mcp.Resource{URI: "test://r", Name: "r"}, func(ctx context.Context, r *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: "test://r", Text: "x"}}}, nil
	})
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{Stateless: true})
	srv := httptest.NewServer(http.NewCrossOriginProtection().Handler(h))
	t.Cleanup(srv.Close)
	return srv.URL, []Tool{{Name: "deploy", InputSchema: schema}}
}

func results(rep *Report) map[string]Result {
	m := map[string]Result{}
	for _, r := range rep.Results {
		m[r.Name] = r
	}
	return m
}

func TestRunConformingServer(t *testing.T) {
	url, tools := conformingEndpoint(t)
	rep := (&Checker{Endpoint: url, Tools: tools}).Run(context.Background())
	got := results(rep)
	if len(rep.SupportedVersions) == 0 || rep.SupportedVersions[0] != Revision {
		t.Errorf("supportedVersions %v", rep.SupportedVersions)
	}
	for _, name := range []string{
		"server/discover", "response Content-Type", "missing MCP-Protocol-Version header",
		"MCP-Protocol-Version differs from _meta", "missing Mcp-Method header", "Mcp-Method differs from body",
		"unsupported protocol version", "unknown method", "missing _meta request metadata",
		"tools/call without Mcp-Name", "Mcp-Name differs from params.name",
		"Mcp-Param-Region matching the body", "Mcp-Param-Region differs from the body",
		"Mcp-Param-Region missing, value in the body", "resource not found", "foreign Origin header",
		"GET on the MCP endpoint", "DELETE on the MCP endpoint", "Mcp-Session-Id is ignored",
	} {
		if r, ok := got[name]; !ok || r.Status != Pass {
			t.Errorf("%s: %+v, want PASS", name, r)
		}
	}
	// go-sdk v1.8.0 does not decode =?base64?…?= (T-20261005-01): either
	// outcome is fine here, but the check must run
	if _, ok := got["Base64-encoded Mcp-Name"]; !ok {
		t.Error("Base64-encoded Mcp-Name was not checked")
	}
}

func TestRunWithoutToolsOrResources(t *testing.T) {
	s := mcp.NewServer(&mcp.Implementation{Name: "empty", Version: "1"}, nil)
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{Stateless: true})
	srv := httptest.NewServer(h)
	defer srv.Close()
	got := results((&Checker{Endpoint: srv.URL}).Run(context.Background()))
	for _, name := range []string{"tools/call header checks", "Mcp-Param-* header checks", "resource not found"} {
		if got[name].Status != Skip {
			t.Errorf("%s: %+v, want SKIP", name, got[name])
		}
	}
	// No origin check in front of this handler: a foreign Origin passes
	if got["foreign Origin header"].Status != Warn {
		t.Errorf("foreign Origin header: %+v, want WARN", got["foreign Origin header"])
	}
}

func TestRunStopsWhenDiscoverFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sessionid must be provided", http.StatusBadRequest)
	}))
	defer srv.Close()
	rep := (&Checker{Endpoint: srv.URL}).Run(context.Background())
	if len(rep.Results) != 1 || rep.Results[0].Status != Fail || !strings.Contains(rep.Results[0].Detail, "checks stopped") {
		t.Fatalf("results %+v, want a single FAIL that stops the checks", rep.Results)
	}
	if !rep.Failed() {
		t.Error("report not failed")
	}
}

func TestPlaceholderArgsAndSetPath(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{
			"n": map[string]any{"type": "integer"}, "b": map[string]any{"type": "boolean"},
			"l": map[string]any{"type": "array"}, "o": map[string]any{"type": "object"}, "s": map[string]any{},
		},
		"required": []any{"n", "b", "l", "o", "s"},
	}
	args := placeholderArgs(schema)
	if args["n"] != 1 || args["b"] != true || args["s"] != "x" {
		t.Errorf("placeholders %v", args)
	}
	setPath(args, []string{"o", "inner", "region"}, "eu")
	inner, _ := args["o"].(map[string]any)["inner"].(map[string]any)
	if inner["region"] != "eu" {
		t.Errorf("setPath: %v", args)
	}
}
