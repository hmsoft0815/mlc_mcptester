package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/internal/conformance"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var callArgs string

func init() {
	callCmd.Flags().StringVarP(&callArgs, "args", "a", "{}", "Tool arguments (JSON)")
	rootCmd.AddCommand(callCmd)
}

// callCmd implements the 'call' command to invoke tools on an MCP server.
var callCmd = &cobra.Command{
	Use:   "call [tool_name]",
	Short: "Call a tool provided by the MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		toolName := args[0]
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

		// Get appropriate transport (stdio or sse).
		transport, err := getTransport(ctx, c, u)
		if err != nil {
			return err
		}

		// Set up the client.
		mcpClient := getClient(verbose)

		// Create a session with the server.
		session, err := mcpClient.Connect(ctx, transport, nil)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer session.Close()

		if verbose {
			_ = session.SetLoggingLevel(ctx, &mcp.SetLoggingLevelParams{Level: "debug"})
		}

		// Parse the JSON arguments provided via the --args flag.
		var toolArgs map[string]any
		if err := json.Unmarshal([]byte(callArgs), &toolArgs); err != nil {
			return fmt.Errorf("failed to parse arguments: %w", err)
		}

		// The result is checked against this the way a strict client checks
		// it; the Go SDK's client does not.
		outputSchema, err := outputSchemaOf(ctx, session, toolName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not list tools, the result is not checked against an output schema: %v\n", err)
		}

		if raw {
			fmt.Println("--- RAW MODE ---")
			meta := map[string]any{"progressToken": fmt.Sprintf("script-progress-%s", toolName)}
			result, err := client.CallToolRaw(ctx, session, toolName, toolArgs, meta)
			if err != nil {
				return fmt.Errorf("failed to call tool (raw): %w", err)
			}
			output, _ := json.MarshalIndent(result, "", "  ")
			fmt.Printf("%s\n", string(output))
			return checkResult(outputSchema, result)
		}

		// Execute the tool call request.
		params := &mcp.CallToolParams{
			Name:      toolName,
			Arguments: toolArgs,
		}
		if verbose {
			params.Meta = mcp.Meta{
				"progressToken": "call-progress-123",
			}
		}

		callResult, err := session.CallTool(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to call tool: %w", err)
		}

		// Print each content item from the result.
		for i, content := range callResult.Content {
			switch c := content.(type) {
			case *mcp.TextContent:
				fmt.Printf("Content %d (Text):\n%s\n", i, c.Text)
			case *mcp.ImageContent:
				fmt.Printf("Content %d (Image): %s data, size %d\n", i, c.MIMEType, len(c.Data))
			default:
				data, _ := json.MarshalIndent(c, "", "  ")
				fmt.Printf("Content %d (%T):\n%s\n", i, c, string(data))
			}
		}

		if callResult.StructuredContent != nil {
			data, _ := json.MarshalIndent(callResult.StructuredContent, "", "  ")
			fmt.Printf("StructuredContent:\n%s\n", string(data))
		}

		if callResult.IsError {
			fmt.Println("Result indicated an error.")
		}

		data, err := json.Marshal(callResult)
		if err != nil {
			return err
		}
		var asMap map[string]any
		if err := json.Unmarshal(data, &asMap); err != nil {
			return err
		}
		return checkResult(outputSchema, asMap)
	},
}

// outputSchemaOf returns the output schema the named tool declares, or nil.
func outputSchemaOf(ctx context.Context, session *mcp.ClientSession, name string) (any, error) {
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			return nil, err
		}
		if tool.Name == name {
			return tool.OutputSchema, nil
		}
	}
	return nil, nil
}

// checkResult turns a result a strict client would reject into a failed call.
func checkResult(outputSchema any, result map[string]any) error {
	if err := conformance.CheckToolResult(outputSchema, result); err != nil {
		return fmt.Errorf("the result violates the MCP specification: %w", err)
	}
	return nil
}
