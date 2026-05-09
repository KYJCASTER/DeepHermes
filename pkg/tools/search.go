package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Glob struct{}

func (t *Glob) Name() string        { return "glob" }
func (t *Glob) Description() string { return "Find files matching a glob pattern. Returns sorted file paths." }
func (t *Glob) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The glob pattern to match (e.g. \"**/*.go\", \"src/**/*.ts\")",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "The directory to search in (defaults to working directory)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *Glob) Execute(ctx context.Context, args map[string]any) (string, error) {
	pattern, _ := args["pattern"].(string)
	searchPath, _ := args["path"].(string)
	if searchPath == "" {
		searchPath = "."
	}

	matches, err := filepath.Glob(filepath.Join(searchPath, pattern))
	if err != nil {
		return "", fmt.Errorf("glob error: %w", err)
	}

	// Also support ** with Walk
	if strings.Contains(pattern, "**") {
		matches = nil
		prefix := strings.Split(pattern, "**")[0]
		prefix = strings.TrimSuffix(prefix, "/")
		suffix := strings.Split(pattern, "**")[1]
		suffix = strings.TrimPrefix(suffix, "/")

		filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(searchPath, path)
			if matched, _ := filepath.Match(prefix+"*"+suffix, rel); matched || strings.HasSuffix(rel, suffix) {
				if !info.IsDir() || suffix == "" {
					matches = append(matches, path)
				}
			}
			return nil
		})
	}

	if len(matches) == 0 {
		return "No files matched.", nil
	}

	var out strings.Builder
	for _, m := range matches {
		fmt.Fprintln(&out, m)
	}
	return out.String(), nil
}

// --- Grep ---

type Grep struct{}

func (t *Grep) Name() string        { return "grep" }
func (t *Grep) Description() string { return "Search for a regex pattern in files. Returns matching file paths or matching lines with context." }
func (t *Grep) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "The regex pattern to search for",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File or directory to search in (defaults to working directory)",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "Glob pattern to filter files (e.g. \"*.go\")",
			},
			"output_mode": map[string]any{
				"type":        "string",
				"enum":        []string{"files_with_matches", "content", "count"},
				"description": "Output mode: files_with_matches (default), content (shows matching lines), count",
			},
			"context": map[string]any{
				"type":        "integer",
				"description": "Number of context lines to show",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *Grep) Execute(ctx context.Context, args map[string]any) (string, error) {
	pattern, _ := args["pattern"].(string)
	searchPath, _ := args["path"].(string)
	globFilter, _ := args["glob"].(string)
	outputMode, _ := args["output_mode"].(string)
	contextLines, _ := args["context"].(float64)

	if searchPath == "" {
		searchPath = "."
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex: %w", err)
	}

	var results []grepMatch
	filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if globFilter != "" {
			if matched, _ := filepath.Match(globFilter, filepath.Base(path)); !matched {
				return nil
			}
		}
		// Skip binary/common dirs
		if shouldSkip(path) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				results = append(results, grepMatch{
					File:    path,
					LineNum: i + 1,
					Content: line,
					Before:  getContext(lines, i, int(contextLines), true),
					After:   getContext(lines, i, int(contextLines), false),
				})
			}
		}
		return nil
	})

	switch outputMode {
	case "count":
		if len(results) == 0 {
			return "0 matches", nil
		}
		counts := make(map[string]int)
		for _, m := range results {
			counts[m.File]++
		}
		var out strings.Builder
		for f, c := range counts {
			fmt.Fprintf(&out, "%s: %d\n", f, c)
		}
		return out.String(), nil
	case "content":
		if len(results) == 0 {
			return "No matches found.", nil
		}
		var out strings.Builder
		for _, m := range results {
			if contextLines > 0 {
				for _, b := range m.Before {
					fmt.Fprintf(&out, "%s-%d\t%s\n", m.File, b.Num, b.Text)
				}
			}
			fmt.Fprintf(&out, "%s:%d\t%s\n", m.File, m.LineNum, m.Content)
			if contextLines > 0 {
				for _, a := range m.After {
					fmt.Fprintf(&out, "%s-%d\t%s\n", m.File, a.Num, a.Text)
				}
			}
		}
		return out.String(), nil
	default:
		// files_with_matches
		seen := make(map[string]bool)
		var out strings.Builder
		for _, m := range results {
			if !seen[m.File] {
				fmt.Fprintln(&out, m.File)
				seen[m.File] = true
			}
		}
		if out.Len() == 0 {
			return "No files matched.", nil
		}
		return out.String(), nil
	}
}

type grepMatch struct {
	File    string
	LineNum int
	Content string
	Before  []contextLine
	After   []contextLine
}

type contextLine struct {
	Num  int
	Text string
}

func getContext(lines []string, idx, n int, before bool) []contextLine {
	var result []contextLine
	start, end := idx-n, idx
	if before {
		start, end = idx-n, idx
	} else {
		start, end = idx+1, idx+1+n
	}
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	for i := start; i < end; i++ {
		if i != idx {
			result = append(result, contextLine{Num: i + 1, Text: lines[i]})
		}
	}
	return result
}

func shouldSkip(path string) bool {
	base := filepath.Base(path)
	skipDirs := []string{".git", "node_modules", ".venv", "vendor", "__pycache__", ".claude"}
	for _, d := range skipDirs {
		if strings.Contains(path, string(filepath.Separator)+d+string(filepath.Separator)) {
			return true
		}
	}
	// Skip binary files by extension
	ext := strings.ToLower(filepath.Ext(path))
	skipExts := map[string]bool{".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".zip": true, ".tar": true, ".gz": true, ".png": true, ".jpg": true,
		".jpeg": true, ".gif": true, ".ico": true, ".pdf": true}
	if skipExts[ext] {
		return true
	}
	_ = base // used for future filtering
	return false
}
