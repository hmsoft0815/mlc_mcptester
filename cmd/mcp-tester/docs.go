package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hmsoft0815/mlc_mcptester/internal/mcpdocs"
	"github.com/spf13/cobra"
)

var (
	docsOutput string
	docsCheck  string
)

func init() {
	docsCmd.Flags().StringVarP(&docsOutput, "output", "o", "", "Write the reference to this file instead of stdout")
	docsCmd.Flags().StringVar(&docsCheck, "check", "", "Exit with an error if this file no longer matches the server")
	rootCmd.AddCommand(docsCmd)
}

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Write a Markdown reference of the server's tools, prompts and resources",
	Long: "Write a Markdown reference of the server's tools, prompts, resources and instructions, read over\n" +
		"the protocol. The text is always English and holds no date or version, so --check fails only\n" +
		"when what the server offers has changed.",
	RunE: runDocs,
}

func runDocs(cmd *cobra.Command, args []string) error {
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
	session, err := getClient(verbose).Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer session.Close()

	srv, err := mcpdocs.Fetch(ctx, session)
	if err != nil {
		return err
	}
	md, err := mcpdocs.Render(srv)
	if err != nil {
		return err
	}
	reportUndescribed(srv)

	switch {
	case docsCheck != "":
		return checkDocs(cmd, md)
	case docsOutput != "":
		return os.WriteFile(docsOutput, []byte(md), 0o644)
	}
	fmt.Print(md)
	return nil
}

// checkDocs fails if the file differs from the rendered reference.
func checkDocs(cmd *cobra.Command, md string) error {
	written, err := os.ReadFile(docsCheck)
	if err != nil {
		return fmt.Errorf("reading %s: %w", docsCheck, err)
	}
	if line := mcpdocs.FirstDifference(string(written), md); line > 0 {
		cmd.SilenceUsage = true
		return fmt.Errorf("%s is out of date (first difference in line %d); regenerate it with 'mcp-tester docs -o %s'", docsCheck, line, docsCheck)
	}
	fmt.Fprintf(os.Stderr, "%s is up to date\n", docsCheck)
	return nil
}

// reportUndescribed names parameters without a description on stderr: the
// reference shows an empty cell, and the model has to guess.
func reportUndescribed(srv *mcpdocs.Server) {
	var missing []string
	for _, t := range srv.Tools {
		missing = append(missing, mcpdocs.UndescribedParams(t.Name, t.InputSchema)...)
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "HINT: %d parameters have no description: %s\n", len(missing), strings.Join(missing, ", "))
	}
}
