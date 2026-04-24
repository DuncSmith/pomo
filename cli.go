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
	action      string // "", "help", or "version"
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
		workDuration = time.Duration(cfg.WorkTime) * time.Minute
	}
	if !hasInterval {
		intervalDuration = time.Duration(cfg.IntervalTime) * time.Minute
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
	now := time.Now()
	model := &Model{
		workDuration:        workDuration,
		restDuration:        restDuration,
		intervalDuration:    intervalDuration,
		remaining:           workDuration,
		isRest:              false,
		progress:            &p,
		currentIntervalName: generateIntervalName(1, now),
		tasks:               []Task{},
		namingMode:          false,
		nameInput:           "",
		taskMode:            false,
		taskInput:           "",
		startedAt:           now,
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
