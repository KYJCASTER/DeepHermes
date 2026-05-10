import { useEffect, useState } from "react";
import { useSessionStore } from "./stores/sessionStore";
import { useSettingsStore } from "./stores/settingsStore";
import { useCoworkStore } from "./stores/coworkStore";
import { EventsOn, WindowGetPosition, WindowGetSize, WindowSetPosition, WindowSetSize } from "./lib/wails";
import { Plus, Sparkles } from "lucide-react";
import TitleBar from "./components/layout/TitleBar";
import Sidebar from "./components/layout/Sidebar";
import StatusBar from "./components/layout/StatusBar";
import ChatView from "./components/chat/ChatView";
import FileBrowser from "./components/files/FileBrowser";
import CoworkPanel from "./components/cowork/CoworkPanel";
import SettingsDialog from "./components/settings/SettingsDialog";
import WelcomeScreen from "./components/settings/WelcomeScreen";
import { useI18n } from "./stores/i18nStore";

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
  const apiKeyStatus = useSettingsStore((s) => s.apiKeyStatus);
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);

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
        appendToStream(evt.sessionId, `\n\nRequest failed: ${evt.data?.message || "Unknown error"}`);
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

    unsubs.push(EventsOn("settings:updated", (data: any) => {
      if (data?.apiKeyStatus) {
        setAPIKeyStatus(data.apiKeyStatus);
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
  }, [appendToStream, finishStream, setSessionStatus, setAPIKeyStatus, upsertSubAgent]);

  useEffect(() => {
    async function init() {
      await loadSettings();
      await loadSessions();
      setLoading(false);
    }
    init();
  }, []);

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
        <div className="app-content flex items-center gap-3 text-sm text-dim">
          <Sparkles size={16} className="agent-running text-accent" />
          DeepHermes starting...
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
        <FileBrowser />
      </div>
      {settingsOpen && <SettingsDialog />}
    </div>
  );
}
