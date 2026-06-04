import { useState } from "react";

type Theme = "dark" | "light";

const STORAGE_KEY = "lm-theme";

function getInitial(): Theme {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    if (v === "dark" || v === "light") return v;
  } catch {}
  return window.matchMedia?.("(prefers-color-scheme: light)").matches ? "light" : "dark";
}

/** Returns [theme, toggle, themeClass] — add themeClass to .lm-root */
export function useTheme(): [Theme, () => void, string] {
  const [theme, setTheme] = useState<Theme>(getInitial);

  const toggle = () => {
    setTheme((prev) => {
      const next = prev === "dark" ? "light" : "dark";
      try { localStorage.setItem(STORAGE_KEY, next); } catch {}
      return next;
    });
  };

  const cls = theme === "light" ? "lm-light" : "lm-dark";

  return [theme, toggle, cls];
}
