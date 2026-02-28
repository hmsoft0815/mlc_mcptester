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
				Prompts: &mcp.PromptCapabilities{ListChanged: true},
			},
		},
	)

	registerTools(s)
	registerResources(s)
	registerPrompts(s)

	transport := &mcp.StdioTransport{}
	session, err := s.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal(err)
	}
	session.Wait()
}

func registerTools(s *mcp.Server) {
	// Echo Tool with Output Schema
	mcp.AddTool(s, &mcp.Tool{
		Name:        "echo",
		Description: "Echoes the input back to the user",
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
				"echo": map[string]any{"type": "string"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		msg := args["message"].(string)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Echo: " + msg}},
		}, map[string]any{"echo": msg}, nil
	})

	// Add Tool with Output Schema
	mcp.AddTool(s, &mcp.Tool{
		Name:        "add",
		Description: "Adds two integers and returns the result",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{"type": "integer"},
				"b": map[string]any{"type": "integer"},
			},
			"required": []string{"a", "b"},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"sum": map[string]any{"type": "integer"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		var a, b int
		if v, ok := args["a"].(float64); ok {
			a = int(v)
		} else {
			a, _ = args["a"].(int)
		}
		if v, ok := args["b"].(float64); ok {
			b = int(v)
		} else {
			b, _ = args["b"].(int)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Result: %d", a+b)}},
		}, map[string]any{"sum": a + b, "a": a, "b": b}, nil
	})

	// Long Running Task with Output Schema
	mcp.AddTool(s, &mcp.Tool{
		Name:        "long_running_task",
		Description: "A task that reports progress and sends logs over time",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"seconds": map[string]any{"type": "integer", "description": "How long to run"},
			},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{"type": "string"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		seconds := 3
		if s, ok := args["seconds"].(float64); ok {
			seconds = int(s)
		}

		_ = request.Session.Log(ctx, &mcp.LoggingMessageParams{
			Level:  "info",
			Logger: "task-worker",
			Data:   fmt.Sprintf("Starting task for %d seconds", seconds),
		})

		for i := 1; i <= seconds; i++ {
			time.Sleep(100 * time.Millisecond)
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
		}, map[string]any{"status": "success"}, nil
	})
}

func registerResources(s *mcp.Server) {
	s.AddResource(&mcp.Resource{
		Name:        "System Clock",
		URI:         "mcp://time",
		Description: "The current server time",
	}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: "mcp://time", Text: time.Now().String()}},
		}, nil
	})
}

func registerPrompts(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "persona_developer",
		Description: "Sets the LLM persona to an expert Go developer",
		Arguments: []*mcp.PromptArgument{
			{Name: "expertise", Description: "Specific field like 'backend' or 'cli'", Required: false},
		},
	}, func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		expertise := request.Params.Arguments["expertise"]
		if expertise == "" {
			expertise = "general"
		}

		return &mcp.GetPromptResult{
			Description: "Persona instructions",
			Messages: []*mcp.PromptMessage{
				{
					Role: "system",
					Content: &mcp.TextContent{
						Text: fmt.Sprintf("You are an expert Go developer with %s expertise. Always follow best practices and provide clean, idiomatic code.", expertise),
					},
				},
			},
		}, nil
	})
}
