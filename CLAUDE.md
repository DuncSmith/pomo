# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Pomodoro timer CLI application built with Go using the Bubbletea TUI framework. Features a terminal UI with visual progress bars, pause/resume, interval naming, task tracking, session summaries (terminal + Markdown file), and system notifications.

## Common Commands

### Build and Run
```bash
go build -o pomo ./cmd/pomo  # Build the executable
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
go test ./...                # Run all tests
go test -v ./...             # Run tests with verbose output
go test -run TestName ./...  # Run a specific test
```

### Dependencies
```bash
go mod tidy                  # Clean up dependencies
go mod download              # Download dependencies
```

## Architecture

### Core Structure
- **Layout**: `cmd/pomo/` holds all source; `go.mod` at repo root
- **Multi-file architecture**: Code is organized by concern across files, all in `package main`
  - `cmd/pomo/main.go` (~47 lines) — `main()` entry point only
  - `cmd/pomo/model.go` (~238 lines) — Types (`Model`, `Task`, message types), `Init`/`Update`/`View`, `formatTime`
  - `cmd/pomo/tasks.go` (~123 lines) — Task management (`activeTask`, `endActiveTask`, `startTask`, `recentTaskNames`), input handlers (`handleNamingInput`, `handleTaskInput`)
  - `cmd/pomo/timer.go` (~55 lines) — Phase/interval logic (`phaseDuration`, `generateIntervalName`, `transitionToRest`, `transitionToWork`)
  - `cmd/pomo/config.go` (~92 lines) — `Config` type, YAML config loading/writing
  - `cmd/pomo/cli.go` (~152 lines) — Version vars, `parseArgsResult`, argument parsing, help text
  - `cmd/pomo/summary.go` (~155 lines) — Session summary output (terminal + Markdown file), `computeTaskTotals`, `buildFrontmatter`, `formatDurationHuman`
  - `cmd/pomo/notification.go` (~29 lines) — Desktop notifications (macOS/Linux)
- **Bubbletea TUI Framework**: Elm Architecture pattern (Model / Update / View); uses `charm.land/bubbletea/v2` and `charm.land/bubbles/v2` (not the `github.com/charmbracelet` paths)
- **Bubbles Components**: Official `progress.Model` component for the gradient progress bar (`progress.WithDefaultBlend()`)

### Types

**`Config`** — User preferences loaded from `~/.config/pomo/config.yaml` (XDG-compliant):
- `WorkTime` (YAML: `work_time`, default `50`): Default work phase minutes
- `IntervalTime` (YAML: `interval_time`, default `60`): Default total interval minutes
- `SessionSummary` (YAML: `session_summary`, nested block via `SessionSummary` type):
  - `Create` (YAML: `create_session_summary`, default `true`): Whether to write a Markdown summary file on quit
  - `Folder` (YAML: `summary_folder`, default `"~/pomos"`): Where to write Markdown session summaries
  - `Tags` (YAML: `summary_tags`, default `["daily", "pomo summary"]`): Custom tags for the YAML frontmatter block. Set by `defaultConfig()` and preserved as the YAML unmarshal default when the field is omitted. An explicit `[]` in config omits the `tags` key in the frontmatter entirely.

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
- `recentTasks []string`: Recent unique task names populated when entering task mode; cleared on exit
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
- `View()`: Returns `tea.View` (via `tea.NewView()`); renders naming/task prompts, or the main timer UI (header, interval name, countdown, progress bar, active task, key hints)
- `phaseDuration()`: Returns `workDuration` or `restDuration` based on `isRest`

**Phase transitions**:
- `transitionToRest(elapsed, now)`: Accumulates `elapsed` work time, sets `isRest=true`, ends active task at `now`
- `transitionToWork(elapsed, now)`: Accumulates `elapsed` rest time, increments counter, generates new interval name, continues last task name into a new Task entry
- `generateIntervalName(n, now)`: Produces `"Morning #N"` / `"Afternoon #N"` / `"Evening #N"` / `"Night #N"` based on time of day

