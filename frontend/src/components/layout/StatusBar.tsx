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

  const color = activeSession?.status !== "idle" && activeSession?.status
    ? "text-yellow"
    : "text-dim";

  return (
    <div className="soft-panel flex h-7 shrink-0 items-center border-t border-border bg-surface/88 px-4 text-xs text-dim">
      <span className={color}>{label}</span>
      <span className="flex-1" />
      <span className="mr-4">{t("settings.model")}: {modelLabel(model)}</span>
      {activeSession && <span>{t("sessions.title")}: {activeSession.messages.length}</span>}
    </div>
  );
}
