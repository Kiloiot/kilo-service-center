// One button scale: heights come from minHeight tokens, so contained, outlined
// and text buttons of a size line up, and large renders as medium.

import { buttonClasses, svgIconClasses } from "@mui/material";
import type { Components, Theme } from "@mui/material/styles";
import { alpha } from "@mui/material/styles";

import { componentSpacing } from "./componentSpacing";
import { accentTextColor, focusVisibleRing } from "./controls";

const { button, iconButton } = componentSpacing;

const HOVER_TINT_OPACITY = 0.1;

/** The button colors the palette does not carry. */
interface ButtonColors {
  containedHover: string;
  outlinedBorder: string;
}

interface ButtonScale {
  minHeight: number;
  paddingX: number;
  fontSize: number;
}

interface IconButtonScale {
  padding: number;
  iconSize: number;
}

const buttonSize = ({ minHeight, paddingX, fontSize }: ButtonScale) => ({
  minHeight,
  padding: `0 ${paddingX}px`,
  fontSize,
});

const buttonIconSize = (fontSize: number) => ({
  "& > *:nth-of-type(1)": { fontSize },
});

const iconButtonSize = ({ padding, iconSize }: IconButtonScale) => ({
  padding,
  [`& .${svgIconClasses.root}`]: { fontSize: iconSize },
});

export const createButtonOverrides = (
  colors: ButtonColors,
): Pick<Components<Theme>, "MuiButton" | "MuiIconButton"> => ({
  MuiButton: {
    defaultProps: { disableElevation: true, disableRipple: true },
    styleOverrides: {
      root: ({ theme }) => ({
        borderRadius: button.radius,
        ...focusVisibleRing(theme),
        [`&.${buttonClasses.disabled}`]: { color: theme.palette.text.disabled },
      }),
      sizeSmall: buttonSize(button.small),
      sizeMedium: buttonSize(button.medium),
      sizeLarge: buttonSize(button.medium),
      iconSizeSmall: buttonIconSize(button.small.iconSize),
      iconSizeMedium: buttonIconSize(button.medium.iconSize),
      iconSizeLarge: buttonIconSize(button.medium.iconSize),
    },
    // Only the default color is repainted; error, warning and the other colors keep their palette styles.
    variants: [
      {
        props: { variant: "contained", color: "primary" },
        style: ({ theme }) => ({
          backgroundColor: theme.palette.primary.main,
          color: theme.palette.primary.contrastText,
          "&:hover": { backgroundColor: colors.containedHover },
        }),
      },
      {
        props: { variant: "outlined", color: "primary" },
        style: ({ theme }) => ({
          borderColor: colors.outlinedBorder,
          color: theme.palette.text.primary,
          "&:hover": {
            borderColor: accentTextColor(theme),
            backgroundColor: alpha(accentTextColor(theme), HOVER_TINT_OPACITY),
          },
        }),
      },
      {
        props: { variant: "text", color: "primary" },
        style: ({ theme }) => ({
          color: accentTextColor(theme),
          "&:hover": {
            backgroundColor: alpha(accentTextColor(theme), HOVER_TINT_OPACITY),
          },
        }),
      },
    ],
  },
  MuiIconButton: {
    defaultProps: { disableRipple: true },
    styleOverrides: {
      root: ({ theme }) => focusVisibleRing(theme),
      sizeSmall: iconButtonSize(iconButton.small),
      sizeMedium: iconButtonSize(iconButton.medium),
      sizeLarge: iconButtonSize(iconButton.medium),
    },
  },
});
