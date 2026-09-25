package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/skillcheck"
	"github.com/spf13/cobra"
)

var skillsVerify bool

func init() {
	skillsCmd.Flags().BoolVar(&skillsVerify, "verify", false, "Read every skill file and compare size, digest and frontmatter with the manifest")
	rootCmd.AddCommand(skillsCmd)
}

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "List and check the skills a server publishes (Skills extension)",
	Long: `Lists the Agent Skills a server publishes via the Skills extension
(io.modelcontextprotocol/skills) — name, description, license, allowed tools,
files — and checks them: manifests, frontmatter and naming rules, skills/get,
error codes, directory reads. With --verify every file is read and compared
with its digest. Useful before enabling a third-party server: skills are
instructions that reach the model.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		config, _ := loadConfig("mcp-tester.yml")
		c, u, err := resolveSettings(config, profile, command, url)
		if err != nil {
			return err
		}
		transport, err := getTransport(ctx, c, u)
		if err != nil {
			return err
		}
		session, err := getClient(verbose).Connect(ctx, transport, nil)
		if err != nil {
			return err
		}
		defer session.Close()

		report := (&skillcheck.Checker{Session: session, Verify: skillsVerify}).Run(ctx)
		if format == "json" {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(out))
		} else {
			printSkills(report)
		}
		if report.Failed() {
			cmd.SilenceUsage = true
			return fmt.Errorf("the published skills violate the Skills extension")
		}
		return nil
	},
}

func printSkills(report *skillcheck.Report) {
	if !report.Declared {
		fmt.Println("The server does not publish skills (no io.modelcontextprotocol/skills extension).")
		return
	}
	fmt.Printf("=== Skills (%d) ===\n", len(report.Skills))
	for _, s := range report.Skills {
		files := fmt.Sprintf("%d file(s), %d bytes", s.Files, s.Bytes)
		if s.Dynamic {
			files = "dynamic content, no digests"
		}
		fmt.Printf("\n%s  (%s)\n  %s\n  %s\n", s.Name, s.URI, s.Description, files)
		if s.License != "" {
			fmt.Printf("  license: %s\n", s.License)
		}
		if s.AllowedTools != "" {
			fmt.Printf("  allowed-tools: %s  <- pre-approves tools, review before enabling\n", s.AllowedTools)
		}
	}
	fmt.Println("\n=== Checks ===")
	for _, r := range report.Results {
		fmt.Printf("[%s] %s  %s\n", r.Status, r.Name, strings.TrimSpace(r.Detail))
	}
}
