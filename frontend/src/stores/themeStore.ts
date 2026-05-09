import { create } from "zustand";

export type ThemeMode = "dark" | "light";

const storageKey = "deephermes.theme";

function initialTheme(): ThemeMode {
  const stored = localStorage.getItem(storageKey);
  if (stored === "dark" || stored === "light") {
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
  document.documentElement.style.colorScheme = theme;
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
      const next = get().theme === "dark" ? "light" : "dark";
      applyTheme(next);
      set({ theme: next });
    },
  };
});
