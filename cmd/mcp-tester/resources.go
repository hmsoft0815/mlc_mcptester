package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var resourceCursor string

func init() {
	resourcesListCmd.Flags().StringVarP(&resourceCursor, "cursor", "C", "", "Pagination cursor for listing resources")
	resourcesCmd.AddCommand(resourcesListCmd)
	resourcesCmd.AddCommand(resourcesTemplatesListCmd)
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

		var params *mcp.ListResourcesParams
		if resourceCursor != "" {
			params = &mcp.ListResourcesParams{Cursor: resourceCursor}
		}

		res, err := session.ListResources(ctx, params)
		if err != nil {
			return err
		}

		for _, r := range res.Resources {
			fmt.Print(i18n.T(i18n.MsgResource, r.Name))
			fmt.Printf("  URI:      %s\n", r.URI)
			fmt.Printf("  MimeType: %s\n", r.MIMEType)
			fmt.Print(i18n.T(i18n.MsgDescription, r.Description))
			if len(r.Icons) > 0 {
				fmt.Println(i18n.T(i18n.MsgIcons))
				for _, icon := range r.Icons {
					fmt.Printf("    - %s\n", icon.Source)
				}
				if checkIcons || downloadIcons != "" {
					checkAndDownloadIcons(r.Icons, downloadIcons)
				}
			}
			fmt.Println("---")
		}

		if res.NextCursor != "" {
			fmt.Printf("Next Cursor: %s\n", res.NextCursor)
		}

		return nil
	},
}

var resourcesTemplatesListCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available resource templates on the MCP server",
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

		res, err := session.ListResourceTemplates(ctx, nil)
		if err != nil {
			return err
		}

		for _, t := range res.ResourceTemplates {
			fmt.Print(i18n.T(i18n.MsgTemplate, t.Name))
			fmt.Printf("  URI:  %s\n", t.URITemplate)
			fmt.Print(i18n.T(i18n.MsgDescription, t.Description))
			if len(t.Icons) > 0 {
				fmt.Println(i18n.T(i18n.MsgIcons))
				for _, icon := range t.Icons {
					fmt.Printf("    - %s\n", icon.Source)
				}
				if checkIcons || downloadIcons != "" {
					checkAndDownloadIcons(t.Icons, downloadIcons)
				}
			}
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
