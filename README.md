# pomo

A simple, interactive Pomodoro timer for the terminal built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

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
# Start a 45-minute work session (default)
./pomo

# Custom work duration
./pomo 25m          # 25 minutes
./pomo 30           # 30 minutes (defaults to minutes)
./pomo 90s          # 90 seconds

# Break timers
./pomo rest         # 15-minute break (default)
./pomo rest 5m      # 5-minute break
```

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
