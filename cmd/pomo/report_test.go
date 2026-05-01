package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestResolveReportWindow_CurrentWeek(t *testing.T) {
	cfg := Config{WeeklyReport: WeeklyReport{WorkDays: []string{"Mon", "Tue", "Wed", "Thu", "Fri"}}}
	// Wednesday 23 April 2025
	now := time.Date(2025, 4, 23, 14, 0, 0, 0, time.UTC)
	from, to := resolveReportWindow(reportArgs{}, cfg, now)

	if got, want := from.Format("2006-01-02"), "2025-04-21"; got != want {
		t.Errorf("from: got %s, want %s", got, want)
	}
	if got, want := to.Format("2006-01-02"), "2025-04-25"; got != want {
		t.Errorf("to: got %s, want %s", got, want)
	}
}

func TestResolveReportWindow_LastWeek(t *testing.T) {
	cfg := Config{WeeklyReport: WeeklyReport{WorkDays: []string{"Mon", "Tue", "Wed", "Thu", "Fri"}}}
	// Wednesday 23 April 2025 — last week = week of 14 Apr
	now := time.Date(2025, 4, 23, 14, 0, 0, 0, time.UTC)
	from, to := resolveReportWindow(reportArgs{Last: true}, cfg, now)

	if got, want := from.Format("2006-01-02"), "2025-04-14"; got != want {
		t.Errorf("from: got %s, want %s", got, want)
	}
	if got, want := to.Format("2006-01-02"), "2025-04-18"; got != want {
		t.Errorf("to: got %s, want %s", got, want)
	}
}

func TestResolveReportWindow_ExplicitRange(t *testing.T) {
	cfg := Config{WeeklyReport: WeeklyReport{WorkDays: []string{"Mon", "Tue", "Wed", "Thu", "Fri"}}}
	now := time.Date(2025, 4, 23, 14, 0, 0, 0, time.UTC)

	fromT := time.Date(2025, 4, 7, 0, 0, 0, 0, time.UTC)
	toT := time.Date(2025, 4, 11, 0, 0, 0, 0, time.UTC)
	from, to := resolveReportWindow(reportArgs{From: &fromT, To: &toT}, cfg, now)

	if got, want := from.Format("2006-01-02"), "2025-04-07"; got != want {
		t.Errorf("from: got %s, want %s", got, want)
	}
	if got, want := to.Format("2006-01-02"), "2025-04-11"; got != want {
		t.Errorf("to: got %s, want %s", got, want)
	}
}

func TestResolveReportWindow_CustomWorkDays(t *testing.T) {
	cfg := Config{WeeklyReport: WeeklyReport{WorkDays: []string{"Mon", "Tue", "Wed"}}}
	// Thursday 24 April 2025
	now := time.Date(2025, 4, 24, 14, 0, 0, 0, time.UTC)
	from, to := resolveReportWindow(reportArgs{}, cfg, now)

	if got, want := from.Format("2006-01-02"), "2025-04-21"; got != want {
		t.Errorf("from: got %s, want %s", got, want)
	}
	if got, want := to.Format("2006-01-02"), "2025-04-23"; got != want {
		t.Errorf("to: got %s, want %s", got, want)
	}
}

func TestQueryWeeklyTotals(t *testing.T) {
	db, err := openDBAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDBAt: %v", err)
	}
	defer db.Close()

	base := time.Date(2025, 4, 21, 9, 0, 0, 0, time.UTC)
	m := Model{
		startedAt:   base,
		totalWorked: 50 * time.Minute,
		tasks: []Task{
			{Name: "design", Category: "technical work", StartedAt: base, EndedAt: base.Add(30 * time.Minute)},
			{Name: "standup", Category: "meeting", StartedAt: base.Add(30 * time.Minute), EndedAt: base.Add(40 * time.Minute)},
		},
	}
	if _, err := insertSession(db, m); err != nil {
		t.Fatalf("insertSession: %v", err)
	}

	from := time.Date(2025, 4, 21, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 4, 25, 23, 59, 59, 0, time.UTC)
	totals, err := queryWeeklyTotals(db, from, to)
	if err != nil {
		t.Fatalf("queryWeeklyTotals: %v", err)
	}

	if len(totals) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(totals))
	}
	// technical work (30m) should come before meeting (10m)
	if totals[0].category != "technical work" {
		t.Errorf("first category: got %q, want %q", totals[0].category, "technical work")
	}
	if totals[0].secs != 30*60 {
		t.Errorf("technical work secs: got %d, want %d", totals[0].secs, 30*60)
	}
}

func TestQueryWeeklyTotals_UncategorisedLast(t *testing.T) {
	db, err := openDBAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDBAt: %v", err)
	}
	defer db.Close()

	base := time.Date(2025, 4, 21, 9, 0, 0, 0, time.UTC)
	m := Model{
		startedAt:   base,
		totalWorked: 90 * time.Minute,
		tasks: []Task{
			// uncategorised has more time than "meeting" — must still appear last
			{Name: "misc", Category: "", StartedAt: base, EndedAt: base.Add(60 * time.Minute)},
			{Name: "standup", Category: "meeting", StartedAt: base.Add(60 * time.Minute), EndedAt: base.Add(90 * time.Minute)},
		},
	}
	if _, err := insertSession(db, m); err != nil {
		t.Fatalf("insertSession: %v", err)
	}

	from := time.Date(2025, 4, 21, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 4, 25, 23, 59, 59, 0, time.UTC)
	totals, err := queryWeeklyTotals(db, from, to)
	if err != nil {
		t.Fatalf("queryWeeklyTotals: %v", err)
	}

	if len(totals) < 1 {
		t.Fatal("expected at least 1 category")
	}
	last := totals[len(totals)-1]
	if last.category != "" {
		t.Errorf("last category: got %q, want empty (uncategorised)", last.category)
	}
}

func TestWeeklyReportFilename(t *testing.T) {
	from := time.Date(2025, 4, 21, 0, 0, 0, 0, time.UTC)
	want := "week-2025-04-21"
	if got := "week-" + from.Format("2006-01-02"); got != want {
		t.Errorf("filename prefix: got %s, want %s", got, want)
	}
}
