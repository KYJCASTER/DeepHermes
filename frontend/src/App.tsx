import { useEffect, useState } from "react";
import { useSessionStore } from "./stores/sessionStore";
import { useSettingsStore } from "./stores/settingsStore";
import { useCoworkStore } from "./stores/coworkStore";
import { useToolActivityStore } from "./stores/toolActivityStore";
import { ApproveToolCall, EventsOn, RejectToolCall, WindowGetPosition, WindowGetSize, WindowSetPosition, WindowSetSize } from "./lib/wails";
import { Plus, ShieldCheck, Sparkles } from "lucide-react";
import TitleBar from "./components/layout/TitleBar";
import Sidebar from "./components/layout/Sidebar";
import StatusBar from "./components/layout/StatusBar";
import ChatView from "./components/chat/ChatView";
import FileBrowser from "./components/files/FileBrowser";
import CoworkPanel from "./components/cowork/CoworkPanel";
import CommandPalette from "./components/command/CommandPalette";
import ToolActivityPanel from "./components/tools/ToolActivityPanel";
import SettingsDialog from "./components/settings/SettingsDialog";
import WelcomeScreen from "./components/settings/WelcomeScreen";
import { useI18n } from "./stores/i18nStore";

type ToolApproval = {
  id: string;
  sessionId?: string;
  toolName: string;
  arguments: string;
  risk: string;
  mode: string;
  preview?: string;
};

function formatToolArguments(value: string) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value || "{}";
  }
}

