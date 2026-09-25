package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTaskTools adds tools that run as tasks (io.modelcontextprotocol/tasks)
// for clients that declare the extension, and synchronously otherwise.
func registerTaskTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "long_job",
		Title:       "Long Job",
		Description: "Works for the given number of seconds; runs as a task if the client supports tasks",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		Seconds float64 `json:"seconds" jsonschema:"How long to work (default 1)"`
	}) (*mcp.CallToolResult, any, error) {
		d := time.Duration(args.Seconds * float64(time.Second))
		if d <= 0 {
			d = time.Second
		}
		select {
		case <-time.After(d):
			return textResult(fmt.Sprintf("Job finished after %s", d)), nil, nil
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ask_name",
		Title:       "Ask Name",
		Description: "Asks for the user's name: inside a task via input_required, otherwise via multi round-trip",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
		question := &mcp.ElicitParams{
			Mode:    "form",
			Message: "What is your name?",
			RequestedSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"name": map[string]any{"type": "string"}},
				"required":   []string{"name"},
			},
		}
		var answer mcp.InputResponse
		if mcptasks.IsTask(ctx) {
			responses, err := mcptasks.RequestInput(ctx, mcp.InputRequestMap{"name": question})
			if err != nil {
				return nil, nil, err
			}
			answer = responses["name"]
		} else if resp, ok := req.Params.InputResponses["name"]; ok {
			answer = resp
		} else {
			return &mcp.CallToolResult{InputRequests: mcp.InputRequestMap{"name": question}}, nil, nil
		}
		res, ok := answer.(*mcp.ElicitResult)
		if !ok || res.Action != "accept" {
			return textResult("No name given"), nil, nil
		}
		return textResult(fmt.Sprintf("Hello, %v!", res.Content["name"])), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "failing_job",
		Title:       "Failing Job",
		Description: "Fails with a JSON-RPC error; as a task it ends in status failed",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
		return nil, nil, &jsonrpc.Error{Code: -32603, Message: "the backend is not reachable"}
	})

	store := &mcptasks.Store{TTL: time.Hour, PollInterval: 200 * time.Millisecond}
	if err := mcptasks.Enable(s, store, "long_job", "ask_name", "failing_job"); err != nil {
		log.Fatalf("enabling tasks: %v", err)
	}
}
