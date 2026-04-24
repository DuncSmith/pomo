package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

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
