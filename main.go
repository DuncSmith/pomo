package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
)

// Set via -ldflags by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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
	workDuration       time.Duration
	restDuration       time.Duration
	intervalDuration   time.Duration
	remaining          time.Duration
	isRest             bool
	paused             bool
	intervalsCompleted int
	totalWorked        time.Duration
	totalRested        time.Duration
	progress           *progress.Model
	quitting           bool
	currentIntervalName string
	namingMode         bool
	nameInput          string
	tasks              []Task
	taskMode           bool
	taskInput          string
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
func (m Model) transitionToRest(elapsed time.Duration) Model {
	m.totalWorked += elapsed
	m.isRest = true
	m.remaining = m.restDuration

	// End the active task when entering rest
	now := time.Now()
	m.endActiveTask(now)

	return m
}

// transitionToWork transitions from rest to work phase.
func (m Model) transitionToWork() Model {
	m.totalRested += m.restDuration
	m.intervalsCompleted++
	m.isRest = false
	m.remaining = m.workDuration

	// Generate a fresh interval name — no carry-over
	m.currentIntervalName = generateIntervalName(m.intervalsCompleted+1, time.Now())

	// Continue the last task (if any) into this new interval
	now := time.Now()
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
			// Track time spent in current phase before quitting
			elapsed := m.phaseDuration() - m.remaining
			if m.isRest {
				m.totalRested += elapsed
			} else {
				m.totalWorked += elapsed
			}
			// End the active task
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
			// Skip current interval
			if m.isRest {
				elapsed := m.phaseDuration() - m.remaining
				m.totalRested += elapsed
				m = m.transitionToWork()
			} else {
				elapsed := m.phaseDuration() - m.remaining
				m = m.transitionToRest(elapsed)
			}
			sendNotification(m.isRest, m.currentIntervalName)
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
		sendNotification(m.isRest, m.currentIntervalName)

		if m.isRest {
			m = m.transitionToWork()
		} else {
			m = m.transitionToRest(m.workDuration)
		}
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
			s.WriteString("Press [space] to pause/resume, [s] to skip, [q] to quit\n")
		} else {
			s.WriteString("Press [space] to pause/resume, [n] to rename, [a] to add task, [s] to skip, [q] to quit\n")
		}
	}

	return tea.NewView(s.String())
}

func main() {
	result, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	switch result.action {
	case "help":
		showHelp()
		os.Exit(0)
	case "version":
		fmt.Println(result.versionInfo)
		os.Exit(0)
	}

	p := tea.NewProgram(*result.model)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(Model); ok && m.quitting {
		printSummary(m)
	}
}

func printSummary(m Model) {
	fmt.Println("\n📊 Session Summary")
	fmt.Printf("  Intervals completed: %d\n", m.intervalsCompleted)
	fmt.Printf("  Total worked: %s\n", formatDurationHuman(m.totalWorked))
	fmt.Printf("  Total rested: %s\n", formatDurationHuman(m.totalRested))
	fmt.Printf("  Interval: %.0fm work | %.0fm rest\n", m.workDuration.Minutes(), m.restDuration.Minutes())

	// Compute per-task totals from the tasks slice
	taskTotals := make(map[string]time.Duration)
	for _, t := range m.tasks {
		if t.Name == "" {
			continue
		}
		end := t.EndedAt
		if end.IsZero() {
			end = time.Now()
		}
		taskTotals[t.Name] += end.Sub(t.StartedAt)
	}

	if len(taskTotals) > 0 {
		fmt.Println("\n  Tasks:")

		names := make([]string, 0, len(taskTotals))
		for name := range taskTotals {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			fmt.Printf("    - %s: %s\n", name, formatDurationHuman(taskTotals[name]))
		}
	}
}

