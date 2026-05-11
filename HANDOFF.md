# DeepHermes 项目交接文档

> **日期**: 2026-05-11
> **当前版本**: 未正式发版 (开发中)
> **技术栈**: Go 1.26.1 + Wails v2.12.0 + React 18 + TypeScript 5.5 + Vite 5 + Tailwind CSS 3 + Zustand 4

---

## 1. 项目简介

DeepHermes 是一个 **Windows 桌面 AI Agent 客户端**，通过 Wails v2 构建（Go 后端 + WebView2 前端）。核心功能：

- 对接 DeepSeek API 进行流式聊天补全
- 提供工具执行环境（文件读写、Bash、Web 搜索等）
- 支持酒馆 RP 模式（角色卡、世界书）
- 安全控制（工具审批、回滚、审计）
- 双运行模式：桌面 GUI（默认）/ CLI（`--cli`）

---

## 2. 环境搭建

### 前置依赖

| 工具 | 版本 | 说明 |
|------|------|------|
| Go | 1.26.1+ | 后端编译 |
| Node.js | 18+ | 前端构建 |
| Wails CLI | v2.12+ | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| WebView2 Runtime | 最新 | Windows 10/11 通常自带 |

### 环境变量

```
DEEPSEEK_API_KEY=sk-xxxx          # 必须
DEEPSEEK_MODEL=deepseek-chat      # 可选，默认 deepseek-chat
DEEPHERMES_PROXY_URL=             # 可选，代理地址
DEEPHERMES_TOOL_MODE=confirm      # read_only / confirm / auto
DEEPHERMES_OCR_*                  # OCR 相关配置
```

### 快速启动

```bash
# 前端
cd frontend && npm install && npm run build

# 开发模式（热重载）
wails dev

# 生产构建
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1
```

> **注意**: 不要用 `go build` 直接编译，Wails 需要特定 build tags (`desktop,production`) 和 ldflags。

---

## 3. 代码架构总览

```
DeepHermes/
├── main.go                    # 入口，CLI/GUI 模式分发
├── app/                       # Wails 绑定层（前端可调用的所有方法）
│   ├── app.go                 # App 主结构体，会话/消息/设置
│   ├── context_compaction.go  # 上下文压缩摘要
│   ├── character_card.go      # 角色卡管理
│   ├── ocr.go                 # OCR 功能
│   └── events/events.go       # 前后端事件定义
├── pkg/
│   ├── agent/                 # Agent 循环：API调用→解析工具→执行→回传
│   │   ├── agent.go           # 核心循环 + sanitizeHistory
│   │   └── prompt.go          # 系统提示词构建
│   ├── api/deepseek.go        # DeepSeek HTTP 客户端，流式 SSE
│   ├── config/config.go       # YAML 配置加载
│   ├── tools/                 # 工具注册表 + 各工具实现
│   │   ├── registry.go        # 工具注册、安全策略、per-tool 覆盖
│   │   ├── bash.go            # Bash 工具
│   │   ├── file.go            # 文件读写工具
│   │   ├── search.go          # Glob/Grep
│   │   └── web.go             # WebFetch/WebSearch
│   ├── memory/                # 持久记忆
│   ├── cowork/                # 多 Agent 协作
│   └── subagent/              # 子 Agent 生成
├── frontend/
│   ├── src/
│   │   ├── stores/            # Zustand 状态管理
│   │   │   ├── sessionStore.ts      # 聊天会话状态
│   │   │   ├── settingsStore.ts     # 用户设置
│   │   │   ├── toolActivityStore.ts # 工具活动追踪+审计
│   │   │   ├── themeStore.ts        # 主题（light/dark/anime）
│   │   │   ├── layoutStore.ts       # 面板布局
│   │   │   └── i18nStore.ts         # 国际化
│   │   ├── components/
│   │   │   ├── chat/          # 聊天视图、消息气泡
│   │   │   ├── settings/      # 设置对话框
│   │   │   ├── tools/         # 工具活动面板
│   │   │   ├── layout/        # 侧栏、标题栏、状态栏
│   │   │   └── command/       # 命令面板
│   │   └── lib/               # 工具函数、Wails 桥接
│   └── wailsjs/go/            # 自动生成的 Go 绑定
├── scripts/                   # 构建脚本
├── .github/workflows/         # CI/CD（已创建但未启用）
└── config.yaml                # 默认配置
```

### 数据流

```
用户输入 → App.SendMessage()
         → Agent.Run() → DeepSeek API (流式 SSE)
         → 解析 tool_calls → 执行工具（需审批则等待用户确认）
         → 结果追加到消息 → 继续循环直到无工具调用
         → 最终回复通过 Wails 事件推送到前端
```

---

