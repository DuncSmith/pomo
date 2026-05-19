package main

import (
	"os/exec"
	"runtime"
)

func sendNotification(isRest bool) {
	var title, message string
	if isRest {
		title = "☕ Break Timer"
		message = "Your break is complete!"
	} else {
		title = "🍅 Pomodoro Timer"
		message = "Pomodoro is complete!"
	}

	if runtime.GOOS == "darwin" {
		cmd := exec.Command("terminal-notifier", "-title", title, "-message", message, "-sound", "default")
		cmd.Run()
	} else {
		cmd := exec.Command("notify-send", title, message)
		cmd.Run()
	}
}
