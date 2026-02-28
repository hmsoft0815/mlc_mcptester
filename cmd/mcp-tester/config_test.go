package main

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test missing file
	config, err := loadConfig("non-existent.yml")
	if err != nil {
		t.Errorf("loadConfig(non-existent) error = %v", err)
	}
	if len(config.Profiles) != 0 {
		t.Error("expected empty profiles for non-existent file")
	}

	// Test valid file
	content := `
profiles:
  test:
    command: "ls"
    url: "http://localhost"
`
	tmpfile, _ := os.CreateTemp("", "mcp-test-*.yml")
	defer os.Remove(tmpfile.Name())
	tmpfile.WriteString(content)
	tmpfile.Close()

	config, err = loadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("loadConfig error = %v", err)
	}
	if p, ok := config.Profiles["test"]; !ok || p.Command != "ls" || p.URL != "http://localhost" {
		t.Errorf("invalid config loaded: %+v", config)
	}
}

func TestResolveSettings(t *testing.T) {
	config := &Config{
		Profiles: map[string]Profile{
			"p1": {Command: "cmd1", URL: "url1"},
		},
	}

	tests := []struct {
		profile string
		cmd     string
		url     string
		wantCmd string
		wantURL string
		wantErr bool
	}{
		{"p1", "", "", "cmd1", "url1", false},
		{"", "manual-cmd", "manual-url", "manual-cmd", "manual-url", false},
		{"unknown", "", "", "", "", true},
	}

	for _, tt := range tests {
		c, u, err := resolveSettings(config, tt.profile, tt.cmd, tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("resolveSettings(%q) error = %v; wantErr %v", tt.profile, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (c != tt.wantCmd || u != tt.wantURL) {
			t.Errorf("resolveSettings(%q) = (%q, %q); want (%q, %q)", tt.profile, c, u, tt.wantCmd, tt.wantURL)
		}
	}
}