func formatDurationHuman(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// parseArgsResult represents the result of parsing command-line arguments
type parseArgsResult struct {
	model       *Model
	action      string // "", "help", or "version"
	versionInfo string
}

func parseArgs(args []string) (*parseArgsResult, error) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return &parseArgsResult{action: "help"}, nil
		}
		if arg == "-v" || arg == "--version" {
			info := fmt.Sprintf("pomo %s (commit: %s, built: %s)", version, commit, date)
			return &parseArgsResult{action: "version", versionInfo: info}, nil
		}
	}

	var workDuration time.Duration
	var intervalDuration time.Duration
	var hasWork, hasInterval bool

	for i := 0; i < len(args); i++ {
		if args[i] == "--interval" || args[i] == "-i" {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--interval requires a value")
			}
			d, err := parseDuration(args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid interval duration: %s", args[i+1])
			}
			intervalDuration = d
			hasInterval = true
			i++
		} else {
			if hasWork {
				return nil, fmt.Errorf("unexpected argument: %s", args[i])
			}
			d, err := parseDuration(args[i])
			if err != nil {
				return nil, fmt.Errorf("invalid duration: %s", args[i])
			}
			workDuration = d
			hasWork = true
		}
	}

	if !hasWork {
		workDuration = 50 * time.Minute
	}
	if !hasInterval {
		intervalDuration = 60 * time.Minute
	}

	if workDuration <= 0 {
		return nil, fmt.Errorf("work duration must be greater than 0")
	}
	if intervalDuration <= 0 {
		return nil, fmt.Errorf("interval duration must be greater than 0")
	}
	if workDuration >= intervalDuration {
		return nil, fmt.Errorf("work duration (%v) must be less than interval duration (%v)", workDuration, intervalDuration)
	}

	restDuration := intervalDuration - workDuration

	p := progress.New(progress.WithDefaultBlend())
	model := &Model{
		workDuration:        workDuration,
		restDuration:        restDuration,
		intervalDuration:    intervalDuration,
		remaining:           workDuration,
		isRest:              false,
		progress:            &p,
		currentIntervalName: generateIntervalName(1, time.Now()),
		tasks:               []Task{},
		namingMode:          false,
		nameInput:           "",
		taskMode:            false,
		taskInput:           "",
	}
	return &parseArgsResult{model: model}, nil
}

func showHelp() {
	fmt.Println("Usage: pomo [duration] [--interval duration]")
	fmt.Println()
	fmt.Println("Runs repeating work/rest intervals until you quit.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  pomo          # 50m work, 10m rest (60m interval)")
	fmt.Println("  pomo 25       # 25m work, 35m rest (60m interval)")
	fmt.Println("  pomo 25m      # 25m work, 35m rest (60m interval)")
	fmt.Println("  pomo 30s      # 30s work timer (60m interval)")
	fmt.Println("  pomo 45 -i 90 # 45m work, 45m rest (90m interval)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -i, --interval  Set interval duration (default: 60m)")
	fmt.Println("  -v, --version   Show version information")
	fmt.Println("  -h, --help      Show this help")
	fmt.Println()
	fmt.Println("Default: 50m work, 10m rest (60m interval)")
}

func formatTime(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func createProgressBar(percentage float64, width int) string {
	filled := int(percentage / 100 * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}

func parseDuration(arg string) (time.Duration, error) {
	var unit = time.Minute

	if strings.HasSuffix(arg, "m") {
		arg = strings.TrimSuffix(arg, "m")
		unit = time.Minute
	} else if strings.HasSuffix(arg, "s") {
		arg = strings.TrimSuffix(arg, "s")
		unit = time.Second
	}

	value, err := strconv.Atoi(arg)
	if err != nil {
		return 0, err
	}

	return time.Duration(value) * unit, nil
}

func sendNotification(isRest bool, intervalName string) {
	var title, message string
	if isRest {
		title = "☕ Break Timer"
		message = "Your break is complete!"
	} else {
		title = "🍅 Pomodoro Timer"
		message = fmt.Sprintf("Pomodoro '%s' is complete!", intervalName)
	}

	if runtime.GOOS == "darwin" {
		cmd := exec.Command("terminal-notifier", "-title", title, "-message", message, "-sound", "default")
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Debug: terminal-notifier not found. Install it with: brew install terminal-notifier\n")
		}
	} else {
		cmd := exec.Command("notify-send", title, message)
		cmd.Run()
	}
}
