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
npm test                                # Run Vitest tests
npx tsc --noEmit                        # Type-check only
```

### Desktop build (Windows)
```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-installer.ps1 -Version 1.0.0
```

**Never use plain `go build`** for desktop artifacts. Wails requires build tags `desktop,production` and ldflags `-w -s -H windowsgui`. The build script handles this. The frontend dist is embedded via `//go:embed all:frontend/dist` in `main.go`, so `cd frontend && npm run build` must succeed before building the Go binary.

## Architecture

### Backend (Go)

- **`main.go`** — Entry point. Parses `--cli` flag. In desktop mode, creates `app.NewApp(cfg)`, passes it to `wails.Run()` with `Bind: []interface{}{desktopApp}` to expose all public `App` methods to the frontend. Uses `SingleInstanceLock` for single-instance behavior.
- **`cli.go`** — CLI mode: REPL loop with the same agent, registry, and tools. No Wails dependency.
- **`app/app.go`** — The `App` struct is the Wails binding surface. Every public method on `App` is callable from the React frontend. Manages sessions (map with `sync.RWMutex`), tool approval channels, tool rollback snapshots, sub-agents, and diagnostics logging.
- **`app/events/events.go`** — Typed event constants (`stream:delta`, `tool:call`, `tool:approval`, `tool:result`, `agent:status`, `cowork:update`, etc.) and payload structs. Events are emitted via `runtime.EventsEmit` and consumed by the frontend with `EventsOn`.
- **`app/session_persistence.go`** — Sessions are stored as versioned JSON files in `~/.deephermes/sessions/` (or `DeepHermesData/sessions/` in portable mode). Corrupt files are quarantined to a `corrupt/` subdirectory. Supports backup/restore and Markdown/JSON export.
- **`app/context_compaction.go`** — When a session exceeds 28 messages, old messages are summarized into a `ContextSummary` string and only the most recent 14 messages are sent to the API. The summary is prepended as a `<dynamic_context>` block in the system prompt.
- **`pkg/agent/agent.go`** — The agent loop: builds system prompt → sends messages to API → if response has `tool_calls`, executes them via the registry → appends tool results → loops (max 10 rounds). `sanitizeHistory` strips `reasoning_content` from non-tool-call assistant messages to avoid DeepSeek API errors. History is trimmed to 40 messages.
- **`pkg/agent/prompt.go`** — Builds the system prompt with XML-tagged sections: `<environment>`, `<mode_profile>`, `<stable_prompt>` (user's initial prompt/role card/world book), `<dynamic_context>` (summary), `<instructions>`, `<available_tools>`. The prompt structure is designed to keep stable sections first for DeepSeek cache hits.
- **`pkg/api/deepseek.go`** — DeepSeek HTTP client. Handles both non-streaming (`ChatContext`) and SSE streaming (`ChatStreamContext`). Retries on 429/5xx with exponential backoff. Thread-safe via `sync.RWMutex` snapshot pattern. Supports proxy, thinking mode (`deepseek-v4-*` models), and `reasoning_effort`.
- **`pkg/tools/registry.go`** — Tool interface: `Name()`, `Description()`, `Parameters()`, `Execute(ctx, args)`. The `Registry` enforces a `Policy` with three safety modes (`read_only`, `confirm`, `auto`), per-tool overrides, bash blocklist, a hard `AllowedDir` workspace boundary, and an approval callback that blocks until the frontend user approves/rejects. Risk classification: `read` (read_file, glob, grep), `network` (web_fetch, web_search), `write` (write_file, edit_file), `shell` (bash).
- **`pkg/config/config.go`** — YAML config with a three-layer merge: project `config.yaml` → user `~/.deephermes/config.yaml` (or portable `DeepHermesData/config.yaml`) → environment variables. Config is saved back to the user-level path.
- **`pkg/deepseek/`** — Context window management (`ContextManager`), token estimation (`ApproxTokens` ≈ len/3), thinking config, and prompt cache hashing.

### Frontend (React + TypeScript)

- **State management**: Zustand stores in `src/stores/` — `sessionStore` (chat state), `settingsStore` (preferences), `themeStore` (light/dark/anime), `layoutStore` (panels), `toolActivityStore` (tool execution tracking + TSV audit log export), `i18nStore` (i18n), `coworkStore` (sub-agents).
- **Wails bridge**: `src/lib/wails.ts` wraps every Go binding with an `invoke(fn, fallback)` pattern — if the Wails bridge is unavailable (e.g. `npm run dev` in a browser), it returns mock data. This allows frontend development without building the Go backend.
- **Auto-generated Go bindings**: `frontend/wailsjs/go/app/App.d.ts`, `App.js`, and `models.ts` must be kept in sync with the Go `App` struct's public methods and types.
- Styling: Tailwind CSS 3. Markdown: react-markdown + remark-gfm + rehype-highlight. Icons: lucide-react.

### Data Flow

```
User input → App.SendMessage() → Agent.RunStreamDetailed()
  → DeepSeek API (streaming SSE) → stream:delta events → frontend updates
  → If tool_calls in response:
      → registry.ExecuteAll() with safety policy check
      → If mode=confirm and risk≠read: emit tool:approval, block on channel
      → User approves/rejects in frontend → channel unblocks
      → tool:call + tool:result events emitted
      → Results appended to messages → loop continues (max 10 rounds)
  → stream:done event with full response, usage, metrics, finish_reason
```

### Session Context Management

Sessions maintain two message lists: `Messages` (user-visible, persisted) and `AgentMessages` (sent to API, may be compacted). When messages exceed 28, older ones are summarized into `ContextSummary` and only the recent 14 are sent. The summary is injected into the system prompt's `<dynamic_context>` section.

## Critical Gotchas

### DeepSeek `reasoning_content` Sanitization
DeepSeek's thinking mode returns `reasoning_content` on assistant messages. **This field must NOT be sent back to the API in regular assistant messages** — it causes a 400 error. The `sanitizeHistory` function in `pkg/agent/agent.go` strips it from all assistant messages except those with tool calls. Do not modify this logic without understanding the constraint.

### Wails Binding Sync
After adding or modifying public methods/structs on `app.App`, the generated TypeScript bindings must be updated:
- `frontend/wailsjs/go/app/App.d.ts`
- `frontend/wailsjs/go/app/App.js`
- `frontend/wailsjs/go/models.ts`

Run `wails generate module` or manually update these files. Missing sync causes runtime failures or TypeScript errors.

### PowerShell Syntax
The bash tool runs **Windows PowerShell** in this project's environment. Use `;` to chain commands, not `&&`. The system prompt in `prompt.go` already tells the model this.

### Build Tags
A plain `go build .` produces an exe that fails at launch with "Wails applications will not build without the correct build tags." Always use the build script.

### Workspace Boundary
Tool file access, glob/grep search, bash working directory, and the Wails file browser are restricted to the app's current working directory via `tools.Policy.AllowedDir` and `tools.ValidatePath`. Keep this boundary check before approval previews or rollback snapshots so rejected paths are not read before they are blocked.

## Configuration

- Default config: `config.yaml` at project root (merged with user-level config)
- Config load order: `config.yaml` → `~/.deephermes/config.yaml` → env vars
- Key env vars: `DEEPSEEK_API_KEY`, `DEEPSEEK_MODEL`, `DEEPHERMES_PROXY_URL`, `DEEPHERMES_TOOL_MODE`, `DEEPHERMES_OCR_*`
- Storage: `~/.deephermes/` (standard) or `DeepHermesData/` next to executable (portable mode)
- Tool safety modes: `read_only` (no writes), `confirm` (user approval for write/shell/network), `auto` (no confirmation)
- App modes: `code` (default), `rp` (roleplay), `writing`, `chat` — affects system prompt framing

## Key Dependencies

- Go 1.26.1, Wails v2.12.0, yaml.v3
- React 18, Vite 5, TypeScript 5.5, Tailwind CSS 3, Zustand 4, Vitest 2, lucide-react
