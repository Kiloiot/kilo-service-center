/**
 * Monospace table text for identifiers (EUIs, opIds, queIds, event types),
 * which never break inside the value.
 */

import type { ReactNode } from "react";

import { Typography } from "@mui/material";

import { getMonoBody2 } from "@utils/typography";
import { componentSpacing } from "@theme/index";

export function MonoText({ children }: { children: ReactNode }) {
  return (
    <Typography
      component="span"
      sx={(theme) => ({
        ...getMonoBody2(theme),
        whiteSpace: componentSpacing.tableValue.unbroken,
      })}
    >
      {children}
    </Typography>
  );
}
