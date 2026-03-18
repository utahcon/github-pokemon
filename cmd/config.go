package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// loadOrCreateConfig loads an existing config or returns an empty one if the file doesn't exist.
func loadOrCreateConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return cfg, nil
}

// saveConfig writes the config to disk, creating parent directories as needed.
func saveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file %s: %w", path, err)
	}

	return nil
}

// configHasOrg returns true if the config already contains the given org/path pair.
func configHasOrg(cfg Config, org, path string) bool {
	for _, entry := range cfg.Orgs {
		if entry.Org == org && entry.Path == path {
			return true
		}
	}
	return false
}

// configLookupOrg returns the first matching OrgConfig for the given org name, or false if not found.
func configLookupOrg(cfg Config, org string) (OrgConfig, bool) {
	for _, entry := range cfg.Orgs {
		if entry.Org == org {
			return entry, true
		}
	}
	return OrgConfig{}, false
}

// promptToSaveConfig asks the user if they want to save the org/path to the config file.
// Returns true if the entry was saved.
func promptToSaveConfig(org, path, cfgPath string) bool {
	fmt.Printf("\nWould you like to save org %q with path %q to your config file? [y/N] ", org, path)

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return false
	}

	cfg, err := loadOrCreateConfig(cfgPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return false
	}

	if configHasOrg(cfg, org, path) {
		fmt.Println("Already in config, skipping.")
		return false
	}

	cfg.Orgs = append(cfg.Orgs, OrgConfig{Org: org, Path: path})

	if err := saveConfig(cfgPath, cfg); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
		return false
	}

	fmt.Printf("Saved to %s\n", cfgPath)
	return true
}
