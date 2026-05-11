// KC-Web Theme Configuration
// Based on Kilo Platform Design System
// Centralized theme management with semantic tokens
//
// GOVERNANCE: All components MUST use semantic tokens from this module.
// No inline hex/RGB colors, no hardcoded spacing values.
// Use theme.palette.*, theme.spacing(), theme.typography.* exclusively.

import type { Theme, ThemeOptions } from "@mui/material/styles";
import { alpha, createTheme } from "@mui/material/styles";

// =============================================================================
// SEMANTIC TOKEN DEFINITIONS
// =============================================================================

/**
 * Semantic spacing scale (base unit: 4px)
 * Use via theme.spacing() or semanticTokens.spacing.*
 */
export const semanticSpacing = {
  xxs: 0.5, // 2px
  xs: 1, // 4px
  sm: 2, // 8px
  md: 4, // 16px
  lg: 6, // 24px
  xl: 8, // 32px
  xxl: 12, // 48px
} as const;

/**
 * Component-specific spacing tokens
 * Use via componentSpacing.{component}.{property}
 */
export const componentSpacing = {
  pagination: {
    containerMt: 2, // Container top margin (theme.spacing units)
    containerGap: 2, // Main flex gap (theme.spacing units)
    controlGap: 1, // Inner control gap (theme.spacing units)
    selectMinWidth: 80, // Page size select min width (px)
  },
} as const;

/**
 * Semantic z-index scale
 * Use via semanticTokens.zIndex.* or theme.zIndex.*
 */
export const semanticZIndex = {
  drawer: 1200,
  modal: 1300,
  snackbar: 1400,
  tooltip: 1500,
} as const;

/**
 * Semantic elevation scale (MUI shadow indices)
 * Use via theme.shadows[semanticTokens.elevation.*]
 */
export const semanticElevation = {
  none: 0,
  low: 1,
  medium: 4,
  high: 8,
  highest: 16,
} as const;

/**
 * Typography configuration
 * Font families: Roboto (display), Roboto Mono (code)
 */
export const semanticTypography = {
  fontFamily: {
    display: '"Roboto", "Helvetica", "Arial", sans-serif',
    code: '"Roboto Mono", "Courier New", monospace',
  },
  fontWeight: {
    regular: 400,
    medium: 500,
    bold: 700,
  },
  fontSize: {
    xs: "12px",
    sm: "14px",
    md: "16px",
    lg: "20px",
    xl: "24px",
    xxl: "32px",
  },
} as const;

/**
 * Neutral color scale (theme-independent)
 * Monotonic from light to dark - for backgrounds, borders, disabled states
 */
export const neutralScale = {
  50: "#FDFDFF", // White
  100: "#F6F6F6", // Light Gray
  200: "#F2F9FF", // Background (slight blue tint)
  300: "#DFDFDF", // Medium Gray
  400: "#C1CCD1", // Disabled
  500: "#909090", // Medium Dark Gray
  600: "#2D2D2D", // Dark Gray
  700: "#1E2E3E", // Primary (mid-dark)
  800: "#0C0C0C", // Black
  900: "#0C0C0C", // Black (intentional ceiling)
} as const;

// =============================================================================
// KILO COLOR PALETTE (backward compatibility + semantic mapping)
// =============================================================================

/**
 * Kilo Color Palette - Core brand colors
 * IMPORTANT: Do not use these directly in components.
 * Use theme.palette.* semantic tokens instead.
 */
