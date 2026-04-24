# Copilot Instructions

## Commands

```bash
go build -o pomo .        # Build
go test                   # Run all tests
go test -v                # Verbose tests
go test -run TestName     # Run a single test (e.g. TestParseDuration, TestFormatTime)
go mod tidy               # Clean up dependencies
```

## Architecture

Single-file Go app (`main.go`) using the [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI framework (Elm Architecture: Model-Update-View).

**Model** holds all state: `workDuration`, `restDuration`, `intervalDuration`, `remaining`, `isRest`, `paused`, `intervalsCompleted`, `totalWorked`, `totalRested`, `quitting`, and the Bubbles `progress.Model`.

**Message flow**: `Init()` starts a 1-second ticker → `tickMsg` decrements `remaining` each tick → when `remaining` reaches 0 a `finishedMsg` fires `sendNotification()` and transitions phases (work→rest→work…) until the user quits.

**CLI parsing** (`parseArgs()`): positional arg sets work duration (bare number = minutes, `m`/`s` suffixes supported); `--interval`/`-i` flag sets the total interval (default 60 min); rest = interval − work.

## Key Conventions

- Progress bar uses Bubbles `progress.Model` with gradient via `m.progress.ViewAs(float)`.
- Progress bar width is clamped to 20–80 characters, accounting for the time display and padding (`msg.Width - 4 - 20`).
- Notifications are fire-and-forget: errors print a debug hint but never crash the app.
- Default durations: **50 min** work, **10 min** rest (60-minute interval).
- On quit, elapsed time in the current phase is added to `totalWorked`/`totalRested` and a session summary is shown.
