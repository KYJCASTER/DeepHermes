import { create } from "zustand";

export type ThemeMode = "dark" | "light" | "anime";

const storageKey = "deephermes.theme";

function initialTheme(): ThemeMode {
  const stored = localStorage.getItem(storageKey);
  if (stored === "dark" || stored === "light" || stored === "anime") {
    return stored;
  }
  return window.matchMedia?.("(prefers-color-scheme: light)").matches ? "light" : "dark";
}

interface ThemeState {
  theme: ThemeMode;
  setTheme: (theme: ThemeMode) => void;
  toggleTheme: () => void;
}

function applyTheme(theme: ThemeMode) {
  document.documentElement.dataset.theme = theme;
  document.documentElement.style.colorScheme = theme === "dark" ? "dark" : "light";
  localStorage.setItem(storageKey, theme);
}

export const useThemeStore = create<ThemeState>((set, get) => {
  const theme = initialTheme();
  applyTheme(theme);

  return {
    theme,
    setTheme: (next) => {
      applyTheme(next);
      set({ theme: next });
    },
    toggleTheme: () => {
      const order: ThemeMode[] = ["light", "anime", "dark"];
      const index = order.indexOf(get().theme);
      const next = order[(index + 1) % order.length];
      applyTheme(next);
      set({ theme: next });
    },
  };
});
