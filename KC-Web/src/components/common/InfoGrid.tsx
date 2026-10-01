import React from "react";

import { Box, Typography } from "@mui/material";

import { componentSpacing } from "@theme/index";

export interface InfoGridItem {
  label: string;
  value: React.ReactNode;
}

/** Label/value pairs laid out in as many columns as the width allows. */
export const InfoGrid: React.FC<{ items: InfoGridItem[] }> = ({ items }) => (
  <Box
    sx={{
      display: "grid",
      gridTemplateColumns: `repeat(auto-fill, minmax(${componentSpacing.infoGrid.minColumnWidth}px, 1fr))`,
      columnGap: componentSpacing.infoGrid.columnGap,
      rowGap: componentSpacing.infoGrid.rowGap,
    }}
  >
    {items.map((item) => (
      <Box key={item.label} sx={{ minWidth: 0 }}>
        <Typography variant="caption" color="text.secondary" component="div">
          {item.label}
        </Typography>
        <Typography
          variant="body2"
          component="div"
          sx={{ overflowWrap: "anywhere" }}
        >
          {item.value}
        </Typography>
      </Box>
    ))}
  </Box>
);
