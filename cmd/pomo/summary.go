package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type taskSummary struct {
	name  string
	total time.Duration
}

type categoryGroup struct {
	category string
	total    time.Duration
	tasks    []taskSummary
}

// groupTasksByCategory groups tasks by category, sorted by total duration descending.
// Uncategorised tasks (empty category) appear last.
func groupTasksByCategory(tasks []Task) []categoryGroup {
	type groupState struct {
		tasks   []taskSummary
		taskIdx map[string]int
		total   time.Duration
	}

	stateMap := make(map[string]*groupState)
	var order []string

	for _, t := range tasks {
		if t.Name == "" {
			continue
		}
		end := t.EndedAt
		if end.IsZero() {
			end = time.Now()
		}
		dur := end.Sub(t.StartedAt)
		cat := t.Category

		if _, exists := stateMap[cat]; !exists {
			stateMap[cat] = &groupState{taskIdx: make(map[string]int)}
			order = append(order, cat)
		}
		gs := stateMap[cat]
		gs.total += dur
		if idx, exists := gs.taskIdx[t.Name]; exists {
			gs.tasks[idx].total += dur
		} else {
			gs.taskIdx[t.Name] = len(gs.tasks)
			gs.tasks = append(gs.tasks, taskSummary{name: t.Name, total: dur})
		}
	}

	var groups []categoryGroup
	for _, cat := range order {
		gs := stateMap[cat]
		groups = append(groups, categoryGroup{
			category: cat,
			total:    gs.total,
			tasks:    gs.tasks,
		})
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].category == "" {
			return false
		}
		if groups[j].category == "" {
			return true
		}
		return groups[i].total > groups[j].total
	})

	return groups
}

// formatGroupHeader renders "category  ─────  duration" for terminal output.
func formatGroupHeader(category string, total time.Duration) string {
	name := category
	if name == "" {
		name = "uncategorised"
	}
	durStr := formatDurationHuman(total)
	const width = 48
	dashCount := width - len(name) - 4 - len(durStr)
	if dashCount < 2 {
		dashCount = 2
	}
	return fmt.Sprintf("%s  %s  %s", name, strings.Repeat("─", dashCount), durStr)
}

// computeTaskTotals aggregates task durations by name, sorted alphabetically.
func computeTaskTotals(tasks []Task) (map[string]time.Duration, []string) {
	totals := make(map[string]time.Duration)
	for _, t := range tasks {
		if t.Name == "" {
			continue
		}
		end := t.EndedAt
		if end.IsZero() {
			end = time.Now()
		}
		totals[t.Name] += end.Sub(t.StartedAt)
	}
	names := make([]string, 0, len(totals))
	for name := range totals {
		names = append(names, name)
	}
	sort.Strings(names)
	return totals, names
}

func printSummary(m Model) {
	fmt.Println("\n📊 Session Summary")
	fmt.Printf("  Intervals completed: %d\n", m.intervalsCompleted)
	fmt.Printf("  Total worked: %s\n", formatDurationHuman(m.totalWorked))
	fmt.Printf("  Total rested: %s\n", formatDurationHuman(m.totalRested))
	fmt.Printf("  Interval: %.0fm work | %.0fm rest\n", m.workDuration.Minutes(), m.restDuration.Minutes())

	if len(m.categories) > 0 {
		groups := groupTasksByCategory(m.tasks)
		if len(groups) > 0 {
			fmt.Println()
			for _, g := range groups {
				fmt.Printf("  %s\n", formatGroupHeader(g.category, g.total))
				for _, ts := range g.tasks {
					fmt.Printf("    %-40s  %s\n", ts.name, formatDurationHuman(ts.total))
				}
				fmt.Println()
			}
		}
	} else {
		totals, names := computeTaskTotals(m.tasks)
		if len(totals) > 0 {
			fmt.Println("\n  Tasks:")
			for _, name := range names {
				fmt.Printf("    - %s: %s\n", name, formatDurationHuman(totals[name]))
			}
		}
	}
}

func writeSummaryFile(m Model, cfg Config) error {
	if !cfg.SessionSummary.Create {
		return nil
	}

	pomosDir := cfg.SessionSummary.Folder
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

	tags := cfg.SessionSummary.Tags
	if tags == nil {
		tags = []string{"daily", "pomo summary"}
	}
	frontmatter := buildFrontmatter(tags, m.startedAt)
	sb.WriteString(frontmatter)

	sb.WriteString(fmt.Sprintf("# Pomo Session — %s\n\n", m.startedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("- **Intervals completed:** %d\n", m.intervalsCompleted))
	sb.WriteString(fmt.Sprintf("- **Total worked:** %s\n", formatDurationHuman(m.totalWorked)))
	sb.WriteString(fmt.Sprintf("- **Total rested:** %s\n", formatDurationHuman(m.totalRested)))
	sb.WriteString(fmt.Sprintf("- **Interval:** %.0fm work | %.0fm rest\n", m.workDuration.Minutes(), m.restDuration.Minutes()))

	if len(m.categories) > 0 {
		groups := groupTasksByCategory(m.tasks)
		if len(groups) > 0 {
			sb.WriteString("\n## Tasks\n")
			for _, g := range groups {
				catName := g.category
				if catName == "" {
					catName = "uncategorised"
				}
				sb.WriteString(fmt.Sprintf("\n**%s** (%s)\n", catName, formatDurationHuman(g.total)))
				for _, ts := range g.tasks {
					sb.WriteString(fmt.Sprintf("- %s: %s\n", ts.name, formatDurationHuman(ts.total)))
				}
			}
		}
	} else {
		totals, names := computeTaskTotals(m.tasks)
		if len(totals) > 0 {
			sb.WriteString("\n## Tasks\n\n")
			for _, name := range names {
				sb.WriteString(fmt.Sprintf("- **%s:** %s\n", name, formatDurationHuman(totals[name])))
			}
		}
	}

	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("could not write summary file: %w", err)
	}

	fmt.Printf("\n  Summary saved to %s/%s\n", cfg.SessionSummary.Folder, filename)
	return nil
}

func buildFrontmatter(tags []string, created time.Time) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", created.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("title: %s\n", created.Format("2006-01-02")))
	if len(tags) > 0 {
		sb.WriteString("tags:\n")
		for _, tag := range tags {
			sb.WriteString(fmt.Sprintf("  - %s\n", tag))
		}
	}
	sb.WriteString("---\n\n")
	return sb.String()
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
