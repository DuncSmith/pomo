package main

import (
	"testing"
	"time"
)

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

func TestTaskSummaryMergesDuplicates(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	tasks := []Task{
		{Name: "Feature A", StartedAt: t1, EndedAt: t1.Add(30 * time.Minute)},
		{Name: "Feature B", StartedAt: t1.Add(30 * time.Minute), EndedAt: t1.Add(50 * time.Minute)},
		{Name: "Feature A", StartedAt: t1.Add(50 * time.Minute), EndedAt: t1.Add(80 * time.Minute)},
	}

	// Compute totals the same way printSummary does
	taskTotals := make(map[string]time.Duration)
	for _, task := range tasks {
		end := task.EndedAt
		if end.IsZero() {
			end = time.Now()
		}
		taskTotals[task.Name] += end.Sub(task.StartedAt)
	}

	if taskTotals["Feature A"] != 60*time.Minute {
		t.Errorf("Expected Feature A total 60m, got %v", taskTotals["Feature A"])
	}
	if taskTotals["Feature B"] != 20*time.Minute {
		t.Errorf("Expected Feature B total 20m, got %v", taskTotals["Feature B"])
	}
}
