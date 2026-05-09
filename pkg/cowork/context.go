package cowork

import (
	"strings"
	"sync"
)

// SharedContext holds common project understanding shared across agents.
type SharedContext struct {
	mu       sync.RWMutex
	project  string
	files    map[string]string   // path → snapshot
	decisions []string            // what each agent decided
	notes    []string
}

func NewSharedContext(project string) *SharedContext {
	return &SharedContext{
		project: project,
		files:   make(map[string]string),
	}
}

func (c *SharedContext) AddFile(path, content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.files) < 20 { // limit snapshots
		c.files[path] = content
	}
}

func (c *SharedContext) AddDecision(decision string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.decisions = append(c.decisions, decision)
	if len(c.decisions) > 50 {
		c.decisions = c.decisions[1:]
	}
}

func (c *SharedContext) AddNote(note string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notes = append(c.notes, note)
}

func (c *SharedContext) ToPrompt() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("<shared-context>\n")
	sb.WriteString("Project: " + c.project + "\n")

	if len(c.files) > 0 {
		sb.WriteString("\nKey files:\n")
		for path, content := range c.files {
			sb.WriteString("- " + path + " (" + truncateStr(content, 100) + ")\n")
		}
	}

	if len(c.decisions) > 0 {
		sb.WriteString("\nRecent decisions:\n")
		for _, d := range c.decisions {
			sb.WriteString("- " + d + "\n")
		}
	}

	sb.WriteString("</shared-context>\n")
	return sb.String()
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
