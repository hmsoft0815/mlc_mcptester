package version

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		bi       *debug.BuildInfo
		expected string
	}{
		{
			name:     "stamped version from ldflags",
			current:  "1.2.0",
			bi:       nil,
			expected: "1.2.0",
		},
		{
			name:    "unstamped with bi module version with v prefix",
			current: "dev",
			bi: &debug.BuildInfo{
				Main: debug.Module{
					Version: "v1.2.0",
				},
			},
			expected: "1.2.0",
		},
		{
			name:    "unstamped with bi module version without v prefix",
			current: "dev",
			bi: &debug.BuildInfo{
				Main: debug.Module{
					Version: "1.2.0",
				},
			},
			expected: "1.2.0",
		},
		{
			name:    "unstamped with (devel)",
			current: "dev",
			bi: &debug.BuildInfo{
				Main: debug.Module{
					Version: "(devel)",
				},
			},
			expected: "dev",
		},
		{
			name:     "unstamped with nil bi",
			current:  "dev",
			bi:       nil,
			expected: "dev",
		},
		{
			name:     "empty current with nil bi",
			current:  "",
			bi:       nil,
			expected: "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveVersion(tt.current, tt.bi)
			if got != tt.expected {
				t.Errorf("resolveVersion(%q, bi) = %q, want %q", tt.current, got, tt.expected)
			}
		})
	}
}
