package mcpdocs

import (
	"strings"
	"unicode"
)

func (w *writer) prompts(prompts []Prompt) {
	w.heading(2, "Prompts")
	if len(prompts) == 0 {
		w.p("The server declares prompts but lists none.\n\n")
		return
	}
	w.p("`prompts/get` returns the finished messages; the client runs them with the tools above.\n\n")
	for _, p := range prompts {
		w.heading(3, cell(p.Name))
		if p.Title != "" {
			w.p("**%s**\n\n", cell(p.Title))
		}
		if d := strings.TrimSpace(p.Description); d != "" {
			w.p("%s\n\n", d)
		}
		if len(p.Arguments) == 0 {
			w.p("No arguments.\n\n")
			continue
		}
		w.p("| Argument | Required | Description |\n|---|---|---|\n")
		for _, a := range p.Arguments {
			w.p("| %s | %s | %s |\n", code(a.Name), yes(a.Required), cell(a.Description))
		}
		w.p("\n")
	}
}

func (w *writer) resources(resources, templates []Resource) {
	w.heading(2, "Resources")
	if len(resources) == 0 && len(templates) == 0 {
		w.p("The server declares resources but lists none.\n\n")
		return
	}
	resourceTable(w, "URI", resources, func(r Resource) string { return r.URI })
	if len(templates) > 0 {
		w.heading(3, "Resource templates")
		resourceTable(w, "URI template", templates, func(r Resource) string { return r.URITemplate })
	}
}

func resourceTable(w *writer, column string, list []Resource, uri func(Resource) string) {
	if len(list) == 0 {
		return
	}
	w.p("| %s | Name | MIME type | Description |\n|---|---|---|---|\n", column)
	for _, r := range list {
		name := r.Name
		if r.Title != "" {
			name = r.Title
		}
		w.p("| %s | %s | %s | %s |\n", code(uri(r)), cell(name), cell(r.MIMEType), cell(r.Description))
	}
	w.p("\n")
}

// instructions writes the server's instructions verbatim in a fence longer
// than any backtick run inside them.
func (w *writer) instructions(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	w.heading(2, "Instructions")
	w.p("The server sends them with `initialize`; clients usually add them to the model's system prompt.\n\n")
	fence := strings.Repeat("`", max(3, longestRun(text, '`')+1))
	w.p("%stext\n%s\n%s\n\n", fence, strings.TrimRight(text, "\n"), fence)
}

func longestRun(s string, c rune) int {
	longest, run := 0, 0
	for _, r := range s {
		if r != c {
			run = 0
			continue
		}
		run++
		longest = max(longest, run)
	}
	return longest
}

// cell makes text fit a table cell: line breaks become <br>, | is escaped.
func cell(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	s = strings.ReplaceAll(s, "\n", "<br>")
	return strings.ReplaceAll(s, "|", `\|`)
}

// code writes an inline code span; a value that contains backticks gets a
// longer delimiter. | is escaped, which GitHub tables honour inside code too.
func code(s string) string {
	s = cell(s)
	if s == "" {
		return ""
	}
	tick := strings.Repeat("`", longestRun(s, '`')+1)
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		return tick + " " + s + " " + tick
	}
	return tick + s + tick
}

func yes(b bool) string {
	if b {
		return "yes"
	}
	return ""
}

// anchor is the id GitHub gives a heading: lower case, spaces to hyphens,
// punctuation other than - and _ dropped.
func anchor(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}
