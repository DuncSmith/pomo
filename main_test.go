package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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
		tasks:            []Task{},
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
		tasks:              []Task{},
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
		tasks:            []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
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

func TestGenerateIntervalName(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		hour     int
		expected string
	}{
		{name: "morning first", n: 1, hour: 8, expected: "Morning #1"},
		{name: "morning boundary start", n: 2, hour: 0, expected: "Morning #2"},
		{name: "morning boundary end", n: 3, hour: 11, expected: "Morning #3"},
		{name: "afternoon first", n: 1, hour: 12, expected: "Afternoon #1"},
		{name: "afternoon mid", n: 4, hour: 14, expected: "Afternoon #4"},
		{name: "afternoon boundary end", n: 2, hour: 16, expected: "Afternoon #2"},
		{name: "evening start", n: 1, hour: 17, expected: "Evening #1"},
		{name: "evening mid", n: 3, hour: 19, expected: "Evening #3"},
		{name: "evening boundary end", n: 2, hour: 20, expected: "Evening #2"},
		{name: "night start", n: 1, hour: 21, expected: "Night #1"},
		{name: "night late", n: 5, hour: 23, expected: "Night #5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 1, 1, tt.hour, 0, 0, 0, time.UTC)
			result := generateIntervalName(tt.n, now)
			if result != tt.expected {
				t.Errorf("generateIntervalName(%d, hour=%d): expected %q, got %q", tt.n, tt.hour, tt.expected, result)
			}
		})
	}
}

func TestStartTask(t *testing.T) {
	now := time.Now()
	m := Model{tasks: []Task{}}

	m.startTask("Write tests", now)

	if len(m.tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(m.tasks))
	}
	if m.tasks[0].Name != "Write tests" {
		t.Errorf("Expected task name 'Write tests', got %q", m.tasks[0].Name)
	}
	if !m.tasks[0].StartedAt.Equal(now) {
		t.Errorf("Expected StartedAt %v, got %v", now, m.tasks[0].StartedAt)
	}
	if !m.tasks[0].EndedAt.IsZero() {
		t.Errorf("Expected EndedAt to be zero for active task")
	}
}

func TestStartTaskEndsExistingTask(t *testing.T) {
	start := time.Now()
	m := Model{tasks: []Task{}}
	m.startTask("First task", start)

	later := start.Add(10 * time.Minute)
	m.startTask("Second task", later)

	if len(m.tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(m.tasks))
	}
	// First task should be ended
	if m.tasks[0].EndedAt.IsZero() {
		t.Error("Expected first task to be ended")
	}
	if !m.tasks[0].EndedAt.Equal(later) {
		t.Errorf("Expected first task EndedAt %v, got %v", later, m.tasks[0].EndedAt)
	}
	// Second task should be active
	if !m.tasks[1].EndedAt.IsZero() {
		t.Error("Expected second task to still be active")
	}
}

func TestActiveTask(t *testing.T) {
	m := Model{tasks: []Task{}}

	// No tasks: no active task
	if m.activeTask() != nil {
		t.Error("Expected nil active task when no tasks")
	}

	now := time.Now()
	m.startTask("My task", now)

	active := m.activeTask()
	if active == nil {
		t.Fatal("Expected active task, got nil")
	}
	if active.Name != "My task" {
		t.Errorf("Expected active task name 'My task', got %q", active.Name)
	}

	// End the task
	m.endActiveTask(now.Add(5 * time.Minute))
	if m.activeTask() != nil {
		t.Error("Expected no active task after ending it")
	}
}

func TestTransitionToRestEndsActiveTask(t *testing.T) {
	start := time.Now()
	m := Model{
		workDuration: 50 * time.Minute,
		restDuration: 10 * time.Minute,
		tasks:        []Task{},
	}
	m.startTask("Active task", start)

	m = m.transitionToRest(50 * time.Minute)

	if m.activeTask() != nil {
		t.Error("Expected no active task after transitioning to rest")
	}
	if m.tasks[0].EndedAt.IsZero() {
		t.Error("Expected task EndedAt to be set after transitioning to rest")
	}
}

