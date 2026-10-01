// One tab style: a compact bar, color-only hover, no ripple, and an underline
// that reads in both modes.

import { tabClasses } from "@mui/material";
import type { Components, Theme } from "@mui/material/styles";

import { componentSpacing } from "./componentSpacing";
import { accentTextColor, insetFocusVisibleRing } from "./controls";

const { tabs } = componentSpacing;

export const tabOverrides: Pick<Components<Theme>, "MuiTabs" | "MuiTab"> = {
  MuiTabs: {
    styleOverrides: {
      root: { minHeight: tabs.minHeight },
      indicator: ({ theme }) => ({
        height: tabs.indicatorHeight,
        backgroundColor: accentTextColor(theme),
      }),
    },
  },
  MuiTab: {
    defaultProps: { disableRipple: true },
    styleOverrides: {
      root: ({ theme }) => ({
        minHeight: tabs.minHeight,
        padding: `0 ${tabs.paddingX}px`,
        gap: theme.spacing(tabs.iconGap),
        fontFamily: theme.typography.button.fontFamily,
        fontSize: tabs.fontSize,
        fontWeight: theme.typography.fontWeightMedium,
        lineHeight: theme.typography.button.lineHeight,
        textTransform: "none",
        color: theme.palette.text.secondary,
        "&:hover": { color: theme.palette.text.primary },
        [`&.${tabClasses.selected}`]: { color: accentTextColor(theme) },
        [`&.${tabClasses.labelIcon}`]: {
          minHeight: tabs.minHeight,
          paddingTop: 0,
          paddingBottom: 0,
        },
        [`& > .${tabClasses.icon}`]: { fontSize: tabs.iconSize, margin: 0 },
        ...insetFocusVisibleRing(theme),
      }),
    },
  },
};
