import { AlertCircle, ArrowRight, ExternalLink, Key, Moon, Palette, Sparkles } from "lucide-react";
import { useSettingsStore } from "../../stores/settingsStore";
import { useI18n } from "../../stores/i18nStore";
import { useThemeStore } from "../../stores/themeStore";
import { useState } from "react";

export default function WelcomeScreen() {
  const [apiKey, setApiKey] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const settings = useSettingsStore();
  const theme = useThemeStore((s) => s.theme);
  const toggleTheme = useThemeStore((s) => s.toggleTheme);
  const { t } = useI18n();
  const themeTitle = theme === "dark" ? t("theme.comic") : t("theme.dark");

  const handleSetup = async () => {
    const key = apiKey.trim();
    if (!key) return;
    setSaving(true);
    setError("");

    try {
      await settings.setAPIKey(key);
      // Success - the store will update apiKeyStatus to "configured"
    } catch (e: any) {
      console.error("API key save error:", e);
      setError(t("welcome.saveError") + ": " + (e?.message || String(e)));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="welcome-shell h-screen text-text">
      <button
        onClick={toggleTheme}
        className="app-content motion-lift absolute right-5 top-5 z-10 flex h-9 w-9 items-center justify-center rounded border border-border bg-surface/80 text-dim backdrop-blur transition hover:text-accent"
        title={themeTitle}
      >
        {theme === "dark" ? <Moon size={16} /> : <Palette size={16} />}
      </button>
      <div className="app-content mx-auto flex h-full max-w-6xl items-center justify-center px-8">
        <div className="grid w-full grid-cols-[1fr_420px] gap-10 max-lg:grid-cols-1 max-lg:max-w-xl">
          <div className="fade-up flex flex-col justify-center">
            <div className="ds-mark mb-6 inline-flex h-12 w-12 items-center justify-center rounded bg-accent/12 text-accent">
              <Sparkles size={26} />
            </div>
            <p className="mb-3 text-xs font-semibold uppercase tracking-[0.28em] text-accent">DeepSeek V4</p>
            <h1 className="mb-3 text-5xl font-semibold tracking-normal text-text max-sm:text-4xl">{t("welcome.title")}</h1>
            <p className="max-w-2xl text-base leading-7 text-dim">{t("welcome.desc")}</p>
          </div>

          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleSetup();
            }}
            className="fade-up-delay panel-card rounded border border-border p-6"
          >
            <div className="mb-5 flex items-center gap-3">
              <div className="flex h-9 w-9 items-center justify-center rounded bg-panel text-accent">
                <Key size={18} />
              </div>
              <div>
                <h2 className="text-base font-semibold text-text">{t("welcome.setup")}</h2>
                <p className="text-xs text-dim">DeepSeek API</p>
              </div>
            </div>

            <p className="mb-4 text-sm leading-6 text-dim">
              {t("welcome.setupDesc")}{" "}
              <code className="rounded bg-panel px-1.5 py-0.5 text-accent">~/.deephermes/config.yaml</code>{" "}
              {t("welcome.setupDesc2")}
            </p>

            <input
              type="password"
              value={apiKey}
              onChange={(e) => { setApiKey(e.target.value); setError(""); }}
              placeholder={t("welcome.placeholder")}
              className="w-full rounded border border-border bg-bg/80 px-3 py-3 font-mono text-sm text-text outline-none transition placeholder:text-dim focus:border-accent focus:ring-2 focus:ring-accent/20"
              autoFocus
            />

            {error && (
              <div className="mt-3 flex items-start gap-2 rounded border border-red/30 bg-red/10 px-3 py-2 text-xs leading-5 text-red">
                <AlertCircle size={14} className="mt-0.5 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <div className="mt-4 flex items-center justify-between text-xs">
              <a
                href="https://platform.deepseek.com/api_keys"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 text-accent transition hover:text-accent-alt"
              >
                {t("welcome.getKey")}
                <ExternalLink size={12} />
              </a>
            </div>

            <button
              type="submit"
              disabled={!apiKey.trim() || saving}
              className="motion-lift mt-5 flex w-full items-center justify-center gap-2 rounded bg-accent px-4 py-3 text-sm font-semibold text-bg transition hover:bg-accent-alt disabled:cursor-not-allowed disabled:opacity-40"
            >
              {saving ? t("welcome.saving") : t("welcome.start")}
              {!saving && <ArrowRight size={16} />}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
