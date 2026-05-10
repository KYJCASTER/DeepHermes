package app

import (
	"strings"
	"testing"

	"github.com/ad201/deephermes/pkg/api"
)

func TestCompactSessionHistorySummarizesOldMessagesAndKeepsRecent(t *testing.T) {
	var history []api.Message
	for i := 0; i < 18; i++ {
		history = append(history,
			api.Message{Role: "user", Content: "user message about character motivation and project detail"},
			api.Message{Role: "assistant", Content: "assistant answer preserving setting and implementation notes"},
		)
	}

	recent, summary := compactSessionHistory(history, "")
	if len(recent) >= len(history) {
		t.Fatalf("expected old messages to be compacted")
	}
	if summary == "" || !strings.Contains(summary, "Earlier conversation summary") {
		t.Fatalf("expected summary, got %q", summary)
	}
	if recent[0].Role != "user" {
		t.Fatalf("expected recent history to start at a user boundary, got %s", recent[0].Role)
	}
}

func TestCompactSessionHistoryLeavesShortHistoryUnchanged(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	recent, summary := compactSessionHistory(history, "kept")
	if len(recent) != len(history) || summary != "kept" {
		t.Fatalf("unexpected compaction recent=%d summary=%q", len(recent), summary)
	}
}
