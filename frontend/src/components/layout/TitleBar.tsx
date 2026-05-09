import { Bot, Languages, Maximize, Minus, Moon, PanelRight, Plus, Settings, Square, Sun, X } from "lucide-react";
import { useSessionStore } from "../../stores/sessionStore";
import { useSettingsStore } from "../../stores/settingsStore";
import { useCoworkStore } from "../../stores/coworkStore";
import { useI18n, LANG_LABELS } from "../../stores/i18nStore";
import { useThemeStore } from "../../stores/themeStore";
import { WindowMinimise, WindowMaximise, WindowUnmaximise, Quit } from "../../lib/wails";
import { useState } from "react";

export default function TitleBar() {
  const sessions = useSessionStore((s) => s.sessions);
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const setActiveSession = useSessionStore((s) => s.setActiveSession);
  const toggleSettings = useSettingsStore((s) => s.togglePanel);
  const toggleCowork = useCoworkStore((s) => s.togglePanel);
  const theme = useThemeStore((s) => s.theme);
  const toggleTheme = useThemeStore((s) => s.toggleTheme);
  const { t, lang, toggleLang } = useI18n();
  const [maximized, setMaximized] = useState(false);

  const handleNewSession = async () => {
    const store = useSessionStore.getState();
    await store.createSession(t("sessions.new"));
  };

  const winMinimize = () => WindowMinimise();
  const winMaximize = () => {
    if (maximized) {
      WindowUnmaximise();
    } else {
      WindowMaximise();
    }
    setMaximized(!maximized);
  };
  const winClose = () => Quit();

  return (
    <div className="soft-panel titlebar-drag flex h-12 shrink-0 select-none items-center border-b border-border bg-surface/90">
      <div className="flex items-center gap-2 px-3">
        <div className="ds-mark flex h-7 w-7 items-center justify-center rounded bg-accent text-bg shadow-sm">
          <Bot size={14} />
        </div>
        <span className="text-sm font-semibold tracking-wide text-text">{t("app.title")}</span>
        <span className="text-xs text-dim">{t("app.subtitle")}</span>
      </div>

      <div className="titlebar-no-drag ml-2 flex flex-1 items-center gap-1 overflow-x-auto">
        {sessions.map((s) => (
          <button
            key={s.id}
            onClick={() => setActiveSession(s.id)}
            className={`motion-lift rounded px-3 py-1.5 text-xs transition whitespace-nowrap ${
              s.id === activeSessionId
                ? "bg-panel text-text shadow-sm"
                : "text-dim hover:bg-panel/80 hover:text-text"
            }`}
          >
            {s.name}
            {s.status === "thinking" && (
              <span className="ml-1 text-yellow">●</span>
            )}
            {s.status === "streaming" && (
              <span className="ml-1 text-accent agent-running">●</span>
            )}
          </button>
        ))}
        <button
          onClick={handleNewSession}
          className="motion-lift titlebar-no-drag flex h-7 w-7 items-center justify-center rounded text-dim transition hover:bg-panel hover:text-text"
          title={t("sidebar.newSession")}
        >
          <Plus size={15} />
        </button>
      </div>

      <div className="titlebar-no-drag ml-2 flex items-center gap-1 pr-1">
        <button
          onClick={toggleLang}
          className="motion-lift flex h-8 min-w-8 items-center justify-center gap-1 rounded px-2 text-xs text-dim transition hover:bg-panel hover:text-accent"
          title={LANG_LABELS[lang]}
        >
          <Languages size={14} />
          <span>{LANG_LABELS[lang]}</span>
        </button>
        <button
          onClick={toggleTheme}
          className="theme-toggle-track relative flex h-8 w-14 items-center rounded-full px-1 transition"
          title={theme === "dark" ? t("theme.light") : t("theme.dark")}
        >
          <span
            className={`flex h-6 w-6 items-center justify-center rounded-full bg-accent text-bg shadow-sm transition-transform duration-300 ${
              theme === "dark" ? "translate-x-6" : "translate-x-0"
            }`}
          >
            {theme === "dark" ? <Moon size={13} /> : <Sun size={13} />}
          </span>
        </button>
        <button
          onClick={toggleCowork}
          className="motion-lift flex h-8 w-8 items-center justify-center rounded text-dim transition hover:bg-panel hover:text-accent"
          title={t("titlebar.cowork")}
        >
          <PanelRight size={15} />
        </button>
        <button
          onClick={toggleSettings}
          className="motion-lift flex h-8 w-8 items-center justify-center rounded text-dim transition hover:bg-panel hover:text-text"
          title={t("titlebar.settings")}
        >
          <Settings size={15} />
        </button>

        <button onClick={winMinimize} className="motion-lift flex h-8 w-8 items-center justify-center rounded text-dim transition hover:bg-panel hover:text-text" title="Minimize">
          <Minus size={14} />
        </button>
        <button onClick={winMaximize} className="motion-lift flex h-8 w-8 items-center justify-center rounded text-dim transition hover:bg-panel hover:text-text" title="Maximize">
          {maximized ? <Maximize size={14} /> : <Square size={12} />}
        </button>
        <button onClick={winClose} className="motion-lift flex h-8 w-8 items-center justify-center rounded text-dim transition hover:bg-red/10 hover:text-red" title="Close">
          <X size={14} />
        </button>
      </div>
    </div>
  );
}
