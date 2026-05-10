import { AlertTriangle, CheckCircle, ClipboardList, Download, FileText, HardDrive, Key, Link, Minimize2, Moon, SlidersHorizontal, Sparkles, Sun, Upload, Wand2, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useSettingsStore } from "../../stores/settingsStore";
import { useI18n } from "../../stores/i18nStore";
import { useThemeStore } from "../../stores/themeStore";
import { CHAT_MODE_PRESETS, chatModePreset, MODEL_OPTIONS, formatCny, formatTokenLimit, modelProfile, supportsThinking } from "../../lib/models";
import { INITIAL_PROMPT_PRESETS } from "../../lib/promptPresets";
import { ExportSettings, GetDiagnostics, ImportSettings } from "../../lib/wails";

const INITIAL_PROMPT_LIMIT = 60000;

export default function SettingsDialog() {
  const settings = useSettingsStore();
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const { t, lang } = useI18n();
  const [apiKey, setApiKey] = useState("");
  const [baseUrl, setBaseUrl] = useState(settings.baseUrl);
  const [initialPrompt, setInitialPrompt] = useState(settings.initialPrompt);
  const [roleCard, setRoleCard] = useState(settings.roleCard);
  const [worldBook, setWorldBook] = useState(settings.worldBook);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");
  const [diagnostics, setDiagnostics] = useState<any>(null);
  const currentProfile = modelProfile(settings.model);

  useEffect(() => {
    GetDiagnostics().then(setDiagnostics).catch((e) => console.error("Failed to load diagnostics:", e));
  }, []);

  const savePartial = (partial: Parameters<typeof settings.save>[0]) => {
    settings.save(partial).catch((e: any) => setError(e?.message || String(e)));
  };

  const applyMode = (mode: string) => {
    const preset = chatModePreset(mode);
    savePartial({
      mode: preset.id,
      model: preset.model,
      maxTokens: preset.maxTokens,
      temperature: preset.temperature,
      thinkingEnabled: preset.thinkingEnabled,
      reasoningDisplay: preset.reasoningDisplay,
    });
  };

  const handleSave = async () => {
    try {
      setError("");
      if (apiKey.trim()) {
        await settings.setAPIKey(apiKey);
        setApiKey("");
      }
      await settings.save({
        model: settings.model,
        mode: settings.mode,
        portable: settings.portable,
        minimizeToTray: settings.minimizeToTray,
        maxTokens: settings.maxTokens,
        temperature: settings.temperature,
        baseUrl,
        thinkingEnabled: settings.thinkingEnabled,
        reasoningDisplay: settings.reasoningDisplay,
        autoCowork: settings.autoCowork,
        initialPrompt,
        roleCard,
        worldBook,
      });
      GetDiagnostics().then(setDiagnostics).catch(() => undefined);
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
      if (!apiKey.trim()) {
        settings.togglePanel();
      }
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  const handleExportSettings = async () => {
    try {
      setError("");
      const path = await ExportSettings();
      if (path) {
        setSaved(true);
        setTimeout(() => setSaved(false), 2500);
        GetDiagnostics().then(setDiagnostics).catch(() => undefined);
      }
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  const handleImportSettings = async () => {
    try {
      setError("");
      await ImportSettings();
      await settings.load();
      GetDiagnostics().then(setDiagnostics).catch(() => undefined);
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4 backdrop-blur-sm" onClick={settings.togglePanel}>
      <div
        className="fade-up panel-card max-h-[88vh] w-[760px] overflow-y-auto rounded border border-border"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-border px-5 py-4">
          <div className="flex items-center gap-3">
            <div className="ds-mark flex h-8 w-8 items-center justify-center rounded bg-panel text-accent">
              <SlidersHorizontal size={17} />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-text">{t("settings.title")}</h2>
              <p className="text-xs text-dim">{t("settings.deepseekProfile")}</p>
            </div>
          </div>
          <button onClick={settings.togglePanel} className="icon-button h-8 w-8" title="Close">
            <X size={18} />
          </button>
        </div>

        <div className="space-y-5 p-5">
          <div>
            <label className="mb-2 flex items-center gap-2 text-sm text-text">
              <Key size={14} />
              {t("settings.apiKey")}
              {settings.apiKeyStatus === "configured" && (
                <span className="flex items-center gap-1 text-xs text-green">
                  <CheckCircle size={12} /> {t("settings.configured")}
                </span>
              )}
            </label>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => {
                setApiKey(e.target.value);
                setError("");
              }}
              placeholder={settings.apiKeyStatus === "configured" ? "************" : "sk-..."}
              className="w-full rounded border border-border bg-bg/80 px-3 py-2 text-sm text-text outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/15"
            />
            <p className="mt-1 text-xs text-dim">
              {saved
                ? t("settings.savedToConfig")
                : settings.apiKeyStatus === "configured"
                  ? t("settings.keyStored")
                  : t("settings.enterKey")}
            </p>
          </div>

          <div>
            <label className="mb-2 block text-sm text-text">{t("settings.appearance")}</label>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-4">
              {[
                { id: "light" as const, label: t("theme.lightMode"), icon: Sun },
                { id: "anime" as const, label: t("theme.animeMode"), icon: Sparkles },
                { id: "dark" as const, label: t("theme.darkMode"), icon: Moon },
              ].map((option) => (
                <button
                  key={option.id}
                  onClick={() => setTheme(option.id)}
                  className={`motion-lift flex items-center gap-2 rounded border px-3 py-2 text-sm transition ${
                    theme === option.id
                      ? "border-accent bg-accent/10 text-text"
                      : "border-border bg-bg/80 text-dim hover:border-dim hover:text-text"
                  }`}
                >
                  <option.icon size={15} />
                  {option.label}
                </button>
              ))}
            </div>
          </div>

          <div className="rounded border border-border bg-bg/80 p-3">
            <div className="mb-3 flex items-start justify-between gap-3">
              <div>
                <label className="flex items-center gap-2 text-sm font-medium text-text">
                  <HardDrive size={14} />
                  {t("settings.desktop")}
                </label>
                <p className="mt-1 text-xs leading-5 text-dim">{t("settings.desktopDesc")}</p>
              </div>
              {diagnostics && (
                <span className="shrink-0 rounded bg-surface px-2 py-1 text-[11px] text-dim">
                  v{diagnostics.version}
                </span>
              )}
            </div>

            <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
              <button
                onClick={handleExportSettings}
                className="motion-lift inline-flex items-center justify-center gap-2 rounded border border-border bg-surface px-3 py-2 text-xs text-text transition hover:border-dim"
              >
                <Download size={13} />
                {t("settings.export")}
              </button>
              <button
                onClick={handleImportSettings}
                className="motion-lift inline-flex items-center justify-center gap-2 rounded border border-border bg-surface px-3 py-2 text-xs text-text transition hover:border-dim"
              >
                <Upload size={13} />
                {t("settings.import")}
              </button>
              <button
                onClick={() => savePartial({ portable: !settings.portable })}
                className={`motion-lift inline-flex items-center justify-center gap-2 rounded border px-3 py-2 text-xs transition ${
                  settings.portable ? "border-accent bg-accent/10 text-text" : "border-border bg-surface text-dim hover:border-dim hover:text-text"
                }`}
              >
                <HardDrive size={13} />
                {settings.portable ? t("settings.portableOn") : t("settings.portableOff")}
              </button>
              <button
                onClick={() => savePartial({ minimizeToTray: !settings.minimizeToTray })}
                className={`motion-lift inline-flex items-center justify-center gap-2 rounded border px-3 py-2 text-xs transition ${
                  settings.minimizeToTray ? "border-accent bg-accent/10 text-text" : "border-border bg-surface text-dim hover:border-dim hover:text-text"
                }`}
              >
                <Minimize2 size={13} />
                {settings.minimizeToTray ? t("settings.trayOn") : t("settings.trayOff")}
              </button>
            </div>

            {diagnostics && (
              <div className="mt-3 rounded border border-border bg-surface/70 p-3">
                <div className="mb-2 flex items-center gap-2 text-xs font-semibold text-text">
                  <ClipboardList size={13} />
                  {t("settings.diagnostics")}
                </div>
                <div className="grid grid-cols-1 gap-1 text-[11px] leading-5 text-dim sm:grid-cols-2">
                  <span>{t("settings.configPath")}: <b className="text-text">{diagnostics.configPath}</b></span>
                  <span>{t("settings.dataDir")}: <b className="text-text">{diagnostics.dataDir}</b></span>
                  <span>{t("settings.sessionsDir")}: <b className="text-text">{diagnostics.sessionsDir}</b></span>
                  <span>{t("settings.windowBehavior")}: <b className="text-text">{diagnostics.minimizeToTray ? t("settings.trayOn") : t("settings.trayOff")}</b></span>
                  <span>{t("settings.buildInfo")}: <b className="text-text">{diagnostics.buildCommit} / {diagnostics.buildDate}</b></span>
                  <span>{t("settings.runtime")}: <b className="text-text">{diagnostics.platform}/{diagnostics.arch} {diagnostics.goVersion}</b></span>
                  <span>{t("settings.sessionCount")}: <b className="text-text">{diagnostics.sessionCount}</b></span>
                </div>
                <div className="mt-3 max-h-28 overflow-y-auto rounded bg-bg/70 p-2 font-mono text-[11px] leading-5 text-dim">
                  {(diagnostics.recentLogs || []).length === 0 ? (
                    <span>{t("settings.noLogs")}</span>
                  ) : (
                    diagnostics.recentLogs.map((log: any, index: number) => (
                      <div key={`${log.time}-${index}`}>
                        [{log.level}] {log.time} {log.message}
                      </div>
                    ))
                  )}
                </div>
              </div>
            )}
          </div>

          <div>
            <label className="mb-2 block text-sm text-text">{t("mode.title")}</label>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {CHAT_MODE_PRESETS.map((preset) => (
                <button
                  key={preset.id}
                  onClick={() => applyMode(preset.id)}
                  className={`motion-lift rounded border p-3 text-left transition ${
                    settings.mode === preset.id
                      ? "border-accent bg-accent/10 text-text"
                      : "border-border bg-bg/80 text-dim hover:border-dim hover:text-text"
                  }`}
                >
                  <span className="block text-sm font-semibold">{t(preset.labelKey)}</span>
                  <span className="mt-1 block text-xs leading-5 text-dim">{t(preset.descriptionKey)}</span>
                  <span className="mt-2 block text-[11px] text-dim">
                    {modelProfile(preset.model).name} / {formatTokenLimit(preset.maxTokens)} / {preset.temperature.toFixed(1)}
                  </span>
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="mb-2 block text-sm text-text">{t("settings.model")}</label>
            <div className="grid grid-cols-2 gap-2">
              {MODEL_OPTIONS.map((option) => (
                <button
                  key={option.id}
                  onClick={() =>
                    savePartial({
                      model: option.id,
                      maxTokens: Math.min(settings.maxTokens, option.maxOutput),
                      thinkingEnabled: option.thinking ? settings.thinkingEnabled : false,
                    })
                  }
                  className={`motion-lift rounded border p-3 text-left transition ${
                    settings.model === option.id
                      ? "border-accent bg-accent/10 text-text"
                      : "border-border bg-bg/80 text-dim hover:border-dim hover:text-text"
                  }`}
                >
                  <div className="mb-1 flex items-center justify-between gap-2">
                    <span className="text-sm font-medium">{option.name}</span>
                    <span className={`rounded px-1.5 py-0.5 text-[11px] ${option.legacy ? "bg-yellow/12 text-yellow" : "bg-panel text-dim"}`}>
                      {option.badge}
                    </span>
                  </div>
                  <p className="text-xs leading-5">{option.description}</p>
                  <div className="mt-2 flex flex-wrap gap-1.5 text-[11px] text-dim">
                    <span className="rounded bg-surface px-1.5 py-0.5">{formatTokenLimit(option.contextWindow)} ctx</span>
                    <span className="rounded bg-surface px-1.5 py-0.5">{formatTokenLimit(option.maxOutput)} out</span>
                    <span className="rounded bg-surface px-1.5 py-0.5">{formatCny(option.priceCny.output)}/M out</span>
                  </div>
                  {option.legacy && option.deprecatesOn && (
                    <div className="mt-2 flex items-center gap-1.5 text-[11px] text-yellow">
                      <AlertTriangle size={12} />
                      {t("settings.legacyModel").replace("{date}", option.deprecatesOn)}
                    </div>
                  )}
                </button>
              ))}
            </div>
          </div>

          <div className="rounded border border-border bg-bg/80 p-3">
            <div className="mb-3 flex items-start justify-between gap-3">
              <div>
                <span className="text-sm font-medium text-text">{t("settings.deepseekProfile")}</span>
                <p className="mt-1 text-xs leading-5 text-dim">
                  {t("settings.deepseekProfileDesc")
                    .replace("{context}", formatTokenLimit(currentProfile.contextWindow))
                    .replace("{output}", formatTokenLimit(currentProfile.maxOutput))}
                </p>
              </div>
              <button
                onClick={() =>
                  savePartial({
                    maxTokens: currentProfile.recommendedMaxTokens,
                    temperature: currentProfile.recommendedTemperature,
                    thinkingEnabled: currentProfile.thinking ? settings.thinkingEnabled : false,
                  })
                }
                className="motion-lift inline-flex shrink-0 items-center gap-1.5 rounded bg-accent px-3 py-1.5 text-xs font-semibold text-bg transition hover:bg-accent-alt"
              >
                <Wand2 size={13} />
                {t("settings.applyRecommended")}
              </button>
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs">
              <div className="rounded bg-surface px-3 py-2">
                <p className="text-dim">{t("settings.recommendedMaxTokens")}</p>
                <p className="mt-1 font-medium text-text">{formatTokenLimit(currentProfile.recommendedMaxTokens)}</p>
              </div>
              <div className="rounded bg-surface px-3 py-2">
                <p className="text-dim">{t("settings.recommendedTemperature")}</p>
                <p className="mt-1 font-medium text-text">{currentProfile.recommendedTemperature.toFixed(1)}</p>
              </div>
              <div className="rounded bg-surface px-3 py-2">
                <p className="text-dim">{t("settings.cachePrice")}</p>
                <p className="mt-1 font-medium text-text">{formatCny(currentProfile.priceCny.cacheHitInput)}/M</p>
              </div>
            </div>
          </div>

          <div className="rounded border border-border bg-bg/80 p-3">
            <div className="mb-3 flex items-start justify-between gap-3">
              <div>
                <label className="flex items-center gap-2 text-sm font-medium text-text">
                  <FileText size={14} />
                  {t("settings.initialPrompt")}
                </label>
                <p className="mt-1 text-xs leading-5 text-dim">{t("settings.initialPromptDesc")}</p>
              </div>
              <span className="shrink-0 text-xs text-dim">
                {initialPrompt.length}/{INITIAL_PROMPT_LIMIT}
              </span>
            </div>

            <div className="mb-3 grid grid-cols-1 gap-2 sm:grid-cols-3">
              {INITIAL_PROMPT_PRESETS.map((preset) => (
                <button
                  key={preset.id}
                  onClick={() => {
                    setInitialPrompt(preset.prompts[lang]);
                    setError("");
                  }}
                  className="motion-lift rounded border border-border bg-surface px-3 py-2 text-left transition hover:border-dim hover:text-text"
                >
                  <span className="block text-xs font-semibold text-text">{t(preset.labelKey)}</span>
                  <span className="mt-1 block text-[11px] leading-4 text-dim">{t(preset.descriptionKey)}</span>
                </button>
              ))}
            </div>

            <textarea
              value={initialPrompt}
              maxLength={INITIAL_PROMPT_LIMIT}
              onChange={(e) => {
                setInitialPrompt(e.target.value);
                setError("");
              }}
              placeholder={t("settings.initialPromptPlaceholder")}
              className="min-h-40 w-full resize-y rounded border border-border bg-bg/80 px-3 py-2 font-mono text-xs leading-5 text-text outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/15"
            />
            <p className="mt-2 text-xs leading-5 text-dim">{t("settings.initialPromptHint")}</p>
            <div className="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-2">
              <div>
                <label className="mb-2 block text-xs font-semibold text-text">{t("settings.roleCard")}</label>
                <textarea
                  value={roleCard}
                  maxLength={INITIAL_PROMPT_LIMIT}
                  onChange={(e) => {
                    setRoleCard(e.target.value);
                    setError("");
                  }}
                  placeholder={t("settings.roleCardPlaceholder")}
                  className="min-h-32 w-full resize-y rounded border border-border bg-bg/80 px-3 py-2 text-xs leading-5 text-text outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/15"
                />
              </div>
              <div>
                <label className="mb-2 block text-xs font-semibold text-text">{t("settings.worldBook")}</label>
                <textarea
                  value={worldBook}
                  maxLength={INITIAL_PROMPT_LIMIT}
                  onChange={(e) => {
                    setWorldBook(e.target.value);
                    setError("");
                  }}
                  placeholder={t("settings.worldBookPlaceholder")}
                  className="min-h-32 w-full resize-y rounded border border-border bg-bg/80 px-3 py-2 text-xs leading-5 text-text outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/15"
                />
              </div>
            </div>
          </div>

          <div>
            <label className="mb-2 flex items-center gap-2 text-sm text-text">
              <Link size={14} />
              Base URL
            </label>
            <input
              value={baseUrl}
              onChange={(e) => {
                setBaseUrl(e.target.value);
                setError("");
              }}
              className="w-full rounded border border-border bg-bg/80 px-3 py-2 text-sm text-text outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/15"
            />
          </div>

          <div>
            <label className="mb-2 block text-sm text-text">
              {t("settings.maxTokens")}: {formatTokenLimit(settings.maxTokens)}
            </label>
            <input
              type="range"
              min="4096"
              max={currentProfile.maxOutput}
              step="1024"
              value={Math.min(settings.maxTokens, currentProfile.maxOutput)}
              onChange={(e) => savePartial({ maxTokens: parseInt(e.target.value) })}
              className="w-full"
            />
            <div className="flex justify-between text-xs text-dim">
              <span>4K</span>
              <span>{formatTokenLimit(currentProfile.maxOutput)}</span>
            </div>
          </div>

          <div>
            <label className="mb-2 block text-sm text-text">
              {t("settings.temperature")}: {settings.temperature.toFixed(1)}
            </label>
            <input
              type="range"
              min="0"
              max="2"
              step="0.1"
              value={settings.temperature}
              onChange={(e) => savePartial({ temperature: parseFloat(e.target.value) })}
              className="w-full"
            />
            <div className="flex justify-between text-xs text-dim">
              <span>0.0 ({t("settings.precise")})</span>
              <span>2.0 ({t("settings.creative")})</span>
            </div>
          </div>

          {supportsThinking(settings.model) && (
            <div className="space-y-3 rounded border border-border bg-bg/80 p-3">
              <div className="flex items-center justify-between">
                <div>
                  <span className="text-sm text-text">{t("chat.thinkingToggle")}</span>
                  <p className="text-xs text-dim">{t("chat.thinkingDesc")}</p>
                </div>
                <button
                  onClick={() => savePartial({ thinkingEnabled: !settings.thinkingEnabled })}
                  className={`relative h-5 w-10 rounded-full transition ${settings.thinkingEnabled ? "bg-accent" : "bg-border"}`}
                >
                  <div className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition ${settings.thinkingEnabled ? "left-5" : "left-0.5"}`} />
                </button>
              </div>
              {settings.thinkingEnabled && (
                <div>
                  <label className="mb-2 block text-xs text-dim">{t("chat.reasoningDisplay")}</label>
                  <div className="grid grid-cols-3 gap-2">
                    {[
                      ["show", t("chat.reasoningShow")],
                      ["collapse", t("chat.reasoningCollapse")],
                      ["hide", t("chat.reasoningHide")],
                    ].map(([id, label]) => (
                      <button
                        key={id}
                        onClick={() => savePartial({ reasoningDisplay: id as any })}
                        className={`motion-lift rounded border px-2 py-1.5 text-xs transition ${
                          settings.reasoningDisplay === id
                            ? "border-accent bg-accent/10 text-text"
                            : "border-border bg-surface text-dim hover:border-dim hover:text-text"
                        }`}
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          <div className="flex items-center justify-between">
            <div>
              <span className="text-sm text-text">{t("settings.autoCowork")}</span>
              <p className="text-xs text-dim">{t("settings.autoCoworkDesc")}</p>
            </div>
            <button
              onClick={() => savePartial({ autoCowork: !settings.autoCowork })}
              className={`relative h-5 w-10 rounded-full transition ${settings.autoCowork ? "bg-accent" : "bg-border"}`}
            >
              <div className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition ${settings.autoCowork ? "left-5" : "left-0.5"}`} />
            </button>
          </div>

          {error && (
            <div className="rounded border border-red/30 bg-red/10 px-3 py-2 text-xs text-red">
              {error}
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 border-t border-border px-5 py-4">
          <button
            onClick={settings.togglePanel}
            className="motion-lift rounded px-4 py-1.5 text-sm text-dim transition hover:bg-panel hover:text-text"
          >
            {t("settings.cancel")}
          </button>
          <button
            onClick={handleSave}
            className="motion-lift rounded bg-accent px-4 py-1.5 text-sm font-semibold text-bg transition hover:bg-accent-alt"
          >
            {t("settings.save")}
          </button>
        </div>
      </div>
    </div>
  );
}
