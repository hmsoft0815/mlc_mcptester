package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerBasicTools(s *mcp.Server) {
	mcp.AddTool(s, echoTool(), echo)
	mcp.AddTool(s, addTool(), add)
	mcp.AddTool(s, progressTestTool(), progressTest)
}

// serverIcons returns the server icon, a new slice for every tool.
func serverIcons() []mcp.Icon { return []mcp.Icon{{Source: serverIcon, MIMEType: "image/svg+xml"}} }

func echoTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "echo",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		Title:       "Echo",
		Description: "Echoes the input back to the user",
		Icons:       serverIcons(),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string", "description": "The text to echo"},
			},
			"required": []string{"message"},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"echo": map[string]any{"type": "string", "description": "The echoed text"},
			},
		},
	}
}

func echo(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	msg, _ := args["message"].(string)
	// Only delivered if the client asked for debug logs (per request since 2026-07-28)
	//lint:ignore SA1019 logging is deprecated since 2026-07-28 (SEP-2577); the reference server demonstrates it while clients support it; remove with T-20260927-05
	_ = request.Session.Log(ctx, &mcp.LoggingMessageParams{Level: "debug", Logger: "echo", Data: "echo called with " + msg})
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Echo: " + msg}},
	}, map[string]any{"echo": msg}, nil
}

// addTool has an output schema.
func addTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "add",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		Title:       "Add Numbers",
		Description: "Adds two numbers together",
		Icons:       serverIcons(),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{"type": "integer", "description": "The first number"},
				"b": map[string]any{"type": "integer", "description": "The second number"},
			},
			"required": []string{"a", "b"},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"sum": map[string]any{"type": "integer", "description": "The sum of a and b"},
				"a":   map[string]any{"type": "integer", "description": "The first number"},
				"b":   map[string]any{"type": "integer", "description": "The second number"},
			},
		},
	}
}

func add(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	aRaw, okA := args["a"]
	bRaw, okB := args["b"]
	if !okA || !okB {
		return nil, nil, fmt.Errorf("invalid params: missing required parameters 'a' and 'b'")
	}
	a, b := intArg(aRaw), intArg(bRaw)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Result: %d", a+b)}},
	}, map[string]any{"sum": a + b, "a": a, "b": b}, nil
}

// intArg reads an integer argument, which arrives as float64 from JSON.
func intArg(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

// progressTestTool simulates a long run, for progress and cancellation.
func progressTestTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "progressTest",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		Title:       "Progress Test",
		Description: "A long running tool to test progress and cancellation",
		Icons:       serverIcons(),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"seconds": map[string]any{"type": "integer", "description": "Seconds to run"},
				"count":   map[string]any{"type": "integer", "description": "Count to run"},
			},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{"type": "string", "description": "The completion status"},
			},
		},
	}
}

func progressTest(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
	seconds := 10
	if v, ok := args["seconds"].(float64); ok {
		seconds = int(v)
	} else if v, ok := args["count"].(float64); ok {
		seconds = int(v)
	}
	for i := 1; i <= seconds; i++ {
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "[Server] Request cancelled!\n")
			return nil, nil, ctx.Err()
		default:
			if token := request.Params.GetProgressToken(); token != nil {
				_ = request.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
					Progress:      float64(i),
					Total:         float64(seconds),
					ProgressToken: token,
				})
			}
			time.Sleep(1 * time.Second)
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Countdown finished!"}},
	}, map[string]any{"status": "completed"}, nil
}
