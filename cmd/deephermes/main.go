package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/config"
	"github.com/ad201/deephermes/pkg/memory"
	"github.com/ad201/deephermes/pkg/plan"
	"github.com/ad201/deephermes/pkg/subagent"
	"github.com/ad201/deephermes/pkg/tools"
	"github.com/ad201/deephermes/web"
)

var (
	cyan   = "\033[36m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	webMode := flag.Bool("web", false, "Start web UI")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Build agent
	client := api.NewClient(cfg.API.BaseURL, cfg.Model, cfg.GetAPIKey(), cfg.API.MaxRetries)
	reg := tools.NewRegistry()
	registerTools(reg)
	workDir := agent.GetWorkDir()
	configureToolPolicy(reg, cfg, workDir)

	ag := agent.New(client, reg, agent.Config{
		WorkDir:     workDir,
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	})

	// Memory system
	memStore := memory.NewStore(cfg.Memory.Dir)
	memStore.Load()

	// Plan mode manager
	planMgr := plan.NewManager(cfg.Plans.Dir)

	if *webMode {
		server, err := web.NewServer(ag, cfg.Web.Port)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error starting web server: %v\n", err)
			os.Exit(1)
		}
		if err := server.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("%sDeepHermes%s — AI Agent Harness (DeepSeek)\n", bold+cyan, reset)
	fmt.Printf("%sModel: %s | Type /help for commands | /quit to exit%s\n", dim, cfg.Model, reset)
	fmt.Println(strings.Repeat("─", 60))

	// Handle Ctrl+C gracefully
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nGoodbye!")
		cancel()
		os.Exit(0)
	}()

	// REPL loop
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if planMgr.IsPlanMode() {
			fmt.Printf("\n%s[plan] %s>%s ", yellow, reset, green)
		} else {
			fmt.Printf("\n%s>%s ", bold+green, reset)
		}
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle slash commands
		if strings.HasPrefix(input, "/") {
			handleCommand(input, ag, cfg, memStore, planMgr, client)
			continue
		}

		// Run agent (with plan mode context if active)
		fmt.Println()
		augmentedInput := input
		if planMgr.IsPlanMode() {
			augmentedInput = planMgr.SystemPromptModifier() + "\n\nUser: " + input
		}

		resp, err := ag.Run(ctx, augmentedInput)
		if err != nil {
			fmt.Printf("%sError: %v%s\n", red, err, reset)
			continue
		}
		fmt.Println(renderMarkdown(resp))
	}

	fmt.Println("\nGoodbye!")
}

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

func configureToolPolicy(reg *tools.Registry, cfg *config.Config, workDir string) {
	if reg == nil || cfg == nil {
		return
	}
	tools.SetWorkingDir(workDir)
	reg.SetPolicy(tools.Policy{
		Mode:          string(tools.ToolModeAuto),
		BashBlocklist: cfg.Safety.BashBlocklist,
		AllowedDir:    workDir,
	})
}

