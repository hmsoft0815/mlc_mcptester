package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/agentskill"
	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/spf13/cobra"
)

var (
	agentSkillHarnesses []string
	agentSkillDir       string
	agentSkillDryRun    bool
)

func init() {
	agentSkillInstallCmd.Flags().StringSliceVar(&agentSkillHarnesses, "harness", nil,
		"Harnesses to install for: "+strings.Join(agentskill.HarnessNames(), ", ")+", all (default: the ones found)")
	agentSkillInstallCmd.Flags().StringVar(&agentSkillDir, "dir", "", "Install into this skills directory instead (the skill goes to <dir>/mcp-tester/SKILL.md)")
	agentSkillInstallCmd.Flags().BoolVar(&agentSkillDryRun, "dry-run", false, "Only show where the skill would go")
	agentSkillCmd.AddCommand(agentSkillInstallCmd, agentSkillPrintCmd)
	rootCmd.AddCommand(agentSkillCmd)
}

var agentSkillCmd = &cobra.Command{
	Use:   "agent-skill",
	Short: "Install the Agent Skill that teaches coding agents to use mcp-tester",
	Long: `mcp-tester carries an Agent Skill (SKILL.md) that tells coding agents such
as Claude Code, Gemini CLI, OpenCode and Codex which command to use when, how
to write .mcp test scripts and how to run them in CI. The skill matches this
binary's version; install it again after an upgrade.

Not to be confused with "mcp-tester skills", which checks the skills an MCP
server publishes.`,
}

var agentSkillPrintCmd = &cobra.Command{
	Use:   "print",
	Short: "Print the skill to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := os.Stdout.Write(agentskill.Content(version.Version))
		return err
	},
}

var agentSkillInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install or update the skill in the user-level skill directories of your harnesses",
	Long: `Installs the skill where each harness loads user-level skills:

  Claude Code   ~/.claude/skills/mcp-tester/
  OpenAI Codex  ~/.agents/skills/mcp-tester/
  Gemini CLI    ~/.gemini/skills/mcp-tester/   (also reads ~/.agents/skills)
  OpenCode      ~/.config/opencode/skills/mcp-tester/
                (also reads ~/.claude/skills and ~/.agents/skills)

Without --harness it installs for every harness whose configuration directory
exists. Directories a harness shares are written once, so no harness loads
the skill twice. An existing skill is replaced.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		if agentSkillDir != "" {
			return installSkillTo([]agentskill.Target{{Dir: agentSkillDir}})
		}
		env, err := agentskill.EnvFromOS()
		if err != nil {
			return err
		}
		names := agentSkillHarnesses
		if len(names) == 1 && names[0] == "all" {
			names = agentskill.HarnessNames()
		}
		if len(names) == 0 {
			names = agentskill.Detect(env)
			if len(names) == 0 {
				return fmt.Errorf("no harness found (looked for %s); choose one with --harness or a directory with --dir",
					strings.Join(agentskill.HarnessNames(), ", "))
			}
		}
		targets, err := agentskill.Plan(env, names)
		if err != nil {
			return err
		}
		return installSkillTo(targets)
	},
}

func installSkillTo(targets []agentskill.Target) error {
	for _, t := range targets {
		serves := ""
		if len(t.Harnesses) > 0 {
			serves = " (" + strings.Join(t.Harnesses, ", ") + ")"
		}
		if agentSkillDryRun {
			fmt.Printf("would install %s%s\n", filepath.Join(t.Dir, agentskill.Name, "SKILL.md"), serves)
			continue
		}
		r, err := agentskill.Install(t.Dir, version.Version)
		if err != nil {
			return fmt.Errorf("installing into %s: %w", t.Dir, err)
		}
		switch {
		case !r.Existed:
			fmt.Printf("installed %s%s\n", r.Path, serves)
		case r.Previous != "" && r.Previous != version.Version:
			fmt.Printf("updated %s %s -> %s%s\n", r.Path, r.Previous, version.Version, serves)
		default:
			fmt.Printf("replaced %s%s\n", r.Path, serves)
		}
	}
	return nil
}
