# DeepHermes

DeepHermes is a Windows desktop agent client optimized for DeepSeek models. It is built with Wails, Go, React, Vite, Tailwind CSS, and Zustand.

## Features

- DeepSeek API key setup and local configuration persistence.
- Default model profile tuned for `deepseek-v4-pro`.
- Streaming chat UI with reasoning content support.
- Agent workspace panels for files, sessions, status, and cowork/subagent flows.
- Light and dark themes with persisted preference.
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
max_tokens: 32768
temperature: 0.7
thinking_enabled: false
auto_cowork: false
```

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
