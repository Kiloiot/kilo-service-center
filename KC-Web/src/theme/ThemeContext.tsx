// Theme Context Provider with Cookie Persistence
// Uses createAppTheme factory for semantic token-based theming

import type { ReactNode } from "react";
import React, {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import CssBaseline from "@mui/material/CssBaseline";
import { ThemeProvider } from "@mui/material/styles";

import { createAppTheme } from "./index";

interface ThemeContextType {
  mode: "light" | "dark";
  toggleTheme: () => void;
}

// Create context
const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

// Cookie utilities
const THEME_COOKIE_NAME = "kc-web-theme-mode";

const getCookie = (name: string): string | null => {
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(";").shift() || null;
  return null;
};

const setCookie = (name: string, value: string, days: number = 365) => {
  const date = new Date();
  date.setTime(date.getTime() + days * 24 * 60 * 60 * 1000);
  const expires = `expires=${date.toUTCString()}`;
  document.cookie = `${name}=${value};${expires};path=/`;
};

// Theme Provider Component
interface ThemeProviderProps {
  children: ReactNode;
}

export const KCThemeProvider: React.FC<ThemeProviderProps> = ({ children }) => {
  // Initialize theme from cookie or system preference
  const [mode, setMode] = useState<"light" | "dark">(() => {
    const savedMode = getCookie(THEME_COOKIE_NAME);
    if (savedMode === "light" || savedMode === "dark") {
      return savedMode;
    }
    // Check system preference
    if (
      window.matchMedia &&
      window.matchMedia("(prefers-color-scheme: dark)").matches
    ) {
      return "dark";
    }
    return "light";
  });

  // Toggle theme function
  const toggleTheme = () => {
    const newMode = mode === "light" ? "dark" : "light";
    setMode(newMode);
    setCookie(THEME_COOKIE_NAME, newMode);
  };

  // Listen for system theme changes
  useEffect(() => {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = (e: MediaQueryListEvent) => {
      if (!getCookie(THEME_COOKIE_NAME)) {
        setMode(e.matches ? "dark" : "light");
      }
    };

    mediaQuery.addEventListener("change", handleChange);
    return () => mediaQuery.removeEventListener("change", handleChange);
  }, []);

  // Memoize theme creation to avoid recreation on every render
  const theme = useMemo(() => createAppTheme(mode), [mode]);

  return (
    <ThemeContext.Provider value={{ mode, toggleTheme }}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        {children}
      </ThemeProvider>
    </ThemeContext.Provider>
  );
};

// Custom hook to use theme context
// eslint-disable-next-line react-refresh/only-export-components
export const useThemeMode = () => {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error("useThemeMode must be used within a KCThemeProvider");
  }
  return context;
};
