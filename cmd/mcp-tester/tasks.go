package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/taskcheck"
	"github.com/spf13/cobra"
)

var (
	tasksTool    string
	tasksArgs    string
	tasksCancel  bool
	tasksTimeout time.Duration
)

func init() {
	tasksCmd.Flags().StringVar(&tasksTool, "tool", "", "Call this tool as a task and check its life (the tool runs; see below)")
	tasksCmd.Flags().StringVarP(&tasksArgs, "args", "a", "{}", "Arguments for --tool (JSON)")
	tasksCmd.Flags().BoolVar(&tasksCancel, "cancel", false, "Cancel the task right after it was created")
	tasksCmd.Flags().DurationVar(&tasksTimeout, "timeout", 2*time.Minute, "How long to wait for a terminal status")
	rootCmd.AddCommand(tasksCmd)
}

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Check a server against the Tasks extension",
	Long: `Checks a server against the Tasks extension (io.modelcontextprotocol/tasks,
spec 2026-07-28). Without --tool no tool runs: tasks/get, tasks/update and
tasks/cancel are probed for the required error codes (-32021 without the
extension, -32602 for an unknown task).

With --tool the tool is called twice, so pick one without side effects or
one you are fine to run: once without the extension (no task may come back)
and once with it, then the task is followed to its end — handle, durable
creation, every tasks/get result, status transitions, input requests
(answered with --elicit/--sample, otherwise the task is cancelled).
MUST violations fail (exit 1).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		var toolArgs map[string]any
		if err := json.Unmarshal([]byte(tasksArgs), &toolArgs); err != nil {
			return fmt.Errorf("--args: %w", err)
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
		session, err := getClient(verbose).Connect(ctx, transport, nil)
		if err != nil {
			return err
		}
		defer session.Close()

		report := (&taskcheck.Checker{
			Session: session, Tool: tasksTool, Args: toolArgs,
			Cancel: tasksCancel, Responder: cliResponder, Timeout: tasksTimeout,
		}).Run(ctx)
		if format == "json" {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(out))
		} else {
			printTasks(report)
		}
		if report.Failed() {
			cmd.SilenceUsage = true
			return fmt.Errorf("the server violates the Tasks extension")
		}
		return nil
	},
}

func printTasks(report *taskcheck.Report) {
	if report.Tool != "" && report.TaskID != "" {
		fmt.Printf("=== Task of %s ===\n%s: %s\n\n", report.Tool, report.TaskID, strings.Join(report.Statuses, " → "))
	}
	fmt.Println("=== Checks ===")
	for _, r := range report.Results {
		fmt.Printf("[%s] %s  %s\n", r.Status, r.Name, strings.TrimSpace(r.Detail))
	}
}
