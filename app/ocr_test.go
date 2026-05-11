package app

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ad201/deephermes/pkg/config"
)

func TestOCRImageOpenAICompatibleProvider(t *testing.T) {
	var sawImage bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ocr-key" {
			t.Fatalf("unexpected authorization %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		rawMessages := body["messages"].([]any)
		content := rawMessages[0].(map[string]any)["content"].([]any)
		for _, part := range content {
			item := part.(map[string]any)
			if item["type"] == "image_url" {
				imageURL := item["image_url"].(map[string]any)["url"].(string)
				sawImage = strings.HasPrefix(imageURL, "data:image/png;base64,")
			}
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Invoice #42"}}]}`))
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.OCR.Enabled = true
	cfg.OCR.BaseURL = server.URL
	cfg.OCR.Model = "vision-test"
	cfg.OCR.APIKey = "ocr-key"
	app := NewApp(cfg)

	result, err := app.OCRImage(OCRImageRequest{
		FileName:   "shot.png",
		MimeType:   "image/png",
		DataBase64: base64.StdEncoding.EncodeToString([]byte("fake png")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawImage {
		t.Fatalf("expected request to include image_url")
	}
	if result.Text != "Invoice #42" {
		t.Fatalf("unexpected OCR text %q", result.Text)
	}
}

func TestOCRImageDisabled(t *testing.T) {
	app := NewApp(config.Default())
	_, err := app.OCRImage(OCRImageRequest{
		FileName:   "shot.png",
		MimeType:   "image/png",
		DataBase64: base64.StdEncoding.EncodeToString([]byte("fake png")),
	})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected disabled error, got %v", err)
	}
}
