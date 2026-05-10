import type { Message, TokenUsage } from "../stores/sessionStore";
import type { ReasoningDisplay } from "../stores/settingsStore";

export type ChatMode = "code" | "rp" | "writing" | "chat";

export interface ModelPriceCny {
  cacheHitInput: number;
  cacheMissInput: number;
  output: number;
}

export interface ModelOption {
  id: string;
  name: string;
  badge: string;
  description: string;
  contextWindow: number;
  maxOutput: number;
  recommendedMaxTokens: number;
  recommendedTemperature: number;
  priceCny: ModelPriceCny;
  legacy?: boolean;
  deprecatesOn?: string;
  thinking?: boolean;
  strengths: string[];
}

export interface ChatModePreset {
  id: ChatMode;
  labelKey: string;
  descriptionKey: string;
  model: string;
  maxTokens: number;
  temperature: number;
  thinkingEnabled: boolean;
  reasoningDisplay: ReasoningDisplay;
}

export const CHAT_MODE_PRESETS: ChatModePreset[] = [
  {
    id: "code",
    labelKey: "mode.code",
    descriptionKey: "mode.codeDesc",
    model: "deepseek-v4-pro",
    maxTokens: 65_536,
    temperature: 0.4,
    thinkingEnabled: true,
    reasoningDisplay: "collapse",
  },
  {
    id: "rp",
    labelKey: "mode.rp",
    descriptionKey: "mode.rpDesc",
    model: "deepseek-v4-flash",
    maxTokens: 49_152,
    temperature: 0.9,
    thinkingEnabled: false,
    reasoningDisplay: "hide",
  },
  {
    id: "writing",
    labelKey: "mode.writing",
    descriptionKey: "mode.writingDesc",
    model: "deepseek-v4-pro",
    maxTokens: 65_536,
    temperature: 0.85,
    thinkingEnabled: false,
    reasoningDisplay: "collapse",
  },
  {
    id: "chat",
    labelKey: "mode.chat",
    descriptionKey: "mode.chatDesc",
    model: "deepseek-v4-flash",
    maxTokens: 8_192,
    temperature: 0.7,
    thinkingEnabled: false,
    reasoningDisplay: "hide",
  },
];

export const MODEL_OPTIONS: ModelOption[] = [
  {
    id: "deepseek-v4-pro",
    name: "DeepSeek V4 Pro",
    badge: "1M / Agent",
    description: "Quality-first model for complex agentic coding, planning, and long-context work.",
    contextWindow: 1_048_576,
    maxOutput: 393_216,
    recommendedMaxTokens: 65_536,
    recommendedTemperature: 0.6,
    priceCny: {
      cacheHitInput: 0.025,
      cacheMissInput: 3,
      output: 6,
    },
    thinking: true,
    strengths: ["Complex coding", "planning", "reasoning", "long-context review"],
  },
  {
    id: "deepseek-v4-flash",
    name: "DeepSeek V4 Flash",
    badge: "1M / Fast",
    description: "Fast and economical model for everyday coding tasks and interactive editing.",
    contextWindow: 1_048_576,
    maxOutput: 393_216,
    recommendedMaxTokens: 32_768,
    recommendedTemperature: 0.7,
    priceCny: {
      cacheHitInput: 0.02,
      cacheMissInput: 1,
      output: 2,
    },
    thinking: true,
    strengths: ["Fast iteration", "simple agents", "drafting", "routine coding"],
  },
  {
    id: "deepseek-reasoner",
    name: "DeepSeek Reasoner",
    badge: "Legacy",
    description: "Compatibility alias for V4 Flash thinking mode. Prefer V4 models for new sessions.",
    contextWindow: 1_048_576,
    maxOutput: 393_216,
    recommendedMaxTokens: 32_768,
    recommendedTemperature: 0.7,
    priceCny: {
      cacheHitInput: 0.02,
      cacheMissInput: 1,
      output: 2,
    },
    legacy: true,
    deprecatesOn: "2026-07-24",
    thinking: true,
    strengths: ["Compatibility", "thinking alias"],
  },
  {
    id: "deepseek-chat",
    name: "DeepSeek Chat",
    badge: "Legacy",
    description: "Compatibility alias for V4 Flash non-thinking mode. Prefer deepseek-v4-flash.",
    contextWindow: 1_048_576,
    maxOutput: 393_216,
    recommendedMaxTokens: 32_768,
    recommendedTemperature: 0.7,
    priceCny: {
      cacheHitInput: 0.02,
      cacheMissInput: 1,
      output: 2,
    },
    legacy: true,
    deprecatesOn: "2026-07-24",
    strengths: ["Compatibility", "non-thinking alias"],
  },
];

