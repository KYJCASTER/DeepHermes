import { useState, useRef, useEffect } from "react";
import { Brain, ChevronDown, Send, Square } from "lucide-react";
import { useSessionStore } from "../../stores/sessionStore";
import { useSettingsStore } from "../../stores/settingsStore";
import { useI18n } from "../../stores/i18nStore";
import { MODEL_OPTIONS, supportsThinking } from "../../lib/models";
import MessageBubble from "./MessageBubble";
import ThinkingBanner from "./ThinkingBanner";

export default function ChatView() {
  const [input, setInput] = useState("");
  const [showThinking, setShowThinking] = useState(true);
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const sessions = useSessionStore((s) => s.sessions);
  const sendMessage = useSessionStore((s) => s.sendMessage);
  const abortMessage = useSessionStore((s) => s.abortMessage);
  const model = useSettingsStore((s) => s.model);
  const thinkingEnabled = useSettingsStore((s) => s.thinkingEnabled);
  const saveSettings = useSettingsStore((s) => s.save);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const { t } = useI18n();
  const session = sessions.find((s) => s.id === activeSessionId);
  const canThink = supportsThinking(model);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [session?.messages]);

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
    el.style.height = Math.min(el.scrollHeight, 200) + "px";
  };

  if (!session) {
    return (
      <div className="flex-1 flex items-center justify-center text-dim">
        Select or create a session
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {canThink && thinkingEnabled && session.status === "thinking" && showThinking && (
        <ThinkingBanner onDismiss={() => setShowThinking(false)} />
      )}

      <div className="flex-1 overflow-y-auto px-6 py-5">
        {session.messages.length === 0 && (
          <div className="fade-up mx-auto mt-20 max-w-md text-center text-dim">
            <div className="ds-mark mx-auto mb-4 flex h-11 w-11 items-center justify-center rounded bg-surface/90 text-accent shadow-sm">
              <Brain size={20} />
            </div>
            <h3 className="mb-2 text-lg font-medium text-text">{t("chat.startConversation")}</h3>
            <p className="text-sm leading-6">{t("chat.startHint")}</p>
          </div>
        )}
        {session.messages.map((msg, i) => (
          <MessageBubble
            key={i}
            role={msg.role}
            content={msg.content}
            isStreaming={
              i === session.messages.length - 1 &&
              msg.role === "assistant" &&
              session.streaming
            }
          />
        ))}
        <div ref={messagesEndRef} />
      </div>

      <div className="soft-panel border-t border-border bg-surface/88 px-5 py-4">
        <div className="mb-3 flex flex-wrap items-center gap-3 text-xs text-dim">
          <label className="relative inline-flex items-center">
            <select
              value={model}
              onChange={(e) => saveSettings({ model: e.target.value }).catch((err) => console.error("Failed to change model:", err))}
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
            <label className="inline-flex items-center gap-2">
              <span>{t("chat.thinkingToggle")}</span>
              <button
                onClick={() => {
                  const next = !thinkingEnabled;
                  saveSettings({ thinkingEnabled: next }).catch((err) => console.error("Failed to change thinking mode:", err));
                }}
                className={`relative h-5 w-9 rounded-full transition ${
                  thinkingEnabled ? "bg-accent" : "bg-border"
                }`}
                title={t("chat.thinkingToggle")}
              >
                <div
                  className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition ${
                    thinkingEnabled ? "left-4" : "left-0.5"
                  }`}
                />
              </button>
            </label>
          )}
        </div>

        <div className="chat-composer flex items-end gap-2 rounded border border-border bg-bg/80 p-2 transition focus-within:border-accent focus-within:ring-2 focus-within:ring-accent/15">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={handleInput}
            onKeyDown={handleKeyDown}
            placeholder={t("chat.placeholder")}
            className="max-h-48 min-h-[42px] flex-1 resize-none bg-transparent px-2 py-2 text-sm leading-6 text-text outline-none placeholder:text-dim"
            rows={1}
            disabled={session.streaming}
          />
          {session.streaming ? (
            <button
              onClick={() => activeSessionId && abortMessage(activeSessionId)}
              className="motion-lift mb-0.5 flex h-9 w-9 items-center justify-center rounded bg-red/15 text-red transition hover:bg-red/25"
              title="Abort"
            >
              <Square size={16} />
            </button>
          ) : (
            <button
              onClick={handleSend}
              disabled={!input.trim()}
              className="motion-lift mb-0.5 flex h-9 w-9 items-center justify-center rounded bg-accent text-bg transition hover:bg-accent-alt disabled:cursor-not-allowed disabled:opacity-30"
              title="Send"
            >
              <Send size={16} />
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
