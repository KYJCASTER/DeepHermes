import { Plus, Trash2, MessageSquare } from "lucide-react";
import { useSessionStore } from "../../stores/sessionStore";
import { useI18n } from "../../stores/i18nStore";

export default function Sidebar() {
  const sessions = useSessionStore((s) => s.sessions);
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const setActiveSession = useSessionStore((s) => s.setActiveSession);
  const createSession = useSessionStore((s) => s.createSession);
  const deleteSession = useSessionStore((s) => s.deleteSession);
  const { t } = useI18n();

  return (
    <aside className="soft-panel flex w-64 shrink-0 flex-col border-r border-border bg-surface/88">
      <div className="flex items-center justify-between border-b border-border px-3 py-3">
        <span className="text-xs font-semibold uppercase tracking-wide text-dim">
          {t("sessions.title")}
        </span>
        <button
          onClick={() => createSession(t("sessions.new"))}
          className="flex h-7 w-7 items-center justify-center rounded text-dim transition hover:bg-bg hover:text-accent"
          title={t("sidebar.newSession")}
        >
          <Plus size={14} />
        </button>
      </div>
      <div className="flex-1 overflow-y-auto">
        {sessions.map((s) => (
          <div
            key={s.id}
            onClick={() => setActiveSession(s.id)}
          className={`motion-lift group mx-2 mt-1 flex cursor-pointer items-center gap-2 rounded px-2.5 py-2 text-sm transition ${
              s.id === activeSessionId
                ? "bg-panel text-text shadow-sm"
                : "text-dim hover:bg-panel/80 hover:text-text"
            }`}
          >
            <MessageSquare size={14} className={s.id === activeSessionId ? "shrink-0 text-accent" : "shrink-0"} />
            <span className="truncate flex-1">{s.name}</span>
            <button
              onClick={(e) => { e.stopPropagation(); deleteSession(s.id); }}
              className="rounded p-1 text-dim opacity-0 transition hover:bg-red/10 hover:text-red group-hover:opacity-100"
              title={t("sidebar.delete")}
            >
              <Trash2 size={12} />
            </button>
          </div>
        ))}
      </div>
      <div className="border-t border-border px-3 py-2 text-xs text-dim">
        {t("sessions.count").replace("{count}", String(sessions.length))}
      </div>
    </aside>
  );
}
