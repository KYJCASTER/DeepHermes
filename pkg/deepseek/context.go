package deepseek

import (
	"strings"

	"github.com/ad201/deephermes/pkg/api"
)

// ContextManager handles long-context window strategies for DeepSeek models.
type ContextManager struct {
	model        string
	maxContext   int // max context tokens for this model
	compactAt    int // compact when usage exceeds this (80% default)
	summaryCache string
}

func NewContextManager(model string) *ContextManager {
	return &ContextManager{
		model:      model,
		maxContext: modelContextLimit(model),
		compactAt:  80,
	}
}

func modelContextLimit(model string) int {
	switch model {
	case "deepseek-chat", "deepseek-reasoner", "deepseek-v4-flash", "deepseek-v4-pro":
		return 1000000
	default:
		return 1000000
	}
}

// ApproxTokens estimates token count from string length.
// Rough heuristic: ~4 chars per token for English text, ~2 for code.
func ApproxTokens(s string) int {
	// Simple character-based estimation
	return len(s) / 3
}

// ShouldCompact returns true if the message list should be compacted.
func (cm *ContextManager) ShouldCompact(messages []api.Message) bool {
	total := 0
	for _, m := range messages {
		total += ApproxTokens(m.Content)
	}
	threshold := cm.maxContext * cm.compactAt / 100
	return total > threshold
}

// Compact produces a compacted message list by summarizing old messages.
func (cm *ContextManager) Compact(messages []api.Message, summaryFunc func([]api.Message) (string, error)) ([]api.Message, error) {
	if len(messages) < 6 {
		return messages, nil
	}

	// Keep system message + last N messages
	keepRecent := 6 // keep last 3 user-assistant pairs
	oldMessages := messages[1 : len(messages)-keepRecent]

	if cm.summaryCache != "" {
		// Use cached summary
		return append(
			[]api.Message{{Role: "system", Content: "Previous conversation summary: " + cm.summaryCache}},
			messages[len(messages)-keepRecent:]...,
		), nil
	}

	summary, err := summaryFunc(oldMessages)
	if err != nil {
		return messages, err
	}

	cm.summaryCache = summary
	compacted := append(
		[]api.Message{
			messages[0], // system prompt
			{Role: "system", Content: "Previous conversation summary: " + summary},
		},
		messages[len(messages)-keepRecent:]...,
	)
	return compacted, nil
}

// TruncateOldest removes oldest messages until under token budget.
func (cm *ContextManager) TruncateOldest(messages []api.Message, maxTokens int) []api.Message {
	total := 0
	// Count from end to keep most recent
	cutIdx := 0
	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := ApproxTokens(messages[i].Content) + ApproxTokens(messages[i].Role)
		if total+msgTokens > maxTokens {
			cutIdx = i + 1
			break
		}
		total += msgTokens
	}
	if cutIdx > 0 && cutIdx < len(messages) {
		return messages[cutIdx:]
	}
	return messages
}

// TokenEstimate returns estimated token usage for a message list.
type TokenEstimate struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
	UsagePercent     int `json:"usagePercent"`
}

func (cm *ContextManager) Estimate(messages []api.Message, completionTokens int) TokenEstimate {
	prompt := 0
	for _, m := range messages {
		prompt += ApproxTokens(m.Content) + ApproxTokens(m.Role)
	}
	total := prompt + completionTokens
	return TokenEstimate{
		PromptTokens:     prompt,
		CompletionTokens: completionTokens,
		TotalTokens:      total,
		UsagePercent:     total * 100 / cm.maxContext,
	}
}

// unused import suppression
var _ = strings.TrimSpace
