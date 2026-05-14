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

var errStopWalk = errors.New("stop walking workspace")

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

	toolApprovalsMu sync.Mutex
	toolApprovals   map[string]chan tools.ApprovalDecision

	toolRollbacksMu sync.Mutex
	toolRollbacks   map[string]ToolRollbackSnapshot
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
	FinishReason string     `json:"finishReason,omitempty"`
	Truncated    bool       `json:"truncated,omitempty"`
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

type ToolRollbackSnapshot struct {
	ToolCallID string
	ToolName   string
	Path       string
	Existed    bool
	Content    []byte
	Mode       os.FileMode
	CreatedAt  time.Time
}

type ToolRollbackResult struct {
	Restored bool   `json:"restored"`
	Deleted  bool   `json:"deleted"`
	Path     string `json:"path"`
	Message  string `json:"message"`
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
	client.UpdateConfig(cfg.API.BaseURL, cfg.Model, cfg.GetAPIKey(), cfg.API.MaxRetries, cfg.API.TimeoutSeconds, cfg.ThinkingEnabled, cfg.API.ProxyURL)
	reg := tools.NewRegistry()
	registerTools(reg)

	app := &App{
		cfg:           cfg,
		client:        client,
		registry:      reg,
		sessions:      make(map[string]*Session),
		subAgents:     make(map[string]*SubAgentRuntime),
		toolApprovals: make(map[string]chan tools.ApprovalDecision),
		toolRollbacks: make(map[string]ToolRollbackSnapshot),
	}
	app.configureToolPolicy()
	return app
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
		a.cfg.API.ProxyURL,
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

func (a *App) configureToolPolicy() {
	a.configureRegistryPolicy(a.registry)
}

func (a *App) configureRegistryPolicy(reg *tools.Registry) {
	if reg == nil {
		return
	}
	workDir := a.GetWorkspaceDir()
	tools.SetWorkingDir(workDir)
	reg.SetPolicy(tools.Policy{
		Mode:          normalizeSafetyToolMode(a.cfg.Safety.ToolMode),
		ToolOverrides: a.cfg.Safety.ToolOverrides,
		BashBlocklist: a.cfg.Safety.BashBlocklist,
		AllowedDir:    workDir,
		Approval:      a.waitForToolApproval,
		OnCall:        a.emitToolCall,
		OnResult:      a.emitToolResult,
	})
}

func (a *App) emitToolCall(ctx context.Context, event tools.ExecutionEvent) {
	a.captureToolRollback(event)
	if a.ctx == nil {
		return
	}
	emit(a.ctx, event.SessionID, events.EventToolCall, events.ToolCallPayload{
		ID:        event.ID,
		Name:      event.ToolName,
		Arguments: event.Arguments,
		Risk:      event.Risk,
	})
}

func (a *App) emitToolResult(ctx context.Context, event tools.ExecutionEvent) {
	if a.ctx == nil {
		return
	}
	rollbackAvailable, rollbackPath := a.rollbackStatusForResult(event)
	emit(a.ctx, event.SessionID, events.EventToolResult, events.ToolResultPayload{
		ToolCallID:        event.ID,
		Name:              event.ToolName,
		Content:           event.Content,
		Success:           event.Success,
		Error:             event.Error,
		Risk:              event.Risk,
		RollbackAvailable: rollbackAvailable,
		RollbackPath:      rollbackPath,
	})
}

func (a *App) captureToolRollback(event tools.ExecutionEvent) {
	if event.ToolName != "write_file" && event.ToolName != "edit_file" {
		return
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(event.Arguments), &args); err != nil {
		return
	}
	path, _ := args["file_path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	snapshot := ToolRollbackSnapshot{
		ToolCallID: event.ID,
		ToolName:   event.ToolName,
		Path:       path,
		CreatedAt:  time.Now(),
	}
	info, statErr := os.Stat(path)
	switch {
	case statErr == nil && !info.IsDir():
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		snapshot.Existed = true
		snapshot.Content = data
		snapshot.Mode = info.Mode()
	case errors.Is(statErr, os.ErrNotExist):
		snapshot.Existed = false
		snapshot.Mode = 0644
	default:
		return
	}
	a.toolRollbacksMu.Lock()
	if a.toolRollbacks == nil {
		a.toolRollbacks = make(map[string]ToolRollbackSnapshot)
	}
	a.toolRollbacks[event.ID] = snapshot
	a.toolRollbacksMu.Unlock()
}

func (a *App) rollbackStatusForResult(event tools.ExecutionEvent) (bool, string) {
	a.toolRollbacksMu.Lock()
	defer a.toolRollbacksMu.Unlock()
	snapshot, ok := a.toolRollbacks[event.ID]
	if !ok {
		return false, ""
	}
	if !event.Success {
		delete(a.toolRollbacks, event.ID)
		return false, ""
	}
	return true, snapshot.Path
}

func (a *App) RollbackToolChange(toolCallID string) (*ToolRollbackResult, error) {
	toolCallID = strings.TrimSpace(toolCallID)
	if toolCallID == "" {
		return nil, fmt.Errorf("tool call id is required")
	}
	a.toolRollbacksMu.Lock()
	snapshot, ok := a.toolRollbacks[toolCallID]
	if ok {
		delete(a.toolRollbacks, toolCallID)
	}
	a.toolRollbacksMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("rollback snapshot not found or already used")
	}
	if snapshot.Existed {
		mode := snapshot.Mode
		if mode == 0 {
			mode = 0644
		}
		if err := os.WriteFile(snapshot.Path, snapshot.Content, mode); err != nil {
			return nil, fmt.Errorf("restore %s: %w", snapshot.Path, err)
		}
		a.recordLog("info", "rolled back tool change: "+snapshot.Path)
		return &ToolRollbackResult{
			Restored: true,
			Path:     snapshot.Path,
			Message:  "File restored to the state before the tool call.",
		}, nil
	}
	if err := os.Remove(snapshot.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("delete new file %s: %w", snapshot.Path, err)
	}
	a.recordLog("info", "deleted file created by tool call: "+snapshot.Path)
	return &ToolRollbackResult{
		Deleted: true,
		Path:    snapshot.Path,
		Message: "New file created by the tool call was deleted.",
	}, nil
}

