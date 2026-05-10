import { Activity, MessageSquare } from "lucide-react";
import { useSessionStore } from "../../stores/sessionStore";
import { useSettingsStore } from "../../stores/settingsStore";
import { useI18n } from "../../stores/i18nStore";
import { modelLabel } from "../../lib/models";

export default function StatusBar() {
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const sessions = useSessionStore((s) => s.sessions);
  const activeSession = sessions.find((s) => s.id === activeSessionId);
  const model = useSettingsStore((s) => s.model);
  const { t } = useI18n();

  const label = activeSession?.status === "thinking"
    ? t("status.thinking")
    : activeSession?.status === "streaming"
      ? t("status.streaming")
      : activeSession?.status === "executing"
        ? t("status.executing")
        : t("status.ready");

  const active = activeSession?.status !== "idle" && activeSession?.status;
  return (
    <div className="rail-panel status-bar flex h-8 shrink-0 items-center gap-3 border-t border-border px-4 text-xs text-dim">
      <span className={`inline-flex min-w-0 shrink-0 items-center gap-1.5 ${active ? "text-yellow" : "text-green"}`}>
        <Activity size={12} />
        <span className="truncate">{label}</span>
      </span>
      <span className="h-3 w-px shrink-0 bg-border" />
      <span className="min-w-0 max-w-[28ch] shrink truncate">{modelLabel(model)}</span>
      <span className="min-w-2 flex-1" />
      {activeSession && (
        <span className="inline-flex shrink-0 items-center gap-1.5">
          <MessageSquare size={12} />
          {activeSession.messages.length}
        </span>
      )}
    </div>
  );
}
