package scripting

import (
	"reflect"
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

func TestParseArgs(t *testing.T) {
	tests := []struct {
		line    string
		want    []string
		wantErr bool
	}{
		{`call_tool add a:1 b:2`, []string{"call_tool", "add", "a:1", "b:2"}, false},
		{`assert_contains "two words"`, []string{"assert_contains", "two words"}, false},
		{`assert_contains 'single "quoted"'`, []string{"assert_contains", `single "quoted"`}, false},
		{`assert_contains "missing properties: [\"code\"]"`, []string{"assert_contains", `missing properties: ["code"]`}, false},
		{`echo "back\\slash"`, []string{"echo", `back\slash`}, false},
		{`echo "C:\temp"`, []string{"echo", `C:\temp`}, false},
		{`echo 'no \"escape\" here'`, []string{"echo", `no \"escape\" here`}, false},
		{`echo "unterminated`, nil, true},
	}

	r := &Runner{}
	for _, tt := range tests {
		got, err := r.parseArgs(tt.line)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseArgs(%q) error = %v; wantErr %v", tt.line, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseArgs(%q) = %q; want %q", tt.line, got, tt.want)
		}
	}
}
