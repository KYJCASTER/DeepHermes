import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { User, Bot } from "lucide-react";

interface Props {
  role: string;
  content: string;
  isStreaming?: boolean;
}

export default function MessageBubble({ role, content, isStreaming }: Props) {
  const isUser = role === "user";
  const isSystem = role === "system" || role === "tool";

  if (isSystem) {
    return (
      <div className="message-bubble my-2 overflow-x-auto rounded border border-border bg-surface/70 px-3 py-2 text-xs text-dim">
        <pre className="system-pre whitespace-pre-wrap font-mono text-xs">{content}</pre>
      </div>
    );
  }

  return (
    <div className={`my-5 flex gap-3 ${isUser ? "justify-end" : "justify-start"}`}>
      {!isUser && (
        <div className="ds-mark mt-1 flex h-7 w-7 shrink-0 items-center justify-center rounded bg-accent/14">
          <Bot size={14} className="text-accent" />
        </div>
      )}
      <div
        className={`message-bubble max-w-[78%] rounded px-4 py-3 shadow-sm ${
          isUser
            ? "border border-accent/25 bg-accent/10 text-text"
            : "border border-border bg-surface/90 text-text"
        } ${isStreaming ? "streaming-cursor" : ""}`}
      >
        {isUser ? (
          <p className="whitespace-pre-wrap text-sm leading-6">{content}</p>
        ) : (
          <div className="markdown-body">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {content}
            </ReactMarkdown>
          </div>
        )}
      </div>
      {isUser && (
        <div className="mt-1 flex h-7 w-7 shrink-0 items-center justify-center rounded bg-green/16">
          <User size={14} className="text-green" />
        </div>
      )}
    </div>
  );
}
