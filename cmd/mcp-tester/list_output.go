package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// printJSON writes v as indented JSON to stdout (--format json).
func printJSON(v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

// checkIconsOf runs --check-icons / --download-icons for one entry.
func checkIconsOf(icons []mcp.Icon) {
	if len(icons) > 0 && (checkIcons || downloadIcons != "") {
		checkAndDownloadIcons(icons, downloadIcons)
	}
}

// formatAnnotations lists the hints that are set, with their values; the
// optional ones (pointers in the SDK) only when the server sent them.
func formatAnnotations(a *mcp.ToolAnnotations) string {
	var parts []string
	if a.Title != "" {
		parts = append(parts, fmt.Sprintf("title=%q", a.Title))
	}
	parts = append(parts, fmt.Sprintf("readOnlyHint=%v", a.ReadOnlyHint))
	if a.DestructiveHint != nil {
		parts = append(parts, fmt.Sprintf("destructiveHint=%v", *a.DestructiveHint))
	}
	parts = append(parts, fmt.Sprintf("idempotentHint=%v", a.IdempotentHint))
	if a.OpenWorldHint != nil {
		parts = append(parts, fmt.Sprintf("openWorldHint=%v", *a.OpenWorldHint))
	}
	return strings.Join(parts, ", ")
}
