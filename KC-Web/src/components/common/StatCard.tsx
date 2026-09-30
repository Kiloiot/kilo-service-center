/** The one summary card: label, value and caption on the left, the icon tile on the right. */

import { Children, type ReactElement, type ReactNode } from "react";

import { Box, Card, CardContent, Chip, Grid, Typography } from "@mui/material";

import { componentSpacing } from "@theme/index";

export type StatCardColor =
  | "primary"
  | "secondary"
  | "success"
  | "warning"
  | "error"
  | "info";

interface StatCardProps {
  label: string;
  value: string | number;
  icon: ReactElement;
  color: StatCardColor;
  caption?: string;
  trendLabel?: string;
  onClick?: () => void;
}

const { statCard } = componentSpacing;

export function StatCard({
  label,
  value,
  icon,
  color,
  caption,
  trendLabel,
  onClick,
}: StatCardProps) {
  return (
    <Card
      data-testid="stat-card"
      onClick={onClick}
      sx={{ height: "100%", cursor: onClick ? "pointer" : "default" }}
    >
      <CardContent>
        <Box
          sx={{
            display: "flex",
            alignItems: "flex-start",
            justifyContent: "space-between",
          }}
        >
          <Box sx={{ flex: 1 }}>
            <Typography color="text.secondary" variant="body2" gutterBottom>
              {label}
            </Typography>
            <Typography
              variant="h4"
              sx={{ fontWeight: componentSpacing.cardTitle.fontWeight }}
            >
              {value}
            </Typography>
            {caption && (
              <Typography variant="caption" color="text.secondary">
                {caption}
              </Typography>
            )}
            {trendLabel && (
              <Box sx={{ mt: statCard.trendGap }}>
                <Chip
                  label={trendLabel}
                  size="small"
                  variant="outlined"
                  color="success"
                />
              </Box>
            )}
          </Box>
          <Box
            sx={{
              p: statCard.iconPadding,
              borderRadius: statCard.iconRadius,
              display: "flex",
              bgcolor: `${color}.main`,
              color: `${color}.contrastText`,
            }}
          >
            {icon}
          </Box>
        </Box>
      </CardContent>
    </Card>
  );
}

/** A summary row: every card takes the same width at each breakpoint. */
export function StatCardRow({ children }: { children: ReactNode }) {
  return (
    <Grid container spacing={statCard.gridGap} sx={{ mb: statCard.rowMb }}>
      {Children.map(
        children,
        (card) => card && <Grid size={statCard.gridSize}>{card}</Grid>,
      )}
    </Grid>
  );
}
