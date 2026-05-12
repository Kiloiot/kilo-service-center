/**
 * Shared organization form fields (name, description, base station toggle,
 * count limits, and tags) used by both AddOrganizationDialog and OrganizationForm.
 */

import React from "react";

import {
  Box,
  FormControlLabel,
  Switch,
  TextField,
  Typography,
} from "@mui/material";

import { ORGANIZATION_FORM } from "@constants/messages";

import TagsEditor from "./TagsEditor";

interface OrganizationFormFieldsProps {
  name: string;
  description: string;
  canHaveBaseStations: boolean;
  maxBsCount: string;
  maxEpCount: string;
  tags: Record<string, string>;
  onNameChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onCanHaveBaseStationsChange: (value: boolean) => void;
  onMaxBsCountChange: (value: string) => void;
  onMaxEpCountChange: (value: string) => void;
  onTagsChange: (value: Record<string, string>) => void;
  autoFocus?: boolean;
}

const OrganizationFormFields: React.FC<OrganizationFormFieldsProps> = ({
  name,
  description,
  canHaveBaseStations,
  maxBsCount,
  maxEpCount,
  tags,
  onNameChange,
  onDescriptionChange,
  onCanHaveBaseStationsChange,
  onMaxBsCountChange,
  onMaxEpCountChange,
  onTagsChange,
  autoFocus = false,
}) => (
  <>
    <TextField
      label={ORGANIZATION_FORM.LABEL_NAME}
      value={name}
      onChange={(e) => onNameChange(e.target.value)}
      helperText={ORGANIZATION_FORM.HELPER_NAME}
      fullWidth
      required
      margin="normal"
      autoFocus={autoFocus}
    />
    <TextField
      label={ORGANIZATION_FORM.LABEL_DESCRIPTION}
      value={description}
      onChange={(e) => onDescriptionChange(e.target.value)}
      helperText={ORGANIZATION_FORM.HELPER_DESCRIPTION}
      fullWidth
      multiline
      rows={3}
      margin="normal"
    />
    <FormControlLabel
      control={
        <Switch
          checked={canHaveBaseStations}
          onChange={(e) => onCanHaveBaseStationsChange(e.target.checked)}
        />
      }
      label={ORGANIZATION_FORM.LABEL_CAN_HAVE_BS}
      sx={{ mt: 1 }}
    />
    <Box sx={{ display: "flex", gap: 2, mt: 2 }}>
      <TextField
        label={ORGANIZATION_FORM.LABEL_MAX_BS_COUNT}
        value={maxBsCount}
        onChange={(e) => onMaxBsCountChange(e.target.value.replace(/\D/g, ""))}
        helperText={ORGANIZATION_FORM.HELPER_MAX_BS_COUNT}
        type="number"
        slotProps={{ htmlInput: { min: 0 } }}
        fullWidth
        disabled={!canHaveBaseStations}
      />
      <TextField
        label={ORGANIZATION_FORM.LABEL_MAX_EP_COUNT}
        value={maxEpCount}
        onChange={(e) => onMaxEpCountChange(e.target.value.replace(/\D/g, ""))}
        helperText={ORGANIZATION_FORM.HELPER_MAX_EP_COUNT}
        type="number"
        slotProps={{ htmlInput: { min: 0 } }}
        fullWidth
      />
    </Box>
    <Box sx={{ mt: 3 }}>
      <Typography variant="subtitle2" gutterBottom>
        {ORGANIZATION_FORM.LABEL_TAGS}
      </Typography>
      <TagsEditor tags={tags} onChange={onTagsChange} />
    </Box>
  </>
);

export default OrganizationFormFields;
