package main

import (
	"fmt"
	"time"
)

func (m Model) phaseDuration() time.Duration {
	if m.isLunch {
		return m.lunchDuration
	}
	if m.isRest {
		return m.restDuration
	}
	return m.workDuration
}

// generateIntervalName returns a time-of-day prefixed name for the nth work interval (1-indexed).
func generateIntervalName(n int, now time.Time) string {
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

// transitionToRest transitions from work to rest phase.
func (m Model) transitionToRest(elapsed time.Duration, now time.Time) Model {
	m.totalWorked += elapsed
	m.isRest = true
	m.remaining = m.restDuration
	m.endActiveTask(now)
	return m
}

// transitionToLunch transitions from work to a lunch break.
func (m Model) transitionToLunch(elapsed time.Duration, now time.Time) Model {
	m.totalWorked += elapsed
	m.isRest = true
	m.isLunch = true
	m.remaining = m.lunchDuration
	m.endActiveTask(now)
	return m
}

// transitionToWork transitions from rest (or lunch) to work phase.
func (m Model) transitionToWork(elapsed time.Duration, now time.Time) Model {
	m.totalRested += elapsed
	m.intervalsCompleted++
	m.isRest = false
	m.isLunch = false
	m.lunchReady = false
	m.remaining = m.workDuration
	// Generate a fresh interval name — no carry-over
	m.currentIntervalName = generateIntervalName(m.intervalsCompleted+1, now)
	// Continue the last task (if any) into this new interval
	if len(m.tasks) > 0 {
		lastName := m.tasks[len(m.tasks)-1].Name
		m.tasks = append(m.tasks, Task{Name: lastName, StartedAt: now})
	}
	return m
}
