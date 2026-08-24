export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#14110e",
        "ink-2": "#1d1814",
        paper: "#f4efe6",
        "paper-dim": "#c9bba8",
        brass: "#c45c26",
        "brass-2": "#e8a25a",
        moss: "#3d6b4f",
        rust: "#9b2c1a",
        line: "#2c261f",
      },
      fontFamily: {
        display: ['Fraunces', 'Songti SC', 'serif'],
        sans: ['IBM Plex Sans', 'Source Han Sans SC', 'sans-serif'],
        mono: ['IBM Plex Mono', 'monospace'],
      },
    },
  },
};
