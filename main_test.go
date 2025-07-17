package main

import (
	"testing"
	"time"
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
			hasError: false,
		},
		{
			name:     "seconds with s suffix",
			input:    "45s",
			expected: 45 * time.Second,
			hasError: false,
		},
		{
			name:     "number without suffix defaults to minutes",
			input:    "25",
			expected: 25 * time.Minute,
			hasError: false,
		},
		{
			name:     "single digit minute",
			input:    "5m",
			expected: 5 * time.Minute,
			hasError: false,
		},
		{
			name:     "single digit second",
			input:    "10s",
			expected: 10 * time.Second,
			hasError: false,
		},
		{
			name:     "zero minutes",
			input:    "0m",
			expected: 0 * time.Minute,
			hasError: false,
		},
		{
			name:     "zero seconds",
			input:    "0s",
			expected: 0 * time.Second,
			hasError: false,
		},
		{
			name:     "invalid number with m suffix",
			input:    "abcm",
			expected: 0,
			hasError: true,
		},
		{
			name:     "invalid number with s suffix",
			input:    "abcs",
			expected: 0,
			hasError: true,
		},
		{
			name:     "invalid number without suffix",
			input:    "abc",
			expected: 0,
			hasError: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
			hasError: true,
		},
		{
			name:     "negative number",
			input:    "-5m",
			expected: -5 * time.Minute,
			hasError: false,
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