export const kiloColors = {
  primary: {
    main: "#1E2E3E", // New primary
    accent: "#1E2E3E", // Same as main per spec
    accentHover: "#3162BD", // Button hover light
    accentLight: "rgba(30, 46, 62, 0.1)",
    accentLightHover: "rgba(30, 46, 62, 0.2)",
  },
  background: {
    light: {
      primary: "#F6F6F6", // Light Gray
      default: "#FDFDFF", // White
      paper: "#F2F9FF", // Background
    },
    dark: {
      primary: "#0C0C0C", // Black
      default: "#2D2D2D", // Dark Gray
      paper: "rgba(12, 12, 12, 0.8)",
    },
  },
  lightMode: {
    darkShades: {
      primary: "#F6F6F6", // Light Gray
      secondary: "#FDFDFF", // White
      ternary: "#DFDFDF", // Medium Gray
      quaternary: "#DFDFDF",
      fifth: "#F6F6F6",
    },
    lightShades: {
      primary: "#0C0C0C", // Black
      secondary: "#2D2D2D", // Dark Gray
      ternary: "#909090", // Medium Dark Gray
      quaternary: "#909090",
    },
    text: {
      primary: "#0C0C0C", // Black
      secondary: "#909090", // Medium Dark Gray
    },
  },
  darkMode: {
    darkShades: {
      primary: "#0C0C0C", // Black
      secondary: "#2D2D2D", // Dark Gray
      ternary: "#2D2D2D",
      quaternary: "#909090",
      fifth: "#909090",
    },
    lightShades: {
      primary: "#FDFDFF", // White
      secondary: "#F6F6F6", // Light Gray
      ternary: "#DFDFDF", // Medium Gray
      quaternary: "#909090", // Medium Dark Gray
    },
    text: {
      primary: "#F6F6F6", // Light Gray (readable on dark)
      secondary: "#909090", // Medium Dark Gray
    },
  },
  alerts: {
    success: {
      light: "#4CAF50", // Success
      dark: "#4CAF50",
    },
    warning: {
      light: "#FFC107", // Warning
      dark: "#FFC107",
    },
    error: "#E53935", // Error
    info: {
      light: "#0277BD", // Darker info for light mode (better contrast on #F6F6F6)
      dark: "#81D4FA", // Light info for dark mode
    },
  },
  // Terminal colors - aligned with new neutrals
  terminal: {
    light: {
      background: "#F6F6F6",
      backgroundAlt: "#DFDFDF",
      text: "#0C0C0C",
      timestamp: "#1E2E3E", // Primary - good contrast on light
      success: "#4CAF50",
      warning: "#FFC107",
      error: "#E53935",
      info: "#0277BD", // Darker for light mode contrast
      data: "#1565C0",
      identifier: "#3162BD",
      packet: "#1565C0",
      header: "#4CAF50",
    },
    dark: {
      background: "#0C0C0C",
      backgroundAlt: "#2D2D2D",
      text: "#F6F6F6",
      timestamp: "#81D4FA", // Light info for dark mode
      success: "#4CAF50",
      warning: "#FFC107",
      error: "#E53935",
      info: "#81D4FA",
      data: "#81D4FA",
      identifier: "#3162BD",
      packet: "#FFC107",
      header: "#4CAF50",
    },
  },
  borders: {
    primary: "rgba(145, 145, 145, 0.1)", // Based on Medium Dark Gray
    secondary: "rgba(145, 145, 145, 0.3)",
    ternary: {
      light: "#DFDFDF", // Medium Gray
      dark: "#2D2D2D", // Dark Gray
    },
  },
  // Additional colors for labels, buttons, disabled states
  additional: {
    buttonHoverLight: "#3162BD",
    buttonHoverDark: "#02111F",
    disabled: "#C1CCD1",
    labelSale: "#FF8080",
    labelNew: "#54EE5B",
    labelHit: "#678AFB",
  },
};

// =============================================================================
// SEMANTIC TOKEN STRUCTURES (per-theme)
// =============================================================================

interface SemanticPalette {
  primary: {
    main: string;
    light: string;
    dark: string;
    contrastText: string;
  };
  secondary: {
    main: string;
    light: string;
    dark: string;
    contrastText: string;
  };
  background: {
    default: string;
    paper: string;
    elevated: string;
  };
  surface: {
    main: string;
    variant: string;
    inverse: string;
  };
  border: {
    default: string;
    subtle: string;
    strong: string;
    focus: string;
  };
  text: {
    primary: string;
    secondary: string;
    disabled: string;
  };
}

interface SemanticStatus {
  success: {
    main: string;
    light: string;
    dark: string;
    contrastText: string;
  };
  warning: {
    main: string;
    light: string;
    dark: string;
    contrastText: string;
  };
  error: {
    main: string;
    light: string;
    dark: string;
    contrastText: string;
  };
  info: {
    main: string;
    light: string;
    dark: string;
    contrastText: string;
  };
}

export interface SemanticTokens {
  palette: SemanticPalette;
  status: SemanticStatus;
  neutral: typeof neutralScale;
  spacing: typeof semanticSpacing;
  zIndex: typeof semanticZIndex;
  elevation: typeof semanticElevation;
  typography: typeof semanticTypography;
}

