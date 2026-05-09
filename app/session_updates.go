package app

import (
	"time"

	"github.com/ad201/deephermes/pkg/api"
)

func (a *App) appendAssistantDelta(sessionID, content, reasoning string) {
	a.sessionsMu.Lock()
	defer a.sessionsMu.Unlock()

	sess, ok := a.sessions[sessionID]
	if !ok {
		return
	}

	msgs := sess.Messages
	if len(msgs) == 0 || msgs[len(msgs)-1].Role != "assistant" {
		msgs = append(msgs, api.Message{Role: "assistant"})
	}
	last := msgs[len(msgs)-1]
	last.Content += content
	last.ReasoningContent += reasoning
	msgs[len(msgs)-1] = last
	sess.Messages = msgs
	sess.UpdatedAt = time.Now()
}

func metricsUsage(metrics *RunMetrics) TokenUsage {
	if metrics == nil {
		return TokenUsage{}
	}
	return metrics.Usage
}
