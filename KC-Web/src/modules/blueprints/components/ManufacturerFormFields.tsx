/**
 * Shared manufacturer name + website text fields used by
 * both AddManufacturerDialog and EditManufacturerDialog.
 */

import React from "react";

import { TextField } from "@mui/material";

import { BLUEPRINT_LABELS } from "@constants/messages";

interface ManufacturerFormFieldsProps {
  name: string;
  website: string;
  onNameChange: (value: string) => void;
  onWebsiteChange: (value: string) => void;
}

const ManufacturerFormFields: React.FC<ManufacturerFormFieldsProps> = ({
  name,
  website,
  onNameChange,
  onWebsiteChange,
}) => (
  <>
    <TextField
      autoFocus
      margin="dense"
      label={BLUEPRINT_LABELS.LABEL_NAME}
      fullWidth
      value={name}
      onChange={(e) => onNameChange(e.target.value)}
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
