package app

import (
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
