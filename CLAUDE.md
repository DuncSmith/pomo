# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a simple Pomodoro timer CLI application built with Go using the Bubbletea TUI framework. The application provides an interactive terminal-based timer with visual progress bars, pause/resume functionality, and system notifications.

## Common Commands

### Build and Run
```bash
go build -o pomo .          # Build the executable
./pomo                      # Run with default 45-minute timer
./pomo 25m                  # Run 25-minute work timer
./pomo rest 5m              # Run 5-minute break timer
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
- `duration` and `remaining`: Timer durations
- `isRest`: Boolean to distinguish work/break timers  
- `paused`: Pause state
- `progress`: Bubbles progress bar component

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
- Help system with usage examples
- Work/rest mode detection via `rest` command

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