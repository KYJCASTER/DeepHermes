package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	stdruntime "runtime"
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
	"gopkg.in/yaml.v3"
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

	logsMu sync.Mutex
	logs   []DiagnosticLog
}

type Session struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Agent          *agent.Agent       `json:"-"`
	Messages       []api.Message      `json:"messages"`
	AgentMessages  []api.Message      `json:"-"`
	ContextSummary string             `json:"contextSummary,omitempty"`
	Model          string             `json:"model"`
	Cancel         context.CancelFunc `json:"-"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
	Usage          TokenUsage         `json:"usage"`
	LastRun        *RunMetrics        `json:"lastRun,omitempty"`
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

type DiagnosticLog struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type AppDiagnostics struct {
	Version        string          `json:"version"`
	BuildCommit    string          `json:"buildCommit"`
	BuildDate      string          `json:"buildDate"`
	GoVersion      string          `json:"goVersion"`
	Platform       string          `json:"platform"`
	Arch           string          `json:"arch"`
	ConfigPath     string          `json:"configPath"`
	DataDir        string          `json:"dataDir"`
	SessionsDir    string          `json:"sessionsDir"`
	Portable       bool            `json:"portable"`
	MinimizeToTray bool            `json:"minimizeToTray"`
	Model          string          `json:"model"`
	Mode           string          `json:"mode"`
	BaseURL        string          `json:"baseUrl"`
	APIKeyStatus   string          `json:"apiKeyStatus"`
	SessionCount   int             `json:"sessionCount"`
	MemoryDir      string          `json:"memoryDir"`
	RecentLogs     []DiagnosticLog `json:"recentLogs"`
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
		a.recordLog("error", fmt.Sprintf("failed to load sessions: %v", err))
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

func (a *App) OnBeforeClose(ctx context.Context) bool {
	if !a.cfg.MinimizeToTray {
		return false
	}
	runtime.WindowHide(ctx)
	a.recordLog("info", "window hidden to background")
	return true
}

func (a *App) HideMainWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowHide(a.ctx)
	a.recordLog("info", "window hidden to background")
}

func (a *App) RestoreMainWindow() {
	if a.ctx == nil {
		return
	}
	runtime.Show(a.ctx)
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
	a.recordLog("info", "window restored")
}

func (a *App) QuitApp() {
	if a.ctx == nil {
		return
	}
	runtime.Quit(a.ctx)
}

func (a *App) agentConfig() agent.Config {
	return a.agentConfigWithSummary("")
}

func (a *App) agentConfigWithSummary(summary string) agent.Config {
	return agent.Config{
		WorkDir:        agent.GetWorkDir(),
		Model:          a.cfg.Model,
		Mode:           normalizeAppMode(a.cfg.Mode),
		MaxTokens:      a.cfg.MaxTokens,
		Temperature:    a.cfg.Temperature,
		InitialPrompt:  a.composedInitialPrompt(),
		ContextSummary: summary,
	}
}

func (a *App) sessionAgentConfig(sess *Session) agent.Config {
	if sess == nil {
		return a.agentConfig()
	}
	return a.agentConfigWithSummary(sess.ContextSummary)
}

func (a *App) composedInitialPrompt() string {
	var parts []string
	if prompt := strings.TrimSpace(a.cfg.InitialPrompt); prompt != "" {
		parts = append(parts, "<session_rules>\n"+prompt+"\n</session_rules>")
	}
	if role := strings.TrimSpace(a.cfg.RoleCard); role != "" {
		parts = append(parts, "<role_card>\n"+role+"\n</role_card>")
	}
	if world := strings.TrimSpace(a.cfg.WorldBook); world != "" {
		parts = append(parts, "<world_book>\n"+world+"\n</world_book>")
	}
	return strings.Join(parts, "\n\n")
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
	a.sessionsMu.Lock()
	defer a.sessionsMu.Unlock()
	for _, sess := range a.sessions {
		sess.Model = a.cfg.Model
		sess.Agent.UpdateConfig(a.sessionAgentConfig(sess))
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

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, string(events.EventSessionUpdate), events.SessionUpdatePayload{
			SessionID: id,
			Name:      name,
			Action:    "created",
		})
	}

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
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	Model                string      `json:"model"`
	CreatedAt            string      `json:"createdAt"`
	UpdatedAt            string      `json:"updatedAt"`
	MsgCount             int         `json:"msgCount"`
	Usage                TokenUsage  `json:"usage"`
	LastRun              *RunMetrics `json:"lastRun,omitempty"`
	ContextSummaryTokens int         `json:"contextSummaryTokens"`
}

func (a *App) ListSessions() []SessionInfo {
	a.sessionsMu.RLock()
	defer a.sessionsMu.RUnlock()

	var list []SessionInfo
	for _, s := range a.sessions {
		list = append(list, SessionInfo{
			ID:                   s.ID,
			Name:                 s.Name,
			Model:                s.Model,
			CreatedAt:            s.CreatedAt.Format(time.RFC3339),
			UpdatedAt:            s.UpdatedAt.Format(time.RFC3339),
			MsgCount:             len(s.Messages),
			Usage:                s.Usage,
			LastRun:              s.LastRun,
			ContextSummaryTokens: approxContextTokens(s.ContextSummary),
		})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt > list[j].UpdatedAt
	})
	return list
}

func sessionAPIHistory(sess *Session) []api.Message {
	if len(sess.AgentMessages) > 0 {
		return append([]api.Message(nil), sess.AgentMessages...)
	}
	return append([]api.Message(nil), sess.Messages...)
}

// ============================================================================
// Chat
// ============================================================================

type SendMessageRequest struct {
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
}

type MessageIndexRequest struct {
	SessionID string `json:"sessionId"`
	Index     int    `json:"index"`
}

type UpdateMessageRequest struct {
	SessionID string `json:"sessionId"`
	Index     int    `json:"index"`
	Content   string `json:"content"`
}

type BranchSessionRequest struct {
	SessionID  string `json:"sessionId"`
	UpToIndex  int    `json:"upToIndex"`
	NameSuffix string `json:"nameSuffix"`
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
	history := a.prepareSessionContextLocked(sess)
	now := time.Now()
	sess.Cancel = cancel
	sess.Model = a.cfg.Model
	sess.UpdatedAt = now
	sess.Messages = append(sess.Messages, api.Message{Role: "user", Content: req.Message})
	sess.Agent.SetMessages(history)
	sess.Agent.UpdateConfig(a.sessionAgentConfig(sess))
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
			a.recordLog("error", err.Error())
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
			s.AgentMessages = s.Agent.Messages()
			s.ContextSummary = sess.ContextSummary
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

func (a *App) runSessionStream(sessionID string, sess *Session, ctx context.Context, cancel context.CancelFunc, userMessage string) {
	emit(a.ctx, sessionID, events.EventAgentStatus, events.AgentStatusPayload{
		Status: "thinking",
		Model:  sess.Model,
	})

	go func() {
		defer func() {
			cancel()
			a.sessionsMu.Lock()
			if s, ok := a.sessions[sessionID]; ok {
				s.Cancel = nil
			}
			a.sessionsMu.Unlock()
		}()

		lastSave := time.Now()
		result, err := sess.Agent.RunStreamDetailed(ctx, userMessage, func(update api.StreamUpdate) error {
			if update.Content == "" && update.ReasoningContent == "" {
				return nil
			}
			a.appendAssistantDelta(sessionID, update.Content, update.ReasoningContent)
			emit(a.ctx, sessionID, events.EventStreamDelta, events.StreamDeltaPayload{
				Content:          update.Content,
				ReasoningContent: update.ReasoningContent,
			})
			if time.Since(lastSave) > 750*time.Millisecond {
				_ = a.saveSessionByID(sessionID)
				lastSave = time.Now()
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				_ = a.saveSessionByID(sessionID)
				emit(a.ctx, sessionID, events.EventStreamDone, events.StreamDonePayload{})
				emit(a.ctx, sessionID, events.EventAgentStatus, events.AgentStatusPayload{
					Status: "idle",
					Model:  sess.Model,
				})
				return
			}
			a.recordLog("error", err.Error())
			emit(a.ctx, sessionID, events.EventError, events.ErrorPayload{
				Message: err.Error(),
				Code:    "AGENT_ERROR",
			})
			emit(a.ctx, sessionID, events.EventAgentStatus, events.AgentStatusPayload{
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
		if s, ok := a.sessions[sessionID]; ok {
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
			s.AgentMessages = s.Agent.Messages()
			s.ContextSummary = sess.ContextSummary
		}
		a.sessionsMu.Unlock()
		_ = a.saveSessionByID(sessionID)

		emit(a.ctx, sessionID, events.EventStreamDone, events.StreamDonePayload{
			FullResponse: result.Content,
			Usage:        metricsUsage(metrics),
			Metrics:      metrics,
		})
		emit(a.ctx, sessionID, events.EventAgentStatus, events.AgentStatusPayload{
			Status: "idle",
			Model:  sess.Model,
		})
	}()
}

func (a *App) UpdateMessage(req UpdateMessageRequest) error {
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return fmt.Errorf("message content is required")
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
	if req.Index < 0 || req.Index >= len(sess.Messages) {
		a.sessionsMu.Unlock()
		return fmt.Errorf("message index out of range")
	}
	msg := sess.Messages[req.Index]
	if msg.Role == "tool" || msg.Role == "system" {
		a.sessionsMu.Unlock()
		return fmt.Errorf("cannot edit %s messages", msg.Role)
	}
	msg.Content = req.Content
	sess.Messages[req.Index] = msg
	sess.UpdatedAt = time.Now()
	sess.AgentMessages = append([]api.Message(nil), sess.Messages...)
	sess.ContextSummary = ""
	sess.Agent.SetMessages(sess.Messages)
	a.sessionsMu.Unlock()

	return a.saveSessionByID(req.SessionID)
}

func (a *App) DeleteMessage(req MessageIndexRequest) error {
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
	if req.Index < 0 || req.Index >= len(sess.Messages) {
		a.sessionsMu.Unlock()
		return fmt.Errorf("message index out of range")
	}
	sess.Messages = append(sess.Messages[:req.Index], sess.Messages[req.Index+1:]...)
	sess.UpdatedAt = time.Now()
	sess.AgentMessages = append([]api.Message(nil), sess.Messages...)
	sess.ContextSummary = ""
	sess.Agent.SetMessages(sess.Messages)
	a.sessionsMu.Unlock()

	return a.saveSessionByID(req.SessionID)
}

func (a *App) BranchSession(req BranchSessionRequest) (*CreateSessionResult, error) {
	a.sessionsMu.Lock()
	source, ok := a.sessions[req.SessionID]
	if !ok {
		a.sessionsMu.Unlock()
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}
	if req.UpToIndex < 0 || req.UpToIndex >= len(source.Messages) {
		a.sessionsMu.Unlock()
		return nil, fmt.Errorf("message index out of range")
	}

	id := fmt.Sprintf("session-%d", time.Now().UnixMilli())
	nameSuffix := strings.TrimSpace(req.NameSuffix)
	if nameSuffix == "" {
		nameSuffix = "Branch"
	}
	name := source.Name + " / " + nameSuffix
	messages := append([]api.Message(nil), source.Messages[:req.UpToIndex+1]...)
	agentMessages := append([]api.Message(nil), messages...)
	now := time.Now()
	ag := agent.New(a.client, a.registry, a.agentConfig())
	ag.SetMessages(agentMessages)
	branch := &Session{
		ID:            id,
		Name:          name,
		Agent:         ag,
		Messages:      messages,
		AgentMessages: agentMessages,
		Model:         source.Model,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	a.sessions[id] = branch
	a.sessionsMu.Unlock()

	if err := a.saveSessionByID(id); err != nil {
		return nil, err
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, string(events.EventSessionUpdate), events.SessionUpdatePayload{
			SessionID: id,
			Name:      name,
			Action:    "created",
		})
	}

	return &CreateSessionResult{
		ID:        id,
		Name:      name,
		Model:     branch.Model,
		CreatedAt: now.Format(time.RFC3339),
	}, nil
}

func (a *App) RegenerateMessage(req MessageIndexRequest) error {
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
	if req.Index < 0 || req.Index >= len(sess.Messages) {
		a.sessionsMu.Unlock()
		return fmt.Errorf("message index out of range")
	}

	userIndex := req.Index
	if sess.Messages[userIndex].Role == "assistant" {
		userIndex--
	}
	if userIndex < 0 || sess.Messages[userIndex].Role != "user" {
		a.sessionsMu.Unlock()
		return fmt.Errorf("regenerate needs a user message or its assistant response")
	}
	userMessage := strings.TrimSpace(sess.Messages[userIndex].Content)
	if userMessage == "" {
		a.sessionsMu.Unlock()
		return fmt.Errorf("message content is required")
	}
	history := append([]api.Message(nil), sess.Messages[:userIndex]...)
	sess.Messages = append([]api.Message(nil), sess.Messages[:userIndex+1]...)
	sess.AgentMessages = append([]api.Message(nil), history...)
	sess.ContextSummary = ""
	history = a.prepareSessionContextLocked(sess)
	ctx, cancel := context.WithCancel(a.ctx)
	now := time.Now()
	sess.Cancel = cancel
	sess.Model = a.cfg.Model
	sess.UpdatedAt = now
	sess.Agent.SetMessages(history)
	sess.Agent.UpdateConfig(a.sessionAgentConfig(sess))
	a.sessionsMu.Unlock()
	_ = a.saveSessionByID(req.SessionID)

	a.runSessionStream(req.SessionID, sess, ctx, cancel, userMessage)
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
	Mode             string  `json:"mode"`
	Portable         bool    `json:"portable"`
	MinimizeToTray   bool    `json:"minimizeToTray"`
	MaxTokens        int     `json:"maxTokens"`
	Temperature      float64 `json:"temperature"`
	BaseURL          string  `json:"baseUrl"`
	ThinkingEnabled  bool    `json:"thinkingEnabled"`
	ReasoningDisplay string  `json:"reasoningDisplay"`
	AutoCowork       bool    `json:"autoCowork"`
	InitialPrompt    string  `json:"initialPrompt"`
	RoleCard         string  `json:"roleCard"`
	WorldBook        string  `json:"worldBook"`
}

func (a *App) GetSettings() AppSettings {
	return AppSettings{
		Model:            a.cfg.Model,
		Mode:             normalizeAppMode(a.cfg.Mode),
		Portable:         a.cfg.Portable,
		MinimizeToTray:   a.cfg.MinimizeToTray,
		MaxTokens:        a.cfg.MaxTokens,
		Temperature:      a.cfg.Temperature,
		BaseURL:          a.cfg.API.BaseURL,
		ThinkingEnabled:  a.cfg.ThinkingEnabled,
		ReasoningDisplay: normalizeReasoningDisplay(a.cfg.ReasoningDisplay),
		AutoCowork:       a.cfg.AutoCowork,
		InitialPrompt:    a.cfg.InitialPrompt,
		RoleCard:         a.cfg.RoleCard,
		WorldBook:        a.cfg.WorldBook,
	}
}

func (a *App) UpdateSettings(settings AppSettings) error {
	settings.Model = strings.TrimSpace(settings.Model)
	settings.Mode = normalizeAppMode(settings.Mode)
	settings.BaseURL = strings.TrimRight(strings.TrimSpace(settings.BaseURL), "/")
	settings.InitialPrompt = normalizeInitialPrompt(settings.InitialPrompt)
	settings.RoleCard = normalizeInitialPrompt(settings.RoleCard)
	settings.WorldBook = normalizeInitialPrompt(settings.WorldBook)
	if settings.Model == "" {
		return fmt.Errorf("model is required")
	}
	if settings.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if settings.MaxTokens <= 0 {
		return fmt.Errorf("max tokens must be positive")
	}
	if len(settings.InitialPrompt) > 60000 {
		return fmt.Errorf("initial prompt is too long; keep it under 60000 characters")
	}
	if len(settings.RoleCard) > 60000 {
		return fmt.Errorf("role card is too long; keep it under 60000 characters")
	}
	if len(settings.WorldBook) > 60000 {
		return fmt.Errorf("world book is too long; keep it under 60000 characters")
	}
	a.cfg.Model = settings.Model
	a.cfg.Mode = settings.Mode
	a.cfg.Portable = settings.Portable
	a.cfg.MinimizeToTray = settings.MinimizeToTray
	a.cfg.MaxTokens = settings.MaxTokens
	a.cfg.Temperature = settings.Temperature
	a.cfg.API.BaseURL = settings.BaseURL
	a.cfg.ThinkingEnabled = settings.ThinkingEnabled
	a.cfg.ReasoningDisplay = normalizeReasoningDisplay(settings.ReasoningDisplay)
	a.cfg.AutoCowork = settings.AutoCowork
	a.cfg.InitialPrompt = settings.InitialPrompt
	a.cfg.RoleCard = settings.RoleCard
	a.cfg.WorldBook = settings.WorldBook
	a.syncClientConfig()
	a.syncSessionConfigs()
	if err := a.cfg.Save(); err != nil {
		a.recordLog("error", fmt.Sprintf("failed to save settings: %v", err))
		return fmt.Errorf("failed to save settings: %w", err)
	}
	a.recordLog("info", "settings saved")
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
		a.recordLog("error", fmt.Sprintf("failed to save API key: %v", err))
		return fmt.Errorf("failed to save API key: %w", err)
	}
	runtime.EventsEmit(a.ctx, "settings:updated", map[string]string{"apiKeyStatus": "configured"})
	return nil
}

func (a *App) ExportSettings() (string, error) {
	defaultName := "deephermes-settings.yaml"
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "Export DeepHermes Settings",
		DefaultFilename:      defaultName,
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{
			{DisplayName: "YAML files (*.yaml;*.yml)", Pattern: "*.yaml;*.yml"},
			{DisplayName: "All files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		a.recordLog("error", fmt.Sprintf("export settings dialog failed: %v", err))
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}

	exported := *a.cfg
	exported.APIKey = ""
	data, err := yaml.Marshal(&exported)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		a.recordLog("error", fmt.Sprintf("export settings failed: %v", err))
		return "", err
	}
	a.recordLog("info", "settings exported to "+path)
	return path, nil
}

func (a *App) ImportSettings() error {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import DeepHermes Settings",
		Filters: []runtime.FileFilter{
			{DisplayName: "YAML files (*.yaml;*.yml)", Pattern: "*.yaml;*.yml"},
			{DisplayName: "All files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		a.recordLog("error", fmt.Sprintf("import settings dialog failed: %v", err))
		return err
	}
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		a.recordLog("error", fmt.Sprintf("read imported settings failed: %v", err))
		return err
	}
	imported := config.Default()
	if err := yaml.Unmarshal(data, imported); err != nil {
		a.recordLog("error", fmt.Sprintf("parse imported settings failed: %v", err))
		return err
	}
	if strings.TrimSpace(imported.APIKey) == "" {
		imported.APIKey = a.cfg.APIKey
	}
	imported.NormalizePaths()
	a.cfg.Model = imported.Model
	a.cfg.Mode = normalizeAppMode(imported.Mode)
	a.cfg.Portable = imported.Portable
	a.cfg.MinimizeToTray = imported.MinimizeToTray
	a.cfg.MaxTokens = imported.MaxTokens
	a.cfg.Temperature = imported.Temperature
	a.cfg.ThinkingEnabled = imported.ThinkingEnabled
	a.cfg.ReasoningDisplay = normalizeReasoningDisplay(imported.ReasoningDisplay)
	a.cfg.AutoCowork = imported.AutoCowork
	a.cfg.InitialPrompt = imported.InitialPrompt
	a.cfg.RoleCard = imported.RoleCard
	a.cfg.WorldBook = imported.WorldBook
	a.cfg.API = imported.API
	a.cfg.APIKey = imported.APIKey
	a.cfg.AllowedTools = append([]string(nil), imported.AllowedTools...)
	a.cfg.Memory = imported.Memory
	a.cfg.Plans = imported.Plans
	a.cfg.Web = imported.Web
	a.syncClientConfig()
	a.syncSessionConfigs()
	if err := a.cfg.Save(); err != nil {
		a.recordLog("error", fmt.Sprintf("save imported settings failed: %v", err))
		return err
	}
	a.recordLog("info", "settings imported from "+path)
	runtime.EventsEmit(a.ctx, "settings:updated", map[string]string{"apiKeyStatus": a.GetAPIKeyStatus()})
	return nil
}

func (a *App) GetDiagnostics() AppDiagnostics {
	a.sessionsMu.RLock()
	sessionCount := len(a.sessions)
	a.sessionsMu.RUnlock()
	sessionsDir, _ := a.sessionsDir()
	return AppDiagnostics{
		Version:        Version,
		BuildCommit:    BuildCommit,
		BuildDate:      BuildDate,
		GoVersion:      stdruntime.Version(),
		Platform:       stdruntime.GOOS,
		Arch:           stdruntime.GOARCH,
		ConfigPath:     a.cfg.ConfigPath(),
		DataDir:        a.cfg.DataDir(),
		SessionsDir:    sessionsDir,
		Portable:       a.cfg.Portable,
		MinimizeToTray: a.cfg.MinimizeToTray,
		Model:          a.cfg.Model,
		Mode:           normalizeAppMode(a.cfg.Mode),
		BaseURL:        a.cfg.API.BaseURL,
		APIKeyStatus:   a.GetAPIKeyStatus(),
		SessionCount:   sessionCount,
		MemoryDir:      a.cfg.Memory.Dir,
		RecentLogs:     a.recentLogs(),
	}
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
	profiles := map[string]any{
		"deepseek-v4-flash": map[string]any{
			"contextWindow":        1048576,
			"maxOutput":            393216,
			"recommendedMaxTokens": 32768,
			"canReason":            true,
			"cacheHitCnyPerMTok":   0.02,
			"cacheMissCnyPerMTok":  1,
			"outputCnyPerMTok":     2,
		},
		"deepseek-v4-pro": map[string]any{
			"contextWindow":        1048576,
			"maxOutput":            393216,
			"recommendedMaxTokens": 65536,
			"canReason":            true,
			"cacheHitCnyPerMTok":   0.025,
			"cacheMissCnyPerMTok":  3,
			"outputCnyPerMTok":     6,
		},
	}
	return map[string]any{
		"current":   a.cfg.Model,
		"available": []string{"deepseek-v4-flash", "deepseek-v4-pro", "deepseek-chat", "deepseek-reasoner"},
		"profiles":  profiles,
		"contextWindow": map[string]int{
			"deepseek-chat":     1048576,
			"deepseek-reasoner": 1048576,
			"deepseek-v4-flash": 1048576,
			"deepseek-v4-pro":   1048576,
		},
		"supportsThinking": []string{"deepseek-reasoner", "deepseek-v4-flash", "deepseek-v4-pro"},
		"legacyAliases": map[string]string{
			"deepseek-chat":     "deepseek-v4-flash non-thinking mode",
			"deepseek-reasoner": "deepseek-v4-flash thinking mode",
		},
		"legacyDeprecationDate": "2026-07-24",
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

func (a *App) recordLog(level, message string) {
	if a == nil {
		return
	}
	a.logsMu.Lock()
	defer a.logsMu.Unlock()
	a.logs = append(a.logs, DiagnosticLog{
		Time:    time.Now().Format(time.RFC3339),
		Level:   level,
		Message: message,
	})
	if len(a.logs) > 100 {
		a.logs = a.logs[len(a.logs)-100:]
	}
}

func (a *App) recentLogs() []DiagnosticLog {
	a.logsMu.Lock()
	defer a.logsMu.Unlock()
	out := append([]DiagnosticLog(nil), a.logs...)
	if len(out) > 50 {
		out = out[len(out)-50:]
	}
	return out
}

func normalizeReasoningDisplay(mode string) string {
	switch mode {
	case "show", "collapse", "hide":
		return mode
	default:
		return "collapse"
	}
}

func normalizeInitialPrompt(prompt string) string {
	prompt = strings.ReplaceAll(prompt, "\r\n", "\n")
	prompt = strings.ReplaceAll(prompt, "\r", "\n")
	return strings.TrimSpace(prompt)
}

func normalizeAppMode(mode string) string {
	switch mode {
	case "code", "rp", "writing", "chat":
		return mode
	default:
		return "code"
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
