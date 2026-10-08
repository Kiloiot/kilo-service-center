import type { UplinkUI } from "@api-types/api";
import { Box, Typography } from "@mui/material";

import { HexText } from "@components/common/HexText";
import { formatUserData } from "@utils/formatters";
import { formatDecodedValues } from "@utils/uplink-summary";

/** A blueprint-decoded payload reads as its values, the hex beneath; otherwise the hex alone. */
export function UplinkUserDataCell({ uplink }: { uplink: UplinkUI }) {
  const hex = <HexText>{formatUserData(uplink.userData)}</HexText>;
  const decoded = formatDecodedValues(uplink);
  if (!decoded) return hex;
  return (
    <Box>
      <Typography variant="body2">{decoded}</Typography>
      <Typography variant="caption" color="text.secondary">
        {hex}
      </Typography>
    </Box>
  );
}
