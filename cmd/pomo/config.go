package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SessionSummary holds configuration for session summary output.
type SessionSummary struct {
	Create bool     `yaml:"create_session_summary"`
	Folder string   `yaml:"summary_folder"`
	Tags   []string `yaml:"summary_tags"`
}

// Config holds user preferences loaded from the YAML config file.
type Config struct {
	WorkTime       int            `yaml:"work_time"`
	IntervalTime   int            `yaml:"interval_time"`
	LunchTime      int            `yaml:"lunch_time"`
	SessionSummary SessionSummary `yaml:"session_summary"`
	Categories     []string       `yaml:"categories"`
}

// builtinCategories are written to the config file on first run.
var builtinCategories = []string{
	"meeting", "technical work", "strategy work", "meeting prep", "121", "chore",
}

// processCategories trims whitespace, deduplicates, and caps at 9 entries.
// Returns nil when the input is empty so callers can treat nil as "not configured".
func processCategories(cats []string) []string {
	if len(cats) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	for _, c := range cats {
		c = strings.TrimSpace(c)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	if len(out) > 9 {
		fmt.Fprintf(os.Stderr, "Warning: only the first 9 categories will be used (%d configured)\n", len(out))
		out = out[:9]
	}
	return out
}

// defaultConfig returns the built-in default configuration.
func defaultConfig() Config {
	return Config{
		WorkTime:     50,
		IntervalTime: 60,
		LunchTime:    60,
		SessionSummary: SessionSummary{
			Create: true,
			Folder: "~/pomos",
			Tags:   []string{"daily", "pomo summary"},
		},
	}
}

// configPath returns the path to the config file, respecting XDG_CONFIG_HOME.
func configPath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "pomo", "config.yaml"), nil
}

// loadConfig reads the config file, creating it with defaults if it doesn't exist.
func loadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return defaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := defaultConfig()
		cfg.Categories = processCategories(builtinCategories)
		if writeErr := writeDefaultConfig(path, cfg); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create config file: %v\n", writeErr)
		}
		return cfg, nil
	}
	if err != nil {
		return defaultConfig(), fmt.Errorf("could not read config file: %w", err)
	}

	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig(), fmt.Errorf("could not parse config file %s: %w", path, err)
	}
	cfg.Categories = processCategories(cfg.Categories)
	return cfg, nil
}

// writeDefaultConfig writes a default config file to the given path.
func writeDefaultConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}
	return nil
}
