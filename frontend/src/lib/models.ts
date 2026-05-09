export interface ModelOption {
  id: string;
  name: string;
  badge: string;
  description: string;
  legacy?: boolean;
  thinking?: boolean;
}

export const MODEL_OPTIONS: ModelOption[] = [
  {
    id: "deepseek-v4-pro",
    name: "DeepSeek V4 Pro",
    badge: "1M",
    description: "Quality-first model for agentic coding and long-context work.",
    thinking: true,
  },
  {
    id: "deepseek-v4-flash",
    name: "DeepSeek V4 Flash",
    badge: "1M",
    description: "Fast and economical model for everyday coding tasks.",
    thinking: true,
  },
  {
    id: "deepseek-reasoner",
    name: "DeepSeek Reasoner",
    badge: "Legacy",
    description: "Compatibility alias. Prefer V4 Pro or V4 Flash for new sessions.",
    legacy: true,
    thinking: true,
  },
  {
    id: "deepseek-chat",
    name: "DeepSeek Chat",
    badge: "Legacy",
    description: "Compatibility alias. Scheduled for retirement by DeepSeek.",
    legacy: true,
  },
];

export function modelLabel(model: string) {
  return MODEL_OPTIONS.find((m) => m.id === model)?.name ?? model;
}

export function supportsThinking(model: string) {
  return MODEL_OPTIONS.find((m) => m.id === model)?.thinking ?? false;
}
