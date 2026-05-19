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
		{name: "zero duration", input: 0, expected: "00:00"},
		{name: "seconds only", input: 30 * time.Second, expected: "00:30"},
		{name: "minutes only", input: 5 * time.Minute, expected: "05:00"},
		{name: "minutes and seconds", input: 3*time.Minute + 45*time.Second, expected: "03:45"},
		{name: "double digit minutes", input: 25*time.Minute + 10*time.Second, expected: "25:10"},
		{name: "hours converted to minutes", input: time.Hour + 30*time.Minute + 15*time.Second, expected: "90:15"},
		{name: "single digit second", input: 2*time.Minute + 5*time.Second, expected: "02:05"},
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
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
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
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		longBreakDuration:  20 * time.Minute,
		pomodorosPerCycle:  4,
		remaining:          0,
		isRest:             false,
		tasks:              []Task{},
	}

	result, cmd := m.Update(finishedMsg{})
	model := result.(Model)

	if !model.isRest {
		t.Error("Expected model to switch to rest phase")
	}
	if model.remaining != 10*time.Minute {
		t.Errorf("Expected remaining to be 10m, got %v", model.remaining)
	}
	if model.pomodorosCompleted != 0 {
		t.Errorf("Expected 0 completed pomodoros, got %d", model.pomodorosCompleted)
	}
	if model.totalWorked != 50*time.Minute {
		t.Errorf("Expected 50m total worked, got %v", model.totalWorked)
	}
	if cmd == nil {
		t.Error("Expected tickCmd to be returned")
	}
}

