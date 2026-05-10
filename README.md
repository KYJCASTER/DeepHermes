# DeepHermes Desktop

DeepSeek-optimized Windows desktop AI agent built with Wails, Go, React, and Vite.

DeepHermes is a Windows desktop agent client optimized for DeepSeek models. It includes a polished light/dark UI, streaming chat, reasoning content support, local configuration, and cowork/subagent panels.

## Features

- DeepSeek API key setup and local configuration persistence.
- Default model profile tuned for `deepseek-v4-pro`.
- Persistent chat sessions restored across app restarts.
- Message editing, deletion, regeneration, and branch-from-message workflows.
- Streaming chat UI with reasoning content support and display controls.
- Token usage, output speed, reasoning token, and DeepSeek cache hit/miss tracking.
- DeepSeek model profiles for V4 Pro and V4 Flash, including context window, output limits, recommended parameters, legacy-model warnings, and estimated CNY cost.
- Custom initial system prompt with tavern-style roleplay and interactive-fiction presets.
- Role card and world book fields for lightweight tavern-style roleplay setup.
- Agent workspace panels for files, sessions, status, and cowork/subagent flows.
- Resizable session and file sidebars with persisted layout preference.
- Portable mode, settings import/export, diagnostics, build metadata, and persisted window size/position.
- Optional close-to-background behavior with single-instance relaunch restoring the hidden window.
- Light, dark, and fresh anime themes with persisted preference.
- DeepSeek-inspired interface styling with smooth panel, button, and empty-state animations.
- Correct Windows desktop packaging through Wails build tags.

## Requirements

- Windows
- Go
- Node.js
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

Build the frontend:

```powershell
cd frontend
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
```

DeepSeek model prices are used only for local estimates in the UI. Prices can change, so check the official DeepSeek pricing page before treating estimates as billing truth.

The initial prompt, role card, and world book are injected as stable system-prompt sections for each request. Keep long-lived writing instructions there to improve continuity and make DeepSeek context caching more effective.

When `portable: true` is enabled, configuration and sessions are written to `DeepHermesData` next to the executable. When `minimize_to_tray: true` is enabled, closing the window hides it in the background; launching the exe again restores the existing window.

You can override the DeepSeek API key with:

```powershell
$env:DEEPSEEK_API_KEY = "your-api-key"
```

You can override the model with:

```powershell
$env:DEEPSEEK_MODEL = "deepseek-v4-pro"
```

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
