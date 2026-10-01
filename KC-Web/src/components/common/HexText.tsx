/**
 * Monospace table text for hex user data, which wraps between any two digits
 * so a long payload never widens its table.
 */

import type { ReactNode } from "react";

import { Typography } from "@mui/material";

import { getMonoBody2 } from "@utils/typography";
import { componentSpacing } from "@theme/index";

export function HexText({ children }: { children: ReactNode }) {
  return (
    <Typography
      component="span"
      sx={(theme) => ({
        ...getMonoBody2(theme),
        wordBreak: componentSpacing.tableValue.breakAnywhere,
      })}
    >
      {children}
    </Typography>
  );
}
