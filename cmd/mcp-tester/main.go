// Package main is the entry point for the mcp-tester CLI.
package main

import (
	"fmt"
	"os"

	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/spf13/cobra"
)

var (
	// command holds the stdio execution command for the MCP server.
	command string
	// url holds the SSE endpoint URL for the MCP server.
	url string
	// profile holds the name of the configuration profile to use.
	profile string
	// verbose enables logging of client activity.
	verbose bool
	// raw enables raw mode to bypass strict SDK unmarshaling.
	raw bool
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:     "mcp-tester",
	Short:   "MCP-Tester is a tool to test Model Context Protocol (MCP) servers",
	Long:    fmt.Sprintf("MCP-Tester v%s - Developed by %s\n\nA command-line tool to test various MCP server transports, list tool schemas, and invoke tools for testing purposes.", version.Version, version.Author),
	Version: version.Version,
}

func init() {
	// Persistent flags are available to every subcommand.
	rootCmd.PersistentFlags().StringVarP(&command, "command", "c", "", "Command to run the MCP server (stdio)")
	rootCmd.PersistentFlags().StringVarP(&url, "url", "u", "", "URL of the MCP server (sse)")
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "Profile from mcp-tester.yml to use")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolVarP(&raw, "raw", "r", false, "Enable raw mode to bypass strict SDK unmarshaling")
}

func main() {
	// Execute the root command.
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
