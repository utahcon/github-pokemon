package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// OrgConfig represents a single organization entry in the config file.
type OrgConfig struct {
	Org  string `yaml:"org"`
	Path string `yaml:"path"`
}

// Config represents the top-level configuration file structure.
type Config struct {
	Orgs            []OrgConfig `yaml:"orgs"`
	Parallel        int         `yaml:"parallel,omitempty"`
	SkipUpdate      bool        `yaml:"skip_update,omitempty"`
	Verbose         bool        `yaml:"verbose,omitempty"`
	IncludeArchived bool        `yaml:"include_archived,omitempty"`
	NoColor         bool        `yaml:"no_color,omitempty"`
}

// defaultConfigPath returns ~/.config/github-pokemon/config.yaml.
func defaultConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "github-pokemon", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".config", "github-pokemon", "config.yaml"), nil
}

// loadConfig reads and parses a YAML config file.
func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	if len(cfg.Orgs) == 0 {
		return Config{}, fmt.Errorf("config file %s has no orgs defined", path)
	}

	for i, entry := range cfg.Orgs {
		if entry.Org == "" {
			return Config{}, fmt.Errorf("config entry %d: \"org\" is required", i+1)
		}
		if entry.Path == "" {
			return Config{}, fmt.Errorf("config entry %d: \"path\" is required", i+1)
		}
	}

	return cfg, nil
}
