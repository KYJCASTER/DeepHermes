import { create } from "zustand";
import { GetSettings, UpdateSettings, SetAPIKey, GetAPIKeyStatus } from "../lib/wails";

interface SettingsState {
  model: string;
  maxTokens: number;
  temperature: number;
  baseUrl: string;
  thinkingEnabled: boolean;
  autoCowork: boolean;
  apiKeyStatus: string;
  isOpen: boolean;

  load: () => Promise<void>;
  save: (s: Partial<SettingsState>) => Promise<void>;
  setAPIKey: (key: string) => Promise<void>;
  setAPIKeyStatus: (status: string) => void;
  togglePanel: () => void;
}

export const useSettingsStore = create<SettingsState>((set, get) => ({
  model: "deepseek-v4-pro",
  maxTokens: 32768,
  temperature: 0.7,
  baseUrl: "https://api.deepseek.com",
  thinkingEnabled: false,
  autoCowork: false,
  apiKeyStatus: "unknown",
  isOpen: false,

  load: async () => {
    try {
      const settings = await GetSettings();
      const status = await GetAPIKeyStatus();
      set({ ...settings, apiKeyStatus: status });
    } catch (e) {
      console.error("Failed to load settings:", e);
    }
  },

  save: async (partial) => {
    const current = get();
    const next = {
      model: partial.model ?? current.model,
      maxTokens: partial.maxTokens ?? current.maxTokens,
      temperature: partial.temperature ?? current.temperature,
      baseUrl: partial.baseUrl ?? current.baseUrl,
      thinkingEnabled: partial.thinkingEnabled ?? current.thinkingEnabled,
      autoCowork: partial.autoCowork ?? current.autoCowork,
    };
    try {
      await UpdateSettings(next);
      set(next);
    } catch (e) {
      console.error("Failed to save settings:", e);
      throw e;
    }
  },

  setAPIKey: async (key: string) => {
    await SetAPIKey(key);
    set({ apiKeyStatus: "configured" });
  },

  setAPIKeyStatus: (status: string) => set({ apiKeyStatus: status }),

  togglePanel: () => set((s) => ({ isOpen: !s.isOpen })),
}));
