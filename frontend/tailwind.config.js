/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: "rgb(var(--color-bg) / <alpha-value>)",
        surface: "rgb(var(--color-surface) / <alpha-value>)",
        panel: "rgb(var(--color-panel) / <alpha-value>)",
        border: "rgb(var(--color-border) / <alpha-value>)",
        text: "rgb(var(--color-text) / <alpha-value>)",
        accent: "rgb(var(--color-accent) / <alpha-value>)",
        "accent-alt": "rgb(var(--color-accent-alt) / <alpha-value>)",
        green: "rgb(var(--color-green) / <alpha-value>)",
        yellow: "rgb(var(--color-yellow) / <alpha-value>)",
        red: "rgb(var(--color-red) / <alpha-value>)",
        dim: "rgb(var(--color-dim) / <alpha-value>)",
      },
    },
  },
  plugins: [],
};
