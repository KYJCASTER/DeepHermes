package app

import (
	"fmt"
	"strings"

	"github.com/ad201/deephermes/app/events"
	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/deepseek"
)

const (
	contextSummaryTriggerMessages = 28
	contextSummaryRecentMessages  = 14
	contextSummaryMaxChars        = 9000
	contextSummaryLineChars       = 420
)

func (a *App) prepareSessionContextLocked(sess *Session) []api.Message {
	history := sessionAPIHistory(sess)
	compacted, summary := compactSessionHistory(history, sess.ContextSummary)
	if len(compacted) < len(history) && a.ctx != nil {
		emit(a.ctx, sess.ID, events.EventContextCompacted, events.ContextCompactedPayload{
			MessagesBefore: len(history),
			MessagesAfter:  len(compacted),
			SummaryTokens:  approxContextTokens(summary),
		})
	}
	sess.AgentMessages = append([]api.Message(nil), compacted...)
	sess.ContextSummary = summary
	return compacted
}

func compactSessionHistory(history []api.Message, existingSummary string) ([]api.Message, string) {
	if len(history) <= contextSummaryTriggerMessages {
		return append([]api.Message(nil), history...), strings.TrimSpace(existingSummary)
	}

	cut := len(history) - contextSummaryRecentMessages
	for cut > 0 && history[cut].Role != "user" {
		cut--
	}
	if cut < 4 {
		return append([]api.Message(nil), history...), strings.TrimSpace(existingSummary)
	}

	oldMessages := history[:cut]
	recent := append([]api.Message(nil), history[cut:]...)
	nextSummary := mergeContextSummary(existingSummary, summarizeMessagesForContext(oldMessages))
	return recent, nextSummary
}

func summarizeMessagesForContext(messages []api.Message) string {
	var sb strings.Builder
	sb.WriteString("Earlier conversation summary for long-context continuity:\n")
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			writeSummaryLine(&sb, "User", msg.Content)
		case "assistant":
			if strings.TrimSpace(msg.Content) != "" {
				writeSummaryLine(&sb, "Assistant", msg.Content)
			} else if len(msg.ToolCalls) > 0 {
				var names []string
				for _, call := range msg.ToolCalls {
					if call.Function.Name != "" {
						names = append(names, call.Function.Name)
					}
				}
				writeSummaryLine(&sb, "Assistant tools", strings.Join(names, ", "))
			}
		case "tool":
			writeSummaryLine(&sb, "Tool result", msg.Content)
		}
		if sb.Len() >= contextSummaryMaxChars {
			break
		}
	}
	return trimContextSummary(sb.String(), contextSummaryMaxChars)
}

func writeSummaryLine(sb *strings.Builder, role, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	content = strings.Join(strings.Fields(content), " ")
	if len([]rune(content)) > contextSummaryLineChars {
		content = string([]rune(content)[:contextSummaryLineChars]) + "..."
	}
	sb.WriteString(fmt.Sprintf("- %s: %s\n", role, content))
}

func mergeContextSummary(existing, next string) string {
	existing = strings.TrimSpace(existing)
	next = strings.TrimSpace(next)
	switch {
	case existing == "":
		return trimContextSummary(next, contextSummaryMaxChars)
	case next == "":
		return trimContextSummary(existing, contextSummaryMaxChars)
	default:
		return trimContextSummary(existing+"\n"+next, contextSummaryMaxChars)
	}
}

func trimContextSummary(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return strings.TrimSpace(string(runes[len(runes)-limit:]))
}

func approxContextTokens(value string) int {
	return deepseek.ApproxTokens(value)
}
