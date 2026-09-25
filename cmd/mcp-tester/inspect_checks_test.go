package main

import (
	"strings"
	"testing"
)

func TestRevisionsBehind(t *testing.T) {
	tests := map[string]int{
		"2026-07-28": 0,
		"2025-11-25": 1,
		"2024-11-05": 4,
		"1999-01-01": -1,
	}
	for version, want := range tests {
		if got := revisionsBehind(version); got != want {
			t.Errorf("revisionsBehind(%q) = %d; want %d", version, got, want)
		}
	}
	if protocolRevisions[0] != latestProtocolRevision {
		t.Errorf("protocolRevisions[0] = %q; want latest %q", protocolRevisions[0], latestProtocolRevision)
	}
}

func TestCheckToolName(t *testing.T) {
	valid := []string{"getUser", "DATA_EXPORT_v2", "admin.tools.list", "a-b", strings.Repeat("x", 128)}
	for _, name := range valid {
		if msg := checkToolName(name); msg != "" {
			t.Errorf("checkToolName(%q) = %q; want valid", name, msg)
		}
	}
	invalid := []string{"", "get user", "a,b", "tool/name", "tööl", strings.Repeat("x", 129)}
	for _, name := range invalid {
		if msg := checkToolName(name); msg == "" {
			t.Errorf("checkToolName(%q) accepted an invalid name", name)
		}
	}
}

func TestInputSchemaIsObject(t *testing.T) {
	tests := []struct {
		schema any
		want   bool
	}{
		{map[string]any{"type": "object"}, true},
		{map[string]any{"type": []any{"object", "null"}}, true},
		{map[string]any{"type": "string"}, false},
		{map[string]any{}, false},
		{nil, false},
	}
	for _, tt := range tests {
		if got := inputSchemaIsObject(tt.schema); got != tt.want {
			t.Errorf("inputSchemaIsObject(%v) = %v; want %v", tt.schema, got, tt.want)
		}
	}
}

func TestCheckIconSource(t *testing.T) {
	allowed := []string{"https://example.com/i.png", "data:image/png;base64,AAAA", "HTTPS://EXAMPLE.COM/i.svg"}
	for _, src := range allowed {
		if msg := checkIconSource(src); msg != "" {
			t.Errorf("checkIconSource(%q) = %q; want allowed", src, msg)
		}
	}
	rejected := []string{"http://example.com/i.png", "javascript:alert(1)", "file:///etc/passwd", "ftp://x/i.png", "ws://x", "icons/i.png"}
	for _, src := range rejected {
		if msg := checkIconSource(src); msg == "" {
			t.Errorf("checkIconSource(%q) allowed a forbidden URI", src)
		}
	}
}

func TestCheckCacheScope(t *testing.T) {
	if msg := checkCacheScope(300000, "public"); msg != "" {
		t.Errorf("valid hints rejected: %s", msg)
	}
	if msg := checkCacheScope(0, "private"); msg != "" {
		t.Errorf("valid hints rejected: %s", msg)
	}
	for _, tt := range []struct {
		ttl   int
		scope string
	}{{-1, "public"}, {0, ""}, {0, "shared"}} {
		if msg := checkCacheScope(tt.ttl, tt.scope); msg == "" {
			t.Errorf("checkCacheScope(%d, %q) accepted invalid hints", tt.ttl, tt.scope)
		}
	}
}
