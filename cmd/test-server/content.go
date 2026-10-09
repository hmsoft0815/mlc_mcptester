package main

import (
	"context"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
