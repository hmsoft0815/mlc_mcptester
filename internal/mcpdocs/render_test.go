package mcpdocs

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderKeepsPropertyOrderAndConstraints(t *testing.T) {
	srv := &Server{
		Name:     "demo",
		HasTools: true,
		Tools: []Tool{{
			Name:        "list_devices",
			Title:       "Devices",
			Description: "Lists devices.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{
				"zone_id":{"type":"string","format":"uuid","description":"only this zone"},
				"limit":{"type":"integer","minimum":1,"maximum":200,"default":100,"description":"max entries"},
				"kind":{"type":"string","enum":["a","b"],"description":"a | b\nor nothing"},
				"no":{"type":"string","pattern":"^EV-\\d{4}$"}
			},"required":["limit"]}`),
			OutputSchema: json.RawMessage(`{"type":"object","properties":{
				"devices":{"type":"array","items":{"type":"object","properties":{"id":{},"label":{}}}},
				"count":{"type":"integer"}
			}}`),
		}},
	}
	md, err := Render(srv)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"| [`list_devices`](#list_devices) | Devices | structured | — |",
		"| `zone_id` | string (uuid) |  | only this zone |\n| `limit` | integer (1–200, default `100`) | yes | max entries |",
		"| `kind` | string (`a`, `b`) |  | a \\| b<br>or nothing |",
		"| `no` | string (pattern `^EV-\\d{4}$`) |  |  |",
		"Result (`structuredContent`): `devices` (id, label), `count`",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in\n%s", want, md)
		}
	}
	if got := undescribed("list_devices", srv.Tools[0].InputSchema); len(got) != 1 || got[0] != "list_devices.no" {
		t.Errorf("undescribed = %v", got)
	}
}

func TestTypeText(t *testing.T) {
	tests := []struct{ schema, want string }{
		{`{"type":["string","null"]}`, "string \\| null"},
		{`{"type":"array","items":{"type":"integer"},"minItems":1}`, "array of integer (items ≥ 1)"},
		{`{"anyOf":[{"type":"string"},{"type":"number"}]}`, "string \\| number"},
		{`{"type":"number","exclusiveMinimum":0}`, "number (> 0)"},
		{`{"type":"string","minLength":1,"maxLength":64}`, "string (length 1–64)"},
		{`{"$ref":"#/$defs/Zone"}`, "Zone"},
		{`{}`, "any"},
		{`true`, "any"},
		{`{"type":"number","minimum":0,"exclusiveMinimum":true}`, "number (> 0)"},
		{`{"type":"string","minLength":1.0}`, "string (length ≥ 1)"},
		{`{"type":"string","pattern":"^(x|y)$","enum":["p|q"]}`, "string (`p\\|q`, pattern `^(x\\|y)$`)"},
		{`{"allOf":[{"$ref":"#/$defs/A"},{"$ref":"#/$defs/B"}]}`, "A & B"},
	}
	for _, tt := range tests {
		if got := parseSchema(json.RawMessage(tt.schema)).typeText(); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.schema, got, tt.want)
		}
	}
}

func TestRenderAnchorsInstructionsAndSections(t *testing.T) {
	srv := &Server{
		Name:         "demo",
		Instructions: "Use ```code``` blocks.",
		HasTools:     true,
		HasPrompts:   true,
		Tools:        []Tool{{Name: "tools", InputSchema: json.RawMessage(`{"type":"object"}`)}},
		Prompts:      []Prompt{{Name: "review"}},
	}
	md, err := Render(srv)
	if err != nil {
		t.Fatal(err)
	}
	// "## Tools" comes first, so the tool named tools gets the suffix
	if !strings.Contains(md, "[`tools`](#tools-1)") {
		t.Errorf("duplicate anchor not numbered:\n%s", md)
	}
	if !strings.Contains(md, "````text\nUse ```code``` blocks.\n````") {
		t.Errorf("instructions fence too short:\n%s", md)
	}
	if strings.Contains(md, "## Resources") {
		t.Error("resources section without the capability")
	}
	if !strings.Contains(md, "No parameters.") || !strings.Contains(md, "Result: text.") {
		t.Errorf("empty schemas not described:\n%s", md)
	}
}

func TestFirstDifference(t *testing.T) {
	if got := FirstDifference("a\r\nb\n", "a\nb\n"); got != 0 {
		t.Errorf("CRLF counted as difference: line %d", got)
	}
	if got := FirstDifference("a\nb\n", "a\nc\n"); got != 2 {
		t.Errorf("got line %d, want 2", got)
	}
	if got := FirstDifference("a\n", "a\nb\n"); got != 2 {
		t.Errorf("shorter file: got line %d, want 2", got)
	}
}

func TestRenderToleratesUnusualSchemas(t *testing.T) {
	srv := &Server{Name: "demo", HasTools: true, Tools: []Tool{{
		Name:         "_debug_",
		InputSchema:  json.RawMessage(`{"type":"object","properties":{"a":true,"b":{"type":"string","required":true}}}`),
		OutputSchema: json.RawMessage(`{"type":"object","properties":{"x":false}}`),
	}}}
	md, err := Render(srv)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"### \\_debug\\_", "[`_debug_`](#_debug_)", "| `a` | any |", "Result (`structuredContent`): `x`"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q in\n%s", want, md)
		}
	}
}

func TestAnchorsLikeGitHub(t *testing.T) {
	w := &writer{anchors: map[string]int{}}
	var got []string
	for _, h := range []string{"foo", "foo", "foo-1", "Get.User", "getuser", "éx²"} {
		got = append(got, w.reserve(h))
	}
	want := []string{"foo", "foo-1", "foo-1-1", "getuser", "getuser-1", "éx²"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("anchors = %v, want %v", got, want)
	}
}
