# DeepHermes Desktop

DeepSeek-optimized Windows desktop AI agent built with Wails, Go, React, and Vite.

DeepHermes is a Windows desktop agent client optimized for DeepSeek models. It includes a polished light/dark UI, streaming chat, reasoning content support, local configuration, and cowork/subagent panels.

## Features

- DeepSeek API key setup and local configuration persistence.
- API key validation before saving, plus visible request timeout, retry, and proxy settings.
- Default model profile tuned for `deepseek-v4-pro`.
- Persistent chat sessions restored across app restarts.
- Session search across recent chat content and metadata.
- Message editing, deletion, regeneration, and branch-from-message workflows.
- Streaming chat UI with reasoning content support and display controls.
- Token usage, output speed, reasoning token, and DeepSeek cache hit/miss tracking.
- DeepSeek finish-reason tracking with one-click continuation when a reply hits the output limit.
- `Ctrl+K` command palette for fast session, settings, cowork, theme, and tool-log actions.
- DeepSeek model profiles for V4 Pro and V4 Flash, including context window, output limits, recommended parameters, legacy-model warnings, and estimated CNY cost.
- Friendlier DeepSeek/API error explanations for invalid keys, 400 request issues, rate limits, timeouts, DNS, and proxy failures.
- Custom initial system prompt with tavern-style roleplay and interactive-fiction presets.
- Role card and world book fields for lightweight tavern-style roleplay setup.
- SillyTavern character-card import from JSON or PNG metadata into the role card and lorebook fields.
- Prompt template library and slash commands such as `/char`, `/lore`, `/summary`, `/export`, `/review`, `/translate`, and `/write`.
- Chat composer file drag-and-drop, pasted image OCR via configurable provider, and local input history navigation.
- `@file` references in the chat composer for quickly searching workspace files and attaching snippets.
- OpenAI-compatible OCR provider settings for screenshot/image text extraction without local OCR dependencies.
- Tool safety modes: read-only, confirm before sensitive tools, or trusted auto-execute.
- Confirmation prompts for destructive chat/session actions and model-initiated write, shell, or network tools.
- Diff previews before approving model-initiated file writes or edits.
- Workspace boundary checks for model tools and file browsing so local file access stays inside the current project directory.
- Tool activity panel showing model-initiated file, command, and network tool calls with arguments, results, and one-click rollback for file writes/edits.
- Per-tool safety overrides, Bash blocklist rules, and TSV audit-log export.
- Session backup/restore, corrupt session quarantine, and Markdown/JSON session export.
- Agent workspace panels for files, sessions, status, and cowork/subagent flows.
- Resizable session and file sidebars with persisted layout preference.
- Portable mode, settings import/export, diagnostics, build metadata, and persisted window size/position.
- Tabbed settings dialog for API, model, prompts, OCR, desktop, and safety controls.
- Optional close-to-background behavior with single-instance relaunch restoring the hidden window.
- Light, dark, and fresh anime themes with persisted preference.
- DeepSeek-inspired interface styling with smooth panel, button, and empty-state animations.
- Correct Windows desktop packaging through Wails build tags.

## Requirements

- Windows
- Go 1.26.1+
- Node.js 18+ (Node 22 is used in CI)
- Wails CLI v2.12+
- Frontend dependencies installed in `frontend/node_modules`

## Quick Start

Run the packaged desktop app:

```powershell
.\build\bin\DeepHermes.exe
```

On first launch, enter your DeepSeek API key in the welcome screen. The app stores configuration under your user profile in `.deephermes`.

## Development

Run frontend type checking:

```powershell
cd frontend
node .\node_modules\typescript\bin\tsc --noEmit
```

Run frontend tests and build:

```powershell
cd frontend
npm test
node .\node_modules\vite\bin\vite.js build
```

Run Go tests from the project root:

```powershell
$env:GOCACHE = "D:\DeepHermes\.cache\go-build"
go test ./...
```

## Windows Build

Use the included build script:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1
```

The executable is written to:

```text
build\bin\DeepHermes.exe
```

To build a Windows NSIS installer, use:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-installer.ps1 -Version 1.0.0
```

