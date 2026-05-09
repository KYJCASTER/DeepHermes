package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ad201/deephermes/app/events"
	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/config"
	"github.com/ad201/deephermes/pkg/tools"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main application struct. Its public methods are bound to the frontend.
type App struct {
	ctx      context.Context
	cfg      *config.Config
	client   *api.Client
	registry *tools.Registry

	sessions   map[string]*Session
	sessionsMu sync.RWMutex

	subAgents   map[string]*SubAgentRuntime
	subAgentsMu sync.RWMutex
}

type Session struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Agent     *agent.Agent       `json:"-"`
	Messages  []api.Message      `json:"messages"`
	Model     string             `json:"model"`
	Cancel    context.CancelFunc `json:"-"`
	CreatedAt time.Time          `json:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt"`
	Usage     TokenUsage         `json:"usage"`
	LastRun   *RunMetrics        `json:"lastRun,omitempty"`
}

type TokenUsage struct {
	PromptTokens          int `json:"promptTokens"`
	CompletionTokens      int `json:"completionTokens"`
	TotalTokens           int `json:"totalTokens"`
	PromptCacheHitTokens  int `json:"promptCacheHitTokens"`
	PromptCacheMissTokens int `json:"promptCacheMissTokens"`
	ReasoningTokens       int `json:"reasoningTokens"`
}

type RunMetrics struct {
	Usage        TokenUsage `json:"usage"`
	StartedAt    string     `json:"startedAt"`
	FirstTokenAt string     `json:"firstTokenAt,omitempty"`
	FinishedAt   string     `json:"finishedAt"`
	FirstTokenMs int64      `json:"firstTokenMs"`
	DurationMs   int64      `json:"durationMs"`
	TokensPerSec float64    `json:"tokensPerSec"`
}

type SubAgentRuntime struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	AgentType string             `json:"agentType"`
	Status    string             `json:"status"`
	Cancel    context.CancelFunc `json:"-"`
	Result    string             `json:"result"`
	CreatedAt time.Time          `json:"createdAt"`
}

// NewApp creates the application with loaded config.
func NewApp(cfg *config.Config) *App {
	client := api.NewClient(cfg.API.BaseURL, cfg.Model, cfg.GetAPIKey(), cfg.API.MaxRetries)
	client.UpdateConfig(cfg.API.BaseURL, cfg.Model, cfg.GetAPIKey(), cfg.API.MaxRetries, cfg.API.TimeoutSeconds, cfg.ThinkingEnabled)
	reg := tools.NewRegistry()
	registerTools(reg)

	return &App{
		cfg:       cfg,
		client:    client,
		registry:  reg,
		sessions:  make(map[string]*Session),
		subAgents: make(map[string]*SubAgentRuntime),
	}
}

func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	if err := a.loadPersistedSessions(); err != nil {
		runtime.LogError(ctx, fmt.Sprintf("failed to load sessions: %v", err))
	}
}

func (a *App) OnShutdown(ctx context.Context) {
	a.sessionsMu.RLock()
	var sessionIDs []string
	for id := range a.sessions {
		sessionIDs = append(sessionIDs, id)
	}
	a.sessionsMu.RUnlock()
	for _, id := range sessionIDs {
		_ = a.saveSessionByID(id)
	}

	// Cancel all running sub-agents
	a.subAgentsMu.Lock()
	for _, sa := range a.subAgents {
		if sa.Cancel != nil {
			sa.Cancel()
		}
	}
	a.subAgentsMu.Unlock()
}

func (a *App) agentConfig() agent.Config {
	return agent.Config{
		WorkDir:     agent.GetWorkDir(),
		Model:       a.cfg.Model,
		MaxTokens:   a.cfg.MaxTokens,
		Temperature: a.cfg.Temperature,
	}
}

func (a *App) syncClientConfig() {
	a.client.UpdateConfig(
		a.cfg.API.BaseURL,
		a.cfg.Model,
		a.cfg.GetAPIKey(),
		a.cfg.API.MaxRetries,
		a.cfg.API.TimeoutSeconds,
		a.cfg.ThinkingEnabled,
	)
}

func (a *App) syncSessionConfigs() {
	cfg := a.agentConfig()
	a.sessionsMu.Lock()
	defer a.sessionsMu.Unlock()
	for _, sess := range a.sessions {
		sess.Model = a.cfg.Model
		sess.Agent.UpdateConfig(cfg)
	}
}

