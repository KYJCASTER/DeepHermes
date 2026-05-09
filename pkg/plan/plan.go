package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Mode string

const (
	Normal Mode = "normal"
	Plan   Mode = "plan"
)

type Manager struct {
	mode   Mode
	dir    string
	plan   *PlanFile
}

type PlanFile struct {
	Path    string
	Content string
}

func NewManager(dir string) *Manager {
	return &Manager{
		mode: Normal,
		dir:  dir,
	}
}

func (m *Manager) Mode() Mode { return m.mode }

func (m *Manager) IsPlanMode() bool { return m.mode == Plan }

func (m *Manager) EnterPlanMode() error {
	os.MkdirAll(m.dir, 0755)
	filename := fmt.Sprintf("plan-%s.md", time.Now().Format("2006-01-02-150405"))
	path := filepath.Join(m.dir, filename)

	template := `# Plan

## Goal

<!-- Describe what you want to accomplish -->

## Approach

<!-- Outline the approach -->

## Steps

1.
2.
3.

## Verification

<!-- How will you verify the changes work? -->
`
	if err := os.WriteFile(path, []byte(template), 0644); err != nil {
		return err
	}

	m.mode = Plan
	m.plan = &PlanFile{Path: path, Content: template}
	return nil
}

func (m *Manager) ExitPlanMode() error {
	m.mode = Normal
	m.plan = nil
	return nil
}

func (m *Manager) PlanPath() string {
	if m.plan != nil {
		return m.plan.Path
	}
	return ""
}

func (m *Manager) PlanContent() string {
	if m.plan == nil {
		return ""
	}
	data, err := os.ReadFile(m.plan.Path)
	if err != nil {
		return ""
	}
	return string(data)
}

func (m *Manager) UpdatePlan(newContent string) error {
	if m.plan == nil {
		return fmt.Errorf("not in plan mode")
	}
	m.plan.Content = newContent
	return os.WriteFile(m.plan.Path, []byte(newContent), 0644)
}

func (m *Manager) SystemPromptModifier() string {
	if m.mode != Plan {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<plan-mode>\n")
	sb.WriteString("You are in PLAN MODE. Do NOT make any changes to files or execute destructive commands.\n")
	sb.WriteString("You can read files and search the codebase to understand the problem.\n")
	sb.WriteString("Design your approach and write it to the plan file.\n")
	if m.plan != nil {
		sb.WriteString(fmt.Sprintf("Current plan file: %s\n", m.plan.Path))
		content := m.PlanContent()
		if content != "" {
			sb.WriteString("\nCurrent plan content:\n")
			sb.WriteString("```\n")
			sb.WriteString(content)
			sb.WriteString("\n```\n")
		}
	}
	sb.WriteString("</plan-mode>\n")
	return sb.String()
}
