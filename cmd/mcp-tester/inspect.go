package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

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

		fmt.Printf("=== MCP Server Inspection: %s ===\n\n", profile)

		recommendations := []string{}
		score := 100

		// 1. Check Version & Identity
		initResult := session.InitializeResult()
		fmt.Printf("[✓] Server: %s (%s)\n", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
		fmt.Printf("[✓] Protocol Version: %s\n", initResult.ProtocolVersion)

		// 2. Check Capabilities
		caps := initResult.Capabilities
		fmt.Print("[i] Capabilities: ")
		features := []string{}
		if caps.Logging != nil {
			features = append(features, "Logging")
		}
		if caps.Resources != nil {
			features = append(features, "Resources")
		}
		if caps.Prompts != nil {
			features = append(features, "Prompts")
		}
		if caps.Tools != nil {
			features = append(features, "Tools")
		}
		fmt.Println(strings.Join(features, ", "))

		// 3. Analyze Prompts (System Context)
		promptsResult, err := session.ListPrompts(ctx, nil)
		if err == nil {
			if len(promptsResult.Prompts) == 0 {
				recommendations = append(recommendations, "WARNUNG: Keine Prompts definiert. Prompts werden dringend empfohlen, um den System-Kontext und die Persona des Servers für das LLM festzulegen.")
				score -= 20
			} else {
				fmt.Printf("[✓] %d Prompts gefunden.\n", len(promptsResult.Prompts))
			}
		} else {
			recommendations = append(recommendations, "WARNUNG: Der Server unterstützt keine Prompts (oder Fehler beim Abrufen).")
			score -= 10
		}

		// 4. Analyze Tools
		toolsResult, err := session.ListTools(ctx, nil)
		if err == nil {
			if len(toolsResult.Tools) == 0 {
				recommendations = append(recommendations, "INFO: Der Server bietet keine Tools an.")
			} else {
				fmt.Printf("[✓] %d Tools gefunden.\n", len(toolsResult.Tools))
				for _, t := range toolsResult.Tools {
					if t.Description == "" {
						recommendations = append(recommendations, fmt.Sprintf("WARNUNG: Tool '%s' hat keine Beschreibung. Das LLM benötigt diese, um den Zweck des Tools zu verstehen.", t.Name))
						score -= 5
					}
					if t.InputSchema == nil {
						recommendations = append(recommendations, fmt.Sprintf("FEHLER: Tool '%s' hat kein Input-Schema.", t.Name))
						score -= 10
					}
					if t.OutputSchema == nil {
						recommendations = append(recommendations, fmt.Sprintf("HINWEIS: Tool '%s' hat kein Output-Schema. Strukturierte Rückgaben helfen dem LLM, die Ergebnisse präziser zu verarbeiten.", t.Name))
						score -= 2
					}
				}
			}
		}

		// 5. Analyze Resources
		resourcesResult, err := session.ListResources(ctx, nil)
		if err == nil {
			if len(resourcesResult.Resources) > 0 {
				fmt.Printf("[✓] %d Ressourcen gefunden.\n", len(resourcesResult.Resources))
			}
		}

		// Output Report
		fmt.Printf("\n--- Quality Report (Score: %d/100) ---\n", score)
		if len(recommendations) == 0 {
			fmt.Println("Perfekt! Der Server folgt allen Best Practices.")
		} else {
			for _, rec := range recommendations {
				fmt.Println("- " + rec)
			}
		}

		return nil
	},
}
