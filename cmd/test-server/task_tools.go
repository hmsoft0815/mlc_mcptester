package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/pkg/mcptasks"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type askNameResult struct {
	Answered bool   `json:"answered" jsonschema:"Whether the user gave a name"`
	Name     string `json:"name,omitempty" jsonschema:"The name the user gave"`
}

// registerTaskTools adds tools that run as tasks (io.modelcontextprotocol/tasks)
// for clients that declare the extension, and synchronously otherwise. With
// broken set, every tasks/* request fails with -32603 (wrong on purpose).
func registerTaskTools(s *mcp.Server, broken bool) {
	// No output schema: the result only reports that the job finished.
	mcp.AddTool(s, &mcp.Tool{
		Name:        "long_job",
		Title:       "Long Job",
		Description: "Works for the given number of seconds; runs as a task if the client supports tasks",
	}, longJob)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "ask_name",
		Title:       "Ask Name",
		Description: "Asks for the user's name: inside a task via input_required, otherwise via multi round-trip",
	}, askName)
	// No output schema: it never returns a result, only a JSON-RPC error.
	mcp.AddTool(s, &mcp.Tool{
		Name:        "failing_job",
		Title:       "Failing Job",
		Description: "Fails with a JSON-RPC error; as a task it ends in status failed",
	}, failingJob)
	// No output schema: the result is a tool error, nothing structured.
	mcp.AddTool(s, &mcp.Tool{
		Name:        "tool_error_job",
		Title:       "Tool Error Job",
		Description: "Returns a tool error (isError); as a task it ends completed, not failed",
	}, toolErrorJob)

	store := &mcptasks.Store{TTL: time.Hour, PollInterval: 200 * time.Millisecond}
	if err := mcptasks.Enable(s, store, "long_job", "ask_name", "failing_job", "tool_error_job"); err != nil {
		log.Fatalf("enabling tasks: %v", err)
	}
	if broken {
		s.AddReceivingMiddleware(breakTasks)
	}
}

type longJobArgs struct {
	Seconds float64 `json:"seconds" jsonschema:"How long to work (default 1)"`
}

func longJob(ctx context.Context, req *mcp.CallToolRequest, args longJobArgs) (*mcp.CallToolResult, any, error) {
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
}

// askNameQuestion returns the elicitation of ask_name, new for every call.
func askNameQuestion() *mcp.ElicitParams {
	return &mcp.ElicitParams{
		Mode:    "form",
		Message: "What is your name?",
		RequestedSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"name": map[string]any{"type": "string"}},
			"required":   []string{"name"},
		},
	}
}

func askName(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, askNameResult, error) {
	var none askNameResult
	question := askNameQuestion()
	var answer mcp.InputResponse
	if mcptasks.IsTask(ctx) {
		responses, err := mcptasks.RequestInput(ctx, mcp.InputRequestMap{"name": question})
		if err != nil {
			return nil, none, err
		}
		answer = responses["name"]
	} else if resp, ok := req.Params.InputResponses["name"]; ok {
		answer = resp
	} else {
		return &mcp.CallToolResult{InputRequests: mcp.InputRequestMap{"name": question}}, none, nil
	}
	res, ok := answer.(*mcp.ElicitResult)
	if !ok || res.Action != "accept" {
		return textResult("No name given"), none, nil
	}
	name := fmt.Sprint(res.Content["name"])
	return textResult("Hello, " + name + "!"), askNameResult{Answered: true, Name: name}, nil
}

func failingJob(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	return nil, nil, &jsonrpc.Error{Code: -32603, Message: "the backend is not reachable"}
}

func toolErrorJob(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	res := textResult("the input could not be processed")
	res.IsError = true
	return res, nil, nil
}

// breakTasks fails every tasks/* request with an internal error.
func breakTasks(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		if strings.HasPrefix(method, "tasks/") {
			return nil, &jsonrpc.Error{Code: -32603, Message: "tasks are broken on purpose (-broken-tasks)"}
		}
		return next(ctx, method, req)
	}
}
