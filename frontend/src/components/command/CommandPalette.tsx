import { Bot, Moon, PanelRight, Plus, Search, Settings, Sparkles, Sun, TerminalSquare, X, Square } from "lucide-react";
import { useMemo, useState } from "react";
import { useCoworkStore } from "../../stores/coworkStore";
import { useI18n } from "../../stores/i18nStore";
import { useSessionStore } from "../../stores/sessionStore";
import { useSettingsStore } from "../../stores/settingsStore";
import { useThemeStore } from "../../stores/themeStore";
import { useToolActivityStore } from "../../stores/toolActivityStore";

interface Props {
  open: boolean;
  onClose: () => void;
}

export default function CommandPalette({ open, onClose }: Props) {
  const { t } = useI18n();
  const [query, setQuery] = useState("");
  const createSession = useSessionStore((s) => s.createSession);
  const abortMessage = useSessionStore((s) => s.abortMessage);
  const sessions = useSessionStore((s) => s.sessions);
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const toggleSettings = useSettingsStore((s) => s.togglePanel);
  const toggleCowork = useCoworkStore((s) => s.togglePanel);
  const toggleTools = useToolActivityStore((s) => s.togglePanel);
  const theme = useThemeStore((s) => s.theme);
  const toggleTheme = useThemeStore((s) => s.toggleTheme);
  const activeSession = sessions.find((s) => s.id === activeSessionId);

  const commands = useMemo(() => [
    {
      id: "new-session",
      title: t("command.newSession"),
      hint: "Ctrl+N",
      icon: Plus,
      run: async () => {
        await createSession(t("sessions.new"));
      },
    },
    {
      id: "settings",
      title: t("command.settings"),
      hint: t("settings.title"),
      icon: Settings,
      run: () => toggleSettings(),
    },
    {
      id: "tools",
      title: t("command.tools"),
      hint: t("tools.activity"),
      icon: TerminalSquare,
      run: () => toggleTools(),
    },
    {
      id: "cowork",
      title: t("command.cowork"),
      hint: t("titlebar.cowork"),
      icon: PanelRight,
      run: () => toggleCowork(),
    },
    {
      id: "theme",
      title: t("command.theme"),
      hint: theme,
      icon: theme === "dark" ? Moon : theme === "anime" ? Sparkles : Sun,
      run: () => toggleTheme(),
    },
    {
      id: "stop",
      title: t("command.stop"),
      hint: activeSession?.streaming ? t("status.streaming") : t("status.ready"),
      icon: Square,
      disabled: !activeSession?.streaming || !activeSessionId,
      run: async () => {
        if (activeSessionId) await abortMessage(activeSessionId);
      },
    },
  ], [abortMessage, activeSession?.streaming, activeSessionId, createSession, t, theme, toggleCowork, toggleSettings, toggleTheme, toggleTools]);

  if (!open) return null;

  const normalized = query.trim().toLowerCase();
  const visible = normalized
    ? commands.filter((command) => `${command.title} ${command.hint} ${command.id}`.toLowerCase().includes(normalized))
    : commands;

  const runCommand = async (command: typeof commands[number]) => {
    if (command.disabled) return;
    await command.run();
    setQuery("");
    onClose();
  };

  return (
    <div className="fixed inset-0 z-[70] bg-black/35 px-4 pt-[12vh] backdrop-blur-sm" onMouseDown={onClose}>
      <div
        className="fade-up panel-card mx-auto w-full max-w-xl overflow-hidden rounded border border-border shadow-2xl"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-3 border-b border-border px-4 py-3">
          <Search size={16} className="text-dim" />
          <input
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Escape") onClose();
              if (e.key === "Enter" && visible[0]) {
                e.preventDefault();
                runCommand(visible[0]);
              }
            }}
            placeholder={t("command.placeholder")}
            className="min-w-0 flex-1 bg-transparent text-sm text-text outline-none placeholder:text-dim"
          />
          <button onClick={onClose} className="icon-button h-7 w-7" title="Close">
            <X size={14} />
          </button>
        </div>

        <div className="max-h-[24rem] overflow-y-auto p-2">
          {visible.map((command) => (
            <button
              key={command.id}
              onClick={() => runCommand(command)}
              disabled={command.disabled}
              className="motion-lift flex w-full items-center gap-3 rounded px-3 py-2.5 text-left transition hover:bg-panel disabled:cursor-not-allowed disabled:opacity-45"
            >
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded bg-accent/10 text-accent">
                <command.icon size={15} />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium text-text">{command.title}</span>
                <span className="block truncate text-xs text-dim">{command.hint}</span>
              </span>
            </button>
          ))}
          {visible.length === 0 && (
            <div className="px-4 py-8 text-center text-xs text-dim">{t("command.empty")}</div>
          )}
        </div>
      </div>
    </div>
  );
}
