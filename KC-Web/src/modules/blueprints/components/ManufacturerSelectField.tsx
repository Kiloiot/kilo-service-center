/**
 * Manufacturer picker of the Add Model form: a MUI menu (not the browser's
 * native list) so a choice made with the mouse or the keyboard both stick.
 */

import React from "react";

import type { ManufacturerUI } from "@api-types/api";
import { MenuItem, TextField } from "@mui/material";

import { BLUEPRINT_LABELS } from "@constants/messages";

interface ManufacturerSelectFieldProps {
  manufacturers: ManufacturerUI[];
  value: string;
  onChange: (manufacturerId: string) => void;
  disabled: boolean;
  error?: string;
}

const ManufacturerSelectField: React.FC<ManufacturerSelectFieldProps> = ({
  manufacturers,
  value,
  onChange,
  disabled,
  error,
}) => {
  const known = manufacturers.some((m) => m.id === value);
  return (
    <TextField
      select
      fullWidth
      required
      margin="dense"
      label={BLUEPRINT_LABELS.LABEL_MANUFACTURER}
      value={known ? value : ""}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      error={!!error}
      helperText={error}
    >
      {manufacturers.map((m) => (
        <MenuItem key={m.id} value={m.id}>
          {m.name}
        </MenuItem>
      ))}
    </TextField>
  );
};

export default ManufacturerSelectField;
