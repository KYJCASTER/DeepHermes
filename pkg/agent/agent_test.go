package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/tools"
)

type fakeLookupTool struct{}

func (fakeLookupTool) Name() string        { return "lookup" }
func (fakeLookupTool) Description() string { return "Lookup test data." }
func (fakeLookupTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
func (fakeLookupTool) Execute(context.Context, map[string]any) (string, error) {
	return "tool-result", nil
}

func TestRunStreamDetailedPassesReasoningBackWithToolCalls(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var request api.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if requestCount == 1 {
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"need tool\"}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"lookup\",\"arguments\":\"{}\"}}]}}]}\n\n"))
			_, _ = w.Write([]byte("data: {\"choices\":[{\"finish_reason\":\"tool_calls\"}]}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}

		if len(request.Messages) < 4 {
			t.Fatalf("expected tool-call transcript in second request, got %#v", request.Messages)
		}
		assistant := request.Messages[2]
		if assistant.Role != "assistant" || assistant.ReasoningContent != "need tool" || len(assistant.ToolCalls) != 1 {
			t.Fatalf("assistant tool-call message lost reasoning: %#v", assistant)
		}
		toolMsg := request.Messages[3]
		if toolMsg.Role != "tool" || toolMsg.ToolCallID != "call_1" || toolMsg.Content != "tool-result" {
			t.Fatalf("unexpected tool result message: %#v", toolMsg)
		}
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"done\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	reg := tools.NewRegistry()
	reg.Register(fakeLookupTool{})
	client := api.NewClient(server.URL, "deepseek-v4-pro", "key", 0)
	client.SetThinking(true)
	ag := New(client, reg, Config{WorkDir: ".", MaxTokens: 1024, Temperature: 0.7})

	result, err := ag.RunStreamDetailed(context.Background(), "use a tool", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "done" {
		t.Fatalf("unexpected result content %q", result.Content)
	}
	if requestCount != 2 {
		t.Fatalf("expected two API requests, got %d", requestCount)
	}
}

func TestSanitizeHistoryKeepsReasoningOnlyForToolCalls(t *testing.T) {
	history := sanitizeHistory([]api.Message{
		{Role: "assistant", Content: "final", ReasoningContent: "drop"},
		{Role: "assistant", Content: "", ReasoningContent: "keep", ToolCalls: []api.ToolCall{{ID: "call_1"}}},
	})

	if history[0].ReasoningContent != "" {
		t.Fatalf("expected final assistant reasoning to be stripped, got %q", history[0].ReasoningContent)
	}
	if history[1].ReasoningContent != "keep" {
		t.Fatalf("expected tool-call reasoning to be kept, got %q", history[1].ReasoningContent)
	}
}