This uses the Wails CLI with `-nsis`, `-platform windows/amd64`, and the same `desktop,production` build tags. NSIS packaging requires the Wails toolchain and its Windows packaging prerequisites to be installed locally.

Do not package this app with a plain `go build` command. Wails desktop builds need the correct build tags. Without them, the executable can show this error:

```text
Wails applications will not build without the correct build tags.
```

The build script handles the required tags:

```powershell
go build -buildvcs=false -tags "desktop,production" -ldflags "-w -s -H windowsgui" -o build\bin\DeepHermes.exe .
```

## Configuration

Default settings live in `config.yaml`:

```yaml
model: deepseek-v4-pro
mode: code
portable: false
minimize_to_tray: false
max_tokens: 32768
temperature: 0.7
thinking_enabled: false
reasoning_display: collapse
auto_cowork: false
initial_prompt: ""
role_card: ""
world_book: ""
api:
  base_url: https://api.deepseek.com
  timeout_seconds: 120
  max_retries: 3
  proxy_url: ""
ocr:
  enabled: false
  provider: openai_compatible
  base_url: ""
  model: ""
  prompt: "Extract all readable text from this image. Preserve line breaks when useful. If there is no readable text, briefly describe the visible content."
  timeout_seconds: 60
  max_image_bytes: 8388608
safety:
  tool_mode: confirm
  tool_overrides: {}
  bash_blocklist: []
```

DeepSeek model prices are used only for local estimates in the UI. Prices can change, so check the official DeepSeek pricing page before treating estimates as billing truth.

The initial prompt, role card, and world book are injected as stable system-prompt sections for each request. Keep long-lived writing instructions there to improve continuity and make DeepSeek context caching more effective.

When `portable: true` is enabled, configuration and sessions are written to `DeepHermesData` next to the executable. When `minimize_to_tray: true` is enabled, closing the window hides it in the background; launching the exe again restores the existing window.

OCR uses an API-based provider only. Configure an OpenAI-compatible vision endpoint, model, and OCR API key in Settings, or set:

```powershell
$env:DEEPHERMES_OCR_API_KEY = "your-ocr-provider-key"
$env:DEEPHERMES_OCR_BASE_URL = "https://api.example.com/v1"
$env:DEEPHERMES_OCR_MODEL = "your-vision-model"
```

You can override the DeepSeek API key with:

```powershell
$env:DEEPSEEK_API_KEY = "your-api-key"
```

Optional network and safety overrides:

```powershell
$env:DEEPHERMES_PROXY_URL = "http://127.0.0.1:7890"
$env:DEEPHERMES_TOOL_MODE = "confirm" # read_only, confirm, or auto
```

Model-initiated file tools, shell execution, workspace search, and the file browser are restricted to the app's current working directory. Launch DeepHermes from the project directory you want the agent to use as its workspace.

You can override the model with:

```powershell
$env:DEEPSEEK_MODEL = "deepseek-v4-pro"
```

## Session Data & Privacy

DeepHermes stores API keys, settings, sessions, memory, and diagnostics locally. Standard mode writes under `~/.deephermes/`; portable mode writes under `DeepHermesData` next to the executable. Session files are versioned JSON. If a session file is corrupt, startup moves it into a `corrupt` folder instead of blocking the app.

Use Settings > Desktop to back up or restore all sessions. Use the chat template menu to export the active session as Markdown or JSON. Audit logs from the tool activity panel export as TSV and may include tool arguments or file paths, so review them before sharing.

Never commit real API keys, exported settings containing secrets, session backups, or diagnostic logs.

## Project Structure

```text
app/        Wails application services and event plumbing
cmd/        CLI and command entry points
frontend/   React/Vite desktop UI
pkg/        Agent, DeepSeek, config, memory, tools, and cowork packages
scripts/    Build scripts
web/        Embedded web assets
```

## Notes

- The frontend build uses relative asset paths so the desktop bundle can load packaged assets reliably.
- The theme preference is saved in browser local storage.
- If the UI looks stale after rebuilding, make sure you launched `build\bin\DeepHermes.exe` from the latest build output.

## License

MIT
