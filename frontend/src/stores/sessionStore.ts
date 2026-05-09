import { create } from "zustand";
import { AbortMessage, CreateSession, DeleteSession, ListSessions, SendMessage } from "../lib/wails";

export interface Message {
  role: string;
  content: string;
}

export interface Session {
  id: string;
  name: string;
  model: string;
  createdAt: string;
  msgCount: number;
  messages: Message[];
  streaming: boolean;
  status: "idle" | "thinking" | "streaming" | "executing";
}

interface SessionStore {
  sessions: Session[];
  activeSessionId: string | null;

  loadSessions: () => Promise<void>;
  createSession: (name: string) => Promise<string>;
  deleteSession: (id: string) => void;
  setActiveSession: (id: string) => void;
  sendMessage: (sessionId: string, message: string) => Promise<void>;
  abortMessage: (sessionId: string) => Promise<void>;
  appendToStream: (sessionId: string, content: string) => void;
  finishStream: (sessionId: string) => void;
  setSessionStatus: (sessionId: string, status: Session["status"]) => void;
  addMessage: (sessionId: string, msg: Message) => void;
}

export const useSessionStore = create<SessionStore>((set, get) => ({
  sessions: [],
  activeSessionId: null,

  loadSessions: async () => {
    try {
      const list = await ListSessions();
      set({
        sessions: list.map((s: any) => ({
          ...s,
          messages: [],
          streaming: false,
          status: "idle" as const,
        })),
      });
    } catch (e) {
      console.error("Failed to load sessions:", e);
    }
  },

  createSession: async (name: string) => {
    const result = await CreateSession(name);
    set((state) => ({
      sessions: [
        {
          ...result,
          msgCount: 0,
          messages: [],
          streaming: false,
          status: "idle" as const,
        },
        ...state.sessions,
      ],
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
              streaming: true,
              status: "thinking",
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

  abortMessage: async (sessionId: string) => {
    await AbortMessage(sessionId);
    get().finishStream(sessionId);
  },

  appendToStream: (sessionId: string, content: string) => {
    set((state) => ({
      sessions: state.sessions.map((s) => {
        if (s.id !== sessionId) return s;
        const msgs = [...s.messages];
        const last = msgs[msgs.length - 1];
        if (last && last.role === "assistant") {
          msgs[msgs.length - 1] = { ...last, content: last.content + content };
        } else {
          msgs.push({ role: "assistant", content });
        }
        return { ...s, messages: msgs, status: "streaming" as const };
      }),
    }));
  },

  finishStream: (sessionId: string) => {
    set((state) => ({
      sessions: state.sessions.map((s) =>
        s.id === sessionId ? { ...s, streaming: false, status: "idle" as const } : s
      ),
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
        s.id === sessionId ? { ...s, messages: [...s.messages, msg] } : s
      ),
    }));
  },
}));