export function modelProfile(model: string) {
  return MODEL_OPTIONS.find((m) => m.id === model) ?? MODEL_OPTIONS[0];
}

export function modelLabel(model: string) {
  return modelProfile(model).name;
}

export function supportsThinking(model: string) {
  return modelProfile(model).thinking ?? false;
}

export function estimateCostCny(model: string, usage?: Partial<TokenUsage>) {
  const profile = modelProfile(model);
  const hit = usage?.promptCacheHitTokens ?? 0;
  const reportedMiss = usage?.promptCacheMissTokens ?? 0;
  const miss = hit + reportedMiss > 0 ? reportedMiss : usage?.promptTokens ?? 0;
  const output = usage?.completionTokens ?? 0;
  return (
    (hit / 1_000_000) * profile.priceCny.cacheHitInput +
    (miss / 1_000_000) * profile.priceCny.cacheMissInput +
    (output / 1_000_000) * profile.priceCny.output
  );
}

export function estimateCacheSavingsCny(model: string, usage?: Partial<TokenUsage>) {
  const profile = modelProfile(model);
  const hit = usage?.promptCacheHitTokens ?? 0;
  return (hit / 1_000_000) * Math.max(profile.priceCny.cacheMissInput - profile.priceCny.cacheHitInput, 0);
}

export function contextUsagePercent(model: string, totalTokens: number) {
  const profile = modelProfile(model);
  return Math.min(100, Math.round((totalTokens / profile.contextWindow) * 100));
}

export function chatModePreset(mode: string) {
  return CHAT_MODE_PRESETS.find((preset) => preset.id === mode) ?? CHAT_MODE_PRESETS[0];
}

export function estimateTextTokens(text = "") {
  return Math.max(0, Math.ceil(text.length / 3));
}

export function estimateMessagesTokens(messages: Message[]) {
  return messages.reduce(
    (total, msg) => total + estimateTextTokens(msg.role) + estimateTextTokens(msg.content) + estimateTextTokens(msg.reasoningContent),
    0
  );
}

export function estimateContextBudget(
  model: string,
  settings: { maxTokens: number; initialPrompt?: string; roleCard?: string; worldBook?: string },
  messages: Message[],
  summaryTokens = 0
) {
  const profile = modelProfile(model);
  const stableTokens =
    estimateTextTokens(settings.initialPrompt) +
    estimateTextTokens(settings.roleCard) +
    estimateTextTokens(settings.worldBook);
  const recentTokens = estimateMessagesTokens(messages);
  const promptTokens = stableTokens + recentTokens + summaryTokens;
  const reservedOutputTokens = Math.min(settings.maxTokens || profile.recommendedMaxTokens, profile.maxOutput);
  const totalReservedTokens = promptTokens + reservedOutputTokens;
  const remainingTokens = Math.max(0, profile.contextWindow - totalReservedTokens);
  const usagePercent = Math.min(100, Math.round((totalReservedTokens / profile.contextWindow) * 100));
  const estimatedCost =
    (promptTokens / 1_000_000) * profile.priceCny.cacheMissInput +
    (reservedOutputTokens / 1_000_000) * profile.priceCny.output;

  return {
    stableTokens,
    summaryTokens,
    recentTokens,
    promptTokens,
    reservedOutputTokens,
    totalReservedTokens,
    remainingTokens,
    usagePercent,
    estimatedCost,
  };
}

export function formatTokenLimit(tokens: number) {
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(tokens % 1_000_000 === 0 ? 0 : 1)}M`;
  if (tokens >= 1024) return `${Math.round(tokens / 1024)}K`;
  return String(tokens);
}

export function formatCny(value: number) {
  if (!Number.isFinite(value) || value <= 0) return "¥0.0000";
  if (value < 0.0001) return "<¥0.0001";
  return `¥${value.toFixed(value < 0.01 ? 4 : 2)}`;
}
