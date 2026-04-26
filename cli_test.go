package main

import (
	"testing"
	"time"
)

func TestVersionVariables(t *testing.T) {
	if version == "" {
		t.Error("version should have a default value")
	}
	if commit == "" {
		t.Error("commit should have a default value")
	}
	if date == "" {
		t.Error("date should have a default value")
	}

	if version != "dev" {
		t.Errorf("Expected default version 'dev', got %q", version)
	}
	if commit != "none" {
		t.Errorf("Expected default commit 'none', got %q", commit)
	}
	if date != "unknown" {
		t.Errorf("Expected default date 'unknown', got %q", date)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		hasError bool
	}{
		{
			name:     "minutes with m suffix",
			input:    "30m",
			expected: 30 * time.Minute,
		},
		{
			name:     "seconds with s suffix",
			input:    "45s",
			expected: 45 * time.Second,
		},
		{
			name:     "number without suffix defaults to minutes",
			input:    "25",
			expected: 25 * time.Minute,
		},
		{
			name:     "single digit minute",
			input:    "5m",
			expected: 5 * time.Minute,
		},
		{
			name:     "single digit second",
			input:    "10s",
			expected: 10 * time.Second,
		},
		{
			name:     "zero minutes",
			input:    "0m",
			expected: 0 * time.Minute,
		},
		{
			name:     "zero seconds",
			input:    "0s",
			expected: 0 * time.Second,
		},
		{
			name:     "invalid number with m suffix",
			input:    "abcm",
			hasError: true,
		},
		{
			name:     "invalid number with s suffix",
			input:    "abcs",
			hasError: true,
		},
		{
			name:     "invalid number without suffix",
			input:    "abc",
			hasError: true,
		},
		{
			name:     "empty string",
			input:    "",
			hasError: true,
		},
		{
			name:     "negative number",
			input:    "-5m",
			expected: -5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDuration(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("For input %q, expected %v, got %v", tt.input, tt.expected, result)
				}
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectError  bool
		expectAction string
		workDuration time.Duration
		restDuration time.Duration
		intervalDur  time.Duration
	}{
		{
			name:         "defaults (no args)",
			args:         []string{},
			workDuration: 50 * time.Minute,
			restDuration: 10 * time.Minute,
			intervalDur:  60 * time.Minute,
		},
		{
			name:         "custom work duration (bare number)",
			args:         []string{"25"},
			workDuration: 25 * time.Minute,
			restDuration: 35 * time.Minute,
			intervalDur:  60 * time.Minute,
		},
		{
			name:         "custom work duration (with m suffix)",
			args:         []string{"30m"},
			workDuration: 30 * time.Minute,
			restDuration: 30 * time.Minute,
			intervalDur:  60 * time.Minute,
		},
		{
			name:         "custom work duration (seconds)",
			args:         []string{"90s"},
			workDuration: 90 * time.Second,
			restDuration: 60*time.Minute - 90*time.Second,
			intervalDur:  60 * time.Minute,
		},
		{
			name:         "custom interval",
			args:         []string{"--interval", "90"},
			workDuration: 50 * time.Minute,
			restDuration: 40 * time.Minute,
			intervalDur:  90 * time.Minute,
		},
		{
			name:         "custom work and interval",
			args:         []string{"45", "-i", "90"},
			workDuration: 45 * time.Minute,
			restDuration: 45 * time.Minute,
			intervalDur:  90 * time.Minute,
		},
		{
			name:         "interval with m suffix",
			args:         []string{"30", "--interval", "75m"},
			workDuration: 30 * time.Minute,
			restDuration: 45 * time.Minute,
			intervalDur:  75 * time.Minute,
		},
		{
			name:         "help flag",
			args:         []string{"-h"},
			expectAction: "help",
		},
		{
			name:         "help flag (long form)",
			args:         []string{"--help"},
			expectAction: "help",
		},
		{
			name:         "version flag",
			args:         []string{"-v"},
			expectAction: "version",
		},
		{
			name:         "version flag (long form)",
			args:         []string{"--version"},
			expectAction: "version",
		},
		{
			name:        "invalid work duration",
			args:        []string{"abc"},
			expectError: true,
		},
		{
			name:        "invalid interval duration",
			args:        []string{"--interval", "xyz"},
			expectError: true,
		},
		{
			name:        "missing interval value",
			args:        []string{"--interval"},
			expectError: true,
		},
		{
			name:        "unexpected argument",
			args:        []string{"25", "30"},
			expectError: true,
		},
		{
			name:        "zero work duration",
			args:        []string{"0"},
			expectError: true,
		},
		{
			name:        "zero interval",
			args:        []string{"-i", "0"},
			expectError: true,
		},
		{
			name:        "work duration >= interval",
			args:        []string{"60", "-i", "60"},
			expectError: true,
		},
		{
			name:        "work duration > interval",
			args:        []string{"70", "-i", "60"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseArgs(tt.args, defaultConfig())

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for args %v, but got none", tt.args)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for args %v: %v", tt.args, err)
				return
			}

			if tt.expectAction != "" {
				if result.action != tt.expectAction {
					t.Errorf("Expected action %q, got %q", tt.expectAction, result.action)
				}
				if tt.expectAction == "version" && result.versionInfo == "" {
					t.Error("Expected versionInfo to be populated for version action")
				}
				return
			}

			if result.model == nil {
				t.Fatal("Expected model to be populated")
			}
			m := result.model
			if m.workDuration != tt.workDuration {
				t.Errorf("Expected workDuration %v, got %v", tt.workDuration, m.workDuration)
			}
			if m.restDuration != tt.restDuration {
				t.Errorf("Expected restDuration %v, got %v", tt.restDuration, m.restDuration)
			}
			if m.intervalDuration != tt.intervalDur {
				t.Errorf("Expected intervalDuration %v, got %v", tt.intervalDur, m.intervalDuration)
			}
			if m.remaining != tt.workDuration {
				t.Errorf("Expected remaining to match workDuration %v, got %v", tt.workDuration, m.remaining)
			}
			if m.isRest {
				t.Error("Expected isRest to be false initially")
			}
			if m.intervalsCompleted != 0 {
				t.Errorf("Expected intervalsCompleted to be 0, got %d", m.intervalsCompleted)
			}
		})
	}
}