func (a *App) waitForToolApproval(ctx context.Context, req tools.ApprovalRequest) (tools.ApprovalDecision, error) {
	if a.ctx == nil {
		return tools.ApprovalDecision{}, fmt.Errorf("tool %s requires approval before the UI is ready", req.ToolName)
	}
	ch := make(chan tools.ApprovalDecision, 1)
	a.toolApprovalsMu.Lock()
	if a.toolApprovals == nil {
		a.toolApprovals = make(map[string]chan tools.ApprovalDecision)
	}
	a.toolApprovals[req.ID] = ch
	a.toolApprovalsMu.Unlock()

	defer func() {
		a.toolApprovalsMu.Lock()
		delete(a.toolApprovals, req.ID)
		a.toolApprovalsMu.Unlock()
	}()

	runtime.EventsEmit(a.ctx, string(events.EventToolApproval), events.ToolApprovalPayload{
		ID:        req.ID,
		SessionID: req.SessionID,
		ToolName:  req.ToolName,
		Arguments: req.Arguments,
		Risk:      req.Risk,
		Mode:      req.Mode,
		Preview:   req.Preview,
	})

	select {
	case decision := <-ch:
		return decision, nil
	case <-ctx.Done():
		return tools.ApprovalDecision{}, ctx.Err()
	}
}

func (a *App) ApproveToolCall(id string) error {
	return a.resolveToolApproval(id, tools.ApprovalDecision{Approved: true})
}

func (a *App) RejectToolCall(id string) error {
	return a.resolveToolApproval(id, tools.ApprovalDecision{Approved: false, Reason: "rejected by user"})
}

func (a *App) resolveToolApproval(id string, decision tools.ApprovalDecision) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("approval id is required")
	}
	a.toolApprovalsMu.Lock()
	ch, ok := a.toolApprovals[id]
	a.toolApprovalsMu.Unlock()
	if !ok {
		return fmt.Errorf("approval request not found or already resolved")
	}
	select {
	case ch <- decision:
	default:
	}
	return nil
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
	ctx, cancel := context.WithCancel(tools.WithSessionID(a.ctx, req.SessionID))
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
			friendly := friendlyErrorMessage(err)
			a.recordLog("error", friendly)
			emit(a.ctx, req.SessionID, events.EventError, events.ErrorPayload{
				Message: friendly,
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
			FinishReason: result.FinishReason,
			Truncated:    isTruncatedFinishReason(result.FinishReason),
		})
		emit(a.ctx, req.SessionID, events.EventAgentStatus, events.AgentStatusPayload{
			Status: "idle",
			Model:  sess.Model,
		})
	}()

	return nil
}

