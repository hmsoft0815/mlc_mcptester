package scripting

import (
	"reflect"
	"strings"
	"testing"
)

func TestReplaceVariables(t *testing.T) {
	r := &Runner{
		variables: map[string]string{
			"FOO": "bar",
			"ID":  "123",
			"w":   "1024",
			"w2":  "123",
		},
	}

	tests := []struct {
		input       string
		expected    string
		expectError bool
	}{
		{"hello $FOO", "hello bar", false},
		{"id is $ID", "id is 123", false},
		{"no var here", "no var here", false},
		{"$FOO$ID", "bar123", false},
		{"$w2 and $w", "123 and 1024", false},
		{"$w and $w2", "1024 and 123", false},
		{"mixed $FOO and $UNKNOWN", "", true},
	}

	for _, tt := range tests {
		result, err := r.replaceVariables(tt.input)
		if (err != nil) != tt.expectError {
			t.Errorf("replaceVariables(%q) error = %v; expectError %v", tt.input, err, tt.expectError)
			continue
		}
		if !tt.expectError && result != tt.expected {
			t.Errorf("replaceVariables(%q) = %q; want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractValue(t *testing.T) {
	r := &Runner{
		lastRawMap: map[string]any{
			"id":   42,
			"name": "tester",
			"nested": map[string]any{
				"key":  "value",
				"list": []any{"a", "b", "c"},
			},
			"content": []any{
				map[string]any{"text": "hello"},
			},
			"structuredContent": map[string]any{
				"size": "512x512",
				"id":   7,
			},
		},
	}

	tests := []struct {
		path     string
		expected any
		wantErr  bool
	}{
		{"id", 42, false},
		{"$.size", "512x512", false},
		{"$.id", 7, false}, // structuredContent wins over a top-level field of the same name
		{"structuredContent.size", "512x512", false},
		{"$.name", "tester", false}, // falls back to the top level
		{"name", "tester", false},
		{"nested.key", "value", false},
		{"nested.list.1", "b", false},
		{"content.0.text", "hello", false},
		{"invalid.path", nil, true},
		{"nested.list.99", nil, true},
	}

	for _, tt := range tests {
		result, err := r.extractValue(tt.path)
		if (err != nil) != tt.wantErr {
			t.Errorf("extractValue(%q) error = %v; wantErr %v", tt.path, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("extractValue(%q) = %v; want %v", tt.path, result, tt.expected)
		}
	}
}

func TestConvertValue(t *testing.T) {
	tests := []struct {
		val      string
		schema   map[string]any
		expected any
	}{
		{"123", map[string]any{"type": "integer"}, 123},
		{"12.34", map[string]any{"type": "number"}, 12.34},
		{"true", map[string]any{"type": "boolean"}, true},
		{"false", map[string]any{"type": "boolean"}, false},
		{"1", map[string]any{"type": "boolean"}, true},
		{"0", map[string]any{"type": "boolean"}, false},
		{"some string", map[string]any{"type": "string"}, "some string"},
		{"123", map[string]any{"type": "string"}, "123"},
		{"not-a-number", map[string]any{"type": "integer"}, "not-a-number"}, // Fallback to string
		{`["a.png", "b.png"]`, map[string]any{"type": "array"}, []any{"a.png", "b.png"}},
		{`["a.png", "b.png"]`, map[string]any{"type": []any{"null", "array"}}, []any{"a.png", "b.png"}},
		{`{"key": "value"}`, map[string]any{"type": "object"}, map[string]any{"key": "value"}},
		{`{"key": "value"}`, map[string]any{"type": []any{"null", "object"}}, map[string]any{"key": "value"}},
		{`[literal]`, map[string]any{"type": "string"}, `[literal]`},
	}

	for _, tt := range tests {
		result := convertValue(tt.val, tt.schema)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("convertValue(%q, %v) = %v (%T); want %v (%T)", tt.val, tt.schema, result, result, tt.expected, tt.expected)
		}
	}
}

func TestEchoCommand(t *testing.T) {
	r := &Runner{}
	if err := r.dispatchParts(nil, 0, []string{"echo", "Hello", "World"}); err != nil {
		t.Fatalf("dispatchParts(echo) error = %v", err)
	}
}

func TestParseComments(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"call_tool echo # comment", "call_tool echo"},
		{"call_tool echo // comment", "call_tool echo"},
		{"  call_tool add 1 2   ", "call_tool add 1 2"},
		{"# full line comment", ""},
		{"// another full line", ""},
	}

	for _, tt := range tests {
		line := strings.TrimSpace(tt.input)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			line = ""
		} else {
			if idx := strings.Index(line, " #"); idx != -1 {
				line = strings.TrimSpace(line[:idx])
			}
			if idx := strings.Index(line, " //"); idx != -1 {
				line = strings.TrimSpace(line[:idx])
			}
		}

		if line != tt.expected {
			t.Errorf("parsing %q = %q; want %q", tt.input, line, tt.expected)
		}
	}
}
