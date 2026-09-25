package main

import (
	"context"
	"encoding/json"
	"fmt"

	mcpclient "github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/internal/httpcheck"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(httpCheckCmd)
}

var httpCheckCmd = &cobra.Command{
	Use:   "http-check",
	Short: "Check a Streamable HTTP endpoint against the transport rules of spec 2026-07-28",
	Long: `Sends hand-built requests to a Streamable HTTP MCP endpoint and checks the
answers: request metadata headers (MCP-Protocol-Version, Mcp-Method, Mcp-Name,
Base64 values), error codes and HTTP status, Origin validation, and that GET,
DELETE and sessions are gone. MUST violations fail (exit 1), SHOULD violations warn.
Authentication: --bearer / --header (OAuth tokens are not reused here).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		config, _ := loadConfig("mcp-tester.yml")
		_, u, err := resolveSettings(config, profile, command, url)
		if err != nil {
			return err
		}
		if u == "" {
			return fmt.Errorf("http-check needs --url (or a profile with a url)")
		}
		httpClient, err := httpClientWithAuth()
		if err != nil {
			return err
		}

		checker := &httpcheck.Checker{Endpoint: u, Client: httpClient}
		// The tools for the tools/call header checks, via a regular session
		if transport, err := getTransport(ctx, "", u); err == nil {
			if session, err := getClient(false).Connect(ctx, transport, nil); err == nil {
				if tools, err := mcpclient.ListAllTools(ctx, session); err == nil {
					for _, t := range tools {
						checker.Tools = append(checker.Tools, httpcheck.Tool{Name: t.Name, InputSchema: t.InputSchema})
					}
				}
				session.Close()
			}
		}

		report := checker.Run(ctx)
		if format == "json" {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Printf("=== Streamable HTTP checks (spec %s): %s ===\n", httpcheck.Revision, u)
			for _, r := range report.Results {
				fmt.Printf("[%s] %-46s %s\n", r.Status, r.Name, r.Detail)
			}
		}
		if report.Failed() {
			cmd.SilenceUsage = true
			return fmt.Errorf("the endpoint violates the Streamable HTTP rules of %s", httpcheck.Revision)
		}
		return nil
	},
}
