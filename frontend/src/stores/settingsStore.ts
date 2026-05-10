import { create } from "zustand";
import { GetSettings, UpdateSettings, SetAPIKey, GetAPIKeyStatus } from "../lib/wails";
import type { ChatMode } from "../lib/models";

export type ReasoningDisplay = "show" | "collapse" | "hide";

interface SettingsState {
  model: string;
  mode: ChatMode;
  portable: boolean;
  minimizeToTray: boolean;
  maxTokens: number;
  temperature: number;
  baseUrl: string;
  thinkingEnabled: boolean;
  reasoningDisplay: ReasoningDisplay;
  autoCowork: boolean;
  initialPrompt: string;
  roleCard: string;
  worldBook: string;
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
  mode: "code",
  portable: false,
  minimizeToTray: false,
  maxTokens: 32768,
  temperature: 0.7,
  baseUrl: "https://api.deepseek.com",
  thinkingEnabled: false,
  reasoningDisplay: "collapse",
  autoCowork: false,
  initialPrompt: "",
  roleCard: "",
  worldBook: "",
  apiKeyStatus: "unknown",
  isOpen: false,

  load: async () => {
    try {
      const settings = await GetSettings();
      const status = await GetAPIKeyStatus();
      set({
        ...settings,
        mode: settings?.mode ?? "code",
        portable: settings?.portable ?? false,
        minimizeToTray: settings?.minimizeToTray ?? false,
        initialPrompt: settings?.initialPrompt ?? "",
        roleCard: settings?.roleCard ?? "",
        worldBook: settings?.worldBook ?? "",
        apiKeyStatus: status,
      });
    } catch (e) {
      console.error("Failed to load settings:", e);
    }
  },

  save: async (partial) => {
    const current = get();
    const next = {
      model: partial.model ?? current.model,
      mode: partial.mode ?? current.mode,
      portable: partial.portable ?? current.portable,
      minimizeToTray: partial.minimizeToTray ?? current.minimizeToTray,
      maxTokens: partial.maxTokens ?? current.maxTokens,
      temperature: partial.temperature ?? current.temperature,
      baseUrl: partial.baseUrl ?? current.baseUrl,
      thinkingEnabled: partial.thinkingEnabled ?? current.thinkingEnabled,
      reasoningDisplay: partial.reasoningDisplay ?? current.reasoningDisplay,
      autoCowork: partial.autoCowork ?? current.autoCowork,
      initialPrompt: partial.initialPrompt ?? current.initialPrompt,
      roleCard: partial.roleCard ?? current.roleCard,
      worldBook: partial.worldBook ?? current.worldBook,
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
