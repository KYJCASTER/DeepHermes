# Repository Guidelines

## Project Structure & Module Organization

DeepHermes is a Windows desktop AI agent built with Wails v2, Go, React, Vite, and Tailwind CSS. The desktop entry point is `main.go`; CLI-oriented code is in `cli.go` and `cmd/deephermes/`. Wails-facing services live in `app/`, including sessions, OCR, metrics, persistence, and frontend-bound methods. Shared backend packages are under `pkg/` (`agent`, `api`, `config`, `tools`, `cowork`, `memory`, `subagent`). The React UI is in `frontend/src/`, with stores in `frontend/src/stores/`, components in `frontend/src/components/`, and Wails-generated bindings in `frontend/wailsjs/`. Scripts live in `scripts/`, build assets in `build/`, and embedded web templates in `web/`.

## Build, Test, and Development Commands

- `go test ./...`: run all Go unit tests from the repository root.
- `go test -run TestName ./pkg/tools/...`: run a focused Go test.
- `cd frontend && npm ci`: install locked frontend dependencies.
- `cd frontend && npm run dev`: start the Vite development server.
- `cd frontend && npm run build`: type-check TypeScript and build frontend assets.
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1`: build `build\bin\DeepHermes.exe`.
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-installer.ps1 -Version 1.0.0`: build the Windows installer.

Do not use plain `go build` for release artifacts; Wails desktop builds require the `desktop,production` tags.

## Coding Style & Naming Conventions

Format Go with `gofmt`; keep package names short, lowercase, and domain-oriented. Test files use Go’s `*_test.go` and `TestXxx` conventions. TypeScript uses two-space indentation, PascalCase React components, `useXStore` Zustand hooks, and double-quoted imports. Prefer existing helpers in `frontend/src/lib/` and backend packages before adding new abstractions. Treat `frontend/wailsjs/` as generated output.

## Testing Guidelines

Backend tests are standard Go tests colocated with implementation files in `app/` and `pkg/`. Add focused tests for agent behavior, config parsing, persistence, tools, and session logic. Frontend CI enforces TypeScript checks via `npx tsc --noEmit`; run `npm run build` before UI-heavy changes.

## Commit & Pull Request Guidelines

Recent history uses concise imperative commits such as `Improve desktop usability and DeepSeek optimization`. Keep subjects action-oriented and under roughly 72 characters. Pull requests should describe behavior changes, list verification commands, link issues, and include screenshots or recordings for visible UI changes. Call out configuration, security, or packaging impacts.

## Security & Configuration Tips

Never commit real API keys or local user data. Use environment overrides such as `DEEPSEEK_API_KEY`, `DEEPHERMES_PROXY_URL`, and `DEEPHERMES_TOOL_MODE` for local testing. Default configuration lives in `config.yaml`; persisted runtime data is stored under `~/.deephermes/` or `DeepHermesData/` in portable mode.
