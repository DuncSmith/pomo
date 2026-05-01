package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const dbSchema = `
CREATE TABLE IF NOT EXISTS sessions (
    id         INTEGER PRIMARY KEY,
    started_at DATETIME NOT NULL,
    ended_at   DATETIME NOT NULL,
    work_secs  INTEGER NOT NULL,
    rest_secs  INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS tasks (
    id         INTEGER PRIMARY KEY,
    session_id INTEGER NOT NULL REFERENCES sessions(id),
    name       TEXT NOT NULL,
    category   TEXT NOT NULL DEFAULT '',
    started_at DATETIME NOT NULL,
    ended_at   DATETIME NOT NULL
);`

// dataPath resolves $XDG_DATA_HOME/pomo/pomo.db, falling back to ~/.local/share/pomo/pomo.db.
func dataPath() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "pomo", "pomo.db"), nil
}

func openDBAt(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("could not create data directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}
	if _, err := db.Exec(dbSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not create schema: %w", err)
	}
	return db, nil
}

func openDB() (*sql.DB, error) {
	path, err := dataPath()
	if err != nil {
		return nil, err
	}
	return openDBAt(path)
}

// insertSession writes the session and all completed tasks in a single transaction.
// Tasks with a zero EndedAt (the active task at quit) are skipped.
func insertSession(db *sql.DB, m Model) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO sessions (started_at, ended_at, work_secs, rest_secs) VALUES (?, ?, ?, ?)`,
		m.startedAt.UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
		int64(m.totalWorked.Seconds()),
		int64(m.totalRested.Seconds()),
	)
	if err != nil {
		return 0, err
	}
	sessionID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, t := range m.tasks {
		if t.EndedAt.IsZero() {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO tasks (session_id, name, category, started_at, ended_at) VALUES (?, ?, ?, ?, ?)`,
			sessionID,
			t.Name,
			t.Category,
			t.StartedAt.UTC().Format(time.RFC3339),
			t.EndedAt.UTC().Format(time.RFC3339),
		); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return sessionID, nil
}
