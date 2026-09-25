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

// B-20260925-01: " #" and " //" inside quotes are not comments
func TestPreprocessLineKeepsQuotedHashes(t *testing.T) {
	r := &Runner{}
	tests := map[string]string{
		`call_tool x heading:"## Plan A > ### Phasen" content:"x"`: `call_tool x heading:"## Plan A > ### Phasen" content:"x"`,
		`assert_contains "a # b"`:                                  `assert_contains "a # b"`,
		`assert_contains 'a # b'`:                                  `assert_contains 'a # b'`,
		`assert_contains "http://x //y"  // real comment`:          `assert_contains "http://x //y"`,
		`echo "say \"# hi\"" # comment`:                            `echo "say \"# hi\""`,
		`echo plain # comment`:                                     `echo plain`,
	}
	for in, want := range tests {
		if got := r.preprocessLine(in); got != want {
			t.Errorf("preprocessLine(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestIndexOutsideQuotes(t *testing.T) {
	if i := indexOutsideQuotes(`call_tool x content:"a << b"`, "<<"); i != -1 {
		t.Errorf("found << inside quotes at %d", i)
	}
	if i := indexOutsideQuotes(`call_tool x <<EOF`, "<<"); i != 12 {
		t.Errorf("heredoc marker at %d, want 12", i)
	}
}

// B-20260925-03: an empty token stays an argument
func TestParseArgsKeepsEmptyTokens(t *testing.T) {
	r := &Runner{}
	for line, want := range map[string][]string{
		`assert_equals "" ""`:         {"assert_equals", "", ""},
		`assert_equals '' x`:          {"assert_equals", "", "x"},
		`assert_string_length "" 0 0`: {"assert_string_length", "", "0", "0"},
	} {
		got, err := r.parseArgs(line)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("parseArgs(%q) = %q, %v; want %q", line, got, err, want)
		}
	}
}

// B-20260925-03: variables are substituted per token, after tokenizing
func TestEmptyAndSpacedVariables(t *testing.T) {
	r := &Runner{variables: map[string]string{"empty": "", "spaced": `a "b" c`}}
	for _, line := range []string{`assert_equals $empty ""`, `assert_string_length $empty 0 0`} {
		parts, err := r.parseArgs(line)
		if err == nil {
			parts, err = r.replaceInParts(parts)
		}
		if err != nil || len(parts) < 3 || parts[1] != "" {
			t.Errorf("%q -> %q, %v; want the empty variable as argument 1", line, parts, err)
		}
	}
	parts, _ := r.parseArgs(`assert_equals $spaced x`)
	parts, _ = r.replaceInParts(parts)
	if len(parts) != 3 || parts[1] != `a "b" c` {
		t.Errorf("a value with spaces and quotes split up: %q", parts)
	}
}
