package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	mcpclient "github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// cliResponder answers input requests (elicitation, sampling) for the one-shot
// commands from --elicit / --sample; unanswered elicitations are declined.
var cliResponder = &mcpclient.Responder{Out: os.Stderr}

// getClient returns a new MCP client with optional logging and notification handlers.
func getClient(verbose bool) *mcp.Client {
	return newClient(verbose, cliResponder, nil)
}

// newClient builds the client with responder answering input requests and the
// roots given by --root. With notes, list-changed and resource-updated
// notifications are recorded, which makes the SDK open subscriptions/listen.
func newClient(verbose bool, responder *mcpclient.Responder, notes *mcpclient.Notifications) *mcp.Client {
	opts := &mcp.ClientOptions{
		// Handler for logging notifications from the server
		LoggingMessageHandler: func(ctx context.Context, req *mcp.LoggingMessageRequest) {
			fmt.Printf("[SERVER LOG] [%s] %s: %v\n", req.Params.Level, req.Params.Logger, req.Params.Data)
		},
		// Handler for progress notifications from the server
		ProgressNotificationHandler: func(ctx context.Context, req *mcp.ProgressNotificationClientRequest) {
			fmt.Printf("[PROGRESS] Token: %v, Done: %.2f, Total: %.2f, Msg: %s\n", req.Params.ProgressToken, req.Params.Progress, req.Params.Total, req.Params.Message)
		},
	}
	if verbose {
		opts.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	responder.Install(opts)
	if notes != nil {
		notes.Install(opts)
	}

	c := mcp.NewClient(
		&mcp.Implementation{
			Name:    version.AppName,
			Version: version.Version,
		},
		opts,
	)
	for _, uri := range rootURIs {
		c.AddRoots(&mcp.Root{URI: uri})
	}
	return c
}

// getTransport returns the appropriate MCP transport based on the provided command or URL.
// It supports CommandTransport for local execution, SSEClientTransport for SSE endpoints,
// and StreamableClientTransport for Streamable HTTP endpoints.
func getTransport(ctx context.Context, command, url string) (mcp.Transport, error) {
	if command != "" {
		return &mcp.CommandTransport{
			Command: exec.CommandContext(ctx, "sh", "-c", command),
		}, nil
	}
	if url != "" {
		httpClient, err := httpClientWithAuth()
		if err != nil {
			return nil, err
		}
		tType := strings.ToLower(strings.TrimSpace(transportType))
		if tType == "sse" || (tType == "" && strings.HasSuffix(strings.TrimRight(url, "/"), "/sse")) {
			if oauthEnabled {
				return nil, fmt.Errorf("--oauth needs the Streamable HTTP transport; the legacy SSE transport has no OAuth support")
			}
			return &mcp.SSEClientTransport{
				Endpoint:   url,
				HTTPClient: httpClient,
			}, nil
		}
		// Default to Streamable HTTP transport for HTTP URLs, or when explicitly requested
		t := &mcp.StreamableClientTransport{
			Endpoint:   url,
			HTTPClient: httpClient,
		}
		if oauthEnabled {
			if t.OAuthHandler, err = newOAuthHandler(); err != nil {
				return nil, fmt.Errorf("setting up OAuth: %w", err)
			}
		}
		return t, nil
	}
	return nil, fmt.Errorf("either --command or --url is required")
}
