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
	case "init":
		path, err := configPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: could not resolve config path:", err)
			os.Exit(1)
		}
		if _, err := os.Stat(path); err == nil {
			fmt.Println("Config file already exists:", path)
			return
		}
		initCfg := defaultConfig()
		if err := writeDefaultConfig(path, initCfg); err != nil {
			fmt.Fprintln(os.Stderr, "error: could not write config:", err)
			os.Exit(1)
		}
		fmt.Println("Config written to", path)
		return
	}

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
	}
}
