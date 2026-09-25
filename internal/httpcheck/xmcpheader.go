package httpcheck

import (
	"fmt"
	"regexp"
	"strings"
)

// ParamHeader is one valid x-mcp-header annotation: the tool parameter at Path
// is mirrored into the Mcp-Param-{Header} HTTP header.
type ParamHeader struct {
	Header string   // the Mcp-Param-{Header} name part
	Path   []string // property names from the root
	Type   string
}

// tchar is the RFC 9110 token character set.
var tcharPattern = regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$")

// XMCPHeaders validates the x-mcp-header annotations of an inputSchema
// (spec 2026-07-28): a non-empty token, unique ignoring case, only on string,
// integer or boolean properties, and only reachable through a chain of
// "properties" keys. It returns the valid annotations and the problems.
func XMCPHeaders(schema any) ([]ParamHeader, []string) {
	var found []ParamHeader
	var problems []string
	var walk func(node map[string]any, path []string)
	walk = func(node map[string]any, path []string) {
		props, _ := node["properties"].(map[string]any)
		for name, raw := range props {
			prop, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			p := append(append([]string{}, path...), name)
			if v, ok := prop["x-mcp-header"]; ok {
				header, _ := v.(string)
				typ, _ := prop["type"].(string)
				found = append(found, ParamHeader{Header: header, Path: p, Type: typ})
			}
			walk(prop, p)
		}
	}
	root, _ := schema.(map[string]any)
	if root == nil {
		return nil, nil
	}
	walk(root, nil)

	if total := countKey(root, "x-mcp-header"); total > len(found) {
		problems = append(problems, fmt.Sprintf("%d annotation(s) outside a plain \"properties\" chain (items, oneOf, $ref, ...)", total-len(found)))
	}

	var valid []ParamHeader
	seen := map[string]bool{}
	for _, h := range found {
		where := strings.Join(h.Path, ".")
		switch {
		case !tcharPattern.MatchString(h.Header):
			problems = append(problems, fmt.Sprintf("%s: %q is not an HTTP token", where, h.Header))
		case seen[strings.ToLower(h.Header)]:
			problems = append(problems, fmt.Sprintf("%s: %q is not unique (case-insensitive)", where, h.Header))
		case h.Type != "string" && h.Type != "integer" && h.Type != "boolean":
			problems = append(problems, fmt.Sprintf("%s: type %q, only string, integer or boolean allowed", where, h.Type))
		default:
			valid = append(valid, h)
		}
		seen[strings.ToLower(h.Header)] = true
	}
	return valid, problems
}

// countKey counts occurrences of key anywhere in a decoded JSON value.
func countKey(v any, key string) int {
	n := 0
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if k == key {
				n++
			}
			n += countKey(child, key)
		}
	case []any:
		for _, child := range t {
			n += countKey(child, key)
		}
	}
	return n
}
