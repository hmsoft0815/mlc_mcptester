package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func init() {
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

		res, err := session.ListPrompts(ctx, nil)
		if err != nil {
			return err
		}

		for _, p := range res.Prompts {
			fmt.Printf("Prompt: %s\n", p.Name)
			fmt.Printf("  Desc: %s\n", p.Description)
			if len(p.Arguments) > 0 {
				fmt.Println("  Arguments:")
				for _, arg := range p.Arguments {
					fmt.Printf("    - %s: %s (required: %v)\n", arg.Name, arg.Description, arg.Required)
				}
			}
			fmt.Println("---")
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

		output, _ := json.MarshalIndent(res, "", "  ")
		fmt.Printf("Prompt Result:\n%s\n", string(output))
		return nil
	},
}
