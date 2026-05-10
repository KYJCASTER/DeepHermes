import type { Lang } from "../stores/i18nStore";

export type InitialPromptPresetId = "blank" | "tavern" | "story";

export interface InitialPromptPreset {
  id: InitialPromptPresetId;
  labelKey: string;
  descriptionKey: string;
  prompts: Record<Lang, string>;
}

export const INITIAL_PROMPT_PRESETS: InitialPromptPreset[] = [
  {
    id: "blank",
    labelKey: "settings.promptPresetBlank",
    descriptionKey: "settings.promptPresetBlankDesc",
    prompts: {
      zh: "",
      en: "",
    },
  },
  {
    id: "tavern",
    labelKey: "settings.promptPresetTavern",
    descriptionKey: "settings.promptPresetTavernDesc",
    prompts: {
      zh: `你正在参与一个沉浸式文字角色扮演。请严格遵守：
- 始终保持角色设定、世界观、关系、语气与记忆连续。
- 用生动但克制的描写推进场景，优先写角色的语言、动作、神态和环境反馈。
- 不要替玩家决定台词、动作、心理或结局；只描写你负责的角色和世界反应。
- 当用户提供角色卡、世界书、开场白或规则时，将它们视为最高优先级的扮演资料。
- 对话格式清晰，可使用引号表示台词，括号表示动作、旁白或场景变化。
- 如果信息不足，顺着当前场景自然回应，必要时用简短问题确认。`,
      en: `You are participating in immersive text roleplay. Follow these rules:
- Stay consistent with the character, setting, relationships, tone, and established memory.
- Advance the scene with vivid but restrained writing, focusing on the character's dialogue, actions, expressions, and environmental feedback.
- Do not decide the player's dialogue, actions, thoughts, or outcome. Only write your character and the world's response.
- Treat any character card, lorebook, opening message, or user rules as the highest-priority roleplay material.
- Keep formatting clear. Use quotes for dialogue and parentheses for actions, narration, or scene changes.
- If details are missing, respond naturally within the current scene or ask a brief clarifying question.`,
    },
  },
  {
    id: "story",
    labelKey: "settings.promptPresetStory",
    descriptionKey: "settings.promptPresetStoryDesc",
    prompts: {
      zh: `你是一个擅长长篇互动叙事的写作伙伴。请保持文风稳定、人物动机清晰、场景连续，并根据用户输入推进剧情。不要直接替用户做决定；优先给出自然的剧情反馈、角色回应和可继续互动的空间。`,
      en: `You are a writing partner for long-form interactive fiction. Keep the prose style stable, character motivations clear, and scenes continuous. Advance the story from the user's input without making decisions for the user; prioritize natural narrative feedback, character responses, and room for continued interaction.`,
    },
  },
];

export function promptPresetText(id: InitialPromptPresetId, lang: Lang) {
  return INITIAL_PROMPT_PRESETS.find((preset) => preset.id === id)?.prompts[lang] ?? "";
}
