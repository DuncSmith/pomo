# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Working from a Plan

When implementing from a plan file, work through every item in the Todo List in order — including documentation updates — before declaring the work done. Do not skip trailing phases (e.g. "Final checks", doc updates) because the code phases are complete.

## Project Overview

A Pomodoro timer CLI application built with Go using the Bubbletea TUI framework. Features a terminal UI with visual progress bars, pause/resume, pomodoro naming, task tracking, session summaries (terminal + Markdown file), and system notifications. Implements the classic technique: a pomodoro followed by a short break, with a long break after every N pomodoros (defaults: 25 / 5 / 10 / 4).

## Common Commands

### Build and Run
```bash
go build -o pomo ./cmd/pomo  # Build the executable
./pomo                       # Defaults: 25m pomodoro / 5m short / 10m long / 4 per cycle
./pomo -w 50                 # 50m pomodoro
./pomo -w 50 -s 10 -L 20     # 50m pomodoro, 10m short break, 20m long break
./pomo -c 3                  # long break every 3rd pomodoro
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
  - `cmd/pomo/model.go` (~316 lines) — Types (`Model`, `Task`, message types), `Init`/`Update`/`View`, `formatTime`
  - `cmd/pomo/tasks.go` (~159 lines) — Task management (`activeTask`, `endActiveTask`, `startTask`, `recentTaskNames`), input handlers (`handleNamingInput`, `handleTaskInput`, `handleCategoryInput`)
  - `cmd/pomo/timer.go` — Phase logic (`phaseDuration`, `currentBreakDuration`, `generatePomodoroName`, `transitionToBreak`, `transitionToPomodoro`, `resetPomodoro`)
  - `cmd/pomo/config.go` — `Config` type, YAML config loading/writing, `processCategories`, `processWorkDays`
  - `cmd/pomo/cli.go` — Version vars, `parseArgsResult`, `reportArgs`, argument parsing, help text
  - `cmd/pomo/summary.go` (~245 lines) — Session summary output (terminal + Markdown file), `groupTasksByCategory`, `computeTaskTotals`, `buildFrontmatter`, `formatDurationHuman`
  - `cmd/pomo/db.go` — SQLite persistence (`dataPath`, `openDB`, `openDBAt`, `insertSession`); uses `modernc.org/sqlite` (pure Go, no CGO)
  - `cmd/pomo/report.go` — `pomo report` subcommand (`resolveReportWindow`, `queryWeeklyTotals`, `printWeeklyReport`, `writeWeeklyReportFile`, `runReport`)
  - `cmd/pomo/notification.go` (~29 lines) — Desktop notifications (macOS/Linux)
- **Bubbletea TUI Framework**: Elm Architecture pattern (Model / Update / View); uses `charm.land/bubbletea/v2` and `charm.land/bubbles/v2` (not the `github.com/charmbracelet` paths)
- **Bubbles Components**: Official `progress.Model` component for the gradient progress bar (`progress.WithDefaultBlend()`)

### Types

**`Config`** — User preferences loaded from `~/.config/pomo/config.yaml` (XDG-compliant):
- `PomodoroTime` (YAML: `pomodoro_time`, default `25`): Pomodoro (work) phase minutes
- `ShortBreakTime` (YAML: `short_break_time`, default `5`): Short break minutes
- `LongBreakTime` (YAML: `long_break_time`, default `10`): Long break minutes
- `PomodorosPerCycle` (YAML: `pomodoros_per_cycle`, default `4`): Number of pomodoros between long breaks
- `AutoStartWork` (YAML: `auto_start_work`, default `false`): Automatically start the next pomodoro when a break ends
- `SessionSummary` (YAML: `session_summary`, nested block via `SessionSummary` type):
  - `Create` (YAML: `create_session_summary`, default `true`): Whether to write a Markdown summary file on quit
  - `Folder` (YAML: `summary_folder`, default `"~/pomos"`): Where to write Markdown session summaries
  - `Tags` (YAML: `summary_tags`, default `["daily", "pomo summary"]`): Custom tags for the YAML frontmatter block. Set by `defaultConfig()` and preserved as the YAML unmarshal default when the field is omitted. An explicit `[]` in config omits the `tags` key in the frontmatter entirely.
- `Categories` (YAML: `categories`, default absent): Optional list of task category names. Absent or empty → category step is skipped entirely, preserving all prior behaviour. On first run, the config file is written with 6 built-in categories (`meeting`, `technical work`, `strategy work`, `meeting prep`, `121`, `chore`). At load time, entries are whitespace-trimmed and deduplicated; only the first 9 unique entries are used (excess logged to stderr).
- `WeeklyReport` (YAML: `weekly_report`, nested block via `WeeklyReport` type):
  - `WorkDays` (YAML: `work_days`, default `["Mon", "Tue", "Wed", "Thu", "Fri"]`): Which weekdays define the report window. Valid values: `Mon Tue Wed Thu Fri Sat Sun`. Invalid entries warn to stderr and are skipped; an empty result falls back to the Mon–Fri default.

**`Model`** — Central Bubbletea state:
- `pomodoroDuration`, `shortBreakDuration`, `longBreakDuration`, `pomodorosPerCycle`: Phase configuration
- `remaining`: Time left in current phase
- `isRest`: Distinguishes work (pomodoro) vs. break phase
- `paused`: Pause state
- `pomodorosCompleted`: Count of fully completed pomodoro+break cycles
- `totalWorked`, `totalRested`: Cumulative time for session summary
- `progress`: Bubbles progress bar component (pointer)
- `quitting`: Set on user quit; triggers summary output in `main()`
- `currentPomodoroName`: Display name for the current pomodoro (e.g. `"Morning #1"`)
- `namingMode` / `nameInput`: State for the inline pomodoro rename prompt
- `taskMode` / `taskInput`: State for the inline task name input prompt
- `recentTasks []string`: Recent unique task names populated when entering task mode; cleared on exit
- `categoryMode bool` / `pendingTask string`: State for the category picker (step 2 of task creation); `pendingTask` holds the confirmed task name from step 1 until a category is selected or the step is cancelled
- `categories []string`: Categories loaded from config, set in `main()` after parsing; nil/empty means category step is skipped
- `tasks []Task`: Append-only log of all task records for the session
- `startedAt`: Session start timestamp (used to name the summary file)
- `pomodoroStartedAt`: Timestamp when the current pomodoro began; used by `resetPomodoro()` to identify which tasks to discard

**`Task`** — Named unit of work:
- `Name string`: User-provided task name
- `Category string`: Category assigned at creation; empty string means uncategorised
- `StartedAt time.Time`: When this task entry began
- `EndedAt time.Time`: When it ended; zero value means currently active

**Message Types**:
- `tickMsg`: 1-second timer tick (drives the countdown)
- `finishedMsg`: Emitted when `remaining` hits 0 (triggers phase transition + notification)
- Built-in `tea.KeyPressMsg` and `tea.WindowSizeMsg`

### Key Functions

**Bubbletea lifecycle**:
- `Init()`: Returns `tickCmd()` to start the 1-second tick loop
- `Update()`: Dispatches on message type; delegates to `handleNamingInput()` / `handleTaskInput()` / `handleCategoryInput()` when those modes are active
- `View()`: Returns `tea.View` (via `tea.NewView()`); renders naming/task/category prompts, or the main timer UI (header, pomodoro name, countdown, progress bar, active task with optional `[category]` tag, key hints)
- `phaseDuration()`: Returns `pomodoroDuration` during work, or `currentBreakDuration()` during a break
- `currentBreakDuration()`: Returns `longBreakDuration` when the just-completed pomodoro is a multiple of `pomodorosPerCycle`, otherwise `shortBreakDuration`

**Phase transitions**:
- `transitionToBreak(elapsed, now)`: Accumulates `elapsed` work time, sets `isRest=true`, sets `remaining` to either the short or long break (via `currentBreakDuration()`), ends active task at `now`, clears `categoryMode`/`pendingTask`
- `transitionToPomodoro(elapsed, now)`: Accumulates `elapsed` rest time, increments `pomodorosCompleted`, generates new pomodoro name, continues last task name **and category** into a new Task entry
- `resetPomodoro()`: Resets the current pomodoro — restores `remaining` to `pomodoroDuration`, removes tasks started during this pomodoro (`StartedAt >= pomodoroStartedAt`), does NOT accumulate elapsed time into `totalWorked`, clears all input modes
- `generatePomodoroName(n, now)`: Produces `"Morning #N"` / `"Afternoon #N"` / `"Evening #N"` / `"Night #N"` based on time of day

**Task tracking** (pointer receivers, mutate in place):
- `activeTask()`: Returns pointer to the last task with a zero `EndedAt`, or nil
- `endActiveTask(at)`: Stamps `EndedAt` on the currently active task
- `startTask(name, at)`: Ends any active task, appends a new Task entry

**Input modes**:
- `handleNamingInput()`: Enter/Esc to commit/cancel; Backspace/Delete are UTF-8 rune-aware
- `handleTaskInput()`: Enter/Esc to commit/cancel; when `taskInput` is empty a digit `1`–`9` selects from `recentTasks` (if in range) or falls through to free-text; if categories are configured, confirming a name transitions to `categoryMode` instead of immediately starting a task
- `handleCategoryInput()`: Single-key selection — `[1]`–`[9]` pick a category, `[0]` assigns none, `[esc]` discards the pending task entirely (nothing added to task list)
- `recentTaskNames(tasks)`: Walks `tasks` in reverse, deduplicates, excludes the active task, returns at most 9 names most-recent-first

**CLI**:
- `parseArgs(args, cfg)`: Parses `--work`/`-w`, `--short-break`/`-s`, `--long-break`/`-L`, `--per-cycle`/`-c` flags; falls back to config values; validates each duration is positive and `--per-cycle ≥ 1`; handles `--help`/`-h`, `--version`/`-v`, and `report` subcommand; any unrecognised argument returns an error
- `parseReportArgs(args)`: Parses `--last`, `--from YYYY-MM-DD`, `--to YYYY-MM-DD`; validates mutual exclusion of `--last` and `--from`/`--to`; validates `--from ≤ --to`
- `parseDuration(arg)`: Accepts `"30"`, `"30m"`, `"30s"`; bare integer = minutes
- `showHelp()`: Prints usage, examples, and options including the `report` subcommand

**Output**:
- `printSummary(m)`: Prints pomodoros completed, total work/rest, and task time. When `m.categories` is non-empty, renders a grouped view (categories sorted by total duration desc, uncategorised last); otherwise renders the original flat list sorted alphabetically
- `writeSummaryFile(m, cfg)`: Writes the same content as a Markdown file to `cfg.SummaryFolder/<startedAt>.md`; skips if `cfg.SessionSummary.Create` is false; grouped format uses `**category** (duration)` headers when categories are configured
- `groupTasksByCategory(tasks)`: Returns `[]categoryGroup` (each with `category`, `total`, `tasks []taskSummary`), sorted by total desc with uncategorised (`""`) forced last; preserves per-category task insertion order
- `formatGroupHeader(category, total)`: Renders `"category  ──────  duration"` using `─` (U+2500) to fill a fixed 48-char line width
- `buildFrontmatter(tags, created)`: Builds the YAML frontmatter block (`---\ncreated: …\ntags:\n  - …\n---`) for the Markdown summary; omits `tags` key when the slice is empty
- `formatTime(d)`: Formats duration as `"MM:SS"` (hours overflow into minutes)
- `formatDurationHuman(d)`: Human-readable format (`"Xh Ym"`, `"Xm Ys"`, `"Xs"`)

**Config**:
- `defaultConfig()`: Returns the built-in default `Config` struct (used as unmarshal base and fallback); `Categories` is nil so an absent `categories` key in existing configs does not activate the category step
- `processCategories(cats)`: Trims whitespace, deduplicates (first-occurrence order), caps at 9 with a stderr warning; returns nil for empty input
- `processWorkDays(days)`: Trims whitespace, validates against the 7 weekday abbreviations, deduplicates; returns Mon–Fri default if result is empty
- `loadConfig()`: Reads YAML config; on first run, writes defaults plus `builtinCategories` to disk and returns that config; always runs `processCategories` and `processWorkDays` on the loaded slices before returning
- `configPath()`: Resolves `$XDG_CONFIG_HOME/pomo/config.yaml` or `~/.config/pomo/config.yaml`

**Database** (`~/.local/share/pomo/pomo.db`, XDG-compliant; pure Go SQLite via `modernc.org/sqlite`):
- `dataPath()`: Resolves `$XDG_DATA_HOME/pomo/pomo.db`, falling back to `~/.local/share/pomo/pomo.db`; creates the directory if absent
- `openDB()`: Gets path from `dataPath()`, delegates to `openDBAt`
- `openDBAt(path)`: Opens the SQLite file, runs `CREATE TABLE IF NOT EXISTS` for both tables, returns handle
- `insertSession(db, m)`: Wraps session row + all completed tasks in a single transaction; skips tasks where `EndedAt.IsZero()`
- Schema: `sessions` (1:1 with a pomo run) and `tasks` (linked by `session_id`); times stored as RFC3339 UTC strings; DB errors on quit are non-fatal warnings

**Report**:
- `resolveReportWindow(args, cfg, now)`: Computes `from`/`to` for the report; default = ISO Monday → last configured work day of current week; `--last` shifts back 7 days; `--from`/`--to` pass through (with end-of-day normalisation on `to`)
- `queryWeeklyTotals(db, from, to)`: `SELECT category, SUM(duration) … GROUP BY category ORDER BY duration DESC`; moves uncategorised (`""`) to last in Go after SQL sort
- `printWeeklyReport(totals, from, to)`: Prints header, 48-char separator, one row per category right-aligned to 48 chars, separator, total
- `writeWeeklyReportFile(totals, from, to, cfg)`: Writes `~/pomos/week-YYYY-MM-DD.md` (idempotent; overwrites on re-run); uses `buildFrontmatter(["weekly-report", "pomo summary"], from)`
- `runReport(cfg, args)`: Orchestrates `openDB` → `resolveReportWindow` → `queryWeeklyTotals` → `printWeeklyReport` → `writeWeeklyReportFile`; DB error here is fatal with a clear message pointing to the DB path

**Notifications**:
- `sendNotification(isRest, pomodoroName)`: macOS uses `terminal-notifier`; Linux uses `notify-send`; errors are non-fatal

### Command Line Interface

```
pomo [-w duration] [--short-break duration] [--long-break duration] [--per-cycle N] [--auto-start-work] [--create-session-summary|--no-create-session-summary] [-h|--help] [-v|--version]
pomo report [--last] [--from YYYY-MM-DD] [--to YYYY-MM-DD]
```

Duration formats: `30` (minutes), `30m`, `30s`. Defaults come from config file.

Validation: pomodoro > 0, short-break > 0, long-break > 0, per-cycle ≥ 1.

The `--create-session-summary` / `--no-create-session-summary` flags override `cfg.SessionSummary.Create` for that run only; omitting them leaves the config value unchanged. The override is stored as `*bool` in `parseArgsResult` (nil = use config).

Report flag validation: `--last` and `--from`/`--to` are mutually exclusive; `--to` requires `--from`; `--from` ≤ `--to`.

### Keyboard Controls

| Key | Effect |
|-----|--------|
| `Space` | Toggle pause/resume |
| `q` / `Ctrl+C` | Quit; print session summary |
| `n` | Enter naming mode (rename current pomodoro; pre-fills current name) |
| `a` | Enter task mode (add/switch task; work phase only) |
| `s` | Skip current phase immediately |
| `r` | Reset current pomodoro (discards elapsed time and pomodoro tasks; work phase only) |
| `Enter` | Confirm input (naming / task mode) |
| `Esc` | Cancel input (naming / task / category mode); at category step discards the pending task entirely |
| `Backspace` / `Delete` | Remove last character (UTF-8 rune-aware) |
| `0`–`9` | Category picker: `[1]`–`[9]` select a category, `[0]` assigns none (category mode only) |

### Testing Strategy

- Test functions split across `main_test.go`, `config_test.go`, `cli_test.go`, `summary_test.go`, `db_test.go`, `report_test.go` (white-box, `package main`)
- Covers: `parseDuration`, `formatTime`, `formatDurationHuman`, `parseArgs`, `--create-session-summary` flag, phase transitions, long-break selection (`TestTransitionToBreakUsesLongBreakAfterPerCycle`, `TestPhaseDurationLongBreak`), quit with partial progress, task tracking, naming/task input modes, config loading, pomodoro name generation, frontmatter generation, recent task picker, reset pomodoro
- DB tests: schema creation, round-trip session+task insert, active-task skipping, empty-task session
- Report tests: window resolution (current week, last week, explicit range, custom work days), weekly total aggregation, uncategorised-last sort, filename format
- Category behaviour degrades gracefully in all existing tests: models without `categories` set bypass `categoryMode` and exercise the original task flow unchanged

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
