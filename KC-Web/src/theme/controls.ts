// Shared states of the interactive controls (buttons, icon buttons, tabs).

import { buttonBaseClasses } from "@mui/material";
import type { CSSObject, Theme } from "@mui/material/styles";

import { componentSpacing } from "./componentSpacing";

const { focusRing } = componentSpacing;

const FOCUS_VISIBLE = `&.${buttonBaseClasses.focusVisible}`;

const ring = (theme: Theme, outlineOffset: number): CSSObject => ({
  [FOCUS_VISIBLE]: {
    outline: `${focusRing.width}px solid ${theme.palette.highlight.main}`,
    outlineOffset,
  },
});

// index.css removes the browser outline, so every control draws this one.
export const focusVisibleRing = (theme: Theme) => ring(theme, focusRing.offset);

// Tabs clip their overflow, so their outline is drawn inside the tab.
export const insetFocusVisibleRing = (theme: Theme) =>
  ring(theme, focusRing.insetOffset);

// Navy reads on the light paper but not on the dark one.
export const accentTextColor = (theme: Theme) =>
  theme.palette.mode === "light"
    ? theme.palette.primary.main
    : theme.palette.text.primary;
