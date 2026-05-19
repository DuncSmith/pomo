# Single-session design

Pomo is intentionally a single-session tool: one process per work-and-rest period, with no cross-session aggregation built in. The SQLite database at `~/.local/share/pomo/pomo.db` and the `pomo report` subcommand were considered and removed because the user reported never running a weekly report in practice; the per-session Markdown files in `~/pomos/<startedAt>.md` are a sufficient historical record, and any future weekly view can be produced ad-hoc with `grep`/`awk` over those files rather than carrying a database, a query layer, a config block, and a CLI subcommand in-tree.

## Consequences

- The `categories` feature (modal picker, config key, built-in list, grouped Markdown summary) was removed at the same time. Its only cross-session payoff was the grouped weekly report; without that, categories did not earn the interface cost of an extra step on every task creation.
- The `weekly_report.work_days` config block was removed.
- The `modernc.org/sqlite` dependency was dropped.
- Existing data in `~/.local/share/pomo/pomo.db` is orphaned. Users who care about it can keep the file; nothing in the codebase reads it.

## Reversal

If cross-session aggregation later becomes load-bearing, the cheaper reinstatement is to read `~/pomos/*.md` rather than re-introducing SQLite. Reach for a database again only if the Markdown-grep approach proves insufficient — that's a real signal, not a default.
