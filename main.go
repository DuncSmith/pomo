package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

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
	progress            *progress.Model
	quitting            bool
	currentIntervalName string
	lastCustomName      string
	intervalNames       map[int]string
	workedDurationsByName map[string]time.Duration
	namingMode          bool
	nameInput           string
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

type renameInputMsg string

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.namingMode {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "enter":
				m.currentIntervalName = m.nameInput
				m.namingMode = false
				m.nameInput = ""
			case "esc":
				m.namingMode = false
				m.nameInput = ""
			case "backspace":
				if len(m.nameInput) > 0 {
					m.nameInput = m.nameInput[:len(m.nameInput)-1]
				}
			default:
				m.nameInput += msg.String()
			}
			return m, nil
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
		case "s":
			// Skip current interval
			if m.isRest {
				// Track time spent in current rest phase before skipping
				elapsed := m.phaseDuration() - m.remaining
				m.totalRested += elapsed
				
				// Transition to work phase
				m.intervalsCompleted++
				m.isRest = false
				m.remaining = m.workDuration
				
				// Inherit the last custom name for the new work interval
				m.currentIntervalName = m.lastCustomName
				
				// Send notification
				m.sendNotification()
				
				return m, tickCmd()
			} else {
				// Track time spent in current work phase before skipping
				elapsed := m.phaseDuration() - m.remaining
				m.totalWorked += elapsed
				
				// Determine the name for the completed interval
				nameToStore := m.currentIntervalName
				if nameToStore == "" {
					nameToStore = fmt.Sprintf("Interval #%d", m.intervalsCompleted+1)
				}
				m.intervalNames[m.intervalsCompleted+1] = nameToStore
				m.workedDurationsByName[nameToStore] += elapsed
				
				// Transition to rest phase
				m.isRest = true
				m.remaining = m.restDuration
				
				// Store current custom name as last custom name for inheritance
				if m.currentIntervalName != "" {
					m.lastCustomName = m.currentIntervalName
				}
				
				// Send notification
				m.sendNotification()
				
				return m, tickCmd()
			}
		}
	case tea.WindowSizeMsg:
		const padding = 4
		const maxWidth = 80
		width := msg.Width - padding - 20
		if width > maxWidth {
			width = maxWidth
		}
		if width < 20 {
			width = 20
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
		m.sendNotification()
		if m.isRest {
            // Rest finished: complete the interval, start new work phase
            m.totalRested += m.restDuration
            m.intervalsCompleted++
            m.isRest = false
            m.remaining = m.workDuration

            // Inherit the last custom name for the new work interval
            m.currentIntervalName = m.lastCustomName

        } else {
            // Work finished: switch to rest phase

            // Determine the name to store for the completed interval
            nameToStore := m.currentIntervalName
            if nameToStore == "" {
                nameToStore = fmt.Sprintf("Interval #%d", m.intervalsCompleted+1)
            }
            m.intervalNames[m.intervalsCompleted+1] = nameToStore
            m.workedDurationsByName[nameToStore] += m.workDuration

            m.totalWorked += m.workDuration
            m.isRest = true
            m.remaining = m.restDuration

            // Store current custom name as last custom name for inheritance
            if m.currentIntervalName != "" {
                m.lastCustomName = m.currentIntervalName
            }
        }
        return m, tickCmd()
    }
    return m, nil
}

func (m Model) View() tea.View {
	var s strings.Builder

	if m.namingMode {
		s.WriteString(fmt.Sprintf("Enter name: %s\n", m.nameInput))
		return tea.NewView(s.String())
	}

	emoji := "🍅"
	title := "Pomodoro Timer"
	currentName := m.currentIntervalName

	if m.isRest {
		emoji = "☕"
		title = "Break Timer"
		currentName = ""
	}

	phaseDur := m.phaseDuration()
	intervalNum := m.intervalsCompleted + 1

	if currentName == "" {
		currentName = fmt.Sprintf("Interval #%d", intervalNum)
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
			s.WriteString(fmt.Sprintf("⏸️  %s %s (PAUSED)\n", timeStr, progressBar))
		} else {
			s.WriteString(fmt.Sprintf("⏰ %s %s\n", timeStr, progressBar))
		}
		s.WriteString("\n")
		if m.isRest {
			s.WriteString("Press [space] to pause/resume, [s] to skip, [q] to quit\n")
		} else {
			s.WriteString("Press [space] to pause/resume, [n] to name, [s] to skip, [q] to quit\n")
		}
	}

	return tea.NewView(s.String())
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

	if len(m.workedDurationsByName) > 0 {
		fmt.Println("\n  Worked Durations by Name:")
		for name, duration := range m.workedDurationsByName {
			fmt.Printf("    - %s: %s\n", name, formatDurationHuman(duration))
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

func parseArgs() (*Model, error) {
	args := os.Args[1:]

	// Check for help and version flags
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			showHelp()
			os.Exit(0)
		}
		if arg == "-v" || arg == "--version" {
			fmt.Printf("pomo %s (commit: %s, built: %s)\n", version, commit, date)
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

	p := progress.New(progress.WithDefaultBlend())
	return &Model{
		workDuration:        workDuration,
		restDuration:        restDuration,
		intervalDuration:    intervalDuration,
		remaining:           workDuration,
		isRest:              false,
		progress:            &p,
		currentIntervalName: "",
		lastCustomName:      "",
		intervalNames:       make(map[int]string),
		workedDurationsByName: make(map[string]time.Duration),
		namingMode:          false,
		nameInput:           "",
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
		intervalName := m.currentIntervalName
		if intervalName == "" {
			intervalName = fmt.Sprintf("Interval #%d", m.intervalsCompleted+1)
		}
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
