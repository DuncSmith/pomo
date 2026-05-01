package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfg = defaultConfig()
	}

	result, err := parseArgs(os.Args[1:], cfg)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	switch result.action {
	case "help":
		showHelp()
		os.Exit(0)
	case "version":
		fmt.Println(result.versionInfo)
		os.Exit(0)
	case "report":
		runReport(cfg, *result.reportArgs)
		return
	}

	// CLI flag overrides config file value
	if result.createSessionSummary != nil {
		cfg.SessionSummary.Create = *result.createSessionSummary
	}

	result.model.categories = cfg.Categories

	p := tea.NewProgram(*result.model)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(Model); ok && m.quitting {
		printSummary(m)
		if err := writeSummaryFile(m, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save summary file: %v\n", err)
		}
		if db, err := openDB(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not open database: %v\n", err)
		} else {
			defer db.Close()
			if _, err := insertSession(db, m); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save session to database: %v\n", err)
			}
		}
	}
}
