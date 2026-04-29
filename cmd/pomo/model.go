package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
)

type tickMsg time.Time
type finishedMsg struct{}

// Task represents a named unit of work assigned to an interval.
type Task struct {
	Name      string
	StartedAt time.Time
	EndedAt   time.Time // zero value means still active
}

type Model struct {
	workDuration        time.Duration
	restDuration        time.Duration
	intervalDuration    time.Duration
	remaining           time.Duration
	isRest              bool
	paused              bool
	intervalsCompleted  int
	totalWorked         time.Duration
	totalRested         time.Duration
	progress            *progress.Model
	quitting            bool
	currentIntervalName string
	namingMode          bool
	nameInput           string
	tasks               []Task
	taskMode            bool
	taskInput           string
	startedAt           time.Time
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) phaseDuration() time.Duration {
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

// activeTask returns a pointer to the currently active (not yet ended) task, or nil.
func (m *Model) activeTask() *Task {
	for i := len(m.tasks) - 1; i >= 0; i-- {
		if m.tasks[i].EndedAt.IsZero() {
			return &m.tasks[i]
		}
	}
	return nil
}

// endActiveTask closes the active task with the given end time.
func (m *Model) endActiveTask(at time.Time) {
	for i := len(m.tasks) - 1; i >= 0; i-- {
		if m.tasks[i].EndedAt.IsZero() {
			m.tasks[i].EndedAt = at
			return
		}
	}
}

// startTask ends any active task and starts a new one.
func (m *Model) startTask(name string, at time.Time) {
	m.endActiveTask(at)
	m.tasks = append(m.tasks, Task{Name: name, StartedAt: at})
}

// transitionToRest transitions from work to rest phase.
func (m Model) transitionToRest(elapsed time.Duration, now time.Time) Model {
	m.totalWorked += elapsed
	m.isRest = true
	m.remaining = m.restDuration
	m.endActiveTask(now)
	return m
}

// transitionToWork transitions from rest to work phase.
func (m Model) transitionToWork(elapsed time.Duration, now time.Time) Model {
	m.totalRested += elapsed
	m.intervalsCompleted++
	m.isRest = false
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

// handleNamingInput processes keyboard input while in naming mode (interval rename).
func (m Model) handleNamingInput(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEnter:
		m.currentIntervalName = m.nameInput
		m.namingMode = false
		m.nameInput = ""
	case tea.KeyEscape:
		m.namingMode = false
		m.nameInput = ""
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.nameInput) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.nameInput)
			m.nameInput = m.nameInput[:len(m.nameInput)-size]
		}
	default:
		if msg.Text != "" {
			m.nameInput += msg.Text
		}
	}
	return m, nil
}

