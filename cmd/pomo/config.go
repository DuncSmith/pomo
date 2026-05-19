package main

import (
	"fmt"
	"os"
	"path/filepath"

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
	PomodoroTime      int            `yaml:"pomodoro_time"`
	ShortBreakTime    int            `yaml:"short_break_time"`
	LongBreakTime     int            `yaml:"long_break_time"`
	PomodorosPerCycle int            `yaml:"pomodoros_per_cycle"`
	SessionSummary    SessionSummary `yaml:"session_summary"`
}

// defaultConfig returns the built-in default configuration.
func defaultConfig() Config {
	return Config{
		PomodoroTime:      25,
		ShortBreakTime:    5,
		LongBreakTime:     10,
		PomodorosPerCycle: 4,
		SessionSummary: SessionSummary{
			Create: true,
			Folder: "~/pomos",
			Tags:   []string{"daily", "pomo-summary"},
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

// loadConfig reads the config file. If no config file exists, built-in defaults are returned.
func loadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return defaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultConfig(), nil
	}
	if err != nil {
		return defaultConfig(), fmt.Errorf("could not read config file: %w", err)
	}

	cfg := defaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig(), fmt.Errorf("could not parse config file %s: %w", path, err)
	}
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
