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

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run a test script against an MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if scriptPath == "" {
			return fmt.Errorf("script path is required")
		}
		config, err := loadConfig("mcp-tester.yml")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		c, u, err := resolveSettings(config, profile, command, url)
		if err != nil {
			return err
		}
		script, err := os.ReadFile(scriptPath)
		if err != nil {
			return fmt.Errorf("failed to read script: %w", err)
		}
		ctx := context.Background()
		transport, err := getTransport(ctx, c, u)
		if err != nil {
			return err
		}
		client := getClient(verbose)
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer session.Close()
		runner := scripting.NewRunner(session, raw)
		_, err = runner.Run(ctx, string(script), format)
		return err
	},
}
