package app

import (
	"testing"
	"time"

	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
)

func TestRunMetricsFromResult(t *testing.T) {
	start := time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
	first := start.Add(500 * time.Millisecond)
	finish := start.Add(2500 * time.Millisecond)

	metrics := runMetricsFromResult(&agent.RunResult{
		StartedAt:    start,
		FirstTokenAt: first,
		FinishedAt:   finish,
		Usage: &api.Usage{
			PromptTokens:          100,
			CompletionTokens:      20,
			TotalTokens:           120,
			PromptCacheHitTokens:  70,
			PromptCacheMissTokens: 30,
			CompletionTokensDetails: &api.CompletionTokensDetails{
				ReasoningTokens: 8,
			},
		},
	})

	if metrics.FirstTokenMs != 500 {
		t.Fatalf("expected first token latency 500ms, got %d", metrics.FirstTokenMs)
	}
	if metrics.DurationMs != 2500 {
		t.Fatalf("expected duration 2500ms, got %d", metrics.DurationMs)
	}
	if metrics.TokensPerSec != 10 {
		t.Fatalf("expected 10 tokens/sec, got %f", metrics.TokensPerSec)
	}
	if metrics.Usage.PromptCacheHitTokens != 70 || metrics.Usage.ReasoningTokens != 8 {
		t.Fatalf("unexpected usage: %#v", metrics.Usage)
	}
}
