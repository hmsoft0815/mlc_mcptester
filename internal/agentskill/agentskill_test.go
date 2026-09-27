package agentskill

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func testEnv(t *testing.T) Env {
	home := t.TempDir()
	return Env{Home: home, ConfigHome: filepath.Join(home, ".config"), CodexHome: filepath.Join(home, ".codex")}
}

func TestContent(t *testing.T) {
	c := Content("1.6.0")
	if !strings.HasPrefix(string(c), "---\nname: mcp-tester\n") {
		t.Errorf("skill does not start with its frontmatter:\n%.80s", c)
	}
	if strings.Contains(string(c), "{{VERSION}}") {
		t.Error("version placeholder left in the skill")
	}
	if got := InstalledVersion(c); got != "1.6.0" {
		t.Errorf("InstalledVersion = %q, want 1.6.0", got)
	}
}

func TestPlan(t *testing.T) {
	e := testEnv(t)
	claude := filepath.Join(e.Home, ".claude", "skills")
	agents := filepath.Join(e.Home, ".agents", "skills")
	gemini := filepath.Join(e.Home, ".gemini", "skills")
	opencode := filepath.Join(e.ConfigHome, "opencode", "skills")

	cases := []struct {
		name  string
		names []string
		want  []Target
	}{
		{"claude alone", []string{"claude"}, []Target{{claude, []string{"Claude Code"}}}},
		{"opencode alone", []string{"opencode"}, []Target{{opencode, []string{"OpenCode"}}}},
		{"opencode shares claude's", []string{"opencode", "claude"}, []Target{{claude, []string{"Claude Code", "OpenCode"}}}},
		{"gemini shares codex's", []string{"gemini", "codex"}, []Target{{agents, []string{"OpenAI Codex", "Gemini CLI"}}}},
		{"claude and gemini", []string{"claude", "gemini"}, []Target{{claude, []string{"Claude Code"}}, {gemini, []string{"Gemini CLI"}}}},
		{"all four", HarnessNames(), []Target{
			{claude, []string{"Claude Code", "OpenCode"}},
			{agents, []string{"OpenAI Codex", "Gemini CLI"}},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Plan(e, c.names)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("Plan = %+v\nwant   %+v", got, c.want)
			}
		})
	}

	if _, err := Plan(e, []string{"vim"}); err == nil {
		t.Error("unknown harness accepted")
	}
}

func TestDetect(t *testing.T) {
	e := testEnv(t)
	for _, d := range []string{filepath.Join(e.Home, ".gemini"), filepath.Join(e.ConfigHome, "opencode")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got := Detect(e); !reflect.DeepEqual(got, []string{"gemini", "opencode"}) {
		t.Errorf("Detect = %v", got)
	}
}

func TestInstall(t *testing.T) {
	dir := t.TempDir()
	r, err := Install(dir, "1.5.0")
	if err != nil || r.Existed {
		t.Fatalf("first install: %+v, %v", r, err)
	}
	r, err = Install(dir, "1.6.0")
	if err != nil || !r.Existed || r.Previous != "1.5.0" {
		t.Fatalf("update: %+v, %v", r, err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, Name, "SKILL.md"))
	if InstalledVersion(data) != "1.6.0" {
		t.Error("update did not replace the skill")
	}
}
