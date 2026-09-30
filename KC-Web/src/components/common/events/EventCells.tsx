/**
 * Cells of the event tables: a severity chip and a summary line with its
 * captions.
 */

import { Box, Chip, Typography } from "@mui/material";

import { eventSeverityChip } from "@utils/chipMappings";

export function SeverityChip({ severity }: { severity: string }) {
  const chip = eventSeverityChip(severity);
  return <Chip label={chip.label} color={chip.color} size="small" />;
}

function CaptionLine({ text }: { text?: string }) {
  if (!text) return null;
  return (
    <Typography variant="caption" color="text.secondary" display="block">
      {text}
    </Typography>
  );
}

/** A summary line; each caption, when present, reads on a line of its own below it. */
export function SummaryCell({
  title,
  caption,
  detail,
}: {
  title: string;
  caption?: string;
  detail?: string;
}) {
  return (
    <Box>
      <Typography variant="body2">{title}</Typography>
      <CaptionLine text={caption} />
      <CaptionLine text={detail} />
    </Box>
  );
}
