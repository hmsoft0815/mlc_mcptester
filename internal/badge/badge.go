// Package badge renders the "mcpcheck" status badge a project can put in its
// README to show that its MCP server passed mcp-tester.
//
// The SVG is self-contained, so the badge works from a plain repository file
// without any badge service. The JSON variant follows the shields.io endpoint
// schema for projects that prefer img.shields.io/endpoint.
package badge

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
)

// Label is the left-hand text of every badge.
const Label = "mcpcheck"

// Colours as shields.io uses them.
const (
	Green  = "#4c1"
	Yellow = "#dfb317"
	Red    = "#e05d44"
)

// warnBelow is the score under which a passing server gets a yellow badge.
const warnBelow = 80

// Result is what an inspection contributes to the badge.
type Result struct {
	// Revision is the protocol revision the server negotiated.
	Revision string
	// Latest is the newest revision mcp-tester checks against.
	Latest string
	Score  int
	// Errors counts protocol errors: MUST violations that break clients.
	Errors int
	// HideScore leaves the score off the badge, for projects that do not
	// want to show a number that rests on mcp-tester's own judgement.
	HideScore bool
	Tester    string
	Checked   time.Time
}

// Message is the right-hand text, e.g. "spec 2026-07-28 · 92/100". The
// "spec" prefix keeps the revision from being read as the test date.
func (r Result) Message() string {
	var parts []string
	if r.Revision != "" {
		parts = append(parts, "spec "+r.Revision)
	}
	switch {
	case r.Errors > 0:
		parts = append(parts, "failing")
	case !r.HideScore:
		parts = append(parts, fmt.Sprintf("%d/100", r.Score))
	case r.Revision == "":
		parts = append(parts, "passing")
	}
	return strings.Join(parts, " · ")
}

// Color is red for protocol errors — they outweigh any score — yellow for a
// low score or an outdated revision, green otherwise.
func (r Result) Color() string {
	switch {
	case r.Errors > 0:
		return Red
	case r.Score < warnBelow, r.Latest != "" && r.Revision != r.Latest:
		return Yellow
	default:
		return Green
	}
}

// Title is the tooltip: when and with which mcp-tester the check ran.
func (r Result) Title() string {
	return fmt.Sprintf("%s: %s — checked %s with mcp-tester %s",
		Label, r.Message(), r.Checked.UTC().Format("2006-01-02"), r.Tester)
}

// SVG renders the badge in the shields.io "flat" style.
func SVG(r Result) []byte {
	msg := r.Message()
	lw, mw := textWidth(Label)+10, textWidth(msg)+10
	total := lw + mw
	esc := html.EscapeString

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">`, total, Label, esc(msg))
	fmt.Fprintf(&b, `<title>%s</title>`, esc(r.Title()))
	b.WriteString(`<linearGradient id="s" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>`)
	fmt.Fprintf(&b, `<clipPath id="r"><rect width="%d" height="20" rx="3" fill="#fff"/></clipPath>`, total)
	fmt.Fprintf(&b, `<g clip-path="url(#r)"><rect width="%d" height="20" fill="#555"/><rect x="%d" width="%d" height="20" fill="%s"/><rect width="%d" height="20" fill="url(#s)"/></g>`,
		lw, lw, mw, r.Color(), total)
	b.WriteString(`<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">`)
	writeText(&b, Label, lw/2, lw-10)
	writeText(&b, msg, lw+mw/2, mw-10)
	b.WriteString(`</g></svg>`)
	return []byte(b.String())
}

// writeText writes a label with the shields.io drop shadow. textLength pins the
// width, so a fallback font cannot overflow the box.
func writeText(b *strings.Builder, s string, x, width int) {
	s = html.EscapeString(s)
	fmt.Fprintf(b, `<text x="%d" y="15" fill="#010101" fill-opacity=".3" textLength="%d">%s</text>`, x, width, s)
	fmt.Fprintf(b, `<text x="%d" y="14" textLength="%d">%s</text>`, x, width, s)
}

// Endpoint is the shields.io endpoint schema
// (https://shields.io/badges/endpoint-badge), extended by fields shields.io
// ignores but a reader of the file can check.
type Endpoint struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
	IsError       bool   `json:"isError,omitempty"`

	Revision string `json:"revision,omitempty"`
	Latest   string `json:"latestRevision,omitempty"`
	Score    int    `json:"score"`
	Errors   int    `json:"errors"`
	Tester   string `json:"tester"`
	Checked  string `json:"checked"`
}

// JSON renders the badge in the shields.io endpoint schema.
func JSON(r Result) ([]byte, error) {
	return json.MarshalIndent(Endpoint{
		SchemaVersion: 1,
		Label:         Label,
		Message:       r.Message(),
		Color:         strings.TrimPrefix(r.Color(), "#"),
		IsError:       r.Errors > 0,
		Revision:      r.Revision,
		Latest:        r.Latest,
		Score:         r.Score,
		Errors:        r.Errors,
		Tester:        r.Tester,
		Checked:       r.Checked.UTC().Format(time.RFC3339),
	}, "", "  ")
}

// textWidth estimates the rendered width of s in 11px Verdana. It need not be
// exact — textLength scales the glyphs — only close enough to look balanced.
func textWidth(s string) int {
	w := 0.0
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
			w += 7
		case c == ' ', c == '·', c == '.', c == ':':
			w += 3.9
		case c == '-', c == '/':
			w += 4.9
		case c == 'i', c == 'l', c == 'j', c == 't', c == 'f', c == 'r':
			w += 3.9
		case c == 'm', c == 'w':
			w += 10.7
		case c >= 'A' && c <= 'Z':
			w += 7.5
		default:
			w += 6.7
		}
	}
	return int(w + 0.5)
}
