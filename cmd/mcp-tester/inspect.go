package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/badge"
	mcpclient "github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

// InspectionReport is the result of inspect, printed with --format json.
type InspectionReport struct {
	ServerName            string   `json:"serverName"`
	ServerVersion         string   `json:"serverVersion"`
	ProtocolVersion       string   `json:"protocolVersion"`
	LatestProtocolVersion string   `json:"latestProtocolVersion"`
	HasInstructions       bool     `json:"hasInstructions"`
	Extensions            []string `json:"extensions,omitempty"`
	Score                 int      `json:"score"`
	Recommendations       []string `json:"recommendations"`
	ToolsFound            int      `json:"toolsFound"`
	PromptsFound          int      `json:"promptsFound"`
	ResourcesFound        int      `json:"resourcesFound"`
	Errors                []string `json:"errors,omitempty"`
	Infos                 []string `json:"infos,omitempty"`         // observations that do not affect the score
	TextOnlyTools         []string `json:"textOnlyTools,omitempty"` // exempt from the output schema check (--text-only)
}

var (
	minScore       int
	badgeSVG       string
	badgeJSON      string
	badgeHideScore bool
	readResources  bool
	textOnlyTools  []string
	hintsPerTool   bool
)

func init() {
	inspectCmd.Flags().IntVar(&minScore, "min-score", 0, "Exit with an error if the score is below this value")
	inspectCmd.Flags().StringVar(&badgeSVG, "badge", "", "Write an mcpcheck status badge (SVG) to this file")
	inspectCmd.Flags().StringVar(&badgeJSON, "badge-json", "", "Write the badge as shields.io endpoint JSON to this file")
	inspectCmd.Flags().BoolVar(&badgeHideScore, "badge-no-score", false, "Leave the quality score off the badge")
	inspectCmd.Flags().BoolVar(&readResources, "read-resources", false, "Also read the first resource and check its cache hints")
	inspectCmd.Flags().StringSliceVar(&textOnlyTools, "text-only", nil, "Tools that return plain text on purpose: no output schema hint and no deduction for them")
	inspectCmd.Flags().BoolVar(&hintsPerTool, "hints-per-tool", false, "One output schema hint per tool instead of a single summary line")
	rootCmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Analyze an MCP server and provide quality recommendations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInspect(cmd)
	},
}

// runInspect connects, runs the checks and reports.
func runInspect(cmd *cobra.Command) error {
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
		return err
	}
	defer session.Close()

	in := newInspector(ctx, session, c, u, textOnlySet(config))
	in.run()
	in.print()

	// Written before the verdict, so a failing server gets a red badge
	// instead of keeping the green one from its last good run.
	if err := writeBadges(in.report); err != nil {
		return err
	}
	return in.verdict(cmd)
}

// textOnlySet joins --text-only and the profile's text_only list.
func textOnlySet(config *Config) map[string]bool {
	textOnly := map[string]bool{}
	for _, name := range textOnlyTools {
		textOnly[name] = true
	}
	if config != nil {
		for _, name := range config.Profiles[profile].TextOnly {
			textOnly[name] = true
		}
	}
	return textOnly
}

// print writes the report as JSON or as the text quality report.
func (in *inspector) print() {
	if format == "json" {
		out, _ := json.MarshalIndent(in.report, "", "  ")
		fmt.Println(string(out))
		return
	}
	fmt.Println(i18n.T(i18n.MsgScore, in.report.Score))
	for _, e := range in.protocolErrors {
		fmt.Println("- " + e)
	}
	if len(in.recommendations) == 0 && len(in.protocolErrors) == 0 {
		fmt.Println(i18n.T(i18n.MsgPerfect))
	} else {
		for _, rec := range in.recommendations {
			fmt.Println("- " + rec)
		}
	}
	for _, i := range in.infos {
		fmt.Println("- " + i)
	}
	if in.declaredOutputSchemas > 0 {
		fmt.Println()
		fmt.Println(i18n.T(i18n.MsgOutputSchemaUnchecked, in.declaredOutputSchemas))
	}
}

// verdict fails the command on protocol errors or a score below --min-score.
// A finished inspection that finds problems is a result, not a usage mistake.
func (in *inspector) verdict(cmd *cobra.Command) error {
	if len(in.protocolErrors) > 0 {
		cmd.SilenceUsage = true
		return fmt.Errorf("%s", i18n.T(i18n.MsgInspectFailed, len(in.protocolErrors)))
	}
	if in.report.Score < minScore {
		cmd.SilenceUsage = true
		return fmt.Errorf("%s", i18n.T(i18n.MsgScoreBelowMin, in.report.Score, minScore))
	}
	return nil
}

// writeBadges writes the badge files the --badge flags ask for.
func writeBadges(report InspectionReport) error {
	if badgeSVG == "" && badgeJSON == "" {
		return nil
	}
	r := badge.Result{
		Revision:  report.ProtocolVersion,
		Latest:    report.LatestProtocolVersion,
		Score:     report.Score,
		Errors:    len(report.Errors),
		HideScore: badgeHideScore,
		Tester:    version.Version,
		Checked:   time.Now(),
	}
	if badgeSVG != "" {
		if err := os.WriteFile(badgeSVG, badge.SVG(r), 0o644); err != nil {
			return fmt.Errorf("writing badge: %w", err)
		}
	}
	if badgeJSON != "" {
		data, err := badge.JSON(r)
		if err != nil {
			return err
		}
		if err := os.WriteFile(badgeJSON, append(data, '\n'), 0o644); err != nil {
			return fmt.Errorf("writing badge JSON: %w", err)
		}
	}
	return nil
}

// sameToolOrder reports whether two tools/list results name the same tools in
// the same order.
func sameToolOrder(a, b []*mcp.Tool) bool {
	names := func(tools []*mcp.Tool) []string {
		out := make([]string, len(tools))
		for i, t := range tools {
			out[i] = t.Name
		}
		return out
	}
	return slices.Equal(names(a), names(b))
}

// listToolsAgain lists the tools a second time for the order check. Since
// 2026-07-28 the SDK serves repeated list calls from its ttlMs cache, so a
// fresh connection is needed to ask the server again.
func listToolsAgain(ctx context.Context, session *mcp.ClientSession, cmdArg, urlArg string) ([]*mcp.Tool, error) {
	if !mcpclient.IsStateless(session) {
		return mcpclient.ListAllTools(ctx, session)
	}
	transport, err := getTransport(ctx, cmdArg, urlArg)
	if err != nil {
		return nil, err
	}
	second, err := getClient(false).Connect(ctx, transport, nil)
	if err != nil {
		return nil, err
	}
	defer second.Close()
	return mcpclient.ListAllTools(ctx, second)
}
