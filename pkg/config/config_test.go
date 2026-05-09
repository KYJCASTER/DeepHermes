package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Model != "deepseek-v4-pro" {
		t.Errorf("expected deepseek-v4-pro, got %s", cfg.Model)
	}
	if cfg.MaxTokens != 32768 {
		t.Errorf("expected 32768, got %d", cfg.MaxTokens)
	}
	if cfg.API.BaseURL != "https://api.deepseek.com" {
		t.Errorf("expected base URL, got %s", cfg.API.BaseURL)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path := filepath.Join(dir, "config.yaml")
	content := `model: deepseek-reasoner
max_tokens: 4096
temperature: 0.5
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "deepseek-reasoner" {
		t.Errorf("expected deepseek-reasoner, got %s", cfg.Model)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("expected 4096, got %d", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.5 {
		t.Errorf("expected 0.5, got %f", cfg.Temperature)
	}
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("DEEPSEEK_MODEL", "deepseek-chat-test")
	defer os.Unsetenv("DEEPSEEK_MODEL")

	cfg := Default()
	cfg.applyEnvOverrides()
	if cfg.Model != "deepseek-chat-test" {
		t.Errorf("expected env override, got %s", cfg.Model)
	}
}

func TestExpandPaths(t *testing.T) {
	cfg := Default()
	cfg.expandPaths()
	home, _ := os.UserHomeDir()
	expected := filepath.Clean(filepath.Join(home, ".deephermes", "memory"))
	if cfg.Memory.Dir != expected {
		t.Errorf("expected %q, got %q", expected, cfg.Memory.Dir)
	}
}
