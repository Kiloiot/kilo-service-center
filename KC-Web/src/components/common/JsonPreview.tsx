/**
 * Read-only, pretty-printed JSON block for raw protocol payloads.
 */

import { Box, Typography } from "@mui/material";

import { getMonoBody2 } from "@utils/typography";
import { JSON_PREVIEW } from "@constants/app";

interface JsonPreviewProps {
  title: string;
  value: unknown;
}

export function JsonPreview({ title, value }: JsonPreviewProps) {
  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>
        {title}
      </Typography>
      <Box
        component="pre"
        sx={(theme) => ({
          ...getMonoBody2(theme),
          m: 0,
          p: JSON_PREVIEW.PADDING,
          maxHeight: JSON_PREVIEW.MAX_HEIGHT,
          overflow: "auto",
          bgcolor: "background.default",
          borderRadius: 1,
          whiteSpace: "pre-wrap",
          wordBreak: "break-all",
        })}
      >
        {JSON.stringify(value, null, JSON_PREVIEW.INDENT)}
      </Box>
    </Box>
  );
}
