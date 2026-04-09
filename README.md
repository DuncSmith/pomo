# pomo

A simple, interactive CLI Pomodoro timer for the terminal built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

<img width="782" height="93" alt="image" src="https://github.com/user-attachments/assets/aebf55c4-b6f3-4cb6-95da-a9419a626410" />

Built using Claude Code.

## Features

- 🍅 **Interactive TUI** - Beautiful terminal interface with real-time progress visualization
- ⏸️ **Pause/Resume** - Press space to pause and resume your timer
- 📊 **Visual Progress** - Gradient progress bar that adapts to your terminal size
- 🔔 **Notifications** - Cross-platform system notifications when timers complete
- ⚡ **Fast & Lightweight** - Single binary with no external dependencies
- 🎨 **Responsive Design** - Automatically adjusts to your terminal width

## Installation

### Prerequisites
- Go 1.24.5 or later

### Build from source
```bash
git clone <repository-url>
cd pomo
go build -o pomo .
```

## Usage

### Basic Usage
```bash
# Start with defaults: 50m work, 10m rest (60m interval), repeating
./pomo

# Custom work duration (rest fills remainder of 60m interval)
./pomo 25            # 25m work, 35m rest
./pomo 30m           # 30m work, 30m rest
./pomo 90s           # 90s work, 58m30s rest

# Custom interval duration
./pomo 45 --interval 90   # 45m work, 45m rest (90m interval)
./pomo -i 90              # 50m work, 40m rest (90m interval)
```

The timer runs work → rest → work → rest continuously until you quit.
On exit, a session summary shows intervals completed and total time worked/rested.

### Keyboard Controls
- `Space` - Pause/resume timer
- `q` or `Ctrl+C` - Quit application

### Help
```bash
./pomo --help
```

## Duration Formats

The timer accepts flexible duration formats:
- `30` - 30 minutes (default unit)
- `25m` - 25 minutes
- `90s` - 90 seconds

## System Notifications

The application supports system notifications on completion:
- **macOS**: Uses `terminal-notifier` (install with `brew install terminal-notifier`)
- **Linux**: Uses `notify-send` (usually pre-installed)

## Development

### Running Tests
```bash
# Run all tests
go test

# Run tests with verbose output
go test -v

# Run a specific test
go test -run TestParseDuration
```

### Dependencies
```bash
# Download dependencies
go mod download

# Clean up dependencies
go mod tidy
```

## Technical Details

- Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI framework
- Uses [Bubbles](https://github.com/charmbracelet/bubbles) progress component
- Follows the Elm Architecture pattern (Model-Update-View)
- Cross-platform support (macOS, Linux, Windows)
