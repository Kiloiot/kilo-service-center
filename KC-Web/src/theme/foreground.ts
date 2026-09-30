// Components that draw the primary color as a foreground: they use the accent,
// which stays readable on the dark paper where navy disappears.

import { checkboxClasses, radioClasses, switchClasses } from "@mui/material";
import type { Components, Theme } from "@mui/material/styles";
import { alpha } from "@mui/material/styles";

import { accentTextColor } from "./controls";

const UNDERLINE_OPACITY = 0.4;

const checkedAccent = (checked: string) => ({
  styleOverrides: {
    colorPrimary: ({ theme }: { theme: Theme }) => ({
      [`&.${checked}`]: { color: accentTextColor(theme) },
    }),
  },
});

export const foregroundOverrides: Pick<
  Components<Theme>,
  "MuiLink" | "MuiCircularProgress" | "MuiCheckbox" | "MuiRadio" | "MuiSwitch"
> = {
  MuiLink: {
    styleOverrides: {
      root: ({ theme }) => ({
        color: accentTextColor(theme),
        textDecorationColor: alpha(accentTextColor(theme), UNDERLINE_OPACITY),
      }),
    },
  },
  MuiCircularProgress: {
    styleOverrides: {
      colorPrimary: ({ theme }) => ({ color: accentTextColor(theme) }),
    },
  },
  MuiCheckbox: checkedAccent(checkboxClasses.checked),
  MuiRadio: checkedAccent(radioClasses.checked),
  MuiSwitch: {
    styleOverrides: {
      switchBase: ({ theme }) => ({
        [`&.${switchClasses.checked}.${switchClasses.colorPrimary}`]: {
          color: accentTextColor(theme),
        },
        [`&.${switchClasses.checked}.${switchClasses.colorPrimary} + .${switchClasses.track}`]:
          { backgroundColor: accentTextColor(theme) },
      }),
    },
  },
};
