package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"encoding/json"
)

func readFileJSONSchema() map[string]any {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to read",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "Line number to start reading from (0-indexed)",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of lines to read",
			},
		},
		"required": []string{"file_path"},
	}
	var result map[string]any
	json.Unmarshal(mustJSON(schema), &result)
	return result
}

type ReadFile struct{}

func (t *ReadFile) Name() string { return "read_file" }
func (t *ReadFile) Description() string {
	return "Read a file from the filesystem. Returns the file contents with line numbers."
}
func (t *ReadFile) Parameters() map[string]any { return readFileJSONSchema() }

func (t *ReadFile) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["file_path"].(string)
	if path == "" {
		return "", fmt.Errorf("file_path is required")
	}
	if err := ValidatePath(AllowedDirFromContext(ctx), path); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading %s: %w", path, err)
	}
	lines := strings.Split(string(data), "\n")
	offset := 0
	if v, ok := args["offset"].(float64); ok {
		offset = int(v)
	}
	limit := len(lines)
	if v, ok := args["limit"].(float64); ok && int(v) > 0 {
		limit = offset + int(v)
		if limit > len(lines) {
			limit = len(lines)
		}
	}

	var out strings.Builder
	for i := offset; i < limit; i++ {
		fmt.Fprintf(&out, "%d\t%s\n", i+1, lines[i])
	}
	return out.String(), nil
}

// --- Write File ---

type WriteFile struct{}

func (t *WriteFile) Name() string { return "write_file" }
func (t *WriteFile) Description() string {
	return "Write a file to the filesystem. Creates parent directories if needed. Overwrites existing files."
}
func (t *WriteFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"file_path", "content"},
	}
}

func (t *WriteFile) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["file_path"].(string)
	content, _ := args["content"].(string)
	if path == "" {
		return "", fmt.Errorf("file_path is required")
	}
	if err := ValidatePath(AllowedDirFromContext(ctx), path); err != nil {
		return "", err
	}
	// Ensure parent directory exists
	if dir := dirOf(path); dir != "" {
		os.MkdirAll(dir, 0755)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("error writing %s: %w", path, err)
	}
	return fmt.Sprintf("File written: %s (%d bytes)", path, len(content)), nil
}

// --- Edit File ---

type EditFile struct{}

func (t *EditFile) Name() string { return "edit_file" }
func (t *EditFile) Description() string {
	return "Perform exact string replacements in a file. When editing text, ensure you preserve exact indentation (tabs/spaces)."
}
func (t *EditFile) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "The text to replace",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "The replacement text",
			},
			"replace_all": map[string]any{
				"type":        "boolean",
				"description": "Replace all occurrences (default false)",
			},
		},
		"required": []string{"file_path", "old_string", "new_string"},
	}
}

func (t *EditFile) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, _ := args["file_path"].(string)
	oldStr, _ := args["old_string"].(string)
	newStr, _ := args["new_string"].(string)
	replaceAll, _ := args["replace_all"].(bool)

	if path == "" {
		return "", fmt.Errorf("file_path is required")
	}
	if err := ValidatePath(AllowedDirFromContext(ctx), path); err != nil {
		return "", err
	}
	if oldStr == newStr {
		return "", fmt.Errorf("old_string and new_string must be different")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading %s: %w", path, err)
	}
	content := string(data)

	count := strings.Count(content, oldStr)
	if count == 0 {
		return "", fmt.Errorf("old_string not found in file")
	}
	if count > 1 && !replaceAll {
		return "", fmt.Errorf("old_string found %d times; use replace_all=true or provide more context", count)
	}

	result := strings.ReplaceAll(content, oldStr, newStr)
	if err := os.WriteFile(path, []byte(result), 0644); err != nil {
		return "", fmt.Errorf("error writing %s: %w", path, err)
	}
	return fmt.Sprintf("File edited: %s (%d replacements)", path, count), nil
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return ""
}

func mustJSON(v map[string]any) []byte {
	b, _ := json.Marshal(v)
	return b
}
