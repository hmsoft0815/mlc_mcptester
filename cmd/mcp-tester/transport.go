package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// getClient returns a new MCP client with optional logging and notification handlers.
func getClient(verbose bool) *mcp.Client {
	opts := &mcp.ClientOptions{
		// sometimes..
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

	return mcp.NewClient(
		&mcp.Implementation{
			Name:    "mcp-tester",
			Version: "0.1.0",
		},
		opts,
	)
}

// getTransport returns the appropriate MCP transport based on the provided command or URL.
// It supports CommandTransport for local execution and SSEClientTransport for remote URLs.
func getTransport(ctx context.Context, command, url string) (mcp.Transport, error) {
	// If a command is provided, use stdio transport via shell execution.
	// hmm - win/mac ?
	if command != "" {
		return &mcp.CommandTransport{
			Command: exec.CommandContext(ctx, "sh", "-c", command),
		}, nil
	}
	// If a URL is provided, use SSE (Server-Sent Events) transport.
	if url != "" {
		return &mcp.SSEClientTransport{
			Endpoint: url,
		}, nil
	}
	// Return an error if neither transport configuration is provided.
	return nil, fmt.Errorf("either --command or --url is required")
}
