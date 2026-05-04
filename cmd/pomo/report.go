package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type categoryTotal struct {
	category string
	secs     int64
}

var weekdayAbbrevs = map[string]time.Weekday{
	"Mon": time.Monday,
	"Tue": time.Tuesday,
	"Wed": time.Wednesday,
	"Thu": time.Thursday,
	"Fri": time.Friday,
	"Sat": time.Saturday,
	"Sun": time.Sunday,
}

// isoMondayOf returns midnight on the Monday of the ISO week containing t.
func isoMondayOf(t time.Time) time.Time {
	wd := t.Weekday()
	daysBack := (int(wd) + 6) % 7 // Mon=0, Tue=1, ..., Sun=6
	d := t.AddDate(0, 0, -daysBack)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
}

// resolveReportWindow returns the from/to time range for the report.
// Default: current ISO work week (Mon through last configured work day).
// --last: shifts back one week.
// --from/--to: explicit range (to defaults to end of today when omitted).
func resolveReportWindow(args reportArgs, cfg Config, now time.Time) (from, to time.Time) {
	if args.From != nil {
		from = time.Date(args.From.Year(), args.From.Month(), args.From.Day(), 0, 0, 0, 0, now.Location())
		if args.To != nil {
			to = time.Date(args.To.Year(), args.To.Month(), args.To.Day(), 23, 59, 59, 0, now.Location())
		} else {
			to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		}
		return
	}

	ref := now
	if args.Last {
		ref = ref.AddDate(0, 0, -7)
	}

	monday := isoMondayOf(ref)

	minOffset, maxOffset := 7, -1
	for _, day := range cfg.WeeklyReport.WorkDays {
		wd, ok := weekdayAbbrevs[day]
		if !ok {
			continue
		}
		offset := (int(wd) + 6) % 7 // Mon=0, Tue=1, ..., Sun=6
		if offset < minOffset {
			minOffset = offset
		}
		if offset > maxOffset {
			maxOffset = offset
		}
	}
	if minOffset > maxOffset {
		minOffset, maxOffset = 0, 4 // fallback Mon–Fri
	}

	fromDay := monday.AddDate(0, 0, minOffset)
	toDay := monday.AddDate(0, 0, maxOffset)

	from = time.Date(fromDay.Year(), fromDay.Month(), fromDay.Day(), 0, 0, 0, 0, monday.Location())
	to = time.Date(toDay.Year(), toDay.Month(), toDay.Day(), 23, 59, 59, 0, monday.Location())
	return
}

// queryWeeklyTotals queries per-category time totals for the given window.
// The uncategorised category ("") is sorted last in Go after the SQL ORDER BY.
func queryWeeklyTotals(db *sql.DB, from, to time.Time) ([]categoryTotal, error) {
	rows, err := db.Query(`
		SELECT category, SUM(strftime('%s', ended_at) - strftime('%s', started_at)) AS total_secs
		FROM tasks
		WHERE started_at >= ? AND ended_at <= ?
		GROUP BY category
		ORDER BY total_secs DESC`,
		from.UTC().Format(time.RFC3339),
		to.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totals []categoryTotal
	for rows.Next() {
		var ct categoryTotal
		if err := rows.Scan(&ct.category, &ct.secs); err != nil {
			return nil, err
		}
		totals = append(totals, ct)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// SQL already ordered by duration desc; move uncategorised to last.
	sort.SliceStable(totals, func(i, j int) bool {
		if totals[i].category == "" {
			return false
		}
		if totals[j].category == "" {
			return true
		}
		return false
	})

	return totals, nil
}

const reportWidth = 48

func reportLine(name, durStr string) string {
	padding := max(reportWidth - len(name) - len(durStr), 1)
	return name + strings.Repeat(" ", padding) + durStr
}

func printWeeklyReport(totals []categoryTotal, from, to time.Time) {
	sep := strings.Repeat("─", reportWidth)
	fmt.Printf("\nWeekly report  %s – %s\n", from.Format("Mon 2 Jan"), to.Format("Mon 2 Jan"))
	fmt.Println(sep)

	var grandTotal int64
	for _, ct := range totals {
		grandTotal += ct.secs
		name := ct.category
		if name == "" {
			name = "uncategorised"
		}
		fmt.Println(reportLine(name, formatDurationHuman(time.Duration(ct.secs)*time.Second)))
	}

	fmt.Println(sep)
	fmt.Println(reportLine("Total", formatDurationHuman(time.Duration(grandTotal)*time.Second)))
}

func writeWeeklyReportFile(totals []categoryTotal, from, to time.Time, cfg Config) error {
	pomosDir := cfg.SessionSummary.Folder
	if strings.HasPrefix(pomosDir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %w", err)
		}
		pomosDir = filepath.Join(homeDir, pomosDir[2:])
	}
	if err := os.MkdirAll(pomosDir, 0755); err != nil {
		return fmt.Errorf("could not create summary directory: %w", err)
	}

	filename := "week-" + from.Format("2006-01-02") + ".md"
	filePath := filepath.Join(pomosDir, filename)

	sep := strings.Repeat("─", reportWidth)
	var sb strings.Builder
	sb.WriteString(buildFrontmatter([]string{"weekly-report", "pomo summary"}, from))
	sb.WriteString(fmt.Sprintf("# Weekly Report — %s – %s\n\n", from.Format("Mon 2 Jan"), to.Format("Mon 2 Jan")))
	sb.WriteString(sep + "\n")

	var grandTotal int64
	for _, ct := range totals {
		grandTotal += ct.secs
		name := ct.category
		if name == "" {
			name = "uncategorised"
		}
		sb.WriteString(reportLine(name, formatDurationHuman(time.Duration(ct.secs)*time.Second)) + "\n")
	}

	sb.WriteString(sep + "\n")
	sb.WriteString(reportLine("Total", formatDurationHuman(time.Duration(grandTotal)*time.Second)) + "\n")

	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("could not write weekly report file: %w", err)
	}
	fmt.Printf("\n  Weekly report saved to %s/%s\n", cfg.SessionSummary.Folder, filename)
	return nil
}

func runReport(cfg Config, args reportArgs) {
	db, err := openDB()
	if err != nil {
		path, _ := dataPath()
		fmt.Fprintf(os.Stderr, "error: could not open database at %s: %v\n", path, err)
		os.Exit(1)
	}
	defer db.Close()

	from, to := resolveReportWindow(args, cfg, time.Now())

	totals, err := queryWeeklyTotals(db, from, to)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: could not query weekly totals: %v\n", err)
		os.Exit(1)
	}

	printWeeklyReport(totals, from, to)

	if err := writeWeeklyReportFile(totals, from, to, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write weekly report file: %v\n", err)
	}
}
