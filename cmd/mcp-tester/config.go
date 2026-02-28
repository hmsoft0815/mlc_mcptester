package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Profile represents a server configuration profile.
type Profile struct {
	Command string `yaml:"command,omitempty"`
	URL     string `yaml:"url,omitempty"`
}

// Config represents the tool's configuration file.
type Config struct {
	Profiles map[string]Profile `yaml:"profiles"`
}

// loadConfig reads the mcp-tester.yml file and returns the configuration.
func loadConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{Profiles: make(map[string]Profile)}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// resolveSettings applies a profile if specified, otherwise uses the command line flags.
func resolveSettings(config *Config, profileName string, cmdArg, urlArg string) (string, string, error) {
	if profileName != "" {
		profile, ok := config.Profiles[profileName]
		if !ok {
			return "", "", fmt.Errorf("profile not found: %s", profileName)
		}
		return profile.Command, profile.URL, nil
	}
	return cmdArg, urlArg, nil
}
