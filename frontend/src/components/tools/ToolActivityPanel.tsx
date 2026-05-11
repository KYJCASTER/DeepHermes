import { CheckCircle, Clock3, Download, Network, RotateCcw, ShieldAlert, TerminalSquare, Trash2, X } from "lucide-react";
import { useI18n } from "../../stores/i18nStore";
import { ToolActivity, useToolActivityStore } from "../../stores/toolActivityStore";
import { RollbackToolChange } from "../../lib/wails";

function formatArgs(value: string) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value || "{}";
  }
}

function riskIcon(risk: string) {
  if (risk === "shell") return TerminalSquare;
  if (risk === "network") return Network;
  if (risk === "write" || risk === "unknown") return ShieldAlert;
  return CheckCircle;
}

function timeLabel(value?: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(date);
}

function statusClass(item: ToolActivity) {
  if (item.status === "failed") return "text-red";
  if (item.status === "running") return "text-accent";
  return "text-green";
}

export default function ToolActivityPanel() {
  const items = useToolActivityStore((s) => s.items);
  const togglePanel = useToolActivityStore((s) => s.togglePanel);
  const clear = useToolActivityStore((s) => s.clear);
  const exportAuditLog = useToolActivityStore((s) => s.exportAuditLog);
  const markRolledBack = useToolActivityStore((s) => s.markRolledBack);
  const { t } = useI18n();

  const downloadAuditLog = () => {
    const content = exportAuditLog();
    const stamp = new Date().toISOString().replace(/[:.]/g, "-");
    const blob = new Blob(["\ufeff", content], { type: "text/tab-separated-values;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `deephermes-tool-audit-${stamp}.tsv`;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  };

  const rollback = async (item: ToolActivity) => {
    if (!item.rollbackAvailable || item.rollbackDone) return;
    if (!window.confirm(t("tools.rollbackConfirm"))) return;
    try {
      const result = await RollbackToolChange(item.id);
      markRolledBack(item.id, result?.message || t("tools.rollbackDone"));
    } catch (e: any) {
      markRolledBack(item.id, e?.message || String(e));
    }
  };

  return (
    <aside className="soft-panel flex w-[22rem] shrink-0 flex-col border-l border-border bg-surface/90">
      <div className="flex items-center justify-between border-b border-border px-3 py-3">
        <div>
          <span className="text-xs font-semibold uppercase tracking-wide text-dim">{t("tools.activity")}</span>
          <p className="mt-0.5 text-[11px] text-dim">{t("tools.activityDesc")}</p>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={downloadAuditLog}
            disabled={items.length === 0}
            className="motion-lift rounded p-1 text-dim transition hover:bg-panel hover:text-text disabled:cursor-not-allowed disabled:opacity-40"
            title={t("tools.exportAudit")}
          >
            <Download size={14} />
          </button>
          <button onClick={clear} className="motion-lift rounded p-1 text-dim transition hover:bg-panel hover:text-text" title={t("tools.clear")}>
            <Trash2 size={14} />
          </button>
          <button onClick={togglePanel} className="motion-lift rounded p-1 text-dim transition hover:bg-panel hover:text-text" title="Close">
            <X size={14} />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {items.length === 0 && (
          <div className="px-5 py-10 text-center text-xs leading-5 text-dim">{t("tools.activityEmpty")}</div>
        )}
        {items.map((item) => {
          const Icon = riskIcon(item.risk);
          return (
            <div key={item.id} className="border-b border-border/60 px-3 py-3">
              <div className="flex items-start gap-2">
                <div className={`mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded bg-panel ${statusClass(item)}`}>
                  <Icon size={14} className={item.status === "running" ? "agent-running" : ""} />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="truncate text-sm font-medium text-text">{item.toolName}</span>
                    <span className="rounded bg-bg px-1.5 py-0.5 text-[10px] text-dim">{item.risk}</span>
                  </div>
                  <div className="mt-1 flex items-center gap-2 text-[11px] text-dim">
                    <Clock3 size={11} />
                    <span>{timeLabel(item.startedAt)}</span>
                    <span className={statusClass(item)}>{t(`tools.status.${item.status}`)}</span>
                  </div>
                  {item.rollbackPath && (
                    <div className="mt-1 truncate text-[11px] text-dim">{item.rollbackPath}</div>
                  )}
                </div>
              </div>

              <details className="mt-2 rounded border border-border bg-bg/60">
                <summary className="cursor-pointer px-2 py-1.5 text-[11px] text-dim">{t("tools.arguments")}</summary>
                <pre className="system-pre max-h-36 overflow-auto border-t border-border px-2 py-2 text-[11px] leading-5 text-dim">
                  {formatArgs(item.arguments)}
                </pre>
              </details>

              {(item.error || item.content) && (
                <pre className={`system-pre mt-2 max-h-32 overflow-auto rounded bg-bg/60 px-2 py-2 text-[11px] leading-5 ${item.error ? "text-red" : "text-dim"}`}>
                  {item.error || item.content}
                </pre>
              )}

              {(item.rollbackAvailable || item.rollbackDone) && (
                <div className="mt-2 flex items-center justify-between gap-2 rounded border border-border bg-bg/55 px-2 py-2">
                  <span className={`min-w-0 truncate text-[11px] ${item.rollbackDone ? "text-green" : "text-dim"}`}>
                    {item.rollbackDone ? item.rollbackMessage || t("tools.rollbackDone") : t("tools.rollbackAvailable")}
                  </span>
                  {!item.rollbackDone && (
                    <button
                      onClick={() => rollback(item)}
                      className="motion-lift inline-flex shrink-0 items-center gap-1 rounded bg-accent px-2 py-1 text-[11px] font-semibold text-bg transition hover:bg-accent-alt"
                    >
                      <RotateCcw size={11} />
                      {t("tools.rollback")}
                    </button>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </aside>
  );
}
