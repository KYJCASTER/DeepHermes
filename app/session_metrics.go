package app

import (
	"strings"
	"time"

	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
)

func tokenUsageFromAPI(usage *api.Usage) TokenUsage {
	if usage == nil {
		return TokenUsage{}
	}
	out := TokenUsage{
		PromptTokens:          usage.PromptTokens,
		CompletionTokens:      usage.CompletionTokens,
		TotalTokens:           usage.TotalTokens,
		PromptCacheHitTokens:  usage.PromptCacheHitTokens,
		PromptCacheMissTokens: usage.PromptCacheMissTokens,
	}
	if usage.CompletionTokensDetails != nil {
		out.ReasoningTokens = usage.CompletionTokensDetails.ReasoningTokens
	}
	return out
}

func (u *TokenUsage) Add(delta TokenUsage) {
	u.PromptTokens += delta.PromptTokens
	u.CompletionTokens += delta.CompletionTokens
	u.TotalTokens += delta.TotalTokens
	u.PromptCacheHitTokens += delta.PromptCacheHitTokens
	u.PromptCacheMissTokens += delta.PromptCacheMissTokens
	u.ReasoningTokens += delta.ReasoningTokens
}

func runMetricsFromResult(result *agent.RunResult) *RunMetrics {
	if result == nil {
		return nil
	}
	started := result.StartedAt
	if started.IsZero() {
		started = time.Now()
	}
	finished := result.FinishedAt
	if finished.IsZero() {
		finished = time.Now()
	}

	usage := tokenUsageFromAPI(result.Usage)
	duration := finished.Sub(started)
	firstTokenMs := int64(0)
	if !result.FirstTokenAt.IsZero() {
		firstTokenMs = result.FirstTokenAt.Sub(started).Milliseconds()
	}

	var tokensPerSec float64
	speedStart := result.FirstTokenAt
	if speedStart.IsZero() {
		speedStart = started
	}
	speedDuration := finished.Sub(speedStart).Seconds()
	if speedDuration > 0 && usage.CompletionTokens > 0 {
		tokensPerSec = float64(usage.CompletionTokens) / speedDuration
	}

	metrics := &RunMetrics{
		Usage:        usage,
		StartedAt:    started.Format(time.RFC3339),
		FinishedAt:   finished.Format(time.RFC3339),
		FirstTokenMs: firstTokenMs,
		DurationMs:   duration.Milliseconds(),
		TokensPerSec: tokensPerSec,
		FinishReason: strings.TrimSpace(result.FinishReason),
		Truncated:    isTruncatedFinishReason(result.FinishReason),
	}
	if !result.FirstTokenAt.IsZero() {
		metrics.FirstTokenAt = result.FirstTokenAt.Format(time.RFC3339)
	}
	return metrics
}

func isTruncatedFinishReason(reason string) bool {
	reason = strings.ToLower(strings.TrimSpace(reason))
	return reason == "length" ||
		reason == "max_tokens" ||
		strings.Contains(reason, "max_token") ||
		strings.Contains(reason, "max_output")
}
