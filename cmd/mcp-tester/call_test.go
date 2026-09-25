package main

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// B-20260925-04: --format json prints the complete result, isError included
func TestResultJSON(t *testing.T) {
	m, err := resultJSON(&mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: "hi"}},
		StructuredContent: map[string]any{"echo": "hi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if m["isError"] != false {
		t.Errorf("isError = %v; want false", m["isError"])
	}
	content, _ := m["content"].([]any)
	if len(content) != 1 || m["structuredContent"] == nil {
		t.Errorf("incomplete result: %v", m)
	}
	m, _ = resultJSON(&mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "boom"}}})
	if m["isError"] != true {
		t.Errorf("isError = %v; want true", m["isError"])
	}
}
