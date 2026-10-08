import React from "react";

import { IconButton, InputAdornment, TextField, Tooltip } from "@mui/material";

import { getMonoBody1 } from "@utils/typography";
import { BASE_STATION_DETAILS, PLACEHOLDER_BS_EUI } from "@constants/messages";
import { CheckCircleIcon, ContentCopyIcon } from "@theme/icons";

interface BaseStationEuiFieldProps {
  value: string;
  error: string | null;
  copied: boolean;
  onChange: (value: string) => void;
  onCopy: () => void;
}

/** Editable base station EUI with a copy button. */
const BaseStationEuiField: React.FC<BaseStationEuiFieldProps> = ({
  value,
  error,
  copied,
  onChange,
  onCopy,
}) => (
  <TextField
    label={BASE_STATION_DETAILS.LABEL_EDIT_EUI}
    value={value}
    onChange={(e) => onChange(e.target.value)}
    error={!!error}
    helperText={error}
    fullWidth
    placeholder={PLACEHOLDER_BS_EUI}
    slotProps={{
      input: {
        endAdornment: (
          <InputAdornment position="end">
            <Tooltip
              title={
                copied
                  ? BASE_STATION_DETAILS.LABEL_EUI_COPIED
                  : BASE_STATION_DETAILS.ACTION_COPY_EUI
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
        sx: (theme) => getMonoBody1(theme),
      },
    }}
    sx={{ mb: 3 }}
  />
);

export default BaseStationEuiField;