// ============================================================================
// Session Management
// ============================================================================

type CreateSessionResult struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Model     string `json:"model"`
	CreatedAt string `json:"createdAt"`
}

func (a *App) CreateSession(name string) (*CreateSessionResult, error) {
	id := fmt.Sprintf("session-%d", time.Now().UnixMilli())
	if strings.TrimSpace(name) == "" {
		name = "New Session"
	}
	ag := agent.New(a.client, a.registry, a.agentConfig())
	now := time.Now()

	sess := &Session{
		ID:        id,
		Name:      name,
		Agent:     ag,
		Model:     a.cfg.Model,
		CreatedAt: now,
		UpdatedAt: now,
	}

	a.sessionsMu.Lock()
	a.sessions[id] = sess
	a.sessionsMu.Unlock()
	_ = a.saveSessionByID(id)

	runtime.EventsEmit(a.ctx, string(events.EventSessionUpdate), events.SessionUpdatePayload{
		SessionID: id,
		Name:      name,
		Action:    "created",
	})

	return &CreateSessionResult{
		ID:        id,
		Name:      name,
		Model:     a.cfg.Model,
		CreatedAt: sess.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (a *App) DeleteSession(sessionID string) error {
	a.sessionsMu.Lock()
	delete(a.sessions, sessionID)
	a.sessionsMu.Unlock()
	_ = a.deleteStoredSession(sessionID)

	runtime.EventsEmit(a.ctx, string(events.EventSessionUpdate), events.SessionUpdatePayload{
		SessionID: sessionID,
		Action:    "deleted",
	})
	return nil
}

type SessionInfo struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Model     string      `json:"model"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
	MsgCount  int         `json:"msgCount"`
	Usage     TokenUsage  `json:"usage"`
	LastRun   *RunMetrics `json:"lastRun,omitempty"`
}

func (a *App) ListSessions() []SessionInfo {
	a.sessionsMu.RLock()
	defer a.sessionsMu.RUnlock()

	var list []SessionInfo
	for _, s := range a.sessions {
		list = append(list, SessionInfo{
			ID:        s.ID,
			Name:      s.Name,
			Model:     s.Model,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
			UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
			MsgCount:  len(s.Messages),
			Usage:     s.Usage,
			LastRun:   s.LastRun,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt > list[j].UpdatedAt
	})
	return list
}

// ============================================================================
// Chat
// ============================================================================

type SendMessageRequest struct {
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
}

func (a *App) SendMessage(req SendMessageRequest) error {
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		return fmt.Errorf("message is required")
	}

	a.sessionsMu.Lock()
	sess, ok := a.sessions[req.SessionID]
	if !ok {
		a.sessionsMu.Unlock()
		return fmt.Errorf("session not found: %s", req.SessionID)
	}
	if sess.Cancel != nil {
		a.sessionsMu.Unlock()
		return fmt.Errorf("session is already running")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	history := append([]api.Message(nil), sess.Messages...)
	now := time.Now()
	sess.Cancel = cancel
	sess.Model = a.cfg.Model
	sess.UpdatedAt = now
	sess.Messages = append(sess.Messages, api.Message{Role: "user", Content: req.Message})
	sess.Agent.SetMessages(history)
	sess.Agent.UpdateConfig(a.agentConfig())
	a.sessionsMu.Unlock()
	_ = a.saveSessionByID(req.SessionID)

	// Emit status: thinking
	emit(a.ctx, req.SessionID, events.EventAgentStatus, events.AgentStatusPayload{
		Status: "thinking",
		Model:  sess.Model,
	})

	go func() {
		defer func() {
			cancel()
			a.sessionsMu.Lock()
			if s, ok := a.sessions[req.SessionID]; ok {
				s.Cancel = nil
			}
			a.sessionsMu.Unlock()
		}()

		lastSave := time.Now()
		result, err := sess.Agent.RunStreamDetailed(ctx, req.Message, func(update api.StreamUpdate) error {
			if update.Content == "" && update.ReasoningContent == "" {
				return nil
			}
			a.appendAssistantDelta(req.SessionID, update.Content, update.ReasoningContent)
			emit(a.ctx, req.SessionID, events.EventStreamDelta, events.StreamDeltaPayload{
				Content:          update.Content,
				ReasoningContent: update.ReasoningContent,
			})
			if time.Since(lastSave) > 750*time.Millisecond {
				_ = a.saveSessionByID(req.SessionID)
				lastSave = time.Now()
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				_ = a.saveSessionByID(req.SessionID)
				emit(a.ctx, req.SessionID, events.EventStreamDone, events.StreamDonePayload{})
				emit(a.ctx, req.SessionID, events.EventAgentStatus, events.AgentStatusPayload{
					Status: "idle",
					Model:  sess.Model,
				})
				return
			}
			emit(a.ctx, req.SessionID, events.EventError, events.ErrorPayload{
				Message: err.Error(),
				Code:    "AGENT_ERROR",
			})
			emit(a.ctx, req.SessionID, events.EventAgentStatus, events.AgentStatusPayload{
				Status: "idle",
				Model:  sess.Model,
			})
			return
		}

		if result == nil {
			result = &agent.RunResult{FinishedAt: time.Now()}
		}
		metrics := runMetricsFromResult(result)
		a.sessionsMu.Lock()
		if s, ok := a.sessions[req.SessionID]; ok {
			if len(s.Messages) == 0 || s.Messages[len(s.Messages)-1].Role != "assistant" {
				s.Messages = append(s.Messages, api.Message{Role: "assistant"})
			}
			last := s.Messages[len(s.Messages)-1]
			last.Content = result.Content
			last.ReasoningContent = result.ReasoningContent
			s.Messages[len(s.Messages)-1] = last
			s.UpdatedAt = time.Now()
			if metrics != nil {
				s.LastRun = metrics
				s.Usage.Add(metrics.Usage)
			}
		}
		a.sessionsMu.Unlock()
		_ = a.saveSessionByID(req.SessionID)

		emit(a.ctx, req.SessionID, events.EventStreamDone, events.StreamDonePayload{
			FullResponse: result.Content,
			Usage:        metricsUsage(metrics),
			Metrics:      metrics,
		})
		emit(a.ctx, req.SessionID, events.EventAgentStatus, events.AgentStatusPayload{
			Status: "idle",
			Model:  sess.Model,
		})
	}()

	return nil
}

func (a *App) AbortMessage(sessionID string) {
	a.sessionsMu.Lock()
	if sess, ok := a.sessions[sessionID]; ok && sess.Cancel != nil {
		sess.Cancel()
		sess.Cancel = nil
		sess.UpdatedAt = time.Now()
	}
	a.sessionsMu.Unlock()
	_ = a.saveSessionByID(sessionID)
	emit(a.ctx, sessionID, events.EventStreamDone, events.StreamDonePayload{})
	emit(a.ctx, sessionID, events.EventAgentStatus, events.AgentStatusPayload{
		Status: "idle",
		Model:  "",
	})
}

func (a *App) GetHistory(sessionID string) []api.Message {
	a.sessionsMu.RLock()
	defer a.sessionsMu.RUnlock()
	if sess, ok := a.sessions[sessionID]; ok {
		return sess.Messages
	}
	return nil
}

// ============================================================================
// Tool Execution (for tool visualization)
// ============================================================================

type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (a *App) ListTools() []ToolInfo {
	var list []ToolInfo
	for _, t := range a.registry.List() {
		list = append(list, ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
		})
	}
	return list
}

// ============================================================================
// File System
// ============================================================================

type FileEntry struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	IsDir    bool        `json:"isDir"`
	Size     int64       `json:"size"`
	Children []FileEntry `json:"children,omitempty"`
}

func (a *App) ListDirectory(dirPath string) ([]FileEntry, error) {
	if dirPath == "" {
		dirPath = "."
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	var result []FileEntry
	for _, e := range entries {
		if shouldHideFileEntry(e.Name()) {
			continue
		}
		info, _ := e.Info()
		entry := FileEntry{
			Name:  e.Name(),
			Path:  filepath.Join(dirPath, e.Name()),
			IsDir: e.IsDir(),
		}
		if info != nil {
			entry.Size = info.Size()
		}
		if e.IsDir() {
			entry.Children = []FileEntry{}
		}
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}

func (a *App) ReadFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) GetWorkspaceDir() string {
	wd, _ := os.Getwd()
	return wd
}

func (a *App) OpenFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select File",
	})
}

func (a *App) OpenDirectoryDialog() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Directory",
	})
}

// ============================================================================
// Settings
// ============================================================================

type AppSettings struct {
	Model            string  `json:"model"`
	MaxTokens        int     `json:"maxTokens"`
	Temperature      float64 `json:"temperature"`
	BaseURL          string  `json:"baseUrl"`
	ThinkingEnabled  bool    `json:"thinkingEnabled"`
	ReasoningDisplay string  `json:"reasoningDisplay"`
	AutoCowork       bool    `json:"autoCowork"`
}

func (a *App) GetSettings() AppSettings {
	return AppSettings{
		Model:            a.cfg.Model,
		MaxTokens:        a.cfg.MaxTokens,
		Temperature:      a.cfg.Temperature,
		BaseURL:          a.cfg.API.BaseURL,
		ThinkingEnabled:  a.cfg.ThinkingEnabled,
		ReasoningDisplay: normalizeReasoningDisplay(a.cfg.ReasoningDisplay),
		AutoCowork:       a.cfg.AutoCowork,
	}
}

func (a *App) UpdateSettings(settings AppSettings) error {
	settings.Model = strings.TrimSpace(settings.Model)
	settings.BaseURL = strings.TrimRight(strings.TrimSpace(settings.BaseURL), "/")
	if settings.Model == "" {
		return fmt.Errorf("model is required")
	}
	if settings.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if settings.MaxTokens <= 0 {
		return fmt.Errorf("max tokens must be positive")
	}
	a.cfg.Model = settings.Model
	a.cfg.MaxTokens = settings.MaxTokens
	a.cfg.Temperature = settings.Temperature
	a.cfg.API.BaseURL = settings.BaseURL
	a.cfg.ThinkingEnabled = settings.ThinkingEnabled
	a.cfg.ReasoningDisplay = normalizeReasoningDisplay(settings.ReasoningDisplay)
	a.cfg.AutoCowork = settings.AutoCowork
	a.syncClientConfig()
	a.syncSessionConfigs()
	if err := a.cfg.Save(); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	return nil
}

func (a *App) SetAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("API key is required")
	}
	a.cfg.APIKey = key
	a.syncClientConfig()
	if err := a.cfg.Save(); err != nil {
		return fmt.Errorf("failed to save API key: %w", err)
	}
	runtime.EventsEmit(a.ctx, "settings:updated", map[string]string{"apiKeyStatus": "configured"})
	return nil
}

func (a *App) GetAPIKeyStatus() string {
	if a.cfg.GetAPIKey() != "" {
		return "configured"
	}
	return "missing"
}

// ============================================================================
// Cowork - Sub-agent Management
// ============================================================================

type SpawnSubAgentRequest struct {
	ParentSessionID string `json:"parentSessionId"`
	Name            string `json:"name"`
	AgentType       string `json:"agentType"`
	Task            string `json:"task"`
}

func (a *App) SpawnSubAgent(req SpawnSubAgentRequest) (string, error) {
	id := fmt.Sprintf("subagent-%d", time.Now().UnixMilli())

	ctx, cancel := context.WithCancel(a.ctx)

	sa := &SubAgentRuntime{
		ID:        id,
		Name:      req.Name,
		AgentType: req.AgentType,
		Status:    "running",
		Cancel:    cancel,
		CreatedAt: time.Now(),
	}

	a.subAgentsMu.Lock()
	a.subAgents[id] = sa
	a.subAgentsMu.Unlock()

	// Emit cowork update
	emitCowork(a.ctx, id, req.Name, "running", req.AgentType, "")

	// Build sub-agent with restricted tools
	subReg := buildRegistryForType(req.AgentType)
	subAgent := agent.New(a.client, subReg, a.agentConfig())

	go func() {
		defer cancel()
		resp, err := subAgent.Run(ctx, buildSubAgentPrompt(req.AgentType, req.Task))

		a.subAgentsMu.Lock()
		defer a.subAgentsMu.Unlock()

		if err != nil {
			sa.Status = "failed"
			sa.Result = err.Error()
			emitCowork(a.ctx, id, req.Name, "failed", req.AgentType, err.Error())
		} else {
			sa.Status = "done"
			sa.Result = resp
			emitCowork(a.ctx, id, req.Name, "done", req.AgentType, truncate(resp, 500))
		}
	}()

	return id, nil
}

func (a *App) CancelSubAgent(subAgentID string) {
	a.subAgentsMu.Lock()
	defer a.subAgentsMu.Unlock()
	if sa, ok := a.subAgents[subAgentID]; ok {
		if sa.Cancel != nil {
			sa.Cancel()
		}
		sa.Status = "failed"
		emitCowork(a.ctx, subAgentID, sa.Name, "failed", sa.AgentType, "cancelled")
	}
}

type SubAgentStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AgentType string `json:"agentType"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	Result    string `json:"result"`
}

func (a *App) GetSubAgents() []SubAgentStatus {
	a.subAgentsMu.RLock()
	defer a.subAgentsMu.RUnlock()

	var list []SubAgentStatus
	for _, sa := range a.subAgents {
		list = append(list, SubAgentStatus{
			ID:        sa.ID,
			Name:      sa.Name,
			AgentType: sa.AgentType,
			Status:    sa.Status,
			CreatedAt: sa.CreatedAt.Format(time.RFC3339),
			Result:    truncate(sa.Result, 200),
		})
	}
	return list
}

// ============================================================================
// DeepSeek-specific
// ============================================================================

func (a *App) SetThinking(enabled bool) {
	a.cfg.ThinkingEnabled = enabled
	a.client.SetThinking(enabled)
	_ = a.cfg.Save()
}

func (a *App) GetModelInfo() map[string]any {
	return map[string]any{
		"current":   a.cfg.Model,
		"available": []string{"deepseek-chat", "deepseek-reasoner", "deepseek-v4-flash", "deepseek-v4-pro"},
		"contextWindow": map[string]int{
			"deepseek-chat":     1000000,
			"deepseek-reasoner": 1000000,
			"deepseek-v4-flash": 1000000,
			"deepseek-v4-pro":   1000000,
		},
		"supportsThinking": []string{"deepseek-reasoner", "deepseek-v4-flash", "deepseek-v4-pro"},
	}
}

// ============================================================================
// Helpers
// ============================================================================

func registerTools(reg *tools.Registry) {
	reg.Register(&tools.ReadFile{})
	reg.Register(&tools.WriteFile{})
	reg.Register(&tools.EditFile{})
	reg.Register(&tools.Bash{})
	reg.Register(&tools.Glob{})
	reg.Register(&tools.Grep{})
	reg.Register(&tools.WebFetch{})
	reg.Register(&tools.WebSearch{})
}

func buildRegistryForType(agentType string) *tools.Registry {
	reg := tools.NewRegistry()
	switch agentType {
	case "explore":
		reg.Register(&tools.ReadFile{})
		reg.Register(&tools.Glob{})
		reg.Register(&tools.Grep{})
		reg.Register(&tools.WebFetch{})
		reg.Register(&tools.WebSearch{})
	case "implement":
		reg.Register(&tools.ReadFile{})
		reg.Register(&tools.WriteFile{})
		reg.Register(&tools.EditFile{})
		reg.Register(&tools.Bash{})
		reg.Register(&tools.Glob{})
		reg.Register(&tools.Grep{})
	case "review":
		reg.Register(&tools.ReadFile{})
		reg.Register(&tools.Glob{})
		reg.Register(&tools.Grep{})
	default:
		registerTools(reg)
	}
	return reg
}

func buildSubAgentPrompt(agentType, task string) string {
	var sb strings.Builder
	switch agentType {
	case "explore":
		sb.WriteString("You are a code exploration agent. Search and read code to answer questions. Report findings concisely. Do NOT modify files.\n\n")
	case "implement":
		sb.WriteString("You are an implementation agent. Write and edit code to complete the task. Be thorough.\n\n")
	case "review":
		sb.WriteString("You are a code review agent. Review changes for bugs, security issues, and code quality. Report findings.\n\n")
	}
	sb.WriteString("Task: ")
	sb.WriteString(task)
	return sb.String()
}

func emit(ctx context.Context, sessionID string, t events.EventType, payload any) {
	evt := events.NewEvent(t, sessionID, payload)
	raw, _ := json.Marshal(evt)
	runtime.EventsEmit(ctx, string(t), string(raw))
}

func emitCowork(ctx context.Context, id, name, status, agentType, result string) {
	runtime.EventsEmit(ctx, string(events.EventCoworkUpdate), events.CoworkUpdatePayload{
		SubAgentID: id,
		Name:       name,
		Status:     status,
		Type:       agentType,
		Result:     result,
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func normalizeReasoningDisplay(mode string) string {
	switch mode {
	case "show", "collapse", "hide":
		return mode
	default:
		return "collapse"
	}
}

func shouldHideFileEntry(name string) bool {
	switch name {
	case ".git", ".gocache-codex", "node_modules", "dist", "build":
		return true
	default:
		return false
	}
}
