package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcpskills"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Test-Icon (Ein kleiner grüner Kreis als SVG)
const serverIcon = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI0OCIgaGVpZ2h0PSI0OCIgdmlld0JveD0iMCAwIDQ4IDQ4Ij48Y2lyY2xlIGN4PSIyNCIgY3k9IjI0IiByPSIyMCIgZmlsbD0iIzRDRkY1MCIvPjwvc3ZnPg=="

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	addr := flag.String("addr", "", "Listen address for HTTP/SSE (e.g. \":8080\"). If empty, uses stdio.")
	withAuth := flag.Bool("auth", false, "With -addr: require OAuth bearer tokens and serve a built-in test authorization server")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Ultimate Test Server v%s\nAuthor: %s\n", version.Version, version.Author)
		return
	}

	ctx := context.Background()
	caps := &mcp.ServerCapabilities{
		Logging:   &mcp.LoggingCapabilities{},
		Tools:     &mcp.ToolCapabilities{ListChanged: true},
		Prompts:   &mcp.PromptCapabilities{ListChanged: true},
		Resources: &mcp.ResourceCapabilities{ListChanged: true, Subscribe: true},
	}
	mcptasks.Declare(caps)
	mcpskills.Declare(caps, true)
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ultimate-test-server",
			Version: version.Version,
		},
		&mcp.ServerOptions{
			Capabilities:       caps,
			CompletionHandler:  complete,
			SubscribeHandler:   subscribe,
			UnsubscribeHandler: unsubscribe,
		},
	)

	registerBasicTools(s)
	registerExtraTools(s)
	registerInputTools(s)
	registerNotifyTools(s)
	registerHeaderTools(s)
	registerTaskTools(s)
	registerSkills(s)
	registerResources(s)
	registerPrompts(s)

	if *addr != "" {
		fmt.Fprintf(os.Stderr, "Starting Ultimate Test Server on %s (SSE: /sse, Streamable HTTP: /mcp)...\n", *addr)
		mux := http.NewServeMux()
		sseHandler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return s }, nil)
		// Stateless: the SDK serves protocol 2026-07-28 over HTTP only in this mode
		streamableHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{Stateless: true})
		// Reject foreign browser origins (DNS rebinding); the spec requires 403
		cop := http.NewCrossOriginProtection()
		var sseH, mcpH http.Handler = cop.Handler(sseHandler), cop.Handler(streamableHandler)
		if *withAuth {
			base := "http://" + *addr
			if strings.HasPrefix(*addr, ":") {
				base = "http://127.0.0.1" + *addr
			}
			as := newAuthServer(base, base+"/mcp")
			as.register(mux)
			sseH, mcpH = as.protect(sseH), as.protect(mcpH)
			fmt.Fprintf(os.Stderr, "OAuth enabled: issuer %s, static token %q\n", base, StaticToken)
		}
		mux.Handle("/sse", sseH)
		mux.Handle("/sse/", sseH)
		mux.Handle("/mcp", mcpH)
		mux.Handle("/mcp/", mcpH)
		mux.Handle("/", mcpH)
		if err := http.ListenAndServe(*addr, mux); err != nil {
			log.Fatalf("Server failed: %v", err)
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
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"echo": map[string]any{"type": "string", "description": "The echoed text"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		msg, _ := args["message"].(string)
		// Only delivered if the client asked for debug logs (per request since 2026-07-28)
		_ = request.Session.Log(ctx, &mcp.LoggingMessageParams{Level: "debug", Logger: "echo", Data: "echo called with " + msg})
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Echo: " + msg}},
		}, map[string]any{"echo": msg}, nil
	})

	// Add Tool with Output Schema
	mcp.AddTool(s, &mcp.Tool{
		Name:        "add",
		Description: "Adds two numbers together",
		Icons: []mcp.Icon{
			{Source: serverIcon, MIMEType: "image/svg+xml"},
		},
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
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		aRaw, okA := args["a"]
		bRaw, okB := args["b"]
		if !okA || !okB {
			return nil, nil, fmt.Errorf("invalid params: missing required parameters 'a' and 'b'")
		}
		var a, b int
		if v, ok := aRaw.(float64); ok {
			a = int(v)
		} else if v, ok := aRaw.(int); ok {
			a = v
		}
		if v, ok := bRaw.(float64); ok {
			b = int(v)
		} else if v, ok := bRaw.(int); ok {
			b = v
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Result: %d", a+b)}},
		}, map[string]any{"sum": a + b, "a": a, "b": b}, nil
	})

	// progressTest Tool (Simulation für Progress und Cancellation)
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
				"count":   map[string]any{"type": "integer", "description": "Count to run"},
			},
		},
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{"type": "string", "description": "The completion status"},
			},
		},
	}, func(ctx context.Context, request *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
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
		Arguments: []*mcp.PromptArgument{{
			Name:        "language",
			Description: "Programming language of the persona (completable)",
		}},
		Icons: []mcp.Icon{
			{Source: serverIcon, MIMEType: "image/svg+xml"},
		},
	}, func(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "Persona instructions",
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: "You are an expert Go developer."},
				},
			},
		}, nil
	})
}

// completionValues are the candidates offered by complete, per reference and argument.
var completionValues = map[string][]string{
	"ref/prompt persona_developer language":     {"go", "golang", "python", "rust", "typescript"},
	"ref/resource file:///logs/{name}.log name": {"access", "app", "error"},
}

// complete serves completion/complete: candidates starting with the typed value.
func complete(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	ref := req.Params.Ref
	target := ref.Name
	if ref.Type == "ref/resource" {
		target = ref.URI
	}
	key := ref.Type + " " + target + " " + req.Params.Argument.Name

	values := []string{}
	for _, v := range completionValues[key] {
		if strings.HasPrefix(v, req.Params.Argument.Value) {
			values = append(values, v)
		}
	}
	return &mcp.CompleteResult{
		Completion: mcp.CompletionResultDetails{Values: values, Total: len(values)},
	}, nil
}