func TestTransitionToWorkContinuesLastTask(t *testing.T) {
	start := time.Now()
	m := Model{
		workDuration:       50 * time.Minute,
		restDuration:       10 * time.Minute,
		intervalsCompleted: 0,
		isRest:             true,
		tasks:              []Task{},
	}
	// Simulate a task that was ended when rest began
	m.tasks = append(m.tasks, Task{
		Name:      "Carry forward task",
		StartedAt: start,
		EndedAt:   start.Add(50 * time.Minute),
	})

	m = m.transitionToWork()

	// Should have created a new task entry continuing the last task name
	if len(m.tasks) != 2 {
		t.Fatalf("Expected 2 task entries, got %d", len(m.tasks))
	}
	if m.tasks[1].Name != "Carry forward task" {
		t.Errorf("Expected continued task name 'Carry forward task', got %q", m.tasks[1].Name)
	}
	if !m.tasks[1].EndedAt.IsZero() {
		t.Error("Expected continued task to be active (no EndedAt)")
	}
}

func TestTransitionToWorkNoTasksNoContinuation(t *testing.T) {
	m := Model{
		workDuration:       50 * time.Minute,
		restDuration:       10 * time.Minute,
		intervalsCompleted: 0,
		isRest:             true,
		tasks:              []Task{},
	}

	m = m.transitionToWork()

	// No tasks to continue
	if len(m.tasks) != 0 {
		t.Errorf("Expected 0 tasks when no previous task, got %d", len(m.tasks))
	}
}

func TestTransitionToWorkGeneratesNewIntervalName(t *testing.T) {
	m := Model{
		workDuration:        50 * time.Minute,
		restDuration:        10 * time.Minute,
		intervalsCompleted:  0,
		isRest:              true,
		currentIntervalName: "My Custom Name",
		tasks:               []Task{},
	}

	m = m.transitionToWork()

	// Name should be auto-generated, not the old custom name
	if m.currentIntervalName == "My Custom Name" {
		t.Error("Expected interval name to be reset on new work phase, not carry over the custom name")
	}
	if m.currentIntervalName == "" {
		t.Error("Expected interval name to be set to an auto-generated value")
	}
}

func TestTaskKeyOpensTaskMode(t *testing.T) {
	m := Model{
		workDuration: 50 * time.Minute,
		restDuration: 10 * time.Minute,
		remaining:    40 * time.Minute,
		isRest:       false,
		tasks:        []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model := result.(Model)

	if !model.taskMode {
		t.Error("Expected taskMode to be true after pressing 'a'")
	}
}

func TestTaskKeyDisabledDuringRest(t *testing.T) {
	m := Model{
		workDuration: 50 * time.Minute,
		restDuration: 10 * time.Minute,
		remaining:    8 * time.Minute,
		isRest:       true,
		tasks:        []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model := result.(Model)

	if model.taskMode {
		t.Error("Expected taskMode to remain false during rest phase")
	}
}

func TestTaskInputConfirm(t *testing.T) {
	now := time.Now()
	m := Model{
		taskMode:  true,
		taskInput: "Fix bug",
		tasks:     []Task{},
	}
	// Simulate a prior active task to confirm it gets ended
	m.tasks = append(m.tasks, Task{Name: "Old task", StartedAt: now.Add(-10 * time.Minute)})

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model := result.(Model)

	if model.taskMode {
		t.Error("Expected taskMode to be false after confirming")
	}
	if model.taskInput != "" {
		t.Error("Expected taskInput to be cleared after confirming")
	}
	// Should have ended old task and created new one
	if len(model.tasks) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(model.tasks))
	}
	if model.tasks[0].EndedAt.IsZero() {
		t.Error("Expected old task to be ended")
	}
	if model.tasks[1].Name != "Fix bug" {
		t.Errorf("Expected new task name 'Fix bug', got %q", model.tasks[1].Name)
	}
}

