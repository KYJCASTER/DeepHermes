package agent

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

func BuildSystemPrompt(cfg *Config, toolNames []string) string {
	var sb strings.Builder

	sb.WriteString("You are DeepHermes, an AI agent powered by DeepSeek. ")
	sb.WriteString("You help users with software engineering tasks by using tools to read, write, and execute code.\n\n")

	// Environment info
	sb.WriteString("<environment>\n")
	sb.WriteString(fmt.Sprintf("  Working directory: %s\n", cfg.WorkDir))
	sb.WriteString(fmt.Sprintf("  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("  Date: %s\n", time.Now().Format("2006-01-02")))
	sb.WriteString("</environment>\n\n")

	// Tool usage instructions
	sb.WriteString("<instructions>\n")
	sb.WriteString("- Use tools to gather information and make changes on the user's system.\n")
	sb.WriteString("- Read files before editing them; never edit a file you haven't read.\n")
	sb.WriteString("- When making edits, preserve exact indentation.\n")
	sb.WriteString("- Prefer editing existing files rather than creating new ones.\n")
	sb.WriteString("- Use the bash tool for shell commands. Never use multiple separate commands when a single combined command will work.\n")
	sb.WriteString("- Default to writing no comments in code. Only add comments when the WHY is non-obvious.\n")
	sb.WriteString("- Follow the user's instructions carefully. If unsure, ask for clarification.\n")
	sb.WriteString("- Write concise responses. Be direct and to the point.\n")
	sb.WriteString("</instructions>\n\n")

	// Tool listing
	sb.WriteString("<available_tools>\n")
	for _, name := range toolNames {
		sb.WriteString(fmt.Sprintf("  - %s\n", name))
	}
	sb.WriteString("</available_tools>\n")

	return sb.String()
}

type Config struct {
	WorkDir     string
	Model       string
	MaxTokens   int
	Temperature float64
}

func GetWorkDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
