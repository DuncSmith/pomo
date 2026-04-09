# Copilot Instructions

## Commands

```bash
go build -o pomo .        # Build
go test                   # Run all tests
go test -v                # Verbose tests
go test -run TestName     # Run a single test (e.g. TestParseDuration, TestFormatTime, TestCreateProgressBar)
go mod tidy               # Clean up dependencies
```

## Architecture

Single-file Go app (`main.go`) using the [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI framework (Elm Architecture: Model-Update-View).

**Model** holds all state: `duration`, `remaining`, `isRest`, `paused`, and the Bubbles `progress.Model`.

**Message flow**: `Init()` starts a 1-second ticker → `tickMsg` decrements `remaining` each tick → when `remaining` reaches 0 a `finishedMsg` fires `sendNotification()` and quits.

**CLI parsing** (`parseArgs()`): bare number defaults to minutes, `m`/`s` suffixes override. The `rest` subcommand sets `isRest=true` with a 15-minute default.

## Key Conventions

- `createProgressBar()` (ASCII `█`/`░`) is kept alongside the Bubbles progress component solely for test compatibility — do not remove it.
- The Bubbles `progress.Model` is the one actually rendered; `createProgressBar()` is never called at runtime.
- Progress bar width is clamped to 20–80 characters, accounting for the time display and padding (`msg.Width - 4 - 20`).
- Notifications are fire-and-forget: errors print a debug hint but never crash the app.
- Default durations: **45 min** (work), **15 min** (rest).
