package main

import (
	"fmt"
	"strings"
	"time"

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
	recentTasks         []string
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
				m.recentTasks = recentTaskNames(m.tasks)
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
		if len(m.recentTasks) > 0 && m.taskInput == "" {
			s.WriteString("Recent tasks:\n")
			for i, name := range m.recentTasks {
				s.WriteString(fmt.Sprintf("  %d  %s\n", i+1, name))
			}
			s.WriteString("\n[1-9] resume   [type] new task   [esc] cancel\n")
		} else {
			s.WriteString(fmt.Sprintf("New task: %s_\n", m.taskInput))
			s.WriteString("\n[enter] confirm   [esc] cancel\n")
		}
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
