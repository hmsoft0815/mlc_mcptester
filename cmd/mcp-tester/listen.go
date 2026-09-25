package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	mcpclient "github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var (
	listenSubscribe []string
	listenDuration  time.Duration
)

func init() {
	listenCmd.Flags().StringArrayVar(&listenSubscribe, "subscribe", nil, "Resource URI to subscribe to (repeatable)")
	listenCmd.Flags().DurationVar(&listenDuration, "duration", 0, "Stop after this time (default: until Ctrl-C)")
	rootCmd.AddCommand(listenCmd)
}

var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "Print list-changed and resource-updated notifications (subscriptions/listen)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		if listenDuration > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, listenDuration)
			defer cancel()
		}

		config, _ := loadConfig("mcp-tester.yml")
		c, u, err := resolveSettings(config, profile, command, url)
		if err != nil {
			return err
		}
		transport, err := getTransport(ctx, c, u)
		if err != nil {
			return err
		}
		notes := &mcpclient.Notifications{Out: os.Stdout}
		session, err := newClient(verbose, cliResponder, notes).Connect(ctx, transport, nil)
		if err != nil {
			return err
		}
		defer session.Close()

		for _, uri := range listenSubscribe {
			if err := session.Subscribe(ctx, &mcp.SubscribeParams{URI: uri}); err != nil {
				return fmt.Errorf("subscribe %s: %w", uri, err)
			}
		}
		fmt.Fprintln(os.Stderr, "Listening for notifications, Ctrl-C to stop...")
		<-ctx.Done()
		return nil
	},
}
