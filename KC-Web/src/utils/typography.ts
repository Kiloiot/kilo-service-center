import type { Theme } from "@mui/material/styles";

// Helper functions that return style objects - defined outside render scope
// These are pure functions that won't recreate objects unnecessarily

export const getMonoBody2 = (theme: Theme) => ({
  ...theme.typography.body2,
  fontFamily: theme.typography.monoFontFamily,
});

export const getMonoBody1 = (theme: Theme) => ({
  ...theme.typography.body1,
  fontFamily: theme.typography.monoFontFamily,
});
