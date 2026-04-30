import type { Theme } from '@mui/material/styles';

// Helper functions that return style objects - defined outside render scope
// These are pure functions that won't recreate objects unnecessarily

export const getMonoTypography = (theme: Theme) => ({
  fontFamily: theme.typography.monoFontFamily,
});

export const getMonoCaption = (theme: Theme) => ({
  ...theme.typography.caption,
  fontFamily: theme.typography.monoFontFamily,
});

export const getMonoBody2 = (theme: Theme) => ({
  ...theme.typography.body2,
  fontFamily: theme.typography.monoFontFamily,
});

export const getMonoBody1 = (theme: Theme) => ({
  ...theme.typography.body1,
  fontFamily: theme.typography.monoFontFamily,
});

// For technical displays (EUI, addresses, etc)
export const getTechnicalTypography = (theme: Theme) => ({
  fontFamily: theme.typography.monoFontFamily,
  letterSpacing: '0.025em', // Slight spacing for readability
});
