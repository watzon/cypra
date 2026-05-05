import {
  createContext,
  type PropsWithChildren,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import { actorThemeEndpoint } from "@/lib/utils";

export type ThemePreference = "system" | "dark" | "light";
type ResolvedMode = "dark" | "light";

interface ThemeContextValue {
  actorKind: "tenant" | "instance";
  preference: ThemePreference;
  mode: ResolvedMode;
  setPreference: (preference: ThemePreference) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);
const storageKey = "cypra.theme";

function resolveMode(preference: ThemePreference): ResolvedMode {
  if (preference !== "system") {
    return preference;
  }
  if (typeof window === "undefined") {
    return "dark";
  }
  return window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark";
}

export function ThemeProvider({ children }: PropsWithChildren) {
  const [actorKind] = useState<"tenant" | "instance">("instance");
  const [preference, setPreferenceState] = useState<ThemePreference>(() => {
    const storage = globalThis.localStorage;
    const stored = typeof storage.getItem === "function" ? storage.getItem(storageKey) : null;
    return stored === "light" || stored === "dark" || stored === "system" ? stored : "system";
  });
  const [mode, setMode] = useState<ResolvedMode>(() => resolveMode(preference));

  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: light)");
    const update = () => setMode(resolveMode(preference));
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, [preference]);

  useEffect(() => {
    document.documentElement.dataset.mode = mode;
  }, [mode]);

  const setPreference = (next: ThemePreference) => {
    setPreferenceState(next);
    localStorage.setItem(storageKey, next);
    void fetch(actorThemeEndpoint(actorKind), {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ metadata: { theme: next } }),
    }).catch(() => undefined);
  };

  const value = useMemo<ThemeContextValue>(
    () => ({ actorKind, preference, mode, setPreference }),
    [actorKind, mode, preference],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const value = useContext(ThemeContext);
  if (!value) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return value;
}
