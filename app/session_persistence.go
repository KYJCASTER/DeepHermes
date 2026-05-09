package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
)

type storedSession struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Model     string        `json:"model"`
	Messages  []api.Message `json:"messages"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
	Usage     TokenUsage    `json:"usage"`
	LastRun   *RunMetrics   `json:"lastRun,omitempty"`
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
		var stored storedSession
		if err := json.Unmarshal(data, &stored); err != nil {
			continue
		}
		if stored.ID == "" {
			continue
		}
		if strings.TrimSpace(stored.Name) == "" {
			stored.Name = "New Session"
		}
		if stored.Model == "" {
			stored.Model = a.cfg.Model
		}
		if stored.CreatedAt.IsZero() {
			stored.CreatedAt = time.Now()
		}
		if stored.UpdatedAt.IsZero() {
			stored.UpdatedAt = stored.CreatedAt
		}

		ag := agent.New(a.client, a.registry, a.agentConfig())
		ag.SetMessages(stored.Messages)
		a.sessions[stored.ID] = &Session{
			ID:        stored.ID,
			Name:      stored.Name,
			Agent:     ag,
			Messages:  stored.Messages,
			Model:     stored.Model,
			CreatedAt: stored.CreatedAt,
			UpdatedAt: stored.UpdatedAt,
			Usage:     stored.Usage,
			LastRun:   stored.LastRun,
		}
	}
	return nil
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
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Abs(filepath.Join(".deephermes", "sessions"))
	}
	return filepath.Join(home, ".deephermes", "sessions"), nil
}

func snapshotSession(sess *Session) storedSession {
	messages := append([]api.Message(nil), sess.Messages...)
	return storedSession{
		ID:        sess.ID,
		Name:      sess.Name,
		Model:     sess.Model,
		Messages:  messages,
		CreatedAt: sess.CreatedAt,
		UpdatedAt: sess.UpdatedAt,
		Usage:     sess.Usage,
		LastRun:   sess.LastRun,
	}
}
