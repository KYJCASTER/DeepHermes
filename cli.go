package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ad201/deephermes/pkg/agent"
	"github.com/ad201/deephermes/pkg/api"
	"github.com/ad201/deephermes/pkg/config"
	"github.com/ad201/deephermes/pkg/tools"
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

func runCLI(cfg *config.Config) {
	client := api.NewClient(cfg.API.BaseURL, cfg.Model, cfg.GetAPIKey(), cfg.API.MaxRetries)
	reg := tools.NewRegistry()
	reg.Register(&tools.ReadFile{})
	reg.Register(&tools.WriteFile{})
	reg.Register(&tools.EditFile{})
	reg.Register(&tools.Bash{})
	reg.Register(&tools.Glob{})
	reg.Register(&tools.Grep{})
	reg.Register(&tools.WebFetch{})
	reg.Register(&tools.WebSearch{})
	workDir := agent.GetWorkDir()
	tools.SetWorkingDir(workDir)
	reg.SetPolicy(tools.Policy{
		Mode:          string(tools.ToolModeAuto),
		BashBlocklist: cfg.Safety.BashBlocklist,
		AllowedDir:    workDir,
	})

	ag := agent.New(client, reg, agent.Config{
		WorkDir:     workDir,
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	})

	fmt.Printf("%sDeepHermes%s — AI Agent CLI\n", bold+cyan, reset)
	fmt.Printf("%sModel: %s | /help | /quit%s\n", dim, cfg.Model, reset)
	fmt.Println(strings.Repeat("─", 60))

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

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("\n%s>%s ", bold+green, reset)
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if strings.HasPrefix(input, "/quit") {
			fmt.Println("Goodbye!")
			return
		}
		if strings.HasPrefix(input, "/help") {
			fmt.Println("\nCommands: /help /clear /quit")
			continue
		}
		if strings.HasPrefix(input, "/clear") {
			ag.Reset()
			fmt.Printf("%sConversation cleared.%s\n", yellow, reset)
			continue
		}

		fmt.Println()
		resp, err := ag.Run(ctx, input)
		if err != nil {
			fmt.Printf("%sError: %v%s\n", red, err, reset)
			continue
		}
		fmt.Println(resp)
	}
}
