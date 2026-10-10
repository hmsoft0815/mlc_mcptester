package main

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFormatAnnotations(t *testing.T) {
	yes, no := true, false
	tests := map[string]struct {
		a    *mcp.ToolAnnotations
		want string
	}{
		"defaults only": {&mcp.ToolAnnotations{}, "readOnlyHint=false, idempotentHint=false"},
		"read only":     {&mcp.ToolAnnotations{ReadOnlyHint: true}, "readOnlyHint=true, idempotentHint=false"},
		"pointers set":  {&mcp.ToolAnnotations{DestructiveHint: &yes, OpenWorldHint: &no}, "readOnlyHint=false, destructiveHint=true, idempotentHint=false, openWorldHint=false"},
		"title":         {&mcp.ToolAnnotations{Title: "Delete", IdempotentHint: true}, `title="Delete", readOnlyHint=false, idempotentHint=true`},
	}
	for name, tt := range tests {
		if got := formatAnnotations(tt.a); got != tt.want {
			t.Errorf("%s: %q, want %q", name, got, tt.want)
		}
	}
}
