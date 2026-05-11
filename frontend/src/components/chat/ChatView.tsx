import { useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import {
  AtSign,
  BarChart3,
  BookOpen,
  Brain,
  ChevronDown,
  Clock3,
  Command,
  Database,
  FileInput,
  Gauge,
  Paperclip,
  ScrollText,
  Send,
  Sparkles,
  Square,
  X,
  Zap,
} from "lucide-react";
import { useSessionStore } from "../../stores/sessionStore";
import { useSettingsStore } from "../../stores/settingsStore";
import { useI18n } from "../../stores/i18nStore";
import { CHAT_TEMPLATES, chatTemplateByCommand, chatTemplateText, type ChatTemplateId } from "../../lib/chatTemplates";
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
import { ClipboardSetText, GetContextSummary, OCRImage, OCRImageFile, OnFileDrop, OnFileDropOff, ReadFileSnippet, SearchWorkspaceFiles, UpdateContextSummary } from "../../lib/wails";
import MessageBubble from "./MessageBubble";
import ThinkingBanner from "./ThinkingBanner";

const INPUT_HISTORY_KEY = "deephermes.inputHistory";
const MAX_INPUT_HISTORY = 80;
const MAX_FILE_SNIPPET_BYTES = 96 * 1024;
const MAX_BROWSER_FILE_BYTES = 96 * 1024;
const IMAGE_EXTENSIONS = new Set([".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp"]);

interface FileSearchResult {
  name: string;
  path: string;
  relativePath: string;
  size: number;
}

interface FileMention {
  query: string;
  start: number;
  end: number;
}

function compactNumber(value: number) {
  return new Intl.NumberFormat(undefined, { notation: "compact", maximumFractionDigits: 1 }).format(value || 0);
}

function cacheRate(hit: number, miss: number) {
  const total = hit + miss;
  return total > 0 ? Math.round((hit / total) * 100) : 0;
}

function formatBytes(value: number) {
  if (!value) return "0 B";
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${Math.round(value / 1024)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

function transcriptMarkdown(sessionName: string, messages: Array<{ role: string; content: string; reasoningContent?: string }>) {
  const lines = [`# ${sessionName}`, ""];
  for (const msg of messages) {
    lines.push(`## ${msg.role}`);
    if (msg.reasoningContent) {
      lines.push("", "> Reasoning omitted from export preview.");
    }
    lines.push("", msg.content || "", "");
  }
  return lines.join("\n").trim() + "\n";
}

function fileToBase64(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error || new Error("Failed to read image"));
    reader.onload = () => {
      const value = String(reader.result || "");
      resolve(value.includes(",") ? value.split(",")[1] : value);
    };
    reader.readAsDataURL(file);
  });
}

function isImagePath(path: string) {
  const ext = path.slice(path.lastIndexOf(".")).toLowerCase();
  return IMAGE_EXTENSIONS.has(ext);
}

function activeFileMention(value: string, cursor: number): FileMention | null {
  const before = value.slice(0, cursor);
  const match = before.match(/(^|\s)@([^\s@]{0,80})$/);
  if (!match) return null;
  const query = match[2] || "";
  const start = before.length - query.length - 1;
  return { query, start, end: cursor };
}

