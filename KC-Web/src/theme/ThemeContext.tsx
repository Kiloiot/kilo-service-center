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

import type { ThemeMode } from "@constants/app";
import { MS_PER_DAY, THEME_MODE, THEME_PREFERENCE } from "@constants/app";
import { APP_ERRORS } from "@constants/messages";

import { createAppTheme } from "./index";

interface ThemeContextType {
  mode: ThemeMode;
  toggleTheme: () => void;
}

// Create context
const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

const getCookie = (name: string): string | null => {
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(";").shift() || null;
  return null;
};

const setCookie = (name: string, value: string) => {
  const date = new Date();
  date.setTime(
    date.getTime() + THEME_PREFERENCE.COOKIE_MAX_AGE_DAYS * MS_PER_DAY,
  );
  const expires = `expires=${date.toUTCString()}`;
  document.cookie = `${name}=${value};${expires};path=${THEME_PREFERENCE.COOKIE_PATH}`;
};

// Theme Provider Component
interface ThemeProviderProps {
  children: ReactNode;
}

export const KCThemeProvider: React.FC<ThemeProviderProps> = ({ children }) => {
  // Initialize theme from cookie or system preference
  const [mode, setMode] = useState<ThemeMode>(() => {
    const savedMode = getCookie(THEME_PREFERENCE.COOKIE_NAME);
    if (savedMode === THEME_MODE.LIGHT || savedMode === THEME_MODE.DARK) {
      return savedMode;
    }
    if (
      window.matchMedia &&
      window.matchMedia(THEME_PREFERENCE.PREFERS_DARK_QUERY).matches
    ) {
      return THEME_MODE.DARK;
    }
    return THEME_MODE.LIGHT;
  });

  // Toggle theme function
  const toggleTheme = () => {
    const newMode =
      mode === THEME_MODE.LIGHT ? THEME_MODE.DARK : THEME_MODE.LIGHT;
    setMode(newMode);
    setCookie(THEME_PREFERENCE.COOKIE_NAME, newMode);
  };

  // Listen for system theme changes
  useEffect(() => {
    const mediaQuery = window.matchMedia(THEME_PREFERENCE.PREFERS_DARK_QUERY);
    const handleChange = (e: MediaQueryListEvent) => {
      if (!getCookie(THEME_PREFERENCE.COOKIE_NAME)) {
        setMode(e.matches ? THEME_MODE.DARK : THEME_MODE.LIGHT);
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
    throw new Error(APP_ERRORS.THEME_CONTEXT_REQUIRED);
  }
  return context;
};
