import { keyframes } from "@mui/material/styles";

import { componentSpacing } from "./componentSpacing";

const pulse = keyframes`
  0% { opacity: 1; }
  50% { opacity: ${componentSpacing.livePulse.dimOpacity}; }
  100% { opacity: 1; }
`;

/** A slow fade in and out that marks an indicator as live. */
export const livePulseAnimation = `${pulse} ${componentSpacing.livePulse.duration} infinite`;
