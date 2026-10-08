/**
 * Shows the protocol flags that are set as small chips (e.g. dlOpen,
 * responseExp); no set flag renders the empty-value placeholder.
 */

import { Box, Chip } from "@mui/material";

import { FLAG_CHIPS_LAYOUT } from "@constants/app";
import { DATA_TABLE } from "@constants/messages";

interface Flag {
  label: string;
  set: boolean;
}

export function FlagChips({ flags }: { flags: readonly Flag[] }) {
  const active = flags.filter((flag) => flag.set);
  if (active.length === 0) return <>{DATA_TABLE.NO_VALUE}</>;
  return (
    <Box sx={{ display: "flex", flexWrap: "wrap", gap: FLAG_CHIPS_LAYOUT.GAP }}>
      {active.map((flag) => (
        <Chip
          key={flag.label}
          label={flag.label}
          size="small"
          variant="outlined"
        />
      ))}
    </Box>
  );
}
