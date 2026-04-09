package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time
type finishedMsg struct{}

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
	progress           progress.Model
	quitting           bool
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// Track time spent in current phase before quitting
			elapsed := m.phaseDuration() - m.remaining
			if m.isRest {
				m.totalRested += elapsed
			} else {
				m.totalWorked += elapsed
			}
			m.quitting = true
			return m, tea.Quit
		case " ":
			m.paused = !m.paused
			return m, nil
		}
	case tea.WindowSizeMsg:
		const padding = 4
		const maxWidth = 80
		m.progress.Width = msg.Width - padding - 20
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		if m.progress.Width < 20 {
			m.progress.Width = 20
		}
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
		m.sendNotification()
		if m.isRest {
			// Rest finished: complete the interval, start new work phase
			m.totalRested += m.restDuration
			m.intervalsCompleted++
			m.isRest = false
			m.remaining = m.workDuration
		} else {
			// Work finished: switch to rest phase
			m.totalWorked += m.workDuration
			m.isRest = true
			m.remaining = m.restDuration
		}
		return m, tickCmd()
	}
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder

	emoji := "🍅"
	title := "Pomodoro Timer"
	if m.isRest {
		emoji = "☕"
		title = "Break Timer"
	}

	phaseDur := m.phaseDuration()
	intervalNum := m.intervalsCompleted + 1

	if phaseDur < time.Minute {
		s.WriteString(fmt.Sprintf("%s %s: %d seconds (interval #%d)\n", emoji, title, int(phaseDur.Seconds()), intervalNum))
	} else {
		s.WriteString(fmt.Sprintf("%s %s: %.0fm (interval #%d)\n", emoji, title, phaseDur.Minutes(), intervalNum))
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
			s.WriteString(fmt.Sprintf("⏸️  %s %s (PAUSED)\n", timeStr, progressBar))
		} else {
			s.WriteString(fmt.Sprintf("⏰ %s %s\n", timeStr, progressBar))
		}
		s.WriteString("\n")
		s.WriteString("Press [space] to pause/resume, [q] to quit\n")
	}

	return s.String()
}

func main() {
	model, err := parseArgs()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(*model)
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

func parseArgs() (*Model, error) {
	args := os.Args[1:]

	// Check for help
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			showHelp()
			os.Exit(0)
		}
	}

	var workDuration time.Duration
	var intervalDuration time.Duration
	var hasWork, hasInterval bool

	// Parse args: positional work duration and --interval flag
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
			i++ // skip value
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

	return &Model{
		workDuration:     workDuration,
		restDuration:     restDuration,
		intervalDuration: intervalDuration,
		remaining:        workDuration,
		isRest:           false,
		progress:         progress.New(progress.WithDefaultGradient()),
	}, nil
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
	var unit = time.Minute // the default unit is minutes
	
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

func (m Model) sendNotification() {
	var title, message string
	if m.isRest {
		title = "☕ Break Timer"
		message = "Your break is complete!"
	} else {
		title = "🍅 Pomodoro Timer"
		message = "Your Pomodoro session is complete!"
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
