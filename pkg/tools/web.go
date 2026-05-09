package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type WebFetch struct{}

func (t *WebFetch) Name() string        { return "web_fetch" }
func (t *WebFetch) Description() string { return "Fetch content from a URL and convert to text. Use for reading web pages, documentation, etc." }
func (t *WebFetch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch content from",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "Optional prompt describing what information to extract from the page",
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetch) Execute(ctx context.Context, args map[string]any) (string, error) {
	url, _ := args["url"].(string)
	if url == "" {
		return "", fmt.Errorf("url is required")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("request error: %w", err)
	}
	req.Header.Set("User-Agent", "DeepHermes/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
	if err != nil {
		return "", fmt.Errorf("read error: %w", err)
	}

	text := htmlToText(string(body))
	if len(text) > 50000 {
		text = text[:50000] + "\n... (truncated)"
	}
	return text, nil
}

type WebSearch struct{}

func (t *WebSearch) Name() string        { return "web_search" }
func (t *WebSearch) Description() string { return "Search the web for information. Returns search results with titles and URLs." }
func (t *WebSearch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query",
			},
		},
		"required": []string{"query"},
	}
}

func (t *WebSearch) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	// Use DuckDuckGo HTML search (no API key needed)
	url := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", strings.ReplaceAll(query, " ", "+"))
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "DeepHermes/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("search error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}
	return extractSearchResults(string(body)), nil
}

func htmlToText(html string) string {
	// Simple HTML to text conversion
	text := html
	// Remove scripts and styles
	for {
		start := strings.Index(strings.ToLower(text), "<script")
		if start == -1 {
			break
		}
		end := strings.Index(strings.ToLower(text), "</script>")
		if end == -1 {
			break
		}
		text = text[:start] + text[end+9:]
	}
	for {
		start := strings.Index(strings.ToLower(text), "<style")
		if start == -1 {
			break
		}
		end := strings.Index(strings.ToLower(text), "</style>")
		if end == -1 {
			break
		}
		text = text[:start] + text[end+8:]
	}
	// Remove remaining HTML tags
	var result strings.Builder
	inTag := false
	for _, c := range text {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(c)
		}
	}
	// Collapse whitespace
	lines := strings.Split(result.String(), "\n")
	var clean []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	return strings.Join(clean, "\n")
}

func extractSearchResults(html string) string {
	text := htmlToText(html)
	if len(text) > 10000 {
		text = text[:10000]
	}
	return "Search results:\n\n" + text
}
