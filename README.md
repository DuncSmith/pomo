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
- 🏷️ **Task Categories** - Assign categories to tasks and see time grouped by category in your session summary
- 📅 **Weekly Reports** - Query a category-time summary for any week with `pomo report`

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
# Start with defaults: 25m pomodoro / 5m short break / 10m long break / 4 per cycle
./pomo

# Custom pomodoro duration
./pomo -w 50             # 50m pomodoro
./pomo -w 30m            # 30m pomodoro
./pomo -w 90s            # 90s pomodoro

# Custom break durations and cycle length
./pomo -w 50 -s 10 -l 20 # 50m pomodoro, 10m short break, 20m long break
./pomo -c 3              # long break every 3rd pomodoro

# Auto-start the next pomodoro when a break ends
./pomo --auto-start-work
```

The timer runs **pomodoro → short break** repeatedly. After every `N`
pomodoros (default `4`), the break is a **long break** instead. The cycle
continues until you quit.

By default, when a break ends, pomo waits for a keypress before starting the next pomodoro.
Use `--auto-start-work` (or config below) to start the next pomodoro immediately when a break ends.

On exit, a session summary shows:
- Pomodoros completed
- Total time worked and rested
- Detailed time breakdown for each task tracked

### Keyboard Controls
- `Space` - Pause/resume timer
- `q` or `Ctrl+C` - Quit application
- `n` - Rename the current pomodoro (press again to rename)
- `a` - Add or switch current task (work phase only)
- `s` - Skip current phase (pomodoro → break or break → next pomodoro)
- `r` - Reset current pomodoro: restores full pomodoro duration and discards all tasks started in this pomodoro (work phase only)

### Pomodoro Naming

Each pomodoro is automatically named based on the time of day — e.g. "Morning #1", "Afternoon #2". This name is shown in the timer header.

During a pomodoro, press `n` to rename it. The prompt pre-fills the existing name; press Enter to confirm or Esc to cancel.

Each new pomodoro gets a fresh auto-generated name. Renaming one pomodoro does not carry the name forward to subsequent pomodoros.

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

If categories are configured (see below), a second step appears immediately after the task name is confirmed:

```
New task: write up quarterly notes

Category:
[1] meeting             [2] technical work   [3] strategy work
[4] meeting prep        [5] 121              [6] chore
[0] none

[esc] cancel
```

Press a single key to select — no Enter needed. Press `0` for no category, or `Esc` to discard the task entirely.

The active task is shown in the timer view. If it has a category, the category appears inline:

```
Task: write up quarterly notes  [strategy work]
```

- Tasks are automatically ended when a break begins or when you switch to a new task.
- When a new pomodoro starts after a break, the previous task name **and category** are carried over automatically.
- Your session summary includes a breakdown of time spent on each task.

### Weekly Reports

After accumulating sessions, run `pomo report` to see a category-time breakdown for the current work week:

```
Weekly report  Mon 28 Apr – Fri 2 May
────────────────────────────────────────────────
technical work                          3h 20m
strategy work                           1h 45m
meeting                                 1h 10m
meeting prep                              25m 0s
uncategorised                             15m 0s
────────────────────────────────────────────────
Total                                   6h 55m
```

The report is also written to `~/pomos/week-YYYY-MM-DD.md` (the date is the Monday of the week), overwriting any previous run for the same week.

```bash
pomo report                              # current work week
pomo report --last                       # previous work week
pomo report --from 2025-04-21 --to 2025-04-25   # explicit date range
pomo report --from 2025-04-21            # from that date to today
```

**Session data** is stored automatically in `~/.local/share/pomo/pomo.db` every time you quit `pomo`. A Markdown summary is also written to `~/pomos/` on each quit.

**Configure the work week** in `~/.config/pomo/config.yaml` (see below for how to create it):

```yaml
weekly_report:
  work_days: [Mon, Tue, Wed, Thu, Fri]   # default; any subset of Mon–Sun
```

### Session Summaries

When you quit `pomo`, it prints a terminal summary and optionally writes a Markdown file for your records.

**Default behavior:** A Markdown summary is written to `~/pomos/<YYYY-MM-DD_HH-MM-SS>.md`.

Each summary file includes a YAML frontmatter block at the top:
```yaml
---
created: 2026-04-17
title: 2026-04-17
tags:
  - daily
  - pomo summary
---
```

The summary contains:
- Pomodoros completed
- Total time worked and rested
- Per-task time breakdown (if tasks were tracked)

When categories are configured, the task section is grouped by category, sorted by total time descending, with uncategorised tasks at the end:

```
  strategy work  ──────────────────  50m 0s
    write up quarterly notes          25m 0s
    review roadmap doc                25m 0s

  technical work  ─────────────────  25m 0s
    fix auth bug                      25m 0s

  uncategorised  ──────────────────  10m 0s
    random admin                      10m 0s
```

Without categories configured, the original flat task list is shown.

**Configuration:** Add a `session_summary` block to `~/.config/pomo/config.yaml` (run `pomo --init` to create the file with all defaults pre-filled):

```yaml
pomodoro_time: 25
short_break_time: 5
long_break_time: 10
pomodoros_per_cycle: 4
auto_start_work: false
session_summary:
  create_session_summary: true
  summary_folder: ~/pomos
  summary_tags:
    - daily
    - pomo summary
categories:
  - meeting
  - technical work
  - strategy work
  - meeting prep
  - 121
  - chore
weekly_report:
  work_days: [Mon, Tue, Wed, Thu, Fri]
```

| Option | Description |
|--------|-------------|
| `pomodoro_time` | Pomodoro (work) duration in minutes (default: `25`) |
| `short_break_time` | Short break duration in minutes (default: `5`) |
| `long_break_time` | Long break duration in minutes (default: `10`) |
| `pomodoros_per_cycle` | Number of pomodoros between long breaks (default: `4`) |
| `auto_start_work` | Automatically start the next pomodoro as soon as a break ends (default: `false`) |
| `create_session_summary` | Whether to write a Markdown file on quit (default: `true`) |
| `summary_folder` | Directory for summary files (default: `~/pomos`) |
| `summary_tags` | Custom tags for the frontmatter block. Omit to use defaults. Set to `[]` to omit the `tags` key entirely. |
| `categories` | List of task categories (up to 9). Omit or leave empty to disable the category step entirely. New installs include the defaults shown above. |
| `weekly_report.work_days` | Weekdays that define the report window (default: `[Mon, Tue, Wed, Thu, Fri]`). Valid values: `Mon Tue Wed Thu Fri Sat Sun`. |

You can also toggle summaries from the command line:
```bash
./pomo --no-create-session-summary   # Disable the Markdown summary for this run
./pomo --create-session-summary      # Explicitly enable (usually the default)
```

### Configuration

`pomo` works out of the box with built-in defaults — no config file needed. To customise the defaults, create a config file with:

```bash
pomo --init
```

This writes `~/.config/pomo/config.yaml` with all options pre-filled. The command is safe to run multiple times — it won't overwrite an existing file.

Edit the file to change any values; `pomo` picks them up on the next run.

### Help
```bash
./pomo --help
./pomo --version
```

## Duration Formats

The `-w`/`--work` flag (and `-s`, `-l`) accept flexible duration formats:
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
