package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestOpenDB_CreatesSchema(t *testing.T) {
	db, err := openDBAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDBAt: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"sessions", "tasks"} {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name); err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestInsertSession_RoundTrip(t *testing.T) {
	db, err := openDBAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDBAt: %v", err)
	}
	defer db.Close()

	now := time.Date(2025, 4, 21, 9, 0, 0, 0, time.UTC)
	m := Model{
		startedAt:   now,
		totalWorked: 50 * time.Minute,
		totalRested: 10 * time.Minute,
		tasks: []Task{
			{Name: "task one", Category: "technical work", StartedAt: now, EndedAt: now.Add(25 * time.Minute)},
			{Name: "task two", Category: "", StartedAt: now.Add(25 * time.Minute), EndedAt: now.Add(50 * time.Minute)},
		},
	}

	id, err := insertSession(db, m)
	if err != nil {
		t.Fatalf("insertSession: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero session id")
	}

	var taskCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE session_id = ?`, id).Scan(&taskCount); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if taskCount != 2 {
		t.Errorf("task count: got %d, want 2", taskCount)
	}

	var workSecs, restSecs int64
	if err := db.QueryRow(`SELECT work_secs, rest_secs FROM sessions WHERE id = ?`, id).Scan(&workSecs, &restSecs); err != nil {
		t.Fatalf("query session: %v", err)
	}
	if want := int64(50 * 60); workSecs != want {
		t.Errorf("work_secs: got %d, want %d", workSecs, want)
	}
	if want := int64(10 * 60); restSecs != want {
		t.Errorf("rest_secs: got %d, want %d", restSecs, want)
	}
}

func TestInsertSession_SkipsActiveTask(t *testing.T) {
	db, err := openDBAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDBAt: %v", err)
	}
	defer db.Close()

	now := time.Date(2025, 4, 21, 9, 0, 0, 0, time.UTC)
	m := Model{
		startedAt:   now,
		totalWorked: 20 * time.Minute,
		tasks: []Task{
			{Name: "finished", Category: "", StartedAt: now, EndedAt: now.Add(20 * time.Minute)},
			{Name: "active", Category: "", StartedAt: now.Add(20 * time.Minute)}, // zero EndedAt
		},
	}

	id, err := insertSession(db, m)
	if err != nil {
		t.Fatalf("insertSession: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE session_id = ?`, id).Scan(&count); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 task (active skipped), got %d", count)
	}
}

func TestInsertSession_EmptyTasks(t *testing.T) {
	db, err := openDBAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("openDBAt: %v", err)
	}
	defer db.Close()

	now := time.Date(2025, 4, 21, 9, 0, 0, 0, time.UTC)
	m := Model{startedAt: now, tasks: []Task{}}

	id, err := insertSession(db, m)
	if err != nil {
		t.Fatalf("insertSession: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE session_id = ?`, id).Scan(&count); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 tasks, got %d", count)
	}
}
