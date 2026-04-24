package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.SummaryFolder != "~/pomos" {
		t.Errorf("Expected SummaryFolder '~/pomos', got %q", cfg.SummaryFolder)
	}
	if cfg.WorkTime != 50 {
		t.Errorf("Expected WorkTime 50, got %d", cfg.WorkTime)
	}
	if cfg.IntervalTime != 60 {
		t.Errorf("Expected IntervalTime 60, got %d", cfg.IntervalTime)
	}
	if !cfg.ProduceSummary {
		t.Error("Expected ProduceSummary to be true by default")
	}
}

func TestLoadConfigCreatesDefaultOnFirstRun(t *testing.T) {
	// Point XDG_CONFIG_HOME at a temp dir so we don't touch the real config
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return defaults
	def := defaultConfig()
	if cfg.WorkTime != def.WorkTime {
		t.Errorf("Expected WorkTime %d, got %d", def.WorkTime, cfg.WorkTime)
	}
	if cfg.IntervalTime != def.IntervalTime {
		t.Errorf("Expected IntervalTime %d, got %d", def.IntervalTime, cfg.IntervalTime)
	}
	if cfg.SummaryFolder != def.SummaryFolder {
		t.Errorf("Expected SummaryFolder %q, got %q", def.SummaryFolder, cfg.SummaryFolder)
	}
	if cfg.ProduceSummary != def.ProduceSummary {
		t.Errorf("Expected ProduceSummary %v, got %v", def.ProduceSummary, cfg.ProduceSummary)
	}

	// Config file should have been created on disk
	expectedPath := filepath.Join(dir, "pomo", "config.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Expected config file to be created on first run, but it doesn't exist")
	}
}

func TestLoadConfigParsesValues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	configDir := filepath.Join(dir, "pomo")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Could not create config dir: %v", err)
	}

	content := "summary_folder: /custom/path\nwork_time: 25\ninterval_time: 45\nproduce_summary: false\n"
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Could not write config file: %v", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.SummaryFolder != "/custom/path" {
		t.Errorf("Expected SummaryFolder '/custom/path', got %q", cfg.SummaryFolder)
	}
	if cfg.WorkTime != 25 {
		t.Errorf("Expected WorkTime 25, got %d", cfg.WorkTime)
	}
	if cfg.IntervalTime != 45 {
		t.Errorf("Expected IntervalTime 45, got %d", cfg.IntervalTime)
	}
	if cfg.ProduceSummary {
		t.Error("Expected ProduceSummary to be false")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	configDir := filepath.Join(dir, "pomo")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Could not create config dir: %v", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(":::invalid yaml:::\n\t\tbad:"), 0644); err != nil {
		t.Fatalf("Could not write config file: %v", err)
	}

	_, err := loadConfig()
	if err == nil {
		t.Error("Expected error for invalid YAML, got none")
	}
}

func TestConfigPath(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
		path, err := configPath()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expected := "/custom/xdg/pomo/config.yaml"
		if path != expected {
			t.Errorf("Expected path %q, got %q", expected, path)
		}
	})

	t.Run("falls back to ~/.config when XDG_CONFIG_HOME unset", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		path, err := configPath()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", "pomo", "config.yaml")
		if path != expected {
			t.Errorf("Expected path %q, got %q", expected, path)
		}
	})
}
