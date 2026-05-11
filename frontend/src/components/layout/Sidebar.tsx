import { Clock3, MessageSquare, Plus, Search, Trash2, X } from "lucide-react";
import type { MouseEvent as ReactMouseEvent } from "react";
import { useMemo, useState } from "react";
import { useSessionStore } from "../../stores/sessionStore";
import { useI18n } from "../../stores/i18nStore";
import { useLayoutStore } from "../../stores/layoutStore";

function formatDate(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }).format(date);
}

function compactNumber(value: number) {
  return new Intl.NumberFormat(undefined, { notation: "compact", maximumFractionDigits: 1 }).format(value || 0);
}

export default function Sidebar() {
  const [query, setQuery] = useState("");
  const sessions = useSessionStore((s) => s.sessions);
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const setActiveSession = useSessionStore((s) => s.setActiveSession);
  const createSession = useSessionStore((s) => s.createSession);
  const deleteSession = useSessionStore((s) => s.deleteSession);
  const sidebarWidth = useLayoutStore((s) => s.sidebarWidth);
  const setSidebarWidth = useLayoutStore((s) => s.setSidebarWidth);
  const { t } = useI18n();
  const filteredSessions = useMemo(() => {
    const keyword = query.trim().toLowerCase();
    if (!keyword) return sessions;
    return sessions.filter((session) => {
      const haystack = [
        session.name,
        session.model,
        ...session.messages.slice(-12).map((message) => message.content),
      ].join("\n").toLowerCase();
      return haystack.includes(keyword);
    });
  }, [query, sessions]);

  const startResize = (event: ReactMouseEvent) => {
    event.preventDefault();
    const onMove = (moveEvent: MouseEvent) => setSidebarWidth(moveEvent.clientX);
    const onUp = () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
  };

  return (
    <aside
      className="rail-panel relative flex shrink-0 flex-col border-r border-border"
      style={{ width: sidebarWidth }}
    >
      <div className="border-b border-border px-4 py-4">
        <div className="mb-3 flex items-center justify-between">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-dim">{t("sessions.title")}</p>
            <p className="mt-1 text-xs text-dim">{t("sessions.count").replace("{count}", String(sessions.length))}</p>
          </div>
          <button
            onClick={() => createSession(t("sessions.new"))}
            className="icon-button h-8 w-8"
            title={t("sidebar.newSession")}
          >
            <Plus size={15} />
          </button>
        </div>
        <button
          onClick={() => createSession(t("sessions.new"))}
          className="motion-lift flex w-full items-center justify-center gap-2 rounded bg-accent px-3 py-2 text-sm font-semibold text-bg transition hover:bg-accent-alt"
        >
          <Plus size={15} />
          {t("sessions.new")}
        </button>
        <div className="mt-3 flex items-center gap-2 rounded border border-border bg-bg/70 px-2 py-1.5">
          <Search size={13} className="shrink-0 text-dim" />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("sessions.search")}
            className="min-w-0 flex-1 bg-transparent text-xs text-text outline-none placeholder:text-dim"
          />
          {query && (
            <button onClick={() => setQuery("")} className="text-dim transition hover:text-text" title={t("settings.cancel")}>
              <X size={13} />
            </button>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-2 py-2">
        {filteredSessions.length === 0 && (
          <div className="flex flex-col items-center px-3 py-10 text-center text-xs leading-5 text-dim">
            <MessageSquare size={28} className="mb-3 text-border" />
            <span>{query ? t("sessions.searchEmpty") : t("sessions.searchEmpty")}</span>
          </div>
        )}
        {filteredSessions.map((s) => {
          const active = s.id === activeSessionId;
          return (
            <div
              key={s.id}
              onClick={() => setActiveSession(s.id)}
              className={`nav-card group mb-2 cursor-pointer rounded px-3 py-3 text-sm ${
                active ? "nav-card-active text-text" : "text-dim"
              }`}
            >
              <div className="flex items-start gap-2">
                <div className={`mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded ${active ? "bg-accent/14 text-accent" : "bg-panel text-dim"}`}>
                  <MessageSquare size={14} />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="truncate font-medium">{s.name}</span>
                    {s.status !== "idle" && <span className="agent-running h-1.5 w-1.5 rounded-full bg-accent" />}
                  </div>
                  <div className="mt-1 flex items-center gap-2 text-[11px] text-dim">
                    <Clock3 size={11} />
                    <span className="truncate">{formatDate(s.updatedAt || s.createdAt)}</span>
                  </div>
                  <div className="mt-2 flex flex-wrap gap-1.5 text-[11px] text-dim">
                    <span className="rounded bg-surface px-1.5 py-0.5">{s.msgCount || s.messages.length} msg</span>
                    <span className="rounded bg-surface px-1.5 py-0.5">{compactNumber(s.usage.totalTokens)} tok</span>
                  </div>
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    if (!window.confirm(t("confirm.deleteSession"))) return;
                    deleteSession(s.id);
                  }}
                  className="rounded p-1 text-dim opacity-0 transition hover:bg-red/10 hover:text-red group-hover:opacity-100"
                  title={t("sidebar.delete")}
                >
                  <Trash2 size={12} />
                </button>
              </div>
            </div>
          );
        })}
      </div>

      <div
        onMouseDown={startResize}
        className="resize-handle absolute right-[-3px] top-0 z-10 h-full w-1.5 cursor-col-resize transition"
      />
    </aside>
  );
}
