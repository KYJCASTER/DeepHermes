package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/tools"
)

type Agent struct {
	client   *api.Client
	registry *tools.Registry
	cfg      Config
	messages []api.Message
}

type RunResult struct {
	Content          string
	ReasoningContent string
	Usage            *api.Usage
	StartedAt        time.Time
	FirstTokenAt     time.Time
	FinishedAt       time.Time
}

func New(client *api.Client, registry *tools.Registry, cfg Config) *Agent {
	return &Agent{
		client:   client,
		registry: registry,
		cfg:      cfg,
	}
}

func (a *Agent) Registry() *tools.Registry { return a.registry }

func (a *Agent) Reset() {
	a.messages = nil
}

func (a *Agent) Messages() []api.Message {
	return append([]api.Message(nil), a.messages...)
}

func (a *Agent) SetMessages(msgs []api.Message) {
	a.messages = sanitizeHistory(msgs)
}

func (a *Agent) UpdateConfig(cfg Config) {
	a.cfg = cfg
}

// Run sends a user message and returns the assistant's text response.
func (a *Agent) Run(ctx context.Context, userMessage string) (string, error) {
	sysPrompt := BuildSystemPrompt(&a.cfg, a.registry.Names())

	msgs := []api.Message{{Role: "system", Content: sysPrompt}}
	msgs = append(msgs, sanitizeHistory(a.messages)...)
	msgs = append(msgs, api.Message{Role: "user", Content: userMessage})

	apiTools := a.registry.ToAPITools()

	for round := 0; round < 10; round++ {
		resp, err := a.client.ChatContext(ctx, msgs, apiTools, a.cfg.MaxTokens, a.cfg.Temperature)
		if err != nil {
			return "", fmt.Errorf("API error: %w", err)
		}
		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}

		msg := resp.Choices[0].Message
		msgs = append(msgs, msg)

		if len(msg.ToolCalls) > 0 {
			results := a.registry.ExecuteAll(ctx, msg.ToolCalls)
			for _, r := range results {
				msgs = append(msgs, api.Message{
					Role:       "tool",
					ToolCallID: r.ToolCallID,
					Content:    r.Content,
				})
			}
			continue
		}

		a.messages = append(a.messages,
			api.Message{Role: "user", Content: userMessage},
			api.Message{
				Role:             "assistant",
				Content:          msg.Content,
				ReasoningContent: msg.ReasoningContent,
			},
		)
		a.trimHistory()
		return msg.Content, nil
	}

	return "", fmt.Errorf("too many tool-calling rounds")
}

// RunStream keeps the legacy streaming API and forwards final-answer deltas.
func (a *Agent) RunStream(ctx context.Context, userMessage string, cb func(delta string) error) (string, error) {
	result, err := a.RunStreamDetailed(ctx, userMessage, func(update api.StreamUpdate) error {
		if update.Content == "" {
			return nil
		}
		return cb(update.Content)
	})
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// RunStreamDetailed streams DeepSeek content, reasoning, tool calls, and usage.
func (a *Agent) RunStreamDetailed(ctx context.Context, userMessage string, cb func(api.StreamUpdate) error) (*RunResult, error) {
	sysPrompt := BuildSystemPrompt(&a.cfg, a.registry.Names())

	msgs := []api.Message{{Role: "system", Content: sysPrompt}}
	msgs = append(msgs, sanitizeHistory(a.messages)...)
	msgs = append(msgs, api.Message{Role: "user", Content: userMessage})

	apiTools := a.registry.ToAPITools()
	result := &RunResult{StartedAt: time.Now()}

	for round := 0; round < 10; round++ {
		var content strings.Builder
		var reasoning strings.Builder
		toolCalls := make(map[int]*api.ToolCall)
		var toolOrder []int

		resp, err := a.client.ChatStreamContext(ctx, msgs, apiTools, a.cfg.MaxTokens, a.cfg.Temperature, func(update api.StreamUpdate) error {
			if update.Usage != nil {
				result.Usage = update.Usage
				if cb != nil {
					return cb(update)
				}
				return nil
			}

			if (update.Content != "" || update.ReasoningContent != "") && result.FirstTokenAt.IsZero() {
				result.FirstTokenAt = time.Now()
			}
			if update.Content != "" {
				content.WriteString(update.Content)
			}
			if update.ReasoningContent != "" {
				reasoning.WriteString(update.ReasoningContent)
			}
			if len(update.ToolCalls) > 0 {
				mergeToolCallDeltas(toolCalls, &toolOrder, update.ToolCalls)
			}
			if cb != nil {
				return cb(update)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("API error: %w", err)
		}
		if resp != nil && resp.Usage != nil {
			result.Usage = resp.Usage
		}

		calls := orderedToolCalls(toolCalls, toolOrder)
		if len(calls) > 0 {
			msgs = append(msgs, api.Message{Role: "assistant", ToolCalls: calls})
			results := a.registry.ExecuteAll(ctx, calls)
			for _, r := range results {
				if cb != nil {
					_ = cb(api.StreamUpdate{Content: fmt.Sprintf("\n[Tool: %s]\n%s\n", r.Name, r.Content)})
				}
				msgs = append(msgs, api.Message{
					Role:       "tool",
					ToolCallID: r.ToolCallID,
					Content:    r.Content,
				})
			}
			continue
		}

		result.Content = content.String()
		result.ReasoningContent = reasoning.String()
		result.FinishedAt = time.Now()

		a.messages = append(a.messages,
			api.Message{Role: "user", Content: userMessage},
			api.Message{
				Role:             "assistant",
				Content:          result.Content,
				ReasoningContent: result.ReasoningContent,
			},
		)
		a.trimHistory()
		return result, nil
	}

	return nil, fmt.Errorf("too many tool-calling rounds")
}

func sanitizeHistory(messages []api.Message) []api.Message {
	out := make([]api.Message, 0, len(messages))
	for _, msg := range messages {
		clean := msg
		clean.ReasoningContent = ""
		out = append(out, clean)
	}
	return out
}

func mergeToolCallDeltas(calls map[int]*api.ToolCall, order *[]int, deltas []api.ToolCall) {
	for i, delta := range deltas {
		idx := delta.Index
		if idx == 0 && delta.ID == "" && len(deltas) > 1 {
			idx = i
		}
		call, ok := calls[idx]
		if !ok {
			call = &api.ToolCall{Index: idx}
			calls[idx] = call
			*order = append(*order, idx)
		}
		if delta.ID != "" {
			call.ID = delta.ID
		}
		if delta.Type != "" {
			call.Type = delta.Type
		}
		if delta.Function.Name != "" {
			call.Function.Name = delta.Function.Name
		}
		if delta.Function.Arguments != "" {
			call.Function.Arguments += delta.Function.Arguments
		}
	}
}

func orderedToolCalls(calls map[int]*api.ToolCall, order []int) []api.ToolCall {
	out := make([]api.ToolCall, 0, len(order))
	for _, idx := range order {
		call := calls[idx]
		if call == nil {
			continue
		}
		if call.Type == "" {
			call.Type = "function"
		}
		out = append(out, *call)
	}
	return out
}

// trimHistory removes oldest messages to keep context manageable.
func (a *Agent) trimHistory() {
	maxMessages := 40
	if len(a.messages) > maxMessages {
		a.messages = a.messages[len(a.messages)-maxMessages:]
	}
}
