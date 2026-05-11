import { Bot, Languages, Maximize, Minus, Moon, PanelRight, Plus, Settings, Sparkles, Square, Sun, TerminalSquare, X } from "lucide-react";
import { useState } from "react";
import { useCoworkStore } from "../../stores/coworkStore";
import { LANG_LABELS, useI18n } from "../../stores/i18nStore";
import { useSessionStore } from "../../stores/sessionStore";
import { useSettingsStore } from "../../stores/settingsStore";
import { useToolActivityStore } from "../../stores/toolActivityStore";
import { useThemeStore } from "../../stores/themeStore";
import { HideMainWindow, Quit, WindowMaximise, WindowMinimise, WindowUnmaximise } from "../../lib/wails";

export default function TitleBar() {
  const sessions = useSessionStore((s) => s.sessions);
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const activeSession = sessions.find((s) => s.id === activeSessionId);
  const toggleSettings = useSettingsStore((s) => s.togglePanel);
  const minimizeToTray = useSettingsStore((s) => s.minimizeToTray);
  const toggleCowork = useCoworkStore((s) => s.togglePanel);
  const toggleTools = useToolActivityStore((s) => s.togglePanel);
  const toolItems = useToolActivityStore((s) => s.items);
  const theme = useThemeStore((s) => s.theme);
  const toggleTheme = useThemeStore((s) => s.toggleTheme);
  const { t, lang, toggleLang } = useI18n();
  const [maximized, setMaximized] = useState(false);
  const themeTitle = theme === "dark" ? t("theme.light") : theme === "light" ? t("theme.anime") : t("theme.dark");
  const ThemeIcon = theme === "dark" ? Moon : theme === "anime" ? Sparkles : Sun;

  const handleNewSession = async () => {
    const store = useSessionStore.getState();
    await store.createSession(t("sessions.new"));
  };

  const winMaximize = () => {
    if (maximized) {
      WindowUnmaximise();
    } else {
      WindowMaximise();
    }
    setMaximized(!maximized);
  };

  const closeWindow = () => {
    if (minimizeToTray) {
      HideMainWindow();
      return;
    }
    Quit();
  };

  return (
    <div className="rail-panel titlebar-drag flex h-12 shrink-0 select-none items-center border-b border-border">
      <div className="flex min-w-0 items-center gap-3 px-3">
        <div className="ds-mark flex h-7 w-7 items-center justify-center rounded bg-accent text-bg shadow-sm">
          <Bot size={14} />
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold tracking-wide text-text">{t("app.title")}</span>
            <span className="rounded bg-panel px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-dim">
              {t("app.subtitle")}
            </span>
          </div>
        </div>
      </div>

      <div className="mx-4 min-w-0 flex-1">
        <div className="truncate text-center text-xs text-dim">
          {activeSession ? activeSession.name : t("sessions.emptyHint")}
        </div>
      </div>

      <div className="titlebar-no-drag flex items-center gap-1 pr-1">
        <button onClick={handleNewSession} className="icon-button h-8 w-8" title={t("sidebar.newSession")}>
          <Plus size={15} />
        </button>
        <button
          onClick={toggleLang}
          className="icon-button h-8 min-w-8 gap-1 px-2 text-xs"
          title={LANG_LABELS[lang]}
        >
          <Languages size={14} />
          <span>{LANG_LABELS[lang]}</span>
        </button>
        <button
          onClick={toggleTheme}
          className="theme-toggle-track relative flex h-8 w-[4.75rem] items-center rounded-full px-1 transition"
          title={themeTitle}
        >
          <span
            className={`flex h-6 w-6 items-center justify-center rounded-full bg-accent text-bg shadow-sm transition-transform duration-300 ${
              theme === "anime" ? "translate-x-[1.55rem]" : theme === "dark" ? "translate-x-[2.75rem]" : "translate-x-0"
            }`}
          >
            <ThemeIcon size={13} />
          </span>
        </button>
        <button onClick={toggleCowork} className="icon-button h-8 w-8" title={t("titlebar.cowork")}>
          <PanelRight size={15} />
        </button>
        <button onClick={toggleTools} className="icon-button relative h-8 w-8" title={t("titlebar.tools")}>
          <TerminalSquare size={15} />
          {toolItems.some((item) => item.status === "running") && (
            <span className="agent-running absolute right-1.5 top-1.5 h-1.5 w-1.5 rounded-full bg-accent" />
          )}
        </button>
        <button onClick={toggleSettings} className="icon-button h-8 w-8" title={t("titlebar.settings")}>
          <Settings size={15} />
        </button>
        <button onClick={WindowMinimise} className="icon-button h-8 w-8" title="Minimize">
          <Minus size={14} />
        </button>
        <button onClick={winMaximize} className="icon-button h-8 w-8" title="Maximize">
          {maximized ? <Maximize size={14} /> : <Square size={12} />}
        </button>
        <button
          onClick={closeWindow}
          className="icon-button h-8 w-8 hover:bg-red/10 hover:text-red"
          title={minimizeToTray ? t("titlebar.hideToTray") : "Close"}
        >
          <X size={14} />
        </button>
      </div>
    </div>
  );
}
