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
	initialPrompt := strings.TrimSpace(cfg.InitialPrompt)

	if initialPrompt != "" {
		sb.WriteString("You are DeepHermes, an AI client powered by DeepSeek.\n\n")

		writeEnvironment(&sb, cfg)

		writeModeProfile(&sb, cfg.Mode)

		sb.WriteString("<stable_prompt>\n")
		sb.WriteString(initialPrompt)
		sb.WriteString("\n</stable_prompt>\n\n")

		writeDynamicContext(&sb, cfg.ContextSummary)

		sb.WriteString("<runtime_rules>\n")
		sb.WriteString("- Treat the user initial prompt as the primary persona, scene, style, and behavior contract.\n")
		sb.WriteString("- Keep replies consistent with the established character, world, tone, relationships, and conversation memory.\n")
		sb.WriteString("- In roleplay or story sessions, do not decide the user's character actions, thoughts, dialogue, or choices.\n")
		sb.WriteString("- Use tools only when the user explicitly asks for local files, code, shell commands, or system actions.\n")
		sb.WriteString("- If tool use is required, read files before editing them and preserve exact indentation.\n")
		writeShellRule(&sb)
		sb.WriteString("- If the prompt lacks a detail needed for continuity, ask briefly or make a light, reversible assumption.\n")
		sb.WriteString("</runtime_rules>\n\n")

		writeToolList(&sb, toolNames)
		return sb.String()
	}

	sb.WriteString("You are DeepHermes, an AI agent powered by DeepSeek. ")
	sb.WriteString("You help users with software engineering tasks by using tools to read, write, and execute code.\n\n")

	writeEnvironment(&sb, cfg)
	writeModeProfile(&sb, cfg.Mode)
	writeDynamicContext(&sb, cfg.ContextSummary)

	// Tool usage instructions
	sb.WriteString("<instructions>\n")
	sb.WriteString("- Use tools to gather information and make changes on the user's system.\n")
	sb.WriteString("- Read files before editing them; never edit a file you haven't read.\n")
	sb.WriteString("- When making edits, preserve exact indentation.\n")
	sb.WriteString("- Prefer editing existing files rather than creating new ones.\n")
	sb.WriteString("- Use the bash tool for shell commands. Never use multiple separate commands when a single combined command will work.\n")
	writeShellRule(&sb)
	sb.WriteString("- Default to writing no comments in code. Only add comments when the WHY is non-obvious.\n")
	sb.WriteString("- Follow the user's instructions carefully. If unsure, ask for clarification.\n")
	sb.WriteString("- Write concise responses. Be direct and to the point.\n")
	sb.WriteString("</instructions>\n\n")

	writeToolList(&sb, toolNames)

	return sb.String()
}

func writeEnvironment(sb *strings.Builder, cfg *Config) {
	sb.WriteString("<environment>\n")
	sb.WriteString(fmt.Sprintf("  Working directory: %s\n", cfg.WorkDir))
	sb.WriteString(fmt.Sprintf("  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	if runtime.GOOS == "windows" {
		sb.WriteString("  Shell: Windows PowerShell syntax via the bash tool\n")
	} else {
		sb.WriteString("  Shell: bash-compatible POSIX shell\n")
	}
	sb.WriteString(fmt.Sprintf("  Date: %s\n", time.Now().Format("2006-01-02")))
	sb.WriteString("</environment>\n\n")
}

func writeShellRule(sb *strings.Builder) {
	if runtime.GOOS == "windows" {
		sb.WriteString("- The bash tool runs Windows PowerShell here: use PowerShell commands and chain commands with `;`, not `&&`.\n")
		return
	}
	sb.WriteString("- The bash tool runs a POSIX shell here: use bash-compatible syntax.\n")
}

func writeModeProfile(sb *strings.Builder, mode string) {
	sb.WriteString("<mode_profile>\n")
	switch mode {
	case "rp":
		sb.WriteString("  Mode: RP\n")
		sb.WriteString("  Focus: character consistency, scene continuity, emotional texture, and player agency.\n")
		sb.WriteString("  Keep stable persona/lore separate from changing scene state; treat recent chat as current action.\n")
	case "writing":
		sb.WriteString("  Mode: Writing\n")
		sb.WriteString("  Focus: long-form prose, structure, style consistency, continuity, and revision quality.\n")
		sb.WriteString("  Preserve the stable brief while evolving the dynamic outline and recent draft context.\n")
	case "chat":
		sb.WriteString("  Mode: Lightweight Chat\n")
		sb.WriteString("  Focus: concise answers, low overhead, and minimal tool use unless explicitly requested.\n")
	default:
		sb.WriteString("  Mode: Code\n")
		sb.WriteString("  Focus: software engineering, codebase accuracy, careful tool use, and concise implementation notes.\n")
		sb.WriteString("  Keep system instructions and tool definitions stable to improve DeepSeek context-cache hits.\n")
	}
	sb.WriteString("</mode_profile>\n\n")
}

func writeDynamicContext(sb *strings.Builder, summary string) {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		sb.WriteString("<dynamic_context>\n")
		sb.WriteString("  No older-message summary yet. Recent chat messages are sent after this system prompt.\n")
		sb.WriteString("</dynamic_context>\n\n")
		return
	}
	sb.WriteString("<dynamic_context>\n")
	sb.WriteString(summary)
	sb.WriteString("\n</dynamic_context>\n\n")
}

func writeToolList(sb *strings.Builder, toolNames []string) {
	sb.WriteString("<available_tools>\n")
	for _, name := range toolNames {
		sb.WriteString(fmt.Sprintf("  - %s\n", name))
	}
	sb.WriteString("</available_tools>\n")
}

type Config struct {
	WorkDir        string
	Model          string
	Mode           string
	MaxTokens      int
	Temperature    float64
	InitialPrompt  string
	ContextSummary string
}

func GetWorkDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
