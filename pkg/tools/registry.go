package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ad201/deephermes/pkg/api"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, args map[string]any) (string, error)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Unregister(name string) {
	delete(r.tools, name)
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) List() []Tool {
	var list []Tool
	for _, t := range r.tools {
		list = append(list, t)
	}
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
	t, ok := r.tools[call.Function.Name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", call.Function.Name)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return "", fmt.Errorf("invalid arguments for %s: %w", call.Function.Name, err)
	}
	return t.Execute(ctx, args)
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
