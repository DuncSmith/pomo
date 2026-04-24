# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Pomodoro timer CLI application built with Go using the Bubbletea TUI framework. Features a terminal UI with visual progress bars, pause/resume, interval naming, task tracking, session summaries (terminal + Markdown file), and system notifications.

## Common Commands

### Build and Run
```bash
go build -o pomo .           # Build the executable
./pomo                       # Run with default 50m work / 10m rest (60m interval)
./pomo 25                    # Run 25m work / 35m rest (60m interval)
./pomo 45 --interval 90     # Run 45m work / 45m rest (90m interval)
./pomo --help                # Show usage information
./pomo --version             # Show version, commit, build date
bin/build                    # Build versioned binary and install to ~/.local/bin
bin/release                  # Bump version and push a semver git tag
```

### Testing
```bash
go test                      # Run all tests
go test -v                   # Run tests with verbose output
go test -run TestName        # Run a specific test
```

### Dependencies
```bash
go mod tidy                  # Clean up dependencies
go mod download              # Download dependencies
```

## Architecture

### Core Structure
- **Multi-file architecture**: Code is organized by concern across files, all in `package main`
  - `main.go` (~290 lines) — Bubbletea `Model`, `Init`/`Update`/`View`, phase transitions, task tracking, input handlers, `formatTime`, `createProgressBar`
  - `config.go` (~85 lines) — `Config` type, YAML config loading/writing
  - `cli.go` (~110 lines) — Version vars, `parseArgsResult`, argument parsing, help text
  - `summary.go` (~100 lines) — Session summary output (terminal + Markdown file), `formatDurationHuman`
  - `notification.go` (~25 lines) — Desktop notifications (macOS/Linux)
- **Bubbletea TUI Framework**: Elm Architecture pattern (Model / Update / View)
- **Bubbles Components**: Official `progress.Model` component for the gradient progress bar

### Types

**`Config`** — User preferences loaded from `~/.config/pomo/config.yaml` (XDG-compliant):
- `SummaryFolder` (default `"~/pomos"`): Where to write Markdown session summaries
- `WorkTime` (default `50`): Default work phase minutes
- `IntervalTime` (default `60`): Default total interval minutes
- `ProduceSummary` (default `true`): Whether to write a Markdown summary file on quit

**`Model`** — Central Bubbletea state:
- `workDuration`, `restDuration`, `intervalDuration`: Phase configuration
- `remaining`: Time left in current phase
- `isRest`: Distinguishes work vs. break phase
- `paused`: Pause state
- `intervalsCompleted`: Count of fully completed work+rest cycles
- `totalWorked`, `totalRested`: Cumulative time for session summary
- `progress`: Bubbles progress bar component (pointer)
- `quitting`: Set on user quit; triggers summary output in `main()`
- `currentIntervalName`: Display name for the current work interval (e.g. `"Morning #1"`)
- `namingMode` / `nameInput`: State for the inline interval rename prompt
- `taskMode` / `taskInput`: State for the inline task name input prompt
- `tasks []Task`: Append-only log of all task records for the session
- `startedAt`: Session start timestamp (used to name the summary file)

**`Task`** — Named unit of work:
- `Name string`: User-provided task name
- `StartedAt time.Time`: When this task entry began
- `EndedAt time.Time`: When it ended; zero value means currently active

**Message Types**:
- `tickMsg`: 1-second timer tick (drives the countdown)
- `finishedMsg`: Emitted when `remaining` hits 0 (triggers phase transition + notification)
- Built-in `tea.KeyPressMsg` and `tea.WindowSizeMsg`

### Key Functions

**Bubbletea lifecycle**:
- `Init()`: Returns `tickCmd()` to start the 1-second tick loop
- `Update()`: Dispatches on message type; delegates to `handleNamingInput()` / `handleTaskInput()` when those modes are active
- `View()`: Renders naming/task prompts, or the main timer UI (header, interval name, countdown, progress bar, active task, key hints)

**Phase transitions**:
- `transitionToRest(elapsed)`: Accumulates work time, sets `isRest=true`, ends active task
- `transitionToWork()`: Accumulates rest time, increments counter, generates new interval name, continues last task name into a new Task entry
- `generateIntervalName(n, now)`: Produces `"Morning #N"` / `"Afternoon #N"` / `"Evening #N"` / `"Night #N"` based on time of day