## 4. 已完成功能清单

### 核心功能
- 多会话持久化
- DeepSeek 流式输出 + 思考过程（reasoning_content）显示控制
- Token/速度/缓存命中统计
- 长上下文预算管理
- 一键续写（finish_reason=length 时触发，隐藏消息方式）

### 角色扮演
- 初始提示词 / 角色卡 / 世界书
- 酒馆 RP 格式支持
- 聊天模板

### 工具系统
- 工具调用确认 / Diff 预览 / 执行日志
- 文件修改回滚
- 三级安全模式（read_only / confirm / auto）
- Per-tool 权限覆盖（后端已实现）
- Bash 命令黑名单（后端已实现）
- 审计日志导出逻辑（Store 已实现，UI 按钮未加）

### UI/UX
- 二次元主题 + Light/Dark
- 文件拖拽 / @file 引用
- OCR（多提供商预设）
- Prompt 模板 / 快捷指令
- 设置导入导出
- 便携模式
- 窗口位置保存 / 托盘隐藏
- 响应式布局 + 无障碍焦点样式
- 上下文摘要查看/编辑

### DevOps
- GitHub Actions CI/CD 流水线（build.yml + release.yml）

---

## 5. 当前进行中的任务

### 任务 #6: 安全控制细化（进行中）

**已完成部分:**
- `pkg/config/config.go` — `SafetyConfig` 新增 `ToolOverrides map[string]string` 和 `BashBlocklist []string`
- `pkg/tools/registry.go` — `Policy` 支持 per-tool 模式覆盖 + bash 黑名单检查
- `frontend/src/stores/toolActivityStore.ts` — `exportAuditLog()` 方法导出 TSV 格式

**未完成部分（接手断点）:**
1. **ToolActivityPanel 导出按钮** — `frontend/src/components/tools/ToolActivityPanel.tsx` 需要添加一个"导出审计日志"按钮，调用 `useToolActivityStore.getState().exportAuditLog()` 并触发文件下载
2. **设置页 Safety Tab** — `SettingsDialog.tsx` 需要添加 per-tool override 的 UI（下拉选择每个工具的安全模式）
3. **Go 后端测试** — `pkg/tools/registry_test.go` 需要补充 per-tool override 和 bash blocklist 的测试用例
4. **TypeScript 类型检查** — 运行 `tsc --noEmit` 确认无类型错误

---

## 6. 待开发任务清单（按优先级排序）

| # | 任务 | 优先级 | 说明 |
|---|------|--------|------|
| 7 | 测试覆盖补强 | 高 | 前端组件测试、Wails 集成测试、DeepSeek API mock 端到端 |
| 8 | 会话存储增强 | 中 | 备份/恢复、数据迁移版本、损坏恢复、多格式导出 |
| 9 | README 和 Release Notes | 中 | 功能截图、安装方式、Key 获取、配置说明、隐私说明 |

### 路线图建议（更远期）
- 一键续写优化迭代（RP 场景微调）
- 安装包优化：代码签名、自动更新通道
- 长上下文增强：压缩前预览、超长会话自动归档、缓存命中提示
- OCR：图片作为多模态消息发送、OCR 结果可编辑确认
- 多 Agent 协作 UI 完善

---

## 7. 关键开发注意事项（踩坑记录）

### 构建相关
- **必须用 Wails 脚本构建**，不能 `go build`。正确命令：`powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-windows.ps1`
- **PowerShell 不支持 `&&`**，用 `;` 或分开执行

### DeepSeek API
- **reasoning_content 清洗**：DeepSeek thinking 模式返回的 `reasoning_content` 在普通 assistant 回复中不能传回 API，否则报错。相关逻辑在 `pkg/agent/agent.go` 的 `sanitizeHistory` — 请勿随意修改
- **finish_reason**：`length`/`max_tokens` 时触发续写，逻辑在前端 `ChatView.tsx`

### Wails 绑定
- 每次修改 `app/` 包的公开方法或结构体后，**必须同步更新**：
  - `frontend/wailsjs/go/app/App.d.ts`
  - `frontend/wailsjs/go/app/App.js`
  - `frontend/wailsjs/go/models.ts`
- 或者运行 `wails generate module` 自动生成

### Git 操作
- **当前有大量未提交改动**（30个文件，约3500行增改），禁止 `git reset --hard`
- Push 前先 `git pull --rebase origin main` 解决远端差异
- **绝对不要提交 API Key**，检查 config.yaml 和日志文件

### 编码问题
- 源码保持 UTF-8
- PowerShell 中文乱码是终端显示问题，不是源码问题

### 验证方式
```bash
# 前端类型检查
cd frontend && npx tsc --noEmit

# Go 测试
go test ./...

# Go 单包测试
go test ./pkg/tools/...
go test -run TestToolOverrides ./pkg/tools/...
```

