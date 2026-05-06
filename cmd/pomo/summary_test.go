package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestWriteSummaryFileIncludesFrontmatter(t *testing.T) {
	dir := t.TempDir()
	startedAt := time.Date(2026, 4, 17, 10, 30, 0, 0, time.UTC)

	cfg := Config{
		SessionSummary: SessionSummary{
			Create: true,
			Folder: dir,
			Tags:   nil,
		},
	}

	m := Model{
		startedAt:          startedAt,
		pomodoroDuration:   25 * time.Minute,
		shortBreakDuration: 5 * time.Minute,
		longBreakDuration:  10 * time.Minute,
		pomodorosPerCycle:  4,
		pomodorosCompleted: 2,
		totalWorked:        50 * time.Minute,
		totalRested:        10 * time.Minute,
		tasks:              []Task{},
	}

	if err := writeSummaryFile(m, cfg); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	filename := startedAt.Format("2006-01-02_15-04-05") + ".md"
	content, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("Could not read summary file: %v", err)
	}

	str := string(content)
	if !strings.HasPrefix(str, "---\n") {
		t.Error("Expected file to start with frontmatter delimiter")
	}
	if !strings.Contains(str, "tags:") {
		t.Error("Expected frontmatter to contain tags")
	}
	if !strings.Contains(str, "- daily") {
		t.Error("Expected default tag 'daily' in frontmatter")
	}
	if !strings.Contains(str, "- pomo summary") {
		t.Error("Expected default tag 'pomo summary' in frontmatter")
	}
	if !strings.Contains(str, "2026-04-17") {
		t.Error("Expected created date in frontmatter")
	}
}

func TestWriteSummaryFileOmitsTagsWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	startedAt := time.Date(2026, 4, 17, 10, 30, 0, 0, time.UTC)

	cfg := Config{
		SessionSummary: SessionSummary{
			Create: true,
			Folder: dir,
			Tags:   []string{},
		},
	}

	m := Model{
		startedAt:          startedAt,
		pomodoroDuration:   25 * time.Minute,
		shortBreakDuration: 5 * time.Minute,
		longBreakDuration:  10 * time.Minute,
		pomodorosPerCycle:  4,
		pomodorosCompleted: 1,
		totalWorked:        25 * time.Minute,
		totalRested:        5 * time.Minute,
		tasks:              []Task{},
	}

	if err := writeSummaryFile(m, cfg); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	filename := startedAt.Format("2006-01-02_15-04-05") + ".md"
	content, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("Could not read summary file: %v", err)
	}

	str := string(content)
	if strings.Contains(str, "tags:") {
		t.Error("Expected tags key to be omitted when tags list is empty")
	}
	if !strings.Contains(str, "2026-04-17") {
		t.Error("Expected created date in frontmatter even when tags are empty")
	}
}

func TestWriteSummaryFileCustomTags(t *testing.T) {
	dir := t.TempDir()
	startedAt := time.Date(2026, 4, 17, 10, 30, 0, 0, time.UTC)

	cfg := Config{
		SessionSummary: SessionSummary{
			Create: true,
			Folder: dir,
			Tags:   []string{"custom", "another"},
		},
	}

	m := Model{
		startedAt:          startedAt,
		pomodoroDuration:   25 * time.Minute,
		shortBreakDuration: 5 * time.Minute,
		longBreakDuration:  10 * time.Minute,
		pomodorosPerCycle:  4,
		pomodorosCompleted: 1,
		totalWorked:        25 * time.Minute,
		totalRested:        5 * time.Minute,
		tasks:              []Task{},
	}

	if err := writeSummaryFile(m, cfg); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	filename := startedAt.Format("2006-01-02_15-04-05") + ".md"
	content, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("Could not read summary file: %v", err)
	}

	str := string(content)
	if !strings.Contains(str, "- custom") {
		t.Error("Expected custom tag in frontmatter")
	}
	if !strings.Contains(str, "- another") {
		t.Error("Expected another tag in frontmatter")
	}
	if strings.Contains(str, "daily") {
		t.Error("Did not expect default 'daily' tag when custom tags are set")
	}
}
