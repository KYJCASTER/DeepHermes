package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// --- Types ---

type Message struct {
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	Name             string     `json:"name,omitempty"`
}

type ToolDef struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolCall struct {
	Index    int          `json:"index,omitempty"`
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ThinkingConfig struct {
	Type string `json:"type"` // "enabled" or "disabled"
}

type ChatRequest struct {
	Model           string          `json:"model"`
	Messages        []Message       `json:"messages"`
	Tools           []ToolDef       `json:"tools,omitempty"`
	MaxTokens       int             `json:"max_tokens"`
	Temperature     float64         `json:"temperature"`
	Stream          bool            `json:"stream"`
	StreamOptions   *StreamOptions  `json:"stream_options,omitempty"`
	Thinking        *ThinkingConfig `json:"thinking,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
	Delta        *Delta  `json:"delta,omitempty"`
}

type Delta struct {
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptCacheHitTokens    int                      `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens   int                      `json:"prompt_cache_miss_tokens"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	Type       string
}

func (e *APIError) Error() string {
	if e == nil {
		return "DeepSeek API error"
	}
	detail := e.Message
	if detail == "" {
		detail = http.StatusText(e.StatusCode)
	}
	prefix := fmt.Sprintf("DeepSeek API %d", e.StatusCode)
	if e.Code != "" {
		prefix += " " + e.Code
	}
	return prefix + ": " + detail
}

type StreamChunk struct {
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type StreamUpdate struct {
	Content          string
	ReasoningContent string
	ToolCalls        []ToolCall
	Usage            *Usage
}

// --- Client ---

type Client struct {
	mu              sync.RWMutex
	baseURL         string
	apiKey          string
	httpClient      *http.Client
	model           string
	maxRetries      int
	thinkingEnabled bool
}

func NewClient(baseURL, model, apiKey string, maxRetries int) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		model:      model,
		maxRetries: maxRetries,
	}
}

type clientSnapshot struct {
	baseURL         string
	apiKey          string
	model           string
	maxRetries      int
	thinkingEnabled bool
	httpClient      *http.Client
}

func (c *Client) snapshot() clientSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return clientSnapshot{
		baseURL:         c.baseURL,
		apiKey:          c.apiKey,
		model:           c.model,
		maxRetries:      c.maxRetries,
		thinkingEnabled: c.thinkingEnabled,
		httpClient:      c.httpClient,
	}
}

// UpdateConfig updates all runtime API settings without rebuilding the app.
func (c *Client) UpdateConfig(baseURL, model, apiKey string, maxRetries, timeoutSeconds int, thinkingEnabled bool) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 120
	}
	if maxRetries < 0 {
		maxRetries = 0
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = strings.TrimRight(baseURL, "/")
	c.model = model
	c.apiKey = apiKey
	c.maxRetries = maxRetries
	c.thinkingEnabled = thinkingEnabled
	c.httpClient = &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
}

// UpdateAPIKey updates the client's API key (for runtime changes).
func (c *Client) UpdateAPIKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.apiKey = key
}

// SetThinking enables or disables thinking mode (for deepseek-reasoner).
func (c *Client) SetThinking(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.thinkingEnabled = enabled
}

func buildThinking(model string, enabled bool) *ThinkingConfig {
	if !strings.HasPrefix(model, "deepseek-v4-") {
		return nil
	}
	if enabled {
		return &ThinkingConfig{Type: "enabled"}
	}
	return &ThinkingConfig{Type: "disabled"}
}

func buildReasoningEffort(model string, thinkingEnabled bool) string {
	if strings.HasPrefix(model, "deepseek-v4-") && thinkingEnabled {
		return "max"
	}
	return ""
}

func (c *Client) do(req *http.Request, cfg clientSnapshot) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
	req.Header.Set("Content-Type", "application/json")

	var lastErr error
	for i := 0; i <= cfg.maxRetries; i++ {
		if i > 0 {
			delay := time.Duration(1<<uint(i-1)) * time.Second
			timer := time.NewTimer(delay)
			select {
			case <-req.Context().Done():
				timer.Stop()
				return nil, req.Context().Err()
			case <-timer.C:
			}
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				req.Body = body
			}
		}
		resp, err := cfg.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = parseAPIError(resp.StatusCode, body)
			continue
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, parseAPIError(resp.StatusCode, body)
		}
		return resp, nil
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func parseAPIError(statusCode int, body []byte) error {
	var wrapped struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
		Message string `json:"message"`
		Code    string `json:"code"`
		Type    string `json:"type"`
	}
	_ = json.Unmarshal(body, &wrapped)

	message := strings.TrimSpace(wrapped.Error.Message)
	code := strings.TrimSpace(wrapped.Error.Code)
	errType := strings.TrimSpace(wrapped.Error.Type)
	if message == "" {
		message = strings.TrimSpace(wrapped.Message)
	}
	if code == "" {
		code = strings.TrimSpace(wrapped.Code)
	}
	if errType == "" {
		errType = strings.TrimSpace(wrapped.Type)
	}
	if message == "" {
		message = strings.TrimSpace(string(body))
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}

	switch statusCode {
	case http.StatusUnauthorized:
		message = "API key is invalid or missing. Check your DeepSeek API key."
	case http.StatusPaymentRequired, http.StatusForbidden:
		if message == http.StatusText(statusCode) {
			message = "Request was rejected. Check DeepSeek account balance, permissions, and model access."
		}
	case http.StatusTooManyRequests:
		if message == http.StatusText(statusCode) {
			message = "Rate limit or concurrency limit reached. Wait a moment and retry."
		}
	}

	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Type:       errType,
	}
}

func (c *Client) Chat(messages []Message, tools []ToolDef, maxTokens int, temperature float64) (*ChatResponse, error) {
	return c.ChatContext(context.Background(), messages, tools, maxTokens, temperature)
}

func (c *Client) ChatContext(ctx context.Context, messages []Message, tools []ToolDef, maxTokens int, temperature float64) (*ChatResponse, error) {
	cfg := c.snapshot()
	reqBody := ChatRequest{
		Model:           cfg.model,
		Messages:        messages,
		Tools:           tools,
		MaxTokens:       maxTokens,
		Temperature:     temperature,
		Stream:          false,
		Thinking:        buildThinking(cfg.model, cfg.thinkingEnabled),
		ReasoningEffort: buildReasoningEffort(cfg.model, cfg.thinkingEnabled),
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req, cfg)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, err
	}
	return &chatResp, nil
}

// StreamCallback receives streamed content, reasoning, tool-call, and usage updates.
type StreamCallback func(update StreamUpdate) error

func (c *Client) ChatStream(messages []Message, tools []ToolDef, maxTokens int, temperature float64, cb StreamCallback) (*ChatResponse, error) {
	return c.ChatStreamContext(context.Background(), messages, tools, maxTokens, temperature, cb)
}

func (c *Client) ChatStreamContext(ctx context.Context, messages []Message, tools []ToolDef, maxTokens int, temperature float64, cb StreamCallback) (*ChatResponse, error) {
	cfg := c.snapshot()
	reqBody := ChatRequest{
		Model:           cfg.model,
		Messages:        messages,
		Tools:           tools,
		MaxTokens:       maxTokens,
		Temperature:     temperature,
		Stream:          true,
		StreamOptions:   &StreamOptions{IncludeUsage: true},
		Thinking:        buildThinking(cfg.model, cfg.thinkingEnabled),
		ReasoningEffort: buildReasoningEffort(cfg.model, cfg.thinkingEnabled),
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.do(req, cfg)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var finalResp ChatResponse
	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer for long lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Usage != nil {
			finalResp.Usage = chunk.Usage
			if err := cb(StreamUpdate{Usage: chunk.Usage}); err != nil {
				return nil, err
			}
		}
		for _, choice := range chunk.Choices {
			if choice.Delta != nil {
				update := StreamUpdate{
					Content:          choice.Delta.Content,
					ReasoningContent: choice.Delta.ReasoningContent,
					ToolCalls:        choice.Delta.ToolCalls,
				}
				if update.Content != "" || update.ReasoningContent != "" || len(update.ToolCalls) > 0 {
					if err := cb(update); err != nil {
						return nil, err
					}
				}
			}
			if choice.FinishReason != "" {
				finalResp.Choices = append(finalResp.Choices, choice)
			}
		}
	}
	return &finalResp, scanner.Err()
}
