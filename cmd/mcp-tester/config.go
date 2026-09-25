package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Profile represents a server configuration profile.
type Profile struct {
	Command   string `yaml:"command,omitempty"`
	URL       string `yaml:"url,omitempty"`
	Transport string `yaml:"transport,omitempty"`
	Disabled  bool   `yaml:"disabled,omitempty"`
	// Headers and Bearer apply to HTTP transports. ${VAR} is expanded from
	// the environment, so secrets need not be stored in the file.
	Headers map[string]string `yaml:"headers,omitempty"`
	Bearer  string            `yaml:"bearer,omitempty"`
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

	if config.Profiles == nil {
		config.Profiles = make(map[string]Profile)
	}

	return &config, nil
}

// saveConfig writes the configuration to the mcp-tester.yml file.
func saveConfig(path string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// resolveSettings applies a profile if specified, otherwise uses the command line flags.
func resolveSettings(config *Config, profileName string, cmdArg, urlArg string) (string, string, error) {
	if profileName != "" {
		profile, ok := config.Profiles[profileName]
		if !ok {
			return "", "", fmt.Errorf("profile not found: %s", profileName)
		}
		if profile.Disabled {
			return "", "", fmt.Errorf("profile '%s' is disabled", profileName)
		}
		if profile.Transport != "" && transportType == "" {
			transportType = profile.Transport
		}
		// Command line flags win over the profile
		for name, value := range profile.Headers {
			headerFlags = append([]string{name + ": " + os.ExpandEnv(value)}, headerFlags...)
		}
		if profile.Bearer != "" && bearerToken == "" {
			bearerToken = os.ExpandEnv(profile.Bearer)
		}
		return profile.Command, profile.URL, nil
	}
	return cmdArg, urlArg, nil
}
