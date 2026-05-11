# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DeepHermes is a Windows desktop AI agent client built with **Wails v2** (Go backend + WebView2 frontend). It integrates with the DeepSeek API for chat completions and provides a tool-execution environment (file I/O, bash, web search, etc.) with safety controls.

The app runs in two modes: desktop GUI (default) or CLI (`--cli` flag). Entry point is `main.go`.

## Build & Development Commands

### Go backend
```bash
go test ./...                           # Run all tests
go test ./pkg/agent/...                 # Run tests for a single package
go test -run TestName ./pkg/tools/...   # Run a single test
```

### Frontend (React/Vite/TypeScript)
```bash
cd frontend
npm install                             # Install dependencies
npm run dev                             # Vite dev server (port 5173)
npm run build                           # TypeScript check + production build
tsc --noEmit                            # Type-check only
```

### Desktop build (Windows)
```powershell
powershell -File scripts/build-windows.ps1
powershell -File scripts/build-installer.ps1 -Version 1.0.0
```
Build uses Wails tags `desktop,production` and ldflags `-w -s -H windowsgui`. Output: `build/bin/DeepHermes.exe`. Frontend must be built first — the dist is embedded via `//go:embed all:frontend/dist`.

## Architecture

### Backend (Go)

- **`app/`** — Wails-bound `App` struct. All public methods on `App` are callable from the React frontend via Wails bindings. Handles sessions, messages, tool approval/rollback, workspace browsing, settings, OCR, and diagnostics.
- **`pkg/api/`** — DeepSeek HTTP client. Streaming SSE responses, token usage tracking, reasoning content extraction.
- **`pkg/agent/`** — Agent loop: sends messages to the API, parses tool calls from responses, executes tools, feeds results back. `prompt.go` builds the system prompt from environment, mode, and available tools.
- **`pkg/tools/`** — Tool registry and implementations (ReadFile, WriteFile, EditFile, Bash, Glob, Grep, WebFetch, WebSearch). Tools have JSON schemas for parameter validation and operate in three safety modes: `read_only`, `confirm`, `auto`.
- **`pkg/config/`** — YAML config loader with environment variable overrides. Supports portable mode (config next to exe) and standard mode (`~/.deephermes/`).
- **`pkg/memory/`** — Persistent user/project memory stored in `~/.deephermes/memory/`.
- **`pkg/cowork/`** — Multi-agent coordination and shared context.
- **`pkg/subagent/`** — Sub-agent spawning (Explore, Plan, GeneralPurpose types).

### Frontend (React + TypeScript)

- **State management**: Zustand stores in `src/stores/` — `sessionStore` (chat state), `settingsStore` (preferences), `themeStore` (light/dark/anime), `layoutStore` (panels), `toolActivityStore` (tool execution tracking), `i18nStore` (i18n).
- **Key components**: `src/components/chat/` (ChatView, MessageBubble), `src/components/settings/` (SettingsDialog), `src/components/tools/` (ToolActivityPanel), `src/components/layout/` (Sidebar, TitleBar).
- **Wails bridge**: `src/lib/wails.ts` wraps Wails runtime calls. Auto-generated Go bindings live in `frontend/wailsjs/go/`.
- Styling: Tailwind CSS. Markdown rendering: react-markdown + remark-gfm + rehype-highlight.

### Data flow

```
User input → App.SendMessage() → Agent.Run() → DeepSeek API (streaming SSE)
    ↓                                              ↓
Tool calls parsed ← streaming response ← token chunks emitted to frontend via Wails events
    ↓
Tool executed (with approval if needed) → result appended to messages → loop continues
```

## Configuration

- Default config: `config.yaml` at project root
- Key env vars: `DEEPSEEK_API_KEY`, `DEEPSEEK_MODEL`, `DEEPHERMES_PROXY_URL`, `DEEPHERMES_TOOL_MODE`, `DEEPHERMES_OCR_*`
- Storage: `~/.deephermes/` (standard) or `DeepHermesData/` next to executable (portable)
- Tool safety modes: `read_only` (no writes), `confirm` (user approval), `auto` (no confirmation)

## Key Dependencies

- Go 1.26.1, Wails v2.12.0, yaml.v3
- React 18, Vite 5, TypeScript 5.5, Tailwind CSS 3, Zustand 4
