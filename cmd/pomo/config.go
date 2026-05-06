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

// WeeklyReport holds configuration for the report subcommand.
type WeeklyReport struct {
	WorkDays []string `yaml:"work_days"`
}

// Config holds user preferences loaded from the YAML config file.
type Config struct {
	PomodoroTime      int            `yaml:"pomodoro_time"`
	ShortBreakTime    int            `yaml:"short_break_time"`
	LongBreakTime     int            `yaml:"long_break_time"`
	PomodorosPerCycle int            `yaml:"pomodoros_per_cycle"`
	AutoStartWork     bool           `yaml:"auto_start_work"`
	SessionSummary    SessionSummary `yaml:"session_summary"`
	Categories        []string       `yaml:"categories"`
	WeeklyReport      WeeklyReport   `yaml:"weekly_report"`
}

// builtinCategories are written to the config file on first run.
var builtinCategories = []string{
	"MEETING", "TECHNICAL WORK", "STRATEGY WORK", "MEETING PREP", "121", "CHORE",
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
		PomodoroTime:      25,
		ShortBreakTime:    5,
		LongBreakTime:     10,
		PomodorosPerCycle: 4,
		AutoStartWork:     false,
		SessionSummary: SessionSummary{
			Create: true,
			Folder: "~/pomos",
			Tags:   []string{"daily", "pomo-summary"},
		},
		WeeklyReport: WeeklyReport{
			WorkDays: []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
		},
	}
}

// validWeekdays is the set of accepted weekday abbreviations.
var validWeekdays = map[string]bool{
	"Mon": true, "Tue": true, "Wed": true, "Thu": true,
	"Fri": true, "Sat": true, "Sun": true,
}

// processWorkDays trims whitespace, validates, and deduplicates work day entries.
// Returns the Mon–Fri default if the result would be empty.
func processWorkDays(days []string) []string {
	defaults := []string{"Mon", "Tue", "Wed", "Thu", "Fri"}
	if len(days) == 0 {
		return defaults
	}
	seen := make(map[string]bool)
	var out []string
	for _, d := range days {
		d = strings.TrimSpace(d)
		if !validWeekdays[d] {
			fmt.Fprintf(os.Stderr, "warning: invalid work day %q (expected Mon–Sun), skipping\n", d)
			continue
		}
		if seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	if len(out) == 0 {
		return defaults
	}
	return out
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
	cfg.Categories = processCategories(cfg.Categories)
	cfg.WeeklyReport.WorkDays = processWorkDays(cfg.WeeklyReport.WorkDays)
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
