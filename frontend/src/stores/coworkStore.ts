import { create } from "zustand";
import { SpawnSubAgent, CancelSubAgent, GetSubAgents } from "../lib/wails";

export interface SubAgent {
  id: string;
  name: string;
  agentType: string;
  status: "pending" | "running" | "done" | "failed";
  createdAt: string;
  result: string;
  parentSessionId: string;
}

interface CoworkStore {
  subAgents: SubAgent[];
  isOpen: boolean;

  togglePanel: () => void;
  spawnSubAgent: (parentSessionId: string, name: string, agentType: string, task: string) => Promise<void>;
  cancelSubAgent: (id: string) => void;
  refreshSubAgents: () => Promise<void>;
  updateSubAgent: (id: string, updates: Partial<SubAgent>) => void;
  upsertSubAgent: (agent: Partial<SubAgent> & { id: string }) => void;
}

export const useCoworkStore = create<CoworkStore>((set, get) => ({
  subAgents: [],
  isOpen: false,

  togglePanel: () => set((s) => ({ isOpen: !s.isOpen })),

  spawnSubAgent: async (parentSessionId, name, agentType, task) => {
    const id = await SpawnSubAgent({ parentSessionId, name, agentType, task });
    set((state) => ({
      subAgents: [
        {
          id,
          name,
          agentType,
          status: "running",
          createdAt: new Date().toISOString(),
          result: "",
          parentSessionId,
        },
        ...state.subAgents,
      ],
    }));
  },

  cancelSubAgent: (id: string) => {
    CancelSubAgent(id);
    set((state) => ({
      subAgents: state.subAgents.map((sa) =>
        sa.id === id ? { ...sa, status: "failed" as const } : sa
      ),
    }));
  },

  refreshSubAgents: async () => {
    const list = await GetSubAgents();
    set({
      subAgents: list.map((sa: any) => ({
        ...sa,
        status: sa.status as SubAgent["status"],
        parentSessionId: "",
      })),
    });
  },

  updateSubAgent: (id, updates) => {
    set((state) => ({
      subAgents: state.subAgents.map((sa) => (sa.id === id ? { ...sa, ...updates } : sa)),
    }));
  },

  upsertSubAgent: (agent) => {
    set((state) => {
      const exists = state.subAgents.some((sa) => sa.id === agent.id);
      if (exists) {
        return {
          subAgents: state.subAgents.map((sa) =>
            sa.id === agent.id ? { ...sa, ...agent } : sa
          ),
        };
      }
      return {
        subAgents: [
          {
            id: agent.id,
            name: agent.name ?? "Agent",
            agentType: agent.agentType ?? "explore",
            status: (agent.status as SubAgent["status"]) ?? "running",
            createdAt: agent.createdAt ?? new Date().toISOString(),
            result: agent.result ?? "",
            parentSessionId: agent.parentSessionId ?? "",
          },
          ...state.subAgents,
        ],
      };
    });
  },
}));
