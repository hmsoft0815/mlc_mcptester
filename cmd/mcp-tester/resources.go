package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func init() {
	resourcesCmd.AddCommand(resourcesListCmd)
	resourcesCmd.AddCommand(resourcesReadCmd)
	rootCmd.AddCommand(resourcesCmd)
}

var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Manage and interact with MCP resources",
}

var resourcesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available resources on the MCP server",
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
		client := getClient(verbose)
		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			return err
		}
		defer session.Close()

		res, err := session.ListResources(ctx, nil)
		if err != nil {
			return err
		}

		for _, r := range res.Resources {
			fmt.Printf("Resource: %s\n", r.Name)
			fmt.Printf("  URI:  %s\n", r.URI)
			fmt.Printf("  Desc: %s\n", r.Description)
			fmt.Println("---")
		}
		return nil
	},
}

var resourcesReadCmd = &cobra.Command{
	Use:   "read [uri]",
	Short: "Read a resource from the MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		uri := args[0]
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

		res, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
		if err != nil {
			return err
		}

		output, _ := json.MarshalIndent(res, "", "  ")
		fmt.Printf("Resource Content:\n%s\n", string(output))
		return nil
	},
}
