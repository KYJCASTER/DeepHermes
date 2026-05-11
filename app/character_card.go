package app

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type CharacterCardImportResult struct {
	Name      string `json:"name"`
	RoleCard  string `json:"roleCard"`
	WorldBook string `json:"worldBook"`
	Source    string `json:"source"`
}

func (a *App) ImportCharacterCard() (*CharacterCardImportResult, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Import SillyTavern Character Card",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "SillyTavern cards (*.json;*.png)", Pattern: "*.json;*.png"},
			{DisplayName: "All files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result, err := parseCharacterCard(data, filepath.Ext(path))
	if err != nil {
		return nil, err
	}
	result.Source = path
	a.recordLog("info", "character card imported from "+path)
	return result, nil
}

func parseCharacterCard(data []byte, ext string) (*CharacterCardImportResult, error) {
	if strings.EqualFold(ext, ".png") || bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		card, err := extractSillyTavernPNGCard(data)
		if err != nil {
			return nil, err
		}
		data = card
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse character card JSON: %w", err)
	}
	cardData := raw
	if nested, ok := raw["data"].(map[string]any); ok {
		cardData = nested
	}
	return buildCharacterCardResult(cardData), nil
}

func buildCharacterCardResult(data map[string]any) *CharacterCardImportResult {
	name := stringField(data, "name")
	sections := []struct {
		Title string
		Value string
	}{
		{"Name", name},
		{"Description", stringField(data, "description")},
		{"Personality", stringField(data, "personality")},
		{"Scenario", stringField(data, "scenario")},
		{"First Message", stringField(data, "first_mes")},
		{"Example Dialogue", stringField(data, "mes_example")},
		{"System Prompt", stringField(data, "system_prompt")},
		{"Post History Instructions", stringField(data, "post_history_instructions")},
		{"Creator Notes", stringField(data, "creator_notes")},
	}

	var role strings.Builder
	for _, section := range sections {
		value := strings.TrimSpace(section.Value)
		if value == "" {
			continue
		}
		fmt.Fprintf(&role, "## %s\n%s\n\n", section.Title, value)
	}

	world := characterBookToText(data["character_book"])
	if world == "" {
		world = characterBookToText(data["world_book"])
	}

	return &CharacterCardImportResult{
		Name:      name,
		RoleCard:  strings.TrimSpace(role.String()),
		WorldBook: world,
	}
}

func stringField(data map[string]any, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func characterBookToText(value any) string {
	book, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	entries, ok := book["entries"].([]any)
	if !ok {
		return ""
	}
	var out strings.Builder
	for _, item := range entries {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if enabled, ok := entry["enabled"].(bool); ok && !enabled {
			continue
		}
		content := strings.TrimSpace(stringField(entry, "content"))
		if content == "" {
			continue
		}
		title := strings.TrimSpace(stringField(entry, "comment"))
		if title == "" {
			title = "Lore Entry"
		}
		keys := flattenStringList(entry["keys"])
		fmt.Fprintf(&out, "## %s\n", title)
		if len(keys) > 0 {
			fmt.Fprintf(&out, "Keys: %s\n\n", strings.Join(keys, ", "))
		}
		fmt.Fprintf(&out, "%s\n\n", content)
	}
	return strings.TrimSpace(out.String())
}

func flattenStringList(value any) []string {
	switch v := value.(type) {
	case []any:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case []string:
		return v
	default:
		return nil
	}
}

func extractSillyTavernPNGCard(data []byte) ([]byte, error) {
	const pngHeader = "\x89PNG\r\n\x1a\n"
	if !bytes.HasPrefix(data, []byte(pngHeader)) {
		return nil, fmt.Errorf("not a PNG file")
	}
	offset := len(pngHeader)
	for offset+12 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		chunkType := string(data[offset+4 : offset+8])
		chunkStart := offset + 8
		chunkEnd := chunkStart + length
		if chunkEnd+4 > len(data) {
			break
		}
		chunk := data[chunkStart:chunkEnd]
		if text, ok := pngTextChunk(chunkType, chunk); ok {
			if decoded, ok := decodeSillyTavernPayload(text); ok {
				return decoded, nil
			}
		}
		offset = chunkEnd + 4
	}
	return nil, fmt.Errorf("no SillyTavern chara metadata found in PNG")
}

func pngTextChunk(chunkType string, chunk []byte) (string, bool) {
	switch chunkType {
	case "tEXt":
		parts := bytes.SplitN(chunk, []byte{0}, 2)
		if len(parts) != 2 || string(parts[0]) != "chara" {
			return "", false
		}
		return string(parts[1]), true
	case "zTXt":
		parts := bytes.SplitN(chunk, []byte{0}, 2)
		if len(parts) != 2 || string(parts[0]) != "chara" || len(parts[1]) < 2 || parts[1][0] != 0 {
			return "", false
		}
		reader, err := zlib.NewReader(bytes.NewReader(parts[1][1:]))
		if err != nil {
			return "", false
		}
		defer reader.Close()
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return "", false
		}
		return string(decompressed), true
	case "iTXt":
		parts := bytes.SplitN(chunk, []byte{0}, 2)
		if len(parts) != 2 || string(parts[0]) != "chara" {
			return "", false
		}
		rest := parts[1]
		if len(rest) < 2 {
			return "", false
		}
		compressed := rest[0] == 1
		rest = rest[2:]
		for i := 0; i < 2; i++ {
			idx := bytes.IndexByte(rest, 0)
			if idx < 0 {
				return "", false
			}
			rest = rest[idx+1:]
		}
		if !compressed {
			return string(rest), true
		}
		reader, err := zlib.NewReader(bytes.NewReader(rest))
		if err != nil {
			return "", false
		}
		defer reader.Close()
		decompressed, err := io.ReadAll(reader)
		if err != nil {
			return "", false
		}
		return string(decompressed), true
	default:
		return "", false
	}
}

func decodeSillyTavernPayload(text string) ([]byte, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, false
	}
	if strings.HasPrefix(text, "{") {
		return []byte(text), true
	}
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err == nil {
		return decoded, true
	}
	decoded, err = base64.RawStdEncoding.DecodeString(text)
	if err == nil {
		return decoded, true
	}
	return nil, false
}
