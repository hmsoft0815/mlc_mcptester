package scripting

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// argsRunner connects a runner to a server whose tool echoes its arguments.
func argsRunner(t *testing.T) *Runner {
	t.Helper()
	ctx := context.Background()
	s := mcp.NewServer(&mcp.Implementation{Name: "args", Version: "1"}, nil)
	mcp.AddTool(s, &mcp.Tool{
		Name: "update",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"description":  map[string]any{"type": "string"},
				"include_body": map[string]any{"type": "boolean"},
			},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(req.Params.Arguments)}}}, nil, nil
	})
	st, ct := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, st, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	r := NewRunner(cs, false)
	r.out = &strings.Builder{}
	return r
}

func runLine(r *Runner, line string) error {
	return r.processLine(context.Background(), 0, line, &runState{})
}

// B-20260925-02: an unknown name is a script error, not a positional value
func TestUnknownNamedArgument(t *testing.T) {
	r := argsRunner(t)
	err := runLine(r, `call_tool update include_bodyy:true`)
	var se *scriptError
	if !errors.As(err, &se) || !strings.Contains(err.Error(), `unknown argument "include_bodyy"`) || !strings.Contains(err.Error(), "description, include_body") {
		t.Fatalf("got %v; want a script error naming the unknown argument and the valid ones", err)
	}
	// expect_error must not hide it
	if err := runLine(r, `expect_error call_tool update author:"x"`); err == nil {
		t.Fatal("expect_error accepted a typo in an argument name")
	}
	// Too many positional values
	if err := runLine(r, `call_tool update "a" true extra`); !errors.As(err, &se) {
		t.Fatalf("got %v; want a script error for the extra argument", err)
	}
}

func TestNamedAndPositionalArguments(t *testing.T) {
	r := argsRunner(t)
	if err := runLine(r, `call_tool update include_body:true description:"## A # b"`); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.lastText, `"description":"## A # b"`) || !strings.Contains(r.lastText, `"include_body":true`) {
		t.Errorf("sent %s", r.lastText)
	}
	// A URL is a positional value, not a named argument "http"
	if err := runLine(r, `call_tool update http://example.com/x`); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.lastText, `"description":"http://example.com/x"`) {
		t.Errorf("sent %s", r.lastText)
	}
}

// call_tool_raw sends arguments unchecked, e.g. an unknown field
func TestCallToolRaw(t *testing.T) {
	r := argsRunner(t)
	if err := runLine(r, `call_tool_raw update '{"author": "x", "include_body": true}'`); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.lastText, `"author":"x"`) {
		t.Errorf("sent %s", r.lastText)
	}
	if err := runLine(r, `call_tool_raw update 'not json'`); err == nil {
		t.Error("accepted arguments that are not a JSON object")
	}
}

// B-20260925-03 end to end: empty values in assertions
func TestEmptyValueAssertions(t *testing.T) {
	r := argsRunner(t)
	r.variables["body"] = ""
	for _, line := range []string{`assert_equals $body ""`, `assert_string_length $body 0 0`, `assert_equals "" ""`} {
		if err := runLine(r, line); err != nil {
			t.Errorf("%s: %v", line, err)
		}
	}
}
