/**
 * Shared manufacturer name + website text fields used by
 * both AddManufacturerDialog and EditManufacturerDialog.
 */

import React from "react";

import { TextField } from "@mui/material";

import { BLUEPRINT_LABELS } from "@constants/messages";

interface ManufacturerFormFieldsProps {
  name: string;
  nameError: string | null;
  website: string;
  onNameChange: (value: string) => void;
  onWebsiteChange: (value: string) => void;
}

const ManufacturerFormFields: React.FC<ManufacturerFormFieldsProps> = ({
  name,
  nameError,
  website,
  onNameChange,
  onWebsiteChange,
}) => (
  <>
    <TextField
      autoFocus
      required
      margin="dense"
      label={BLUEPRINT_LABELS.LABEL_NAME}
      fullWidth
      value={name}
      onChange={(e) => onNameChange(e.target.value)}
      error={!!nameError}
      helperText={nameError}
    />
    <TextField
      margin="dense"
      label={BLUEPRINT_LABELS.LABEL_WEBSITE}
      fullWidth
      value={website}
      onChange={(e) => onWebsiteChange(e.target.value)}
    />
  </>
);

export default ManufacturerFormFields;
