// Package mcpdocs writes a Markdown reference of an MCP server — its tools,
// prompts, resources and instructions — from what the server sends over the
// protocol, so it works for any server regardless of language and SDK.
package mcpdocs

import (
	"context"
	"encoding/json"
	"fmt"

	mcpclient "github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxPages stops a server whose nextCursor never runs out.
const maxPages = 1000

// Server is what the reference is written from.
type Server struct {
	Name         string
	Instructions string
	Tools        []Tool
	Prompts      []Prompt
	Resources    []Resource
	Templates    []Resource
	// Whether the server declares the capability; a section is written only then
	HasTools, HasPrompts, HasResources bool
}

// Tool is a tool as listed by tools/list; the schemas stay raw so their
// properties keep the server's order.
type Tool struct {
	Name         string          `json:"name"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"inputSchema"`
	OutputSchema json.RawMessage `json:"outputSchema"`
	Annotations  *struct {
		Title           string `json:"title"`
		ReadOnlyHint    *bool  `json:"readOnlyHint"`
		DestructiveHint *bool  `json:"destructiveHint"`
		IdempotentHint  *bool  `json:"idempotentHint"`
		OpenWorldHint   *bool  `json:"openWorldHint"`
	} `json:"annotations"`
}

// Prompt is a prompt as listed by prompts/list.
type Prompt struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Arguments   []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Required    bool   `json:"required"`
	} `json:"arguments"`
}

// Resource is a resource or a resource template.
type Resource struct {
	URI         string `json:"uri"`
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	MIMEType    string `json:"mimeType"`
}

// Fetch reads everything the reference shows from a connected session. A
// list fails the fetch only if the server declares its capability.
func Fetch(ctx context.Context, session *mcp.ClientSession) (*Server, error) {
	init := session.InitializeResult()
	caps := init.Capabilities
	srv := &Server{
		Name:         init.ServerInfo.Name,
		Instructions: init.Instructions,
		HasTools:     caps.Tools != nil,
		HasPrompts:   caps.Prompts != nil,
		HasResources: caps.Resources != nil,
	}
	if init.ServerInfo.Title != "" {
		srv.Name = init.ServerInfo.Title
	}
	var err error
	if srv.HasTools {
		if srv.Tools, err = listAll[Tool](ctx, session, "tools/list", "tools"); err != nil {
			return nil, err
		}
	}
	if srv.HasPrompts {
		if srv.Prompts, err = listAll[Prompt](ctx, session, "prompts/list", "prompts"); err != nil {
			return nil, err
		}
	}
	if srv.HasResources {
		if srv.Resources, err = listAll[Resource](ctx, session, "resources/list", "resources"); err != nil {
			return nil, err
		}
		if srv.Templates, err = listAll[Resource](ctx, session, "resources/templates/list", "resourceTemplates"); err != nil {
			return nil, err
		}
	}
	return srv, nil
}

// listAll follows nextCursor through every page of a list method, reading
// the raw result.
func listAll[T any](ctx context.Context, session *mcp.ClientSession, method, field string) ([]T, error) {
	var all []T
	cursor := ""
	for range maxPages {
		params := map[string]any{"_meta": mcpclient.RequestMeta(session, nil)}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, err := mcpclient.CallRawJSON(ctx, session, method, params)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", method, err)
		}
		var page map[string]json.RawMessage
		if err := json.Unmarshal(raw, &page); err != nil {
			return nil, fmt.Errorf("%s: %w", method, err)
		}
		var items []T
		if err := json.Unmarshal(page[field], &items); err != nil && page[field] != nil {
			return nil, fmt.Errorf("%s: reading %s: %w", method, field, err)
		}
		all = append(all, items...)
		var next string
		_ = json.Unmarshal(page["nextCursor"], &next)
		if next == "" || next == cursor {
			return all, nil
		}
		cursor = next
	}
	return nil, fmt.Errorf("%s: more than %d pages", method, maxPages)
}
