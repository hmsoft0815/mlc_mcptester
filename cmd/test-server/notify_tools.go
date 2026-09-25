package main

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerNotifyTools adds tools that make the server send notifications, so
// clients can test subscriptions/listen: list changes and resource updates.
func registerNotifyTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "add_tool",
		Title:       "Add Tool",
		Description: "Adds a tool at runtime; the server sends notifications/tools/list_changed",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		Name string `json:"name" jsonschema:"Name of the new tool (prefixed with dyn_)"`
	}) (*mcp.CallToolResult, any, error) {
		name := "dyn_" + args.Name
		mcp.AddTool(s, &mcp.Tool{Name: name, Description: "Tool added at runtime"},
			func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
				return textResult("Hello from " + name), nil, nil
			})
		return textResult("Added tool " + name), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "touch_resource",
		Title:       "Touch Resource",
		Description: "Reports a resource as changed; subscribers get notifications/resources/updated",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		URI string `json:"uri" jsonschema:"URI of the changed resource"`
	}) (*mcp.CallToolResult, any, error) {
		if err := s.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{URI: args.URI}); err != nil {
			return nil, nil, fmt.Errorf("notifying: %w", err)
		}
		return textResult("Touched " + args.URI), nil, nil
	})
}

// Subscriptions are tracked by the SDK; the handlers only accept them.
func subscribe(context.Context, *mcp.SubscribeRequest) error     { return nil }
func unsubscribe(context.Context, *mcp.UnsubscribeRequest) error { return nil }
