import { Activity, Database, Gauge, MessageSquare } from "lucide-react";
import { useSessionStore } from "../../stores/sessionStore";
import { useSettingsStore } from "../../stores/settingsStore";
import { useI18n } from "../../stores/i18nStore";
import { modelLabel } from "../../lib/models";

function compactNumber(value: number) {
  return new Intl.NumberFormat(undefined, { notation: "compact", maximumFractionDigits: 1 }).format(value || 0);
}

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
  const cacheTotal = (activeSession?.usage.promptCacheHitTokens || 0) + (activeSession?.usage.promptCacheMissTokens || 0);
  const cacheRate = cacheTotal > 0 ? Math.round(((activeSession?.usage.promptCacheHitTokens || 0) / cacheTotal) * 100) : 0;

  return (
    <div className="rail-panel flex h-8 shrink-0 items-center gap-3 border-t border-border px-4 text-xs text-dim">
      <span className={`inline-flex items-center gap-1.5 ${active ? "text-yellow" : "text-green"}`}>
        <Activity size={12} />
        {label}
      </span>
      <span className="h-3 w-px bg-border" />
      <span>{modelLabel(model)}</span>
      <span className="flex-1" />
      {activeSession && (
        <>
          <span className="hidden items-center gap-1.5 sm:inline-flex">
            <Database size={12} />
            {compactNumber(activeSession.usage.totalTokens)} tok
          </span>
          <span className="hidden items-center gap-1.5 md:inline-flex">
            <Gauge size={12} />
            {cacheRate}% cache
          </span>
          <span className="inline-flex items-center gap-1.5">
            <MessageSquare size={12} />
            {activeSession.messages.length}
          </span>
        </>
      )}
    </div>
  );
}
