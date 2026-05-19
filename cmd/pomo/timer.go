package main

import (
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

// transitionToBreak transitions from a pomodoro to a break (short or long).
func (m Model) transitionToBreak(elapsed time.Duration, now time.Time) Model {
	m.totalWorked += elapsed
	m.isRest = true
	m.remaining = m.currentBreakDuration()
	m.endActiveTask(now)
	return m
}

// transitionToPomodoro transitions from a break to a fresh pomodoro.
func (m Model) transitionToPomodoro(elapsed time.Duration, now time.Time) Model {
	m.totalRested += elapsed
	m.pomodorosCompleted++
	m.isRest = false
	m.remaining = m.pomodoroDuration
	// Continue the last task (if any) into this new pomodoro
	if len(m.tasks) > 0 {
		last := m.tasks[len(m.tasks)-1]
		m.tasks = append(m.tasks, Task{Name: last.Name, StartedAt: now})
	}
	return m
}