---

## 8. 未提交代码说明

当前工作区有 **30 个已修改文件 + 多个新增文件** 未提交，包含最近 5 个已完成任务和 1 个进行中任务的全部代码：

### 修改的文件（核心）
| 文件 | 改动内容 |
|------|----------|
| `app/app.go` | 新增 ContinueLastResponse, GetContextSummary, UpdateContextSummary, ArchiveSession, ListOCRPresets 等方法 |
| `pkg/agent/agent.go` | Agent 循环支持续写模式 |
| `pkg/api/deepseek.go` | finish_reason 识别优化 |
| `pkg/config/config.go` | SafetyConfig 新增 ToolOverrides + BashBlocklist |
| `pkg/tools/registry.go` | per-tool 安全模式覆盖、bash 黑名单 |
| `frontend/src/stores/sessionStore.ts` | continueLastResponse action |
| `frontend/src/stores/toolActivityStore.ts` | exportAuditLog() |
| `frontend/src/stores/settingsStore.ts` | OCR 预设支持 |
| `frontend/src/components/settings/SettingsDialog.tsx` | OCR 预设 UI + 上下文摘要编辑 |
| `frontend/src/styles/index.css` | 无障碍样式 + 性能优化 CSS |

### 新增文件
| 文件 | 说明 |
|------|------|
| `.github/workflows/build.yml` | CI 流水线 |
| `.github/workflows/release.yml` | 自动发布 |
| `app/character_card.go` | 角色卡管理 |
| `app/ocr.go` | OCR 功能 |
| `frontend/src/components/tools/ToolActivityPanel.tsx` | 工具活动面板 |
| `frontend/src/components/command/CommandPalette.tsx` | 命令面板 |
| `frontend/src/stores/i18nStore.ts` | 国际化 |
| `frontend/src/lib/chatTemplates.ts` | 聊天模板 |

### 建议提交策略
建议按功能分批提交：
1. `feat: add continue-generation for truncated responses`
2. `feat: UI accessibility, performance, and responsive improvements`
3. `ci: add GitHub Actions build and release workflows`
4. `feat: context compaction summary view/edit`
5. `feat: OCR provider presets and error enhancement`
6. `feat: per-tool safety overrides and bash blocklist (WIP)`

---

## 9. 已知问题和技术债

1. **类型检查未通过验证** — 最近一轮改动后尚未跑 `tsc --noEmit`，可能有类型错误
2. **Go 测试未全量跑** — per-tool override 相关测试新增但未验证通过
3. **Wails 绑定可能需要重新生成** — 如果 `app/` 新增方法后手动更新的绑定有遗漏
4. **CI 流水线未实际触发** — `.github/workflows/` 创建但未 push 到远端验证
5. **安全控制 UI 缺失** — 后端逻辑已就绪，前端操作界面未完成
6. **无自动化前端测试** — 没有 Jest/Vitest/Playwright 配置

---

## 10. 配置文件说明

### `config.yaml`（项目根目录）

```yaml
api:
  base_url: "https://api.deepseek.com"
  model: "deepseek-chat"
  max_tokens: 4096
  temperature: 0.7

safety:
  mode: "confirm"           # 全局默认安全模式
  tool_overrides:           # per-tool 覆盖（新增）
    ReadFile: "auto"
    Bash: "confirm"
  bash_blocklist:           # bash 命令黑名单（新增）
    - "rm -rf /"
    - "format"

context:
  max_budget: 128000        # 最大上下文 token 数
```

### 存储位置
- 标准模式: `~/.deephermes/`（会话数据、记忆、日志）
- 便携模式: `DeepHermesData/`（与 exe 同目录）

---

## 11. 快速接手检查清单

- [ ] 克隆仓库，安装 Go 1.26.1+、Node 18+、Wails CLI
- [ ] 设置 `DEEPSEEK_API_KEY` 环境变量
- [ ] `cd frontend && npm install`
- [ ] `wails dev` 启动开发模式，验证能正常运行
- [ ] 阅读本文档第 5 节，从安全控制的未完成部分继续
- [ ] 运行 `go test ./...` 和 `npx tsc --noEmit` 确认当前代码状态
- [ ] 决定是否先分批提交当前工作区的改动

---

## 12. 联系方式和资源

- **仓库**: GitHub (ad201/deephermes)
- **DeepSeek API 文档**: https://platform.deepseek.com/docs
- **Wails 文档**: https://wails.io/docs/
- **项目 CLAUDE.md**: 包含更详细的架构说明和开发命令

---

*本文档由前任开发者通过 Claude Code 生成于 2026-05-11，反映交接时的项目状态。*