export default function ChatView() {
  const [input, setInput] = useState("");
  const [showThinking, setShowThinking] = useState(true);
  const [showUsage, setShowUsage] = useState(false);
  const [showSummaryEditor, setShowSummaryEditor] = useState(false);
  const [summaryDraft, setSummaryDraft] = useState("");
  const [showTemplates, setShowTemplates] = useState(false);
  const [dragActive, setDragActive] = useState(false);
  const [composerNotice, setComposerNotice] = useState("");
  const [inputHistory, setInputHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [fileMention, setFileMention] = useState<FileMention | null>(null);
  const [fileSuggestions, setFileSuggestions] = useState<FileSearchResult[]>([]);
  const [fileSuggestionIndex, setFileSuggestionIndex] = useState(0);
  const [fileSearchLoading, setFileSearchLoading] = useState(false);
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const sessions = useSessionStore((s) => s.sessions);
  const sendMessage = useSessionStore((s) => s.sendMessage);
  const abortMessage = useSessionStore((s) => s.abortMessage);
  const editMessage = useSessionStore((s) => s.editMessage);
  const deleteMessage = useSessionStore((s) => s.deleteMessage);
  const regenerateMessage = useSessionStore((s) => s.regenerateMessage);
  const branchSession = useSessionStore((s) => s.branchSession);
  const continueLastResponse = useSessionStore((s) => s.continueLastResponse);
  const model = useSettingsStore((s) => s.model);
  const mode = useSettingsStore((s) => s.mode);
  const maxTokens = useSettingsStore((s) => s.maxTokens);
  const thinkingEnabled = useSettingsStore((s) => s.thinkingEnabled);
  const reasoningDisplay = useSettingsStore((s) => s.reasoningDisplay);
  const initialPrompt = useSettingsStore((s) => s.initialPrompt);
  const roleCard = useSettingsStore((s) => s.roleCard);
  const worldBook = useSettingsStore((s) => s.worldBook);
  const ocrEnabled = useSettingsStore((s) => s.ocrEnabled);
  const hasInitialPrompt = useSettingsStore((s) => Boolean(s.initialPrompt.trim() || s.roleCard.trim() || s.worldBook.trim()));
  const saveSettings = useSettingsStore((s) => s.save);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const { t, lang } = useI18n();
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

  useEffect(() => {
    try {
      const stored = JSON.parse(localStorage.getItem(INPUT_HISTORY_KEY) || "[]");
      if (Array.isArray(stored)) {
        setInputHistory(stored.filter((item) => typeof item === "string"));
      }
    } catch {
      setInputHistory([]);
    }
  }, []);

  useEffect(() => {
    if (!fileMention) {
      setFileSuggestions([]);
      setFileSearchLoading(false);
      return;
    }
    let cancelled = false;
    setFileSearchLoading(true);
    const timer = window.setTimeout(() => {
      SearchWorkspaceFiles(fileMention.query, 12)
        .then((items: FileSearchResult[]) => {
          if (cancelled) return;
          setFileSuggestions(items || []);
          setFileSuggestionIndex(0);
        })
        .catch((err) => {
          if (!cancelled) {
            setFileSuggestions([]);
            setComposerNotice(err?.message || String(err));
          }
        })
        .finally(() => {
          if (!cancelled) setFileSearchLoading(false);
        });
    }, 140);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [fileMention?.query, fileMention?.start, fileMention?.end]);

  const resizeTextarea = () => {
    requestAnimationFrame(() => {
      if (!textareaRef.current) return;
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = Math.min(textareaRef.current.scrollHeight, 220) + "px";
    });
  };

  const setDraft = (value: string) => {
    setInput(value);
    setHistoryIndex(-1);
    resizeTextarea();
  };

  const appendDraft = (value: string, baseOverride?: string) => {
    setInput((current) => {
      const source = baseOverride ?? current;
      const prefix = source.trim() ? `${source.trimEnd()}\n\n` : "";
      return prefix + value.trim();
    });
    setHistoryIndex(-1);
    setFileMention(null);
    setFileSuggestions([]);
    resizeTextarea();
    textareaRef.current?.focus();
  };

  const rememberInput = (value: string) => {
    const trimmed = value.trim();
    if (!trimmed || trimmed.startsWith("/")) return;
    const next = [trimmed, ...inputHistory.filter((item) => item !== trimmed)].slice(0, MAX_INPUT_HISTORY);
    setInputHistory(next);
    localStorage.setItem(INPUT_HISTORY_KEY, JSON.stringify(next));
  };

  const clearInput = () => {
    setInput("");
    setHistoryIndex(-1);
    setFileMention(null);
    setFileSuggestions([]);
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
  };

  const sendPlainMessage = (text: string, remember = true) => {
    if (!activeSessionId || session?.streaming) return;
    if (remember) {
      rememberInput(text);
    }
    sendMessage(activeSessionId, text).catch((e) => console.error("Failed to send message:", e));
    clearInput();
  };

  const applyTemplate = (id: ChatTemplateId) => {
    const text = chatTemplateText(id, lang);
    if (!text && id !== "export") return;
    appendDraft(text);
    setShowTemplates(false);
    setComposerNotice(t("chat.templateInserted"));
  };

  const exportSessionToClipboard = async () => {
    if (!session) return;
    await ClipboardSetText(transcriptMarkdown(session.name, session.messages));
    setComposerNotice(t("chat.exportCopied"));
  };

  const runSlashCommand = (raw: string) => {
    const [command] = raw.trim().split(/\s+/, 1);
    const template = chatTemplateByCommand(command);
    if (!template) {
      setComposerNotice(t("chat.commandUnknown"));
      return true;
    }
    if (template.id === "export") {
      exportSessionToClipboard().catch((err) => setComposerNotice(err?.message || String(err)));
      clearInput();
      return true;
    }
    if (template.id === "summary") {
      sendPlainMessage(chatTemplateText("summary", lang));
      return true;
    }
    setDraft(chatTemplateText(template.id, lang));
    setShowTemplates(false);
    setComposerNotice(t("chat.templateInserted"));
    return true;
  };

  const handleSend = () => {
    const text = input.trim();
    if (!text || !activeSessionId || session?.streaming) return;
    if (text.startsWith("/")) {
      runSlashCommand(text);
      return;
    }
    sendPlainMessage(text);
  };

  const attachFilePaths = async (paths: string[], baseDraft?: string) => {
    if (!paths.length) return;
    const snippets: string[] = [];
    for (const path of paths.slice(0, 6)) {
      try {
        if (isImagePath(path)) {
          if (!ocrEnabled) {
            snippets.push(`[Image file: ${path}]\n${t("chat.ocrDisabled")}`);
            continue;
          }
          const result = await OCRImageFile(path);
          snippets.push(`[OCR: ${path}]\nProvider: ${result.provider} / ${result.model}\n\n${result.text}`);
          continue;
        }
        const snippet = await ReadFileSnippet(path, MAX_FILE_SNIPPET_BYTES);
        if (snippet.binary) {
          snippets.push(`[File skipped: ${snippet.name}]\nPath: ${snippet.path}\nReason: binary or unsupported text preview.`);
          continue;
        }
        snippets.push(
          [
            `[Attached file: ${snippet.name}]`,
            `Path: ${snippet.path}`,
            `Size: ${formatBytes(snippet.size)}${snippet.truncated ? " (truncated)" : ""}`,
            "",
            "```text",
            snippet.content,
            "```",
          ].join("\n")
        );
      } catch (err: any) {
        snippets.push(`[File failed: ${path}]\n${err?.message || String(err)}`);
      }
    }
    appendDraft(snippets.join("\n\n"), baseDraft);
    setComposerNotice(t("chat.fileAttached"));
  };

  const attachSuggestedFile = (file: FileSearchResult) => {
    const mention = fileMention;
    if (!mention) return;
    const before = input.slice(0, mention.start).trimEnd();
    const after = input.slice(mention.end).trimStart();
    const baseDraft = [before, after].filter(Boolean).join(" ");
    setFileMention(null);
    setFileSuggestions([]);
    setInput(baseDraft);
    attachFilePaths([file.path], baseDraft).catch((err) => setComposerNotice(err?.message || String(err)));
  };

  const attachBrowserFiles = async (files: File[]) => {
    const snippets: string[] = [];
    for (const file of files.slice(0, 6)) {
      if (file.type.startsWith("image/")) {
        if (!ocrEnabled) {
          snippets.push(
            `[Pasted image: ${file.name || "clipboard-image"}]\nType: ${file.type || "image"}\nSize: ${formatBytes(file.size)}\n${t("chat.ocrDisabled")}`
          );
          continue;
        }
        const result = await OCRImage({
          fileName: file.name || "clipboard-image",
          mimeType: file.type || "image/png",
          dataBase64: await fileToBase64(file),
        });
        snippets.push(`[OCR: ${file.name || "clipboard-image"}]\nProvider: ${result.provider} / ${result.model}\n\n${result.text}`);
        continue;
      }
      const text = await file.text();
      const truncated = text.length > MAX_BROWSER_FILE_BYTES;
      snippets.push(
        [
          `[Attached file: ${file.name}]`,
          `Size: ${formatBytes(file.size)}${truncated ? " (truncated)" : ""}`,
          "",
          "```text",
          truncated ? text.slice(0, MAX_BROWSER_FILE_BYTES) : text,
          "```",
        ].join("\n")
      );
    }
    appendDraft(snippets.join("\n\n"));
    setComposerNotice(files.some((file) => file.type.startsWith("image/")) && ocrEnabled ? t("chat.ocrComplete") : t("chat.fileAttached"));
  };

  useEffect(() => {
    OnFileDrop((_, __, paths) => {
      attachFilePaths(paths).catch((err) => setComposerNotice(err?.message || String(err)));
    }, true);
    return () => OnFileDropOff();
  }, [activeSessionId, inputHistory, lang, ocrEnabled]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (fileMention && (fileSuggestions.length > 0 || fileSearchLoading)) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setFileSuggestionIndex((index) => Math.min(index + 1, Math.max(0, fileSuggestions.length - 1)));
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setFileSuggestionIndex((index) => Math.max(0, index - 1));
        return;
      }
      if ((e.key === "Enter" || e.key === "Tab") && fileSuggestions[fileSuggestionIndex]) {
        e.preventDefault();
        attachSuggestedFile(fileSuggestions[fileSuggestionIndex]);
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        setFileMention(null);
        setFileSuggestions([]);
        return;
      }
    }
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
      return;
    }
    if (e.key === "ArrowUp" && !e.shiftKey && inputHistory.length && textareaRef.current?.selectionStart === 0) {
      e.preventDefault();
      const nextIndex = Math.min(historyIndex + 1, inputHistory.length - 1);
      setHistoryIndex(nextIndex);
      setInput(inputHistory[nextIndex]);
      resizeTextarea();
      return;
    }
    if (e.key === "ArrowDown" && !e.shiftKey && historyIndex >= 0) {
      e.preventDefault();
      const nextIndex = historyIndex - 1;
      setHistoryIndex(nextIndex);
      setInput(nextIndex >= 0 ? inputHistory[nextIndex] : "");
      resizeTextarea();
    }
  };

  const handleInput = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value;
    setInput(value);
    setHistoryIndex(-1);
    setFileMention(activeFileMention(value, e.target.selectionStart));
    const el = e.target;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 220) + "px";
  };

  const refreshFileMention = () => {
    const cursor = textareaRef.current?.selectionStart ?? input.length;
    setFileMention(activeFileMention(input, cursor));
  };

  const handlePaste = (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
    const files = Array.from(e.clipboardData.files || []);
    if (!files.length) return;
    e.preventDefault();
    attachBrowserFiles(files).catch((err) => setComposerNotice(err?.message || String(err)));
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragActive(false);
    const files = Array.from(e.dataTransfer.files || []).filter((file) => !(file as any).path);
    if (!files.length) return;
    attachBrowserFiles(files).catch((err) => setComposerNotice(err?.message || String(err)));
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
  const lastMessage = session.messages[session.messages.length - 1];
  const canContinueOutput = Boolean(session.lastRun?.truncated && !session.streaming && lastMessage?.role === "assistant");

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

  const continueOutput = () => {
    if (!activeSessionId) return;
    continueLastResponse(activeSessionId).catch((e) => console.error("Failed to continue:", e));
    setComposerNotice(t("chat.continueStarted"));
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
                  {(session.contextSummaryTokens ?? 0) > 0 && (
                    <div className="mt-2 border-t border-border pt-2">
                      <div className="flex items-center justify-between">
                        <span className="text-xs text-dim">{t("context.summary")} (~{session.contextSummaryTokens} tok)</span>
                        <button
                          onClick={async () => {
                            if (!activeSessionId) return;
                            const result = await GetContextSummary(activeSessionId);
                            setSummaryDraft(result.summary || "");
                            setShowSummaryEditor((v) => !v);
                          }}
                          className="text-xs text-accent transition hover:underline"
                        >
                          {showSummaryEditor ? t("context.hideSummary") : t("context.editSummary")}
                        </button>
                      </div>
                      {showSummaryEditor && (
                        <div className="mt-2">
                          <textarea
                            value={summaryDraft}
                            onChange={(e) => setSummaryDraft(e.target.value)}
                            className="min-h-[80px] w-full resize-y rounded border border-border bg-bg/80 px-2 py-1.5 text-xs leading-5 text-text outline-none transition focus:border-accent"
                          />
                          <div className="mt-1.5 flex justify-end gap-2">
                            <button
                              onClick={() => setShowSummaryEditor(false)}
                              className="rounded border border-border px-2 py-1 text-xs text-dim hover:text-text"
                            >
                              {t("settings.cancel")}
                            </button>
                            <button
                              onClick={async () => {
                                if (!activeSessionId) return;
                                await UpdateContextSummary({ sessionId: activeSessionId, summary: summaryDraft });
                                setShowSummaryEditor(false);
                              }}
                              className="rounded bg-accent px-2 py-1 text-xs font-semibold text-bg hover:bg-accent-alt"
                            >
                              {t("settings.save")}
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
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
                      {session.lastRun.finishReason && (
                        <div className={`usage-card ${session.lastRun.truncated ? "text-yellow" : ""}`}>
                          <span className="usage-card-label">{t("status.finishReason")}</span>
                          <strong>{session.lastRun.finishReason}</strong>
                        </div>
                      )}
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

      <div className="flex-1 overflow-y-auto px-3 py-4 sm:px-6 sm:py-6">
        <div className="mx-auto flex w-full max-w-5xl flex-col">
          {session.messages.length === 0 && (
            <div className="fade-up mx-auto mt-12 w-full max-w-3xl text-center text-dim sm:mt-16">
              <div className="ds-mark mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-xl bg-accent/10 text-accent">
                <Sparkles size={26} />
              </div>
              <h3 className="mb-2 text-xl font-semibold text-text sm:text-2xl">{t("chat.startConversation")}</h3>
              <p className="mx-auto max-w-lg text-sm leading-6">
                {hasInitialPrompt ? t("chat.startHintPrompt") : t("chat.startHint")}
              </p>
              <div className="mt-6 grid grid-cols-1 gap-2.5 sm:mt-8 sm:grid-cols-2">
                {quickPrompts.map((prompt) => (
                  <button
                    key={prompt}
                    onClick={() => {
                      setInput(prompt);
                      textareaRef.current?.focus();
                    }}
                    className="motion-lift group rounded-lg border border-border bg-surface/70 px-4 py-3.5 text-left text-sm text-text transition hover:border-accent/40 hover:bg-panel"
                  >
                    <span className="inline-block text-accent/60 transition group-hover:text-accent">&rarr;</span>{" "}
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
          {canContinueOutput && (
            <div className="continue-output-banner fade-up mx-auto mb-4 mt-1 flex w-full max-w-2xl flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <p className="text-sm font-semibold text-text">{t("chat.outputTruncated")}</p>
                <p className="mt-1 text-xs leading-5 text-dim">{t("chat.outputTruncatedHint")}</p>
              </div>
              <button
                onClick={continueOutput}
                className="motion-lift inline-flex shrink-0 items-center justify-center gap-2 rounded bg-accent px-3 py-2 text-xs font-semibold text-bg transition hover:bg-accent-alt"
              >
                <Sparkles size={14} />
                {t("chat.continueOutput")}
              </button>
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>
      </div>

      <footer className="shrink-0 px-3 pb-3 sm:px-6 sm:pb-5">
        <div
          className={`composer-shell mx-auto max-w-5xl rounded p-3 ${dragActive ? "composer-drop-active" : ""}`}
          onDragEnter={(e) => {
            e.preventDefault();
            setDragActive(true);
          }}
          onDragOver={(e) => e.preventDefault()}
          onDragLeave={() => setDragActive(false)}
          onDrop={handleDrop}
        >
          <div className="mb-3 flex flex-wrap items-center gap-2 text-xs text-dim">
            <div className="composer-menu-anchor">
              <button
                onClick={() => setShowTemplates((value) => !value)}
                className={`metric-pill transition ${showTemplates ? "border-accent/40 text-accent" : ""}`}
                title={t("chat.templates")}
              >
                <Command size={13} />
                {t("chat.templates")}
              </button>
              {showTemplates && (
                <div className="composer-template-popover">
                  {CHAT_TEMPLATES.map((template) => {
                    const Icon =
                      template.id === "character"
                        ? BookOpen
                        : template.id === "lore"
                          ? ScrollText
                          : template.id === "export"
                            ? FileInput
                            : Sparkles;
                    return (
                      <button
                        key={template.id}
                        onClick={() => {
                          if (template.id === "export") {
                            exportSessionToClipboard().catch((err) => setComposerNotice(err?.message || String(err)));
                            setShowTemplates(false);
                            return;
                          }
                          if (template.id === "summary") {
                            sendPlainMessage(chatTemplateText("summary", lang));
                            setShowTemplates(false);
                            return;
                          }
                          applyTemplate(template.id);
                        }}
                        className="composer-template-item"
                      >
                        <Icon size={14} />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-text">{t(template.labelKey)}</span>
                          <span className="block truncate text-[11px] text-dim">
                            {template.command} / {t(template.descriptionKey)}
                          </span>
                        </span>
                      </button>
                    );
                  })}
                </div>
              )}
            </div>

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
            <span className="hidden items-center gap-1 md:inline-flex">
              <Paperclip size={12} />
              {t("chat.dropHint")}
            </span>
            <span>{session.messages.length} {t("sessions.title")}</span>
          </div>

          {composerNotice && (
            <div className="mb-2 flex items-center justify-between gap-3 rounded border border-accent/20 bg-accent/10 px-3 py-2 text-xs text-text">
              <span>{composerNotice}</span>
              <button onClick={() => setComposerNotice("")} className="text-dim transition hover:text-text" title={t("settings.cancel")}>
                <X size={13} />
              </button>
            </div>
          )}

          {fileMention && (
            <div className="composer-file-suggestions">
              <div className="flex items-center gap-2 border-b border-border px-3 py-2 text-xs text-dim">
                <AtSign size={13} />
                <span>{t("chat.fileMention")}</span>
                <span className="ml-auto">{fileSearchLoading ? t("chat.searchingFiles") : `${fileSuggestions.length}`}</span>
              </div>
              {!fileSearchLoading && fileSuggestions.length === 0 && (
                <div className="px-3 py-3 text-xs text-dim">{t("chat.noFileMatches")}</div>
              )}
              {fileSuggestions.map((file, index) => (
                <button
                  key={file.path}
                  onMouseDown={(event) => {
                    event.preventDefault();
                    attachSuggestedFile(file);
                  }}
                  onMouseEnter={() => setFileSuggestionIndex(index)}
                  className={`composer-file-item ${index === fileSuggestionIndex ? "composer-file-item-active" : ""}`}
                >
                  <FileInput size={14} />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-text">{file.name}</span>
                    <span className="block truncate text-[11px] text-dim">{file.relativePath}</span>
                  </span>
                  <span className="shrink-0 text-[11px] text-dim">{formatBytes(file.size)}</span>
                </button>
              ))}
            </div>
          )}

          <div
            className="composer-drop-target flex items-end gap-2 rounded border border-border bg-bg/80 p-2 transition focus-within:border-accent focus-within:ring-2 focus-within:ring-accent/15"
            style={{ "--wails-drop-target": "drop" } as CSSProperties}
          >
            {dragActive && (
              <div className="composer-drop-overlay">
                <FileInput size={18} />
                <span>{t("chat.dropFiles")}</span>
              </div>
            )}
            <textarea
              ref={textareaRef}
              value={input}
              onChange={handleInput}
              onKeyDown={handleKeyDown}
              onKeyUp={refreshFileMention}
              onClick={refreshFileMention}
              onPaste={handlePaste}
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
