package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
)

// Set via -ldflags by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// parseArgsResult represents the result of parsing command-line arguments
type parseArgsResult struct {
	model       *Model
	action      string // "", "help", "version", or "init"
	versionInfo string
}

func parseArgs(args []string, cfg Config) (*parseArgsResult, error) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return &parseArgsResult{action: "help"}, nil
		}
		if arg == "-v" || arg == "--version" {
			info := fmt.Sprintf("pomo %s (commit: %s, built: %s)", version, commit, date)
			return &parseArgsResult{action: "version", versionInfo: info}, nil
		}
		if arg == "--init" {
			return &parseArgsResult{action: "init"}, nil
		}
	}

	var pomodoroDuration time.Duration
	var hasPomodoro bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--work", "-w":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--work requires a value")
			}
			d, err := parseDuration(args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid work duration: %s", args[i+1])
			}
			pomodoroDuration = d
			hasPomodoro = true
			i++
		default:
			return nil, fmt.Errorf("unknown argument: %s", args[i])
		}
	}

	if !hasPomodoro {
		pomodoroDuration = time.Duration(cfg.PomodoroTime) * time.Minute
	}
	shortBreakDuration := time.Duration(cfg.ShortBreakTime) * time.Minute
	longBreakDuration := time.Duration(cfg.LongBreakTime) * time.Minute
	pomodorosPerCycle := cfg.PomodorosPerCycle

	if pomodoroDuration <= 0 {
		return nil, fmt.Errorf("pomodoro duration must be greater than 0")
	}
	if shortBreakDuration <= 0 {
		return nil, fmt.Errorf("short-break duration must be greater than 0")
	}
	if longBreakDuration <= 0 {
		return nil, fmt.Errorf("long-break duration must be greater than 0")
	}
	if pomodorosPerCycle < 1 {
		return nil, fmt.Errorf("pomodoros_per_cycle must be at least 1")
	}

	p := progress.New(progress.WithDefaultBlend())
	now := time.Now()
	model := &Model{
		pomodoroDuration:   pomodoroDuration,
		shortBreakDuration: shortBreakDuration,
		longBreakDuration:  longBreakDuration,
		pomodorosPerCycle:  pomodorosPerCycle,
		remaining:          pomodoroDuration,
		isRest:             false,
		progress:           &p,
		tasks:              []Task{},
		taskMode:           false,
		taskInput:          "",
		startedAt:          now,
	}
	return &parseArgsResult{model: model}, nil
}

func showHelp() {
	fmt.Println("Usage: pomo [-w duration]")
	fmt.Println("       pomo --init")
	fmt.Println()
	fmt.Println("Runs the classic Pomodoro technique: a pomodoro followed by a short break,")
	fmt.Println("with a longer break after every N pomodoros, until you quit.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  pomo            # use config (or built-in defaults if no config)")
	fmt.Println("  pomo -w 50      # 50m pomodoro, other durations from config")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -w, --work      Work (pomodoro) duration (default: 25m)")
	fmt.Println("      --init      Create default config at ~/.config/pomo/config.yaml")
	fmt.Println("  -v, --version   Show version information")
	fmt.Println("  -h, --help      Show this help")
	fmt.Println()
	fmt.Println("Edit ~/.config/pomo/config.yaml to change durations or per-cycle count.")
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