func TestPhaseTransitionRestToWorkAutoStarts(t *testing.T) {
	m := Model{
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		longBreakDuration:  20 * time.Minute,
		pomodorosPerCycle:  4,
		remaining:          0,
		isRest:             true,
		pomodorosCompleted: 0,
		totalWorked:        50 * time.Minute,
		tasks:              []Task{},
	}

	result, cmd := m.Update(finishedMsg{})
	model := result.(Model)

	if model.isRest {
		t.Error("Expected model to switch to work phase automatically")
	}
	if model.remaining != 50*time.Minute {
		t.Errorf("Expected remaining to be 50m, got %v", model.remaining)
	}
	if model.pomodorosCompleted != 1 {
		t.Errorf("Expected 1 completed pomodoro, got %d", model.pomodorosCompleted)
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
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		longBreakDuration:  20 * time.Minute,
		pomodorosPerCycle:  4,
		remaining:          30 * time.Minute,
		isRest:             false,
		tasks:              []Task{},
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
	if m.tasks[0].EndedAt.IsZero() {
		t.Error("Expected first task to be ended")
	}
	if !m.tasks[0].EndedAt.Equal(later) {
		t.Errorf("Expected first task EndedAt %v, got %v", later, m.tasks[0].EndedAt)
	}
	if !m.tasks[1].EndedAt.IsZero() {
		t.Error("Expected second task to still be active")
	}
}

func TestActiveTask(t *testing.T) {
	m := Model{tasks: []Task{}}

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

	m.endActiveTask(now.Add(5 * time.Minute))
	if m.activeTask() != nil {
		t.Error("Expected no active task after ending it")
	}
}

func TestTransitionToBreakEndsActiveTask(t *testing.T) {
	start := time.Now()
	m := Model{
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		tasks:              []Task{},
	}
	m.startTask("Active task", start)

	m = m.transitionToBreak(50*time.Minute, time.Now())

	if m.activeTask() != nil {
		t.Error("Expected no active task after transitioning to rest")
	}
	if m.tasks[0].EndedAt.IsZero() {
		t.Error("Expected task EndedAt to be set after transitioning to rest")
	}
}

func TestTransitionToPomodoroContinuesLastTask(t *testing.T) {
	start := time.Now()
	m := Model{
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		pomodorosCompleted: 0,
		isRest:             true,
		tasks:              []Task{},
	}
	m.tasks = append(m.tasks, Task{
		Name:      "Carry forward task",
		StartedAt: start,
		EndedAt:   start.Add(50 * time.Minute),
	})

	m = m.transitionToPomodoro(10*time.Minute, time.Now())

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

func TestTransitionToPomodoroNoTasksNoContinuation(t *testing.T) {
	m := Model{
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		pomodorosCompleted: 0,
		isRest:             true,
		tasks:              []Task{},
	}

	m = m.transitionToPomodoro(10*time.Minute, time.Now())

	if len(m.tasks) != 0 {
		t.Errorf("Expected 0 tasks when no previous task, got %d", len(m.tasks))
	}
}

func TestTransitionToBreakUsesLongBreakAfterPerCycle(t *testing.T) {
	cases := []struct {
		name             string
		completedBefore  int
		expectedDuration time.Duration
	}{
		{name: "1st pomodoro -> short", completedBefore: 0, expectedDuration: 5 * time.Minute},
		{name: "2nd pomodoro -> short", completedBefore: 1, expectedDuration: 5 * time.Minute},
		{name: "3rd pomodoro -> short", completedBefore: 2, expectedDuration: 5 * time.Minute},
		{name: "4th pomodoro -> long", completedBefore: 3, expectedDuration: 15 * time.Minute},
		{name: "5th pomodoro -> short", completedBefore: 4, expectedDuration: 5 * time.Minute},
		{name: "8th pomodoro -> long", completedBefore: 7, expectedDuration: 15 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{
				pomodoroDuration:   25 * time.Minute,
				shortBreakDuration: 5 * time.Minute,
				longBreakDuration:  15 * time.Minute,
				pomodorosPerCycle:  4,
				pomodorosCompleted: tc.completedBefore,
				tasks:              []Task{},
			}
			m = m.transitionToBreak(25*time.Minute, time.Now())
			if !m.isRest {
				t.Fatal("Expected isRest to be true after transitionToBreak")
			}
			if m.remaining != tc.expectedDuration {
				t.Errorf("Expected remaining %v, got %v", tc.expectedDuration, m.remaining)
			}
		})
	}
}

func TestPhaseDurationLongBreak(t *testing.T) {
	m := Model{
		pomodoroDuration:   25 * time.Minute,
		shortBreakDuration: 5 * time.Minute,
		longBreakDuration:  15 * time.Minute,
		pomodorosPerCycle:  4,
		pomodorosCompleted: 3,
		isRest:             true,
	}
	if m.phaseDuration() != 15*time.Minute {
		t.Errorf("Expected phaseDuration to be 15m (long break), got %v", m.phaseDuration())
	}

	m.pomodorosCompleted = 0
	if m.phaseDuration() != 5*time.Minute {
		t.Errorf("Expected phaseDuration to be 5m (short break), got %v", m.phaseDuration())
	}

	m.isRest = false
	if m.phaseDuration() != 25*time.Minute {
		t.Errorf("Expected phaseDuration to be 25m (pomodoro), got %v", m.phaseDuration())
	}
}

func TestTaskKeyOpensTaskMode(t *testing.T) {
	m := Model{
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		remaining:          40 * time.Minute,
		isRest:             false,
		tasks:              []Task{},
	}

	result, _ := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	model := result.(Model)

	if !model.taskMode {
		t.Error("Expected taskMode to be true after pressing 'a'")
	}
}

func TestTaskKeyDisabledDuringRest(t *testing.T) {
	m := Model{
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		remaining:          8 * time.Minute,
		isRest:             true,
		tasks:              []Task{},
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
	m.tasks = append(m.tasks, Task{Name: "Old task", StartedAt: now.Add(-10 * time.Minute)})

	result, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model := result.(Model)

	if model.taskMode {
		t.Error("Expected taskMode to be false after confirming")
	}
	if model.taskInput != "" {
		t.Error("Expected taskInput to be cleared after confirming")
	}
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
		{Name: "task2", StartedAt: now.Add(10 * time.Minute)},
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
		taskMode:    true,
		taskInput:   "",
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
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		remaining:          40 * time.Minute,
		isRest:             false,
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
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		longBreakDuration:  20 * time.Minute,
		pomodorosPerCycle:  4,
		remaining:          30 * time.Minute,
		isRest:             false,
		tasks:              []Task{},
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
		pomodoroDuration:   50 * time.Minute,
		shortBreakDuration: 10 * time.Minute,
		longBreakDuration:  20 * time.Minute,
		pomodorosPerCycle:  4,
		remaining:          7 * time.Minute,
		isRest:             true,
		tasks:              []Task{},
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
