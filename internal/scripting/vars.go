package scripting

import (
	"fmt"
	"strconv"
	"strings"
)

func (r *Runner) replaceVariables(line string) string {
	for name, val := range r.variables {
		line = strings.ReplaceAll(line, "$"+name, val)
	}
	return line
}

// extractValue finds a value in the last response using dot notation (e.g., "structuredContent.0.id")
func (r *Runner) extractValue(path string) (any, error) {
	if path == "rawResponse" {
		return r.lastResponse, nil
	}

	trimmedPath := strings.TrimPrefix(path, "structuredContent.")
	trimmedPath = strings.TrimPrefix(trimmedPath, "$.")

	if r.lastRawMap == nil {
		return nil, fmt.Errorf("no previous response available")
	}

	parts := strings.Split(trimmedPath, ".")
	var current any = r.lastRawMap

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]any:
			current = v[part]
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("invalid array index %q", part)
			}
			current = v[idx]
		default:
			return nil, fmt.Errorf("cannot navigate path at %q", part)
		}
	}

	if current == nil {
		return nil, fmt.Errorf("path %q not found", path)
	}
	return current, nil
}
