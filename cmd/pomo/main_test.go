package main

import (
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

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

	m = m.transitionToRest(50*time.Minute, time.Now())

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

	m = m.transitionToWork(10*time.Minute, time.Now())

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

	m = m.transitionToWork(10*time.Minute, time.Now())

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

	m = m.transitionToWork(10*time.Minute, time.Now())

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

func TestRecentTaskNamesEmpty(t *testing.T) {
	names := recentTaskNames([]Task{})
	if len(names) != 0 {
		t.Errorf("Expected no names for empty task list, got %v", names)
	}
}

func TestRecentTaskNamesExcludesActive(t *testing.T) {
	now := time.Now()
	tasks := []Task{
		{Name: "task1", StartedAt: now, EndedAt: now.Add(10 * time.Minute)},
		{Name: "task2", StartedAt: now.Add(10 * time.Minute)}, // active (zero EndedAt)
	}
	names := recentTaskNames(tasks)
	for _, n := range names {
		if n == "task2" {
			t.Error("Expected active task to be excluded from recent list")
		}
	}
	if len(names) != 1 || names[0] != "task1" {
		t.Errorf("Expected [task1], got %v", names)
	}
}

func TestRecentTaskNamesMostRecentFirst(t *testing.T) {
	now := time.Now()
	tasks := []Task{
		{Name: "first", StartedAt: now, EndedAt: now.Add(10 * time.Minute)},
		{Name: "second", StartedAt: now.Add(10 * time.Minute), EndedAt: now.Add(20 * time.Minute)},
		{Name: "third", StartedAt: now.Add(20 * time.Minute), EndedAt: now.Add(30 * time.Minute)},
	}
	names := recentTaskNames(tasks)
	if len(names) != 3 {
		t.Fatalf("Expected 3 names, got %d", len(names))
	}
	if names[0] != "third" || names[1] != "second" || names[2] != "first" {
		t.Errorf("Expected most-recent-first order, got %v", names)
	}
}

func TestRecentTaskNamesDeduplicated(t *testing.T) {
	now := time.Now()
	tasks := []Task{
		{Name: "task1", StartedAt: now, EndedAt: now.Add(10 * time.Minute)},
		{Name: "task2", StartedAt: now.Add(10 * time.Minute), EndedAt: now.Add(20 * time.Minute)},
		{Name: "task1", StartedAt: now.Add(20 * time.Minute), EndedAt: now.Add(30 * time.Minute)},
	}
	names := recentTaskNames(tasks)
	if len(names) != 2 {
		t.Fatalf("Expected 2 deduplicated names, got %v", names)
	}
	// task1 appeared most recently, so it should be first
	if names[0] != "task1" || names[1] != "task2" {
		t.Errorf("Expected [task1 task2], got %v", names)
	}
}

func TestRecentTaskNamesMaxNine(t *testing.T) {
	now := time.Now()
	tasks := make([]Task, 11)
	for i := range tasks {
		tasks[i] = Task{
			Name:      fmt.Sprintf("task%d", i+1),
			StartedAt: now.Add(time.Duration(i) * time.Minute),
			EndedAt:   now.Add(time.Duration(i+1) * time.Minute),
		}
	}
	names := recentTaskNames(tasks)
	if len(names) != 9 {
		t.Errorf("Expected max 9 names, got %d", len(names))
	}
}

func TestTaskPickerSelectByDigit(t *testing.T) {
	now := time.Now()
	m := Model{
		taskMode: true,
		taskInput: "",
		recentTasks: []string{"task1", "task2"},
		tasks: []Task{
			{Name: "task1", StartedAt: now, EndedAt: now.Add(10 * time.Minute)},
			{Name: "task2", StartedAt: now.Add(10 * time.Minute), EndedAt: now.Add(20 * time.Minute)},
		},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	model := result.(Model)

	if model.taskMode {
		t.Error("Expected taskMode to be false after selecting from picker")
	}
	if model.recentTasks != nil {
		t.Error("Expected recentTasks to be cleared after selection")
	}
	active := model.activeTask()
	if active == nil || active.Name != "task1" {
		t.Errorf("Expected active task 'task1', got %v", active)
	}
}

func TestTaskPickerDigitOutOfRange(t *testing.T) {
	m := Model{
		taskMode:    true,
		taskInput:   "",
		recentTasks: []string{"task1", "task2"},
		tasks:       []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: '5', Text: "5"})
	model := result.(Model)

	if !model.taskMode {
		t.Error("Expected taskMode to remain true when digit is out of range")
	}
	if model.taskInput != "5" {
		t.Errorf("Expected digit to fall through to taskInput, got %q", model.taskInput)
	}
}

func TestTaskPickerEscClearsRecentTasks(t *testing.T) {
	m := Model{
		taskMode:    true,
		taskInput:   "",
		recentTasks: []string{"task1"},
		tasks:       []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	model := result.(Model)

	if model.taskMode {
		t.Error("Expected taskMode to be false after Esc")
	}
	if model.recentTasks != nil {
		t.Error("Expected recentTasks to be cleared after Esc")
	}
}

func TestTaskPickerDigitAfterTypingIsText(t *testing.T) {
	m := Model{
		taskMode:    true,
		taskInput:   "my",
		recentTasks: []string{"task1"},
		tasks:       []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	model := result.(Model)

	if model.taskInput != "my1" {
		t.Errorf("Expected digit to append to taskInput when not empty, got %q", model.taskInput)
	}
	if !model.taskMode {
		t.Error("Expected taskMode to remain true while typing")
	}
}

func TestTaskKeyPopulatesRecentTasks(t *testing.T) {
	now := time.Now()
	m := Model{
		workDuration: 50 * time.Minute,
		restDuration: 10 * time.Minute,
		remaining:    40 * time.Minute,
		isRest:       false,
		tasks: []Task{
			{Name: "old task", StartedAt: now, EndedAt: now.Add(10 * time.Minute)},
		},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model := result.(Model)

	if !model.taskMode {
		t.Error("Expected taskMode to be true")
	}
	if len(model.recentTasks) != 1 || model.recentTasks[0] != "old task" {
		t.Errorf("Expected recentTasks [old task], got %v", model.recentTasks)
	}
}

func TestSkipWorkTracksPartialTime(t *testing.T) {
	m := Model{
		workDuration:     50 * time.Minute,
		restDuration:     10 * time.Minute,
		intervalDuration: 60 * time.Minute,
		remaining:        30 * time.Minute, // 20 minutes into work
		isRest:           false,
		tasks:            []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model := result.(Model)

	if model.totalWorked != 20*time.Minute {
		t.Errorf("Expected 20m partial work tracked, got %v", model.totalWorked)
	}
	if !model.isRest {
		t.Error("Expected to transition to rest phase after skipping work")
	}
}

func TestSkipRestTracksPartialTime(t *testing.T) {
	m := Model{
		workDuration:     50 * time.Minute,
		restDuration:     10 * time.Minute,
		intervalDuration: 60 * time.Minute,
		remaining:        7 * time.Minute, // 3 minutes into rest
		isRest:           true,
		tasks:            []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	model := result.(Model)

	if model.totalRested != 3*time.Minute {
		t.Errorf("Expected 3m partial rest tracked, got %v", model.totalRested)
	}
	if model.isRest {
		t.Error("Expected to transition to work phase after skipping rest")
	}
}

func TestRecentCategoryForTask(t *testing.T) {
	now := time.Now()
	tasks := []Task{
		{Name: "emails", Category: "chore", StartedAt: now, EndedAt: now.Add(5 * time.Minute)},
		{Name: "design", Category: "strategy work", StartedAt: now.Add(5 * time.Minute), EndedAt: now.Add(15 * time.Minute)},
		{Name: "emails", Category: "meeting prep", StartedAt: now.Add(15 * time.Minute), EndedAt: now.Add(20 * time.Minute)},
	}

	// Should return the most recent category for "emails"
	cat, found := recentCategoryForTask(tasks, "emails")
	if !found || cat != "meeting prep" {
		t.Errorf("Expected ('meeting prep', true), got (%q, %v)", cat, found)
	}

	// Should return category for "design"
	cat, found = recentCategoryForTask(tasks, "design")
	if !found || cat != "strategy work" {
		t.Errorf("Expected ('strategy work', true), got (%q, %v)", cat, found)
	}

	// Unknown task returns not found
	cat, found = recentCategoryForTask(tasks, "unknown")
	if found {
		t.Errorf("Expected not found for unknown task, got (%q, %v)", cat, found)
	}

	// Uncategorised task returns not found
	uncatTasks := []Task{
		{Name: "emails", Category: "", StartedAt: now, EndedAt: now.Add(5 * time.Minute)},
	}
	cat, found = recentCategoryForTask(uncatTasks, "emails")
	if found {
		t.Errorf("Expected not found for uncategorised task, got (%q, %v)", cat, found)
	}
}

func TestTaskPickerResumeCategorisedTaskSkipsCategoryMode(t *testing.T) {
	now := time.Now()
	m := Model{
		taskMode:    true,
		taskInput:   "",
		recentTasks: []string{"emails"},
		categories:  []string{"chore", "meeting prep", "technical work"},
		tasks: []Task{
			{Name: "emails", Category: "meeting prep", StartedAt: now, EndedAt: now.Add(10 * time.Minute)},
		},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	model := result.(Model)

	if model.taskMode {
		t.Error("Expected taskMode to be false")
	}
	if model.categoryMode {
		t.Error("Expected categoryMode to be false — categorised task should skip category picker")
	}
	active := model.activeTask()
	if active == nil || active.Name != "emails" {
		t.Errorf("Expected active task 'emails', got %v", active)
	}
	if active != nil && active.Category != "meeting prep" {
		t.Errorf("Expected category 'meeting prep', got %q", active.Category)
	}
}

func TestTaskPickerResumeUncategorisedTaskEntersCategoryMode(t *testing.T) {
	now := time.Now()
	m := Model{
		taskMode:    true,
		taskInput:   "",
		recentTasks: []string{"emails"},
		categories:  []string{"chore", "meeting prep"},
		tasks: []Task{
			{Name: "emails", Category: "", StartedAt: now, EndedAt: now.Add(10 * time.Minute)},
		},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	model := result.(Model)

	if model.taskMode {
		t.Error("Expected taskMode to be false")
	}
	if !model.categoryMode {
		t.Error("Expected categoryMode to be true for uncategorised task")
	}
	if model.pendingTask != "emails" {
		t.Errorf("Expected pendingTask 'emails', got %q", model.pendingTask)
	}
}

func TestResetIntervalResetsTimer(t *testing.T) {
	now := time.Now()
	m := Model{
		workDuration:      50 * time.Minute,
		restDuration:      10 * time.Minute,
		remaining:         30 * time.Minute, // 20 mins elapsed
		isRest:            false,
		totalWorked:       0,
		tasks:             []Task{},
		intervalStartedAt: now.Add(-20 * time.Minute),
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: -1, Text: "r"})
	model := result.(Model)

	if model.remaining != 50*time.Minute {
		t.Errorf("Expected remaining reset to 50m, got %v", model.remaining)
	}
	if model.totalWorked != 0 {
		t.Error("Expected totalWorked to remain 0 (elapsed time discarded)")
	}
}

func TestResetIntervalRemovesCurrentIntervalTasks(t *testing.T) {
	intervalStart := time.Now()
	oldTask := Task{
		Name:      "old task",
		StartedAt: intervalStart.Add(-60 * time.Minute), // from a previous interval
		EndedAt:   intervalStart.Add(-10 * time.Minute),
	}
	currentTask := Task{
		Name:      "current task",
		StartedAt: intervalStart.Add(5 * time.Minute), // started during this interval
	}

	m := Model{
		workDuration:      50 * time.Minute,
		restDuration:      10 * time.Minute,
		remaining:         30 * time.Minute,
		isRest:            false,
		tasks:             []Task{oldTask, currentTask},
		intervalStartedAt: intervalStart,
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: -1, Text: "r"})
	model := result.(Model)

	if len(model.tasks) != 1 {
		t.Fatalf("Expected 1 task after reset, got %d", len(model.tasks))
	}
	if model.tasks[0].Name != "old task" {
		t.Errorf("Expected preserved task to be 'old task', got %q", model.tasks[0].Name)
	}
}

func TestResetIntervalIgnoredDuringRest(t *testing.T) {
	m := Model{
		workDuration: 50 * time.Minute,
		restDuration: 10 * time.Minute,
		remaining:    5 * time.Minute,
		isRest:       true,
		tasks:        []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: -1, Text: "r"})
	model := result.(Model)

	if model.remaining != 5*time.Minute {
		t.Errorf("Expected remaining unchanged at 5m during rest, got %v", model.remaining)
	}
}

func TestResetIntervalIgnoredDuringLunch(t *testing.T) {
	m := Model{
		workDuration:  50 * time.Minute,
		restDuration:  10 * time.Minute,
		lunchDuration: 30 * time.Minute,
		remaining:     15 * time.Minute,
		isRest:        true,
		isLunch:       true,
		tasks:         []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: -1, Text: "r"})
	model := result.(Model)

	if model.remaining != 15*time.Minute {
		t.Errorf("Expected remaining unchanged at 15m during lunch, got %v", model.remaining)
	}
}

func TestResetIntervalClearsInputModes(t *testing.T) {
	now := time.Now()
	m := Model{
		workDuration:      50 * time.Minute,
		restDuration:      10 * time.Minute,
		remaining:         30 * time.Minute,
		isRest:            false,
		tasks:             []Task{},
		intervalStartedAt: now,
	}

	// Test that resetInterval clears all modes directly
	m.namingMode = true
	m.nameInput = "test"
	m.taskMode = true
	m.taskInput = "test"
	m.categoryMode = true
	m.pendingTask = "test"
	m.recentTasks = []string{"a", "b"}

	result := m.resetInterval()

	if result.namingMode {
		t.Error("Expected namingMode to be cleared")
	}
	if result.nameInput != "" {
		t.Error("Expected nameInput to be cleared")
	}
	if result.taskMode {
		t.Error("Expected taskMode to be cleared")
	}
	if result.taskInput != "" {
		t.Error("Expected taskInput to be cleared")
	}
	if result.categoryMode {
		t.Error("Expected categoryMode to be cleared")
	}
	if result.pendingTask != "" {
		t.Error("Expected pendingTask to be cleared")
	}
	if result.recentTasks != nil {
		t.Error("Expected recentTasks to be cleared")
	}
}

func TestResetIntervalPreservesTotalWorkedFromPriorIntervals(t *testing.T) {
	now := time.Now()
	m := Model{
		workDuration:      50 * time.Minute,
		restDuration:      10 * time.Minute,
		remaining:         30 * time.Minute,
		isRest:            false,
		totalWorked:       50 * time.Minute, // from a prior completed interval
		tasks:             []Task{},
		intervalStartedAt: now.Add(-20 * time.Minute),
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: -1, Text: "r"})
	model := result.(Model)

	if model.totalWorked != 50*time.Minute {
		t.Errorf("Expected totalWorked to remain 50m (prior interval preserved), got %v", model.totalWorked)
	}
}