**Task tracking** (pointer receivers, mutate in place):
- `activeTask()`: Returns pointer to the last task with a zero `EndedAt`, or nil
- `endActiveTask(at)`: Stamps `EndedAt` on the currently active task
- `startTask(name, at)`: Ends any active task, appends a new Task entry

**Input modes**:
- `handleNamingInput()`: Enter/Esc to commit/cancel; Backspace/Delete are UTF-8 rune-aware
- `handleTaskInput()`: Enter/Esc to commit/cancel; when `taskInput` is empty a digit `1`–`9` selects from `recentTasks` (if in range) or falls through to free-text
- `recentTaskNames(tasks)`: Walks `tasks` in reverse, deduplicates, excludes the active task, returns at most 9 names most-recent-first

**CLI**:
- `parseArgs(args, cfg)`: Parses positional work duration + `--interval`/`-i` flag; falls back to config values; validates constraints; handles `--help`/`-h` and `--version`/`-v`
- `parseDuration(arg)`: Accepts `"30"`, `"30m"`, `"30s"`; bare integer = minutes
- `showHelp()`: Prints usage, examples, and options

**Output**:
- `printSummary(m)`: Prints intervals completed, total work/rest, and per-task time (aggregated by name) to stdout
- `writeSummaryFile(m, cfg)`: Writes the same content as a Markdown file to `cfg.SummaryFolder/<startedAt>.md`; skips if `cfg.SessionSummary.Create` is false
- `buildFrontmatter(tags, created)`: Builds the YAML frontmatter block (`---\ncreated: …\ntags:\n  - …\n---`) for the Markdown summary; omits `tags` key when the slice is empty
- `formatTime(d)`: Formats duration as `"MM:SS"` (hours overflow into minutes)
- `formatDurationHuman(d)`: Human-readable format (`"Xh Ym"`, `"Xm Ys"`, `"Xs"`)

**Config**:
- `defaultConfig()`: Returns the built-in default `Config` struct (used as unmarshal base and fallback)
- `loadConfig()`: Reads YAML config; on first run, writes defaults to disk via `writeDefaultConfig()` and returns defaults
- `configPath()`: Resolves `$XDG_CONFIG_HOME/pomo/config.yaml` or `~/.config/pomo/config.yaml`

**Notifications**:
- `sendNotification(isRest, intervalName)`: macOS uses `terminal-notifier`; Linux uses `notify-send`; errors are non-fatal

### Command Line Interface

```
pomo [duration] [--interval duration] [--create-session-summary|--no-create-session-summary] [-h|--help] [-v|--version]
```

Duration formats: `30` (minutes), `30m`, `30s`. Defaults come from config file.

Validation: work > 0, interval > 0, work < interval. Rest = interval − work.

The `--create-session-summary` / `--no-create-session-summary` flags override `cfg.SessionSummary.Create` for that run only; omitting them leaves the config value unchanged. The override is stored as `*bool` in `parseArgsResult` (nil = use config).

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

- ~44 test functions split across `main_test.go`, `config_test.go`, `cli_test.go`, `summary_test.go` (white-box, `package main`)
- Covers: `parseDuration`, `formatTime`, `formatDurationHuman`, `parseArgs`, `--create-session-summary` flag, phase transitions, quit with partial progress, task tracking, naming/task input modes, config loading, interval name generation, frontmatter generation, recent task picker

### Progress Bar Implementation

Uses `m.progress.ViewAs(float)` — Bubbles `progress.Model` with gradient.

Responsive width: `terminal_width - 4 - 20`, clamped to `[20, 80]`.

### Notifications

- macOS: `terminal-notifier`; Linux: `notify-send` — both silently ignore errors
- Called as `go sendNotification(...)` from `Update()` so it never blocks the tick loop

### Release Pipeline

- GoReleaser cross-compiles for linux/darwin/windows × amd64/arm64 with CGO disabled
- `ldflags -X` injects `version`, `commit`, `date` into package vars (default to `"dev"`, `"none"`, `"unknown"`)
- `bin/release` script handles local semver tag creation; GitHub Actions runs GoReleaser on `v*` tag push
- CI workflow runs tests and build on push/PR to main
