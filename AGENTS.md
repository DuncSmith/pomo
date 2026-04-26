# 🤖 OpenCode Agent Guide (Pomo CLI)

## 🎯 Project Overview
A command-line Pomodoro timer built with Go and the Bubbletea TUI framework. It manages work/rest cycles, tracks user tasks, and generates detailed session summaries in Markdown format. The core logic follows the Elm Architecture pattern (Model -> Update -> View).

## ⚙️ High-Signal Developer Commands & Workflow
*   **Dependencies**: Always run `go mod tidy` to clean up dependencies before major changes.
*   **Build/Install**: Use `./pomo --version` and review `CLAUDE.md` for release steps. The standard build is:
    ```bash
    bin/build
    # This builds a versioned binary installed in ~/.local/bin
    ```
*   **Testing**: Testing is done with the standard Go toolchain:
    *   Run all tests: `go test`
    *   Verbose output: `go test -v`
    *   Specific test: `go test -run TestName`

## 🧠 Architecture & Workflow Quirks (Mistake Prevention)
1.  **Task/Naming Scope**: Task tracking (`a`) and interval naming (`n`) are *only* available/relevant during the **Work Phase**. They do not work or track anything meaningful during the Rest Phase.
2.  **Config Priority**: Duration parsing follows a specific hierarchy: Command Line Args > Config File Value > Hardcoded Defaults (e.g., WorkTime, IntervalTime).
3.  **Summary Generation**:
    *   The summary file is written in Markdown and must include YAML frontmatter (`tags`, `created`).
    *   Config path resolution uses XDG standards: `$XDG_CONFIG_HOME/pomo/config.yaml`.
4.  **Phase Transitions**: Be aware that state transitions (e.g., `transitionToRest`) are responsible for accumulating time and ending the active task, requiring careful handling of pointer receivers (`*Task`).

## 🐞 Common Pitfalls & Constraints
*   **Duration Input**: Accepts multiple formats: bare integer (minutes), `Xm`, or `Xs`. Parsing logic is complex and must be tested thoroughly.
*   **Summary Tags**: If the user explicitly provides empty tags in the YAML config (`summary_tags: []`), the `tags` key **must not** appear in the final frontmatter block; otherwise, it must resolve to default values (e.g., `["daily", "pomo summary"]`).
*   **Cross-Platform Notifications**: macOS uses `terminal-notifier`, and Linux uses `notify-send`. Implementations should be fire-and-forget and non-blocking.

## 💡 Summary
The codebase is mature, with strong separation of concerns (`main.go` for logic, `config.go`/`summary.go` for state/output). When modifying core functions, ensure that the Bubbletea lifecycle (Init/Update/View) remains consistent.