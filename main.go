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
	duration  time.Duration
	remaining time.Duration
	isRest    bool
	paused    bool
	progress  progress.Model
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case " ":
			m.paused = !m.paused
			return m, nil
		}
	case tea.WindowSizeMsg:
		const padding = 4
		const maxWidth = 80
		m.progress.Width = msg.Width - padding - 20 // Leave space for time and percentage
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
		return m, tea.Quit
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
	
	if m.duration < time.Minute {
		s.WriteString(fmt.Sprintf("%s %s: %d seconds\n", emoji, title, int(m.duration.Seconds())))
	} else {
		s.WriteString(fmt.Sprintf("%s %s: %.1f minutes\n", emoji, title, m.duration.Minutes()))
	}
	s.WriteString("\n")
	
	elapsed := m.duration - m.remaining
	percentage := float64(elapsed) / float64(m.duration) * 100
	
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
			s.WriteString(fmt.Sprintf("⏸️  %s %s %.1f%% (PAUSED)\n", timeStr, progressBar, percentage))
		} else {
			s.WriteString(fmt.Sprintf("⏰ %s %s %.1f%%\n", timeStr, progressBar, percentage))
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
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs() (*Model, error) {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		showHelp()
		os.Exit(0)
	}
	
	if len(os.Args) > 1 && os.Args[1] == "rest" {
		duration := 15 * time.Minute
		if len(os.Args) > 2 {
			var err error
			duration, err = parseDuration(os.Args[2])
			if err != nil {
				return nil, fmt.Errorf("invalid duration: %s", os.Args[2])
			}
		}
		return &Model{
			duration:  duration,
			remaining: duration,
			isRest:    true,
			progress:  progress.New(progress.WithDefaultGradient()),
		}, nil
	}
	
	duration := 45 * time.Minute
	if len(os.Args) > 1 {
		var err error
		duration, err = parseDuration(os.Args[1])
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %s", os.Args[1])
		}
	}
	
	return &Model{
		duration:  duration,
		remaining: duration,
		isRest:    false,
		progress:  progress.New(progress.WithDefaultGradient()),
	}, nil
}

func showHelp() {
	fmt.Println("Usage: pomo [duration] | pomo rest [duration]")
	fmt.Println("Examples:")
	fmt.Println("  pomo 30     # 30 minutes work timer")
	fmt.Println("  pomo 30m    # 30 minutes work timer")
	fmt.Println("  pomo 30s    # 30 seconds work timer")
	fmt.Println("  pomo rest   # 15 minute break timer")
	fmt.Println("  pomo rest 5m # 5 minute break timer")
	fmt.Println("Default: 45 minutes work timer")
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
