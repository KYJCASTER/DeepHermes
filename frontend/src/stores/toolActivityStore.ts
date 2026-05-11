import { create } from "zustand";

export type ToolActivityStatus = "running" | "done" | "failed";

export interface ToolActivity {
  id: string;
  sessionId?: string;
  toolName: string;
  arguments: string;
  risk: string;
  content?: string;
  error?: string;
  status: ToolActivityStatus;
  startedAt: string;
  finishedAt?: string;
  rollbackAvailable?: boolean;
  rollbackPath?: string;
  rollbackDone?: boolean;
  rollbackMessage?: string;
}

interface ToolActivityStore {
  items: ToolActivity[];
  isOpen: boolean;
  addCall: (payload: Partial<ToolActivity> & { id: string; toolName: string }) => void;
  finishCall: (payload: {
    toolCallId?: string;
    id?: string;
    name?: string;
    content?: string;
    error?: string;
    success?: boolean;
    risk?: string;
    rollbackAvailable?: boolean;
    rollbackPath?: string;
  }) => void;
  markRolledBack: (id: string, message: string) => void;
  clear: () => void;
  togglePanel: () => void;
  exportAuditLog: () => string;
}

export const useToolActivityStore = create<ToolActivityStore>((set, get) => ({
  items: [],
  isOpen: false,

  addCall: (payload) => {
    set((state) => {
      const exists = state.items.some((item) => item.id === payload.id);
      const next: ToolActivity = {
        id: payload.id,
        sessionId: payload.sessionId,
        toolName: payload.toolName,
        arguments: payload.arguments || "{}",
        risk: payload.risk || "unknown",
        status: "running",
        startedAt: new Date().toISOString(),
      };
      const items = exists ? state.items.map((item) => (item.id === payload.id ? { ...item, ...next } : item)) : [next, ...state.items];
      return { items: items.slice(0, 80) };
    });
  },

  finishCall: (payload) => {
    const id = payload.toolCallId || payload.id || "";
    if (!id) return;
    set((state) => {
      const exists = state.items.some((item) => item.id === id);
      const update = {
        content: payload.content,
        error: payload.error,
        risk: payload.risk || "unknown",
        status: payload.success === false ? "failed" as const : "done" as const,
        finishedAt: new Date().toISOString(),
        rollbackAvailable: payload.rollbackAvailable,
        rollbackPath: payload.rollbackPath,
      };
      if (!exists) {
        return {
          items: [
            {
              id,
              toolName: payload.name || "tool",
              arguments: "{}",
              startedAt: new Date().toISOString(),
              ...update,
            },
            ...state.items,
          ].slice(0, 80),
        };
      }
      return {
        items: state.items.map((item) => (item.id === id ? { ...item, ...update, risk: update.risk || item.risk } : item)),
      };
    });
  },

  markRolledBack: (id, message) => {
    set((state) => ({
      items: state.items.map((item) =>
        item.id === id
          ? {
              ...item,
              rollbackAvailable: false,
              rollbackDone: true,
              rollbackMessage: message,
            }
          : item
      ),
    }));
  },

  clear: () => set({ items: [] }),
  togglePanel: () => set((state) => ({ isOpen: !state.isOpen })),

  exportAuditLog: () => {
    const { items } = get();
    const lines = items.map((item) => {
      const status = item.rollbackDone ? "rolled_back" : item.status;
      return [item.startedAt, status, item.toolName, item.risk, item.arguments].join("\t");
    });
    return ["timestamp\tstatus\ttool\trisk\targuments", ...lines].join("\n");
  },
}));
