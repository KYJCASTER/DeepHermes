package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ad201/deephermes/pkg/api"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, args map[string]any) (string, error)
}

type ToolMode string

const (
	ToolModeReadOnly ToolMode = "read_only"
	ToolModeConfirm  ToolMode = "confirm"
	ToolModeAuto     ToolMode = "auto"
)

type ToolRisk string

const (
	ToolRiskRead    ToolRisk = "read"
	ToolRiskNetwork ToolRisk = "network"
	ToolRiskWrite   ToolRisk = "write"
	ToolRiskShell   ToolRisk = "shell"
	ToolRiskUnknown ToolRisk = "unknown"
)

type ApprovalRequest struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	ToolName  string `json:"toolName"`
	Arguments string `json:"arguments"`
	Risk      string `json:"risk"`
	Mode      string `json:"mode"`
	Preview   string `json:"preview,omitempty"`
}

type ApprovalDecision struct {
	Approved bool
	Reason   string
}

type ApprovalFunc func(context.Context, ApprovalRequest) (ApprovalDecision, error)

type ExecutionEvent struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
	ToolName  string `json:"toolName"`
	Arguments string `json:"arguments"`
	Risk      string `json:"risk"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
	Success   bool   `json:"success"`
}

type EventFunc func(context.Context, ExecutionEvent)

type Policy struct {
	Mode          string
	ToolOverrides map[string]string
	BashBlocklist []string
	AllowedDir    string
	Approval      ApprovalFunc
	OnCall        EventFunc
	OnResult      EventFunc
}

type contextKey string

const sessionIDContextKey contextKey = "deephermes_session_id"
const allowedDirContextKey contextKey = "deephermes_allowed_dir"

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	if strings.TrimSpace(sessionID) == "" {
		return ctx
	}
	return context.WithValue(ctx, sessionIDContextKey, sessionID)
}

func SessionIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(sessionIDContextKey).(string)
	return v
}

func WithAllowedDir(ctx context.Context, allowedDir string) context.Context {
	allowedDir = strings.TrimSpace(allowedDir)
	if allowedDir == "" {
		return ctx
	}
	return context.WithValue(ctx, allowedDirContextKey, allowedDir)
}

func AllowedDirFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(allowedDirContextKey).(string)
	return v
}

type Registry struct {
	mu     sync.RWMutex
	tools  map[string]Tool
	policy Policy
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
		policy: Policy{
			Mode: string(ToolModeAuto),
		},
	}
}

func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) SetPolicy(policy Policy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	policy.Mode = string(normalizeToolMode(policy.Mode))
	r.policy = policy
}

func (r *Registry) AllowedDir() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.policy.AllowedDir
}

func (r *Registry) List() []Tool {
	var list []Tool
	r.mu.RLock()
	for _, t := range r.tools {
		list = append(list, t)
	}
	r.mu.RUnlock()
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name() < list[j].Name()
	})
	return list
}

func (r *Registry) Names() []string {
	var names []string
	for _, t := range r.List() {
		names = append(names, t.Name())
	}
	return names
}

func (r *Registry) ToAPITools() []api.ToolDef {
	var defs []api.ToolDef
	for _, t := range r.List() {
		defs = append(defs, api.ToolDef{
			Type: "function",
			Function: api.FunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return defs
}

func (r *Registry) Execute(ctx context.Context, call api.ToolCall) (string, error) {
	r.mu.RLock()
	t, ok := r.tools[call.Function.Name]
	policy := r.policy
	r.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", call.Function.Name)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments for %s: %w", call.Function.Name, err)
	}
	event := ExecutionEvent{
		ID:        toolCallID(call),
		SessionID: SessionIDFromContext(ctx),
		ToolName:  call.Function.Name,
		Arguments: call.Function.Arguments,
		Risk:      string(classifyToolRisk(call.Function.Name)),
	}
	if err := validateToolWorkspace(call.Function.Name, args, policy.AllowedDir); err != nil {
		if policy.OnResult != nil {
			resultEvent := event
			resultEvent.Error = err.Error()
			resultEvent.Success = false
			policy.OnResult(ctx, resultEvent)
		}
		return "", err
	}
	if policy.OnCall != nil {
		policy.OnCall(ctx, event)
	}
	if err := authorizeTool(ctx, call, args, policy); err != nil {
		if policy.OnResult != nil {
			resultEvent := event
			resultEvent.Error = err.Error()
			resultEvent.Success = false
			policy.OnResult(ctx, resultEvent)
		}
		return "", err
	}
	ctx = WithAllowedDir(ctx, policy.AllowedDir)
	output, err := t.Execute(ctx, args)
	if policy.OnResult != nil {
		resultEvent := event
		resultEvent.Content = output
		resultEvent.Success = err == nil
		if err != nil {
			resultEvent.Error = err.Error()
		}
		policy.OnResult(ctx, resultEvent)
	}
	return output, err
}

func authorizeTool(ctx context.Context, call api.ToolCall, args map[string]any, policy Policy) error {
	if call.Function.Name == "bash" && len(policy.BashBlocklist) > 0 {
		cmd, _ := args["command"].(string)
		for _, pattern := range policy.BashBlocklist {
			if strings.Contains(cmd, pattern) {
				return fmt.Errorf("tool bash blocked: command contains blocked pattern %q", pattern)
			}
		}
	}

	mode := normalizeToolMode(policy.Mode)
	if override, ok := policy.ToolOverrides[call.Function.Name]; ok {
		mode = normalizeToolMode(override)
	}

	risk := classifyToolRisk(call.Function.Name)
	switch mode {
	case ToolModeAuto:
		return nil
	case ToolModeReadOnly:
		if risk == ToolRiskRead {
			return nil
		}
		return fmt.Errorf("tool %s blocked by read-only mode (%s)", call.Function.Name, risk)
	case ToolModeConfirm:
		if risk == ToolRiskRead {
			return nil
		}
		if policy.Approval == nil {
			return fmt.Errorf("tool %s requires approval but no approval handler is configured", call.Function.Name)
		}
		req := ApprovalRequest{
			ID:        fmt.Sprintf("%s-%d", call.ID, time.Now().UnixNano()),
			SessionID: SessionIDFromContext(ctx),
			ToolName:  call.Function.Name,
			Arguments: call.Function.Arguments,
			Risk:      string(risk),
			Mode:      string(mode),
			Preview:   previewToolChange(call.Function.Name, args),
		}
		decision, err := policy.Approval(ctx, req)
		if err != nil {
			return err
		}
		if !decision.Approved {
			reason := strings.TrimSpace(decision.Reason)
			if reason == "" {
				reason = "rejected by user"
			}
			return fmt.Errorf("tool %s rejected: %s", call.Function.Name, reason)
		}
		return nil
	default:
		return nil
	}
}

func validateToolWorkspace(name string, args map[string]any, allowedDir string) error {
	if strings.TrimSpace(allowedDir) == "" {
		return nil
	}
	var target string
	switch name {
	case "read_file", "write_file", "edit_file":
		target, _ = args["file_path"].(string)
	case "glob":
		target, _ = args["path"].(string)
		if strings.TrimSpace(target) == "" {
			target = "."
		}
		pattern, _ := args["pattern"].(string)
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			if filepath.IsAbs(pattern) {
				target = pattern
			} else {
				target = filepath.Join(target, pattern)
			}
		}
	case "grep":
		target, _ = args["path"].(string)
		if strings.TrimSpace(target) == "" {
			target = "."
		}
	default:
		return nil
	}
	if strings.TrimSpace(target) == "" {
		return nil
	}
	if err := ValidatePath(allowedDir, target); err != nil {
		return fmt.Errorf("tool %s blocked: %w", name, err)
	}
	return nil
}

func toolCallID(call api.ToolCall) string {
	if strings.TrimSpace(call.ID) != "" {
		return call.ID
	}
	return fmt.Sprintf("tool-%d", time.Now().UnixNano())
}

func normalizeToolMode(mode string) ToolMode {
	switch ToolMode(strings.TrimSpace(mode)) {
	case ToolModeReadOnly:
		return ToolModeReadOnly
	case ToolModeConfirm:
		return ToolModeConfirm
	case ToolModeAuto:
		return ToolModeAuto
	default:
		return ToolModeConfirm
	}
}

func classifyToolRisk(name string) ToolRisk {
	switch name {
	case "read_file", "glob", "grep":
		return ToolRiskRead
	case "web_fetch", "web_search":
		return ToolRiskNetwork
	case "write_file", "edit_file":
		return ToolRiskWrite
	case "bash":
		return ToolRiskShell
	default:
		return ToolRiskUnknown
	}
}

func previewToolChange(name string, args map[string]any) string {
	switch name {
	case "write_file":
		path, _ := args["file_path"].(string)
		next, _ := args["content"].(string)
		if strings.TrimSpace(path) == "" {
			return ""
		}
		current, err := os.ReadFile(path)
		if err != nil {
			return limitPreview(fmt.Sprintf("New file: %s\n\n+++ proposed\n%s", path, next))
		}
		return unifiedPreview(path, string(current), next)
	case "edit_file":
		path, _ := args["file_path"].(string)
		oldStr, _ := args["old_string"].(string)
		newStr, _ := args["new_string"].(string)
		replaceAll, _ := args["replace_all"].(bool)
		if strings.TrimSpace(path) == "" || oldStr == "" || oldStr == newStr {
			return ""
		}
		current, err := os.ReadFile(path)
		if err != nil {
			return "Diff preview unavailable: " + err.Error()
		}
		before := string(current)
		count := strings.Count(before, oldStr)
		if count == 0 {
			return "Diff preview unavailable: old_string was not found in the current file."
		}
		after := before
		if replaceAll {
			after = strings.ReplaceAll(before, oldStr, newStr)
		} else {
			after = strings.Replace(before, oldStr, newStr, 1)
		}
		return unifiedPreview(path, before, after)
	default:
		return ""
	}
}

func unifiedPreview(path, before, after string) string {
	if before == after {
		return ""
	}
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")
	first := firstChangedLine(beforeLines, afterLines)
	if first < 0 {
		return ""
	}
	start := first - 4
	if start < 0 {
		start = 0
	}
	endBefore := first + 16
	if endBefore > len(beforeLines) {
		endBefore = len(beforeLines)
	}
	endAfter := first + 16
	if endAfter > len(afterLines) {
		endAfter = len(afterLines)
	}
	var out strings.Builder
	fmt.Fprintf(&out, "--- %s (current)\n+++ %s (proposed)\n@@ line %d @@\n", path, path, start+1)
	for i := start; i < endBefore; i++ {
		fmt.Fprintf(&out, "- %s\n", beforeLines[i])
	}
	for i := start; i < endAfter; i++ {
		fmt.Fprintf(&out, "+ %s\n", afterLines[i])
	}
	return limitPreview(out.String())
}

func firstChangedLine(a, b []string) int {
	max := len(a)
	if len(b) > max {
		max = len(b)
	}
	for i := 0; i < max; i++ {
		var av, bv string
		if i < len(a) {
			av = a[i]
		}
		if i < len(b) {
			bv = b[i]
		}
		if av != bv {
			return i
		}
	}
	return -1
}

func limitPreview(value string) string {
	const maxPreviewBytes = 16000
	if len(value) <= maxPreviewBytes {
		return value
	}
	return value[:maxPreviewBytes] + "\n... diff preview truncated ..."
}

func ValidatePath(allowedDir, target string) error {
	allowedDir = strings.TrimSpace(allowedDir)
	if allowedDir == "" {
		return nil
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	absAllowed, err := resolveBoundaryPath(allowedDir)
	if err != nil {
		return fmt.Errorf("cannot resolve workspace directory: %w", err)
	}
	absTarget, err := resolveBoundaryPath(target)
	if err != nil {
		return fmt.Errorf("cannot resolve target path: %w", err)
	}
	if pathWithin(absAllowed, absTarget) {
		return nil
	}
	return fmt.Errorf("path %s is outside the allowed workspace %s", absTarget, absAllowed)
}

func resolveBoundaryPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved), nil
	}

	var missing []string
	current := abs
	for {
		parent := filepath.Dir(current)
		if parent == current {
			return abs, nil
		}
		missing = append(missing, filepath.Base(current))
		current = parent
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			resolved = filepath.Clean(resolved)
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		}
	}
}

func pathWithin(absAllowed, absTarget string) bool {
	absAllowed = filepath.Clean(absAllowed)
	absTarget = filepath.Clean(absTarget)
	if samePath(absAllowed, absTarget) {
		return true
	}
	prefix := absAllowed + string(filepath.Separator)
	if runtime.GOOS == "windows" {
		return strings.HasPrefix(strings.ToLower(absTarget), strings.ToLower(prefix))
	}
	return strings.HasPrefix(absTarget, prefix)
}

func samePath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
}

func (r *Registry) ExecuteAll(ctx context.Context, calls []api.ToolCall) []ToolResult {
	var results []ToolResult
	for _, call := range calls {
		output, err := r.Execute(ctx, call)
		if err != nil {
			output = fmt.Sprintf("Error: %v", err)
		}
		results = append(results, ToolResult{
			ToolCallID: call.ID,
			Name:       call.Function.Name,
			Content:    output,
		})
	}
	return results
}
