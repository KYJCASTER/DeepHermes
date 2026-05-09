package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ReadFile{})

	tool, ok := reg.Get("read_file")
	if !ok {
		t.Fatal("expected read_file tool to be registered")
	}
	if tool.Name() != "read_file" {
		t.Errorf("expected read_file, got %s", tool.Name())
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ReadFile{})
	reg.Register(&Glob{})

	names := reg.Names()
	if len(names) != 2 {
		t.Errorf("expected 2 tools, got %d", len(names))
	}
}

func TestRegistryToAPITools(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ReadFile{})

	defs := reg.ToAPITools()
	if len(defs) != 1 {
		t.Fatalf("expected 1 tool def, got %d", len(defs))
	}
	if defs[0].Type != "function" {
		t.Errorf("expected function type, got %s", defs[0].Type)
	}
	if defs[0].Function.Name != "read_file" {
		t.Errorf("expected read_file, got %s", defs[0].Function.Name)
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line one\nline two\nline three\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &ReadFile{}
	output, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestReadFileWithOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line one\nline two\nline three\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &ReadFile{}
	output, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
		"offset":    float64(1),
		"limit":     float64(1),
	})
	if err != nil {
		t.Fatal(err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestReadFileNotFound(t *testing.T) {
	tool := &ReadFile{}
	_, err := tool.Execute(context.Background(), map[string]any{
		"file_path": "/nonexistent/file.txt",
	})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "test.txt")

	tool := &WriteFile{}
	output, err := tool.Execute(context.Background(), map[string]any{
		"file_path": path,
		"content":   "hello world",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}

	// Verify file was written
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got %s", string(data))
	}
}

func TestEditFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "edit.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	tool := &EditFile{}
	output, err := tool.Execute(context.Background(), map[string]any{
		"file_path":  path,
		"old_string": "world",
		"new_string": "gopher",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hello gopher" {
		t.Errorf("expected 'hello gopher', got %s", string(data))
	}
}

func TestGlob(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte(""), 0644)

	tool := &Glob{}
	output, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "*.go",
		"path":    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if output == "" || output == "No files matched." {
		t.Error("expected files to match")
	}
}

func TestGrep(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main\nfunc main() {\n}\n"), 0644)

	tool := &Grep{}
	output, err := tool.Execute(context.Background(), map[string]any{
		"pattern": "func main",
		"path":    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if output == "" || output == "No files matched." {
		t.Error("expected files to match")
	}
}
