import type { Lang } from "../stores/i18nStore";

export type ChatTemplateId = "character" | "lore" | "summary" | "export" | "review" | "translate" | "writing";

export interface ChatTemplate {
  id: ChatTemplateId;
  command: string;
  labelKey: string;
  descriptionKey: string;
  prompts: Record<Lang, string>;
}

export const CHAT_TEMPLATES: ChatTemplate[] = [
  {
    id: "character",
    command: "/char",
    labelKey: "template.character",
    descriptionKey: "template.characterDesc",
    prompts: {
      zh: `请根据以下信息整理一张可用于酒馆/RP 的角色卡：

名称：
身份：
外貌：
性格：
说话风格：
关系与动机：
禁忌与边界：
示例台词：

请输出结构清晰、可直接粘贴到角色卡区域的版本。`,
      en: `Create a tavern/RP character card from the notes below:

Name:
Role:
Appearance:
Personality:
Speech style:
Relationships and motivation:
Boundaries:
Example dialogue:

Return a clear version that can be pasted directly into the character-card field.`,
    },
  },
  {
    id: "lore",
    command: "/lore",
    labelKey: "template.lore",
    descriptionKey: "template.loreDesc",
    prompts: {
      zh: `请整理一份世界书/设定集条目：

世界观：
地点：
组织：
关键人物：
规则或限制：
当前剧情状态：
可触发关键词：

请保持条目精炼、稳定，适合放入长期上下文。`,
      en: `Prepare a lorebook entry:

Setting:
Locations:
Factions:
Key characters:
Rules or constraints:
Current story state:
Trigger keywords:

Keep it concise, stable, and suitable for long-context reuse.`,
    },
  },
  {
    id: "summary",
    command: "/summary",
    labelKey: "template.summary",
    descriptionKey: "template.summaryDesc",
    prompts: {
      zh: "请总结当前会话，保留关键事实、人物设定、未完成任务、重要决定和下一步行动。输出要便于之后恢复上下文。",
      en: "Summarize the current conversation, preserving key facts, character settings, unfinished tasks, important decisions, and next actions. Make it suitable for restoring context later.",
    },
  },
  {
    id: "export",
    command: "/export",
    labelKey: "template.export",
    descriptionKey: "template.exportDesc",
    prompts: {
      zh: "",
      en: "",
    },
  },
  {
    id: "review",
    command: "/review",
    labelKey: "template.review",
    descriptionKey: "template.reviewDesc",
    prompts: {
      zh: "请审查当前改动，优先列出可能的 bug、行为回归、边界情况和缺失测试。按严重程度排序，并给出可执行的修复建议。",
      en: "Review the current changes. Prioritize possible bugs, behavioral regressions, edge cases, and missing tests. Sort by severity and include actionable fixes.",
    },
  },
  {
    id: "translate",
    command: "/translate",
    labelKey: "template.translate",
    descriptionKey: "template.translateDesc",
    prompts: {
      zh: "请把下面内容翻译成自然、准确、适合产品文档的中文，并保留代码、路径、命令和专有名词：\n\n",
      en: "Translate the following into natural, accurate English for product documentation. Preserve code, paths, commands, and proper nouns:\n\n",
    },
  },
  {
    id: "writing",
    command: "/write",
    labelKey: "template.writing",
    descriptionKey: "template.writingDesc",
    prompts: {
      zh: "请把下面内容扩写成一段自然、有节奏、适合发布或视频口播的文字。保持信息准确，不要夸大：\n\n",
      en: "Rewrite the following into natural, well-paced copy suitable for release notes or a video script. Keep it accurate and avoid exaggeration:\n\n",
    },
  },
];

export function chatTemplateText(id: ChatTemplateId, lang: Lang) {
  return CHAT_TEMPLATES.find((template) => template.id === id)?.prompts[lang] ?? "";
}

export function chatTemplateByCommand(command: string) {
  const normalized = command.trim().toLowerCase();
  return CHAT_TEMPLATES.find((template) => template.command === normalized);
}
