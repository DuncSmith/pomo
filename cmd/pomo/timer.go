package main

import (
	"fmt"
	"time"
)

func (m Model) phaseDuration() time.Duration {
	if m.isRest {
		return m.currentBreakDuration()
	}
	return m.pomodoroDuration
}

// currentBreakDuration returns the long break duration if the just-completed
// pomodoro is a multiple of pomodorosPerCycle, otherwise the short break.
func (m Model) currentBreakDuration() time.Duration {
	completed := m.pomodorosCompleted + 1
	if m.pomodorosPerCycle > 0 && completed%m.pomodorosPerCycle == 0 {
		return m.longBreakDuration
	}
	return m.shortBreakDuration
}

// generatePomodoroName returns a time-of-day prefixed name for the nth pomodoro (1-indexed).
func generatePomodoroName(n int, now time.Time) string {
	hour := now.Hour()
	var prefix string
	switch {
	case hour < 12:
		prefix = "Morning"
	case hour < 17:
		prefix = "Afternoon"
	case hour < 21:
		prefix = "Evening"
	default:
		prefix = "Night"
	}
	return fmt.Sprintf("%s #%d", prefix, n)
}

// resetPomodoro restarts the current pomodoro from scratch,
// discarding all elapsed time and tasks created during this pomodoro.
func (m Model) resetPomodoro() Model {
	m.remaining = m.pomodoroDuration
	// Remove tasks that were started during this pomodoro
	kept := m.tasks[:0:0]
	for _, t := range m.tasks {
		if t.StartedAt.Before(m.pomodoroStartedAt) {
			kept = append(kept, t)
		}
	}
	m.tasks = kept
	// Clear any active input modes
	m.namingMode = false
	m.nameInput = ""
	m.taskMode = false
	m.taskInput = ""
	m.recentTasks = nil
	m.categoryMode = false
	m.pendingTask = ""
	return m
}

// transitionToBreak transitions from a pomodoro to a break (short or long).
func (m Model) transitionToBreak(elapsed time.Duration, now time.Time) Model {
	m.totalWorked += elapsed
	m.isRest = true
	m.remaining = m.currentBreakDuration()
	m.waitingForWorkStart = false
	m.restFinishedAt = time.Time{}
	m.endActiveTask(now)
	m.categoryMode = false
	m.pendingTask = ""
	return m
}

// transitionToPomodoro transitions from a break to a fresh pomodoro.
func (m Model) transitionToPomodoro(elapsed time.Duration, now time.Time) Model {
	m.totalRested += elapsed
	m.pomodorosCompleted++
	m.isRest = false
	m.waitingForWorkStart = false
	m.restFinishedAt = time.Time{}
	m.remaining = m.pomodoroDuration
	// Generate a fresh pomodoro name — no carry-over
	m.currentPomodoroName = generatePomodoroName(m.pomodorosCompleted+1, now)
	m.pomodoroStartedAt = now
	// Continue the last task (if any) into this new pomodoro, carrying its category
	if len(m.tasks) > 0 {
		last := m.tasks[len(m.tasks)-1]
		m.tasks = append(m.tasks, Task{Name: last.Name, Category: last.Category, StartedAt: now})
	}
	return m
}
