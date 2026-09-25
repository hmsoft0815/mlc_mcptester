package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/appcheck"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(appsCmd)
}

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "List and check the interactive UIs a server delivers (MCP Apps extension)",
	Long: `Lists the MCP Apps (io.modelcontextprotocol/ui) a server delivers: which
tools render with which ui:// resource, tools hidden from the model, and what
each app asks the host for — external domains (CSP), permissions (camera,
microphone, geolocation, clipboard), a dedicated origin. Checks the resources
(ui:// scheme, MIME type text/html;profile=mcp-app, HTML document). mcp-tester
announces itself as a UI-capable host, so servers expose their UI tools; it
does not render the apps.`,
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
		session, err := newClient(verbose, cliResponder, nil, appcheck.Declare).Connect(ctx, transport, nil)
		if err != nil {
			return err
		}
		defer session.Close()

		report := appcheck.Run(ctx, session)
		if format == "json" {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(out))
		} else {
			printApps(report)
		}
		if report.Failed() {
			cmd.SilenceUsage = true
			return fmt.Errorf("the MCP Apps of this server violate the extension")
		}
		return nil
	},
}

func printApps(report *appcheck.Report) {
	fmt.Printf("=== MCP Apps (%d) ===\n", len(report.Apps))
	for _, a := range report.Apps {
		fmt.Printf("\n%s  (%d bytes)\n  tools: %s\n", a.URI, a.Bytes, strings.Join(a.Tools, ", "))
		if len(a.AppOnly) > 0 {
			fmt.Printf("  hidden from the model (app only): %s\n", strings.Join(a.AppOnly, ", "))
		}
		if len(a.CSP) == 0 {
			fmt.Println("  external domains: none")
		}
		for key, domains := range a.CSP {
			fmt.Printf("  %s: %s\n", key, strings.Join(domains, ", "))
		}
		if len(a.Permissions) > 0 {
			fmt.Printf("  permissions: %s  <- review before enabling\n", strings.Join(a.Permissions, ", "))
		}
		if a.Domain != "" {
			fmt.Printf("  dedicated origin: %s\n", a.Domain)
		}
	}
	fmt.Println("\n=== Checks ===")
	for _, r := range report.Results {
		fmt.Printf("[%s] %s  %s\n", r.Status, r.Name, r.Detail)
	}
}
