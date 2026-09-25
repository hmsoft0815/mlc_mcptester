package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Published MCP protocol revisions, newest first. Update together with
// docs/SPEC_COVERAGE*.md when a new revision is released.
var protocolRevisions = []string{
	"2026-07-28",
	"2025-11-25",
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}

const latestProtocolRevision = "2026-07-28"

// revisionsBehind returns how many published revisions lie between version and
// the latest one, or -1 for a version this tester does not know.
func revisionsBehind(version string) int {
	return slices.Index(protocolRevisions, version)
}

// toolNamePattern is the character set the spec recommends for tool names
// (2025-11-25 and later): ASCII letters, digits, underscore, hyphen and dot.
var toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// checkToolName returns why name violates the spec's naming rules, or "".
func checkToolName(name string) string {
	switch {
	case len(name) == 0:
		return "empty"
	case len(name) > 128:
		return fmt.Sprintf("%d characters, at most 128 allowed", len(name))
	case !toolNamePattern.MatchString(name):
		return "only A-Z, a-z, 0-9, _, - and . are allowed"
	}
	return ""
}

// inputSchemaIsObject reports whether a tool's inputSchema declares type
// "object", as every spec example and strict clients expect.
func inputSchemaIsObject(schema any) bool {
	m, ok := schema.(map[string]any)
	if !ok {
		return false
	}
	switch t := m["type"].(type) {
	case string:
		return t == "object"
	case []any:
		return slices.Contains(t, any("object"))
	}
	return false
}

// checkIconSource returns why an icon URI is not allowed, or "". Clients must
// accept only https and data: URIs and reject everything else.
func checkIconSource(src string) string {
	lower := strings.ToLower(strings.TrimSpace(src))
	if strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "data:") {
		return ""
	}
	if i := strings.Index(lower, ":"); i > 0 {
		return fmt.Sprintf("scheme %q is not allowed, only https and data:", lower[:i])
	}
	return "relative URI, only https and data: are allowed"
}

// checkCacheScope returns why a list result's cache hints are not valid, or "".
func checkCacheScope(ttlMs int, scope string) string {
	if ttlMs < 0 {
		return fmt.Sprintf("ttlMs %d is negative", ttlMs)
	}
	switch scope {
	case "public", "private":
		return ""
	case "":
		return "cacheScope is missing"
	}
	return fmt.Sprintf("cacheScope %q is neither \"public\" nor \"private\"", scope)
}
