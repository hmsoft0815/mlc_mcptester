package main

import (
	"context"
	"fmt"
	mcpclient "github.com/hmsoft0815/mlc_mcptester/internal/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(loggingCmd)
}

var loggingCmd = &cobra.Command{
	Use:   "logging [level]",
	Short: "Set the logging level on the MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		level := args[0]
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
		client := getClient(verbose)
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			return err
		}
		defer session.Close()

		if mcpclient.IsStateless(session) {
			cmd.SilenceUsage = true
			return fmt.Errorf("protocol %s has no logging/setLevel; the level is sent with each request: use 'call --log-level %s' or 'logging %s' in a test script",
				session.InitializeResult().ProtocolVersion, level, level)
		}
		fmt.Printf("Setting logging level to %s...\n", level)
		if err := session.SetLoggingLevel(ctx, &mcp.SetLoggingLevelParams{Level: mcp.LoggingLevel(level)}); err != nil {
			return fmt.Errorf("failed to set logging level: %w", err)
		}
		fmt.Println("Logging level set successfully!")
		return nil
	},
}
