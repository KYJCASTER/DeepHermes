# Release Notes

## Unreleased

### Added

- DeepSeek V4-oriented settings, model profiles, finish-reason tracking, and one-click continuation for truncated responses.
- Context compaction summary viewing and editing.
- OCR provider presets for pasted screenshots and image files.
- Tool activity panel with approval previews, rollback support, TSV audit-log export, per-tool safety overrides, and Bash blocklist rules.
- Session backup/restore, corrupt session quarantine, and Markdown/JSON session export.
- Command palette, prompt templates, slash commands, character-card import, `@file` references, and improved workspace panels.
- GitHub Actions workflows for Go tests, frontend type checks, Windows builds, and tag-based releases.
- Contributor and handoff documentation.

### Changed

- Improved desktop layout, accessibility focus states, responsive panels, and light/dark/anime themes.
- Expanded local diagnostics, settings import/export, proxy/timeout/retry controls, portable mode, and tray behavior.
- Hardened DeepSeek reasoning-content handling so non-tool assistant reasoning is stripped before reuse.

### Tests

- Added Go coverage for session persistence, session actions, OCR, character cards, config, API errors, streaming tool calls, tool policy overrides, and Bash blocklists.
- Added frontend Vitest coverage for tool activity retention and TSV audit-log export.

### Notes

- Windows packaging must use the Wails build scripts; plain `go build` does not include the required desktop build tags.
- API keys and OCR keys are stored locally. Do not publish exported settings, session backups, or diagnostics without reviewing them for secrets.
