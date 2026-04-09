package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			input:    0,
			expected: "00:00",
		},
		{
			name:     "seconds only",
			input:    30 * time.Second,
			expected: "00:30",
		},
		{
			name:     "minutes only",
			input:    5 * time.Minute,
			expected: "05:00",
		},
		{
			name:     "minutes and seconds",
			input:    3*time.Minute + 45*time.Second,
			expected: "03:45",
		},
		{
			name:     "double digit minutes",
			input:    25*time.Minute + 10*time.Second,
			expected: "25:10",
		},
		{
			name:     "hours converted to minutes",
			input:    time.Hour + 30*time.Minute + 15*time.Second,
			expected: "90:15",
		},
		{
			name:     "single digit second",
			input:    2*time.Minute + 5*time.Second,
			expected: "02:05",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTime(tt.input)
			if result != tt.expected {
				t.Errorf("For input %v, expected %q, got %q", tt.input, tt.expected, result)
			}
		})
	}
}

func TestCreateProgressBar(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		width      int
		expected   string
	}{
		{
			name:       "zero percentage",
			percentage: 0,
			width:      10,
			expected:   "░░░░░░░░░░",
		},
		{
			name:       "full percentage",
			percentage: 100,
			width:      10,
			expected:   "██████████",
		},
		{
			name:       "half percentage",
			percentage: 50,
			width:      10,
			expected:   "█████░░░░░",
		},
		{
			name:       "25 percent",
			percentage: 25,
			width:      20,
			expected:   "█████░░░░░░░░░░░░░░░",
		},
		{
			name:       "75 percent",
			percentage: 75,
			width:      8,
			expected:   "██████░░",
		},
		{
			name:       "width of 1",
			percentage: 60,
			width:      1,
			expected:   "░",
		},
		{
			name:       "width of 1 zero percent",
			percentage: 0,
			width:      1,
			expected:   "░",
		},
		{
			name:       "decimal percentage rounds down",
			percentage: 33.3,
			width:      9,
			expected:   "██░░░░░░░",
		},
		{
			name:       "decimal percentage rounds down 2",
			percentage: 66.7,
			width:      9,
			expected:   "██████░░░",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createProgressBar(tt.percentage, tt.width)
			if result != tt.expected {
				t.Errorf("For percentage %v and width %d, expected %q, got %q", tt.percentage, tt.width, tt.expected, result)
			}
		})
	}
}

func TestFormatDurationHuman(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{
			name:     "zero",
			input:    0,
			expected: "0s",
		},
		{
			name:     "seconds only",
			input:    30 * time.Second,
			expected: "30s",
		},
		{
			name:     "minutes and seconds",
			input:    5*time.Minute + 30*time.Second,
			expected: "5m 30s",
		},
		{
			name:     "hours and minutes",
			input:    2*time.Hour + 15*time.Minute,
			expected: "2h 15m",
		},
		{
			name:     "exact minutes",
			input:    50 * time.Minute,
			expected: "50m 0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDurationHuman(tt.input)
			if result != tt.expected {
				t.Errorf("For input %v, expected %q, got %q", tt.input, tt.expected, result)
			}
		})
	}
}

func TestPhaseDuration(t *testing.T) {
	m := Model{
		workDuration: 50 * time.Minute,
		restDuration: 10 * time.Minute,
	}

	if m.phaseDuration() != 50*time.Minute {
		t.Errorf("Expected work duration, got %v", m.phaseDuration())
	}

	m.isRest = true
	if m.phaseDuration() != 10*time.Minute {
		t.Errorf("Expected rest duration, got %v", m.phaseDuration())
	}
}

func TestPhaseTransitionWorkToRest(t *testing.T) {
	m := Model{
		workDuration:     50 * time.Minute,
		restDuration:     10 * time.Minute,
		intervalDuration: 60 * time.Minute,
		remaining:        0,
		isRest:           false,
	}

	result, cmd := m.Update(finishedMsg{})
	model := result.(Model)

	if !model.isRest {
		t.Error("Expected model to switch to rest phase")
	}
	if model.remaining != 10*time.Minute {
		t.Errorf("Expected remaining to be 10m, got %v", model.remaining)
	}
	if model.intervalsCompleted != 0 {
		t.Errorf("Expected 0 completed intervals, got %d", model.intervalsCompleted)
	}
	if model.totalWorked != 50*time.Minute {
		t.Errorf("Expected 50m total worked, got %v", model.totalWorked)
	}
	if cmd == nil {
		t.Error("Expected tickCmd to be returned")
	}
}

func TestPhaseTransitionRestToWork(t *testing.T) {
	m := Model{
		workDuration:       50 * time.Minute,
		restDuration:       10 * time.Minute,
		intervalDuration:   60 * time.Minute,
		remaining:          0,
		isRest:             true,
		intervalsCompleted: 0,
		totalWorked:        50 * time.Minute,
	}

	result, cmd := m.Update(finishedMsg{})
	model := result.(Model)

	if model.isRest {
		t.Error("Expected model to switch to work phase")
	}
	if model.remaining != 50*time.Minute {
		t.Errorf("Expected remaining to be 50m, got %v", model.remaining)
	}
	if model.intervalsCompleted != 1 {
		t.Errorf("Expected 1 completed interval, got %d", model.intervalsCompleted)
	}
	if model.totalRested != 10*time.Minute {
		t.Errorf("Expected 10m total rested, got %v", model.totalRested)
	}
	if cmd == nil {
		t.Error("Expected tickCmd to be returned")
	}
}

func TestQuitTracksPartialProgress(t *testing.T) {
	m := Model{
		workDuration:     50 * time.Minute,
		restDuration:     10 * time.Minute,
		intervalDuration: 60 * time.Minute,
		remaining:        30 * time.Minute,
		isRest:           false,
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	model := result.(Model)

	if !model.quitting {
		t.Error("Expected quitting to be true")
	}
	if model.totalWorked != 20*time.Minute {
		t.Errorf("Expected 20m partial work tracked, got %v", model.totalWorked)
	}
}

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

	// Verify defaults for dev builds
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