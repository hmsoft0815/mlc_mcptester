package main

import (
	"context"
	"fmt"

	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tools provided by the MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		config, err := loadConfig("mcp-tester.yml")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
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
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer session.Close()

		toolsResult, err := session.ListTools(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}

		for _, tool := range toolsResult.Tools {
			fmt.Print(i18n.T(i18n.MsgTool, tool.Name))
			fmt.Print(i18n.T(i18n.MsgDescription, tool.Description))
			if len(tool.Icons) > 0 {
				fmt.Println(i18n.T(i18n.MsgIcons))
				for _, icon := range tool.Icons {
					fmt.Printf("  - %s (%s)\n", icon.Source, icon.MIMEType)
				}
				if checkIcons || downloadIcons != "" {
					checkAndDownloadIcons(tool.Icons, downloadIcons)
				}
			}
			fmt.Print(i18n.T(i18n.MsgInputSchema, tool.InputSchema))
			if tool.OutputSchema != nil {
				fmt.Print(i18n.T(i18n.MsgOutputSchema, tool.OutputSchema))
			}
			if tool.Annotations != nil {
				fmt.Print(i18n.T(i18n.MsgAnnotations, tool.Annotations))
			}
			fmt.Println("---")
		}

		return nil
	},
}
