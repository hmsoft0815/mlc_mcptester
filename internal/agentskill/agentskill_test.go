package agentskill

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hmsoft0815/mlc_mcptester/pkg/mcpskills"
	"github.com/hmsoft0815/mlc_mcptester/skills"
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
	if got := InstalledVersion(c); got != "1.6.0" {
		t.Errorf("InstalledVersion = %q, want 1.6.0", got)
	}
	if n := len(versionLine.FindAll(c, -1)); n != 1 {
		t.Errorf("%d version lines, want exactly one", n)
	}
}

// The skill at skills/mcp-tester/SKILL.md follows the Agent Skills format,
// as mcp-tester itself checks it for servers that publish skills.
func TestSkillFormat(t *testing.T) {
	fm, err := mcpskills.ParseFrontmatter([]byte(skills.MCPTester))
	if err != nil {
		t.Fatal(err)
	}
	if problems := mcpskills.ValidateFrontmatter(fm); len(problems) > 0 {
		t.Errorf("frontmatter: %v", problems)
	}
	if fm["name"] != Name {
		t.Errorf("name %v, want the directory name %q", fm["name"], Name)
	}
	for _, field := range []string{"license", "compatibility"} {
		if _, ok := fm[field]; !ok {
			t.Errorf("frontmatter lacks %s", field)
		}
	}
	if lines := strings.Count(skills.MCPTester, "\n"); lines > 500 {
		t.Errorf("%d lines; keep a skill under 500 and move details to references", lines)
	}
}

// release-prep keeps the skill's version in step with VERSION.
func TestSkillVersionMatchesRelease(t *testing.T) {
	data, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Skip("no VERSION file")
	}
	if got, want := InstalledVersion([]byte(skills.MCPTester)), strings.TrimSpace(string(data)); got != want {
		t.Errorf("skills/mcp-tester/SKILL.md names %q, VERSION is %q: run task release-prep", got, want)
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
