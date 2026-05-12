/**
 * Base station configuration summary showing name and EUI with copy button.
 * Shared between the partial-success and success views in the commissioning dialog.
 */

import React from "react";

import { Box, IconButton, Paper, Tooltip, Typography } from "@mui/material";
import type { Theme } from "@mui/material";

import {
  ACTION_COPIED,
  ACTION_COPY,
  LABEL_BS_EUI_DISPLAY,
  LABEL_NAME,
  SECTION_BS_CONFIG,
} from "@constants/messages";
import { CheckCircleIcon, ContentCopyIcon } from "@theme/icons";

interface BsConfigSummaryProps {
  name: string;
  eui: string;
  copiedField: string | null;
  onCopy: (value: string, field: string) => void;
  /** Style getter for monospace text (getMonoBody2). */
  monoSxGetter: (theme: Theme) => Record<string, unknown>;
  children?: React.ReactNode;
}

const BsConfigSummary: React.FC<BsConfigSummaryProps> = ({
  name,
  eui,
  copiedField,
  onCopy,
  monoSxGetter,
  children,
}) => (
  <Paper variant="outlined" sx={{ p: 2, mb: 3 }}>
    <Typography variant="subtitle2" gutterBottom>
      {SECTION_BS_CONFIG}
    </Typography>

    <Box sx={{ mt: 2 }}>
      {/* Name */}
      <Box
        display="flex"
        alignItems="center"
        justifyContent="space-between"
        sx={{ mb: 1 }}
      >
        <Typography variant="body2" color="text.secondary">
          {LABEL_NAME}
        </Typography>
        <Typography variant="body2">{name}</Typography>
      </Box>

      {/* EUI with copy */}
      <Box
        display="flex"
        alignItems="center"
        justifyContent="space-between"
        sx={{ mb: 1 }}
      >
        <Typography variant="body2" color="text.secondary">
          {LABEL_BS_EUI_DISPLAY}
        </Typography>
        <Box display="flex" alignItems="center" gap={1}>
          <Typography variant="body2" sx={(theme) => monoSxGetter(theme)}>
            {eui}
          </Typography>
          <Tooltip title={copiedField === "eui" ? ACTION_COPIED : ACTION_COPY}>
            <IconButton size="small" onClick={() => onCopy(eui, "eui")}>
              {copiedField === "eui" ? (
                <CheckCircleIcon fontSize="small" color="success" />
              ) : (
                <ContentCopyIcon fontSize="small" />
              )}
            </IconButton>
          </Tooltip>
        </Box>
      </Box>

      {children}
    </Box>
  </Paper>
);

export default BsConfigSummary;
