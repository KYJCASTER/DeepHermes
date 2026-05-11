package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/config"
)

func TestSessionPersistenceRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cfg := config.Default()
	first := NewApp(cfg)
	now := time.Now().UTC()
	first.sessions["session-test"] = &Session{
		ID:        "session-test",
		Name:      "Saved Chat",
		Agent:     agent.New(first.client, first.registry, first.agentConfig()),
		Model:     cfg.Model,
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []api.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi", ReasoningContent: "thinking"},
		},
		AgentMessages: []api.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", ReasoningContent: "need tool", ToolCalls: []api.ToolCall{{ID: "call_1", Type: "function"}}},
			{Role: "tool", ToolCallID: "call_1", Content: "tool-result"},
			{Role: "assistant", Content: "hi"},
		},
		ContextSummary: "Earlier plot and project state.",
		Usage:          TokenUsage{TotalTokens: 42, PromptCacheHitTokens: 10},
		LastRun: &RunMetrics{
			Usage:        TokenUsage{TotalTokens: 42},
			StartedAt:    now.Format(time.RFC3339),
			FinishedAt:   now.Add(time.Second).Format(time.RFC3339),
			DurationMs:   1000,
			TokensPerSec: 12.5,
		},
	}
	if err := first.saveSessionByID("session-test"); err != nil {
		t.Fatal(err)
	}

	second := NewApp(cfg)
	if err := second.loadPersistedSessions(); err != nil {
		t.Fatal(err)
	}
	loaded := second.sessions["session-test"]
	if loaded == nil {
		t.Fatal("expected session to be loaded")
	}
	if loaded.Name != "Saved Chat" {
		t.Fatalf("expected name Saved Chat, got %q", loaded.Name)
	}
	if got := loaded.Messages[1].ReasoningContent; got != "thinking" {
		t.Fatalf("expected reasoning to persist, got %q", got)
	}
	if got := len(loaded.AgentMessages); got != 4 {
		t.Fatalf("expected API history to persist, got %d messages", got)
	}
	if got := loaded.AgentMessages[1].ReasoningContent; got != "need tool" {
		t.Fatalf("expected tool-call reasoning to persist in API history, got %q", got)
	}
	if got := loaded.ContextSummary; got != "Earlier plot and project state." {
		t.Fatalf("expected context summary to persist, got %q", got)
	}
	if loaded.Usage.TotalTokens != 42 {
		t.Fatalf("expected total tokens 42, got %d", loaded.Usage.TotalTokens)
	}
	if loaded.LastRun == nil || loaded.LastRun.TokensPerSec != 12.5 {
		t.Fatalf("expected last run metrics to persist, got %#v", loaded.LastRun)
	}
}

func TestLoadPersistedSessionsQuarantinesCorruptFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cfg := config.Default()
	app := NewApp(cfg)
	dir, err := app.sessionsDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	badPath := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(badPath, []byte("{not-json"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := app.loadPersistedSessions(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(badPath); !os.IsNotExist(err) {
		t.Fatalf("expected corrupt session file to be moved, stat err=%v", err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "corrupt", "broken.json.*.corrupt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one quarantined corrupt file, got %v", matches)
	}
}

func TestSessionBackupRestoreAndExport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cfg := config.Default()
	app := NewApp(cfg)
	now := time.Now().UTC()
	app.sessions["session-export"] = &Session{
		ID:        "session-export",
		Name:      "Export Chat",
		Agent:     agent.New(app.client, app.registry, app.agentConfig()),
		Model:     cfg.Model,
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []api.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi", ReasoningContent: "thinking"},
		},
		ContextSummary: "Project context.",
		Usage:          TokenUsage{TotalTokens: 12},
	}

	backupPath := filepath.Join(t.TempDir(), "sessions.json")
	count, err := app.backupSessionsToPath(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one backed-up session, got %d", count)
	}

	restored := NewApp(cfg)
	count, err = restored.restoreSessionsFromPath(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one restored session, got %d", count)
	}
	if restored.sessions["session-export"] == nil {
		t.Fatal("expected restored session")
	}

	mdPath := filepath.Join(t.TempDir(), "session.md")
	if err := restored.exportSessionToPath("session-export", "markdown", mdPath); err != nil {
		t.Fatal(err)
	}
	md, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "# Export Chat") || !strings.Contains(string(md), "Project context.") {
		t.Fatalf("unexpected markdown export:\n%s", string(md))
	}

	jsonPath := filepath.Join(t.TempDir(), "session.json")
	if err := restored.exportSessionToPath("session-export", "json", jsonPath); err != nil {
		t.Fatal(err)
	}
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonData), `"version": 1`) || !strings.Contains(string(jsonData), `"id": "session-export"`) {
		t.Fatalf("unexpected json export:\n%s", string(jsonData))
	}

	restoredSingle := NewApp(cfg)
	count, err = restoredSingle.restoreSessionsFromPath(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || restoredSingle.sessions["session-export"] == nil {
		t.Fatalf("expected single exported session to restore, count=%d", count)
	}
}