export default function App() {
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const loadSessions = useSessionStore((s) => s.loadSessions);
  const appendToStream = useSessionStore((s) => s.appendToStream);
  const finishStream = useSessionStore((s) => s.finishStream);
  const setSessionStatus = useSessionStore((s) => s.setSessionStatus);
  const loadSettings = useSettingsStore((s) => s.load);
  const setAPIKeyStatus = useSettingsStore((s) => s.setAPIKeyStatus);
  const settingsOpen = useSettingsStore((s) => s.isOpen);
  const coworkOpen = useCoworkStore((s) => s.isOpen);
  const upsertSubAgent = useCoworkStore((s) => s.upsertSubAgent);
  const toolActivityOpen = useToolActivityStore((s) => s.isOpen);
  const addToolCall = useToolActivityStore((s) => s.addCall);
  const finishToolCall = useToolActivityStore((s) => s.finishCall);
  const apiKeyStatus = useSettingsStore((s) => s.apiKeyStatus);
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [toolApprovals, setToolApprovals] = useState<ToolApproval[]>([]);
  const [commandOpen, setCommandOpen] = useState(false);

  // Listen for events from Go backend. This must be registered before any
  // conditional render path so React keeps a stable hook order after setup.
  useEffect(() => {
    const parseEvent = (raw: unknown) => {
      if (typeof raw === "string") {
        return JSON.parse(raw);
      }
      return raw as any;
    };

    const unsubs: Array<() => void> = [];

    unsubs.push(EventsOn("stream:delta", (raw: unknown) => {
      try {
        const evt = parseEvent(raw);
        appendToStream(evt.sessionId, evt.data?.content || "", evt.data?.reasoningContent || "");
      } catch (e) {
        console.error("Failed to handle stream delta:", e);
      }
    }));

    unsubs.push(EventsOn("stream:done", (raw: unknown) => {
      try {
        const evt = parseEvent(raw);
        if (evt?.sessionId) {
          finishStream(evt.sessionId, evt.data?.metrics);
        }
      } catch (e) {
        console.error("Failed to handle stream done:", e);
      }
    }));

    unsubs.push(EventsOn("agent:status", (raw: unknown) => {
      try {
        const evt = parseEvent(raw);
        if (evt?.sessionId) {
          setSessionStatus(evt.sessionId, evt.data?.status || "idle");
        }
      } catch (e) {
        console.error("Failed to handle agent status:", e);
      }
    }));

    unsubs.push(EventsOn("error", (raw: unknown) => {
      try {
        const evt = parseEvent(raw);
        const msg = evt.data?.message || "Unknown error";
        appendToStream(evt.sessionId, `\n\n> **Error:** ${msg}`);
        finishStream(evt.sessionId);
      } catch (e) {
        console.error("Failed to handle backend error:", e);
      }
    }));

    unsubs.push(EventsOn("cowork:update", (raw: unknown) => {
      try {
        const evt = parseEvent(raw);
        upsertSubAgent({
          id: evt.subAgentId,
          name: evt.name,
          agentType: evt.type,
          status: evt.status,
          result: evt.result,
        });
      } catch (e) {
        console.error("Failed to handle cowork update:", e);
      }
    }));

    unsubs.push(EventsOn("context:compacted", (raw: unknown) => {
      try {
        const evt = parseEvent(raw);
        const { messagesBefore, messagesAfter, summaryTokens } = evt.data || {};
        console.info(
          `Context compacted: ${messagesBefore} → ${messagesAfter} messages, summary ~${summaryTokens} tokens`
        );
      } catch (e) {
        console.error("Failed to handle context compacted:", e);
      }
    }));

    unsubs.push(EventsOn("settings:updated", (data: any) => {
      if (data?.apiKeyStatus) {
        setAPIKeyStatus(data.apiKeyStatus);
      }
    }));

    unsubs.push(EventsOn("tool:approval", (payload: ToolApproval) => {
      if (!payload?.id) return;
      setToolApprovals((items) => {
        if (items.some((item) => item.id === payload.id)) return items;
        return [...items, payload];
      });
    }));

    unsubs.push(EventsOn("tool:call", (raw: unknown) => {
      try {
        const evt = parseEvent(raw);
        if (!evt.data?.id || !evt.data?.name) return;
        addToolCall({
          id: evt.data?.id,
          sessionId: evt.sessionId,
          toolName: evt.data?.name,
          arguments: evt.data?.arguments,
          risk: evt.data?.risk,
        });
      } catch (e) {
        console.error("Failed to handle tool call:", e);
      }
    }));

    unsubs.push(EventsOn("tool:result", (raw: unknown) => {
      try {
        const evt = parseEvent(raw);
        finishToolCall(evt.data || {});
      } catch (e) {
        console.error("Failed to handle tool result:", e);
      }
    }));

    return () => {
      unsubs.forEach((unsubscribe) => {
        try {
          unsubscribe();
        } catch (e) {
          console.error("Failed to unsubscribe event:", e);
        }
      });
    };
  }, [appendToStream, finishStream, setSessionStatus, setAPIKeyStatus, upsertSubAgent, addToolCall, finishToolCall]);

  const resolveToolApproval = async (id: string, approve: boolean) => {
    setToolApprovals((items) => items.filter((item) => item.id !== id));
    try {
      if (approve) {
        await ApproveToolCall(id);
      } else {
        await RejectToolCall(id);
      }
    } catch (e) {
      console.error("Failed to resolve tool approval:", e);
    }
  };

  useEffect(() => {
    async function init() {
      await loadSettings();
      await loadSessions();
      setLoading(false);
    }
    init();
  }, []);

  useEffect(() => {
    const onKeyDown = async (event: KeyboardEvent) => {
      const mod = event.ctrlKey || event.metaKey;
      const target = event.target as HTMLElement | null;
      const editing = Boolean(target?.closest("input, textarea, select, [contenteditable='true']"));
      if (mod && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setCommandOpen((open) => !open);
      }
      if (mod && event.key.toLowerCase() === "n" && !editing) {
        event.preventDefault();
        const store = useSessionStore.getState();
        await store.createSession(t("sessions.new"));
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [t]);

  useEffect(() => {
    if (loading || apiKeyStatus !== "configured") return;
    const raw = localStorage.getItem("deephermes.windowBounds");
    if (raw) {
      try {
        const bounds = JSON.parse(raw);
        if (bounds?.w >= 900 && bounds?.h >= 600) {
          WindowSetSize(bounds.w, bounds.h);
        }
        if (Number.isFinite(bounds?.x) && Number.isFinite(bounds?.y)) {
          WindowSetPosition(bounds.x, bounds.y);
        }
      } catch (e) {
        console.error("Failed to restore window bounds:", e);
      }
    }

    const saveBounds = async () => {
      try {
        const [size, position] = await Promise.all([WindowGetSize(), WindowGetPosition()]);
        localStorage.setItem(
          "deephermes.windowBounds",
          JSON.stringify({ w: size.w, h: size.h, x: position.x, y: position.y })
        );
      } catch (e) {
        console.error("Failed to save window bounds:", e);
      }
    };
    const timer = window.setInterval(saveBounds, 2500);
    window.addEventListener("beforeunload", saveBounds);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener("beforeunload", saveBounds);
      saveBounds();
    };
  }, [loading, apiKeyStatus]);

  if (loading) {
    return (
      <div className="welcome-shell flex h-screen items-center justify-center text-text">
        <div className="app-content fade-up flex flex-col items-center gap-4 text-sm text-dim">
          <div className="ds-mark flex h-12 w-12 items-center justify-center rounded-xl bg-accent/12 text-accent">
            <Sparkles size={22} className="agent-running" />
          </div>
          <span className="text-xs tracking-wide">{t("app.loading")}</span>
        </div>
      </div>
    );
  }

  if (apiKeyStatus !== "configured") {
    return <WelcomeScreen />;
  }

  return (
    <div className="app-shell flex h-screen flex-col bg-bg">
      <TitleBar />
      <div className="app-content flex flex-1 overflow-hidden bg-bg/78">
        <Sidebar />
        <main className="workspace-main flex min-w-0 flex-1 flex-col overflow-hidden">
          {activeSessionId ? (
            <ChatView />
          ) : (
            <div className="flex flex-1 items-center justify-center px-8 text-dim">
              <div className="fade-up panel-card w-full max-w-lg rounded border border-border px-8 py-9 text-center">
                <div className="ds-mark mx-auto mb-5 flex h-12 w-12 items-center justify-center rounded bg-accent/12 text-accent">
                  <Sparkles size={24} />
                </div>
                <h2 className="mb-2 text-2xl font-semibold text-text">DeepHermes</h2>
                <p className="text-sm">{t("sessions.emptyHint")}</p>
                <button
                  onClick={async () => {
                    const store = useSessionStore.getState();
                    await store.createSession(t("sessions.new"));
                  }}
                  className="motion-lift titlebar-no-drag mt-5 inline-flex items-center gap-2 rounded bg-accent px-4 py-2 text-sm font-semibold text-bg transition hover:bg-accent-alt disabled:opacity-40"
                >
                  <Plus size={16} />
                  {t("sessions.new")}
                </button>
              </div>
            </div>
          )}
          <StatusBar />
        </main>
        {coworkOpen && <CoworkPanel />}
        {toolActivityOpen && <ToolActivityPanel />}
        <FileBrowser />
      </div>
      {settingsOpen && <SettingsDialog />}
      <CommandPalette open={commandOpen} onClose={() => setCommandOpen(false)} />
      {toolApprovals.length > 0 && (
        <div className="fixed bottom-5 right-5 z-[60] flex max-h-[80vh] w-[min(420px,calc(100vw-2rem))] flex-col gap-3 overflow-y-auto">
          {toolApprovals.map((approval) => (
            <div key={approval.id} className="fade-up panel-card rounded border border-border p-4 shadow-xl">
              <div className="mb-3 flex items-start gap-3">
                <div className="ds-mark flex h-9 w-9 shrink-0 items-center justify-center rounded bg-accent/12 text-accent">
                  <ShieldCheck size={17} />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <h3 className="truncate text-sm font-semibold text-text">{t("tools.approvalTitle")}</h3>
                    <span className="rounded bg-surface px-2 py-0.5 text-[11px] text-dim">{approval.risk}</span>
                  </div>
                  <p className="mt-1 text-xs text-dim">{t("tools.approvalDesc").replace("{tool}", approval.toolName)}</p>
                </div>
              </div>
              <pre className="system-pre max-h-40 overflow-y-auto rounded border border-border bg-bg/75 p-3 text-xs leading-5 text-dim">
                {formatToolArguments(approval.arguments)}
              </pre>
              {approval.preview && (
                <div className="mt-2 rounded border border-yellow/25 bg-yellow/8">
                  <div className="border-b border-yellow/20 px-3 py-2 text-xs font-semibold text-text">
                    {t("tools.diffPreview")}
                  </div>
                  <pre className="system-pre max-h-48 overflow-y-auto p-3 text-xs leading-5 text-text">
                    {approval.preview}
                  </pre>
                </div>
              )}
              <div className="mt-3 flex justify-end gap-2">
                <button
                  onClick={() => resolveToolApproval(approval.id, false)}
                  className="motion-lift rounded border border-border px-3 py-1.5 text-xs text-dim transition hover:border-red/50 hover:text-red"
                >
                  {t("tools.reject")}
                </button>
                <button
                  onClick={() => resolveToolApproval(approval.id, true)}
                  className="motion-lift rounded bg-accent px-3 py-1.5 text-xs font-semibold text-bg transition hover:bg-accent-alt"
                >
                  {t("tools.approve")}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
