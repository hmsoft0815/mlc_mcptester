package scripting

import (
	"testing"
)

func TestPreprocessLine(t *testing.T) {
	r := &Runner{}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty line", "", ""},
		{"whitespace line", "   ", ""},
		{"simple command", "call_tool echo", "call_tool echo"},
		{"command with whitespace", "  call_tool echo  ", "call_tool echo"},
		{"comment line #", "# this is a comment", ""},
		{"comment line //", "// this is a comment", ""},
		{"trailing comment #", "call_tool echo # comment", "call_tool echo"},
		{"trailing comment //", "call_tool echo // comment", "call_tool echo"},
		{"complex line", "  assert_contains \"Hello World\" // check result  ", "assert_contains \"Hello World\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.preprocessLine(tt.input)
			if got != tt.expected {
				t.Errorf("preprocessLine(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
