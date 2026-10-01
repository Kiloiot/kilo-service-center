/**
 * The Service Center product logo. The one-color artwork is a mask filled
 * with the accent, so the same asset reads on the light and the dark paper.
 */

import { Box } from "@mui/material";

import { LOGO } from "@constants/app";
import { accentTextColor } from "@theme/controls";

const MASK = `url(${LOGO.PATH}) center / contain no-repeat`;

export function ProductLogo({ width }: { width: number }) {
  return (
    <Box
      role="img"
      aria-label={LOGO.ALT}
      sx={{
        width,
        aspectRatio: LOGO.ASPECT_RATIO,
        bgcolor: accentTextColor,
        mask: MASK,
        WebkitMask: MASK,
      }}
    />
  );
}
