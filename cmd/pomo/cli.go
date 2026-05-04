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
	}

	if len(args) > 0 && args[0] == "report" {
		rArgs, err := parseReportArgs(args[1:])
		if err != nil {
			return nil, err
		}
		return &parseArgsResult{action: "report", reportArgs: rArgs}, nil
	}

	var workDuration time.Duration
	var intervalDuration time.Duration
	var hasWork, hasInterval bool
	var createSessionSummary *bool
	autoStartWork := cfg.AutoStartWork

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--interval", "-i":
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
		case "--create-session-summary":
			t := true
			createSessionSummary = &t
		case "--no-create-session-summary":
			f := false
			createSessionSummary = &f
		case "--auto-start-work":
			autoStartWork = true
		default:
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
	lunchDuration := time.Duration(cfg.LunchTime) * time.Minute

	p := progress.New(progress.WithDefaultBlend())
	now := time.Now()
	model := &Model{
		workDuration:        workDuration,
		restDuration:        restDuration,
		intervalDuration:    intervalDuration,
		lunchDuration:       lunchDuration,
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
		intervalStartedAt:   now,
		autoStartWork:       autoStartWork,
	}
	return &parseArgsResult{model: model, createSessionSummary: createSessionSummary}, nil
}

func showHelp() {
	fmt.Println("Usage: pomo [work] [--interval duration] [--auto-start-work]")
	fmt.Println("       pomo report [--last] [--from YYYY-MM-DD] [--to YYYY-MM-DD]")
	fmt.Println()
	fmt.Println("Runs repeating work/rest intervals until you quit.")
	fmt.Println("Rest time is always interval − work and cannot be set directly.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  pomo               # 50m work (10m rest)")
	fmt.Println("  pomo 25            # 25m work (35m rest)")
	fmt.Println("  pomo 25m           # 25m work (35m rest)")
	fmt.Println("  pomo 30s           # 30s work (59m30s rest)")
	fmt.Println("  pomo 45 -i 90      # 45m work, 90m interval (45m rest)")
	fmt.Println("  pomo report        # weekly category summary (current week)")
	fmt.Println("  pomo report --last # previous week")
	fmt.Println()
	fmt.Println("Timer options:")
	fmt.Println("  -i, --interval              Set total interval duration; rest = interval − work (default: 60m)")
	fmt.Println("      --auto-start-work           Automatically start work when rest ends (default: false)")
	fmt.Println("      --create-session-summary    Write a Markdown summary on quit (default: true)")
	fmt.Println("      --no-create-session-summary Disable writing the Markdown summary")
	fmt.Println()
	fmt.Println("Report options:")
	fmt.Println("      --last          Show previous work week instead of current")
	fmt.Println("      --from DATE     Start date (YYYY-MM-DD)")
	fmt.Println("      --to DATE       End date (YYYY-MM-DD; defaults to today when omitted)")
	fmt.Println()
	fmt.Println("General options:")
	fmt.Println("  -v, --version               Show version information")
	fmt.Println("  -h, --help                  Show this help")
	fmt.Println()
	fmt.Println("Default: 50m work (10m rest, 60m interval)")
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
