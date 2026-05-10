package app

import (
	"testing"
	"time"

	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/config"
)

func TestUpdateDeleteAndBranchSessionMessages(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cfg := config.Default()
	app := NewApp(cfg)
	now := time.Now()
	app.sessions["session-actions"] = &Session{
		ID:        "session-actions",
		Name:      "Action Chat",
		Agent:     agent.New(app.client, app.registry, app.agentConfig()),
		Model:     cfg.Model,
		CreatedAt: now,
		UpdatedAt: now,
		Messages: []api.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "next"},
		},
	}

	if err := app.UpdateMessage(UpdateMessageRequest{SessionID: "session-actions", Index: 0, Content: "hello edited"}); err != nil {
		t.Fatal(err)
	}
	if got := app.sessions["session-actions"].Messages[0].Content; got != "hello edited" {
		t.Fatalf("expected edited content, got %q", got)
	}

	if err := app.DeleteMessage(MessageIndexRequest{SessionID: "session-actions", Index: 1}); err != nil {
		t.Fatal(err)
	}
	if got := len(app.sessions["session-actions"].Messages); got != 2 {
		t.Fatalf("expected 2 messages after delete, got %d", got)
	}

	branch, err := app.BranchSession(BranchSessionRequest{SessionID: "session-actions", UpToIndex: 0, NameSuffix: "Fork"})
	if err != nil {
		t.Fatal(err)
	}
	if branch == nil || branch.ID == "" {
		t.Fatalf("expected branch result, got %#v", branch)
	}
	if got := len(app.sessions[branch.ID].Messages); got != 1 {
		t.Fatalf("expected branched history length 1, got %d", got)
	}
	if got := app.sessions[branch.ID].Messages[0].Content; got != "hello edited" {
		t.Fatalf("expected branched edited content, got %q", got)
	}
}
