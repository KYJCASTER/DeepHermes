import { create } from "zustand";

const sidebarKey = "deephermes.layout.sidebarWidth";
const filePanelKey = "deephermes.layout.filePanelWidth";

function readNumber(key: string, fallback: number) {
  const value = Number(localStorage.getItem(key));
  return Number.isFinite(value) && value > 0 ? value : fallback;
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

interface LayoutState {
  sidebarWidth: number;
  filePanelWidth: number;
  setSidebarWidth: (width: number) => void;
  setFilePanelWidth: (width: number) => void;
}

export const useLayoutStore = create<LayoutState>((set) => ({
  sidebarWidth: readNumber(sidebarKey, 272),
  filePanelWidth: readNumber(filePanelKey, 320),
  setSidebarWidth: (width) => {
    const next = clamp(width, 220, 420);
    localStorage.setItem(sidebarKey, String(next));
    set({ sidebarWidth: next });
  },
  setFilePanelWidth: (width) => {
    const next = clamp(width, 240, 520);
    localStorage.setItem(filePanelKey, String(next));
    set({ filePanelWidth: next });
  },
}));
