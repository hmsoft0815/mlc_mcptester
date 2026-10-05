package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hmsoft0815/mlc_mcptester/internal/badge"
	mcpclient "github.com/hmsoft0815/mlc_mcptester/internal/client"
	"github.com/hmsoft0815/mlc_mcptester/internal/httpcheck"
	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/hmsoft0815/mlc_mcptester/internal/skillcheck"
	"github.com/hmsoft0815/mlc_mcptester/internal/version"
	"github.com/hmsoft0815/mlc_mcptester/pkg/mcpskills"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

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

// listFailurePenalty is deducted for each list request that fails although its
// capability is declared: clients break on it, so it outweighs any style hint.
const listFailurePenalty = 50

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

// deductions sums penalties per category and caps each category, so that a
// server with many tools is not punished for the same mistake without bound.
type deductions struct {
	sums map[string]int
	caps map[string]int
}

func newDeductions(caps map[string]int) *deductions {
	return &deductions{sums: map[string]int{}, caps: caps}
}

func (d *deductions) add(category string, points int) { d.sums[category] += points }

func (d *deductions) total() int {
	total := 0
	for category, sum := range d.sums {
		if limit, ok := d.caps[category]; ok && sum > limit {
			sum = limit
		}
		total += sum
	}
	return total
}

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Analyze an MCP server and provide quality recommendations",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		config, _ := loadConfig("mcp-tester.yml")
		c, u, err := resolveSettings(config, profile, command, url)
		if err != nil {
			return err
		}
		textOnly := map[string]bool{}
		for _, name := range textOnlyTools {
			textOnly[name] = true
		}
		if config != nil {
			for _, name := range config.Profiles[profile].TextOnly {
				textOnly[name] = true
			}
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

		report := InspectionReport{LatestProtocolVersion: latestProtocolRevision}
		recommendations := []string{}
		protocolErrors := []string{}
		score := 100
		listFailed := func(method string, err error) {
			protocolErrors = append(protocolErrors, i18n.T(i18n.MsgListFailed, method, err))
			score -= listFailurePenalty
		}
		warn := func(msg string) { recommendations = append(recommendations, msg) }
		infos := []string{}
		info := func(msg string) { infos = append(infos, msg) }
		d := newDeductions(map[string]int{
			"description":  20,
			"outputSchema": 10,
			"protocol":     30,
			"toolName":     15,
			"duplicate":    10,
			"schemaType":   15,
			"title":        5,
			"icon":         15,
			"cache":        3,
			"cacheScope":   10,
			"xMCPHeader":   20,
		})
		// inspect never calls a tool — it cannot know which ones are free of
		// side effects — so a declared schema is all it can see, not whether the
		// results honour it.
		declaredOutputSchemas := 0
		// Missing output schemas are collected so they cost one line, not one per tool
		var noOutputSchema, exemptTextOnly []string

		initResult := session.InitializeResult()
		report.ServerName = initResult.ServerInfo.Name
		report.ServerVersion = initResult.ServerInfo.Version
		report.ProtocolVersion = initResult.ProtocolVersion
		report.HasInstructions = initResult.Instructions != ""
		modern := report.ProtocolVersion >= latestProtocolRevision

		switch behind := revisionsBehind(report.ProtocolVersion); {
		case behind < 0:
			warn(i18n.T(i18n.MsgUnknownProtocol, report.ProtocolVersion))
			d.add("protocol", 10)
		case behind > 0:
			warn(i18n.T(i18n.MsgOutdatedProtocol, report.ProtocolVersion, behind, latestProtocolRevision))
			d.add("protocol", 10*behind)
		}

		caps := initResult.Capabilities
		for name := range caps.Extensions {
			report.Extensions = append(report.Extensions, name)
		}
		sort.Strings(report.Extensions)

		if format == "text" {
			fmt.Print(i18n.T(i18n.MsgInspectionTitle, profile))
			fmt.Print(i18n.T(i18n.MsgServerInfo, report.ServerName, report.ServerVersion))
			fmt.Print(i18n.T(i18n.MsgProtocolVersion, report.ProtocolVersion))
			fmt.Println(i18n.T(i18n.MsgCapabilities))

			fmt.Print(i18n.T(i18n.MsgTools, caps.Tools != nil))
			if caps.Tools != nil && caps.Tools.ListChanged {
				fmt.Print(i18n.T(i18n.MsgSubscription))
			}
			fmt.Println()

			fmt.Print(i18n.T(i18n.MsgPrompts, caps.Prompts != nil))
			if caps.Prompts != nil && caps.Prompts.ListChanged {
				fmt.Print(i18n.T(i18n.MsgSubscription))
			}
			fmt.Println()

			fmt.Print(i18n.T(i18n.MsgResources, caps.Resources != nil))
			fmt.Println()

			fmt.Print(i18n.T(i18n.MsgCompletions, caps.Completions != nil))
			//lint:ignore SA1019 logging is deprecated since 2026-07-28 (SEP-2577) but still used by servers and regular before; remove with T-20260927-05
			fmt.Print(i18n.T(i18n.MsgLogging, caps.Logging != nil))
			fmt.Print(i18n.T(i18n.MsgProgress))
			fmt.Print(i18n.T(i18n.MsgCancel))
			if report.HasInstructions {
				fmt.Print(i18n.T(i18n.MsgInstructions, len(initResult.Instructions)))
			} else {
				fmt.Print(i18n.T(i18n.MsgNoInstructions))
			}
			if len(report.Extensions) > 0 {
				fmt.Print(i18n.T(i18n.MsgExtensions, strings.Join(report.Extensions, ", ")))
			}
		}

		checkIcons := func(owner string, icons []mcp.Icon) {
			for _, icon := range icons {
				if msg := checkIconSource(icon.Source); msg != "" {
					warn(i18n.T(i18n.MsgInvalidIcon, owner, msg, icon.Source))
					d.add("icon", 5)
				}
			}
		}
		// Cache hints are part of cacheable results since 2026-07-28. Read from
		// the wire: the SDK turns a missing ttlMs into 0 and fills cacheScope.
		authenticated := u != "" && (bearerToken != "" || len(headerFlags) > 0 ||
			oauthEnabled || oauthClientCredentials || oauthEnterprise)
		cacheHints := func(method string, params map[string]any) (float64, string, bool) {
			if params == nil {
				params = map[string]any{}
			}
			params["_meta"] = mcpclient.RequestMeta(session, nil)
			result, err := mcpclient.CallRaw(ctx, session, method, params)
			if err != nil {
				return 0, "", false
			}
			ttl, scope, problems := checkCacheHints(result)
			if len(problems) > 0 {
				warn(i18n.T(i18n.MsgCacheHints, method, strings.Join(problems, "; ")))
				d.add("cache", 1)
			}
			return ttl, scope, true
		}
		// Collected per finding, so the INFO lines name all lists at once
		var staleLists, publicLists []string
		checkCache := func(method string) {
			if !modern {
				return
			}
			ttl, scope, ok := cacheHints(method, nil)
			if !ok {
				return
			}
			if ttl == 0 {
				staleLists = append(staleLists, method)
			}
			if scope == "public" && authenticated {
				publicLists = append(publicLists, method)
			}
		}

		checkIcons("server "+report.ServerName, initResult.ServerInfo.Icons)

		prompts, err := mcpclient.ListAllPrompts(ctx, session)
		if err != nil && caps.Prompts != nil {
			listFailed("prompts/list", err)
		}
		if err == nil {
			report.PromptsFound = len(prompts)
			if report.PromptsFound == 0 {
				warn(i18n.T(i18n.MsgNoPrompts))
				score -= 20
			} else if format == "text" {
				fmt.Print(i18n.T(i18n.MsgFound, report.PromptsFound, "prompts"))
			}
			for _, p := range prompts {
				checkIcons("prompt '"+p.Name+"'", p.Icons)
			}
			checkCache("prompts/list")
		}

		tools, err := mcpclient.ListAllTools(ctx, session)
		if err != nil && caps.Tools != nil {
			listFailed("tools/list", err)
		}
		if err == nil {
			report.ToolsFound = len(tools)
			if report.ToolsFound > 0 {
				if format == "text" {
					fmt.Print(i18n.T(i18n.MsgFound, report.ToolsFound, "tools"))
				}

				totalDeductionInputSchema := 0
				totalBonusSafety := 0
				seen := map[string]bool{}

				for _, t := range tools {
					if msg := checkToolName(t.Name); msg != "" {
						warn(i18n.T(i18n.MsgInvalidToolName, t.Name, msg))
						d.add("toolName", 3)
					}
					if seen[t.Name] {
						warn(i18n.T(i18n.MsgDuplicateToolName, t.Name))
						d.add("duplicate", 5)
					}
					seen[t.Name] = true
					if t.Title == "" && (t.Annotations == nil || t.Annotations.Title == "") {
						warn(i18n.T(i18n.MsgNoTitle, t.Name))
						d.add("title", 1)
					}
					if t.Description == "" {
						warn(i18n.T(i18n.MsgNoDescription, t.Name))
						d.add("description", 5)
					}
					if t.InputSchema == nil {
						warn(i18n.T(i18n.MsgNoInputSchema, t.Name))
						totalDeductionInputSchema += 10
					} else if !inputSchemaIsObject(t.InputSchema) {
						warn(i18n.T(i18n.MsgInputSchemaNotObject, t.Name))
						d.add("schemaType", 5)
					}
					switch {
					case t.OutputSchema != nil:
						declaredOutputSchemas++
					case textOnly[t.Name]:
						exemptTextOnly = append(exemptTextOnly, t.Name)
					default:
						noOutputSchema = append(noOutputSchema, t.Name)
						d.add("outputSchema", 1)
					}
					checkIcons("tool '"+t.Name+"'", t.Icons)
					if _, problems := httpcheck.XMCPHeaders(t.InputSchema); len(problems) > 0 {
						warn(i18n.T(i18n.MsgInvalidXMCPHeader, t.Name, strings.Join(problems, "; ")))
						d.add("xMCPHeader", 10)
					}

					// Bonus for safety annotations (readOnlyHint)
					if t.Annotations != nil {
						if t.Annotations.ReadOnlyHint {
							totalBonusSafety += 2
						}
					}
				}

				if hintsPerTool {
					for _, name := range noOutputSchema {
						warn(i18n.T(i18n.MsgNoOutputSchema, name))
					}
				} else if len(noOutputSchema) > 0 {
					warn(i18n.T(i18n.MsgNoOutputSchemaSummary, len(noOutputSchema), strings.Join(noOutputSchema, ", ")))
				}
				if len(exemptTextOnly) > 0 {
					info(i18n.T(i18n.MsgTextOnlyTools, strings.Join(exemptTextOnly, ", ")))
					report.TextOnlyTools = exemptTextOnly
				}

				// The spec asks for a deterministic order so clients can cache the list
				if again, err := listToolsAgain(ctx, session, c, u); err == nil && !sameToolOrder(tools, again) {
					warn(i18n.T(i18n.MsgToolOrderUnstable))
					score -= 5
				}

				if totalBonusSafety > 20 {
					totalBonusSafety = 20
				}
				score -= totalDeductionInputSchema
				score += totalBonusSafety
			}
			checkCache("tools/list")
		}

		resources, err := mcpclient.ListAllResources(ctx, session)
		if err != nil && caps.Resources != nil {
			listFailed("resources/list", err)
		}
		if err == nil {
			report.ResourcesFound = len(resources)
			if report.ResourcesFound > 0 && format == "text" {
				fmt.Print(i18n.T(i18n.MsgFound, report.ResourcesFound, "resources"))
			}
			for _, r := range resources {
				checkIcons("resource '"+r.URI+"'", r.Icons)
			}
			checkCache("resources/list")
			// Resource contents are typically per user: "public" under
			// credentials can leak them through shared caches
			switch {
			case !modern || len(resources) == 0:
			case readResources:
				uri := resources[0].URI
				if _, scope, ok := cacheHints("resources/read", map[string]any{"uri": uri}); !ok {
					info(i18n.T(i18n.MsgResourceReadFailed, uri, "no result"))
				} else if scope == "public" && authenticated {
					warn(i18n.T(i18n.MsgCachePublicResource, uri))
					d.add("cacheScope", 10)
				}
			case authenticated:
				info(i18n.T(i18n.MsgReadResourcesHint))
			}
		}

		if len(staleLists) > 0 {
			info(i18n.T(i18n.MsgCacheStale, strings.Join(staleLists, ", ")))
		}
		if len(publicLists) > 0 {
			info(i18n.T(i18n.MsgCachePublicList, strings.Join(publicLists, ", ")))
		}

		// Skills are instructions that reach the model: count them, details via 'skills'
		if _, ok := caps.Extensions[mcpskills.Extension]; ok {
			skillsReport := (&skillcheck.Checker{Session: session}).Run(ctx)
			if format == "text" {
				fmt.Print(i18n.T(i18n.MsgFound, len(skillsReport.Skills), "skills"))
			}
			if skillsReport.Failed() {
				warn(i18n.T(i18n.MsgSkillsInvalid))
				score -= 10
			}
		}

		// Logging is deprecated as of 2026-07-28 (SEP-2577): no deduction
		// either way, but a server that still declares it should know that
		// support is running out. Older revisions keep it as a regular feature.
		//lint:ignore SA1019 logging is deprecated since 2026-07-28 (SEP-2577) but still used by servers and regular before; remove with T-20260927-05
		if modern && caps.Logging != nil {
			info(i18n.T(i18n.MsgDeprecatedLogging))
		}

		score -= d.total()

		// Clamp score to 0-100
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}

		report.Score = score
		report.Recommendations = recommendations
		report.Errors = protocolErrors
		report.Infos = infos

		if format == "json" {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Println(i18n.T(i18n.MsgScore, score))
			for _, e := range protocolErrors {
				fmt.Println("- " + e)
			}
			if len(recommendations) == 0 && len(protocolErrors) == 0 {
				fmt.Println(i18n.T(i18n.MsgPerfect))
			} else {
				for _, rec := range recommendations {
					fmt.Println("- " + rec)
				}
			}
			for _, i := range infos {
				fmt.Println("- " + i)
			}
			if declaredOutputSchemas > 0 {
				fmt.Println()
				fmt.Println(i18n.T(i18n.MsgOutputSchemaUnchecked, declaredOutputSchemas))
			}
		}

		// Written before the verdict, so a failing server gets a red badge
		// instead of keeping the green one from its last good run.
		if err := writeBadges(report); err != nil {
			return err
		}

		// A finished inspection that finds problems is a result, not a usage mistake
		if len(protocolErrors) > 0 {
			cmd.SilenceUsage = true
			return fmt.Errorf("%s", i18n.T(i18n.MsgInspectFailed, len(protocolErrors)))
		}
		if score < minScore {
			cmd.SilenceUsage = true
			return fmt.Errorf("%s", i18n.T(i18n.MsgScoreBelowMin, score, minScore))
		}
		return nil
	},
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
