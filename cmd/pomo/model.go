package main

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type tickMsg time.Time
type finishedMsg struct{}

// Task represents a named unit of work assigned to a pomodoro.
type Task struct {
	Name      string
	Category  string // empty means uncategorised
	StartedAt time.Time
	EndedAt   time.Time // zero value means still active
}

type Model struct {
	pomodoroDuration    time.Duration
	shortBreakDuration  time.Duration
	longBreakDuration   time.Duration
	pomodorosPerCycle   int
	remaining           time.Duration
	isRest              bool
	paused              bool
	pomodorosCompleted  int
	totalWorked         time.Duration
	totalRested         time.Duration
	progress            *progress.Model
	quitting            bool
	currentPomodoroName string
	namingMode          bool
	nameInput           string
	tasks               []Task
	taskMode            bool
	taskInput           string
	recentTasks         []string
	categoryMode        bool
	pendingTask         string
	categories          []string
	waitingForWorkStart bool
	restFinishedAt      time.Time
	autoStartWork       bool
	startedAt           time.Time
	pomodoroStartedAt   time.Time
	width               int // terminal width, updated via WindowSizeMsg
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

	if m.categoryMode {
		if kp, ok := msg.(tea.KeyPressMsg); ok {
			return m.handleCategoryInput(kp)
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.waitingForWorkStart {
			now := time.Now()
			waited := now.Sub(m.restFinishedAt)
			if waited < 0 {
				waited = 0
			}
			breakDuration := m.phaseDuration()
			m.waitingForWorkStart = false
			m.restFinishedAt = time.Time{}
			m = m.transitionToPomodoro(breakDuration+waited, now)
			// Don't return tickCmd() here — the tick loop is already running
			// from when waitingForWorkStart was set. Returning another tickCmd
			// would create a duplicate tick chain, doubling the countdown speed.
			return m, nil
		}
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
				m.nameInput = m.currentPomodoroName // Pre-fill with current name
			}
			return m, nil
		case "a":
			if !m.isRest {
				m.recentTasks = recentTaskNames(m.tasks)
				m.taskMode = true
				m.taskInput = ""
			}
			return m, nil
		case "r":
			if !m.isRest {
				m = m.resetPomodoro()
				return m, nil
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
			go sendNotification(wasRest, m.currentPomodoroName)
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		barW := viewBarWidth(msg.Width)
		m.progress.SetWidth(barW)
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
			breakDuration := m.phaseDuration()
			if m.autoStartWork {
				m = m.transitionToPomodoro(breakDuration, now)
			} else {
				m.waitingForWorkStart = true
				m.restFinishedAt = now
				go sendNotification(wasRest, m.currentPomodoroName)
				return m, tickCmd()
			}
		} else {
			m = m.transitionToBreak(m.pomodoroDuration, now)
		}
		go sendNotification(wasRest, m.currentPomodoroName)
		return m, tickCmd()
	}
	return m, nil
}

