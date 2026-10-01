/**
 * Table text for a formatted date and time, which never breaks between the
 * date and the time.
 */

import { Typography } from "@mui/material";

import { componentSpacing } from "@theme/index";

export function TimeText({ children }: { children: string }) {
  return (
    <Typography
      component="span"
      variant="body2"
      sx={{ whiteSpace: componentSpacing.tableValue.unbroken }}
    >
      {children}
    </Typography>
  );
}
