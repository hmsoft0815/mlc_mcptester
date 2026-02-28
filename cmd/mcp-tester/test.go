package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hmsoft0815/mlc_mcptester/internal/scripting"
	"github.com/spf13/cobra"
)

var scriptPath string

func init() {
	testCmd.Flags().StringVarP(&scriptPath, "script", "s", "", "Path to the test script")
	rootCmd.AddCommand(testCmd)
}

// testCmd implements the 'test' command to execute MCP test scripts.
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run a test script against an MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure a script path is provided.
		if scriptPath == "" {
			return fmt.Errorf("script path is required")
		}

		// Load configuration file
		config, err := loadConfig("mcp-tester.yml")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Resolve settings from profile or flags
		c, u, err := resolveSettings(config, profile, command, url)
		if err != nil {
			return err
		}

		// Read the content of the test script file.
		script, err := os.ReadFile(scriptPath)
		if err != nil {
			return fmt.Errorf("failed to read script: %w", err)
		}

		ctx := context.Background()
		// Initialize the requested transport (stdio or sse).
		transport, err := getTransport(ctx, c, u)
		if err != nil {
			return err
		}

		// Set up the MCP client.
		client := getClient(verbose)

		// Connect to the MCP server.
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer session.Close()

		// Run the script using the scripting runner.
		runner := scripting.NewRunner(session, raw)
		return runner.Run(ctx, string(script))
	},
}
