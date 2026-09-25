package scripting

import "strings"

// preprocessLine trims whitespace and removes comments from a script line.
// A trailing comment (" #" or " //") counts only outside quotes, so "## A"
// and URLs stay intact inside string literals.
func (r *Runner) preprocessLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
		return ""
	}

	for _, marker := range []string{" #", " //"} {
		if idx := indexOutsideQuotes(line, marker); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
	}
	return line
}

// indexOutsideQuotes returns the first index of substr in line that is not
// inside a quoted string, or -1. Quoting follows parseArgs: inside double
// quotes \" and \\ are escapes, single quotes are verbatim.
func indexOutsideQuotes(line, substr string) int {
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote == '"' && c == '\\' && i+1 < len(line) && (line[i+1] == '"' || line[i+1] == '\\'):
			i++
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case strings.HasPrefix(line[i:], substr):
			return i
		}
	}
	return -1
}