/**
 * Light theme semantic tokens
 */
const lightSemanticTokens: SemanticTokens = {
  palette: {
    primary: {
      main: kiloColors.primary.main,
      light: kiloColors.primary.accent,
      dark: kiloColors.primary.accentHover,
      contrastText: "#FDFDFF", // White - readable on #1E2E3E
    },
    secondary: {
      main: kiloColors.lightMode.lightShades.secondary,
      light: kiloColors.lightMode.darkShades.secondary,
      dark: kiloColors.lightMode.lightShades.ternary,
      contrastText: kiloColors.lightMode.text.primary,
    },
    background: {
      default: kiloColors.background.light.default,
      paper: kiloColors.background.light.primary,
      elevated: kiloColors.lightMode.darkShades.secondary,
    },
    surface: {
      main: kiloColors.lightMode.darkShades.primary,
      variant: kiloColors.lightMode.darkShades.ternary,
      inverse: kiloColors.darkMode.darkShades.primary,
    },
    border: {
      default: kiloColors.borders.primary,
      subtle: "rgba(145, 145, 145, 0.05)",
      strong: kiloColors.borders.secondary,
      focus: kiloColors.primary.accent,
    },
    text: {
      primary: kiloColors.lightMode.text.primary,
      secondary: kiloColors.lightMode.text.secondary,
      disabled: kiloColors.additional.disabled,
    },
  },
  status: {
    success: {
      main: kiloColors.alerts.success.light,
      light: alpha(kiloColors.alerts.success.light, 0.1),
      dark: "#2E7D32",
      contrastText: "#FDFDFF", // White
    },
    warning: {
      main: kiloColors.alerts.warning.light,
      light: alpha(kiloColors.alerts.warning.light, 0.1),
      dark: "#F57C00",
      contrastText: "#0C0C0C", // Black - readable on yellow
    },
    error: {
      main: kiloColors.alerts.error,
      light: alpha(kiloColors.alerts.error, 0.1),
      dark: "#C62828",
      contrastText: "#FDFDFF", // White
    },
    info: {
      main: kiloColors.alerts.info.light, // Darker info for light mode
      light: alpha(kiloColors.alerts.info.light, 0.1),
      dark: "#01579B",
      contrastText: "#FDFDFF", // White - info is dark blue in light mode
    },
  },
  neutral: neutralScale,
  spacing: semanticSpacing,
  zIndex: semanticZIndex,
  elevation: semanticElevation,
  typography: semanticTypography,
};

/**
 * Dark theme semantic tokens
 */
const darkSemanticTokens: SemanticTokens = {
  palette: {
    primary: {
      main: kiloColors.primary.main,
      light: kiloColors.primary.accent,
      dark: kiloColors.primary.accentHover,
      contrastText: "#FDFDFF", // White
    },
    secondary: {
      main: kiloColors.darkMode.lightShades.secondary,
      light: kiloColors.darkMode.lightShades.primary,
      dark: kiloColors.darkMode.lightShades.ternary,
      contrastText: kiloColors.darkMode.darkShades.primary,
    },
    background: {
      default: kiloColors.background.dark.default,
      paper: kiloColors.background.dark.primary,
      elevated: kiloColors.darkMode.darkShades.ternary,
    },
    surface: {
      main: kiloColors.darkMode.darkShades.secondary,
      variant: kiloColors.darkMode.darkShades.quaternary,
      inverse: kiloColors.lightMode.darkShades.primary,
    },
    border: {
      default: kiloColors.borders.primary,
      subtle: "rgba(145, 145, 145, 0.05)",
      strong: kiloColors.borders.secondary,
      focus: kiloColors.primary.accent,
    },
    text: {
      primary: kiloColors.darkMode.text.primary,
      secondary: kiloColors.darkMode.text.secondary,
      disabled: kiloColors.additional.disabled,
    },
  },
  status: {
    success: {
      main: kiloColors.alerts.success.dark,
      light: alpha(kiloColors.alerts.success.dark, 0.15),
      dark: kiloColors.alerts.success.light,
      contrastText: "#0C0C0C", // Black on bright green
    },
    warning: {
      main: kiloColors.alerts.warning.dark,
      light: alpha(kiloColors.alerts.warning.dark, 0.15),
      dark: kiloColors.alerts.warning.light,
      contrastText: "#0C0C0C", // Black on yellow
    },
    error: {
      main: kiloColors.alerts.error,
      light: alpha(kiloColors.alerts.error, 0.15),
      dark: "#EF5350",
      contrastText: "#FDFDFF", // White on red
    },
    info: {
      main: kiloColors.alerts.info.dark, // Light info for dark mode
      light: alpha(kiloColors.alerts.info.dark, 0.15),
      dark: "#4FC3F7",
      contrastText: "#0C0C0C", // Black - info #81D4FA is light
    },
  },
  neutral: neutralScale,
  spacing: semanticSpacing,
  zIndex: semanticZIndex,
  elevation: semanticElevation,
  typography: semanticTypography,
};

