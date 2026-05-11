package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type OCRImageRequest struct {
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	DataBase64 string `json:"dataBase64"`
}

type OCRImageResult struct {
	Text     string `json:"text"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Error    string `json:"error,omitempty"`
}

type OCRProviderPreset struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Model   string `json:"model"`
}

func (a *App) ListOCRPresets() []OCRProviderPreset {
	return []OCRProviderPreset{
		{ID: "openai", Name: "OpenAI (GPT-4o)", BaseURL: "https://api.openai.com/v1", Model: "gpt-4o"},
		{ID: "openai_mini", Name: "OpenAI (GPT-4o-mini)", BaseURL: "https://api.openai.com/v1", Model: "gpt-4o-mini"},
		{ID: "deepseek", Name: "DeepSeek (VL)", BaseURL: "https://api.deepseek.com", Model: "deepseek-vl"},
		{ID: "siliconflow", Name: "SiliconFlow", BaseURL: "https://api.siliconflow.cn/v1", Model: "Qwen/Qwen2.5-VL-72B-Instruct"},
		{ID: "together", Name: "Together AI", BaseURL: "https://api.together.xyz/v1", Model: "Qwen/Qwen2.5-VL-72B-Instruct"},
	}
}

type ocrChatRequest struct {
	Model       string           `json:"model"`
	Messages    []ocrChatMessage `json:"messages"`
	Temperature float64          `json:"temperature"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
}

type ocrChatMessage struct {
	Role    string           `json:"role"`
	Content []ocrContentPart `json:"content"`
}

type ocrContentPart struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *ocrImageURL `json:"image_url,omitempty"`
}

type ocrImageURL struct {
	URL string `json:"url"`
}

type ocrChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

func (a *App) OCRImage(req OCRImageRequest) (*OCRImageResult, error) {
	req.FileName = strings.TrimSpace(req.FileName)
	req.MimeType = strings.TrimSpace(req.MimeType)
	req.DataBase64 = strings.TrimSpace(req.DataBase64)
	if req.FileName == "" {
		req.FileName = "clipboard-image"
	}
	if req.MimeType == "" {
		req.MimeType = "image/png"
	}
	if req.DataBase64 == "" {
		return nil, fmt.Errorf("image data is required")
	}
	return a.runOCRProvider(req.FileName, req.MimeType, req.DataBase64)
}

func (a *App) OCRImageFile(path string) (*OCRImageResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("image path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	mimeType := imageMimeType(path, data)
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, fmt.Errorf("file is not an image: %s", path)
	}
	return a.runOCRProvider(filepath.Base(path), mimeType, base64.StdEncoding.EncodeToString(data))
}

func (a *App) runOCRProvider(fileName, mimeType, dataBase64 string) (*OCRImageResult, error) {
	cfg := a.cfg.OCR
	result := &OCRImageResult{
		Provider: normalizeOCRProvider(cfg.Provider),
		Model:    cfg.Model,
	}
	if !cfg.Enabled {
		result.Error = "OCR provider is disabled — enable it in Settings → OCR"
		return result, fmt.Errorf("%s", result.Error)
	}
	if normalizeOCRProvider(cfg.Provider) != "openai_compatible" {
		result.Error = fmt.Sprintf("unsupported OCR provider: %s", cfg.Provider)
		return result, fmt.Errorf("%s", result.Error)
	}
	apiKey := a.cfg.GetOCRAPIKey()
	if strings.TrimSpace(apiKey) == "" {
		result.Error = "OCR API key is required — set it in Settings → OCR"
		return result, fmt.Errorf("%s", result.Error)
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		result.Error = "OCR base URL is required — configure it in Settings → OCR"
		return result, fmt.Errorf("%s", result.Error)
	}
	if strings.TrimSpace(cfg.Model) == "" {
		result.Error = "OCR model is required — configure it in Settings → OCR"
		return result, fmt.Errorf("%s", result.Error)
	}
	imageBytes, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid image data: %w", err)
	}
	maxImageBytes := cfg.MaxImageBytes
	if maxImageBytes <= 0 {
		maxImageBytes = 8 * 1024 * 1024
	}
	if len(imageBytes) > maxImageBytes {
		return nil, fmt.Errorf("image is too large for OCR (%d bytes > %d bytes)", len(imageBytes), maxImageBytes)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, fmt.Errorf("unsupported image type: %s", mimeType)
	}

	prompt := strings.TrimSpace(cfg.Prompt)
	if prompt == "" {
		prompt = "Extract all readable text from this image."
	}
	body, err := json.Marshal(ocrChatRequest{
		Model:       strings.TrimSpace(cfg.Model),
		Temperature: 0,
		MaxTokens:   2000,
		Messages: []ocrChatMessage{
			{
				Role: "user",
				Content: []ocrContentPart{
					{Type: "text", Text: prompt},
					{Type: "image_url", ImageURL: &ocrImageURL{URL: "data:" + mimeType + ";base64," + dataBase64}},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, ocrChatCompletionsURL(cfg.BaseURL), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := (&http.Client{Timeout: time.Duration(timeout) * time.Second}).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var parsed ocrChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("OCR response parse failed: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return nil, fmt.Errorf("OCR API %d: %s", response.StatusCode, parsed.Error.Message)
		}
		return nil, fmt.Errorf("OCR API %d: %s", response.StatusCode, strings.TrimSpace(string(raw)))
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("OCR API returned no choices")
	}
	text := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if text == "" {
		text = "(No readable text detected.)"
	}
	a.recordLog("info", "OCR completed for "+fileName)
	return &OCRImageResult{
		Text:     text,
		Provider: normalizeOCRProvider(cfg.Provider),
		Model:    cfg.Model,
	}, nil
}

func normalizeOCRProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "openai", "openai-compatible", "openai_compatible":
		return "openai_compatible"
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func ocrChatCompletionsURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}

func imageMimeType(path string, data []byte) string {
	if ext := strings.ToLower(filepath.Ext(path)); ext != "" {
		if mimeType := mime.TypeByExtension(ext); strings.HasPrefix(mimeType, "image/") {
			return strings.Split(mimeType, ";")[0]
		}
	}
	return http.DetectContentType(data)
}
