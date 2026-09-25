package main

import (
	"context"
	"encoding/json"
	"fmt"

	mcpclient "github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/spf13/cobra"
)

var completeContext string

func init() {
	completeCmd.Flags().StringVar(&completeContext, "context", "", `Previously resolved arguments (JSON object, e.g. '{"owner":"acme"}')`)
	rootCmd.AddCommand(completeCmd)
}

var completeCmd = &cobra.Command{
	Use:   "complete <prompt:name|resource:uri> <argument> [value]",
	Short: "Ask the server for argument completions (completion/complete)",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, arg, value := args[0], args[1], ""
		if len(args) == 3 {
			value = args[2]
		}
		var contextArgs map[string]string
		if completeContext != "" {
			if err := json.Unmarshal([]byte(completeContext), &contextArgs); err != nil {
				return fmt.Errorf("failed to parse --context: %w", err)
			}
		}

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

		res, err := mcpclient.Complete(ctx, session, ref, arg, value, contextArgs)
		if err != nil {
			return err
		}

		if format == "json" {
			out, _ := json.MarshalIndent(res.Completion, "", "  ")
			fmt.Println(string(out))
			return nil
		}
		for _, v := range res.Completion.Values {
			fmt.Println(v)
		}
		if res.Completion.HasMore {
			fmt.Printf("(%d of %d, more available)\n", len(res.Completion.Values), res.Completion.Total)
		}
		return nil
	},
}
