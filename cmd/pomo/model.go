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

// Task represents a named unit of work assigned to a pomodoro.
type Task struct {
	Name      string
	StartedAt time.Time
	EndedAt   time.Time // zero value means still active
}

type Model struct {
	pomodoroDuration   time.Duration
	shortBreakDuration time.Duration
	longBreakDuration  time.Duration
	pomodorosPerCycle  int
	remaining          time.Duration
	isRest             bool
	paused             bool
	pomodorosCompleted int
	totalWorked        time.Duration
	totalRested        time.Duration
	progress           *progress.Model
	quitting           bool
	tasks              []Task
	taskMode           bool
	taskInput          string
	recentTasks        []string
	startedAt          time.Time
	width              int // terminal width, updated via WindowSizeMsg
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
			wasRest := m.isRest
			if m.isRest {
				m = m.transitionToPomodoro(elapsed, now)
			} else {
				m = m.transitionToBreak(elapsed, now)
			}
			go sendNotification(wasRest)
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.progress.SetWidth(barWidth(msg.Width))
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
			m = m.transitionToPomodoro(m.phaseDuration(), now)
		} else {
			m = m.transitionToBreak(m.pomodoroDuration, now)
		}
		go sendNotification(wasRest)
		return m, tickCmd()
	}
	return m, nil
}

func (m Model) View() tea.View {
	var s strings.Builder

	// Task input mode
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

	// Main view
	var emoji, phase string
	if m.isRest {
		emoji, phase = "☕", "Break"
	} else {
		emoji, phase = "🍅", "Focus"
	}

	phaseDur := m.phaseDuration()
	elapsed := phaseDur - m.remaining
	if elapsed < 0 {
		elapsed = 0
	}
	progressPct := 0.0
	if phaseDur > 0 {
		progressPct = float64(elapsed) / float64(phaseDur)
	}

	// Header
	s.WriteString(fmt.Sprintf("%s %s\n\n", emoji, phase))

	// Countdown
	clockLine := "  " + formatTime(m.remaining)
	if m.paused {
		clockLine += "  " + dimSt.Render("(paused)")
	}
	s.WriteString(clockLine + "\n")

	// Progress bar
	s.WriteString("  " + m.progress.ViewAs(progressPct) + "\n\n")

	// Task line (work phase only)
	if !m.isRest {
		if t := m.activeTask(); t != nil {
			s.WriteString("  › " + t.Name + "\n\n")
		} else {
			s.WriteString("  " + taskDimSt.Render("› no task set") + "\n\n")
		}
	}

	// Key hints
	s.WriteString("  " + keyHints(m.isRest) + "\n")

	return tea.NewView(s.String())
}

func formatTime(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
