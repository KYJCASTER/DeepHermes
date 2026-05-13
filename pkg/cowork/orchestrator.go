package cowork

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/tools"
)

type TaskStatus string

const (
	TaskPending TaskStatus = "pending"
	TaskRunning TaskStatus = "running"
	TaskDone    TaskStatus = "done"
	TaskFailed  TaskStatus = "failed"
)

type Subtask struct {
	ID           string     `json:"id"`
	Description  string     `json:"description"`
	AgentType    string     `json:"agentType"`
	AssignedTo   string     `json:"assignedTo"`
	Status       TaskStatus `json:"status"`
	Result       string     `json:"result"`
	Dependencies []string   `json:"dependencies"`
}

type TaskPlan struct {
	MainTask     string              `json:"mainTask"`
	Subtasks     []Subtask           `json:"subtasks"`
	Dependencies map[string][]string `json:"dependencies"`
}

type Orchestrator struct {
	client  *api.Client
	context *SharedContext
	tasks   map[string]*TaskPlan
	mu      sync.RWMutex
}

func NewOrchestrator(client *api.Client, sharedCtx *SharedContext) *Orchestrator {
	return &Orchestrator{
		client:  client,
		context: sharedCtx,
		tasks:   make(map[string]*TaskPlan),
	}
}

// Decompose uses the LLM to analyze a task and split it into subtasks.
func (o *Orchestrator) Decompose(ctx context.Context, task string) (*TaskPlan, error) {
	prompt := fmt.Sprintf(`Analyze the following task and break it down into independent subtasks.

Task: %s

Output a JSON object with:
- "subtasks": array of { "description": "...", "agentType": "explore|implement|review" }
- "dependencies": map of subtask_index -> [dependent_indices]

Only include truly independent subtasks. Maximum 5 subtasks.`, task)

	// Use a simple API call to decompose
	messages := []api.Message{
		{Role: "system", Content: "You are a task decomposition assistant. Output ONLY valid JSON."},
		{Role: "user", Content: prompt},
	}

	resp, err := o.client.ChatContext(ctx, messages, nil, 2048, 0.3)
	if err != nil {
		return nil, fmt.Errorf("decompose error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response")
	}

	// Parse JSON from response
	content := resp.Choices[0].Message.Content
	content = extractJSON(content)

	var plan TaskPlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		// If JSON parse fails, create single-task plan
		plan = TaskPlan{
			MainTask: task,
			Subtasks: []Subtask{{
				ID:          "task-0",
				Description: task,
				AgentType:   "general-purpose",
				Status:      TaskPending,
			}},
		}
	}

	plan.MainTask = task
	for i := range plan.Subtasks {
		plan.Subtasks[i].ID = fmt.Sprintf("task-%d", i)
		plan.Subtasks[i].Status = TaskPending
	}

	o.mu.Lock()
	o.tasks[task] = &plan
	o.mu.Unlock()

	return &plan, nil
}

// DispatchResult holds the result of a dispatched subtask.
type DispatchResult struct {
	SubtaskID string
	Output    string
	Error     error
}

// Dispatch runs a subtask using a sub-agent.
func (o *Orchestrator) Dispatch(ctx context.Context, plan *TaskPlan, subtask Subtask, cfg agent.Config) DispatchResult {
	reg := buildRegistry(subtask.AgentType)
	configureToolPolicy(reg, cfg.WorkDir)
	ag := agent.New(o.client, reg, cfg)

	prompt := buildPrompt(subtask.AgentType, subtask.Description)

	output, err := ag.Run(ctx, prompt)

	o.mu.Lock()
	for i := range plan.Subtasks {
		if plan.Subtasks[i].ID == subtask.ID {
			if err != nil {
				plan.Subtasks[i].Status = TaskFailed
				plan.Subtasks[i].Result = err.Error()
			} else {
				plan.Subtasks[i].Status = TaskDone
				plan.Subtasks[i].Result = output
			}
			break
		}
	}
	o.mu.Unlock()

	if err != nil {
		return DispatchResult{SubtaskID: subtask.ID, Error: err}
	}
	return DispatchResult{SubtaskID: subtask.ID, Output: output}
}

// RunAll decomposes and dispatches all subtasks.
func (o *Orchestrator) RunAll(ctx context.Context, task string, cfg agent.Config) (*TaskPlan, []DispatchResult, error) {
	plan, err := o.Decompose(ctx, task)
	if err != nil {
		return nil, nil, err
	}

	var wg sync.WaitGroup
	results := make([]DispatchResult, len(plan.Subtasks))

	for i, st := range plan.Subtasks {
		wg.Add(1)
		go func(idx int, subtask Subtask) {
			defer wg.Done()

			o.mu.Lock()
			plan.Subtasks[idx].Status = TaskRunning
			o.mu.Unlock()

			results[idx] = o.Dispatch(ctx, plan, subtask, cfg)
		}(i, st)
	}

	wg.Wait()
	return plan, results, nil
}

func buildRegistry(agentType string) *tools.Registry {
	reg := tools.NewRegistry()
	switch agentType {
	case "explore", "review":
		reg.Register(&tools.ReadFile{})
		reg.Register(&tools.Glob{})
		reg.Register(&tools.Grep{})
		reg.Register(&tools.WebFetch{})
		reg.Register(&tools.WebSearch{})
	case "implement", "general-purpose":
		reg.Register(&tools.ReadFile{})
		reg.Register(&tools.WriteFile{})
		reg.Register(&tools.EditFile{})
		reg.Register(&tools.Bash{})
		reg.Register(&tools.Glob{})
		reg.Register(&tools.Grep{})
		reg.Register(&tools.WebFetch{})
		reg.Register(&tools.WebSearch{})
	}
	return reg
}

func configureToolPolicy(reg *tools.Registry, workDir string) {
	if reg == nil || strings.TrimSpace(workDir) == "" {
		return
	}
	tools.SetWorkingDir(workDir)
	reg.SetPolicy(tools.Policy{
		Mode:       string(tools.ToolModeAuto),
		AllowedDir: workDir,
	})
}

func buildPrompt(agentType, task string) string {
	var sb strings.Builder
	switch agentType {
	case "explore":
		sb.WriteString("You are a code exploration agent. Search, read, and report findings. Do NOT edit files.\n\n")
	case "implement":
		sb.WriteString("You are a code implementation agent. Write and edit code to complete the task.\n\n")
	case "review":
		sb.WriteString("You are a code review agent. Review for bugs, security, quality. Do NOT edit files.\n\n")
	default:
		sb.WriteString("You are a general-purpose AI agent.\n\n")
	}
	sb.WriteString("Task: " + task + "\n\n")
	sb.WriteString("Be thorough but concise. Report back with clear results.")
	return sb.String()
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "```json"); idx >= 0 {
		s = s[idx+7:]
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
	} else if idx := strings.Index(s, "```"); idx >= 0 {
		s = s[idx+3:]
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
	}
	s = strings.TrimSpace(s)
	return s
}

// unused import suppression
var _ = time.Now
