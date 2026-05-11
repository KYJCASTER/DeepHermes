import { create } from "zustand";
import { GetSettings, UpdateSettings, SetAPIKey, GetAPIKeyStatus, SetOCRAPIKey, GetOCRAPIKeyStatus } from "../lib/wails";
import type { ChatMode } from "../lib/models";

export type ReasoningDisplay = "show" | "collapse" | "hide";
export type ToolMode = "read_only" | "confirm" | "auto";
export type ToolOverrides = Record<string, ToolMode>;

function normalizeToolMode(value: unknown): ToolMode {
  if (value === "read_only" || value === "confirm" || value === "auto") return value;
  return "confirm";
}

function normalizeToolOverrides(value: unknown): ToolOverrides {
  if (!value || typeof value !== "object" || Array.isArray(value)) return {};
  return Object.entries(value as Record<string, string>).reduce<ToolOverrides>((acc, [name, mode]) => {
    if (mode === "read_only" || mode === "confirm" || mode === "auto") {
      acc[name] = mode;
    }
    return acc;
  }, {});
}

function normalizeBashBlocklist(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === "string").map((item) => item.trim()).filter(Boolean);
}

interface SettingsState {
  model: string;
  mode: ChatMode;
  portable: boolean;
  minimizeToTray: boolean;
  maxTokens: number;
  temperature: number;
  baseUrl: string;
  apiTimeout: number;
  apiMaxRetries: number;
  apiProxyUrl: string;
  thinkingEnabled: boolean;
  reasoningDisplay: ReasoningDisplay;
  autoCowork: boolean;
  toolMode: ToolMode;
  toolOverrides: ToolOverrides;
  bashBlocklist: string[];
  initialPrompt: string;
  roleCard: string;
  worldBook: string;
  ocrEnabled: boolean;
  ocrProvider: string;
  ocrBaseUrl: string;
  ocrModel: string;
  ocrPrompt: string;
  ocrTimeout: number;
  apiKeyStatus: string;
  ocrKeyStatus: string;
  isOpen: boolean;

  load: () => Promise<void>;
  save: (s: Partial<SettingsState>) => Promise<void>;
  setAPIKey: (key: string) => Promise<void>;
  setOCRAPIKey: (key: string) => Promise<void>;
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
  apiTimeout: 120,
  apiMaxRetries: 3,
  apiProxyUrl: "",
  thinkingEnabled: false,
  reasoningDisplay: "collapse",
  autoCowork: false,
  toolMode: "confirm",
  toolOverrides: {},
  bashBlocklist: [],
  initialPrompt: "",
  roleCard: "",
  worldBook: "",
  ocrEnabled: false,
  ocrProvider: "openai_compatible",
  ocrBaseUrl: "",
  ocrModel: "",
  ocrPrompt: "Extract all readable text from this image. Preserve line breaks when useful. If there is no readable text, briefly describe the visible content.",
  ocrTimeout: 60,
  apiKeyStatus: "unknown",
  ocrKeyStatus: "unknown",
  isOpen: false,

  load: async () => {
    try {
      const settings = await GetSettings();
      const status = await GetAPIKeyStatus();
      const ocrStatus = await GetOCRAPIKeyStatus();
      set({
        ...settings,
        mode: settings?.mode ?? "code",
        portable: settings?.portable ?? false,
        minimizeToTray: settings?.minimizeToTray ?? false,
        apiTimeout: settings?.apiTimeout ?? 120,
        apiMaxRetries: settings?.apiMaxRetries ?? 3,
        apiProxyUrl: settings?.apiProxyUrl ?? "",
        initialPrompt: settings?.initialPrompt ?? "",
        roleCard: settings?.roleCard ?? "",
        worldBook: settings?.worldBook ?? "",
        toolMode: normalizeToolMode(settings?.toolMode),
        toolOverrides: normalizeToolOverrides(settings?.toolOverrides),
        bashBlocklist: normalizeBashBlocklist(settings?.bashBlocklist),
        ocrEnabled: settings?.ocrEnabled ?? false,
        ocrProvider: settings?.ocrProvider ?? "openai_compatible",
        ocrBaseUrl: settings?.ocrBaseUrl ?? "",
        ocrModel: settings?.ocrModel ?? "",
        ocrPrompt: settings?.ocrPrompt ?? "",
        ocrTimeout: settings?.ocrTimeout ?? 60,
        apiKeyStatus: status,
        ocrKeyStatus: ocrStatus,
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
      apiTimeout: partial.apiTimeout ?? current.apiTimeout,
      apiMaxRetries: partial.apiMaxRetries ?? current.apiMaxRetries,
      apiProxyUrl: partial.apiProxyUrl ?? current.apiProxyUrl,
      thinkingEnabled: partial.thinkingEnabled ?? current.thinkingEnabled,
      reasoningDisplay: partial.reasoningDisplay ?? current.reasoningDisplay,
      autoCowork: partial.autoCowork ?? current.autoCowork,
      toolMode: partial.toolMode ?? current.toolMode,
      toolOverrides: partial.toolOverrides ?? current.toolOverrides,
      bashBlocklist: partial.bashBlocklist ?? current.bashBlocklist,
      initialPrompt: partial.initialPrompt ?? current.initialPrompt,
      roleCard: partial.roleCard ?? current.roleCard,
      worldBook: partial.worldBook ?? current.worldBook,
      ocrEnabled: partial.ocrEnabled ?? current.ocrEnabled,
      ocrProvider: partial.ocrProvider ?? current.ocrProvider,
      ocrBaseUrl: partial.ocrBaseUrl ?? current.ocrBaseUrl,
      ocrModel: partial.ocrModel ?? current.ocrModel,
      ocrPrompt: partial.ocrPrompt ?? current.ocrPrompt,
      ocrTimeout: partial.ocrTimeout ?? current.ocrTimeout,
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

  setOCRAPIKey: async (key: string) => {
    await SetOCRAPIKey(key);
    set({ ocrKeyStatus: "configured" });
  },

  setAPIKeyStatus: (status: string) => set({ apiKeyStatus: status }),

  togglePanel: () => set((s) => ({ isOpen: !s.isOpen })),
}));
