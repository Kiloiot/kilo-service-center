// The left navigation keeps its own control look; the app's button scale does not reach it.

import { buttonClasses } from "@mui/material";
import type { CSSObject, Theme } from "@mui/material/styles";
import { alpha } from "@mui/material/styles";

import { componentSpacing } from "./componentSpacing";
import { highlightSurface } from "./highlight";

const { userMenuTrigger: trigger } = componentSpacing;

const hoverShadow = (theme: Theme) =>
  `0 ${trigger.hoverShadowOffsetY}px ${trigger.hoverShadowBlur}px ${alpha(
    theme.palette.common.black,
    trigger.hoverShadowOpacity[theme.palette.mode],
  )}`;

export const userMenuTriggerStyle = (theme: Theme): CSSObject => ({
  width: "100%",
  minHeight: "auto",
  justifyContent: "flex-start",
  textTransform: "none",
  color: theme.palette.text.primary,
  fontSize: trigger.fontSize,
  padding: theme.spacing(trigger.paddingY, trigger.paddingX),
  transition: trigger.transition,
  [`& .${buttonClasses.startIcon} > *:nth-of-type(1)`]: {
    fontSize: trigger.iconSize,
  },
  [`&.${buttonClasses.focusVisible}`]: { outline: "none" },
  "&:hover": {
    ...highlightSurface(theme.palette.highlight),
    transform: `translateY(${trigger.hoverLift}px)`,
    boxShadow: hoverShadow(theme),
  },
});
