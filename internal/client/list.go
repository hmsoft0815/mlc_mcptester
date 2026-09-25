package client

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListAllTools follows nextCursor through every page of tools/list. A single
// ListTools call returns only the first page, so servers that paginate would
// seem to have fewer tools.
func ListAllTools(ctx context.Context, session *mcp.ClientSession) ([]*mcp.Tool, error) {
	var tools []*mcp.Tool
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// ListAllPrompts follows nextCursor through every page of prompts/list.
func ListAllPrompts(ctx context.Context, session *mcp.ClientSession) ([]*mcp.Prompt, error) {
	var prompts []*mcp.Prompt
	for prompt, err := range session.Prompts(ctx, nil) {
		if err != nil {
			return nil, err
		}
		prompts = append(prompts, prompt)
	}
	return prompts, nil
}

// ListAllResources follows nextCursor through every page of resources/list.
func ListAllResources(ctx context.Context, session *mcp.ClientSession) ([]*mcp.Resource, error) {
	var resources []*mcp.Resource
	for resource, err := range session.Resources(ctx, nil) {
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}
