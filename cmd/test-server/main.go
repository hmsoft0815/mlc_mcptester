package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ultimate-test-server",
			Version: "2.0.0",
		},
		&mcp.ServerOptions{
			Capabilities: &mcp.ServerCapabilities{
				Logging: &mcp.LoggingCapabilities{},
				Tools:   &mcp.ToolCapabilities{ListChanged: true},
			},
		},
	)

	// --- 1. TOOLS ---

	// Echo Tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "echo",
		Description: "Echoes the input",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Echo: " + args["message"].(string)}}}, nil, nil
	})

	// Add Tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "add",
		Description: "Adds two numbers",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{"type": "integer"},
				"b": map[string]any{"type": "integer"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		var a, b int
		if v, ok := args["a"].(float64); ok { a = int(v) } else { a, _ = args["a"].(int) }
		if v, ok := args["b"].(float64); ok { b = int(v) } else { b, _ = args["b"].(int) }
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Result: %d", a+b)}},
		}, map[string]any{"sum": a + b, "a": a, "b": b}, nil
	})

	// Long Running Task
	mcp.AddTool(s, &mcp.Tool{
		Name:        "long_running_task",
		Description: "A task that reports progress and sends logs",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"seconds": map[string]any{"type": "integer", "description": "How long to run"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		seconds := 3
		if s, ok := args["seconds"].(float64); ok { seconds = int(s) }

		_ = request.Session.Log(ctx, &mcp.LoggingMessageParams{
			Level:  "info",
			Logger: "task-worker",
			Data:   fmt.Sprintf("Starting task for %d seconds", seconds),
		})

		for i := 1; i <= seconds; i++ {
			time.Sleep(100 * time.Millisecond) // Faster for testing
			if request.Params.GetProgressToken() != nil {
				_ = request.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
					Progress:      float64(i),
					Total:         float64(seconds),
					ProgressToken: request.Params.GetProgressToken(),
					Message:       fmt.Sprintf("Step %d", i),
				})
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Task completed"}},
		}, nil, nil
	})

	// --- 2. RESOURCES ---
	s.AddResource(&mcp.Resource{
		Name: "Clock",
		URI:  "mcp://time",
	}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: "mcp://time", Text: time.Now().String()}},
		}, nil
	})

	// --- 3. PROMPTS ---
	s.AddPrompt(&mcp.Prompt{
		Name: "hello",
	}, func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Messages: []*mcp.PromptMessage{{Role: "assistant", Content: &mcp.TextContent{Text: "Hello!"}}},
		}, nil
	})

	transport := &mcp.StdioTransport{}
	session, err := s.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal(err)
	}
	session.Wait()
}
