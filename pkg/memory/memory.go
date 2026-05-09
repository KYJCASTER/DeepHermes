package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type MemoryType string

const (
	User      MemoryType = "user"
	Feedback  MemoryType = "feedback"
	Project   MemoryType = "project"
	Reference MemoryType = "reference"
)

type Entry struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Type        MemoryType `yaml:"type"`
	Created     time.Time  `yaml:"created"`
	Updated     time.Time  `yaml:"updated"`
	Content     string     `yaml:"-"`
	FilePath    string     `yaml:"-"`
}

type Store struct {
	dir     string
	index   []*Entry
	loaded  bool
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) Load() error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}
	// Read index
	indexPath := filepath.Join(s.dir, "MEMORY.md")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		s.loaded = true
		return nil // no memories yet
	}

	s.index = nil
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [") {
			continue
		}
		// Parse: - [Title](file.md) — description
		endBracket := strings.Index(line, "]")
		endParen := strings.Index(line, ")")
		if endBracket < 0 || endParen < 0 {
			continue
		}
		title := line[3:endBracket]
		file := line[endBracket+2 : endParen]
		desc := ""
		if dash := strings.Index(line, " — "); dash > 0 {
			desc = line[dash+len(" — "):]
		}

		entry := &Entry{
			Name:        title,
			Description: desc,
			FilePath:    filepath.Join(s.dir, file),
		}
		s.index = append(s.index, entry)
	}

	// Load each entry's frontmatter
	for _, e := range s.index {
		s.loadEntry(e)
	}
	s.loaded = true
	return nil
}

func (s *Store) loadEntry(e *Entry) {
	data, err := os.ReadFile(e.FilePath)
	if err != nil {
		return
	}
	content := string(data)
	// Parse YAML frontmatter between --- markers
	parts := strings.SplitN(content, "---", 3)
	if len(parts) >= 3 {
		var fm struct {
			Name        string     `yaml:"name"`
			Description string     `yaml:"description"`
			Type        MemoryType `yaml:"type"`
			Created     time.Time  `yaml:"created"`
			Updated     time.Time  `yaml:"updated"`
		}
		if err := yaml.Unmarshal([]byte(parts[1]), &fm); err == nil {
			e.Name = fm.Name
			e.Description = fm.Description
			e.Type = fm.Type
			e.Created = fm.Created
			e.Updated = fm.Updated
		}
		e.Content = strings.TrimSpace(parts[2])
	}
}

func (s *Store) Save(entry *Entry) error {
	if !s.loaded {
		s.Load()
	}
	now := time.Now()
	if entry.Created.IsZero() {
		entry.Created = now
	}
	entry.Updated = now

	// Build file path
	filename := sanitizeFilename(entry.Name) + ".md"
	entry.FilePath = filepath.Join(s.dir, filename)

	// Build content with frontmatter
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %q\n", entry.Name))
	sb.WriteString(fmt.Sprintf("description: %q\n", entry.Description))
	sb.WriteString(fmt.Sprintf("type: %q\n", entry.Type))
	sb.WriteString(fmt.Sprintf("created: %q\n", entry.Created.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated: %q\n", entry.Updated.Format(time.RFC3339)))
	sb.WriteString("---\n\n")
	sb.WriteString(entry.Content)

	if err := os.WriteFile(entry.FilePath, []byte(sb.String()), 0644); err != nil {
		return err
	}

	// Update index
	found := false
	for i, e := range s.index {
		if e.FilePath == entry.FilePath {
			s.index[i] = entry
			found = true
			break
		}
	}
	if !found {
		s.index = append(s.index, entry)
	}

	return s.writeIndex()
}

func (s *Store) Delete(name string) error {
	if !s.loaded {
		s.Load()
	}
	var newIndex []*Entry
	for _, e := range s.index {
		if e.Name == name {
			os.Remove(e.FilePath)
		} else {
			newIndex = append(newIndex, e)
		}
	}
	s.index = newIndex
	return s.writeIndex()
}

func (s *Store) List() []*Entry {
	if !s.loaded {
		s.Load()
	}
	sort.Slice(s.index, func(i, j int) bool {
		return s.index[i].Updated.After(s.index[j].Updated)
	})
	return s.index
}

func (s *Store) Get(name string) *Entry {
	if !s.loaded {
		s.Load()
	}
	for _, e := range s.index {
		if e.Name == name {
			return e
		}
	}
	return nil
}

func (s *Store) GetByType(t MemoryType) []*Entry {
	if !s.loaded {
		s.Load()
	}
	var result []*Entry
	for _, e := range s.index {
		if e.Type == t {
			result = append(result, e)
		}
	}
	return result
}

func (s *Store) SystemPromptContext() string {
	if !s.loaded {
		s.Load()
	}
	if len(s.index) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<memory>\n")
	for _, e := range s.index {
		sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", e.Type, e.Name, truncate(e.Content, 200)))
	}
	sb.WriteString("</memory>\n")
	return sb.String()
}

func (s *Store) writeIndex() error {
	var sb strings.Builder
	sb.WriteString("# Memory Index\n\n")
	for _, e := range s.index {
		sb.WriteString(fmt.Sprintf("- [%s](%s) — %s\n",
			e.Name, filepath.Base(e.FilePath), e.Description))
	}
	return os.WriteFile(filepath.Join(s.dir, "MEMORY.md"), []byte(sb.String()), 0644)
}

func sanitizeFilename(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, name)
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
