import { create } from "zustand";
import {
  AbortMessage,
  BranchSession,
  ContinueLastResponse,
  CreateSession,
  DeleteMessageAt,
  DeleteSession,
  GetHistory,
  ListSessions,
  RegenerateMessage,
  SendMessage,
  UpdateMessage,
} from "../lib/wails";

export interface Message {
  role: string;
  content: string;
  reasoningContent?: string;
}

export interface TokenUsage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  promptCacheHitTokens: number;
  promptCacheMissTokens: number;
  reasoningTokens: number;
}

export interface RunMetrics {
  usage: TokenUsage;
  startedAt: string;
  firstTokenAt?: string;
  finishedAt: string;
  firstTokenMs: number;
  durationMs: number;
  tokensPerSec: number;
  finishReason?: string;
  truncated?: boolean;
}

export interface Session {
  id: string;
  name: string;
  model: string;
  createdAt: string;
  updatedAt?: string;
  msgCount: number;
  messages: Message[];
  streaming: boolean;
  status: "idle" | "thinking" | "streaming" | "executing";
  usage: TokenUsage;
  lastRun?: RunMetrics;
  contextSummaryTokens?: number;
}

interface SessionStore {
  sessions: Session[];
  activeSessionId: string | null;

  loadSessions: () => Promise<void>;
  createSession: (name: string) => Promise<string>;
  deleteSession: (id: string) => void;
  setActiveSession: (id: string) => void;
  sendMessage: (sessionId: string, message: string) => Promise<void>;
  editMessage: (sessionId: string, index: number, content: string) => Promise<void>;
  deleteMessage: (sessionId: string, index: number) => Promise<void>;
  regenerateMessage: (sessionId: string, index: number) => Promise<void>;
  branchSession: (sessionId: string, upToIndex: number) => Promise<string>;
  continueLastResponse: (sessionId: string) => Promise<void>;
  abortMessage: (sessionId: string) => Promise<void>;
  appendToStream: (sessionId: string, content: string, reasoningContent?: string) => void;
  finishStream: (sessionId: string, metrics?: RunMetrics) => void;
  setSessionStatus: (sessionId: string, status: Session["status"]) => void;
  addMessage: (sessionId: string, msg: Message) => void;
}

const emptyUsage = (): TokenUsage => ({
  promptTokens: 0,
  completionTokens: 0,
  totalTokens: 0,
  promptCacheHitTokens: 0,
  promptCacheMissTokens: 0,
  reasoningTokens: 0,
});

function normalizeUsage(value: any): TokenUsage {
  return {
    ...emptyUsage(),
    ...(value || {}),
  };
}

function addUsage(a: TokenUsage, b: TokenUsage): TokenUsage {
  return {
    promptTokens: a.promptTokens + b.promptTokens,
    completionTokens: a.completionTokens + b.completionTokens,
    totalTokens: a.totalTokens + b.totalTokens,
    promptCacheHitTokens: a.promptCacheHitTokens + b.promptCacheHitTokens,
    promptCacheMissTokens: a.promptCacheMissTokens + b.promptCacheMissTokens,
    reasoningTokens: a.reasoningTokens + b.reasoningTokens,
  };
}

function normalizeMessage(msg: any): Message {
  return {
    role: msg.role,
    content: msg.content || "",
    reasoningContent: msg.reasoningContent || msg.reasoning_content || "",
  };
}

function normalizeSession(raw: any, messages: Message[] = []): Session {
  return {
    id: raw.id,
    name: raw.name,
    model: raw.model,
    createdAt: raw.createdAt,
    updatedAt: raw.updatedAt,
    msgCount: raw.msgCount ?? messages.length,
    messages,
    streaming: false,
    status: "idle",
    usage: normalizeUsage(raw.usage),
    lastRun: raw.lastRun,
    contextSummaryTokens: raw.contextSummaryTokens ?? 0,
  };
}

