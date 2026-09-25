package client

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListAllToolsFollowsPages(t *testing.T) {
	ctx := context.Background()

	// PageSize 1 forces one page per tool
	server := mcp.NewServer(&mcp.Implementation{Name: "paged", Version: "1.0.0"}, &mcp.ServerOptions{PageSize: 1})
	for i := range 3 {
		mcp.AddTool(server, &mcp.Tool{Name: fmt.Sprintf("tool_%d", i)},
			func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
				return &mcp.CallToolResult{}, nil, nil
			})
	}

	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	if _, err := server.Connect(ctx, &mcp.IOTransport{Reader: r1, Writer: w2}, nil); err != nil {
		t.Fatalf("failed to connect server: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.IOTransport{Reader: r2, Writer: w1}, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer session.Close()

	first, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(first.Tools) != 1 || first.NextCursor == "" {
		t.Fatalf("test setup: expected a first page of 1 tool with a cursor, got %d tools, cursor %q", len(first.Tools), first.NextCursor)
	}

	tools, err := ListAllTools(ctx, session)
	if err != nil {
		t.Fatalf("ListAllTools failed: %v", err)
	}
	if len(tools) != 3 {
		t.Errorf("ListAllTools returned %d tools, want 3", len(tools))
	}
}
