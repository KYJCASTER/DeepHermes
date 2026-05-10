import { useEffect, useMemo, useRef, useState } from "react";
import { BarChart3, Brain, ChevronDown, Clock3, Database, Gauge, ScrollText, Send, Sparkles, Square, X, Zap } from "lucide-react";
import { useSessionStore } from "../../stores/sessionStore";
import { useSettingsStore } from "../../stores/settingsStore";
import { useI18n } from "../../stores/i18nStore";
import {
  CHAT_MODE_PRESETS,
  MODEL_OPTIONS,
  chatModePreset,
  contextUsagePercent,
  estimateCacheSavingsCny,
  estimateContextBudget,
  estimateCostCny,
  formatCny,
  formatTokenLimit,
  modelLabel,
  modelProfile,
  supportsThinking,
} from "../../lib/models";
import MessageBubble from "./MessageBubble";
import ThinkingBanner from "./ThinkingBanner";

function compactNumber(value: number) {
  return new Intl.NumberFormat(undefined, { notation: "compact", maximumFractionDigits: 1 }).format(value || 0);
}

function cacheRate(hit: number, miss: number) {
  const total = hit + miss;
  return total > 0 ? Math.round((hit / total) * 100) : 0;
}

export default function ChatView() {
  const [input, setInput] = useState("");
  const [showThinking, setShowThinking] = useState(true);
  const [showUsage, setShowUsage] = useState(false);
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const sessions = useSessionStore((s) => s.sessions);
  const sendMessage = useSessionStore((s) => s.sendMessage);
  const abortMessage = useSessionStore((s) => s.abortMessage);
  const editMessage = useSessionStore((s) => s.editMessage);
  const deleteMessage = useSessionStore((s) => s.deleteMessage);
  const regenerateMessage = useSessionStore((s) => s.regenerateMessage);
  const branchSession = useSessionStore((s) => s.branchSession);
  const model = useSettingsStore((s) => s.model);
  const mode = useSettingsStore((s) => s.mode);
  const maxTokens = useSettingsStore((s) => s.maxTokens);
  const thinkingEnabled = useSettingsStore((s) => s.thinkingEnabled);
  const reasoningDisplay = useSettingsStore((s) => s.reasoningDisplay);
  const initialPrompt = useSettingsStore((s) => s.initialPrompt);
  const roleCard = useSettingsStore((s) => s.roleCard);
  const worldBook = useSettingsStore((s) => s.worldBook);
  const hasInitialPrompt = useSettingsStore((s) => Boolean(s.initialPrompt.trim() || s.roleCard.trim() || s.worldBook.trim()));
  const saveSettings = useSettingsStore((s) => s.save);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const { t } = useI18n();
  const session = sessions.find((s) => s.id === activeSessionId);
  const canThink = supportsThinking(model);
  const profile = modelProfile(model);

  const quickPrompts = useMemo(
    () =>
      hasInitialPrompt
        ? [t("chat.rpQuickScene"), t("chat.rpQuickContinue"), t("chat.rpQuickCharacter"), t("chat.rpQuickWorld")]
        : [t("chat.quickExplain"), t("chat.quickRefactor"), t("chat.quickReview"), t("chat.quickPlan")],
    [hasInitialPrompt, t]
  );

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [session?.messages]);

  useEffect(() => {
    setShowUsage(false);
  }, [activeSessionId]);

  const handleSend = () => {
    const text = input.trim();
    if (!text || !activeSessionId || session?.streaming) return;
    sendMessage(activeSessionId, text).catch((e) => console.error("Failed to send message:", e));
    setInput("");
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleInput = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value);
    const el = e.target;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 220) + "px";
  };

  if (!session) {
    return (
      <div className="workspace-main flex flex-1 items-center justify-center text-dim">
        {t("sessions.emptyHint")}
      </div>
    );
  }

  const currentCacheRate = cacheRate(session.usage.promptCacheHitTokens, session.usage.promptCacheMissTokens);
  const estimatedCost = estimateCostCny(model, session.usage);
  const cacheSavings = estimateCacheSavingsCny(model, session.usage);
  const budget = estimateContextBudget(
    model,
    { maxTokens, initialPrompt, roleCard, worldBook },
    session.messages,
    session.contextSummaryTokens || 0
  );
  const contextPercent = contextUsagePercent(model, budget.totalReservedTokens);
  const statusTone = session.status !== "idle" ? "text-yellow" : "text-green";
  const currentMode = chatModePreset(mode);

  const applyMode = (nextMode: string) => {
    const preset = chatModePreset(nextMode);
    saveSettings({
      mode: preset.id,
      model: preset.model,
      maxTokens: preset.maxTokens,
      temperature: preset.temperature,
      thinkingEnabled: preset.thinkingEnabled,
      reasoningDisplay: preset.reasoningDisplay,
    }).catch((err) => console.error("Failed to apply mode:", err));
  };

  return (
    <div className="workspace-main flex flex-1 flex-col overflow-hidden">
      <header className="workspace-header shrink-0 px-4 py-3 sm:px-6 sm:py-4">
        <div className="chat-header-layout mx-auto w-full max-w-6xl">
          <div className="min-w-0 flex-1">
            <div className="mb-1 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-xs text-dim">
              <span className={`h-2 w-2 shrink-0 rounded-full ${session.status === "idle" ? "bg-green" : "agent-running bg-yellow"}`} />
              <span className={`shrink-0 ${statusTone}`}>
                {session.status === "thinking"
                  ? t("status.thinking")
                  : session.status === "streaming"
                    ? t("status.streaming")
                    : session.status === "executing"
                      ? t("status.executing")
                      : t("status.ready")}
              </span>
              <span className="shrink-0 text-border">/</span>
              <span className="min-w-0 max-w-full truncate">{modelLabel(model)}</span>
            </div>
            <h1 className="truncate text-xl font-semibold text-text">{session.name}</h1>
          </div>

          <div className="usage-popover-anchor">
            <button
              onClick={() => setShowUsage((value) => !value)}
              className={`usage-button ${showUsage ? "border-accent/40 text-accent" : ""}`}
              title={t("status.usageDetails")}
            >
              <BarChart3 size={14} />
              {t("status.usage")}
              <ChevronDown size={13} className={`transition ${showUsage ? "rotate-180" : ""}`} />
            </button>

            {showUsage && (
              <div className="usage-popover">
                <div className="mb-3 flex items-center justify-between gap-3">
                  <div>
                    <p className="text-sm font-semibold text-text">{t("status.usageDetails")}</p>
                    <p className="mt-0.5 text-xs text-dim">{modelLabel(model)}</p>
                  </div>
                  <button onClick={() => setShowUsage(false)} className="icon-button h-7 w-7" title={t("settings.cancel")}>
                    <X size={14} />
                  </button>
                </div>

                <div className="context-budget-card">
                  <div className="mb-2 flex items-center justify-between gap-3">
                    <span className="text-xs font-semibold text-text">{t("context.budget")}</span>
                    <span className="text-xs text-dim">{budget.usagePercent}%</span>
                  </div>
                  <div className="context-budget-track">
                    <div className="context-budget-fill" style={{ width: `${Math.max(2, budget.usagePercent)}%` }} />
                  </div>
                  <div className="mt-2 grid grid-cols-2 gap-2 text-xs">
                    <span className="text-dim">
                      {t("context.used")} <strong className="text-text">{formatTokenLimit(budget.totalReservedTokens)}</strong>
                    </span>
                    <span className="text-dim">
                      {t("context.remaining")} <strong className="text-text">{formatTokenLimit(budget.remainingTokens)}</strong>
                    </span>
                    <span className="text-dim">
                      {t("context.prompt")} <strong className="text-text">{formatTokenLimit(budget.promptTokens)}</strong>
                    </span>
                    <span className="text-dim">
                      {t("context.estimate")} <strong className="text-text">{formatCny(budget.estimatedCost)}</strong>
                    </span>
                  </div>
                </div>

                <div className="usage-grid">
                  <div className="usage-card">
                    <span className="usage-card-label">
                      <Database size={13} />
                      {t("status.tokens")}
                    </span>
                    <strong>{compactNumber(session.usage.totalTokens)}</strong>
                  </div>
                  <div className="usage-card">
                    <span className="usage-card-label">
                      <Gauge size={13} />
                      {t("status.cache")}
                    </span>
                    <strong>{currentCacheRate}%</strong>
                  </div>
                  <div className="usage-card">
                    <span className="usage-card-label">
                      <Brain size={13} />
                      {t("status.context")}
                    </span>
                    <strong>{contextPercent}% / {formatTokenLimit(profile.contextWindow)}</strong>
                  </div>
                  <div className="usage-card">
                    <span className="usage-card-label">{t("status.cost")}</span>
                    <strong>{formatCny(estimatedCost)}</strong>
                  </div>
                  {session.lastRun && (
                    <>
                      <div className="usage-card">
                        <span className="usage-card-label">
                          <Zap size={13} />
                          {t("status.speed")}
                        </span>
                        <strong>{session.lastRun.tokensPerSec.toFixed(1)} tok/s</strong>
                      </div>
                      <div className="usage-card">
                        <span className="usage-card-label">
                          <Clock3 size={13} />
                          {t("status.firstToken")}
                        </span>
                        <strong>{session.lastRun.firstTokenMs} ms</strong>
                      </div>
                    </>
                  )}
                  {hasInitialPrompt && (
                    <div className="usage-card">
                      <span className="usage-card-label">
                        <ScrollText size={13} />
                        {t("status.initialPrompt")}
                      </span>
                      <strong>{t("status.enabled")}</strong>
                    </div>
                  )}
                  {cacheSavings > 0 && (
                    <div className="usage-card text-green">
                      <span className="usage-card-label">{t("status.saved")}</span>
                      <strong>{formatCny(cacheSavings)}</strong>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </header>

      {canThink && thinkingEnabled && session.status === "thinking" && showThinking && (
        <ThinkingBanner onDismiss={() => setShowThinking(false)} />
      )}

      <div className="flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex w-full max-w-5xl flex-col">
          {session.messages.length === 0 && (
            <div className="fade-up mx-auto mt-16 w-full max-w-3xl text-center text-dim">
              <div className="ds-mark mx-auto mb-5 flex h-12 w-12 items-center justify-center rounded bg-surface/90 text-accent shadow-sm">
                <Sparkles size={22} />
              </div>
              <h3 className="mb-2 text-2xl font-semibold text-text">{t("chat.startConversation")}</h3>
              <p className="mx-auto max-w-lg text-sm leading-6">
                {hasInitialPrompt ? t("chat.startHintPrompt") : t("chat.startHint")}
              </p>
              <div className="mt-7 grid grid-cols-1 gap-2 sm:grid-cols-2">
                {quickPrompts.map((prompt) => (
                  <button
                    key={prompt}
                    onClick={() => {
                      setInput(prompt);
                      textareaRef.current?.focus();
                    }}
                    className="motion-lift rounded border border-border bg-surface/70 px-4 py-3 text-left text-sm text-text transition hover:border-accent/40 hover:bg-panel"
                  >
                    {prompt}
                  </button>
                ))}
              </div>
            </div>
          )}

          {session.messages.map((msg, i) => (
            <MessageBubble
              key={i}
              role={msg.role}
              content={msg.content}
              reasoningContent={msg.reasoningContent}
              isStreaming={i === session.messages.length - 1 && msg.role === "assistant" && session.streaming}
              disabled={session.streaming}
              onEdit={(content) => {
                if (activeSessionId) return editMessage(activeSessionId, i, content);
              }}
              onDelete={() => {
                if (activeSessionId) return deleteMessage(activeSessionId, i);
              }}
              onRegenerate={() => {
                if (activeSessionId) return regenerateMessage(activeSessionId, i);
              }}
              onBranch={() => {
                if (activeSessionId) {
                  branchSession(activeSessionId, i).catch((err) => console.error("Failed to branch session:", err));
                }
              }}
            />
          ))}
          <div ref={messagesEndRef} />
        </div>
      </div>

      <footer className="shrink-0 px-6 pb-5">
        <div className="composer-shell mx-auto max-w-5xl rounded p-3">
          <div className="mb-3 flex flex-wrap items-center gap-2 text-xs text-dim">
            <label className="relative inline-flex items-center">
              <select
                value={mode}
                onChange={(e) => applyMode(e.target.value)}
                className="appearance-none rounded border border-border bg-bg/80 py-1.5 pl-3 pr-8 text-xs text-text outline-none transition hover:border-dim focus:border-accent focus:ring-2 focus:ring-accent/15"
                title={t("mode.title")}
              >
                {CHAT_MODE_PRESETS.map((preset) => (
                  <option key={preset.id} value={preset.id}>
                    {t(preset.labelKey)}
                  </option>
                ))}
              </select>
              <ChevronDown size={14} className="pointer-events-none absolute right-2 text-dim" />
            </label>

            <span className="hidden rounded bg-surface px-1.5 py-0.5 text-[11px] text-dim md:inline-flex">
              {t(currentMode.descriptionKey)}
            </span>

            <label className="relative inline-flex items-center">
              <select
                value={model}
                onChange={(e) =>
                  saveSettings({ model: e.target.value }).catch((err) => console.error("Failed to change model:", err))
                }
                className="appearance-none rounded border border-border bg-bg/80 py-1.5 pl-3 pr-8 text-xs text-text outline-none transition hover:border-dim focus:border-accent focus:ring-2 focus:ring-accent/15"
              >
                {MODEL_OPTIONS.map((option) => (
                  <option key={option.id} value={option.id}>
                    {option.name}
                  </option>
                ))}
              </select>
              <ChevronDown size={14} className="pointer-events-none absolute right-2 text-dim" />
            </label>

            {canThink && (
              <button
                onClick={() => {
                  const next = !thinkingEnabled;
                  saveSettings({ thinkingEnabled: next }).catch((err) =>
                    console.error("Failed to change thinking mode:", err)
                  );
                }}
                className={`metric-pill transition ${thinkingEnabled ? "border-accent/40 text-accent" : ""}`}
                title={t("chat.thinkingToggle")}
              >
                <Brain size={13} />
                {t("chat.thinkingToggle")}
              </button>
            )}

            {canThink && thinkingEnabled && (
              <label className="relative inline-flex items-center">
                <select
                  value={reasoningDisplay}
                  onChange={(e) =>
                    saveSettings({ reasoningDisplay: e.target.value as any }).catch((err) =>
                      console.error("Failed to change reasoning display:", err)
                    )
                  }
                  className="appearance-none rounded border border-border bg-bg/80 py-1.5 pl-3 pr-7 text-xs text-text outline-none transition hover:border-dim focus:border-accent focus:ring-2 focus:ring-accent/15"
                >
                  <option value="show">{t("chat.reasoningShow")}</option>
                  <option value="collapse">{t("chat.reasoningCollapse")}</option>
                  <option value="hide">{t("chat.reasoningHide")}</option>
                </select>
                <ChevronDown size={13} className="pointer-events-none absolute right-2 text-dim" />
              </label>
            )}

            <span className="flex-1" />
            <span>{session.messages.length} {t("sessions.title")}</span>
          </div>

          <div className="flex items-end gap-2 rounded border border-border bg-bg/80 p-2 transition focus-within:border-accent focus-within:ring-2 focus-within:ring-accent/15">
            <textarea
              ref={textareaRef}
              value={input}
              onChange={handleInput}
              onKeyDown={handleKeyDown}
              placeholder={hasInitialPrompt ? t("chat.rpPlaceholder") : t("chat.placeholder")}
              className="max-h-56 min-h-[48px] flex-1 resize-none bg-transparent px-2 py-2 text-sm leading-6 text-text outline-none placeholder:text-dim"
              rows={1}
              disabled={session.streaming}
            />
            {session.streaming ? (
              <button
                onClick={() => activeSessionId && abortMessage(activeSessionId)}
                className="motion-lift mb-0.5 flex h-10 w-10 items-center justify-center rounded bg-red/15 text-red transition hover:bg-red/25"
                title="Abort"
              >
                <Square size={16} />
              </button>
            ) : (
              <button
                onClick={handleSend}
                disabled={!input.trim()}
                className="motion-lift mb-0.5 flex h-10 w-10 items-center justify-center rounded bg-accent text-bg transition hover:bg-accent-alt disabled:cursor-not-allowed disabled:opacity-30"
                title="Send"
              >
                <Send size={17} />
              </button>
            )}
          </div>
        </div>
      </footer>
    </div>
  );
}