func handleCommand(input string, ag *agent.Agent, cfg *config.Config, memStore *memory.Store, planMgr *plan.Manager, client *api.Client) {
	parts := strings.Fields(input)
	cmd := parts[0]

	switch cmd {
	case "/help":
		fmt.Println()
		fmt.Printf("%sCommands:%s\n", bold, reset)
		fmt.Println("  /help       Show this help")
		fmt.Println("  /clear      Clear conversation history")
		fmt.Println("  /tools      List available tools")
		fmt.Println("  /history    Show conversation history")
		fmt.Println("  /config     Show current configuration")
		fmt.Println("  /model <n>  Switch model")
		fmt.Println("  /plan       Enter plan mode")
		fmt.Println("  /plan-exit  Exit plan mode")
		fmt.Println("  /memory     Manage memories (list|add|del)")
		fmt.Println("  /explore    Launch explore sub-agent")
		fmt.Println("  /web        Start web UI")
		fmt.Println("  /quit       Exit DeepHermes")

	case "/clear":
		ag.Reset()
		fmt.Printf("%sConversation cleared.%s\n", yellow, reset)

	case "/tools":
		fmt.Println()
		fmt.Printf("%sAvailable tools:%s\n", bold, reset)
		for _, name := range ag.Registry().Names() {
			if t, ok := ag.Registry().Get(name); ok {
				fmt.Printf("  %s%s%s — %s\n", green, name, reset, t.Description())
			}
		}

	case "/history":
		msgs := ag.Messages()
		if len(msgs) == 0 {
			fmt.Printf("%sNo conversation history.%s\n", dim, reset)
		} else {
			fmt.Println()
			for _, m := range msgs {
				role := m.Role
				content := m.Content
				if len(content) > 120 {
					content = content[:120] + "..."
				}
				fmt.Printf("  %s[%s]%s %s\n", dim, role, reset, content)
			}
		}

	case "/config":
		fmt.Println()
		fmt.Printf("Model:       %s\n", cfg.Model)
		fmt.Printf("Max Tokens:  %d\n", cfg.MaxTokens)
		fmt.Printf("Temperature: %.2f\n", cfg.Temperature)
		fmt.Printf("Base URL:    %s\n", cfg.API.BaseURL)
		fmt.Printf("Memory Dir:  %s\n", cfg.Memory.Dir)
		fmt.Printf("Plans Dir:   %s\n", cfg.Plans.Dir)

	case "/model":
		if len(parts) > 1 {
			cfg.Model = parts[1]
			fmt.Printf("%sModel changed to: %s (restart required for full effect)%s\n", yellow, parts[1], reset)
		} else {
			fmt.Printf("Current model: %s\n", cfg.Model)
			fmt.Println("Usage: /model <model-name>")
		}

	case "/plan":
		if planMgr.IsPlanMode() {
			fmt.Printf("%sAlready in plan mode. Plan: %s%s\n", yellow, planMgr.PlanPath(), reset)
			return
		}
		if err := planMgr.EnterPlanMode(); err != nil {
			fmt.Printf("%sError entering plan mode: %v%s\n", red, err, reset)
			return
		}
		fmt.Printf("%sPlan mode activated. Plan file: %s%s\n", yellow, planMgr.PlanPath(), reset)
		fmt.Println("You are now in read-only mode. Design your approach, then /plan-exit to implement.")

	case "/plan-exit":
		if !planMgr.IsPlanMode() {
			fmt.Printf("%sNot in plan mode.%s\n", dim, reset)
			return
		}
		planMgr.ExitPlanMode()
		fmt.Printf("%sPlan mode exited. Normal mode restored.%s\n", green, reset)

	case "/memory":
		handleMemoryCommand(parts, memStore)

	case "/explore":
		if len(parts) < 2 {
			fmt.Println("Usage: /explore <task description>")
			return
		}
		task := strings.Join(parts[1:], " ")
		fmt.Printf("%sLaunching explore agent for: %s%s\n", dim, task, reset)
		sa := subagent.Spawn(subagent.Explore, client, agent.Config{
			WorkDir:     agent.GetWorkDir(),
			Model:       cfg.Model,
			MaxTokens:   cfg.MaxTokens,
			Temperature: cfg.Temperature,
		}, subagent.BuildSubAgentPrompt(subagent.Explore, task))

		// Wait for result with a reasonable timeout
		ctx := context.Background()
		result := sa.Wait(ctx)
		if result.Error != nil {
			fmt.Printf("%sExplore error: %v%s\n", red, result.Error, reset)
		} else {
			fmt.Printf("\n%s--- Explore Result ---%s\n%s\n%s-----------------------%s\n",
				cyan, reset, result.Output, cyan, reset)
		}

	case "/web":
		fmt.Printf("%sStarting web UI on http://localhost:%d ...%s\n", green, cfg.Web.Port, reset)
		server, err := web.NewServer(ag, cfg.Web.Port)
		if err != nil {
			fmt.Printf("%sError: %v%s\n", red, err, reset)
			return
		}
		go server.Start()
		fmt.Println("Web UI running. Press Ctrl+C to stop the web server.")

	case "/quit":
		fmt.Println("Goodbye!")
		os.Exit(0)

	default:
		fmt.Printf("%sUnknown command: %s%s\n", red, cmd, reset)
		fmt.Println("Type /help for available commands.")
	}
}

func handleMemoryCommand(parts []string, memStore *memory.Store) {
	if len(parts) < 2 {
		fmt.Println("Usage: /memory list|add|del [args...]")
		return
	}

	switch parts[1] {
	case "list":
		entries := memStore.List()
		if len(entries) == 0 {
			fmt.Printf("%sNo memories stored.%s\n", dim, reset)
			return
		}
		fmt.Println()
		for _, e := range entries {
			typeColors := map[memory.MemoryType]string{
				memory.User:      cyan,
				memory.Feedback:  yellow,
				memory.Project:   green,
				memory.Reference: dim,
			}
			c := typeColors[e.Type]
			if c == "" {
				c = reset
			}
			fmt.Printf("  %s[%s]%s %s — %s\n", c, e.Type, reset, e.Name, e.Description)
		}

	case "add":
		if len(parts) < 5 {
			fmt.Println("Usage: /memory add <type> <name> <description>")
			fmt.Println("Types: user, feedback, project, reference")
			return
		}
		entry := &memory.Entry{
			Type:        memory.MemoryType(parts[2]),
			Name:        parts[3],
			Description: strings.Join(parts[4:], " "),
			Content:     strings.Join(parts[4:], " "),
		}
		if err := memStore.Save(entry); err != nil {
			fmt.Printf("%sError saving memory: %v%s\n", red, err, reset)
			return
		}
		fmt.Printf("%sMemory saved: %s%s\n", green, entry.Name, reset)

	case "del":
		if len(parts) < 3 {
			fmt.Println("Usage: /memory del <name>")
			return
		}
		if err := memStore.Delete(parts[2]); err != nil {
			fmt.Printf("%sError: %v%s\n", red, err, reset)
			return
		}
		fmt.Printf("%sMemory deleted: %s%s\n", yellow, parts[2], reset)

	default:
		fmt.Println("Usage: /memory list|add|del [args...]")
	}
}

func renderMarkdown(text string) string {
	var result strings.Builder
	lines := strings.Split(text, "\n")
	inCode := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCode = !inCode
			result.WriteString(dim)
			result.WriteString(line)
			result.WriteString(reset)
			result.WriteByte('\n')
			continue
		}
		if inCode {
			result.WriteString(dim)
			result.WriteString("  ")
			result.WriteString(line)
			result.WriteString(reset)
		} else {
			result.WriteString(line)
		}
		result.WriteByte('\n')
	}
	return result.String()
}
