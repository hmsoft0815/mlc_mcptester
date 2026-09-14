package conformance

import (
	"strings"
	"testing"
)

var schema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"count": map[string]any{"type": "integer"},
		"name":  map[string]any{"type": "string"},
	},
	"required": []any{"count"},
}

func TestCheckToolResult(t *testing.T) {
	text := []any{map[string]any{"type": "text", "text": `{"count": 2}`}}

	tests := []struct {
		name    string
		schema  any
		result  map[string]any
		wantErr string // empty: must pass
	}{
		{
			name:   "no schema declared, nothing required",
			schema: nil,
			result: map[string]any{"content": text},
		},
		{
			name:    "schema declared, text only",
			schema:  schema,
			result:  map[string]any{"content": text},
			wantErr: "no structuredContent",
		},
		{
			name:    "schema declared, structuredContent null",
			schema:  schema,
			result:  map[string]any{"content": text, "structuredContent": nil},
			wantErr: "no structuredContent",
		},
		{
			name:   "matching structuredContent",
			schema: schema,
			result: map[string]any{"content": text, "structuredContent": map[string]any{"count": float64(2), "name": "x"}},
		},
		{
			name:   "an integer decoded as a Go int still counts as an integer",
			schema: schema,
			result: map[string]any{"structuredContent": map[string]any{"count": 2}},
		},
		{
			name:    "a required field is missing",
			schema:  schema,
			result:  map[string]any{"structuredContent": map[string]any{"name": "x"}},
			wantErr: "does not match",
		},
		{
			name:    "a field has the wrong type",
			schema:  schema,
			result:  map[string]any{"structuredContent": map[string]any{"count": "two"}},
			wantErr: "does not match",
		},
		{
			name:   "a tool error is exempt",
			schema: schema,
			result: map[string]any{"content": text, "isError": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckToolResult(tt.schema, tt.result)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Errorf("passed, want an error containing %q", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Errorf("error %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
