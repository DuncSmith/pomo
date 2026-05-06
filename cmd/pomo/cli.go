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

// reportArgs holds parsed flags for the report subcommand.
type reportArgs struct {
	Last bool
	From *time.Time
	To   *time.Time
}

// parseArgsResult represents the result of parsing command-line arguments
type parseArgsResult struct {
	model                *Model
	action               string // "", "help", "version", or "report"
	versionInfo          string
	createSessionSummary *bool // nil means "use config value"
	reportArgs           *reportArgs
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

	if len(args) > 0 && args[0] == "report" {
		rArgs, err := parseReportArgs(args[1:])
		if err != nil {
			return nil, err
		}
		return &parseArgsResult{action: "report", reportArgs: rArgs}, nil
	}

	var pomodoroDuration, shortBreakDuration, longBreakDuration time.Duration
	var pomodorosPerCycle int
	var hasPomodoro, hasShortBreak, hasLongBreak, hasPerCycle bool
	var createSessionSummary *bool
	autoStartWork := cfg.AutoStartWork

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--short-break", "-s":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--short-break requires a value")
			}
			d, err := parseDuration(args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid short-break duration: %s", args[i+1])
			}
			shortBreakDuration = d
			hasShortBreak = true
			i++
		case "--long-break", "-L":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--long-break requires a value")
			}
			d, err := parseDuration(args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid long-break duration: %s", args[i+1])
			}
			longBreakDuration = d
			hasLongBreak = true
			i++
		case "--per-cycle", "-c":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--per-cycle requires a value")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid --per-cycle value: %s", args[i+1])
			}
			pomodorosPerCycle = n
			hasPerCycle = true
			i++
		case "--create-session-summary":
			t := true
			createSessionSummary = &t
		case "--no-create-session-summary":
			f := false
			createSessionSummary = &f
		case "--auto-start-work":
			autoStartWork = true
		default:
			if hasPomodoro {
				return nil, fmt.Errorf("unexpected argument: %s", args[i])
			}
			d, err := parseDuration(args[i])
			if err != nil {
				return nil, fmt.Errorf("invalid duration: %s", args[i])
			}
			pomodoroDuration = d
			hasPomodoro = true
		}
	}

	if !hasPomodoro {
		pomodoroDuration = time.Duration(cfg.PomodoroTime) * time.Minute
	}
	if !hasShortBreak {
		shortBreakDuration = time.Duration(cfg.ShortBreakTime) * time.Minute
	}
	if !hasLongBreak {
		longBreakDuration = time.Duration(cfg.LongBreakTime) * time.Minute
	}
	if !hasPerCycle {
		pomodorosPerCycle = cfg.PomodorosPerCycle
	}

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
		return nil, fmt.Errorf("--per-cycle must be at least 1")
	}

	p := progress.New(progress.WithDefaultBlend())
	now := time.Now()
	model := &Model{
		pomodoroDuration:    pomodoroDuration,
		shortBreakDuration:  shortBreakDuration,
		longBreakDuration:   longBreakDuration,
		pomodorosPerCycle:   pomodorosPerCycle,
		remaining:           pomodoroDuration,
		isRest:              false,
		progress:            &p,
		currentPomodoroName: generatePomodoroName(1, now),
		tasks:               []Task{},
		namingMode:          false,
		nameInput:           "",
		taskMode:            false,
		taskInput:           "",
		startedAt:           now,
		pomodoroStartedAt:   now,
		autoStartWork:       autoStartWork,
	}
	return &parseArgsResult{model: model, createSessionSummary: createSessionSummary}, nil
}

func showHelp() {
	fmt.Println("Usage: pomo [pomodoro] [--short-break duration] [--long-break duration] [--per-cycle N] [--auto-start-work]")
	fmt.Println("       pomo --init")
	fmt.Println("       pomo report [--last] [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
	fmt.Println()
	fmt.Println("Runs the classic Pomodoro technique: a pomodoro followed by a short break,")
	fmt.Println("with a longer break after every N pomodoros, until you quit.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  pomo                   # 25m pomodoro / 5m short / 10m long / 4 per cycle")
	fmt.Println("  pomo 50                # 50m pomodoro")
	fmt.Println("  pomo 50 -s 10 -L 20    # 50m pomodoro, 10m short break, 20m long break")
	fmt.Println("  pomo -c 3              # long break every 3rd pomodoro")
	fmt.Println("  pomo report            # weekly category summary (current week)")
	fmt.Println("  pomo report --last     # previous week")
	fmt.Println()
	fmt.Println("Timer options:")
	fmt.Println("  -s, --short-break           Short break duration (default: 5m)")
	fmt.Println("  -L, --long-break            Long break duration (default: 10m)")
	fmt.Println("  -c, --per-cycle             Number of pomodoros between long breaks (default: 4)")
	fmt.Println("      --auto-start-work           Automatically start the next pomodoro when a break ends (default: false)")
	fmt.Println("      --create-session-summary    Write a Markdown summary on quit (default: true)")
	fmt.Println("      --no-create-session-summary Disable writing the Markdown summary")
	fmt.Println()
	fmt.Println("Report options:")
	fmt.Println("      --last          Show previous work week instead of current")
	fmt.Println("      --from DATE     Start date (YYYY-MM-DD)")
	fmt.Println("      --to DATE       End date (YYYY-MM-DD; defaults to today when omitted)")
	fmt.Println()
	fmt.Println("General options:")
	fmt.Println("      --init                  Create default config at ~/.config/pomo/config.yaml")
	fmt.Println("  -v, --version               Show version information")
	fmt.Println("  -h, --help                  Show this help")
	fmt.Println()
	fmt.Println("Default: 25m pomodoro / 5m short break / 10m long break / 4 per cycle")
}

func parseReportArgs(args []string) (*reportArgs, error) {
	result := &reportArgs{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--last":
			result.Last = true
		case "--from":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--from requires a date (YYYY-MM-DD)")
			}
			t, err := time.Parse("2006-01-02", args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid --from date %q (expected YYYY-MM-DD)", args[i+1])
			}
			result.From = &t
			i++
		case "--to":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--to requires a date (YYYY-MM-DD)")
			}
			t, err := time.Parse("2006-01-02", args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid --to date %q (expected YYYY-MM-DD)", args[i+1])
			}
			result.To = &t
			i++
		default:
			return nil, fmt.Errorf("unknown report flag: %s", args[i])
		}
	}

	if result.Last && (result.From != nil || result.To != nil) {
		return nil, fmt.Errorf("--last cannot be used with --from or --to")
	}
	if result.To != nil && result.From == nil {
		return nil, fmt.Errorf("--to requires --from")
	}
	if result.From != nil && result.To != nil && result.From.After(*result.To) {
		return nil, fmt.Errorf("--from must be on or before --to")
	}

	return result, nil
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
