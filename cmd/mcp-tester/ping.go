package main

import (
	"context"
	"fmt"
	mcpclient "github.com/hmsoft0815/mlc_mcptester/internal/client"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pingCmd)
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Send a ping request to the MCP server",
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

		fmt.Println("Sending ping...")
		method, err := mcpclient.Ping(ctx, session)
		if err != nil {
			return fmt.Errorf("ping failed: %w", err)
		}
		if method != "ping" {
			fmt.Printf("Protocol %s has no ping; the server answered %s.\n", session.InitializeResult().ProtocolVersion, method)
		}
		fmt.Println("Ping successful!")
		return nil
	},
}
