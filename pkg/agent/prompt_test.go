package agent

import (
	"strings"
	"testing"
)

func TestBuildSystemPromptUsesInitialPromptAsPrimaryBehavior(t *testing.T) {
	prompt := BuildSystemPrompt(&Config{
		WorkDir:        ".",
		Mode:           "rp",
		InitialPrompt:  "Stay in character as the tavern keeper.",
		ContextSummary: "The guest is searching for a lost map.",
	}, []string{"read_file"})

	if !strings.Contains(prompt, "<stable_prompt>\nStay in character as the tavern keeper.\n</stable_prompt>") {
		t.Fatalf("expected initial prompt block, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "<dynamic_context>\nThe guest is searching for a lost map.\n</dynamic_context>") {
		t.Fatalf("expected dynamic context block, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Mode: RP") {
		t.Fatalf("expected RP mode profile, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "primary persona, scene, style, and behavior contract") {
		t.Fatalf("expected runtime rule prioritizing the initial prompt")
	}
	if !strings.Contains(prompt, "do not decide the user's character actions") {
		t.Fatalf("expected roleplay agency guardrail")
	}
	if strings.Contains(prompt, "software engineering tasks") {
		t.Fatalf("custom initial prompt should avoid default coding-agent framing")
	}
}
