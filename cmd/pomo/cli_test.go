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
		{name: "minutes with m suffix", input: "30m", expected: 30 * time.Minute},
		{name: "seconds with s suffix", input: "45s", expected: 45 * time.Second},
		{name: "number without suffix defaults to minutes", input: "25", expected: 25 * time.Minute},
		{name: "single digit minute", input: "5m", expected: 5 * time.Minute},
		{name: "single digit second", input: "10s", expected: 10 * time.Second},
		{name: "zero minutes", input: "0m", expected: 0 * time.Minute},
		{name: "zero seconds", input: "0s", expected: 0 * time.Second},
		{name: "invalid number with m suffix", input: "abcm", hasError: true},
		{name: "invalid number with s suffix", input: "abcs", hasError: true},
		{name: "invalid number without suffix", input: "abc", hasError: true},
		{name: "empty string", input: "", hasError: true},
		{name: "negative number", input: "-5m", expected: -5 * time.Minute},
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
			name:              "custom work duration (-w flag)",
			args:              []string{"-w", "50"},
			pomodoroDuration:  50 * time.Minute,
			shortBreak:        5 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
		},
		{
			name:              "custom work duration (--work long flag)",
			args:              []string{"--work", "40"},
			pomodoroDuration:  40 * time.Minute,
			shortBreak:        5 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
		},
		{
			name:              "custom work duration (with m suffix)",
			args:              []string{"-w", "30m"},
			pomodoroDuration:  30 * time.Minute,
			shortBreak:        5 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
		},
		{
			name:              "custom work duration (seconds)",
			args:              []string{"-w", "90s"},
			pomodoroDuration:  90 * time.Second,
			shortBreak:        5 * time.Minute,
			longBreak:         10 * time.Minute,
			pomodorosPerCycle: 4,
		},
		{name: "help flag", args: []string{"-h"}, expectAction: "help"},
		{name: "help flag (long form)", args: []string{"--help"}, expectAction: "help"},
		{name: "version flag", args: []string{"-v"}, expectAction: "version"},
		{name: "version flag (long form)", args: []string{"--version"}, expectAction: "version"},
		{name: "init flag", args: []string{"--init"}, expectAction: "init"},
		{name: "invalid work duration", args: []string{"-w", "abc"}, expectError: true},
		{name: "missing work value", args: []string{"-w"}, expectError: true},
		{name: "unknown argument", args: []string{"badarg"}, expectError: true},
		{name: "removed --short-break flag is unknown", args: []string{"--short-break", "8"}, expectError: true},
		{name: "removed --long-break flag is unknown", args: []string{"--long-break", "20"}, expectError: true},
		{name: "removed --per-cycle flag is unknown", args: []string{"--per-cycle", "3"}, expectError: true},
		{name: "removed --auto-start-work flag is unknown", args: []string{"--auto-start-work"}, expectError: true},
		{name: "zero work duration", args: []string{"-w", "0"}, expectError: true},
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
		})
	}
}

func TestParseArgsUsesConfigDefaults(t *testing.T) {
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
}

func TestParseArgsCLIWorkOverridesConfig(t *testing.T) {
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
	result, err := parseArgs([]string{"-w", "45"}, cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.model.pomodoroDuration != 45*time.Minute {
		t.Errorf("Expected CLI pomodoro 45m to override config 30m, got %v", result.model.pomodoroDuration)
	}
	if result.model.shortBreakDuration != 7*time.Minute {
		t.Errorf("Expected shortBreakDuration 7m from config (untouched by CLI), got %v", result.model.shortBreakDuration)
	}
	if result.model.longBreakDuration != 15*time.Minute {
		t.Errorf("Expected longBreakDuration 15m from config (untouched by CLI), got %v", result.model.longBreakDuration)
	}
	if result.model.pomodorosPerCycle != 3 {
		t.Errorf("Expected pomodorosPerCycle 3 from config (untouched by CLI), got %d", result.model.pomodorosPerCycle)
	}
}
