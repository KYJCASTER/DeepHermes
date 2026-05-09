package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatStreamContextParsesReasoningAndUsage(t *testing.T) {
	var request ChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"think\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"answer\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15,\"prompt_cache_hit_tokens\":7,\"prompt_cache_miss_tokens\":3,\"completion_tokens_details\":{\"reasoning_tokens\":2}}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "deepseek-v4-pro", "key", 0)
	client.SetThinking(true)

	var reasoning string
	var content string
	var usage *Usage
	resp, err := client.ChatStreamContext(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, 1024, 0.7, func(update StreamUpdate) error {
		reasoning += update.ReasoningContent
		content += update.Content
		if update.Usage != nil {
			usage = update.Usage
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !request.Stream || request.StreamOptions == nil || !request.StreamOptions.IncludeUsage {
		t.Fatalf("expected stream with include_usage, got %#v", request.StreamOptions)
	}
	if request.Thinking == nil || request.Thinking.Type != "enabled" {
		t.Fatalf("expected thinking enabled, got %#v", request.Thinking)
	}
	if reasoning != "think" || content != "answer" {
		t.Fatalf("unexpected stream content reasoning=%q content=%q", reasoning, content)
	}
	if usage == nil || usage.PromptCacheHitTokens != 7 || usage.CompletionTokensDetails.ReasoningTokens != 2 {
		t.Fatalf("unexpected usage %#v", usage)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 15 {
		t.Fatalf("expected final response usage, got %#v", resp.Usage)
	}
}