// =============================================================================
// THEME FACTORY
// =============================================================================

/**
 * Get semantic tokens for the specified mode
 */
export const getSemanticTokens = (mode: "light" | "dark"): SemanticTokens => {
  return mode === "light" ? lightSemanticTokens : darkSemanticTokens;
};

/**
 * Base typography configuration (shared between themes)
 */
const baseTypography = {
  fontFamily: semanticTypography.fontFamily.display,
  // Custom mono font family for EUI displays, code, etc.
  monoFontFamily: semanticTypography.fontFamily.code,
  h1: {
    fontFamily: semanticTypography.fontFamily.display,
    fontSize: "40px",
    lineHeight: "48px",
    fontWeight: semanticTypography.fontWeight.medium,
  },
  h2: {
    fontFamily: semanticTypography.fontFamily.display,
    fontSize: "32px",
    lineHeight: "40px",
    fontWeight: semanticTypography.fontWeight.medium,
  },
  h3: {
    fontFamily: semanticTypography.fontFamily.display,
    fontSize: "24px",
    lineHeight: "32px",
    fontWeight: semanticTypography.fontWeight.medium,
  },
  h4: {
    fontFamily: semanticTypography.fontFamily.display,
    fontSize: "20px",
    lineHeight: "28px",
    fontWeight: semanticTypography.fontWeight.medium,
  },
  h5: {
    fontFamily: semanticTypography.fontFamily.display,
    fontSize: "16px",
    lineHeight: "24px",
    fontWeight: semanticTypography.fontWeight.medium,
  },
  h6: {
    fontFamily: semanticTypography.fontFamily.display,
    fontSize: "14px",
    lineHeight: "20px",
    fontWeight: semanticTypography.fontWeight.medium,
  },
  body1: {
    fontFamily: semanticTypography.fontFamily.display,
    fontSize: "14px",
    lineHeight: "20px",
    fontWeight: semanticTypography.fontWeight.regular,
  },
  body2: {
    fontFamily: semanticTypography.fontFamily.display,
    fontSize: "12px",
    lineHeight: "16px",
    fontWeight: semanticTypography.fontWeight.regular,
  },
  button: {
    fontFamily: semanticTypography.fontFamily.code,
    fontSize: "16px",
    lineHeight: "20px",
    fontWeight: semanticTypography.fontWeight.medium,
    textTransform: "none" as const,
  },
  caption: {
    fontFamily: semanticTypography.fontFamily.display,
    fontSize: "12px",
    lineHeight: "16px",
    fontWeight: semanticTypography.fontWeight.regular,
  },
  overline: {
    fontFamily: semanticTypography.fontFamily.code,
    fontSize: "12px",
    lineHeight: "16px",
    fontWeight: semanticTypography.fontWeight.medium,
    textTransform: "uppercase" as const,
  },
};

/**
 * Base spacing (4px unit)
 */
const spacing = 4;

/**
 * Breakpoints including Kilo-specific ones
 */
const breakpoints = {
  values: {
    xs: 0,
    sm: 600,
    md: 900,
    xm: 1024, // Kilo custom
    lg: 1200,
    xl: 1536,
    xxl: 1920, // Kilo custom
  },
};

/**
 * Create component overrides based on semantic tokens
 */
