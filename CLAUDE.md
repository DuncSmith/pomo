# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Working from a Plan

When implementing from a plan file, work through every item in the Todo List in order — including documentation updates — before declaring the work done. Do not skip trailing phases (e.g. "Final checks", doc updates) because the code phases are complete.

## Project Overview

A minimal Pomodoro timer CLI built with Go using the Bubbletea TUI framework. Single-session, no cross-session aggregation. Implements the classic technique: a pomodoro followed by a short break, with a long break after every N pomodoros (defaults: 25 / 5 / 10 / 4). Produces a per-session Markdown summary on quit.

See `CONTEXT.md` for the domain glossary and `docs/adr/0001-single-session-design.md` for the rationale behind the single-session decision.

## Common Commands

### Build and Run
```bash
go build -o pomo ./cmd/pomo  # Build the executable
./pomo                       # Use config (or built-in defaults if no config)
./pomo -w 50                 # 50m pomodoro, other durations from config
./pomo --init                # Create default config at ~/.config/pomo/config.yaml
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
- **Layout**: `cmd/pomo/` holds all source; `go.mod` at repo root.
- **Multi-file architecture**: all in `package main`:
  - `cmd/pomo/main.go` — `main()` entry point only
  - `cmd/pomo/model.go` — Types (`Model`, `Task`, message types), `Init`/`Update`/`View`, `formatTime`
  - `cmd/pomo/tasks.go` — Task management (`activeTask`, `endActiveTask`, `startTask`, `recentTaskNames`) and `handleTaskInput`
  - `cmd/pomo/timer.go` — Phase logic (`phaseDuration`, `currentBreakDuration`, `transitionToBreak`, `transitionToPomodoro`)
  - `cmd/pomo/config.go` — `Config` type, YAML config loading/writing
  - `cmd/pomo/cli.go` — Version vars, `parseArgsResult`, argument parsing, help text
  - `cmd/pomo/summary.go` — Session summary output (terminal + Markdown file), `computeTaskTotals`, `buildFrontmatter`, `formatDurationHuman`
  - `cmd/pomo/view.go` — Lipgloss styles + small layout helpers (`barWidth`, `keyHints`)
  - `cmd/pomo/notification.go` — Desktop notifications (macOS/Linux)
- **Bubbletea TUI Framework**: Elm Architecture (Model / Update / View); uses `charm.land/bubbletea/v2` and `charm.land/bubbles/v2` (not the `github.com/charmbracelet` paths).
- **Bubbles Components**: Official `progress.Model` for the gradient progress bar (`progress.WithDefaultBlend()`).

### Types

**`Config`** — User preferences loaded from `~/.config/pomo/config.yaml` (XDG-compliant):
- `PomodoroTime` (YAML: `pomodoro_time`, default `25`): work-phase minutes
- `ShortBreakTime` (YAML: `short_break_time`, default `5`): short break minutes
- `LongBreakTime` (YAML: `long_break_time`, default `10`): long break minutes
- `PomodorosPerCycle` (YAML: `pomodoros_per_cycle`, default `4`): number of pomodoros between long breaks
- `SessionSummary` (YAML: `session_summary`, nested):
  - `Create` (YAML: `create_session_summary`, default `true`): whether to write a Markdown summary file on quit
  - `Folder` (YAML: `summary_folder`, default `"~/pomos"`): where to write Markdown summaries
  - `Tags` (YAML: `summary_tags`, default `["daily", "pomo-summary"]`): tags for the YAML frontmatter block. An explicit `[]` in config omits the `tags` key entirely.

**`Model`** — Central Bubbletea state:
- `pomodoroDuration`, `shortBreakDuration`, `longBreakDuration`, `pomodorosPerCycle`: phase configuration
- `remaining`: time left in current phase
- `isRest`: distinguishes work (pomodoro) vs. break phase
- `paused`: pause state
- `pomodorosCompleted`: count of fully completed pomodoros
- `totalWorked`, `totalRested`: cumulative time for session summary
- `progress`: Bubbles progress bar component (pointer)
- `quitting`: set on user quit; triggers summary output in `main()`
- `tasks []Task`: append-only log of all task records for the session
- `taskMode` / `taskInput`: state for the inline task name input prompt
- `recentTasks []string`: recent unique task names populated when entering task mode; cleared on exit
- `startedAt`: session start timestamp (used to name the summary file)
- `width`: terminal width, updated via `WindowSizeMsg`

**`Task`** — Named unit of work:
- `Name string`: user-provided task name
- `StartedAt time.Time`: when this task entry began
- `EndedAt time.Time`: when it ended; zero value means currently active

**Message types**:
- `tickMsg`: 1-second timer tick (drives the countdown)
- `finishedMsg`: emitted when `remaining` hits 0 (triggers phase transition + notification)
- Built-in `tea.KeyPressMsg` and `tea.WindowSizeMsg`

### Key Functions

**Bubbletea lifecycle**:
- `Init()`: Returns `tickCmd()` to start the 1-second tick loop
- `Update()`: Dispatches on message type; delegates to `handleTaskInput()` when task mode is active
- `View()`: Returns `tea.View` (via `tea.NewView()`); renders the task-input prompt or the main minimal view (header / countdown / progress bar / task line / key hints)
- `phaseDuration()`: Returns `pomodoroDuration` during work, or `currentBreakDuration()` during a break
- `currentBreakDuration()`: Returns `longBreakDuration` when the just-completed pomodoro is a multiple of `pomodorosPerCycle`, otherwise `shortBreakDuration`

**Phase transitions** (both auto-transition; no user gate):
- `transitionToBreak(elapsed, now)`: Accumulates `elapsed` work time, sets `isRest=true`, sets `remaining` to short or long break (via `currentBreakDuration()`), ends active task
- `transitionToPomodoro(elapsed, now)`: Accumulates `elapsed` rest time, increments `pomodorosCompleted`, resets `remaining` to `pomodoroDuration`, continues last task name into a new Task entry

**Task tracking** (pointer receivers, mutate in place):
- `activeTask()`: Returns pointer to the last task with a zero `EndedAt`, or nil
- `endActiveTask(at)`: Stamps `EndedAt` on the currently active task
- `startTask(name, at)`: Ends any active task, appends a new Task entry

**Input modes**:
- `handleTaskInput()`: Enter/Esc to commit/cancel; Backspace/Delete are UTF-8 rune-aware; when `taskInput` is empty a digit `1`–`9` selects from `recentTasks` (if in range) or falls through to free-text
- `recentTaskNames(tasks)`: Walks `tasks` in reverse, deduplicates, excludes the active task, returns at most 9 names most-recent-first

**CLI**:
- `parseArgs(args, cfg)`: Parses `--work`/`-w`; falls back to `cfg.PomodoroTime`; validates each duration is positive and `pomodoros_per_cycle ≥ 1`; handles `--help`/`-h`, `--version`/`-v`, `--init`; any unrecognised argument returns an error
- `parseDuration(arg)`: Accepts `"30"`, `"30m"`, `"30s"`; bare integer = minutes
- `showHelp()`: Prints usage, examples, options

**Output**:
- `printSummary(m)`: Prints pomodoros completed, total work/rest, and a flat alphabetical list of tasks with totals
- `writeSummaryFile(m, cfg)`: Writes the same content as a Markdown file to `cfg.SessionSummary.Folder/<startedAt>.md`; skips if `cfg.SessionSummary.Create` is false
- `computeTaskTotals(tasks)`: Aggregates task durations by name, sorted alphabetically
- `buildFrontmatter(tags, created)`: Builds the YAML frontmatter block; omits the `tags` key when the slice is empty
- `formatTime(d)`: Formats duration as `"MM:SS"` (hours overflow into minutes)
- `formatDurationHuman(d)`: Human-readable format (`"Xh Ym"`, `"Xm Ys"`, `"Xs"`)

**Config**:
- `defaultConfig()`: Returns the built-in default `Config`
- `loadConfig()`: Reads YAML config; returns defaults if no file exists (does NOT auto-write); errors on malformed YAML
- `writeDefaultConfig(path, cfg)`: Writes a config file. Triggered explicitly by `pomo --init`
- `configPath()`: Resolves `$XDG_CONFIG_HOME/pomo/config.yaml` or `~/.config/pomo/config.yaml`

**Notifications**:
- `sendNotification(isRest bool)`: macOS uses `terminal-notifier`; Linux uses `notify-send`; errors are non-fatal

### Command Line Interface

```
pomo [-w duration]
pomo --init
pomo -h | --help
pomo -v | --version
```

Duration formats: `30` (minutes), `30m`, `30s`. Other durations (`short_break_time`, `long_break_time`, `pomodoros_per_cycle`) come from the config file only — no CLI flags.

Validation: pomodoro > 0, short-break > 0, long-break > 0, pomodoros_per_cycle ≥ 1.

### Keyboard Controls

| Key | Effect |
|-----|--------|
| `Space` | Toggle pause/resume |
| `q` / `Ctrl+C` | Quit; print session summary |
| `a` | Enter task mode (add/switch task; work phase only) |
| `s` | Skip current phase immediately (auto-transitions to the other phase) |
| `Enter` | Confirm task input |
| `Esc` | Cancel task input |
| `Backspace` / `Delete` | Remove last character (UTF-8 rune-aware) |
| `1`–`9` | In task mode with empty input: resume the corresponding recent task |

### Testing Strategy

- Test functions split across `main_test.go`, `config_test.go`, `cli_test.go`, `summary_test.go` (white-box, `package main`)
- Covers: `parseDuration`, `formatTime`, `formatDurationHuman`, `parseArgs`, phase transitions (both directions auto-transition), long-break selection (`TestTransitionToBreakUsesLongBreakAfterPerCycle`, `TestPhaseDurationLongBreak`), quit with partial progress, skip with partial progress, task tracking, task input mode, recent task picker, config loading, frontmatter generation

### TUI Layout

Minimal four-block layout:

```
🍅 Focus            ← header (emoji + phase label)

  14:32             ← countdown (small text), with "(paused)" suffix if paused
  ████████░░░░░░░░  ← gradient progress bar (Bubbles)

  › task name       ← active task (work phase only)

  [space] pause   [a] task   [s] skip   [q] quit   ← key hints
```

Break view drops the task line and the `[a]` hint.

Progress bar width: `terminal_width - 4`, clamped to `[20, 80]`.

### Notifications

- macOS: `terminal-notifier`; Linux: `notify-send` — both silently ignore errors
- Called as `go sendNotification(...)` from `Update()` so it never blocks the tick loop

### Release Pipeline

- GoReleaser cross-compiles for linux/darwin/windows × amd64/arm64 with CGO disabled
- `ldflags -X` injects `version`, `commit`, `date` into package vars (default to `"dev"`, `"none"`, `"unknown"`)
- `bin/release` script handles local semver tag creation; GitHub Actions runs GoReleaser on `v*` tag push
- CI workflow runs tests and build on push/PR to main

## Agent skills

### Issue tracker

Issues are tracked as GitHub issues on `DuncSmith/pomo` via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Uses the default canonical triage label vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context repo: one `CONTEXT.md` and `docs/adr/` at the repo root (created lazily by `/grill-with-docs`). See `docs/agents/domain.md`.