func TestTaskInputCancel(t *testing.T) {
	m := Model{
		taskMode:  true,
		taskInput: "Work in progress",
		tasks:     []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	model := result.(Model)

	if model.taskMode {
		t.Error("Expected taskMode to be false after pressing esc")
	}
	if len(model.tasks) != 0 {
		t.Error("Expected no tasks to be created on cancel")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.SummaryFolder != "~/pomos" {
		t.Errorf("Expected SummaryFolder '~/pomos', got %q", cfg.SummaryFolder)
	}
	if cfg.WorkTime != 50 {
		t.Errorf("Expected WorkTime 50, got %d", cfg.WorkTime)
	}
	if cfg.IntervalTime != 60 {
		t.Errorf("Expected IntervalTime 60, got %d", cfg.IntervalTime)
	}
	if !cfg.ProduceSummary {
		t.Error("Expected ProduceSummary to be true by default")
	}
}

func TestLoadConfigCreatesDefaultOnFirstRun(t *testing.T) {
	// Point XDG_CONFIG_HOME at a temp dir so we don't touch the real config
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return defaults
	def := defaultConfig()
	if cfg.WorkTime != def.WorkTime {
		t.Errorf("Expected WorkTime %d, got %d", def.WorkTime, cfg.WorkTime)
	}
	if cfg.IntervalTime != def.IntervalTime {
		t.Errorf("Expected IntervalTime %d, got %d", def.IntervalTime, cfg.IntervalTime)
	}
	if cfg.SummaryFolder != def.SummaryFolder {
		t.Errorf("Expected SummaryFolder %q, got %q", def.SummaryFolder, cfg.SummaryFolder)
	}
	if cfg.ProduceSummary != def.ProduceSummary {
		t.Errorf("Expected ProduceSummary %v, got %v", def.ProduceSummary, cfg.ProduceSummary)
	}

	// Config file should have been created on disk
	expectedPath := filepath.Join(dir, "pomo", "config.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Expected config file to be created on first run, but it doesn't exist")
	}
}

func TestLoadConfigParsesValues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	configDir := filepath.Join(dir, "pomo")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Could not create config dir: %v", err)
	}

	content := "summary_folder: /custom/path\nwork_time: 25\ninterval_time: 45\nproduce_summary: false\n"
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Could not write config file: %v", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.SummaryFolder != "/custom/path" {
		t.Errorf("Expected SummaryFolder '/custom/path', got %q", cfg.SummaryFolder)
	}
	if cfg.WorkTime != 25 {
		t.Errorf("Expected WorkTime 25, got %d", cfg.WorkTime)
	}
	if cfg.IntervalTime != 45 {
		t.Errorf("Expected IntervalTime 45, got %d", cfg.IntervalTime)
	}
	if cfg.ProduceSummary {
		t.Error("Expected ProduceSummary to be false")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	configDir := filepath.Join(dir, "pomo")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Could not create config dir: %v", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(":::invalid yaml:::\n\t\tbad:"), 0644); err != nil {
		t.Fatalf("Could not write config file: %v", err)
	}

	_, err := loadConfig()
	if err == nil {
		t.Error("Expected error for invalid YAML, got none")
	}
}

func TestConfigPath(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
		path, err := configPath()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expected := "/custom/xdg/pomo/config.yaml"
		if path != expected {
			t.Errorf("Expected path %q, got %q", expected, path)
		}
	})

	t.Run("falls back to ~/.config when XDG_CONFIG_HOME unset", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		path, err := configPath()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", "pomo", "config.yaml")
		if path != expected {
			t.Errorf("Expected path %q, got %q", expected, path)
		}
	})
}

func TestParseArgsUsesConfigDefaults(t *testing.T) {
	cfg := Config{
		WorkTime:       30,
		IntervalTime:   75,
		SummaryFolder:  "~/pomos",
		ProduceSummary: true,
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

func TestParseArgsCLIOverridesConfig(t *testing.T) {
	cfg := Config{
		WorkTime:       30,
		IntervalTime:   75,
		SummaryFolder:  "~/pomos",
		ProduceSummary: true,
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
