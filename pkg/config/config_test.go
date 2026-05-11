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
	if cfg.Mode != "code" {
		t.Errorf("expected code mode, got %s", cfg.Mode)
	}
	if cfg.MaxTokens != 32768 {
		t.Errorf("expected 32768, got %d", cfg.MaxTokens)
	}
	if cfg.ReasoningDisplay != "collapse" {
		t.Errorf("expected collapse reasoning display, got %s", cfg.ReasoningDisplay)
	}
	if cfg.InitialPrompt != "" {
		t.Errorf("expected empty initial prompt, got %q", cfg.InitialPrompt)
	}
	if cfg.RoleCard != "" || cfg.WorldBook != "" {
		t.Errorf("expected empty roleplay fields, got role=%q world=%q", cfg.RoleCard, cfg.WorldBook)
	}
	if cfg.API.BaseURL != "https://api.deepseek.com" {
		t.Errorf("expected base URL, got %s", cfg.API.BaseURL)
	}
	if cfg.API.TimeoutSeconds != 120 || cfg.API.MaxRetries != 3 {
		t.Errorf("expected API timeout/retries defaults, got %d/%d", cfg.API.TimeoutSeconds, cfg.API.MaxRetries)
	}
	if cfg.Safety.ToolMode != "confirm" {
		t.Errorf("expected confirm tool mode, got %s", cfg.Safety.ToolMode)
	}
	if cfg.OCR.Enabled {
		t.Errorf("expected OCR disabled by default")
	}
	if cfg.OCR.Provider != "openai_compatible" {
		t.Errorf("expected openai_compatible OCR provider, got %s", cfg.OCR.Provider)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path := filepath.Join(dir, "config.yaml")
	content := `model: deepseek-reasoner
mode: rp
max_tokens: 4096
temperature: 0.5
initial_prompt: |
  Stay in character.
role_card: |
  Name: Mira
world_book: |
  City: Lumen
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
	if cfg.Mode != "rp" {
		t.Errorf("expected rp mode, got %s", cfg.Mode)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("expected 4096, got %d", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.5 {
		t.Errorf("expected 0.5, got %f", cfg.Temperature)
	}
	if cfg.InitialPrompt != "Stay in character.\n" {
		t.Errorf("expected initial prompt to load, got %q", cfg.InitialPrompt)
	}
	if cfg.RoleCard != "Name: Mira\n" || cfg.WorldBook != "City: Lumen\n" {
		t.Errorf("expected roleplay fields to load, got role=%q world=%q", cfg.RoleCard, cfg.WorldBook)
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
