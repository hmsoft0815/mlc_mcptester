package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	addr := flag.String("addr", "", "Listen address for SSE (e.g. ':8080'). If empty, uses stdio.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Ultimate Test Server v%s\n", version.Version)
		fmt.Fprintf(os.Stderr, "Developed by %s\n\n", version.Author)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("Ultimate Test Server v%s\nAuthor: %s\n", version.Version, version.Author)
		return
	}

	ctx := context.Background()
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ultimate-test-server",
			Version: version.Version,
		},
		&mcp.ServerOptions{
			Capabilities: &mcp.ServerCapabilities{
				Logging: &mcp.LoggingCapabilities{},
				Tools:   &mcp.ToolCapabilities{ListChanged: true},
				Prompts: &mcp.PromptCapabilities{ListChanged: true},
			},
		},
	)

	registerBasicTools(s)
	registerProcessTools(s)
	registerResources(s)
	registerPrompts(s)

	if *addr != "" {
		fmt.Fprintf(os.Stderr, "Starting Ultimate Test Server v%s (by %s) on SSE (%s)...\n", version.Version, version.Author, *addr)
		handler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return s }, nil)
		if err := http.ListenAndServe(*addr, handler); err != nil {
			log.Fatalf("SSE server failed: %v", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Starting Ultimate Test Server v%s (by %s) on stdio...\n", version.Version, version.Author)
		transport := &mcp.StdioTransport{}
		session, err := s.Connect(ctx, transport, nil)
		if err != nil {
			log.Fatal(err)
		}
		session.Wait()
	}
}

func registerBasicTools(s *mcp.Server) {
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
}

func registerProcessTools(s *mcp.Server) {

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

		paramsJSON, _ := json.Marshal(request.Params)
		_ = request.Session.Log(ctx, &mcp.LoggingMessageParams{
			Level:  "info",
			Logger: "task-worker",
			Data:   fmt.Sprintf("Starting task. Params: %s", string(paramsJSON)),
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

	// Progress & Cancellation Test Tool
	mcp.AddTool(s, &mcp.Tool{
		Name:        "progressTest",
		Description: "A tool to test progress notifications and request cancellation. Counts down from a start value.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"count": map[string]any{"type": "integer", "description": "Start value for the countdown", "default": 10},
			},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"finalStatus": map[string]any{"type": "string"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		count := 10
		if c, ok := args["count"].(float64); ok {
			count = int(c)
		}

		fmt.Fprintf(os.Stderr, "[progressTest] Starting countdown from %d\n", count)

		for i := count; i >= 0; i-- {
			// 1. Check for Cancellation
			select {
			case <-ctx.Done():
				fmt.Fprintf(os.Stderr, "[progressTest] Task cancelled by client at %d\n", i)
				return nil, nil, ctx.Err()
			default:
				// Continue
			}

			// 2. Log
			_ = request.Session.Log(ctx, &mcp.LoggingMessageParams{
				Level: "info",
				Data:  fmt.Sprintf("Countdown: %d", i),
			})

			// 3. Notify Progress
			if request.Params.GetProgressToken() != nil {
				_ = request.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
					Progress:      float64(count - i),
					Total:         float64(count),
					Message:       fmt.Sprintf("Counting down: %d", i),
					ProgressToken: request.Params.GetProgressToken(),
				})
			}

			time.Sleep(1 * time.Second)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Countdown finished!"}},
		}, map[string]any{"finalStatus": "completed"}, nil
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
