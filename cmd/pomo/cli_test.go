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
		name              string
		args              []string
		expectError       bool
		expectAction      string
		pomodoroDuration  time.Duration
		shortBreak        time.Duration
		longBreak         time.Duration
		pomodorosPerCycle int
		autoStart         bool
	}{
		{
			name:              "defaults (no args)",
			args:              []string{},
			pomodoroDuration:  25 * time.Minute,
			shortBreak:        5 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
		},
		{
			name:              "custom pomodoro duration (bare number)",
			args:              []string{"50"},
			pomodoroDuration:  50 * time.Minute,
			shortBreak:        5 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
		},
		{
			name:              "custom pomodoro duration (with m suffix)",
			args:              []string{"30m"},
			pomodoroDuration:  30 * time.Minute,
			shortBreak:        5 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
		},
		{
			name:              "custom pomodoro duration (seconds)",
			args:              []string{"90s"},
			pomodoroDuration:  90 * time.Second,
			shortBreak:        5 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
		},
		{
			name:              "custom short break",
			args:              []string{"--short-break", "8"},
			pomodoroDuration:  25 * time.Minute,
			shortBreak:        8 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
		},
		{
			name:              "custom long break",
			args:              []string{"--long-break", "20"},
			pomodoroDuration:  25 * time.Minute,
			shortBreak:        5 * time.Minute,
			longBreak:         20 * time.Minute,
			pomodorosPerCycle: 4,
		},
		{
			name:              "custom per-cycle",
			args:              []string{"--per-cycle", "3"},
			pomodoroDuration:  25 * time.Minute,
			shortBreak:        5 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 3,
		},
		{
			name:              "all flags combined",
			args:              []string{"50", "-s", "10", "-L", "20", "-c", "3"},
			pomodoroDuration:  50 * time.Minute,
			shortBreak:        10 * time.Minute,
			longBreak:         20 * time.Minute,
			pomodorosPerCycle: 3,
		},
		{
			name:              "auto start work flag",
			args:              []string{"--auto-start-work"},
			pomodoroDuration:  25 * time.Minute,
			shortBreak:        5 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
			autoStart:         true,
		},
		{
			name:              "short break with m suffix",
			args:              []string{"30", "--short-break", "7m"},
			pomodoroDuration:  30 * time.Minute,
			shortBreak:        7 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
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
			name:        "invalid pomodoro duration",
			args:        []string{"abc"},
			expectError: true,
		},
		{
			name:        "invalid short-break duration",
			args:        []string{"--short-break", "xyz"},
			expectError: true,
		},
		{
			name:        "missing short-break value",
			args:        []string{"--short-break"},
			expectError: true,
		},
		{
			name:        "invalid long-break duration",
			args:        []string{"--long-break", "xyz"},
			expectError: true,
		},
		{
			name:        "missing long-break value",
			args:        []string{"--long-break"},
			expectError: true,
		},
		{
			name:        "invalid per-cycle value",
			args:        []string{"--per-cycle", "abc"},
			expectError: true,
		},
		{
			name:        "missing per-cycle value",
			args:        []string{"--per-cycle"},
			expectError: true,
		},
		{
			name:        "unexpected argument",
			args:        []string{"25", "30"},
			expectError: true,
		},
		{
			name:        "zero pomodoro duration",
			args:        []string{"0"},
			expectError: true,
		},
		{
			name:        "zero short break",
			args:        []string{"-s", "0"},
			expectError: true,
		},
		{
			name:        "zero long break",
			args:        []string{"-L", "0"},
			expectError: true,
		},
		{
			name:        "zero per-cycle",
			args:        []string{"-c", "0"},
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
			if m.pomodoroDuration != tt.pomodoroDuration {
				t.Errorf("Expected pomodoroDuration %v, got %v", tt.pomodoroDuration, m.pomodoroDuration)
			}
			if m.shortBreakDuration != tt.shortBreak {
				t.Errorf("Expected shortBreakDuration %v, got %v", tt.shortBreak, m.shortBreakDuration)
			}
			if m.longBreakDuration != tt.longBreak {
				t.Errorf("Expected longBreakDuration %v, got %v", tt.longBreak, m.longBreakDuration)
			}
			if m.pomodorosPerCycle != tt.pomodorosPerCycle {
				t.Errorf("Expected pomodorosPerCycle %d, got %d", tt.pomodorosPerCycle, m.pomodorosPerCycle)
			}
			if m.remaining != tt.pomodoroDuration {
				t.Errorf("Expected remaining to match pomodoroDuration %v, got %v", tt.pomodoroDuration, m.remaining)
			}
			if m.isRest {
				t.Error("Expected isRest to be false initially")
			}
			if m.pomodorosCompleted != 0 {
				t.Errorf("Expected pomodorosCompleted to be 0, got %d", m.pomodorosCompleted)
			}
			if m.autoStartWork != tt.autoStart {
				t.Errorf("Expected autoStartWork %v, got %v", tt.autoStart, m.autoStartWork)
			}
		})
	}
}

func TestParseArgsUsesConfigDefaults(t *testing.T) {
	cfg := Config{
		PomodoroTime:      30,
		ShortBreakTime:    7,
		LongBreakTime:     15,
		PomodorosPerCycle: 3,
		AutoStartWork:     true,
		SessionSummary: SessionSummary{
			Folder: "~/pomos",
			Create: true,
		},
	}
	result, err := parseArgs([]string{}, cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.model.pomodoroDuration != 30*time.Minute {
		t.Errorf("Expected pomodoroDuration 30m from config, got %v", result.model.pomodoroDuration)
	}
	if result.model.shortBreakDuration != 7*time.Minute {
		t.Errorf("Expected shortBreakDuration 7m from config, got %v", result.model.shortBreakDuration)
	}
	if result.model.longBreakDuration != 15*time.Minute {
		t.Errorf("Expected longBreakDuration 15m from config, got %v", result.model.longBreakDuration)
	}
	if result.model.pomodorosPerCycle != 3 {
		t.Errorf("Expected pomodorosPerCycle 3 from config, got %d", result.model.pomodorosPerCycle)
	}
	if !result.model.autoStartWork {
		t.Error("Expected autoStartWork true from config")
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
		PomodoroTime:      30,
		ShortBreakTime:    7,
		LongBreakTime:     15,
		PomodorosPerCycle: 3,
		SessionSummary: SessionSummary{
			Folder: "~/pomos",
			Create: true,
		},
	}
	result, err := parseArgs([]string{"45", "-s", "8", "-L", "20", "-c", "5"}, cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.model.pomodoroDuration != 45*time.Minute {
		t.Errorf("Expected CLI pomodoro 45m to override config 30m, got %v", result.model.pomodoroDuration)
	}
	if result.model.shortBreakDuration != 8*time.Minute {
		t.Errorf("Expected CLI short-break 8m to override config 7m, got %v", result.model.shortBreakDuration)
	}
	if result.model.longBreakDuration != 20*time.Minute {
		t.Errorf("Expected CLI long-break 20m to override config 15m, got %v", result.model.longBreakDuration)
	}
	if result.model.pomodorosPerCycle != 5 {
		t.Errorf("Expected CLI per-cycle 5 to override config 3, got %d", result.model.pomodorosPerCycle)
	}
}
