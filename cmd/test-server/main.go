package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcpskills"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Test-Icon (Ein kleiner grüner Kreis als SVG)
const serverIcon = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI0OCIgaGVpZ2h0PSI0OCIgdmlld0JveD0iMCAwIDQ4IDQ4Ij48Y2lyY2xlIGN4PSIyNCIgY3k9IjI0IiByPSIyMCIgZmlsbD0iIzRDRkY1MCIvPjwvc3ZnPg=="

// instructions reach the model via server/discover (initialize before 2026-07-28).
const instructions = `Reference server for mcp-tester: every tool demonstrates one protocol feature.
Tools ending in _job run as tasks when the client supports the Tasks extension.
confirm_delete, ask_name and connect_account ask the user; summarize asks the client's model; list_roots asks for roots.`

// cacheHints sets the cache-control fields (2026-07-28): lists may be cached
// for a minute by anyone — list_changed notifications announce changes —
// while resource contents are per user and stale at once, the clock changes.
func cacheHints(ctx context.Context, req mcp.Request, c *mcp.Cacheable) {
	switch req.(type) {
	case *mcp.ListToolsRequest, *mcp.ListPromptsRequest, *mcp.ListResourcesRequest, *mcp.ListResourceTemplatesRequest:
		c.TTLMs, c.CacheScope = 60_000, "public"
	case *mcp.ReadResourceRequest:
		c.TTLMs, c.CacheScope = 0, "private"
	}
}

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	addr := flag.String("addr", "", "Listen address for HTTP/SSE (e.g. \":8080\"). If empty, uses stdio.")
	withAuth := flag.Bool("auth", false, "With -addr: require OAuth bearer tokens and serve a built-in test authorization server")
	brokenTasks := flag.Bool("broken-tasks", false, "Answer every tasks/* request with an internal error, to test the Tasks checks")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Ultimate Test Server v%s\nAuthor: %s\n", version.Version, version.Author)
		return
	}
	s := newServer(*brokenTasks)
	if *addr != "" {
		serveHTTP(s, *addr, *withAuth)
	} else {
		serveStdio(s)
	}
}

// newServer builds the reference server with all its tools, resources,
// prompts, skills and apps.
func newServer(brokenTasks bool) *mcp.Server {
	caps := &mcp.ServerCapabilities{
		//lint:ignore SA1019 logging is deprecated since 2026-07-28 (SEP-2577); the reference server demonstrates it while clients support it; remove with T-20260927-05
		Logging:   &mcp.LoggingCapabilities{},
		Tools:     &mcp.ToolCapabilities{ListChanged: true},
		Prompts:   &mcp.PromptCapabilities{ListChanged: true},
		Resources: &mcp.ResourceCapabilities{ListChanged: true, Subscribe: true},
	}
	mcptasks.Declare(caps)
	mcpskills.Declare(caps, true)
	caps.AddExtension("io.modelcontextprotocol/ui", map[string]any{}) // MCP Apps, see app_tools.go
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "ultimate-test-server",
			Version: version.Version,
		},
		&mcp.ServerOptions{
			Capabilities:       caps,
			Instructions:       instructions,
			CompletionHandler:  complete,
			SubscribeHandler:   subscribe,
			UnsubscribeHandler: unsubscribe,
			// Small pages, so clients have to follow nextCursor: there are
			// more tools than fit on one page.
			PageSize:     5,
			SetCacheable: cacheHints,
		},
	)
	for _, register := range []func(*mcp.Server){
		registerBasicTools, registerExtraTools, registerInputTools, registerNotifyTools, registerHeaderTools,
	} {
		register(s)
	}
	registerTaskTools(s, brokenTasks)
	for _, register := range []func(*mcp.Server){registerSkills, registerApps, registerResources, registerPrompts} {
		register(s)
	}
	return s
}

// serveHTTP serves SSE at /sse and Streamable HTTP at /mcp, with withAuth
// behind the built-in test authorization server.
func serveHTTP(s *mcp.Server, addr string, withAuth bool) {
	fmt.Fprintf(os.Stderr, "Starting Ultimate Test Server on %s (SSE: /sse, Streamable HTTP: /mcp)...\n", addr)
	mux := http.NewServeMux()
	sseH, mcpH := mcpHandlers(s)
	if withAuth {
		base := "http://" + addr
		if strings.HasPrefix(addr, ":") {
			base = "http://127.0.0.1" + addr
		}
		as := newAuthServer(base, base+"/mcp")
		as.register(mux)
		sseH, mcpH = as.protect(sseH), as.protect(mcpH)
		fmt.Fprintf(os.Stderr, "OAuth enabled: issuer %s, static tokens %q and %q (second user), client %s/%s, test IdP %s/idp (ID token %q)\n", base, StaticToken, OtherStaticToken, TestClientID, TestClientSecret, base, TestIDToken)
	}
	mux.Handle("/sse", sseH)
	mux.Handle("/sse/", sseH)
	mux.Handle("/mcp", mcpH)
	mux.Handle("/mcp/", mcpH)
	mux.Handle("/", mcpH)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// mcpHandlers returns the SSE and the Streamable HTTP handler.
func mcpHandlers(s *mcp.Server) (sse, streamable http.Handler) {
	sseHandler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return s }, nil)
	// Stateless: the SDK serves protocol 2026-07-28 over HTTP only in this mode
	streamableHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{Stateless: true})
	// Reject foreign browser origins (DNS rebinding); the spec requires 403
	cop := http.NewCrossOriginProtection()
	return cop.Handler(sseHandler), cop.Handler(mcptasks.GuardListenHandler(streamableHandler))
}

func serveStdio(s *mcp.Server) {
	fmt.Fprintf(os.Stderr, "Starting Ultimate Test Server on stdio...\n")
	session, err := s.Connect(context.Background(), mcptasks.GuardListen(&mcp.StdioTransport{}), nil)
	if err != nil {
		log.Fatal(err)
	}
	session.Wait()
}
