package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Timer struct {
	duration time.Duration
	isRest   bool
}

func main() {
	timer, err := parseArgs()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	
	timer.displayHeader()
	timer.start()
}

func parseArgs() (*Timer, error) {
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
		return &Timer{duration: duration, isRest: true}, nil
	}
	
	duration := 45 * time.Minute
	if len(os.Args) > 1 {
		var err error
		duration, err = parseDuration(os.Args[1])
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %s", os.Args[1])
		}
	}
	
	return &Timer{duration: duration, isRest: false}, nil
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

func (t *Timer) displayHeader() {
	emoji := "🍅"
	title := "Pomodoro Timer"
	if t.isRest {
		emoji = "☕"
		title = "Break Timer"
	}
	
	if t.duration < time.Minute {
		fmt.Printf("%s %s: %d seconds\n", emoji, title, int(t.duration.Seconds()))
	} else {
		fmt.Printf("%s %s: %.1f minutes\n", emoji, title, t.duration.Minutes())
	}
	fmt.Println()
}

func (t *Timer) start() {
	remaining := t.duration
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	
	for remaining > 0 {
		elapsed := t.duration - remaining
		percentage := float64(elapsed) / float64(t.duration) * 100
		
		fmt.Print("\033[2K\r")
		fmt.Printf("⏰ %s [%s] %.1f%%", 
			formatTime(remaining),
			createProgressBar(percentage, 30),
			percentage)
		
		<-ticker.C
		remaining -= time.Second
	}
	
	fmt.Print("\033[2K\r")
	if t.isRest {
		fmt.Println("🎉 Break completed!")
	} else {
		fmt.Println("🎉 Pomodoro completed!")
	}
	
	t.sendNotification()
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

func (t *Timer) sendNotification() {
	var title, message string
	if t.isRest {
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
