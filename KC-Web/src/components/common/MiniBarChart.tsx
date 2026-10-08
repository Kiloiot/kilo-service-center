import React from "react";

import type { MiniBar } from "@api-types/chart";
import { Box, Tooltip } from "@mui/material";

import { PERCENT_SCALE } from "@constants/app";
import { accentTextColor } from "@theme/controls";
import { componentSpacing } from "@theme/index";

interface MiniBarChartProps {
  bars: MiniBar[];
  /** Value of a full-height bar; defaults to the largest bar. */
  max?: number;
  ariaLabel: string;
}

/** A compact bar series scaled to its largest value. */
export const MiniBarChart: React.FC<MiniBarChartProps> = ({
  bars,
  max,
  ariaLabel,
}) => {
  const top = max ?? Math.max(0, ...bars.map((bar) => bar.value));
  return (
    <Box
      role="img"
      aria-label={ariaLabel}
      sx={{
        display: "flex",
        alignItems: "flex-end",
        gap: componentSpacing.miniBarChart.barGap,
        height: componentSpacing.miniBarChart.height,
      }}
    >
      {bars.map((bar) => (
        <Tooltip key={bar.key} title={bar.label}>
          <Box
            sx={{
              flex: 1,
              minHeight: componentSpacing.miniBarChart.minBarHeight,
              height: top > 0 ? `${(bar.value / top) * PERCENT_SCALE}%` : 0,
              bgcolor: accentTextColor,
              borderRadius: componentSpacing.miniBarChart.barRadius,
            }}
          />
        </Tooltip>
      ))}
    </Box>
  );
};
