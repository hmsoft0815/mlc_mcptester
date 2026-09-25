package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerInputTools adds tools that need input from the client via multi
// round-trip requests (SEP-2322): the first call returns inputRequests, the
// retry carries the client's inputResponses. For clients on older protocol
// versions the SDK turns these into server-to-client requests.
func registerInputTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "confirm_delete",
		Title:       "Confirm Delete",
		Description: "Asks the user to confirm before deleting an item (elicitation, form mode)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		Item string `json:"item" jsonschema:"Item to delete"`
	}) (*mcp.CallToolResult, any, error) {
		resp, ok := req.Params.InputResponses["confirm"]
		if !ok {
			return &mcp.CallToolResult{InputRequests: mcp.InputRequestMap{
				"confirm": &mcp.ElicitParams{
					Mode:    "form",
					Message: fmt.Sprintf("Delete %q?", args.Item),
					RequestedSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"confirm": map[string]any{"type": "boolean", "description": "Really delete"},
							"reason":  map[string]any{"type": "string", "description": "Why"},
						},
						"required": []string{"confirm"},
					},
				},
			}}, nil, nil
		}
		res, ok := resp.(*mcp.ElicitResult)
		if !ok {
			return nil, nil, fmt.Errorf("unexpected input response %T", resp)
		}
		switch {
		case res.Action == "accept" && res.Content["confirm"] == true:
			return textResult(fmt.Sprintf("Deleted %s (reason: %v)", args.Item, res.Content["reason"])), nil, nil
		case res.Action == "accept":
			return textResult("Not deleted: not confirmed"), nil, nil
		}
		return textResult("Not deleted: user chose " + res.Action), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "summarize",
		Title:       "Summarize",
		Description: "Asks the client's LLM to summarize a text (sampling)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		Text string `json:"text" jsonschema:"Text to summarize"`
	}) (*mcp.CallToolResult, any, error) {
		resp, ok := req.Params.InputResponses["llm"]
		if !ok {
			return &mcp.CallToolResult{InputRequests: mcp.InputRequestMap{
				"llm": &mcp.CreateMessageParams{
					MaxTokens: 100,
					Messages: []*mcp.SamplingMessage{{
						Role:    "user",
						Content: &mcp.TextContent{Text: "Summarize in one sentence: " + args.Text},
					}},
				},
			}}, nil, nil
		}
		// The SDK hands sampling results over in either shape
		var model string
		var text *mcp.TextContent
		switch res := resp.(type) {
		case *mcp.CreateMessageResult:
			model = res.Model
			text, _ = res.Content.(*mcp.TextContent)
		case *mcp.CreateMessageWithToolsResult:
			model = res.Model
			for _, c := range res.Content {
				if t, ok := c.(*mcp.TextContent); ok {
					text = t
					break
				}
			}
		default:
			return nil, nil, fmt.Errorf("unexpected input response %T", resp)
		}
		if text == nil {
			return nil, nil, fmt.Errorf("sampling returned no text")
		}
		return textResult(fmt.Sprintf("Summary (%s): %s", model, text.Text)), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_roots",
		Title:       "List Roots",
		Description: "Lists the client's roots (roots/list)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
		resp, ok := req.Params.InputResponses["roots"]
		if !ok {
			return &mcp.CallToolResult{InputRequests: mcp.InputRequestMap{
				"roots": &mcp.ListRootsParams{},
			}}, nil, nil
		}
		res, ok := resp.(*mcp.ListRootsResult)
		if !ok {
			return nil, nil, fmt.Errorf("unexpected input response %T", resp)
		}
		uris := make([]string, len(res.Roots))
		for i, r := range res.Roots {
			uris[i] = r.URI
		}
		return textResult(fmt.Sprintf("Roots (%d): %s", len(uris), strings.Join(uris, ", "))), nil, nil
	})
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}
