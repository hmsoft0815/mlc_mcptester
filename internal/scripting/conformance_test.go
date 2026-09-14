package scripting

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// A call to a tool whose result a strict client would reject must fail the
// script, in both call modes. The server is built with the SDK's low-level
// AddTool, which leaves the result exactly as the handler returns it — the
// high-level AddTool would fill in structuredContent and hide the defect.
func TestCallToolChecksTheResultAgainstTheOutputSchema(t *testing.T) {
	ctx := context.Background()

	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"count": map[string]any{"type": "integer"}},
		"required":   []any{"count"},
	}
	input := map[string]any{"type": "object"}
	returning := func(res *mcp.CallToolResult) mcp.ToolHandler {
		return func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) { return res, nil }
	}
	text := []mcp.Content{&mcp.TextContent{Text: `{"count": 2}`}}

	s := mcp.NewServer(&mcp.Implementation{Name: "conformance", Version: "0"}, nil)
	s.AddTool(&mcp.Tool{Name: "text_only", InputSchema: input, OutputSchema: schema},
		returning(&mcp.CallToolResult{Content: text}))
	s.AddTool(&mcp.Tool{Name: "wrong_shape", InputSchema: input, OutputSchema: schema},
		returning(&mcp.CallToolResult{Content: text, StructuredContent: map[string]any{"count": "two"}}))
	s.AddTool(&mcp.Tool{Name: "conforming", InputSchema: input, OutputSchema: schema},
		returning(&mcp.CallToolResult{Content: text, StructuredContent: map[string]any{"count": 2}}))
	s.AddTool(&mcp.Tool{Name: "no_schema", InputSchema: input},
		returning(&mcp.CallToolResult{Content: text}))

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	for _, raw := range []bool{false, true} {
		r := NewRunner(session, raw)
		for tool, wantErr := range map[string]string{
			"text_only":   "no structuredContent",
			"wrong_shape": "does not match",
			"conforming":  "",
			"no_schema":   "",
		} {
			err := r.handleCallTool(ctx, 0, "call_tool "+tool)
			switch {
			case wantErr == "" && err != nil:
				t.Errorf("raw=%v %s: unexpected error: %v", raw, tool, err)
			case wantErr != "" && err == nil:
				t.Errorf("raw=%v %s: passed, but a strict client rejects this result", raw, tool)
			case wantErr != "" && !strings.Contains(err.Error(), wantErr):
				t.Errorf("raw=%v %s: error %q, want it to contain %q", raw, tool, err, wantErr)
			}
		}
	}
}
