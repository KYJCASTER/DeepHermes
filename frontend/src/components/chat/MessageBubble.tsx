import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { useState } from "react";
import { Bot, Brain, ChevronDown, ChevronRight, TerminalSquare, User } from "lucide-react";
import { useSettingsStore } from "../../stores/settingsStore";
import { useI18n } from "../../stores/i18nStore";

interface Props {
  role: string;
  content: string;
  reasoningContent?: string;
  isStreaming?: boolean;
}

export default function MessageBubble({ role, content, reasoningContent = "", isStreaming }: Props) {
  const isUser = role === "user";
  const isSystem = role === "system" || role === "tool";
  const reasoningDisplay = useSettingsStore((s) => s.reasoningDisplay);
  const { t } = useI18n();
  const [expanded, setExpanded] = useState(reasoningDisplay === "show");
  const hasReasoning = !isUser && reasoningContent.trim().length > 0 && reasoningDisplay !== "hide";

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
    <div className={`message-bubble my-5 flex gap-3 ${isUser ? "justify-end" : "justify-start"}`}>
      {!isUser && (
        <div className="ds-mark mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded bg-accent/14">
          <Bot size={15} className="text-accent" />
        </div>
      )}

      <div className={`max-w-[82%] ${isUser ? "order-first" : ""}`}>
        <div className={`mb-1 flex items-center gap-2 text-[11px] uppercase tracking-wide text-dim ${isUser ? "justify-end" : ""}`}>
          <span>{isUser ? "You" : "DeepHermes"}</span>
          {isStreaming && <span className="agent-running h-1.5 w-1.5 rounded-full bg-accent" />}
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

          {isUser ? (
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
        <div className="mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded bg-green/16">
          <User size={15} className="text-green" />
        </div>
      )}
    </div>
  );
}
