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

	if r.lastRawMap == nil {
		return nil, fmt.Errorf("no previous response available")
	}

	// 1. Try resolving the path as-is (works for nested structuredContent or top-level $. fields if literally present)
	if val, err := r.resolvePath(path); err == nil {
		return val, nil
	}

	// 2. Try stripping "virtual" prefixes
	trimmedPath := strings.TrimPrefix(path, "structuredContent.")
	trimmedPath = strings.TrimPrefix(trimmedPath, "$.")

	if trimmedPath != path {
		if val, err := r.resolvePath(trimmedPath); err == nil {
			return val, nil
		}
	}

	return nil, fmt.Errorf("path %q not found", path)
}

func (r *Runner) resolvePath(path string) (any, error) {
	parts := strings.Split(path, ".")
	var current any = r.lastRawMap

	for _, part := range parts {
		if part == "" || part == "$" {
			continue
		}

		switch v := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("key %q not found", part)
			}
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
		return nil, fmt.Errorf("value at path %q is nil", path)
	}
	return current, nil
}