export const useSessionStore = create<SessionStore>((set, get) => ({
  sessions: [],
  activeSessionId: null,

  loadSessions: async () => {
    try {
      const list = await ListSessions();
      const sessions = await Promise.all(
        (list || []).map(async (s: any) => {
          const history = await GetHistory(s.id);
          return normalizeSession(s, (history || []).map(normalizeMessage));
        })
      );
      const current = get().activeSessionId;
      set({
        sessions,
        activeSessionId: current && sessions.some((s) => s.id === current) ? current : sessions[0]?.id ?? null,
      });
    } catch (e) {
      console.error("Failed to load sessions:", e);
    }
  },

  createSession: async (name: string) => {
    const result = await CreateSession(name);
    set((state) => ({
      sessions: [normalizeSession({ ...result, usage: emptyUsage() }), ...state.sessions],
      activeSessionId: result.id,
    }));
    return result.id;
  },

  deleteSession: (id: string) => {
    DeleteSession(id);
    set((state) => {
      const sessions = state.sessions.filter((s) => s.id !== id);
      return {
        sessions,
        activeSessionId: state.activeSessionId === id ? sessions[0]?.id ?? null : state.activeSessionId,
      };
    });
  },

  setActiveSession: (id: string) => set({ activeSessionId: id }),

  sendMessage: async (sessionId: string, message: string) => {
    set((state) => ({
      sessions: state.sessions.map((s) =>
        s.id === sessionId
          ? {
              ...s,
              messages: [...s.messages, { role: "user", content: message }],
              msgCount: s.messages.length + 1,
              streaming: true,
              status: "thinking",
              lastRun: undefined,
            }
          : s
      ),
    }));
    try {
      await SendMessage({ sessionId, message });
    } catch (e: any) {
      const detail = e?.message || String(e);
      get().appendToStream(sessionId, `\n\nRequest failed: ${detail}`);
      get().finishStream(sessionId);
      throw e;
    }
  },

  editMessage: async (sessionId: string, index: number, content: string) => {
    const trimmed = content.trim();
    if (!trimmed) return;
    set((state) => ({
      sessions: state.sessions.map((s) => {
        if (s.id !== sessionId) return s;
        const messages = s.messages.map((msg, i) => (i === index ? { ...msg, content: trimmed } : msg));
        return { ...s, messages };
      }),
    }));
    await UpdateMessage({ sessionId, index, content: trimmed });
  },

  deleteMessage: async (sessionId: string, index: number) => {
    set((state) => ({
      sessions: state.sessions.map((s) => {
        if (s.id !== sessionId) return s;
        const messages = s.messages.filter((_, i) => i !== index);
        return { ...s, messages, msgCount: messages.length };
      }),
    }));
    await DeleteMessageAt({ sessionId, index });
  },

  regenerateMessage: async (sessionId: string, index: number) => {
    let keepUntil = index;
    const session = get().sessions.find((s) => s.id === sessionId);
    if (session?.messages[index]?.role === "assistant") {
      keepUntil = index - 1;
    }
    set((state) => ({
      sessions: state.sessions.map((s) => {
        if (s.id !== sessionId) return s;
        const messages = s.messages.slice(0, Math.max(0, keepUntil) + 1);
        return {
          ...s,
          messages,
          msgCount: messages.length,
          streaming: true,
          status: "thinking" as const,
          lastRun: undefined,
        };
      }),
    }));
    try {
      await RegenerateMessage({ sessionId, index });
    } catch (e: any) {
      const detail = e?.message || String(e);
      get().appendToStream(sessionId, `\n\nRequest failed: ${detail}`);
      get().finishStream(sessionId);
      throw e;
    }
  },

  branchSession: async (sessionId: string, upToIndex: number) => {
    const result = await BranchSession({ sessionId, upToIndex, nameSuffix: "Branch" });
    const history = await GetHistory(result.id);
    const session = normalizeSession(
      {
        ...result,
        usage: emptyUsage(),
        msgCount: history?.length ?? 0,
      },
      (history || []).map(normalizeMessage)
    );
    set((state) => ({
      sessions: [session, ...state.sessions],
      activeSessionId: result.id,
    }));
    return result.id;
  },

  continueLastResponse: async (sessionId: string) => {
    set((state) => ({
      sessions: state.sessions.map((s) =>
        s.id === sessionId
          ? { ...s, streaming: true, status: "thinking" as const, lastRun: undefined }
          : s
      ),
    }));
    try {
      await ContinueLastResponse(sessionId);
    } catch (e: any) {
      const detail = e?.message || String(e);
      get().appendToStream(sessionId, `\n\nContinuation failed: ${detail}`);
      get().finishStream(sessionId);
      throw e;
    }
  },

  abortMessage: async (sessionId: string) => {
    await AbortMessage(sessionId);
    get().finishStream(sessionId);
  },

  appendToStream: (sessionId: string, content: string, reasoningContent = "") => {
    set((state) => ({
      sessions: state.sessions.map((s) => {
        if (s.id !== sessionId) return s;
        const msgs = [...s.messages];
        const last = msgs[msgs.length - 1];
        if (last && last.role === "assistant") {
          msgs[msgs.length - 1] = {
            ...last,
            content: last.content + content,
            reasoningContent: (last.reasoningContent || "") + reasoningContent,
          };
        } else {
          msgs.push({ role: "assistant", content, reasoningContent });
        }
        return { ...s, messages: msgs, msgCount: msgs.length, status: "streaming" as const };
      }),
    }));
  },

  finishStream: (sessionId: string, metrics?: RunMetrics) => {
    set((state) => ({
      sessions: state.sessions.map((s) => {
        if (s.id !== sessionId) return s;
        return {
          ...s,
          streaming: false,
          status: "idle" as const,
          msgCount: s.messages.length,
          lastRun: metrics || s.lastRun,
          usage: metrics ? addUsage(s.usage, normalizeUsage(metrics.usage)) : s.usage,
        };
      }),
    }));
  },

  setSessionStatus: (sessionId: string, status: Session["status"]) => {
    set((state) => ({
      sessions: state.sessions.map((s) => (s.id === sessionId ? { ...s, status } : s)),
    }));
  },

  addMessage: (sessionId: string, msg: Message) => {
    set((state) => ({
      sessions: state.sessions.map((s) =>
        s.id === sessionId ? { ...s, messages: [...s.messages, msg], msgCount: s.messages.length + 1 } : s
      ),
    }));
  },
}));
