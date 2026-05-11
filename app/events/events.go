package events

import (
	"encoding/json"
	"time"
)

type EventType string

const (
	EventStreamDelta   EventType = "stream:delta"
	EventStreamDone    EventType = "stream:done"
	EventToolCall      EventType = "tool:call"
	EventToolApproval  EventType = "tool:approval"
	EventToolResult    EventType = "tool:result"
	EventAgentStatus   EventType = "agent:status"
	EventError         EventType = "error"
	EventCoworkUpdate  EventType = "cowork:update"
	EventSessionUpdate    EventType = "session:update"
	EventContextCompacted EventType = "context:compacted"
)

type AppEvent struct {
	Type      EventType       `json:"type"`
	SessionID string          `json:"sessionId"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

func NewEvent(t EventType, sessionID string, data any) AppEvent {
	raw, _ := json.Marshal(data)
	return AppEvent{
		Type:      t,
		SessionID: sessionID,
		Data:      raw,
		Timestamp: time.Now(),
	}
}

// Event payloads

type StreamDeltaPayload struct {
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoningContent,omitempty"`
}

type StreamDonePayload struct {
	FullResponse string `json:"fullResponse"`
	TokensUsed   int    `json:"tokensUsed"`
	Usage        any    `json:"usage,omitempty"`
	Metrics      any    `json:"metrics,omitempty"`
	FinishReason string `json:"finishReason,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
}

type ToolCallPayload struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Risk      string `json:"risk,omitempty"`
}

type ToolApprovalPayload struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	ToolName  string `json:"toolName"`
	Arguments string `json:"arguments"`
	Risk      string `json:"risk"`
	Mode      string `json:"mode"`
	Preview   string `json:"preview,omitempty"`
}

type ToolResultPayload struct {
	ToolCallID        string `json:"toolCallId"`
	Name              string `json:"name"`
	Content           string `json:"content"`
	Success           bool   `json:"success"`
	Error             string `json:"error,omitempty"`
	Risk              string `json:"risk,omitempty"`
	RollbackAvailable bool   `json:"rollbackAvailable,omitempty"`
	RollbackPath      string `json:"rollbackPath,omitempty"`
}

type AgentStatusPayload struct {
	Status string `json:"status"` // "idle", "thinking", "executing", "streaming"
	Model  string `json:"model"`
}

type ErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

type CoworkUpdatePayload struct {
	SubAgentID string `json:"subAgentId"`
	Name       string `json:"name"`
	Status     string `json:"status"` // "pending", "running", "done", "failed"
	Type       string `json:"type"`   // "explore", "implement", "review"
	Progress   string `json:"progress"`
	Result     string `json:"result,omitempty"`
}

type SessionUpdatePayload struct {
	SessionID string `json:"sessionId"`
	Name      string `json:"name"`
	Action    string `json:"action"` // "created", "deleted", "renamed"
}

type ContextCompactedPayload struct {
	MessagesBefore int `json:"messagesBefore"`
	MessagesAfter  int `json:"messagesAfter"`
	SummaryTokens  int `json:"summaryTokens"`
}
