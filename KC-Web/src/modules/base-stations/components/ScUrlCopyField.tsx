/**
 * Read-only Service Center URL field with a copy-to-clipboard button.
 * Used in BaseStationDetails for both the main edit view and
 * the post-regeneration certificate view.
 */

import React from "react";

import { IconButton, InputAdornment, TextField, Tooltip } from "@mui/material";
import type { Theme } from "@mui/material";

import { BASE_STATION_DETAILS } from "@constants/messages";
import { CheckCircleIcon, ContentCopyIcon } from "@theme/icons";

interface ScUrlCopyFieldProps {
  value: string;
  copied: boolean;
  onCopy: () => void;
  /** Style getter for monospace input body (getMonoBody1). */
  inputSxGetter: (theme: Theme) => Record<string, unknown>;
  /** Bottom margin. Defaults to 3. */
  mb?: number;
}

const ScUrlCopyField: React.FC<ScUrlCopyFieldProps> = ({
  value,
  copied,
  onCopy,
  inputSxGetter,
  mb = 3,
}) => (
  <TextField
    label={BASE_STATION_DETAILS.LABEL_SC_URL}
    value={value}
    fullWidth
    slotProps={{
      input: {
        readOnly: true,
        endAdornment: (
          <InputAdornment position="end">
            <Tooltip
              title={
                copied
                  ? BASE_STATION_DETAILS.LABEL_SC_URL_COPIED
                  : BASE_STATION_DETAILS.ACTION_COPY_SC_URL
              }
            >
              <IconButton size="small" onClick={onCopy} edge="end">
                {copied ? (
                  <CheckCircleIcon fontSize="small" color="success" />
                ) : (
                  <ContentCopyIcon fontSize="small" />
                )}
              </IconButton>
            </Tooltip>
          </InputAdornment>
        ),
        sx: (theme: Theme) => inputSxGetter(theme),
      },
    }}
    sx={{ mb }}
  />
);

export default ScUrlCopyField;
