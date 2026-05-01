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

### Download a release

Pre-built binaries are available on the [Releases](https://github.com/DuncSmith/pomo/releases) page for Linux, macOS, and Windows (amd64/arm64).

Download the archive for your platform, extract it, and place `pomo` in your `$PATH`.

### Build and install from source

```bash
git clone https://github.com/DuncSmith/pomo.git
cd pomo
bin/build
```

`bin/build` compiles `pomo` with the correct version info from the latest git tag and installs it to `~/.local/bin`. Make sure `~/.local/bin` is in your `$PATH`.

## Usage

### Basic Usage
```bash
# Start with defaults: 50m work (10m rest), repeating
./pomo

# Custom work duration (rest = 60m interval − work)
./pomo 25            # 25m work (35m rest)
./pomo 30m           # 30m work (30m rest)
./pomo 90s           # 90s work (58m30s rest)

# Custom interval duration (rest = interval − work)
./pomo 45 --interval 90   # 45m work, 90m interval (45m rest)
./pomo -i 90              # 50m work, 90m interval (40m rest)
```

Rest time is always derived as **interval − work** and cannot be set directly.

The timer runs work → rest → work → rest continuously until you quit.
On exit, a session summary shows:
- Intervals completed
- Total time worked and rested  
- Breakdown of work time by interval name (if named)
- Detailed time breakdown for each task tracked

### Keyboard Controls
- `Space` - Pause/resume timer
- `q` or `Ctrl+C` - Quit application
- `n` - Name current work interval (press again to rename)
- `a` - Add or switch current task (work phase only)
- `s` - Skip current phase (work → rest or rest → work)
- `l` - Take a lunch break (work phase only)

### Interval Naming
During work phases, press `n` to give a name to your current interval. This name will be:
- Saved in your session summary
- Inherited by subsequent intervals (so you don't need to rename every time)
- Shown in the session summary with tracked time per name

Example usage:
```bash
# Start a timer
./pomo 25

# During work phase, press 'n' and type "Email cleanup"
# The interval will be named "Email cleanup"
# After break, next work interval will inherit "Email cleanup"
# Press 'n' again to rename if needed
```

### Lunch Breaks

During a work phase, press `l` to take a lunch break. This ends the current work interval early and starts a special lunch timer (default: 60 minutes).

When the lunch timer expires, the app pauses and waits — it won't automatically start the next interval.

Press `Enter` or `Space` when you're ready and a new work interval begins, with your previous task automatically resumed.

You can also skip lunch early with `s`, or pause/resume it with `Space` like any other phase.

**Configure the lunch duration** in `~/.config/pomo/config.yaml`:

```yaml
lunch_time: 60   # minutes (default: 60)
```

### Tracking Tasks

During work phases, press `a` to name your current task or switch to a different one.

- **First task of the session:** a blank prompt appears — type a name and press Enter.
- **With previous tasks:** a numbered list of recently-used task names appears. Press the corresponding number key to resume that task instantly, or start typing to create a new one.

```
Recent tasks:
  1  Email cleanup
  2  Code review

[1-9] resume   [type] new task   [esc] cancel
```

- Tasks are automatically ended when a rest phase begins or when you switch to a new task.
- When a new work interval starts after a break, the previous task name is carried over automatically.
- Your session summary includes a breakdown of time spent on each task.

### Session Summaries

When you quit `pomo`, it prints a terminal summary and optionally writes a Markdown file for your records.

**Default behavior:** A Markdown summary is written to `~/pomos/<YYYY-MM-DD_HH-MM-SS>.md`.

Each summary file includes a YAML frontmatter block at the top:
```yaml
---
tags:
  - daily
  - pomo summary
created: 2026-04-17
---
```

The summary contains:
- Intervals completed
- Total time worked and rested
- Per-task time breakdown (if tasks were tracked)

**Configuration:** Add a `session_summary` block to your config file (`~/.config/pomo/config.yaml`):

```yaml
work_time: 50
interval_time: 60
lunch_time: 60
session_summary:
  create_session_summary: true
  summary_folder: ~/pomos
  summary_tags:
    - daily
    - pomo summary
```

| Option | Description |
|--------|-------------|
| `create_session_summary` | Whether to write a Markdown file on quit (default: `true`) |
| `summary_folder` | Directory for summary files (default: `~/pomos`) |
| `summary_tags` | Custom tags for the frontmatter block. Omit to use defaults. Set to `[]` to omit the `tags` key entirely. |

You can also toggle summaries from the command line:
```bash
./pomo --no-create-session-summary   # Disable the Markdown summary for this run
./pomo --create-session-summary      # Explicitly enable (usually the default)
```

### Help
```bash
./pomo --help
./pomo --version
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

## Releasing

This project uses [GoReleaser](https://goreleaser.com/) with GitHub Actions to produce cross-compiled binaries.

To create a new release, use the helper script:

```bash
bin/release patch    # bump patch version (v0.1.0 → v0.1.1)
bin/release minor    # bump minor version (v0.1.0 → v0.2.0)
bin/release major    # bump major version (v0.1.0 → v1.0.0)
bin/release v1.2.3   # set explicit version
```

The script validates your working tree, bumps the version, creates an annotated tag, and pushes it. The release workflow then automatically builds binaries for all platforms, generates a changelog, and creates a GitHub Release with attached artifacts.

## Technical Details

- Built with [Bubbletea](https://github.com/charmbracelet/bubbletea) TUI framework
- Uses [Bubbles](https://github.com/charmbracelet/bubbles) progress component
- Follows the Elm Architecture pattern (Model-Update-View)
- Cross-platform support (macOS, Linux, Windows)
