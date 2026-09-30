/**
 * Shared organization form fields (name, description and tags) used by both
 * AddOrganizationDialog and OrganizationForm.
 */

import React from "react";

import { Box, TextField, Typography } from "@mui/material";

import { ORGANIZATION_FORM } from "@constants/messages";
import { componentSpacing } from "@theme/index";

import TagsEditor from "./TagsEditor";

interface OrganizationFormFieldsProps {
  name: string;
  description: string;
  tags: Record<string, string>;
  onNameChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onTagsChange: (value: Record<string, string>) => void;
  autoFocus?: boolean;
}

const OrganizationFormFields: React.FC<OrganizationFormFieldsProps> = ({
  name,
  description,
  tags,
  onNameChange,
  onDescriptionChange,
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
      rows={componentSpacing.textArea.rows}
      margin="normal"
    />
    <Box sx={{ mt: 3 }}>
      <Typography variant="subtitle2" gutterBottom>
        {ORGANIZATION_FORM.LABEL_TAGS}
      </Typography>
      <TagsEditor tags={tags} onChange={onTagsChange} />
    </Box>
  </>
);

export default OrganizationFormFields;
