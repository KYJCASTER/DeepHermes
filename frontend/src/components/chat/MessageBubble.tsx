import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { useState } from "react";
import { Bot, Brain, Check, ChevronDown, ChevronRight, GitBranch, Pencil, RotateCcw, TerminalSquare, Trash2, User, X } from "lucide-react";
import { useSettingsStore } from "../../stores/settingsStore";
import { useI18n } from "../../stores/i18nStore";

interface Props {
  role: string;
  content: string;
  reasoningContent?: string;
  isStreaming?: boolean;
  disabled?: boolean;
  onEdit?: (content: string) => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  onRegenerate?: () => Promise<void> | void;
  onBranch?: () => Promise<void> | void;
}

export default function MessageBubble({
  role,
  content,
  reasoningContent = "",
  isStreaming,
  disabled,
  onEdit,
  onDelete,
  onRegenerate,
  onBranch,
}: Props) {
  const isUser = role === "user";
  const isSystem = role === "system" || role === "tool";
  const reasoningDisplay = useSettingsStore((s) => s.reasoningDisplay);
  const { t } = useI18n();
  const [expanded, setExpanded] = useState(reasoningDisplay === "show");
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(content);
  const hasReasoning = !isUser && reasoningContent.trim().length > 0 && reasoningDisplay !== "hide";

  const saveEdit = async () => {
    const next = draft.trim();
    if (!next || next === content) {
      setEditing(false);
      setDraft(content);
      return;
    }
    await onEdit?.(next);
    setEditing(false);
  };

  if (isSystem) {
    return (
      <div className="message-bubble my-3 flex gap-3">
        <div className="mt-1 flex h-7 w-7 shrink-0 items-center justify-center rounded bg-panel text-dim">
          <TerminalSquare size={14} />
        </div>
        <div className="message-card flex-1 overflow-x-auto rounded px-3 py-2 text-xs text-dim">
          <pre className="system-pre whitespace-pre-wrap font-mono text-xs">{content}</pre>
        </div>
      </div>
    );
  }

  return (
    <div className={`message-bubble my-4 flex gap-3 sm:my-5 ${isUser ? "justify-end" : "justify-start"}`}>
      {!isUser && (
        <div className="ds-mark mt-1 hidden h-8 w-8 shrink-0 items-center justify-center rounded bg-accent/14 sm:flex">
          <Bot size={15} className="text-accent" />
        </div>
      )}

      <div className={`max-w-[92%] sm:max-w-[82%] ${isUser ? "order-first" : ""}`}>
        <div className={`mb-1 flex items-center gap-2 text-[11px] uppercase tracking-wide text-dim ${isUser ? "justify-end" : ""}`}>
          <span>{isUser ? "You" : "DeepHermes"}</span>
          {isStreaming && <span className="agent-running h-1.5 w-1.5 rounded-full bg-accent" />}
          {!disabled && !editing && (
            <div className={`flex items-center gap-1 normal-case tracking-normal ${isUser ? "order-first" : ""}`}>
              {onEdit && (
                <button onClick={() => setEditing(true)} className="message-action" title={t("chat.editMessage")}>
                  <Pencil size={12} />
                </button>
              )}
              {onRegenerate && (
                <button onClick={() => onRegenerate()} className="message-action" title={t("chat.regenerateMessage")}>
                  <RotateCcw size={12} />
                </button>
              )}
              {onBranch && (
                <button onClick={() => onBranch()} className="message-action" title={t("chat.branchMessage")}>
                  <GitBranch size={12} />
                </button>
              )}
              {onDelete && (
                <button
                  onClick={() => {
                    if (!window.confirm(t("confirm.deleteMessage"))) return;
                    onDelete();
                  }}
                  className="message-action hover:text-red"
                  title={t("chat.deleteMessage")}
                >
                  <Trash2 size={12} />
                </button>
              )}
            </div>
          )}
        </div>
        <div
          className={`message-card rounded px-4 py-3 ${
            isUser ? "message-card-user text-text" : "text-text"
          } ${isStreaming ? "streaming-cursor" : ""}`}
        >
          {hasReasoning && (
            <div className="mb-3 rounded border border-accent/15 bg-accent/5 text-xs text-dim">
              <button
                onClick={() => setExpanded((v) => !v)}
                className="flex w-full items-center gap-2 px-3 py-2 text-left text-accent transition hover:bg-accent/10"
              >
                <Brain size={13} />
                <span className="font-medium">{t("chat.reasoning")}</span>
                <span className="flex-1" />
                {expanded ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
              </button>
              {expanded && (
                <pre className="system-pre max-h-64 overflow-y-auto whitespace-pre-wrap px-3 pb-3 pt-0 font-mono text-xs leading-5 text-dim">
                  {reasoningContent}
                </pre>
              )}
            </div>
          )}

          {editing ? (
            <div className="space-y-2">
              <textarea
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                className="min-h-28 w-full resize-y rounded border border-border bg-bg/80 px-3 py-2 text-sm leading-6 text-text outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/15"
                autoFocus
              />
              <div className="flex justify-end gap-2">
                <button
                  onClick={() => {
                    setEditing(false);
                    setDraft(content);
                  }}
                  className="motion-lift inline-flex items-center gap-1 rounded border border-border px-2 py-1 text-xs text-dim hover:text-text"
                >
                  <X size={12} />
                  {t("settings.cancel")}
                </button>
                <button
                  onClick={saveEdit}
                  className="motion-lift inline-flex items-center gap-1 rounded bg-accent px-2 py-1 text-xs font-semibold text-bg hover:bg-accent-alt"
                >
                  <Check size={12} />
                  {t("settings.save")}
                </button>
              </div>
            </div>
          ) : isUser ? (
            <p className="whitespace-pre-wrap text-sm leading-6">{content}</p>
          ) : (
            <div className="markdown-body">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {content || (isStreaming ? t("chat.answering") : "")}
              </ReactMarkdown>
            </div>
          )}
        </div>
      </div>

      {isUser && (
        <div className="mt-1 hidden h-8 w-8 shrink-0 items-center justify-center rounded bg-green/16 sm:flex">
          <User size={15} className="text-green" />
        </div>
      )}
    </div>
  );
}
