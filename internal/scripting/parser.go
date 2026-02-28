package scripting

import "strings"

// preprocessLine trims whitespace and removes comments from a script line.
func (r *Runner) preprocessLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
		return ""
	}

	// Remove trailing comments
	if idx := strings.Index(line, " #"); idx != -1 {
		line = strings.TrimSpace(line[:idx])
	}
	if idx := strings.Index(line, " //"); idx != -1 {
		line = strings.TrimSpace(line[:idx])
	}

	return line
}
