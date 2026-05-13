package tools

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type Bash struct{}

func (t *Bash) Name() string { return "bash" }
func (t *Bash) Description() string {
	if runtime.GOOS == "windows" {
		return "Execute a Windows PowerShell command. Returns stdout and stderr output."
	}
	return "Execute a shell command. Returns stdout and stderr output."
}
func (t *Bash) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": shellCommandDescription(),
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout in milliseconds (max 600000)",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Clear, concise description of what this command does",
			},
		},
		"required": []string{"command"},
	}
}

var workingDir string

func SetWorkingDir(dir string) { workingDir = dir }
func GetWorkingDir() string    { return workingDir }

func getShell() string {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("pwsh.exe"); err == nil {
			return "pwsh.exe"
		}
		if _, err := exec.LookPath("powershell.exe"); err == nil {
			return "powershell.exe"
		}
		return "cmd"
	}
	return "bash"
}

func getShellArgs(shell, command string) []string {
	switch shell {
	case "pwsh.exe":
		return []string{"-NoProfile", "-NonInteractive", "-Command", command}
	case "powershell.exe":
		return []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", normalizeWindowsPowerShellCommand(command)}
	case "cmd":
		return []string{"/C", command}
	default:
		return []string{"-c", command}
	}
}

func shellCommandDescription() string {
	if runtime.GOOS == "windows" {
		return "The Windows PowerShell command to execute. Use PowerShell syntax; use ';' to chain commands instead of '&&'."
	}
	return "The shell command to execute"
}

func normalizeWindowsPowerShellCommand(command string) string {
	if !strings.Contains(command, "&&") {
		return command
	}

	var b strings.Builder
	b.Grow(len(command))
	inSingle := false
	inDouble := false
	for i := 0; i < len(command); i++ {
		ch := command[i]
		if ch == '`' && !inSingle {
			b.WriteByte(ch)
			if i+1 < len(command) {
				i++
				b.WriteByte(command[i])
			}
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			b.WriteByte(ch)
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			b.WriteByte(ch)
			continue
		}
		if !inSingle && !inDouble && ch == '&' && i+1 < len(command) && command[i+1] == '&' {
			b.WriteByte(';')
			i++
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

func (t *Bash) Execute(ctx context.Context, args map[string]any) (string, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return "", fmt.Errorf("command is required")
	}

	timeout := 120 * time.Second
	if v, ok := args["timeout"].(float64); ok && v > 0 {
		timeout = time.Duration(v) * time.Millisecond
		if timeout > 600*time.Second {
			timeout = 600 * time.Second
		}
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	shell := getShell()
	shellArgs := getShellArgs(shell, command)
	cmd := exec.CommandContext(execCtx, shell, shellArgs...)
	if workingDir != "" {
		if err := ValidatePath(AllowedDirFromContext(ctx), workingDir); err != nil {
			return "", err
		}
		cmd.Dir = workingDir
	}

	output, err := cmd.CombinedOutput()
	if execCtx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out after %v", timeout)
	}
	if err != nil {
		return fmt.Sprintf("Exit code: %v\n%s", err, string(output)), nil
	}
	return string(output), nil
}
