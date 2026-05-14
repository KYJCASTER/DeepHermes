import { create } from "zustand";

export type ThemeMode = "comic" | "dark";

const storageKey = "deephermes.theme";

function initialTheme(): ThemeMode {
  const stored = localStorage.getItem(storageKey);
  if (stored === "dark" || stored === "comic") {
    return stored;
  }
  return "comic";
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
      const order: ThemeMode[] = ["comic", "dark"];
      const index = order.indexOf(get().theme);
      const next = order[(index + 1) % order.length];
      applyTheme(next);
      set({ theme: next });
    },
  };
});
