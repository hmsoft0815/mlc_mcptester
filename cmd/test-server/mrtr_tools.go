package main

import (
	"context"
	"fmt"
	neturl "net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type confirmDeleteResult struct {
	Deleted bool   `json:"deleted" jsonschema:"Whether the item was deleted"`
	Reason  string `json:"reason,omitempty" jsonschema:"The user's reason, or why nothing was deleted"`
}

type connectAccountResult struct {
	Connected bool `json:"connected" jsonschema:"Whether the user completed the sign-in"`
}

type summarizeResult struct {
	Summary string `json:"summary" jsonschema:"The summary"`
	Model   string `json:"model" jsonschema:"The model that wrote it"`
}

type listRootsResult struct {
	Roots []string `json:"roots" jsonschema:"URIs of the client's roots"`
}

// registerInputTools adds tools that need input from the client via multi
// round-trip requests (SEP-2322): the first call returns inputRequests, the
// retry carries the client's inputResponses. For clients on older protocol
// versions the SDK turns these into server-to-client requests.
func registerInputTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "confirm_delete",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true)},
		Title:       "Confirm Delete",
		Description: "Asks the user to confirm before deleting an item (elicitation, form mode)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		Item string `json:"item" jsonschema:"Item to delete"`
	}) (*mcp.CallToolResult, confirmDeleteResult, error) {
		var none confirmDeleteResult
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
			}}, none, nil
		}
		res, ok := resp.(*mcp.ElicitResult)
		if !ok {
			return nil, none, fmt.Errorf("unexpected input response %T", resp)
		}
		switch {
		case res.Action == "accept" && res.Content["confirm"] == true:
			reason, _ := res.Content["reason"].(string)
			return textResult(fmt.Sprintf("Deleted %s (reason: %v)", args.Item, res.Content["reason"])),
				confirmDeleteResult{Deleted: true, Reason: reason}, nil
		case res.Action == "accept":
			return textResult("Not deleted: not confirmed"), confirmDeleteResult{Reason: "not confirmed"}, nil
		}
		return textResult("Not deleted: user chose " + res.Action), confirmDeleteResult{Reason: "user chose " + res.Action}, nil
	})

	// URL mode: the user leaves the client for a page of the server (e.g. a
	// third-party sign-in); nothing sensitive passes through the client.
	mcp.AddTool(s, &mcp.Tool{
		Name:        "connect_account",
		Title:       "Connect Account",
		Description: "Asks the user to sign in to a demo service in the browser (elicitation, URL mode)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		Service string `json:"service" jsonschema:"Service to connect"`
	}) (*mcp.CallToolResult, connectAccountResult, error) {
		var none connectAccountResult
		resp, ok := req.Params.InputResponses["signin"]
		if !ok {
			return &mcp.CallToolResult{InputRequests: mcp.InputRequestMap{
				"signin": &mcp.ElicitParams{
					Mode:          "url",
					Message:       fmt.Sprintf("Sign in to %s to connect your account", args.Service),
					URL:           "https://example.com/connect?service=" + neturl.QueryEscape(args.Service),
					ElicitationID: "signin-" + args.Service,
				},
			}}, none, nil
		}
		res, ok := resp.(*mcp.ElicitResult)
		if !ok {
			return nil, none, fmt.Errorf("unexpected input response %T", resp)
		}
		if res.Action != "accept" {
			return textResult("Not connected: user chose " + res.Action), none, nil
		}
		return textResult("Connected " + args.Service), connectAccountResult{Connected: true}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "summarize",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		Title:       "Summarize",
		Description: "Asks the client's LLM to summarize a text (sampling)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		Text string `json:"text" jsonschema:"Text to summarize"`
	}) (*mcp.CallToolResult, summarizeResult, error) {
		var none summarizeResult
		resp, ok := req.Params.InputResponses["llm"]
		if !ok {
			return &mcp.CallToolResult{InputRequests: mcp.InputRequestMap{
				//lint:ignore SA1019 sampling is deprecated since 2026-07-28 (SEP-2577); the reference server demonstrates it while clients support it; remove with T-20260927-05
				"llm": &mcp.CreateMessageParams{
					MaxTokens: 100,
					//lint:ignore SA1019 sampling is deprecated since 2026-07-28 (SEP-2577); the reference server demonstrates it while clients support it; remove with T-20260927-05
					Messages: []*mcp.SamplingMessage{{
						Role:    "user",
						Content: &mcp.TextContent{Text: "Summarize in one sentence: " + args.Text},
					}},
				},
			}}, none, nil
		}
		// The SDK hands sampling results over in either shape
		var model string
		var text *mcp.TextContent
		switch res := resp.(type) {
		//lint:ignore SA1019 sampling is deprecated since 2026-07-28 (SEP-2577); the reference server demonstrates it while clients support it; remove with T-20260927-05
		case *mcp.CreateMessageResult:
			model = res.Model
			text, _ = res.Content.(*mcp.TextContent)
		//lint:ignore SA1019 sampling is deprecated since 2026-07-28 (SEP-2577); the reference server demonstrates it while clients support it; remove with T-20260927-05
		case *mcp.CreateMessageWithToolsResult:
			model = res.Model
			for _, c := range res.Content {
				if t, ok := c.(*mcp.TextContent); ok {
					text = t
					break
				}
			}
		default:
			return nil, none, fmt.Errorf("unexpected input response %T", resp)
		}
		if text == nil {
			return nil, none, fmt.Errorf("sampling returned no text")
		}
		return textResult(fmt.Sprintf("Summary (%s): %s", model, text.Text)), summarizeResult{Summary: text.Text, Model: model}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_roots",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		Title:       "List Roots",
		Description: "Lists the client's roots (roots/list)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, listRootsResult, error) {
		var none listRootsResult
		resp, ok := req.Params.InputResponses["roots"]
		if !ok {
			return &mcp.CallToolResult{InputRequests: mcp.InputRequestMap{
				//lint:ignore SA1019 roots are deprecated since 2026-07-28 (SEP-2577); the reference server demonstrates it while clients support it; remove with T-20260927-05
				"roots": &mcp.ListRootsParams{},
			}}, none, nil
		}
		//lint:ignore SA1019 roots are deprecated since 2026-07-28 (SEP-2577); the reference server demonstrates it while clients support it; remove with T-20260927-05
		res, ok := resp.(*mcp.ListRootsResult)
		if !ok {
			return nil, none, fmt.Errorf("unexpected input response %T", resp)
		}
		uris := make([]string, len(res.Roots))
		for i, r := range res.Roots {
			uris[i] = r.URI
		}
		return textResult(fmt.Sprintf("Roots (%d): %s", len(uris), strings.Join(uris, ", "))), listRootsResult{Roots: uris}, nil
	})
}

func ptr[T any](v T) *T { return &v }

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}
