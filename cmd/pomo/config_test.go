package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.SessionSummary.Folder != "~/pomos" {
		t.Errorf("Expected SessionSummary.Folder '~/pomos', got %q", cfg.SessionSummary.Folder)
	}
	if cfg.WorkTime != 50 {
		t.Errorf("Expected WorkTime 50, got %d", cfg.WorkTime)
	}
	if cfg.IntervalTime != 60 {
		t.Errorf("Expected IntervalTime 60, got %d", cfg.IntervalTime)
	}
	if cfg.LunchTime != 60 {
		t.Errorf("Expected LunchTime 60, got %d", cfg.LunchTime)
	}
	if cfg.AutoStartWork {
		t.Error("Expected AutoStartWork to be false by default")
	}
	if !cfg.SessionSummary.Create {
		t.Error("Expected SessionSummary.Create to be true by default")
	}
	expectedTags := []string{"daily", "pomo-summary"}
	if len(cfg.SessionSummary.Tags) != len(expectedTags) {
		t.Fatalf("Expected %d default tags, got %d", len(expectedTags), len(cfg.SessionSummary.Tags))
	}
	for i, tag := range expectedTags {
		if cfg.SessionSummary.Tags[i] != tag {
			t.Errorf("Expected default tag %q at index %d, got %q", tag, i, cfg.SessionSummary.Tags[i])
		}
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
	if cfg.SessionSummary.Folder != def.SessionSummary.Folder {
		t.Errorf("Expected SessionSummary.Folder %q, got %q", def.SessionSummary.Folder, cfg.SessionSummary.Folder)
	}
	if cfg.SessionSummary.Create != def.SessionSummary.Create {
		t.Errorf("Expected SessionSummary.Create %v, got %v", def.SessionSummary.Create, cfg.SessionSummary.Create)
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

	content := "work_time: 25\ninterval_time: 45\nauto-start-work-interval: true\nsession_summary:\n  create_session_summary: false\n  summary_folder: /custom/path\n"
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Could not write config file: %v", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.SessionSummary.Folder != "/custom/path" {
		t.Errorf("Expected SessionSummary.Folder '/custom/path', got %q", cfg.SessionSummary.Folder)
	}
	if cfg.WorkTime != 25 {
		t.Errorf("Expected WorkTime 25, got %d", cfg.WorkTime)
	}
	if cfg.IntervalTime != 45 {
		t.Errorf("Expected IntervalTime 45, got %d", cfg.IntervalTime)
	}
	if !cfg.AutoStartWork {
		t.Error("Expected AutoStartWork to be true")
	}
	if cfg.SessionSummary.Create {
		t.Error("Expected SessionSummary.Create to be false")
	}
	// When summary_tags is omitted in config file, defaults are retained
	expectedDefaultTags := []string{"daily", "pomo-summary"}
	if len(cfg.SessionSummary.Tags) != len(expectedDefaultTags) {
		t.Fatalf("Expected %d default tags when omitted, got %d", len(expectedDefaultTags), len(cfg.SessionSummary.Tags))
	}
	for i, tag := range expectedDefaultTags {
		if cfg.SessionSummary.Tags[i] != tag {
			t.Errorf("Expected default tag %q at index %d, got %q", tag, i, cfg.SessionSummary.Tags[i])
		}
	}
}

func TestLoadConfigParsesTags(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	configDir := filepath.Join(dir, "pomo")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Could not create config dir: %v", err)
	}

	content := "work_time: 25\ninterval_time: 45\nsession_summary:\n  create_session_summary: true\n  summary_folder: /custom/path\n  summary_tags:\n    - tag\n    - another\n"
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Could not write config file: %v", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedTags := []string{"tag", "another"}
	if len(cfg.SessionSummary.Tags) != len(expectedTags) {
		t.Fatalf("Expected %d tags, got %d", len(expectedTags), len(cfg.SessionSummary.Tags))
	}
	for i, tag := range expectedTags {
		if cfg.SessionSummary.Tags[i] != tag {
			t.Errorf("Expected tag %q at index %d, got %q", tag, i, cfg.SessionSummary.Tags[i])
		}
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
