# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a simple Pomodoro timer CLI application built with Go using the Bubbletea TUI framework. The application provides an interactive terminal-based timer with visual progress bars, pause/resume functionality, and system notifications.

## Common Commands

### Build and Run
```bash
go build -o pomo .          # Build the executable
./pomo                      # Run with default 50m work / 10m rest interval
./pomo 25                   # Run 25m work / 35m rest interval
./pomo 45 --interval 90    # Run 45m work / 45m rest (90m interval)
./pomo --help               # Show usage information
```

### Testing
```bash
go test                     # Run all tests
go test -v                  # Run tests with verbose output
go test -run TestName       # Run a specific test
```

### Dependencies
```bash
go mod tidy                 # Clean up dependencies
go mod download             # Download dependencies
```

## Architecture

### Core Structure
- **Single-file architecture**: All code is in `main.go` with a focused, minimal design
- **Bubbletea TUI Framework**: Uses the Elm Architecture pattern (Model-Update-View)
- **Bubbles Components**: Uses the official progress bar component for visual feedback

### Key Components

**Model struct**: Central state containing:
- `workDuration`, `restDuration`, `intervalDuration`: Interval configuration
- `remaining`: Time left in current phase
- `isRest`: Boolean to distinguish work/break phases
- `paused`: Pause state
- `intervalsCompleted`: Count of fully completed work+rest cycles
- `totalWorked`, `totalRested`: Cumulative time tracking for session summary
- `progress`: Bubbles progress bar component
- `quitting`: Whether user initiated quit

**Message Types**:
- `tickMsg`: Timer tick events (every second)
- `finishedMsg`: Timer completion events
- Built-in `tea.KeyMsg` and `tea.WindowSizeMsg` for input/resize handling

**Core Functions**:
- `Init()`: Starts the timer ticker
- `Update()`: Handles all events (keyboard, timer ticks, window resize)
- `View()`: Renders the TUI with progress bar and time display

### Command Line Interface
- Argument parsing in `parseArgs()` supports duration formats: `30`, `30m`, `30s`
- Supports `--interval` / `-i` flag to override default 60-minute interval
- Help system with usage examples
- Validates work > 0, interval > 0, work < interval
- Rest duration is computed as interval - work

### Testing Strategy
- Comprehensive unit tests for utility functions: `parseDuration()`, `formatTime()`, `createProgressBar()`
- Test coverage includes edge cases, error conditions, and various input formats
- The custom `createProgressBar()` function is maintained for test compatibility alongside the Bubbles progress component

### Notifications
- Cross-platform system notifications via `terminal-notifier` (macOS) and `notify-send` (Linux)
- Graceful fallback when notification tools are unavailable

### Progress Bar Implementation
- Uses Bubbles progress component with gradient styling
- Responsive width handling (20-80 characters) based on terminal size
- Maintains legacy `createProgressBar()` function for existing tests