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

var (
	callArgs     string
	callLogLevel string
	callAsTask   bool
)

func init() {
	callCmd.Flags().StringVarP(&callArgs, "args", "a", "{}", "Tool arguments (JSON)")
	callCmd.Flags().StringVar(&callLogLevel, "log-level", "", "Server log level for this call (debug, info, notice, warning, error, ...)")
	callCmd.Flags().BoolVar(&callAsTask, "task", false, "Offer to run the call as a task (Tasks extension) and poll it to the end")
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

		logLevel := callLogLevel
		if logLevel == "" && verbose {
			logLevel = "debug"
		}
		// Up to 2025-11-25 the level is session state; since 2026-07-28 it
		// travels with the request (below)
		if logLevel != "" && !client.IsStateless(session) {
			if err := session.SetLoggingLevel(ctx, &mcp.SetLoggingLevelParams{Level: mcp.LoggingLevel(logLevel)}); err != nil && callLogLevel != "" {
				return fmt.Errorf("failed to set logging level: %w", err)
			}
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
		if logLevel != "" && client.IsStateless(session) {
			params.Meta = client.WithLogLevel(params.Meta, logLevel)
		}

		var callResult *mcp.CallToolResult
		if callAsTask {
			callResult, err = callTask(ctx, session, toolName, toolArgs)
		} else {
			callResult, err = session.CallTool(ctx, params)
		}
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

// callTask runs the call as a task: it declares the Tasks extension, polls
// the task and returns its final result. The server may also answer at once.
func callTask(ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) (*mcp.CallToolResult, error) {
	var roots []*mcp.Root
	for _, uri := range rootURIs {
		roots = append(roots, &mcp.Root{URI: uri})
	}
	tc := &client.TaskClient{Session: session, Responder: cliResponder, Roots: roots, Out: os.Stderr}
	res, done, err := tc.Start(ctx, name, args)
	if err != nil {
		return nil, err
	}
	if done {
		fmt.Fprintln(os.Stderr, "[TASK] the server answered synchronously")
	} else {
		task, err := tc.Wait(ctx, res["taskId"].(string))
		if err != nil {
			return nil, err
		}
		if res, err = client.TaskResult(task); err != nil {
			return nil, err
		}
	}
	data, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	var result mcp.CallToolResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decoding the task result: %w", err)
	}
	return &result, nil
}
