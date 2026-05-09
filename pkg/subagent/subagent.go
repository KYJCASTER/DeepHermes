package subagent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/tools"
)

type AgentType string

const (
	Explore       AgentType = "explore"
	Plan          AgentType = "plan"
	GeneralPurpose AgentType = "general-purpose"
)

type SubAgent struct {
	Type     AgentType
	agent    *agent.Agent
	registry *tools.Registry
	client   *api.Client
	cfg      agent.Config
	mu       sync.Mutex
	done     chan Result
}

type Result struct {
	Output  string
	Error   error
	AgentID string
}

func Spawn(agentType AgentType, client *api.Client, cfg agent.Config, prompt string) *SubAgent {
	reg := buildRegistry(agentType)
	ag := agent.New(client, reg, cfg)

	sa := &SubAgent{
		Type:     agentType,
		agent:    ag,
		registry: reg,
		client:   client,
		cfg:      cfg,
		done:     make(chan Result, 1),
	}

	go sa.run(prompt)
	return sa
}

func (sa *SubAgent) Wait(ctx context.Context) Result {
	select {
	case r := <-sa.done:
		return r
	case <-ctx.Done():
		return Result{Error: ctx.Err()}
	}
}

func (sa *SubAgent) Done() <-chan Result {
	return sa.done
}

func (sa *SubAgent) run(prompt string) {
	ctx := context.Background()
	output, err := sa.agent.Run(ctx, prompt)
	sa.done <- Result{Output: output, Error: err}
}

func buildRegistry(agentType AgentType) *tools.Registry {
	reg := tools.NewRegistry()
	switch agentType {
	case Explore:
		reg.Register(&tools.ReadFile{})
		reg.Register(&tools.Glob{})
		reg.Register(&tools.Grep{})
		reg.Register(&tools.WebFetch{})
		reg.Register(&tools.WebSearch{})
	case Plan:
		reg.Register(&tools.ReadFile{})
		reg.Register(&tools.Glob{})
		reg.Register(&tools.Grep{})
		reg.Register(&tools.WebFetch{})
	case GeneralPurpose:
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

func BuildSubAgentPrompt(agentType AgentType, task string) string {
	var sb strings.Builder
	switch agentType {
	case Explore:
		sb.WriteString("You are a code exploration agent. Your job is to search and read code to answer questions.\n")
		sb.WriteString("Do NOT modify files. Only read, search, and report findings.\n")
		sb.WriteString("Be thorough but concise in your report.\n")
	case Plan:
		sb.WriteString("You are a software architecture planning agent. Your job is to design solutions.\n")
		sb.WriteString("Read relevant code, think through trade-offs, and produce a concrete plan.\n")
		sb.WriteString("Do NOT modify files. Output your plan as markdown.\n")
	case GeneralPurpose:
		sb.WriteString("You are a general-purpose AI agent. Complete the task using available tools.\n")
		sb.WriteString("Be thorough and efficient.\n")
	}
	sb.WriteString("\n---\n\n")
	sb.WriteString(fmt.Sprintf("Task: %s\n", task))
	return sb.String()
}
