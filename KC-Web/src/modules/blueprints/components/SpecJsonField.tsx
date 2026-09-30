/**
 * Blueprint specification editor: a plain fixed-height textarea. MUI's
 * multiline field autosizes by measuring layout on every keystroke, which
 * dropped characters and stalled the page on long specs.
 */

import React from "react";

import { inputBaseClasses, TextField } from "@mui/material";

import { BLUEPRINT_LABELS } from "@constants/messages";

interface SpecJsonFieldProps {
  value: string;
  onChange: (value: string) => void;
  rows: number;
  label?: string;
  error?: string | null;
}

const SpecJsonField: React.FC<SpecJsonFieldProps> = ({
  value,
  onChange,
  rows,
  label,
  error,
}) => (
  <TextField
    fullWidth
    required
    margin="dense"
    label={label}
    value={value}
    onChange={(e) => onChange(e.target.value)}
    error={!!error}
    helperText={error ?? BLUEPRINT_LABELS.HELPER_SPEC_JSON}
    slotProps={{
      input: {
        inputComponent: "textarea",
        sx: (theme) => ({
          [`& .${inputBaseClasses.input}`]: {
            height: "auto",
            resize: "vertical",
            fontFamily: theme.typography.monoFontFamily,
            fontSize: theme.typography.body2.fontSize,
          },
        }),
      },
      htmlInput: { rows, spellCheck: false },
    }}
  />
);

export default SpecJsonField;
