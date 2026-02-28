package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

// listCmd implements the 'list' command to discover tools on an MCP server.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tools provided by the MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

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

		// Determine the transport from resolved settings.
		transport, err := getTransport(ctx, c, u)
		if err != nil {
			return err
		}

		// Initialize the MCP client.
		client := getClient(verbose)

		// Connect and establish an MCP session.
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer session.Close()

		// Retrieve the list of available tools from the server.
		toolsResult, err := session.ListTools(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}

		// Display tool details including their JSON schemas.
		for _, tool := range toolsResult.Tools {
			fmt.Printf("Tool: %s\n", tool.Name)
			fmt.Printf("Description: %s\n", tool.Description)
			fmt.Printf("Schema: %+v\n", tool.InputSchema)
			fmt.Println("---")
		}

		return nil
	},
}
