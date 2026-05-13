package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ad201/deephermes/pkg/api"
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

func TestRegistryReadOnlyBlocksWriteTools(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&WriteFile{})
	reg.SetPolicy(Policy{Mode: string(ToolModeReadOnly)})

	_, err := reg.Execute(context.Background(), api.ToolCall{
		ID: "call-write",
		Function: api.FunctionCall{
			Name:      "write_file",
			Arguments: `{"file_path":"test.txt","content":"blocked"}`,
		},
	})
	if err == nil {
		t.Fatal("expected read-only mode to block write_file")
	}
}

func TestRegistryConfirmPolicyUsesApprovalCallback(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&WriteFile{})
	dir := t.TempDir()
	path := filepath.Join(dir, "approved.txt")
	var requested ApprovalRequest
	reg.SetPolicy(Policy{
		Mode: string(ToolModeConfirm),
		Approval: func(ctx context.Context, req ApprovalRequest) (ApprovalDecision, error) {
			requested = req
			return ApprovalDecision{Approved: true}, nil
		},
	})

	_, err := reg.Execute(WithSessionID(context.Background(), "session-1"), api.ToolCall{
		ID: "call-write",
		Function: api.FunctionCall{
			Name:      "write_file",
			Arguments: `{"file_path":"` + filepath.ToSlash(path) + `","content":"approved"}`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if requested.SessionID != "session-1" {
		t.Fatalf("expected session id to be forwarded, got %q", requested.SessionID)
	}
	if requested.Risk != string(ToolRiskWrite) {
		t.Fatalf("expected write risk, got %q", requested.Risk)
	}
}

func TestRegistryToolOverrideAllowsWriteInReadOnlyMode(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&WriteFile{})
	dir := t.TempDir()
	path := filepath.Join(dir, "override.txt")
	reg.SetPolicy(Policy{
		Mode: string(ToolModeReadOnly),
		ToolOverrides: map[string]string{
			"write_file": string(ToolModeAuto),
		},
	})

	_, err := reg.Execute(context.Background(), api.ToolCall{
		ID: "call-write-override",
		Function: api.FunctionCall{
			Name:      "write_file",
			Arguments: `{"file_path":"` + filepath.ToSlash(path) + `","content":"allowed"}`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "allowed" {
		t.Fatalf("expected override to allow write, got %q", string(data))
	}
}

func TestRegistryToolOverrideCanTightenPolicy(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&WriteFile{})
	reg.SetPolicy(Policy{
		Mode: string(ToolModeAuto),
		ToolOverrides: map[string]string{
			"write_file": string(ToolModeReadOnly),
		},
	})

	_, err := reg.Execute(context.Background(), api.ToolCall{
		ID: "call-write-blocked",
		Function: api.FunctionCall{
			Name:      "write_file",
			Arguments: `{"file_path":"blocked.txt","content":"blocked"}`,
		},
	})
	if err == nil {
		t.Fatal("expected tool override to block write_file")
	}
	if !strings.Contains(err.Error(), "read-only mode") {
		t.Fatalf("expected read-only error, got %v", err)
	}
}

func TestRegistryBashBlocklistBlocksCommand(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Bash{})
	reg.SetPolicy(Policy{
		Mode:          string(ToolModeAuto),
		BashBlocklist: []string{"rm -rf /"},
	})

	_, err := reg.Execute(context.Background(), api.ToolCall{
		ID: "call-bash-blocked",
		Function: api.FunctionCall{
			Name:      "bash",
			Arguments: `{"command":"echo before && rm -rf /"}`,
		},
	})
	if err == nil {
		t.Fatal("expected bash blocklist to reject command")
	}
	if !strings.Contains(err.Error(), "blocked pattern") {
		t.Fatalf("expected blocklist error, got %v", err)
	}
}

func TestRegistryAllowedDirBlocksReadOutsideWorkspace(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ReadFile{})
	workspace := t.TempDir()
	outside := t.TempDir()
	path := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(path, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	reg.SetPolicy(Policy{
		Mode:       string(ToolModeAuto),
		AllowedDir: workspace,
	})

	_, err := reg.Execute(context.Background(), api.ToolCall{
		ID: "call-read-outside",
		Function: api.FunctionCall{
			Name:      "read_file",
			Arguments: `{"file_path":"` + filepath.ToSlash(path) + `"}`,
		},
	})
	if err == nil {
		t.Fatal("expected allowed dir to block read_file outside workspace")
	}
	if !strings.Contains(err.Error(), "outside the allowed workspace") {
		t.Fatalf("expected workspace boundary error, got %v", err)
	}
}

func TestRegistryAllowedDirBlocksWriteBeforeApproval(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&WriteFile{})
	workspace := t.TempDir()
	outside := t.TempDir()
	path := filepath.Join(outside, "blocked.txt")
	approvalCalled := false
	reg.SetPolicy(Policy{
		Mode:       string(ToolModeConfirm),
		AllowedDir: workspace,
		Approval: func(ctx context.Context, req ApprovalRequest) (ApprovalDecision, error) {
			approvalCalled = true
			return ApprovalDecision{Approved: true}, nil
		},
	})

	_, err := reg.Execute(context.Background(), api.ToolCall{
		ID: "call-write-outside",
		Function: api.FunctionCall{
			Name:      "write_file",
			Arguments: `{"file_path":"` + filepath.ToSlash(path) + `","content":"blocked"}`,
		},
	})
	if err == nil {
		t.Fatal("expected allowed dir to block write_file outside workspace")
	}
	if approvalCalled {
		t.Fatal("expected workspace boundary check to run before approval callback")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("expected outside file not to be created, stat err: %v", statErr)
	}
}

func TestRegistryAllowedDirAllowsDefaultGrepPath(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&Grep{})
	workspace := t.TempDir()
	path := filepath.Join(workspace, "main.go")
	if err := os.WriteFile(path, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)
	reg.SetPolicy(Policy{
		Mode:       string(ToolModeAuto),
		AllowedDir: workspace,
	})

	output, err := reg.Execute(context.Background(), api.ToolCall{
		ID: "call-grep-default",
		Function: api.FunctionCall{
			Name:      "grep",
			Arguments: `{"pattern":"func main","glob":"*.go"}`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "main.go") {
		t.Fatalf("expected grep to find file inside workspace, got %q", output)
	}
}

func TestValidatePathRejectsSiblingWithSamePrefix(t *testing.T) {
	parent := t.TempDir()
	workspace := filepath.Join(parent, "repo")
	sibling := filepath.Join(parent, "repo-other")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sibling, 0755); err != nil {
		t.Fatal(err)
	}

	err := ValidatePath(workspace, filepath.Join(sibling, "file.txt"))
	if err == nil {
		t.Fatal("expected sibling directory with same prefix to be rejected")
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
