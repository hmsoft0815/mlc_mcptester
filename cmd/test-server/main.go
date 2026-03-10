package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Test-Icon (Ein kleiner grüner Kreis als SVG)
const serverIcon = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI0OCIgaGVpZ2h0PSI0OCIgdmlld0JveD0iMCAwIDQ4IDQ4Ij48Y2lyY2xlIGN4PSIyNCIgY3k9IjI0IiByPSIyMCIgZmlsbD0iIzRDRkY1MCIvPjwvc3ZnPg=="

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	addr := flag.String("addr", "", "Listen address for SSE (e.g. \":8080\"). If empty, uses stdio.")
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
				Logging:   &mcp.LoggingCapabilities{},
				Tools:     &mcp.ToolCapabilities{ListChanged: true},
				Prompts:   &mcp.PromptCapabilities{ListChanged: true},
				Resources: &mcp.ResourceCapabilities{ListChanged: true, Subscribe: true},
			},
		},
	)

	registerBasicTools(s)
	registerResources(s)
	registerPrompts(s)

	if *addr != "" {
		fmt.Fprintf(os.Stderr, "Starting Ultimate Test Server on SSE (%s)...\n", *addr)
		handler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return s }, nil)
		if err := http.ListenAndServe(*addr, handler); err != nil {
			log.Fatalf("SSE server failed: %v", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Starting Ultimate Test Server on stdio...\n")
		transport := &mcp.StdioTransport{}
		session, err := s.Connect(ctx, transport, nil)
		if err != nil {
			log.Fatal(err)
		}
		session.Wait()
	}
}

func registerBasicTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "echo",
		Description: "Echoes the input back to the user",
		Icons: []mcp.Icon{
			{Source: serverIcon, MIMEType: "image/svg+xml"},
		},
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string", "description": "The text to echo"},
			},
			"required": []string{"message"},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		msg, _ := args["message"].(string)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Echo: " + msg}},
		}, map[string]any{"echo": msg}, nil
	})

	// progressTest Tool (Simulation für Cancellation)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "progressTest",
		Description: "A long running tool to test progress and cancellation",
		Icons: []mcp.Icon{
			{Source: serverIcon, MIMEType: "image/svg+xml"},
		},
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"seconds": map[string]any{"type": "integer", "description": "Seconds to run"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		seconds := 10
		if v, ok := args["seconds"].(float64); ok {
			seconds = int(v)
		}

		for i := 0; i < seconds; i++ {
			select {
			case <-ctx.Done():
				fmt.Fprintf(os.Stderr, "[Server] Request cancelled!\n")
				return nil, nil, ctx.Err()
			default:
				if token := request.Params.GetProgressToken(); token != nil {
					_ = request.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
						Progress:      float64(i + 1),
						Total:         float64(seconds),
						ProgressToken: token,
					})
				}
				time.Sleep(1 * time.Second)
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Task finished"}},
		}, nil, nil
	})
}

func registerResources(s *mcp.Server) {
	s.AddResource(&mcp.Resource{
		Name:        "System Clock",
		URI:         "mcp://time",
		Description: "The current server time",
		Icons: []mcp.Icon{
			{Source: serverIcon, MIMEType: "image/svg+xml"},
		},
	}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: "mcp://time", Text: time.Now().String()}},
		}, nil
	})

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "Log File",
		URITemplate: "file:///logs/{name}.log",
		Description: "Access server log files by name",
	}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: request.Params.URI, Text: "Log content for " + request.Params.URI}},
		}, nil
	})
}

func registerPrompts(s *mcp.Server) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "persona_developer",
		Description: "Sets the LLM persona to an expert Go developer",
		Icons: []mcp.Icon{
			{Source: serverIcon, MIMEType: "image/svg+xml"},
		},
	}, func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "Persona instructions",
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{Text: "You are an expert Go developer."},
				},
			},
		}, nil
	})
}
