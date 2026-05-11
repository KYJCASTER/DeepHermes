package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const storedSessionVersion = 1

type storedSession struct {
	Version        int           `json:"version"`
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Model          string        `json:"model"`
	Messages       []api.Message `json:"messages"`
	AgentMessages  []api.Message `json:"agentMessages,omitempty"`
	ContextSummary string        `json:"contextSummary,omitempty"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
	Usage          TokenUsage    `json:"usage"`
	LastRun        *RunMetrics   `json:"lastRun,omitempty"`
}

type sessionBackup struct {
	Version   int             `json:"version"`
	CreatedAt time.Time       `json:"createdAt"`
	Sessions  []storedSession `json:"sessions"`
}

type SessionStorageResult struct {
	Path     string `json:"path"`
	Sessions int    `json:"sessions"`
}

type ExportSessionRequest struct {
	SessionID string `json:"sessionId"`
	Format    string `json:"format"`
}

func (a *App) loadPersistedSessions() error {
	dir, err := a.sessionsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	a.sessionsMu.Lock()
	defer a.sessionsMu.Unlock()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		stored, err := parseStoredSession(data)
		if err != nil {
			_ = a.quarantineCorruptSession(filepath.Join(dir, entry.Name()), err)
			continue
		}
		if stored.ID == "" {
			continue
		}
		stored = normalizeStoredSession(stored, a.cfg.Model)

		ag := agent.New(a.client, a.registry, a.agentConfig())
		agentMessages := append([]api.Message(nil), stored.AgentMessages...)
		if len(agentMessages) == 0 {
			agentMessages = append([]api.Message(nil), stored.Messages...)
		}
		ag.SetMessages(agentMessages)
		a.sessions[stored.ID] = &Session{
			ID:             stored.ID,
			Name:           stored.Name,
			Agent:          ag,
			Messages:       stored.Messages,
			AgentMessages:  ag.Messages(),
			ContextSummary: stored.ContextSummary,
			Model:          stored.Model,
			CreatedAt:      stored.CreatedAt,
			UpdatedAt:      stored.UpdatedAt,
			Usage:          stored.Usage,
			LastRun:        stored.LastRun,
		}
	}
	return nil
}

func (a *App) BackupSessions() (*SessionStorageResult, error) {
	path, err := a.sessionSaveDialog("Backup DeepHermes Sessions", "deephermes-sessions-backup.json", "JSON files (*.json)", "*.json")
	if err != nil || strings.TrimSpace(path) == "" {
		return nil, err
	}
	count, err := a.backupSessionsToPath(path)
	if err != nil {
		a.recordLog("error", fmt.Sprintf("session backup failed: %v", err))
		return nil, err
	}
	a.recordLog("info", fmt.Sprintf("sessions backed up to %s", path))
	return &SessionStorageResult{Path: path, Sessions: count}, nil
}

func (a *App) RestoreSessions() (*SessionStorageResult, error) {
	path, err := a.sessionOpenDialog("Restore DeepHermes Sessions", "JSON files (*.json)", "*.json")
	if err != nil || strings.TrimSpace(path) == "" {
		return nil, err
	}
	count, err := a.restoreSessionsFromPath(path)
	if err != nil {
		a.recordLog("error", fmt.Sprintf("session restore failed: %v", err))
		return nil, err
	}
	a.recordLog("info", fmt.Sprintf("sessions restored from %s", path))
	return &SessionStorageResult{Path: path, Sessions: count}, nil
}

func (a *App) ExportSession(req ExportSessionRequest) (*SessionStorageResult, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	format := normalizeSessionExportFormat(req.Format)
	if req.SessionID == "" {
		return nil, fmt.Errorf("session id is required")
	}
	name := a.sessionExportDefaultName(req.SessionID, format)
	filterName := "Markdown files (*.md)"
	pattern := "*.md"
	if format == "json" {
		filterName = "JSON files (*.json)"
		pattern = "*.json"
	}
	path, err := a.sessionSaveDialog("Export DeepHermes Session", name, filterName, pattern)
	if err != nil || strings.TrimSpace(path) == "" {
		return nil, err
	}
	if err := a.exportSessionToPath(req.SessionID, format, path); err != nil {
		a.recordLog("error", fmt.Sprintf("session export failed: %v", err))
		return nil, err
	}
	a.recordLog("info", fmt.Sprintf("session exported to %s", path))
	return &SessionStorageResult{Path: path, Sessions: 1}, nil
}

func (a *App) saveSessionByID(sessionID string) error {
	a.sessionsMu.RLock()
	sess, ok := a.sessions[sessionID]
	if !ok {
		a.sessionsMu.RUnlock()
		return nil
	}
	stored := snapshotSession(sess)
	a.sessionsMu.RUnlock()
	return a.saveStoredSession(stored)
}

func (a *App) saveStoredSession(stored storedSession) error {
	dir, err := a.sessionsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	stored = normalizeStoredSession(stored, a.cfg.Model)
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, stored.ID+".json"), data, 0600)
}

func (a *App) deleteStoredSession(sessionID string) error {
	dir, err := a.sessionsDir()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(dir, sessionID+".json"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (a *App) sessionsDir() (string, error) {
	if a.cfg != nil {
		return a.cfg.SessionsDir(), nil
	}
	return filepath.Abs(filepath.Join(".deephermes", "sessions"))
}

func snapshotSession(sess *Session) storedSession {
	messages := append([]api.Message(nil), sess.Messages...)
	agentMessages := append([]api.Message(nil), sess.AgentMessages...)
	if len(agentMessages) == 0 {
		agentMessages = append([]api.Message(nil), messages...)
	}
	return storedSession{
		Version:        storedSessionVersion,
		ID:             sess.ID,
		Name:           sess.Name,
		Model:          sess.Model,
		Messages:       messages,
		AgentMessages:  agentMessages,
		ContextSummary: sess.ContextSummary,
		CreatedAt:      sess.CreatedAt,
		UpdatedAt:      sess.UpdatedAt,
		Usage:          sess.Usage,
		LastRun:        sess.LastRun,
	}
}

func parseStoredSession(data []byte) (storedSession, error) {
	var stored storedSession
	if err := json.Unmarshal(data, &stored); err != nil {
		return storedSession{}, err
	}
	return stored, nil
}

func normalizeStoredSession(stored storedSession, defaultModel string) storedSession {
	if stored.Version <= 0 {
		stored.Version = storedSessionVersion
	}
	if strings.TrimSpace(stored.Name) == "" {
		stored.Name = "New Session"
	}
	if stored.Model == "" {
		stored.Model = defaultModel
	}
	if stored.CreatedAt.IsZero() {
		stored.CreatedAt = time.Now()
	}
	if stored.UpdatedAt.IsZero() {
		stored.UpdatedAt = stored.CreatedAt
	}
	return stored
}

func (a *App) quarantineCorruptSession(path string, cause error) error {
	dir := filepath.Join(filepath.Dir(path), "corrupt")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	name := filepath.Base(path) + "." + time.Now().UTC().Format("20060102T150405Z") + ".corrupt"
	target := filepath.Join(dir, name)
	if err := os.Rename(path, target); err != nil {
		return err
	}
	a.recordLog("warn", fmt.Sprintf("moved corrupt session %s to %s: %v", path, target, cause))
	return nil
}

func (a *App) backupSessionsToPath(path string) (int, error) {
	a.sessionsMu.RLock()
	sessions := make([]storedSession, 0, len(a.sessions))
	for _, sess := range a.sessions {
		sessions = append(sessions, snapshotSession(sess))
	}
	a.sessionsMu.RUnlock()
	backup := sessionBackup{
		Version:   storedSessionVersion,
		CreatedAt: time.Now().UTC(),
		Sessions:  sessions,
	}
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return 0, err
	}
	return len(sessions), nil
}

func (a *App) restoreSessionsFromPath(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var backup sessionBackup
	if err := json.Unmarshal(data, &backup); err != nil || len(backup.Sessions) == 0 {
		var single storedSession
		if singleErr := json.Unmarshal(data, &single); singleErr != nil {
			if err != nil {
				return 0, err
			}
			return 0, singleErr
		}
		if strings.TrimSpace(single.ID) != "" {
			backup.Sessions = []storedSession{single}
		}
		if len(backup.Sessions) == 0 {
			return 0, fmt.Errorf("backup contains no sessions")
		}
	}
	count := 0
	for _, stored := range backup.Sessions {
		stored = normalizeStoredSession(stored, a.cfg.Model)
		if strings.TrimSpace(stored.ID) == "" {
			continue
		}
		ag := agent.New(a.client, a.registry, a.agentConfig())
		agentMessages := append([]api.Message(nil), stored.AgentMessages...)
		if len(agentMessages) == 0 {
			agentMessages = append([]api.Message(nil), stored.Messages...)
		}
		ag.SetMessages(agentMessages)
		sess := &Session{
			ID:             stored.ID,
			Name:           stored.Name,
			Agent:          ag,
			Messages:       append([]api.Message(nil), stored.Messages...),
			AgentMessages:  ag.Messages(),
			ContextSummary: stored.ContextSummary,
			Model:          stored.Model,
			CreatedAt:      stored.CreatedAt,
			UpdatedAt:      stored.UpdatedAt,
			Usage:          stored.Usage,
			LastRun:        stored.LastRun,
		}
		a.sessionsMu.Lock()
		a.sessions[stored.ID] = sess
		a.sessionsMu.Unlock()
		if err := a.saveStoredSession(snapshotSession(sess)); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (a *App) exportSessionToPath(sessionID, format, path string) error {
	a.sessionsMu.RLock()
	sess, ok := a.sessions[sessionID]
	if !ok {
		a.sessionsMu.RUnlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}
	stored := snapshotSession(sess)
	a.sessionsMu.RUnlock()

	var data []byte
	var err error
	switch normalizeSessionExportFormat(format) {
	case "json":
		data, err = json.MarshalIndent(stored, "", "  ")
	default:
		data = []byte(sessionMarkdown(stored))
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func sessionMarkdown(stored storedSession) string {
	var out bytes.Buffer
	fmt.Fprintf(&out, "# %s\n\n", strings.TrimSpace(stored.Name))
	fmt.Fprintf(&out, "- ID: `%s`\n", stored.ID)
	fmt.Fprintf(&out, "- Model: `%s`\n", stored.Model)
	fmt.Fprintf(&out, "- Created: %s\n", stored.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(&out, "- Updated: %s\n", stored.UpdatedAt.Format(time.RFC3339))
	if strings.TrimSpace(stored.ContextSummary) != "" {
		fmt.Fprintf(&out, "\n## Context Summary\n\n%s\n", stored.ContextSummary)
	}
	for _, msg := range stored.Messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			role = "message"
		}
		fmt.Fprintf(&out, "\n## %s\n\n", role)
		if strings.TrimSpace(msg.ReasoningContent) != "" {
			fmt.Fprintln(&out, "> Reasoning omitted from export.")
			fmt.Fprintln(&out)
		}
		fmt.Fprintln(&out, msg.Content)
	}
	return strings.TrimSpace(out.String()) + "\n"
}

func normalizeSessionExportFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return "json"
	default:
		return "markdown"
	}
}

func (a *App) sessionExportDefaultName(sessionID, format string) string {
	a.sessionsMu.RLock()
	name := sessionID
	if sess, ok := a.sessions[sessionID]; ok && strings.TrimSpace(sess.Name) != "" {
		name = sess.Name
	}
	a.sessionsMu.RUnlock()
	name = safeExportFileName(name)
	if normalizeSessionExportFormat(format) == "json" {
		return name + ".json"
	}
	return name + ".md"
}

func safeExportFileName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "session"
	}
	replacer := strings.NewReplacer("\\", "-", "/", "-", ":", "-", "*", "-", "?", "", "\"", "", "<", "", ">", "", "|", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, " .")
	if value == "" {
		return "session"
	}
	if len(value) > 80 {
		value = value[:80]
	}
	return value
}

func (a *App) sessionSaveDialog(title, defaultName, filterName, pattern string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                title,
		DefaultFilename:      defaultName,
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{
			{DisplayName: filterName, Pattern: pattern},
			{DisplayName: "All files (*.*)", Pattern: "*.*"},
		},
	})
}

func (a *App) sessionOpenDialog(title, filterName, pattern string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: filterName, Pattern: pattern},
			{DisplayName: "All files (*.*)", Pattern: "*.*"},
		},
	})
}
