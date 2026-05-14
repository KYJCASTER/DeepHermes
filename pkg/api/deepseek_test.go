package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestChatContextReturnsReadableAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key","type":"authentication_error","code":"invalid_api_key"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "deepseek-v4-flash", "bad", 0)
	_, err := client.ChatContext(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, 1024, 0.7)
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized || apiErr.Code != "invalid_api_key" {
		t.Fatalf("unexpected API error: %#v", apiErr)
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Fatalf("expected friendly API key message, got %q", err.Error())
	}
}

func TestChatContextRetriesTransientEOF(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Fatal(err)
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "deepseek-v4-flash", "key", 1)
	resp, err := client.ChatContext(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, 1024, 0.7)
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if got := resp.Choices[0].Message.Content; got != "ok" {
		t.Fatalf("unexpected response %q", got)
	}
}

func TestRetryErrorMatchesMaxRetriesExceeded(t *testing.T) {
	err := &RetryError{Err: io.EOF}
	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Fatalf("expected RetryError to match ErrMaxRetriesExceeded")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected RetryError to unwrap EOF")
	}
}

func TestChatStreamFallsBackWhenStreamDisconnectsBeforeData(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var request ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Stream {
			if requests == 1 {
				hijacker, ok := w.(http.Hijacker)
				if !ok {
					t.Fatal("server does not support hijacking")
				}
				conn, _, err := hijacker.Hijack()
				if err != nil {
					t.Fatal(err)
				}
				_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\n\r\n"))
				if tcp, ok := conn.(*net.TCPConn); ok {
					_ = tcp.CloseWrite()
				}
				_ = conn.Close()
				return
			}
			t.Fatalf("unexpected second stream request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"fallback"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "deepseek-v4-flash", "key", 0)
	var content string
	var usage *Usage
	resp, err := client.ChatStreamContext(context.Background(), []Message{{Role: "user", Content: "hi"}}, nil, 1024, 0.7, func(update StreamUpdate) error {
		content += update.Content
		if update.Usage != nil {
			usage = update.Usage
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if content != "fallback" {
		t.Fatalf("expected fallback content, got %q", content)
	}
	if usage == nil || usage.TotalTokens != 2 {
		t.Fatalf("expected fallback usage, got %#v", usage)
	}
	if resp == nil || len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "fallback" {
		t.Fatalf("unexpected fallback response %#v", resp)
	}
}