const createComponentOverrides = (
  tokens: SemanticTokens,
  mode: "light" | "dark",
): ThemeOptions["components"] => ({
  MuiButton: {
    styleOverrides: {
      root: {
        borderRadius: 8,
        padding: "12px 24px",
        transition: "all 0.2s ease-in-out",
        "&:hover": {
          transform: "translateY(-1px)",
          boxShadow:
            mode === "light"
              ? "0 4px 12px rgba(0, 0, 0, 0.1)"
              : "0 4px 12px rgba(0, 0, 0, 0.3)",
        },
        "&.Mui-disabled": {
          backgroundColor: alpha(kiloColors.additional.disabled, 0.3),
          color: tokens.palette.text.disabled,
        },
      },
      contained: {
        backgroundColor: tokens.palette.primary.main,
        color: tokens.palette.primary.contrastText,
        "&:hover": {
          // Use action.hover color for contained buttons
          backgroundColor:
            mode === "light"
              ? kiloColors.additional.buttonHoverLight
              : kiloColors.additional.buttonHoverDark,
        },
      },
      outlined: {
        borderColor: tokens.palette.border.strong,
        color: tokens.palette.text.primary,
        "&:hover": {
          borderColor: tokens.palette.primary.main,
          backgroundColor: alpha(tokens.palette.primary.main, 0.1),
        },
      },
      text: {
        color: tokens.palette.primary.main,
        "&:hover": {
          backgroundColor: alpha(tokens.palette.primary.main, 0.1),
        },
      },
    },
  },
  MuiPaper: {
    styleOverrides: {
      root: {
        backgroundImage: "none",
        borderRadius: 12,
        border: `1px solid ${tokens.palette.border.default}`,
        ...(mode === "dark" && {
          backgroundColor: kiloColors.background.dark.paper,
        }),
      },
    },
  },
  MuiCard: {
    styleOverrides: {
      root: {
        borderRadius: 16,
        boxShadow:
          mode === "light"
            ? "0 2px 8px rgba(0, 0, 0, 0.05)"
            : "0 2px 8px rgba(0, 0, 0, 0.3)",
        transition: "all 0.3s ease-in-out",
        "&:hover": {
          transform: "translateY(-2px)",
          boxShadow:
            mode === "light"
              ? "0 8px 24px rgba(0, 0, 0, 0.1)"
              : "0 8px 24px rgba(0, 0, 0, 0.5)",
        },
      },
    },
  },
  MuiTextField: {
    styleOverrides: {
      root: {
        "& .MuiOutlinedInput-root": {
          borderRadius: 8,
          "& fieldset": {
            borderColor: tokens.palette.border.strong,
          },
          "&:hover fieldset": {
            borderColor: tokens.palette.primary.light,
          },
          "&.Mui-focused fieldset": {
            borderColor: tokens.palette.border.focus,
          },
        },
      },
    },
  },
  MuiChip: {
    styleOverrides: {
      root: {
        borderRadius: 6,
        fontFamily: semanticTypography.fontFamily.code,
        fontSize: "12px",
      },
    },
  },
  MuiAlert: {
    styleOverrides: {
      root: {
        borderRadius: 8,
      },
      // Light mode: use darker text colors for better visibility
      standardSuccess: {
        ...(mode === "light" && {
          color: tokens.status.success.dark,
          "& .MuiAlert-icon": {
            color: tokens.status.success.dark,
          },
        }),
      },
      standardError: {
        ...(mode === "light" && {
          color: tokens.status.error.dark,
          "& .MuiAlert-icon": {
            color: tokens.status.error.dark,
          },
        }),
      },
      standardWarning: {
        ...(mode === "light" && {
          color: tokens.status.warning.dark,
          "& .MuiAlert-icon": {
            color: tokens.status.warning.dark,
          },
        }),
      },
      standardInfo: {
        ...(mode === "light" && {
          color: tokens.status.info.dark,
          "& .MuiAlert-icon": {
            color: tokens.status.info.dark,
          },
        }),
      },
    },
  },
});

/**
 * Create a complete MUI theme from semantic tokens
 *
 * @param mode - 'light' or 'dark' theme mode
 * @returns Complete MUI Theme object
 *
 * @example
 * const theme = createAppTheme('dark');
 * <ThemeProvider theme={theme}>...</ThemeProvider>
 */
