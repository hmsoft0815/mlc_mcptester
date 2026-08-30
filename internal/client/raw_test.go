package client

import (
	"context"
	"io"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCallToolRaw(t *testing.T) {
	ctx := context.Background()

	// Create test server
	server := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name: "test_echo",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"msg": map[string]any{"type": "string"},
			},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		msg, _ := args["msg"].(string)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "echo:" + msg}},
		}, map[string]any{"echo": msg}, nil
	})

	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	serverTransport := &mcp.IOTransport{Reader: r1, Writer: w2}
	clientTransport := &mcp.IOTransport{Reader: r2, Writer: w1}

	_, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer session.Close()

	// Test CallToolRaw
	res, err := CallToolRaw(ctx, session, "test_echo", map[string]any{"msg": "hello"}, nil)
	if err != nil {
		t.Fatalf("CallToolRaw failed: %v", err)
	}

	if res == nil {
		t.Fatalf("expected non-nil result")
	}

	content, ok := res["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("expected content in result: %+v", res)
	}
}
