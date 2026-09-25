package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hmsoft0815/mlc_mcptester/internal/authcheck"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(authCheckCmd)
}

var authCheckCmd = &cobra.Command{
	Use:   "auth-check",
	Short: "Show and check how an HTTP server is protected (OAuth, auth extensions)",
	Long: `Discovers how a Streamable HTTP server is protected — 401 challenge,
Protected Resource Metadata, authorization server metadata — lists the flows it
offers (authorization code + PKCE, client credentials, enterprise-managed via
ID-JAG) with the mcp-tester flags for each, and checks the metadata against
spec 2026-07-28 and the auth extensions. MUST violations fail (exit 1).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, _ := loadConfig("mcp-tester.yml")
		_, u, err := resolveSettings(config, profile, command, url)
		if err != nil {
			return err
		}
		if u == "" {
			return fmt.Errorf("auth-check needs --url (or a profile with a url)")
		}
		// No credentials here: the check looks at what an unauthenticated client sees
		report := authcheck.Check(context.Background(), u, nil)

		if format == "json" {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("=== Authorization: %s ===\n", u)
			if report.Protected && report.Discovery != nil {
				fmt.Printf("resource:             %s\n", report.Discovery.Resource)
				fmt.Printf("authorization server: %s\n", report.Discovery.AuthServer())
			}
			if len(report.Flows) > 0 {
				fmt.Println("\nFlows:")
				for _, f := range report.Flows {
					mark := "no "
					if f.Supported {
						mark = "yes"
					}
					fmt.Printf("  [%s] %-40s %s", mark, f.Name, f.Flag)
					if f.Detail != "" {
						fmt.Printf("  (%s)", f.Detail)
					}
					fmt.Println()
				}
			}
			fmt.Println("\nChecks:")
			for _, r := range report.Results {
				fmt.Printf("[%s] %-36s %s\n", r.Status, r.Name, r.Detail)
			}
		}
		if report.Failed() {
			cmd.SilenceUsage = true
			return fmt.Errorf("the authorization setup violates the specification")
		}
		return nil
	},
}
