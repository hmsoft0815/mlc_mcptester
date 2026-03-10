package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var promptCursor string

func init() {
	promptsListCmd.Flags().StringVarP(&promptCursor, "cursor", "C", "", "Pagination cursor for listing prompts")
	promptsCmd.AddCommand(promptsListCmd)
	promptsCmd.AddCommand(promptsGetCmd)
	rootCmd.AddCommand(promptsCmd)
}

var promptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "Manage and interact with MCP prompts",
}

var promptsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available prompts on the MCP server",
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

		var params *mcp.ListPromptsParams
		if promptCursor != "" {
			params = &mcp.ListPromptsParams{Cursor: promptCursor}
		}

		res, err := session.ListPrompts(ctx, params)
		if err != nil {
			return err
		}

		for _, p := range res.Prompts {
			fmt.Print(i18n.T(i18n.MsgPrompt, p.Name))
			fmt.Print(i18n.T(i18n.MsgDescription, p.Description))
			if len(p.Icons) > 0 {
				fmt.Println(i18n.T(i18n.MsgIcons))
				for _, icon := range p.Icons {
					fmt.Printf("    - %s\n", icon.Source)
				}
				if checkIcons || downloadIcons != "" {
					checkAndDownloadIcons(p.Icons, downloadIcons)
				}
			}
			if len(p.Arguments) > 0 {
				fmt.Println("  Arguments:")
				for _, arg := range p.Arguments {
					fmt.Printf("    - %s: %s (required: %v)\n", arg.Name, arg.Description, arg.Required)
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

var promptArgs string

func init() {
	promptsGetCmd.Flags().StringVarP(&promptArgs, "args", "a", "{}", "Prompt arguments (JSON)")
}

var promptsGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Get a prompt from the MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
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

		var pArgs map[string]string
		if err := json.Unmarshal([]byte(promptArgs), &pArgs); err != nil {
			return fmt.Errorf("failed to parse arguments: %w", err)
		}

		res, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
			Name:      name,
			Arguments: pArgs,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Prompt Description: %s\n", res.Description)
		fmt.Println("Messages:")
		for i, msg := range res.Messages {
			fmt.Printf("  [%d] Role: %s\n", i, msg.Role)
			switch c := msg.Content.(type) {
			case *mcp.TextContent:
				fmt.Printf("      Text: %s\n", c.Text)
			case *mcp.ImageContent:
				fmt.Printf("      Image: [%s] (%d bytes)\n", c.MIMEType, len(c.Data))
			case *mcp.EmbeddedResource:
				if c.Resource != nil {
					fmt.Printf("      Resource: %s (%s)\n", c.Resource.URI, c.Resource.MIMEType)
				}
			case *mcp.ResourceLink:
				fmt.Printf("      Resource Link: %s (%s)\n", c.URI, c.MIMEType)
			default:
				output, _ := json.Marshal(c)
				fmt.Printf("      Other Content: %s\n", string(output))
			}
		}

		return nil
	},
}
