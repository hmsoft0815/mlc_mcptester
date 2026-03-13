package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hmsoft0815/mlc_mcptester/internal/i18n"
	"github.com/spf13/cobra"
)

type InspectionReport struct {
	ServerName      string   `json:"serverName"`
	ServerVersion   string   `json:"serverVersion"`
	ProtocolVersion string   `json:"protocolVersion"`
	Score           int      `json:"score"`
	Recommendations []string `json:"recommendations"`
	ToolsFound      int      `json:"toolsFound"`
	PromptsFound    int      `json:"promptsFound"`
	ResourcesFound  int      `json:"resourcesFound"`
}

func init() {
	rootCmd.AddCommand(inspectCmd)
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

		report := InspectionReport{}
		recommendations := []string{}
		score := 100

		initResult := session.InitializeResult()
		report.ServerName = initResult.ServerInfo.Name
		report.ServerVersion = initResult.ServerInfo.Version
		report.ProtocolVersion = initResult.ProtocolVersion

		if format == "text" {
			fmt.Print(i18n.T(i18n.MsgInspectionTitle, profile))
			fmt.Print(i18n.T(i18n.MsgServerInfo, report.ServerName, report.ServerVersion))
			fmt.Print(i18n.T(i18n.MsgProtocolVersion, report.ProtocolVersion))
			fmt.Println(i18n.T(i18n.MsgCapabilities))
		}

		caps := initResult.Capabilities
		if format == "text" {
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

			fmt.Print(i18n.T(i18n.MsgLogging, caps.Logging != nil))
			fmt.Print(i18n.T(i18n.MsgProgress))
			fmt.Print(i18n.T(i18n.MsgCancel))
		}

		promptsResult, err := session.ListPrompts(ctx, nil)
		if err == nil {
			report.PromptsFound = len(promptsResult.Prompts)
			if report.PromptsFound == 0 {
				recommendations = append(recommendations, i18n.T(i18n.MsgNoPrompts))
				score -= 20
			} else if format == "text" {
				fmt.Print(i18n.T(i18n.MsgFound, report.PromptsFound, "prompts"))
			}
		}

		toolsResult, err := session.ListTools(ctx, nil)
		if err == nil {
			report.ToolsFound = len(toolsResult.Tools)
			if report.ToolsFound > 0 {
				if format == "text" {
					fmt.Print(i18n.T(i18n.MsgFound, report.ToolsFound, "tools"))
				}

				totalDeductionDescription := 0
				totalDeductionOutputSchema := 0
				totalDeductionInputSchema := 0
				totalBonusSafety := 0

				for _, t := range toolsResult.Tools {
					if t.Description == "" {
						recommendations = append(recommendations, i18n.T(i18n.MsgNoDescription, t.Name))
						totalDeductionDescription += 5
					}
					if t.InputSchema == nil {
						recommendations = append(recommendations, i18n.T(i18n.MsgNoInputSchema, t.Name))
						totalDeductionInputSchema += 10
					}
					if t.OutputSchema == nil {
						recommendations = append(recommendations, i18n.T(i18n.MsgNoOutputSchema, t.Name))
						totalDeductionOutputSchema += 1
					}

					// Bonus for safety annotations (readOnlyHint)
					if t.Annotations != nil {
						if t.Annotations.ReadOnlyHint {
							totalBonusSafety += 2
						}
					}
				}

				// Apply caps to scoring
				if totalDeductionDescription > 20 {
					totalDeductionDescription = 20
				}
				if totalDeductionOutputSchema > 10 {
					totalDeductionOutputSchema = 10
				}
				if totalBonusSafety > 20 {
					totalBonusSafety = 20
				}

				score -= totalDeductionDescription
				score -= totalDeductionInputSchema
				score -= totalDeductionOutputSchema
				score += totalBonusSafety
			}
		}

		resourcesResult, err := session.ListResources(ctx, nil)
		if err == nil {
			report.ResourcesFound = len(resourcesResult.Resources)
			if report.ResourcesFound > 0 && format == "text" {
				fmt.Print(i18n.T(i18n.MsgFound, report.ResourcesFound, "resources"))
			}
		}

		if caps.Logging == nil {
			recommendations = append(recommendations, i18n.T(i18n.MsgNoLogging))
			score -= 5
		}

		// Clamp score to 0-100
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}

		report.Score = score
		report.Recommendations = recommendations

		if format == "json" {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Println(i18n.T(i18n.MsgScore, score))
			if len(recommendations) == 0 {
				fmt.Println(i18n.T(i18n.MsgPerfect))
			} else {
				for _, rec := range recommendations {
					fmt.Println("- " + rec)
				}
			}
		}

		return nil
	},
}