func (m Model) View() tea.View {
	var s strings.Builder

	// ── Special input modes (minimal UI, unchanged behaviour) ─────────
	if m.namingMode {
		s.WriteString(fmt.Sprintf("Rename pomodoro: %s_\n", m.nameInput))
		s.WriteString("\n[enter] confirm   [esc] cancel\n")
		return tea.NewView(s.String())
	}

	if m.waitingForWorkStart {
		s.WriteString("☕ Break Timer\n\n")
		s.WriteString("Break is over!\n\n")
		s.WriteString("Press any key to begin work\n")
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

	if m.categoryMode {
		s.WriteString(fmt.Sprintf("New task: %s\n\n", m.pendingTask))
		s.WriteString("Category:\n")
		for i, cat := range m.categories {
			s.WriteString(fmt.Sprintf("[%d] %-20s", i+1, cat))
			if (i+1)%3 == 0 {
				s.WriteString("\n")
			}
		}
		if len(m.categories)%3 != 0 {
			s.WriteString("\n")
		}
		s.WriteString("[0] none\n\n")
		s.WriteString("[esc] cancel\n")
		return tea.NewView(s.String())
	}

	// ── Layout constants ───────────────────────────────────────────────
	const maxUIWidth = 100
	width := m.width
	if width == 0 || width > maxUIWidth {
		width = maxUIWidth
	}

	// ── Phase metadata ─────────────────────────────────────────────────
	phaseDur := m.phaseDuration()
	var phase, emoji, sessionName string
	switch {
	case m.isRest:
		phase, emoji, sessionName = "rest", "☕", "Break Timer"
	default:
		phase, emoji = "work", "🍅"
		sessionName = m.currentPomodoroName
		if sessionName == "" {
			sessionName = generatePomodoroName(m.pomodorosCompleted+1, time.Now())
		}
	}

	// ── 1 & 2. BREADCRUMB + HEADER with right-aligned badge ───────────
	crumbs := dimSt.Render("pomo › " + strings.ToLower(sessionName) + " › " + phase)
	var badgeText string
	if m.isRest {
		badgeText = fmt.Sprintf("%.0f MIN BREAK", phaseDur.Minutes())
	} else {
		badgeText = fmt.Sprintf("%.0f MIN FOCUS", m.pomodoroDuration.Minutes())
	}
	badge := badgeSt.Render(badgeText)
	badgeLines := strings.Split(badge, "\n")
	badgeW := lipgloss.Width(badge)

	headerLeftLines := []string{crumbs, emoji + " " + sessionName}
	for len(headerLeftLines) < len(badgeLines) {
		headerLeftLines = append(headerLeftLines, "")
	}
	leftW := width - badgeW
	if leftW < 10 {
		leftW = 10
	}
	for i, bl := range badgeLines {
		ll := ""
		if i < len(headerLeftLines) {
			ll = headerLeftLines[i]
		}
		pad := leftW - lipgloss.Width(ll)
		if pad < 0 {
			pad = 0
		}
		s.WriteString(ll + strings.Repeat(" ", pad) + bl + "\n")
	}

	// ── 3. CLOCK (3-row big digits) ────────────────────────────────────
	elapsed := phaseDur - m.remaining
	if elapsed < 0 {
		elapsed = 0
	}
	pct := 0.0
	if phaseDur > 0 {
		pct = float64(elapsed) / float64(phaseDur) * 100
	}
	clockRows := bigClockRows(formatTime(m.remaining))
	pctLabel := dimSt.Render(fmt.Sprintf("  %.0f%% elapsed", pct))
	for i, row := range clockRows {
		styled := clockSt.Render(row)
		if i == len(clockRows)-1 {
			pausedSuffix := ""
			if m.paused {
				pausedSuffix = "  " + pauseSt.Render("PAUSED")
			}
			s.WriteString(styled + pctLabel + pausedSuffix + "\n")
		} else {
			s.WriteString(styled + "\n")
		}
	}

	// ── 4. PROGRESS BAR + MINUTE MARKERS ─────────────────────────────
	barWidth := viewBarWidth(width)
	progressPct := 0.0
	if phaseDur > 0 {
		progressPct = float64(elapsed) / float64(phaseDur)
	}
	s.WriteString("  " + m.progress.ViewAs(progressPct) + "\n")
	s.WriteString("  " + minuteMarkers(elapsed, phaseDur, barWidth) + "\n")

	// ── 5. TASK BOX ───────────────────────────────────────────────────
	s.WriteByte('\n')
	if !m.isRest {
		// Width() in lipgloss v2 = outer rendered width (borders + padding + content).
		// wrapAt = Width - horizontalBorder(2) - hPad(2) = Width - 4.
		// Target outer = width-4, so Width = width-4 and wrapAt = width-8.
		// Use NBSP in taskHint so the word wrapper treats it as one unbreakable token.
		styleW := width - 4
		textAreaW := styleW - 4 // wrapAt: Width - border(2) - padding(2)
		if textAreaW < 10 {
			textAreaW = 10
			styleW = textAreaW + 4
		}
		var taskLeft string
		if t := m.activeTask(); t != nil {
			if t.Category != "" {
				taskLeft = "› " + t.Name + "  " + dimSt.Render("["+t.Category+"]")
			} else {
				taskLeft = "› " + t.Name
			}
		} else {
			taskLeft = taskDimSt.Render("› no task set")
		}
		taskHint := dimSt.Render("[a] add task")
		gap := textAreaW - lipgloss.Width(taskLeft) - lipgloss.Width(taskHint)
		if gap < 1 {
			gap = 1
		}
		inner := taskLeft + strings.Repeat(" ", gap) + taskHint
		s.WriteString(taskBoxSt.Width(styleW).Render(inner) + "\n")
	}

	// ── 6. POMODORO DOTS ──────────────────────────────────────────────
	s.WriteByte('\n')
	s.WriteString("  " + pomodoroDots(m) + "\n")

	// ── 7. DIVIDER ────────────────────────────────────────────────────
	s.WriteByte('\n')
	divW := width - 4
	if divW < 20 {
		divW = 20
	}
	s.WriteString("  " + dimSt.Render(strings.Repeat("─", divW)) + "\n\n")

	// ── 8. KEYBINDING PANEL ───────────────────────────────────────────
	s.WriteString(renderKeybindings(m))

	// ── 9. STATUS BAR ─────────────────────────────────────────────────
	s.WriteByte('\n')
	s.WriteString(renderStatusBar(m) + "\n")

	return tea.NewView(s.String())
}

func formatTime(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
