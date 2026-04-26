package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func printSummary(m Model) {
	fmt.Println("\n📊 Session Summary")
	fmt.Printf("  Intervals completed: %d\n", m.intervalsCompleted)
	fmt.Printf("  Total worked: %s\n", formatDurationHuman(m.totalWorked))
	fmt.Printf("  Total rested: %s\n", formatDurationHuman(m.totalRested))
	fmt.Printf("  Interval: %.0fm work | %.0fm rest\n", m.workDuration.Minutes(), m.restDuration.Minutes())

	// Compute per-task totals from the tasks slice
	taskTotals := make(map[string]time.Duration)
	for _, t := range m.tasks {
		if t.Name == "" {
			continue
		}
		end := t.EndedAt
		if end.IsZero() {
			end = time.Now()
		}
		taskTotals[t.Name] += end.Sub(t.StartedAt)
	}

	if len(taskTotals) > 0 {
		fmt.Println("\n  Tasks:")

		names := make([]string, 0, len(taskTotals))
		for name := range taskTotals {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			fmt.Printf("    - %s: %s\n", name, formatDurationHuman(taskTotals[name]))
		}
	}
}

func writeSummaryFile(m Model, cfg Config) error {
	if !cfg.CreateSessionSummary {
		return nil
	}

	pomosDir := cfg.SummaryFolder
	// Expand leading ~ to the user's home directory
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

	filename := m.startedAt.Format("2006-01-02_15-04-05") + ".md"
	filePath := filepath.Join(pomosDir, filename)

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Pomo Session — %s\n\n", m.startedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("- **Intervals completed:** %d\n", m.intervalsCompleted))
	sb.WriteString(fmt.Sprintf("- **Total worked:** %s\n", formatDurationHuman(m.totalWorked)))
	sb.WriteString(fmt.Sprintf("- **Total rested:** %s\n", formatDurationHuman(m.totalRested)))
	sb.WriteString(fmt.Sprintf("- **Interval:** %.0fm work | %.0fm rest\n", m.workDuration.Minutes(), m.restDuration.Minutes()))

	// Compute per-task totals
	taskTotals := make(map[string]time.Duration)
	for _, t := range m.tasks {
		if t.Name == "" {
			continue
		}
		end := t.EndedAt
		if end.IsZero() {
			end = time.Now()
		}
		taskTotals[t.Name] += end.Sub(t.StartedAt)
	}

	if len(taskTotals) > 0 {
		sb.WriteString("\n## Tasks\n\n")

		names := make([]string, 0, len(taskTotals))
		for name := range taskTotals {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			sb.WriteString(fmt.Sprintf("- **%s:** %s\n", name, formatDurationHuman(taskTotals[name])))
		}
	}

	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("could not write summary file: %w", err)
	}

	fmt.Printf("\n  Summary saved to %s/%s\n", cfg.SummaryFolder, filename)
	return nil
}

func formatDurationHuman(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
