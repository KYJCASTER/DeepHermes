package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ad201/deephermes/pkg/config"
	"github.com/ad201/deephermes/pkg/tools"
)

func TestReadFileSnippetTruncatesText(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	path := filepath.Join(dir, "notes.txt")
	content := strings.Repeat("abc", 100)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	app := NewApp(config.Default())
	snippet, err := app.ReadFileSnippet(path, 12)
	if err != nil {
		t.Fatal(err)
	}
	if snippet.Binary {
		t.Fatalf("expected text file, got binary")
	}
	if !snippet.Truncated {
		t.Fatalf("expected truncated snippet")
	}
	if snippet.Content != content[:12] {
		t.Fatalf("unexpected content %q", snippet.Content)
	}
}

func TestReadFileSnippetHidesBinaryContent(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	path := filepath.Join(dir, "image.bin")
	if err := os.WriteFile(path, []byte{0, 1, 2, 3, 4}, 0600); err != nil {
		t.Fatal(err)
	}

	app := NewApp(config.Default())
	snippet, err := app.ReadFileSnippet(path, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if !snippet.Binary {
		t.Fatalf("expected binary file")
	}
	if snippet.Content != "" {
		t.Fatalf("binary content should be hidden")
	}
}

func TestFileBrowserMethodsRejectOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	app := NewApp(config.Default())

	if _, err := app.ListDirectory(outside); err == nil {
		t.Fatal("expected ListDirectory to reject outside workspace")
	}
	if _, err := app.ReadFileContent(outsideFile); err == nil {
		t.Fatal("expected ReadFileContent to reject outside workspace")
	}
	if _, err := app.ReadFileSnippet(outsideFile, 1024); err == nil {
		t.Fatal("expected ReadFileSnippet to reject outside workspace")
	}
}

func TestSearchWorkspaceFiles(t *testing.T) {
	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "alpha_notes.go"), []byte("package src"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node_modules", "alpha_hidden.go"), []byte("package hidden"), 0600); err != nil {
		t.Fatal(err)
	}

	app := NewApp(config.Default())
	results, err := app.SearchWorkspaceFiles("alpha", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one visible result, got %d: %#v", len(results), results)
	}
	if results[0].RelativePath != "src/alpha_notes.go" {
		t.Fatalf("unexpected result: %#v", results[0])
	}
}

func TestRollbackToolChangeRestoresExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "edit.txt")
	if err := os.WriteFile(path, []byte("before"), 0600); err != nil {
		t.Fatal(err)
	}
	app := NewApp(config.Default())
	event := tools.ExecutionEvent{
		ID:        "call-restore",
		ToolName:  "write_file",
		Arguments: `{"file_path":"` + filepath.ToSlash(path) + `","content":"after"}`,
	}
	app.captureToolRollback(event)
	if err := os.WriteFile(path, []byte("after"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := app.RollbackToolChange("call-restore")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Restored {
		t.Fatalf("expected restored result: %#v", result)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "before" {
		t.Fatalf("expected original content, got %q", string(data))
	}
}

func TestRollbackToolChangeDeletesNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")
	app := NewApp(config.Default())
	event := tools.ExecutionEvent{
		ID:        "call-delete",
		ToolName:  "write_file",
		Arguments: `{"file_path":"` + filepath.ToSlash(path) + `","content":"created"}`,
	}
	app.captureToolRollback(event)
	if err := os.WriteFile(path, []byte("created"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := app.RollbackToolChange("call-delete")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Deleted {
		t.Fatalf("expected deleted result: %#v", result)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be deleted, stat err=%v", err)
	}
}
