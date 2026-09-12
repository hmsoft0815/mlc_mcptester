package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerExtraTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "error_trigger",
		Description: "Triggers a simulated server error for testing error handling",
		Icons: []mcp.Icon{
			{Source: serverIcon, MIMEType: "image/svg+xml"},
		},
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"code":    map[string]any{"type": "integer", "description": "Error code"},
				"message": map[string]any{"type": "string", "description": "Error message"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		msg, _ := args["message"].(string)
		if msg == "" {
			msg = "Simulated Server Error"
		}
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		}, nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "join_strings",
		Description: "Joins a list of strings (tests array argument support)",
		Icons: []mcp.Icon{
			{Source: serverIcon, MIMEType: "image/svg+xml"},
		},
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"paths": map[string]any{
					"type":        []any{"null", "array"},
					"description": "List of paths or strings to join",
					"items":       map[string]any{"type": "string"},
				},
			},
			"required": []string{"paths"},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"joined": map[string]any{"type": "string", "description": "The joined string"},
				"count":  map[string]any{"type": "integer", "description": "Number of elements"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		pathsRaw, ok := args["paths"]
		if !ok || pathsRaw == nil {
			return nil, nil, fmt.Errorf("missing paths parameter")
		}
		var paths []string
		if list, ok := pathsRaw.([]any); ok {
			for _, item := range list {
				paths = append(paths, fmt.Sprint(item))
			}
		}
		joined := strings.Join(paths, ", ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Joined: " + joined}},
		}, map[string]any{"joined": joined, "count": len(paths)}, nil
	})
}
