package mcpskills

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListParams are the params of skills/list.
type ListParams struct {
	mcp.ParamsBase
	Cursor string `json:"cursor,omitempty"`
}

// ListResult is the result of skills/list.
type ListResult struct {
	mcp.ResultBase
	ResultType string  `json:"resultType"`
	Skills     []Skill `json:"skills"`
	NextCursor string  `json:"nextCursor,omitempty"`
	TTLMs      int     `json:"ttlMs"`
	CacheScope string  `json:"cacheScope"`
}

// URIParams are the params of skills/get and resources/directory/read.
type URIParams struct {
	mcp.ParamsBase
	URI    string `json:"uri"`
	Cursor string `json:"cursor,omitempty"`
}

// GetResult is the result of skills/get.
type GetResult struct {
	mcp.ResultBase
	ResultType string `json:"resultType"`
	Skill      Skill  `json:"skill"`
	TTLMs      int    `json:"ttlMs"`
	CacheScope string `json:"cacheScope"`
}

// DirectoryResult is the result of resources/directory/read.
type DirectoryResult struct {
	mcp.ResultBase
	ResultType string          `json:"resultType"`
	Resources  []*mcp.Resource `json:"resources"`
	NextCursor string          `json:"nextCursor,omitempty"`
}

func (c *catalog) list(ctx context.Context, _ *mcp.ServerSession, p *ListParams) (*ListResult, error) {
	skills := c.skills
	if skills == nil {
		skills = []Skill{}
	}
	return &ListResult{ResultType: "complete", Skills: skills, TTLMs: c.ttlMs, CacheScope: "public"}, nil
}

func (c *catalog) get(ctx context.Context, _ *mcp.ServerSession, p *URIParams) (*GetResult, error) {
	sk, ok := c.byURI[p.URI]
	if !ok {
		return nil, &jsonrpc.Error{Code: codeInvalidParams, Message: fmt.Sprintf("not a skill served here: %q", p.URI)}
	}
	return &GetResult{ResultType: "complete", Skill: sk, TTLMs: c.ttlMs, CacheScope: "public"}, nil
}

func (c *catalog) readDir(ctx context.Context, _ *mcp.ServerSession, p *URIParams) (*DirectoryResult, error) {
	children, ok := c.dirs[strings.TrimSuffix(p.URI, "/")]
	if !ok {
		return nil, &jsonrpc.Error{Code: codeInvalidParams, Message: fmt.Sprintf("not a directory resource: %q", p.URI)}
	}
	sorted := append([]*mcp.Resource{}, children...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].URI < sorted[j].URI })
	return &DirectoryResult{ResultType: "complete", Resources: sorted}, nil
}
