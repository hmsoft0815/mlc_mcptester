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
		},
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"hello $FOO", "hello bar"},
		{"id is $ID", "id is 123"},
		{"no var here", "no var here"},
		{"$FOO$ID", "bar123"},
		{"mixed $FOO and $UNKNOWN", "mixed bar and $UNKNOWN"},
	}

	for _, tt := range tests {
		result := r.replaceVariables(tt.input)
		if result != tt.expected {
			t.Errorf("replaceVariables(%q) = %q; want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractValue(t *testing.T) {
	r := &Runner{
		lastRawMap: map[string]any{
			"id": 42,
			"name": "tester",
			"nested": map[string]any{
				"key": "value",
				"list": []any{"a", "b", "c"},
			},
			"content": []any{
				map[string]any{"text": "hello"},
			},
		},
	}

	tests := []struct {
		path     string
		expected any
		wantErr  bool
	}{
		{"id", 42, false},
		{"$.id", 42, false},
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
	}

	for _, tt := range tests {
		result := convertValue(tt.val, tt.schema)
		if !reflect.DeepEqual(result, tt.expected) {
			t.Errorf("convertValue(%q, %v) = %v (%T); want %v (%T)", tt.val, tt.schema, result, result, tt.expected, tt.expected)
		}
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
