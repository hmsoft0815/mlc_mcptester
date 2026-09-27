package badge

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

var checked = time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)

func TestMessageAndColor(t *testing.T) {
	cases := []struct {
		name  string
		r     Result
		msg   string
		color string
	}{
		{"good", Result{Revision: "2026-07-28", Latest: "2026-07-28", Score: 92}, "spec 2026-07-28 · 92/100", Green},
		{"low score", Result{Revision: "2026-07-28", Latest: "2026-07-28", Score: 70}, "spec 2026-07-28 · 70/100", Yellow},
		{"old revision", Result{Revision: "2025-11-25", Latest: "2026-07-28", Score: 95}, "spec 2025-11-25 · 95/100", Yellow},
		{"errors beat score", Result{Revision: "2026-07-28", Latest: "2026-07-28", Score: 100, Errors: 1}, "spec 2026-07-28 · failing", Red},
		{"score hidden", Result{Revision: "2026-07-28", Latest: "2026-07-28", Score: 92, HideScore: true}, "spec 2026-07-28", Green},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.r.Message(); got != c.msg {
				t.Errorf("Message() = %q, want %q", got, c.msg)
			}
			if got := c.r.Color(); got != c.color {
				t.Errorf("Color() = %q, want %q", got, c.color)
			}
		})
	}
}

func TestSVG(t *testing.T) {
	r := Result{Revision: "2026-07-28", Latest: "2026-07-28", Score: 92, Tester: "1.6.0", Checked: checked}
	svg := string(SVG(r))
	for _, want := range []string{
		`<svg xmlns="http://www.w3.org/2000/svg"`,
		`>mcpcheck</text>`,
		`>spec 2026-07-28 · 92/100</text>`,
		`checked 2026-09-27 with mcp-tester 1.6.0`,
		`fill="` + Green + `"`,
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("SVG lacks %q:\n%s", want, svg)
		}
	}
}

func TestJSON(t *testing.T) {
	r := Result{Revision: "2026-07-28", Latest: "2026-07-28", Score: 100, Errors: 2, Tester: "1.6.0", Checked: checked}
	data, err := JSON(r)
	if err != nil {
		t.Fatal(err)
	}
	var e Endpoint
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatal(err)
	}
	if e.SchemaVersion != 1 || e.Label != "mcpcheck" || e.Color != "e05d44" || !e.IsError || e.Checked != "2026-09-27T10:00:00Z" {
		t.Errorf("unexpected endpoint: %+v", e)
	}
}
