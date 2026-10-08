import React from "react";

import { Alert, Box, CircularProgress, Paper, Typography } from "@mui/material";

import { getErrorMessage } from "@utils/error-message";
import { componentSpacing } from "@theme/index";

interface DataCardProps {
  title: string;
  subtitle?: string;
  /** Controls rendered at the right of the header (filters, buttons). */
  action?: React.ReactNode;
  isLoading?: boolean;
  error?: unknown;
  errorFallback?: string;
  children: React.ReactNode;
}

/** A titled card that renders a spinner, an error or its content. */
export const DataCard: React.FC<DataCardProps> = ({
  title,
  subtitle,
  action,
  isLoading = false,
  error,
  errorFallback,
  children,
}) => (
  <Paper sx={{ p: componentSpacing.dataCard.padding, height: "100%" }}>
    <Box
      sx={{
        display: "flex",
        justifyContent: "space-between",
        alignItems: "flex-start",
        flexWrap: "wrap",
        gap: componentSpacing.dataCard.headerGap,
        mb: componentSpacing.dataCard.headerMb,
      }}
    >
      <Box>
        <Typography
          variant="h6"
          component="h2"
          sx={{ fontWeight: componentSpacing.cardTitle.fontWeight }}
        >
          {title}
        </Typography>
        {subtitle && (
          <Typography variant="body2" color="text.secondary">
            {subtitle}
          </Typography>
        )}
      </Box>
      {action}
    </Box>
    {isLoading ? (
      <Box sx={{ display: "flex", justifyContent: "center" }}>
        <CircularProgress size={componentSpacing.spinner.section} />
      </Box>
    ) : error ? (
      <Alert severity="error">{getErrorMessage(error, errorFallback)}</Alert>
    ) : (
      children
    )}
  </Paper>
);
