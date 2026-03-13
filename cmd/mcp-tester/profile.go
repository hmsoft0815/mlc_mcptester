package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileAddCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileEnableCmd)
	profileCmd.AddCommand(profileDisableCmd)

	profileAddCmd.Flags().StringVarP(&addCommand, "command", "c", "", "Command to run the MCP server (stdio)")
	profileAddCmd.Flags().StringVarP(&addURL, "url", "u", "", "URL of the MCP server (sse)")
}

var (
	addCommand string
	addURL     string
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage MCP server profiles",
}

var profileAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new MCP server profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if addCommand == "" && addURL == "" {
			return fmt.Errorf("either --command (-c) or --url (-u) must be provided")
		}
		config, err := loadConfig("mcp-tester.yml")
		if err != nil {
			return err
		}

		config.Profiles[name] = Profile{
			Command: addCommand,
			URL:     addURL,
		}

		if err := saveConfig("mcp-tester.yml", config); err != nil {
			return err
		}
		fmt.Printf("Profile '%s' added successfully.\n", name)
		return nil
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an MCP server profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		config, err := loadConfig("mcp-tester.yml")
		if err != nil {
			return err
		}

		if _, ok := config.Profiles[name]; !ok {
			return fmt.Errorf("profile '%s' not found", name)
		}

		delete(config.Profiles, name)

		if err := saveConfig("mcp-tester.yml", config); err != nil {
			return err
		}
		fmt.Printf("Profile '%s' deleted successfully.\n", name)
		return nil
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all MCP server profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := loadConfig("mcp-tester.yml")
		if err != nil {
			return err
		}

		if len(config.Profiles) == 0 {
			fmt.Println("No profiles found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tTYPE\tVALUE")

		keys := make([]string, 0, len(config.Profiles))
		for k := range config.Profiles {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, name := range keys {
			p := config.Profiles[name]
			status := "enabled"
			if p.Disabled {
				status = "disabled"
			}
			pType := "stdio"
			pValue := p.Command
			if p.URL != "" {
				pType = "sse"
				pValue = p.URL
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, status, pType, pValue)
		}
		w.Flush()
		return nil
	},
}

var profileEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable an MCP server profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setProfileDisabled(args[0], false)
	},
}

var profileDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable an MCP server profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setProfileDisabled(args[0], true)
	},
}

func setProfileDisabled(name string, disabled bool) error {
	config, err := loadConfig("mcp-tester.yml")
	if err != nil {
		return err
	}

	p, ok := config.Profiles[name]
	if !ok {
		return fmt.Errorf("profile '%s' not found", name)
	}

	p.Disabled = disabled
	config.Profiles[name] = p

	if err := saveConfig("mcp-tester.yml", config); err != nil {
		return err
	}

	status := "enabled"
	if disabled {
		status = "disabled"
	}
	fmt.Printf("Profile '%s' %s successfully.\n", name, status)
	return nil
}