func (a *App) ContinueLastResponse(sessionID string) error {
	a.sessionsMu.Lock()
	sess, ok := a.sessions[sessionID]
	if !ok {
		a.sessionsMu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}
	if sess.Cancel != nil {
		a.sessionsMu.Unlock()
		return fmt.Errorf("session is already running")
	}
	if len(sess.Messages) == 0 || sess.Messages[len(sess.Messages)-1].Role != "assistant" {
		a.sessionsMu.Unlock()
		return fmt.Errorf("no assistant message to continue")
	}

	existingContent := sess.Messages[len(sess.Messages)-1].Content
	existingReasoning := sess.Messages[len(sess.Messages)-1].ReasoningContent

	continuePrompt := "Continue from where the previous assistant reply was truncated. Do not repeat existing text. Preserve the current context, voice, and format."

	ctx, cancel := context.WithCancel(tools.WithSessionID(a.ctx, sessionID))
	history := a.prepareSessionContextLocked(sess)
	now := time.Now()
	sess.Cancel = cancel
	sess.Model = a.cfg.Model
	sess.UpdatedAt = now
	sess.Agent.SetMessages(history)
	sess.Agent.UpdateConfig(a.sessionAgentConfig(sess))
	a.sessionsMu.Unlock()
	_ = a.saveSessionByID(sessionID)

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
		result, err := sess.Agent.RunStreamDetailed(ctx, continuePrompt, func(update api.StreamUpdate) error {
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
			friendly := friendlyErrorMessage(err)
			a.recordLog("error", friendly)
			emit(a.ctx, sessionID, events.EventError, events.ErrorPayload{
				Message: friendly,
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
			if len(s.Messages) > 0 && s.Messages[len(s.Messages)-1].Role == "assistant" {
				last := s.Messages[len(s.Messages)-1]
				last.Content = existingContent + result.Content
				last.ReasoningContent = existingReasoning + result.ReasoningContent
				s.Messages[len(s.Messages)-1] = last
			}
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
			FullResponse: existingContent + result.Content,
			Usage:        metricsUsage(metrics),
			Metrics:      metrics,
			FinishReason: result.FinishReason,
			Truncated:    isTruncatedFinishReason(result.FinishReason),
		})
		emit(a.ctx, sessionID, events.EventAgentStatus, events.AgentStatusPayload{
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
			friendly := friendlyErrorMessage(err)
			a.recordLog("error", friendly)
			emit(a.ctx, sessionID, events.EventError, events.ErrorPayload{
				Message: friendly,
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
			FinishReason: result.FinishReason,
			Truncated:    isTruncatedFinishReason(result.FinishReason),
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
	ctx, cancel := context.WithCancel(tools.WithSessionID(a.ctx, req.SessionID))
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

type ContextSummaryResult struct {
	Summary string `json:"summary"`
	Tokens  int    `json:"tokens"`
}

func (a *App) GetContextSummary(sessionID string) ContextSummaryResult {
	a.sessionsMu.RLock()
	defer a.sessionsMu.RUnlock()
	if sess, ok := a.sessions[sessionID]; ok {
		return ContextSummaryResult{
			Summary: sess.ContextSummary,
			Tokens:  approxContextTokens(sess.ContextSummary),
		}
	}
	return ContextSummaryResult{}
}

type UpdateContextSummaryRequest struct {
	SessionID string `json:"sessionId"`
	Summary   string `json:"summary"`
}

func (a *App) UpdateContextSummary(req UpdateContextSummaryRequest) error {
	a.sessionsMu.Lock()
	sess, ok := a.sessions[req.SessionID]
	if !ok {
		a.sessionsMu.Unlock()
		return fmt.Errorf("session not found: %s", req.SessionID)
	}
	sess.ContextSummary = req.Summary
	sess.UpdatedAt = time.Now()
	a.sessionsMu.Unlock()
	return a.saveSessionByID(req.SessionID)
}

func (a *App) ArchiveSession(sessionID string) error {
	a.sessionsMu.Lock()
	sess, ok := a.sessions[sessionID]
	if !ok {
		a.sessionsMu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}
	if sess.Cancel != nil {
		a.sessionsMu.Unlock()
		return fmt.Errorf("cannot archive running session")
	}
	sess.Name = "[Archived] " + sess.Name
	sess.UpdatedAt = time.Now()
	a.sessionsMu.Unlock()
	_ = a.saveSessionByID(sessionID)

	runtime.EventsEmit(a.ctx, string(events.EventSessionUpdate), events.SessionUpdatePayload{
		SessionID: sessionID,
		Action:    "updated",
	})
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

type FileSnippet struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
	Binary    bool   `json:"binary"`
}

type FileSearchResult struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	RelativePath string `json:"relativePath"`
	Size         int64  `json:"size"`
}

func (a *App) ListDirectory(dirPath string) ([]FileEntry, error) {
	if dirPath == "" {
		dirPath = "."
	}
	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, err
	}
	if err := a.validateWorkspacePath(dirPath); err != nil {
		return nil, err
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

func (a *App) SearchWorkspaceFiles(query string, limit int) ([]FileSearchResult, error) {
	query = strings.TrimSpace(query)
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	maxCandidates := limit * 20
	root, err := filepath.Abs(a.GetWorkspaceDir())
	if err != nil {
		return nil, err
	}
	if err := a.validateWorkspacePath(root); err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimPrefix(query, "@"))
	var results []FileSearchResult
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if err := a.validateWorkspacePath(path); err != nil {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if entry.IsDir() {
			if path != root && shouldHideFileEntry(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldHideFileEntry(name) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if needle != "" {
			haystack := strings.ToLower(rel + " " + name)
			if !strings.Contains(haystack, needle) {
				return nil
			}
		}
		info, _ := entry.Info()
		item := FileSearchResult{
			Name:         name,
			Path:         path,
			RelativePath: rel,
		}
		if info != nil {
			item.Size = info.Size()
		}
		results = append(results, item)
		if len(results) >= maxCandidates {
			return errStopWalk
		}
		return nil
	})
	if errors.Is(err, errStopWalk) {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(results, func(i, j int) bool {
		if fileSearchRank(results[i], needle) != fileSearchRank(results[j], needle) {
			return fileSearchRank(results[i], needle) < fileSearchRank(results[j], needle)
		}
		return strings.ToLower(results[i].RelativePath) < strings.ToLower(results[j].RelativePath)
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (a *App) ReadFileContent(path string) (string, error) {
	if err := a.validateWorkspacePath(path); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (a *App) ReadFileSnippet(path string, maxBytes int) (*FileSnippet, error) {
	if maxBytes <= 0 {
		maxBytes = 96 * 1024
	}
	if err := a.validateWorkspacePath(path); err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	truncated := len(data) > maxBytes
	if truncated {
		data = data[:maxBytes]
	}
	snippet := &FileSnippet{
		Name:      filepath.Base(path),
		Path:      path,
		Size:      info.Size(),
		Content:   string(data),
		Truncated: truncated,
		Binary:    looksBinaryContent(data),
	}
	if snippet.Binary {
		snippet.Content = ""
	}
	return snippet, nil
}

func (a *App) GetWorkspaceDir() string {
	wd, _ := os.Getwd()
	return wd
}

func (a *App) validateWorkspacePath(path string) error {
	return tools.ValidatePath(a.GetWorkspaceDir(), path)
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
	Model            string            `json:"model"`
	Mode             string            `json:"mode"`
	Portable         bool              `json:"portable"`
	MinimizeToTray   bool              `json:"minimizeToTray"`
	MaxTokens        int               `json:"maxTokens"`
	Temperature      float64           `json:"temperature"`
	BaseURL          string            `json:"baseUrl"`
	APITimeout       int               `json:"apiTimeout"`
	APIMaxRetries    int               `json:"apiMaxRetries"`
	APIProxyURL      string            `json:"apiProxyUrl"`
	ThinkingEnabled  bool              `json:"thinkingEnabled"`
	ReasoningDisplay string            `json:"reasoningDisplay"`
	AutoCowork       bool              `json:"autoCowork"`
	ToolMode         string            `json:"toolMode"`
	ToolOverrides    map[string]string `json:"toolOverrides"`
	BashBlocklist    []string          `json:"bashBlocklist"`
	InitialPrompt    string            `json:"initialPrompt"`
	RoleCard         string            `json:"roleCard"`
	WorldBook        string            `json:"worldBook"`
	OCREnabled       bool              `json:"ocrEnabled"`
	OCRProvider      string            `json:"ocrProvider"`
	OCRBaseURL       string            `json:"ocrBaseUrl"`
	OCRModel         string            `json:"ocrModel"`
	OCRPrompt        string            `json:"ocrPrompt"`
	OCRTimeout       int               `json:"ocrTimeout"`
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
		APITimeout:       a.cfg.API.TimeoutSeconds,
		APIMaxRetries:    a.cfg.API.MaxRetries,
		APIProxyURL:      a.cfg.API.ProxyURL,
		ThinkingEnabled:  a.cfg.ThinkingEnabled,
		ReasoningDisplay: normalizeReasoningDisplay(a.cfg.ReasoningDisplay),
		AutoCowork:       a.cfg.AutoCowork,
		ToolMode:         normalizeSafetyToolMode(a.cfg.Safety.ToolMode),
		ToolOverrides:    normalizeSafetyToolOverrides(a.cfg.Safety.ToolOverrides),
		BashBlocklist:    normalizeBashBlocklist(a.cfg.Safety.BashBlocklist),
		InitialPrompt:    a.cfg.InitialPrompt,
		RoleCard:         a.cfg.RoleCard,
		WorldBook:        a.cfg.WorldBook,
		OCREnabled:       a.cfg.OCR.Enabled,
		OCRProvider:      normalizeOCRProvider(a.cfg.OCR.Provider),
		OCRBaseURL:       a.cfg.OCR.BaseURL,
		OCRModel:         a.cfg.OCR.Model,
		OCRPrompt:        a.cfg.OCR.Prompt,
		OCRTimeout:       a.cfg.OCR.TimeoutSeconds,
	}
}

func (a *App) UpdateSettings(settings AppSettings) error {
	settings.Model = strings.TrimSpace(settings.Model)
	settings.Mode = normalizeAppMode(settings.Mode)
	settings.BaseURL = strings.TrimRight(strings.TrimSpace(settings.BaseURL), "/")
	settings.APIProxyURL = strings.TrimSpace(settings.APIProxyURL)
	settings.ToolMode = normalizeSafetyToolMode(settings.ToolMode)
	settings.ToolOverrides = normalizeSafetyToolOverrides(settings.ToolOverrides)
	settings.BashBlocklist = normalizeBashBlocklist(settings.BashBlocklist)
	settings.InitialPrompt = normalizeInitialPrompt(settings.InitialPrompt)
	settings.RoleCard = normalizeInitialPrompt(settings.RoleCard)
	settings.WorldBook = normalizeInitialPrompt(settings.WorldBook)
	settings.OCRBaseURL = strings.TrimRight(strings.TrimSpace(settings.OCRBaseURL), "/")
	settings.OCRModel = strings.TrimSpace(settings.OCRModel)
	settings.OCRPrompt = normalizeInitialPrompt(settings.OCRPrompt)
	if settings.Model == "" {
		return fmt.Errorf("model is required")
	}
	if settings.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if settings.MaxTokens <= 0 {
		return fmt.Errorf("max tokens must be positive")
	}
	if settings.APITimeout <= 0 {
		settings.APITimeout = 120
	}
	if settings.APIMaxRetries < 0 {
		settings.APIMaxRetries = 0
	}
	if settings.APIMaxRetries > 10 {
		settings.APIMaxRetries = 10
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
	if settings.OCRTimeout <= 0 {
		settings.OCRTimeout = 60
	}
	a.cfg.Model = settings.Model
	a.cfg.Mode = settings.Mode
	a.cfg.Portable = settings.Portable
	a.cfg.MinimizeToTray = settings.MinimizeToTray
	a.cfg.MaxTokens = settings.MaxTokens
	a.cfg.Temperature = settings.Temperature
	a.cfg.API.BaseURL = settings.BaseURL
	a.cfg.API.TimeoutSeconds = settings.APITimeout
	a.cfg.API.MaxRetries = settings.APIMaxRetries
	a.cfg.API.ProxyURL = settings.APIProxyURL
	a.cfg.ThinkingEnabled = settings.ThinkingEnabled
	a.cfg.ReasoningDisplay = normalizeReasoningDisplay(settings.ReasoningDisplay)
	a.cfg.AutoCowork = settings.AutoCowork
	a.cfg.Safety.ToolMode = settings.ToolMode
	a.cfg.Safety.ToolOverrides = settings.ToolOverrides
	a.cfg.Safety.BashBlocklist = settings.BashBlocklist
	a.cfg.InitialPrompt = settings.InitialPrompt
	a.cfg.RoleCard = settings.RoleCard
	a.cfg.WorldBook = settings.WorldBook
	a.cfg.OCR.Enabled = settings.OCREnabled
	a.cfg.OCR.Provider = normalizeOCRProvider(settings.OCRProvider)
	a.cfg.OCR.BaseURL = settings.OCRBaseURL
	a.cfg.OCR.Model = settings.OCRModel
	a.cfg.OCR.Prompt = settings.OCRPrompt
	a.cfg.OCR.TimeoutSeconds = settings.OCRTimeout
	if a.cfg.OCR.MaxImageBytes <= 0 {
		a.cfg.OCR.MaxImageBytes = 8 * 1024 * 1024
	}
	a.configureToolPolicy()
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

func (a *App) SetOCRAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("OCR API key is required")
	}
	a.cfg.OCR.APIKey = key
	if err := a.cfg.Save(); err != nil {
		a.recordLog("error", fmt.Sprintf("failed to save OCR API key: %v", err))
		return fmt.Errorf("failed to save OCR API key: %w", err)
	}
	runtime.EventsEmit(a.ctx, "settings:updated", map[string]string{"ocrKeyStatus": "configured"})
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
	exported.OCR.APIKey = ""
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
	if strings.TrimSpace(imported.OCR.APIKey) == "" {
		imported.OCR.APIKey = a.cfg.OCR.APIKey
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
	a.cfg.OCR = imported.OCR
	a.cfg.Safety = imported.Safety
	a.cfg.Safety.ToolMode = normalizeSafetyToolMode(a.cfg.Safety.ToolMode)
	a.cfg.AllowedTools = append([]string(nil), imported.AllowedTools...)
	a.cfg.Memory = imported.Memory
	a.cfg.Plans = imported.Plans
	a.cfg.Web = imported.Web
	a.configureToolPolicy()
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

type APIKeyTestRequest struct {
	APIKey         string `json:"apiKey"`
	BaseURL        string `json:"baseUrl"`
	Model          string `json:"model"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
	MaxRetries     int    `json:"maxRetries"`
	ProxyURL       string `json:"proxyUrl"`
}

type APIKeyTestResult struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latencyMs"`
}

func (a *App) TestAPIKey(req APIKeyTestRequest) (*APIKeyTestResult, error) {
	key := strings.TrimSpace(req.APIKey)
	if key == "" {
		key = a.cfg.GetAPIKey()
	}
	if key == "" {
		return &APIKeyTestResult{OK: false, Message: "API key is missing"}, nil
	}
	baseURL := strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	if baseURL == "" {
		baseURL = a.cfg.API.BaseURL
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = a.cfg.Model
	}
	timeoutSeconds := req.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = a.cfg.API.TimeoutSeconds
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 120
	}
	maxRetries := req.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	if maxRetries > 3 {
		maxRetries = 3
	}
	proxyURL := strings.TrimSpace(req.ProxyURL)
	if proxyURL == "" {
		proxyURL = a.cfg.API.ProxyURL
	}

	client := api.NewClient(baseURL, model, key, maxRetries)
	client.UpdateConfig(baseURL, model, key, maxRetries, timeoutSeconds, false, proxyURL)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	start := time.Now()
	_, err := client.ChatContext(ctx, []api.Message{{Role: "user", Content: "Reply with OK."}}, nil, 8, 0)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		message := friendlyErrorMessage(err)
		a.recordLog("error", "API key test failed: "+message)
		return &APIKeyTestResult{OK: false, Message: message, LatencyMs: latency}, nil
	}
	a.recordLog("info", fmt.Sprintf("API key test succeeded in %dms", latency))
	return &APIKeyTestResult{OK: true, Message: "API key is valid", LatencyMs: latency}, nil
}

func (a *App) GetOCRAPIKeyStatus() string {
	if a.cfg.GetOCRAPIKey() != "" {
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

	ctx, cancel := context.WithCancel(tools.WithSessionID(a.ctx, req.ParentSessionID))

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
	a.configureRegistryPolicy(subReg)
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

func normalizeSafetyToolMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case string(tools.ToolModeReadOnly):
		return string(tools.ToolModeReadOnly)
	case string(tools.ToolModeAuto):
		return string(tools.ToolModeAuto)
	case string(tools.ToolModeConfirm):
		return string(tools.ToolModeConfirm)
	default:
		return string(tools.ToolModeConfirm)
	}
}

func normalizeSafetyToolOverrides(overrides map[string]string) map[string]string {
	if len(overrides) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(overrides))
	for name, mode := range overrides {
		name = strings.TrimSpace(name)
		mode = strings.TrimSpace(mode)
		if name == "" || mode == "" {
			continue
		}
		normalized[name] = normalizeSafetyToolMode(mode)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeBashBlocklist(patterns []string) []string {
	if len(patterns) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(patterns))
	normalized := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if _, ok := seen[pattern]; ok {
			continue
		}
		seen[pattern] = struct{}{}
		normalized = append(normalized, pattern)
	}
	return normalized
}

func friendlyErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	lower := strings.ToLower(message)
	if friendly := friendlyNetworkErrorMessage(lower, message); friendly != "" {
		return friendly
	}
	switch {
	case strings.Contains(lower, "reasoning_content"):
		return "DeepSeek 拒绝了请求：reasoning_content 不能作为普通历史消息传回。请重试；如果仍出现，关闭思考显示或新建会话可绕过旧历史。原始错误：" + message
	case strings.Contains(lower, "401") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "api key"):
		return "API Key 无效或未配置。请在设置里重新粘贴 DeepSeek API Key，并使用“测试 Key”验证。原始错误：" + message
	case strings.Contains(lower, "429") || strings.Contains(lower, "rate limit"):
		return "请求被限流或并发过高。稍等一会儿再试，或降低并发/输出长度。原始错误：" + message
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded"):
		return "请求超时。可以在设置的 API 网络页提高超时时间，或检查代理/网络连接。原始错误：" + message
	case strings.Contains(lower, "proxyconnect") || strings.Contains(lower, "proxy") || strings.Contains(lower, "connectex"):
		return "网络或代理连接失败。请检查设置里的代理 URL 是否可用，或先清空代理再测试。原始错误：" + message
	case strings.Contains(lower, "no such host") || strings.Contains(lower, "dns"):
		return "无法解析 API 地址。请检查 Base URL、DNS 或代理设置。原始错误：" + message
	case strings.Contains(lower, "400") || strings.Contains(lower, "invalid_request_error"):
		return "DeepSeek 返回 400 请求错误。通常是模型参数、消息格式或上下文内容不符合接口要求。原始错误：" + message
	default:
		return message
	}
}

func friendlyNetworkErrorMessage(lower, message string) string {
	switch {
	case strings.Contains(lower, "eof"),
		strings.Contains(lower, "connection reset"),
		strings.Contains(lower, "connection aborted"),
		strings.Contains(lower, "broken pipe"),
		strings.Contains(lower, "server closed idle connection"),
		strings.Contains(lower, "forcibly closed"):
		return "The DeepSeek connection was interrupted before a response completed. This is usually a transient network, proxy, or upstream issue. Retry once; if it repeats, test the API key in Settings, check proxy stability, or increase API timeout. Original error: " + message
	default:
		return ""
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

func looksBinaryContent(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	sampleSize := len(data)
	if sampleSize > 4096 {
		sampleSize = 4096
	}
	sample := data[:sampleSize]
	noise := 0
	for _, b := range sample {
		if b == 0 {
			return true
		}
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			noise++
		}
	}
	return float64(noise)/float64(len(sample)) > 0.02
}

func shouldHideFileEntry(name string) bool {
	switch name {
	case ".git", ".gocache-codex", "node_modules", "dist", "build":
		return true
	default:
		return false
	}
}

func fileSearchRank(result FileSearchResult, needle string) int {
	if needle == "" {
		return 3
	}
	name := strings.ToLower(result.Name)
	rel := strings.ToLower(result.RelativePath)
	switch {
	case name == needle:
		return 0
	case strings.HasPrefix(name, needle):
		return 1
	case strings.Contains(name, needle):
		return 2
	case strings.Contains(rel, needle):
		return 3
	default:
		return 4
	}
}
