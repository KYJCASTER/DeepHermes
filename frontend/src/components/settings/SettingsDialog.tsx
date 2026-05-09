import { CheckCircle, Key, Link, Moon, SlidersHorizontal, Sun, X } from "lucide-react";
import { useSettingsStore } from "../../stores/settingsStore";
import { useI18n } from "../../stores/i18nStore";
import { useThemeStore } from "../../stores/themeStore";
import { useState } from "react";
import { MODEL_OPTIONS, supportsThinking } from "../../lib/models";

export default function SettingsDialog() {
  const settings = useSettingsStore();
  const theme = useThemeStore((s) => s.theme);
  const setTheme = useThemeStore((s) => s.setTheme);
  const { t } = useI18n();
  const [apiKey, setApiKey] = useState("");
  const [baseUrl, setBaseUrl] = useState(settings.baseUrl);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  const savePartial = (partial: Parameters<typeof settings.save>[0]) => {
    settings.save(partial).catch((e: any) => setError(e?.message || String(e)));
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
        maxTokens: settings.maxTokens,
        temperature: settings.temperature,
        baseUrl,
        thinkingEnabled: settings.thinkingEnabled,
        reasoningDisplay: settings.reasoningDisplay,
        autoCowork: settings.autoCowork,
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
      if (!apiKey.trim()) {
        settings.togglePanel();
      }
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 px-4 backdrop-blur-sm" onClick={settings.togglePanel}>
      <div
        className="fade-up panel-card max-h-[86vh] w-[620px] overflow-y-auto rounded border border-border"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <div className="flex items-center gap-3">
            <div className="ds-mark flex h-8 w-8 items-center justify-center rounded bg-panel text-accent">
              <SlidersHorizontal size={17} />
            </div>
            <h2 className="text-lg font-semibold text-text">{t("settings.title")}</h2>
          </div>
          <button onClick={settings.togglePanel} className="motion-lift rounded p-1 text-dim transition hover:bg-panel hover:text-text" title="Close">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-5">
          <div>
            <label className="flex items-center gap-2 text-sm text-text mb-2">
              <Key size={14} />
              {t("settings.apiKey")}
              {settings.apiKeyStatus === "configured" && (
                <span className="text-green text-xs flex items-center gap-1">
                  <CheckCircle size={12} /> {t("settings.configured")}
                </span>
              )}
            </label>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => { setApiKey(e.target.value); setError(""); }}
              placeholder={settings.apiKeyStatus === "configured" ? "●●●●●●●●" : "sk-..."}
              className="w-full rounded border border-border bg-bg/80 px-3 py-2 text-sm text-text outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/15"
            />
            <p className="text-xs text-dim mt-1">
              {saved
                ? t("settings.savedToConfig")
                : settings.apiKeyStatus === "configured"
                ? t("settings.keyStored")
                : t("settings.enterKey")}
            </p>
          </div>

          <div>
            <label className="mb-2 block text-sm text-text">{t("settings.appearance")}</label>
            <div className="grid grid-cols-2 gap-2">
              {[
                { id: "light" as const, label: t("theme.lightMode"), icon: Sun },
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

          <div>
            <label className="mb-2 block text-sm text-text">{t("settings.model")}</label>
            <div className="grid grid-cols-2 gap-2">
              {MODEL_OPTIONS.map((option) => (
                <button
                  key={option.id}
                  onClick={() => savePartial({
                    model: option.id,
                    thinkingEnabled: option.thinking ? settings.thinkingEnabled : false,
                  })}
                  className={`motion-lift rounded border p-3 text-left transition ${
                    settings.model === option.id
                      ? "border-accent bg-accent/10 text-text"
                      : "border-border bg-bg/80 text-dim hover:border-dim hover:text-text"
                  }`}
                >
                  <div className="mb-1 flex items-center justify-between gap-2">
                    <span className="text-sm font-medium">{option.name}</span>
                    <span className="rounded bg-panel px-1.5 py-0.5 text-[11px] text-dim">{option.badge}</span>
                  </div>
                  <p className="text-xs leading-5">{option.description}</p>
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="mb-2 flex items-center gap-2 text-sm text-text">
              <Link size={14} />
              Base URL
            </label>
            <input
              value={baseUrl}
              onChange={(e) => { setBaseUrl(e.target.value); setError(""); }}
              className="w-full rounded border border-border bg-bg/80 px-3 py-2 text-sm text-text outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/15"
            />
          </div>

          <div>
            <label className="text-sm text-text mb-2 block">
              {t("settings.maxTokens")}: {settings.maxTokens}
            </label>
            <input
              type="range"
              min="4096"
              max="131072"
              step="1024"
              value={settings.maxTokens}
              onChange={(e) => savePartial({ maxTokens: parseInt(e.target.value) })}
              className="w-full"
            />
            <div className="flex justify-between text-xs text-dim">
              <span>4K</span>
              <span>128K</span>
            </div>
          </div>

          <div>
            <label className="text-sm text-text mb-2 block">
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
                  className={`relative h-5 w-10 rounded-full transition ${
                    settings.thinkingEnabled ? "bg-accent" : "bg-border"
                  }`}
                >
                  <div
                    className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition ${
                      settings.thinkingEnabled ? "left-5" : "left-0.5"
                    }`}
                  />
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
              className={`relative h-5 w-10 rounded-full transition ${
                settings.autoCowork ? "bg-accent" : "bg-border"
              }`}
            >
              <div
                className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition ${
                  settings.autoCowork ? "left-5" : "left-0.5"
                }`}
              />
            </button>
          </div>

          {error && (
            <div className="rounded border border-red/30 bg-red/10 px-3 py-2 text-xs text-red">
              {error}
            </div>
          )}
        </div>

        <div className="px-5 py-4 border-t border-border flex justify-end gap-2">
          <button
            onClick={settings.togglePanel}
            className="motion-lift rounded px-4 py-1.5 text-sm text-dim hover:bg-panel hover:text-text transition"
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
