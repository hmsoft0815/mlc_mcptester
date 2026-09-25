package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerHeaderTools adds a tool whose region parameter is mirrored into the
// Mcp-Param-Region HTTP header (x-mcp-header, spec 2026-07-28), so clients and
// http-check can exercise header mirroring and its server-side validation.
func registerHeaderTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "route_query",
		Title:       "Route Query",
		Description: "Runs a query in a region; the region travels as the Mcp-Param-Region header",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"region": map[string]any{"type": "string", "description": "Region to run in", "x-mcp-header": "Region"},
				"query":  map[string]any{"type": "string", "description": "The query"},
			},
			"required": []string{"region", "query"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		return textResult(fmt.Sprintf("Ran %v in %v", args["query"], args["region"])), nil, nil
	})
}
