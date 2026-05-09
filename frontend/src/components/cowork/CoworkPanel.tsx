import { X, Bot, Play, Square } from "lucide-react";
import { useCoworkStore } from "../../stores/coworkStore";
import { useSessionStore } from "../../stores/sessionStore";
import { useI18n } from "../../stores/i18nStore";
import { useState } from "react";

export default function CoworkPanel() {
  const subAgents = useCoworkStore((s) => s.subAgents);
  const togglePanel = useCoworkStore((s) => s.togglePanel);
  const spawnSubAgent = useCoworkStore((s) => s.spawnSubAgent);
  const cancelSubAgent = useCoworkStore((s) => s.cancelSubAgent);
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const { t } = useI18n();

  const [task, setTask] = useState("");
  const [agentType, setAgentType] = useState("explore");

  const handleSpawn = () => {
    if (!task.trim() || !activeSessionId) return;
    spawnSubAgent(activeSessionId, task.slice(0, 30), agentType, task);
    setTask("");
  };

  const statusColor = (s: string) => {
    switch (s) {
      case "running": return "text-accent";
      case "done": return "text-green";
      case "failed": return "text-red";
      default: return "text-dim";
    }
  };

  return (
    <aside className="soft-panel flex w-80 shrink-0 flex-col border-l border-border bg-surface/88">
      <div className="flex items-center justify-between border-b border-border px-3 py-3">
        <span className="text-xs font-semibold text-dim uppercase">{t("cowork.title")}</span>
        <button onClick={togglePanel} className="motion-lift rounded p-1 text-dim transition hover:bg-panel hover:text-text" title="Close">
          <X size={14} />
        </button>
      </div>

      <div className="space-y-2 border-b border-border p-3">
        <select
          value={agentType}
          onChange={(e) => setAgentType(e.target.value)}
          className="w-full rounded border border-border bg-bg/80 px-2 py-2 text-xs text-text outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/15"
        >
          <option value="explore">{t("cowork.explore")}</option>
          <option value="implement">{t("cowork.implement")}</option>
          <option value="review">{t("cowork.review")}</option>
        </select>
        <textarea
          value={task}
          onChange={(e) => setTask(e.target.value)}
          placeholder={t("cowork.task")}
          className="w-full resize-none rounded border border-border bg-bg/80 px-2 py-2 text-xs text-text outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/15"
          rows={2}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              handleSpawn();
            }
          }}
        />
        <button
          onClick={handleSpawn}
          disabled={!task.trim()}
          className="motion-lift flex w-full items-center justify-center gap-1 rounded bg-accent px-2 py-2 text-xs font-semibold text-bg transition hover:bg-accent-alt disabled:opacity-30"
        >
          <Play size={12} /> {t("cowork.spawn")}
        </button>
      </div>

      <div className="flex-1 overflow-y-auto">
        {subAgents.length === 0 && (
          <div className="text-center text-dim text-xs mt-8 px-4">
            {t("cowork.empty")}
          </div>
        )}
        {subAgents.map((sa) => (
          <div key={sa.id} className="message-bubble border-b border-border/60 px-3 py-3">
            <div className="flex items-center gap-2">
              <Bot size={14} className={statusColor(sa.status) + (sa.status === "running" ? " agent-running" : "")} />
              <span className="text-sm text-text truncate flex-1">{sa.name}</span>
              {sa.status === "running" && (
                <button
                  onClick={() => cancelSubAgent(sa.id)}
                  className="rounded p-1 text-red transition hover:bg-red/10"
                >
                  <Square size={12} />
                </button>
              )}
            </div>
            <div className="flex gap-2 mt-1 text-xs">
              <span className={statusColor(sa.status)}>{sa.status}</span>
              <span className="text-dim">{sa.agentType}</span>
            </div>
            {sa.result && (
              <p className="mt-1 truncate text-xs text-dim">{sa.result}</p>
            )}
          </div>
        ))}
      </div>
    </aside>
  );
}
