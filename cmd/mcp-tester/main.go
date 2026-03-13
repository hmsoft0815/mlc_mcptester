// Package main is the entry point for the mcp-tester CLI.
package main

import (
	"fmt"
	"os"

	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/spf13/cobra"
)

var (
	command       string
	url           string
	profile       string
	verbose       bool
	raw           bool
	checkIcons    bool
	downloadIcons string
	lang          string
	format        string
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:     "mcp-tester",
	Short:   "MCP-Tester is a tool to test Model Context Protocol (MCP) servers",
	Long:    fmt.Sprintf("MCP-Tester v%s - Developed by %s\n\nA command-line tool to test various MCP server transports, list tool schemas, and invoke tools for testing purposes.", version.Version, version.Author),
	Version: version.Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		i18n.Lang = lang
	},
}

func init() {
	// Persistent flags are available to every subcommand.
	rootCmd.PersistentFlags().StringVarP(&command, "command", "c", "", "Command to run the MCP server (stdio)")
	rootCmd.PersistentFlags().StringVarP(&url, "url", "u", "", "URL of the MCP server (sse)")
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "Profile from mcp-tester.yml to use")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolVarP(&raw, "raw", "r", false, "Enable raw mode to bypass strict SDK unmarshaling")
	rootCmd.PersistentFlags().BoolVar(&checkIcons, "check-icons", false, "Check if icon URIs are reachable")
	rootCmd.PersistentFlags().StringVar(&downloadIcons, "download-icons", "", "Download icons to the specified directory")
	rootCmd.PersistentFlags().StringVar(&lang, "lang", "en", "Language for output (en, de)")
	rootCmd.PersistentFlags().StringVar(&format, "format", "text", "Output format (text, json)")
}

func main() {
	// Execute the root command.
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
