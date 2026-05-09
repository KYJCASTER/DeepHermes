package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/tools"
)

type Agent struct {
	client   *api.Client
	registry *tools.Registry
	cfg      Config
	messages []api.Message
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
	return a.messages
}

func (a *Agent) SetMessages(msgs []api.Message) {
	a.messages = msgs
}

func (a *Agent) UpdateConfig(cfg Config) {
	a.cfg = cfg
}

// Run sends a user message and returns the assistant's text response.
// It handles the tool-calling loop internally.
func (a *Agent) Run(ctx context.Context, userMessage string) (string, error) {
	// Build system prompt
	sysPrompt := BuildSystemPrompt(&a.cfg, a.registry.Names())

	// Build message list for this turn
	msgs := []api.Message{
		{Role: "system", Content: sysPrompt},
	}
	msgs = append(msgs, a.messages...)
	msgs = append(msgs, api.Message{Role: "user", Content: userMessage})

	apiTools := a.registry.ToAPITools()

	// Tool calling loop (max 10 rounds to prevent infinite loops)
	for round := 0; round < 10; round++ {
		resp, err := a.client.ChatContext(ctx, msgs, apiTools, a.cfg.MaxTokens, a.cfg.Temperature)
		if err != nil {
			return "", fmt.Errorf("API error: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]
		msg := choice.Message

		// Append assistant message to our message list
		msgs = append(msgs, msg)

		// If there are tool calls, execute them
		if len(msg.ToolCalls) > 0 {
			results := a.registry.ExecuteAll(ctx, msg.ToolCalls)
			for _, r := range results {
				msgs = append(msgs, api.Message{
					Role:       "tool",
					ToolCallID: r.ToolCallID,
					Content:    r.Content,
				})
			}
			// Continue loop to let LLM process tool results
			continue
		}

		// No tool calls — this is the final response
		// Update conversation history
		a.messages = append(a.messages,
			api.Message{Role: "user", Content: userMessage},
			api.Message{Role: "assistant", Content: msg.Content},
		)
		a.trimHistory()

		return msg.Content, nil
	}

	return "", fmt.Errorf("too many tool-calling rounds")
}

// RunStream is like Run but streams content deltas to the callback.
func (a *Agent) RunStream(ctx context.Context, userMessage string, cb func(delta string) error) (string, error) {
	sysPrompt := BuildSystemPrompt(&a.cfg, a.registry.Names())

	msgs := []api.Message{
		{Role: "system", Content: sysPrompt},
	}
	msgs = append(msgs, a.messages...)
	msgs = append(msgs, api.Message{Role: "user", Content: userMessage})

	apiTools := a.registry.ToAPITools()

	var fullContent strings.Builder

	for round := 0; round < 10; round++ {
		// Use non-streaming for rounds with tool calls; streaming for final
		resp, err := a.client.ChatContext(ctx, msgs, apiTools, a.cfg.MaxTokens, a.cfg.Temperature)
		if err != nil {
			return "", fmt.Errorf("API error: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]
		msg := choice.Message

		msgs = append(msgs, msg)

		if len(msg.ToolCalls) > 0 {
			results := a.registry.ExecuteAll(ctx, msg.ToolCalls)
			for _, r := range results {
				if err := cb(fmt.Sprintf("\n[Tool: %s]\n%s\n", r.Name, r.Content)); err != nil {
					return "", err
				}
				msgs = append(msgs, api.Message{
					Role:       "tool",
					ToolCallID: r.ToolCallID,
					Content:    r.Content,
				})
			}
			continue
		}

		// Stream the final content
		if err := cb(msg.Content); err != nil {
			return "", err
		}
		fullContent.WriteString(msg.Content)

		a.messages = append(a.messages,
			api.Message{Role: "user", Content: userMessage},
			api.Message{Role: "assistant", Content: msg.Content},
		)
		a.trimHistory()
		return fullContent.String(), nil
	}

	return "", fmt.Errorf("too many tool-calling rounds")
}

// trimHistory removes oldest messages to keep context manageable.
// Simple strategy: keep last N messages (pairs of user+assistant).
func (a *Agent) trimHistory() {
	// Keep last 20 messages (10 conversation turns)
	maxMessages := 40
	if len(a.messages) > maxMessages {
		a.messages = a.messages[len(a.messages)-maxMessages:]
	}
}