// handleTaskInput processes keyboard input while in task mode.
func (m Model) handleTaskInput(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEnter:
		if strings.TrimSpace(m.taskInput) != "" {
			m.startTask(strings.TrimSpace(m.taskInput), time.Now())
		}
		m.taskMode = false
		m.taskInput = ""
	case tea.KeyEscape:
		m.taskMode = false
		m.taskInput = ""
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.taskInput) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.taskInput)
			m.taskInput = m.taskInput[:len(m.taskInput)-size]
		}
	default:
		if msg.Text != "" {
			m.taskInput += msg.Text
		}
	}
	return m, nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Intercept keyboard input when in naming mode, but let other messages
	// (e.g. tickMsg, WindowSizeMsg) fall through so the tick loop keeps running.
	if m.namingMode {
		if kp, ok := msg.(tea.KeyPressMsg); ok {
			return m.handleNamingInput(kp)
		}
	}

	// Intercept keyboard input when in task mode, but let other messages
	// (e.g. tickMsg, WindowSizeMsg) fall through so the tick loop keeps running.
	if m.taskMode {
		if kp, ok := msg.(tea.KeyPressMsg); ok {
			return m.handleTaskInput(kp)
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			elapsed := m.phaseDuration() - m.remaining
			if m.isRest {
				m.totalRested += elapsed
			} else {
				m.totalWorked += elapsed
			}
			m.endActiveTask(time.Now())
			m.quitting = true
			return m, tea.Quit
		case "space":
			m.paused = !m.paused
			return m, nil
		case "n":
			if !m.isRest {
				m.namingMode = true
				m.nameInput = m.currentIntervalName // Pre-fill with current name
			}
			return m, nil
		case "a":
			if !m.isRest {
				m.taskMode = true
				m.taskInput = ""
			}
			return m, nil
		case "s":
			now := time.Now()
			elapsed := m.phaseDuration() - m.remaining
			if m.isRest {
				m = m.transitionToWork(elapsed, now)
			} else {
				m = m.transitionToRest(elapsed, now)
			}
			go sendNotification(m.isRest, m.currentIntervalName)
			return m, nil
		}
	case tea.WindowSizeMsg:
		const padding = 4
		const timeDisplayWidth = 20
		const minBarWidth = 20
		const maxBarWidth = 80

		width := msg.Width - padding - timeDisplayWidth
		if width > maxBarWidth {
			width = maxBarWidth
		}
		if width < minBarWidth {
			width = minBarWidth
		}
		m.progress.SetWidth(width)
		return m, nil
	case tickMsg:
		if !m.paused && m.remaining > 0 {
			m.remaining -= time.Second
			if m.remaining <= 0 {
				return m, func() tea.Msg { return finishedMsg{} }
			}
		}
		return m, tickCmd()
	case finishedMsg:
		now := time.Now()
		wasRest := m.isRest
		if m.isRest {
			m = m.transitionToWork(m.restDuration, now)
		} else {
			m = m.transitionToRest(m.workDuration, now)
		}
		go sendNotification(wasRest, m.currentIntervalName)
		return m, tickCmd()
	}
	return m, nil
}

func (m Model) View() tea.View {
	var s strings.Builder

	if m.namingMode {
		s.WriteString(fmt.Sprintf("Rename interval: %s_\n", m.nameInput))
		s.WriteString("\n[enter] confirm   [esc] cancel\n")
		return tea.NewView(s.String())
	}

	if m.taskMode {
		s.WriteString(fmt.Sprintf("Add task: %s_\n", m.taskInput))
		s.WriteString("\n[enter] confirm   [esc] cancel\n")
		return tea.NewView(s.String())
	}

	emoji := "🍅"
	title := "Pomodoro Timer"

	if m.isRest {
		emoji = "☕"
		title = "Break Timer"
	}

	phaseDur := m.phaseDuration()
	intervalNum := m.intervalsCompleted + 1
	currentName := m.currentIntervalName
	if currentName == "" {
		currentName = generateIntervalName(intervalNum, time.Now())
	}

	if phaseDur < time.Minute {
		s.WriteString(fmt.Sprintf("%s %s: %s (%d seconds)\n", emoji, title, currentName, int(phaseDur.Seconds())))
	} else {
		s.WriteString(fmt.Sprintf("%s %s: %s (%.0fm)\n", emoji, title, currentName, phaseDur.Minutes()))
	}
	s.WriteString("\n")

	elapsed := phaseDur - m.remaining
	percentage := float64(elapsed) / float64(phaseDur) * 100

	if m.remaining <= 0 {
		if m.isRest {
			s.WriteString("🎉 Break completed!\n")
		} else {
			s.WriteString("🎉 Pomodoro completed!\n")
		}
	} else {
		timeStr := formatTime(m.remaining)
		progressPercent := percentage / 100.0
		progressBar := m.progress.ViewAs(progressPercent)

		if m.paused {
			s.WriteString(fmt.Sprintf("⏰ %s %s (PAUSED)\n", timeStr, progressBar))
		} else {
			s.WriteString(fmt.Sprintf("⏰ %s %s\n", timeStr, progressBar))
		}

		// Show active task
		if !m.isRest {
			if t := m.activeTask(); t != nil {
				s.WriteString(fmt.Sprintf("   Task: %s\n", t.Name))
			} else {
				s.WriteString("   Task: (none)\n")
			}
		}

		s.WriteString("\n")
		if m.isRest {
			s.WriteString("Press [s] to skip interval, [space] to pause/resume, [q] to quit\n")
		} else {
			s.WriteString("[n] rename interval  [s] skip interval  [a] add task\n")
			s.WriteString("[space] pause/resume  [q] quit\n")
		}
	}

	return tea.NewView(s.String())
}

func formatTime(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