func TestParseArgsUsesConfigDefaults(t *testing.T) {
	cfg := Config{
		WorkTime:     30,
		IntervalTime: 75,
		SessionSummary: SessionSummary{
			Folder: "~/pomos",
			Create: true,
		},
	}
	result, err := parseArgs([]string{}, cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.model.workDuration != 30*time.Minute {
		t.Errorf("Expected workDuration 30m from config, got %v", result.model.workDuration)
	}
	if result.model.intervalDuration != 75*time.Minute {
		t.Errorf("Expected intervalDuration 75m from config, got %v", result.model.intervalDuration)
	}
	if result.model.restDuration != 45*time.Minute {
		t.Errorf("Expected restDuration 45m (75-30), got %v", result.model.restDuration)
	}
}

func TestParseArgsCreateSessionSummaryFlag(t *testing.T) {
	t.Run("--create-session-summary sets override to true", func(t *testing.T) {
		result, err := parseArgs([]string{"--create-session-summary"}, defaultConfig())
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.createSessionSummary == nil {
			t.Fatal("Expected createSessionSummary to be set, got nil")
		}
		if !*result.createSessionSummary {
			t.Error("Expected createSessionSummary to be true")
		}
	})

	t.Run("--no-create-session-summary sets override to false", func(t *testing.T) {
		result, err := parseArgs([]string{"--no-create-session-summary"}, defaultConfig())
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.createSessionSummary == nil {
			t.Fatal("Expected createSessionSummary to be set, got nil")
		}
		if *result.createSessionSummary {
			t.Error("Expected createSessionSummary to be false")
		}
	})

	t.Run("no flag leaves override as nil", func(t *testing.T) {
		result, err := parseArgs([]string{}, defaultConfig())
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.createSessionSummary != nil {
			t.Errorf("Expected createSessionSummary to be nil, got %v", *result.createSessionSummary)
		}
	})
}

func TestParseArgsCLIOverridesConfig(t *testing.T) {
	cfg := Config{
		WorkTime:     30,
		IntervalTime: 75,
		SessionSummary: SessionSummary{
			Folder: "~/pomos",
			Create: true,
		},
	}
	result, err := parseArgs([]string{"45", "-i", "90"}, cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.model.workDuration != 45*time.Minute {
		t.Errorf("Expected CLI work 45m to override config 30m, got %v", result.model.workDuration)
	}
	if result.model.intervalDuration != 90*time.Minute {
		t.Errorf("Expected CLI interval 90m to override config 75m, got %v", result.model.intervalDuration)
	}
}