export const createAppTheme = (mode: "light" | "dark"): Theme => {
  const tokens = getSemanticTokens(mode);

  return createTheme({
    palette: {
      mode,
      primary: {
        main: tokens.palette.primary.main,
        light: tokens.palette.primary.light,
        dark: tokens.palette.primary.dark,
        contrastText: tokens.palette.primary.contrastText,
      },
      secondary: {
        main: tokens.palette.secondary.main,
        light: tokens.palette.secondary.light,
        dark: tokens.palette.secondary.dark,
        contrastText: tokens.palette.secondary.contrastText,
      },
      background: {
        default: tokens.palette.background.default,
        paper: tokens.palette.background.paper,
      },
      text: {
        primary: tokens.palette.text.primary,
        secondary: tokens.palette.text.secondary,
        disabled: tokens.palette.text.disabled,
      },
      success: {
        main: tokens.status.success.main,
        light: tokens.status.success.light,
        dark: tokens.status.success.dark,
        contrastText: tokens.status.success.contrastText,
      },
      warning: {
        main: tokens.status.warning.main,
        light: tokens.status.warning.light,
        dark: tokens.status.warning.dark,
        contrastText: tokens.status.warning.contrastText,
      },
      error: {
        main: tokens.status.error.main,
        light: tokens.status.error.light,
        dark: tokens.status.error.dark,
        contrastText: tokens.status.error.contrastText,
      },
      info: {
        main: tokens.status.info.main,
        light: tokens.status.info.light,
        dark: tokens.status.info.dark,
        contrastText: tokens.status.info.contrastText,
      },
      divider: tokens.palette.border.default,
      // Action colors for global propagation
      action: {
        hover:
          mode === "light"
            ? kiloColors.additional.buttonHoverLight
            : kiloColors.additional.buttonHoverDark,
        disabled: kiloColors.additional.disabled,
        disabledBackground: alpha(kiloColors.additional.disabled, 0.3),
      },
    },
    typography: baseTypography,
    spacing,
    breakpoints,
    shape: {
      borderRadius: 8,
    },
    zIndex: tokens.zIndex,
    components: createComponentOverrides(tokens, mode),
  });
};

// =============================================================================
// BACKWARD COMPATIBILITY EXPORTS
// =============================================================================

/**
 * Pre-built light theme (for backward compatibility)
 * @deprecated Use createAppTheme('light') instead
 */
export const lightTheme: Theme = createAppTheme("light");

/**
 * Pre-built dark theme (for backward compatibility)
 * @deprecated Use createAppTheme('dark') instead
 */
export const darkTheme: Theme = createAppTheme("dark");

/**
 * Get theme by mode (for backward compatibility)
 * @deprecated Use createAppTheme(mode) instead
 */
export const getTheme = (mode: "light" | "dark"): Theme => {
  return createAppTheme(mode);
};

/**
 * Spacing helper (for backward compatibility)
 * @deprecated Use theme.spacing(units) instead
 */
export const getSpacing = (units: number): number => units * spacing;

// =============================================================================
// TERMINAL STYLES (for CLI-like components)
// =============================================================================

/**
 * Terminal style helper for CLI-like components
 * Uses semantic tokens internally
 */
export const getTerminalStyles = (mode: "light" | "dark") => {
  const colors =
    mode === "light" ? kiloColors.terminal.light : kiloColors.terminal.dark;

  return {
    container: {
      bgcolor: colors.background,
      border: "1px solid",
      borderColor: kiloColors.borders.secondary,
      borderRadius: 1,
      p: 1,
    },
    line: {
      py: 0.5,
      px: 1,
      fontFamily: semanticTypography.fontFamily.code,
      fontSize: "0.875rem",
      bgcolor: colors.background,
      color: colors.text,
      whiteSpace: "nowrap" as const,
      overflow: "auto",
      "&:hover": {
        bgcolor: colors.backgroundAlt,
      },
    },
    header: {
      px: 1,
      py: 0.5,
      bgcolor: colors.backgroundAlt,
      borderBottom: "1px solid",
      borderColor: kiloColors.borders.secondary,
      fontFamily: semanticTypography.fontFamily.code,
      color: colors.header,
      fontWeight: "bold",
      textTransform: "uppercase" as const,
    },
    colors,
  };
};

// =============================================================================
// TIME FORMATTING HELPERS
// =============================================================================

/**
 * Format time in 24h format
 */
export const formatTime24h = (date: Date | string): string => {
  const d = typeof date === "string" ? new Date(date) : date;
  return d.toLocaleTimeString("en-US", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};

/**
 * Format time for terminal/log displays with brackets
 */
export const formatTerminalTime = (date: Date | string): string => {
  return `[${formatTime24h(date)}]`;
};
