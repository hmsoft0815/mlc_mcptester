// Package agentskill carries the Agent Skill that teaches coding agents how to
// use mcp-tester, and installs it where each harness looks for skills.
//
// The skill is embedded in the binary, so the installed instructions always
// match the installed tester. Skills belong to the user's harness, not to a
// project, so they go into the user-level skill directories.
package agentskill

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Name is the skill's directory name and frontmatter name.
const Name = "mcp-tester"

//go:embed SKILL.md
var source string

// Content returns the skill with the tester version filled in.
func Content(version string) []byte {
	return []byte(strings.ReplaceAll(source, "{{VERSION}}", version))
}

var versionLine = regexp.MustCompile(`(?m)^\s*mcp-tester-version:\s*"?([^"\n]*)"?\s*$`)

// InstalledVersion reads the tester version from an installed skill file, or
// "" if it has none.
func InstalledVersion(data []byte) string {
	if m := versionLine.FindSubmatch(data); m != nil {
		return string(m[1])
	}
	return ""
}

// Env holds the directories the harness locations derive from.
type Env struct {
	Home string
	// ConfigHome is $XDG_CONFIG_HOME, default ~/.config.
	ConfigHome string
	// CodexHome is $CODEX_HOME, default ~/.codex.
	CodexHome string
}

// EnvFromOS reads Env from the environment.
func EnvFromOS() (Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Env{}, err
	}
	e := Env{Home: home, ConfigHome: os.Getenv("XDG_CONFIG_HOME"), CodexHome: os.Getenv("CODEX_HOME")}
	if e.ConfigHome == "" {
		e.ConfigHome = filepath.Join(home, ".config")
	}
	if e.CodexHome == "" {
		e.CodexHome = filepath.Join(home, ".codex")
	}
	return e, nil
}

// Harness is a coding agent that loads skills from user-level directories.
type Harness struct {
	Name  string
	Label string
	// Marker is the directory whose existence shows the harness is installed.
	Marker func(Env) string
	// Primary is where the skill goes if no other target covers the harness.
	Primary func(Env) string
	// AlsoReads lists further skill directories the harness loads; a skill
	// written there for another harness reaches this one too.
	AlsoReads func(Env) []string
}

func claudeSkills(e Env) string { return filepath.Join(e.Home, ".claude", "skills") }
func agentsSkills(e Env) string { return filepath.Join(e.Home, ".agents", "skills") }
func noOthers(Env) []string     { return nil }

// Harnesses in planning order: those that read one directory only come first,
// so the others can share their targets.
var Harnesses = []Harness{
	{
		Name: "claude", Label: "Claude Code",
		Marker:    func(e Env) string { return filepath.Join(e.Home, ".claude") },
		Primary:   claudeSkills,
		AlsoReads: noOthers,
	},
	{
		// Codex reads ~/.agents/skills; $CODEX_HOME/skills is deprecated.
		Name: "codex", Label: "OpenAI Codex",
		Marker:    func(e Env) string { return e.CodexHome },
		Primary:   agentsSkills,
		AlsoReads: noOthers,
	},
	{
		Name: "gemini", Label: "Gemini CLI",
		Marker:    func(e Env) string { return filepath.Join(e.Home, ".gemini") },
		Primary:   func(e Env) string { return filepath.Join(e.Home, ".gemini", "skills") },
		AlsoReads: func(e Env) []string { return []string{agentsSkills(e)} },
	},
	{
		Name: "opencode", Label: "OpenCode",
		Marker:    func(e Env) string { return filepath.Join(e.ConfigHome, "opencode") },
		Primary:   func(e Env) string { return filepath.Join(e.ConfigHome, "opencode", "skills") },
		AlsoReads: func(e Env) []string { return []string{claudeSkills(e), agentsSkills(e)} },
	},
}

// HarnessNames lists the valid --harness values.
func HarnessNames() []string {
	names := make([]string, len(Harnesses))
	for i, h := range Harnesses {
		names[i] = h.Name
	}
	return names
}

// Detect returns the harnesses whose marker directory exists.
func Detect(e Env) []string {
	var found []string
	for _, h := range Harnesses {
		if info, err := os.Stat(h.Marker(e)); err == nil && info.IsDir() {
			found = append(found, h.Name)
		}
	}
	return found
}

// Target is one skill directory and the harnesses it serves.
type Target struct {
	// Dir is the skills root; the skill goes to Dir/Name/SKILL.md.
	Dir       string
	Harnesses []string
}

// Plan picks as few skill directories as reach all given harnesses, so no
// harness loads the same skill twice.
func Plan(e Env, names []string) ([]Target, error) {
	for _, n := range names {
		if !slices.Contains(HarnessNames(), n) {
			return nil, fmt.Errorf("unknown harness %q (known: %s)", n, strings.Join(HarnessNames(), ", "))
		}
	}
	var targets []Target
	index := map[string]int{}
	for _, h := range Harnesses {
		if !slices.Contains(names, h.Name) {
			continue
		}
		dir := h.Primary(e)
		for _, other := range h.AlsoReads(e) {
			if _, ok := index[other]; ok {
				dir = other
				break
			}
		}
		if i, ok := index[dir]; ok {
			targets[i].Harnesses = append(targets[i].Harnesses, h.Label)
			continue
		}
		index[dir] = len(targets)
		targets = append(targets, Target{Dir: dir, Harnesses: []string{h.Label}})
	}
	if len(targets) == 0 {
		return nil, errors.New("no harness selected")
	}
	return targets, nil
}

// Result reports one written skill file.
type Result struct {
	Path string
	// Previous is the tester version of the replaced skill; "" for a new
	// install or a file without version.
	Previous string
	Existed  bool
}

// Install writes the skill into skillsDir/Name/SKILL.md.
func Install(skillsDir, version string) (Result, error) {
	dir := filepath.Join(skillsDir, Name)
	r := Result{Path: filepath.Join(dir, "SKILL.md")}
	if old, err := os.ReadFile(r.Path); err == nil {
		r.Existed = true
		r.Previous = InstalledVersion(old)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return r, err
	}
	return r, os.WriteFile(r.Path, Content(version), 0o644)
}