**Task tracking** (pointer receivers, mutate in place):
- `activeTask()`: Returns pointer to the last task with a zero `EndedAt`, or nil
- `endActiveTask(at)`: Stamps `EndedAt` on the currently active task
- `startTask(name, at)`: Ends any active task, appends a new Task entry

**Input modes**:
- `handleNamingInput()`: Enter/Esc to commit/cancel; Backspace/Delete are UTF-8 rune-aware
- `handleTaskInput()`: Same logic; on Enter calls `startTask()` if input is non-empty

**CLI**:
- `parseArgs(args, cfg)`: Parses positional work duration + `--interval`/`-i` flag; falls back to config values; validates constraints; handles `--help`/`-h` and `--version`/`-v`
- `parseDuration(arg)`: Accepts `"30"`, `"30m"`, `"30s"`; bare integer = minutes
- `showHelp()`: Prints usage, examples, and options

**Output**:
- `printSummary(m)`: Prints intervals completed, total work/rest, and per-task time (aggregated by name) to stdout
- `writeSummaryFile(m, cfg)`: Writes the same content as a Markdown file to `cfg.SummaryFolder/<startedAt>.md`
- `formatTime(d)`: Formats duration as `"MM:SS"` (hours overflow into minutes)
- `formatDurationHuman(d)`: Human-readable format (`"Xh Ym"`, `"Xm Ys"`, `"Xs"`)

**Config**:
- `loadConfig()`: Reads YAML config; on first run, writes defaults to disk via `writeDefaultConfig()` and returns defaults
- `configPath()`: Resolves `$XDG_CONFIG_HOME/pomo/config.yaml` or `~/.config/pomo/config.yaml`

**Notifications**:
- `sendNotification(isRest, intervalName)`: macOS uses `terminal-notifier`; Linux uses `notify-send`; errors are non-fatal

### Command Line Interface

```
pomo [duration] [--interval duration] [-h|--help] [-v|--version]
```

Duration formats: `30` (minutes), `30m`, `30s`. Defaults come from config file.

Validation: work > 0, interval > 0, work < interval. Rest = interval − work.

### Keyboard Controls

| Key | Effect |
|-----|--------|
| `Space` | Toggle pause/resume |
| `q` / `Ctrl+C` | Quit; print session summary |
| `n` | Enter naming mode (rename current work interval; pre-fills current name) |
| `a` | Enter task mode (add/switch task; work phase only) |
| `s` | Skip current phase immediately |
| `Enter` | Confirm input (naming / task mode) |
| `Esc` | Cancel input (naming / task mode) |
| `Backspace` / `Delete` | Remove last character (UTF-8 rune-aware) |

### Testing Strategy

- ~30 test functions split across `main_test.go`, `config_test.go`, `cli_test.go`, `summary_test.go` (white-box, `package main`)
- Covers: `parseDuration`, `formatTime`, `createProgressBar`, `formatDurationHuman`, `parseArgs`, phase transitions, quit with partial progress, task tracking, naming/task input modes, config loading, interval name generation
- **Do not remove `createProgressBar()`** — it is never called at runtime but is required by existing tests

### Progress Bar Implementation

Two implementations coexist:
1. **`createProgressBar()`** — legacy ASCII (`█`/`░`); used only in tests
2. **`m.progress.ViewAs(float)`** — Bubbles `progress.Model` with gradient; used at runtime

Responsive width: `terminal_width - 4 - 20`, clamped to `[20, 80]`.

### Notifications

- macOS: `terminal-notifier` (prints `brew install` hint on failure)
- Linux: `notify-send` (errors silently ignored)
- Fire-and-forget; never blocks the TUI

### Release Pipeline

- GoReleaser cross-compiles for linux/darwin/windows × amd64/arm64 with CGO disabled
- `ldflags -X` injects `version`, `commit`, `date` into package vars (default to `"dev"`, `"none"`, `"unknown"`)
- `bin/release` script handles local semver tag creation; GitHub Actions runs GoReleaser on `v*` tag push
- CI workflow runs tests and build on push/PR to main
