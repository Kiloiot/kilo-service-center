import React from "react";

import {
  Box,
  Button,
  Card,
  CardContent,
  FormControlLabel,
  Switch,
  TextField,
  Typography,
} from "@mui/material";

import { USER_FORM } from "@constants/messages";
import { DeleteIcon, SaveIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import type { UserProfileForm } from "../utils/user-profile";

interface UserProfileCardProps {
  form: UserProfileForm;
  isSaving: boolean;
  isDeleting: boolean;
  onChange: <K extends keyof UserProfileForm>(
    key: K,
    value: UserProfileForm[K],
  ) => void;
  onSave: () => void;
  onDeleteClick: () => void;
}

const UserProfileCard: React.FC<UserProfileCardProps> = ({
  form,
  isSaving,
  isDeleting,
  onChange,
  onSave,
  onDeleteClick,
}) => (
  <Card>
    <CardContent>
      <Typography variant="h6" gutterBottom>
        {USER_FORM.DIALOG_TITLE_EDIT}
      </Typography>

      <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 2 }}>
        <TextField
          label={USER_FORM.LABEL_EMAIL}
          type="email"
          value={form.email}
          onChange={(e) => onChange("email", e.target.value)}
          fullWidth
        />

        <TextField
          label={USER_FORM.LABEL_NOTE}
          value={form.note}
          onChange={(e) => onChange("note", e.target.value)}
          multiline
          rows={componentSpacing.textArea.compactRows}
          fullWidth
        />

        <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
          <FormControlLabel
            control={
              <Switch
                checked={form.isActive}
                onChange={(e) => onChange("isActive", e.target.checked)}
              />
            }
            label={USER_FORM.LABEL_IS_ACTIVE}
          />
          <FormControlLabel
            control={
              <Switch
                checked={form.isUserAdmin}
                onChange={(e) => onChange("isUserAdmin", e.target.checked)}
              />
            }
            label={USER_FORM.LABEL_IS_ADMIN}
          />
          <FormControlLabel
            control={
              <Switch
                checked={form.isTenantManager}
                onChange={(e) => onChange("isTenantManager", e.target.checked)}
              />
            }
            label={USER_FORM.LABEL_IS_TENANT_MGR}
          />
          <FormControlLabel
            control={
              <Switch
                checked={form.isBaseStationManager}
                onChange={(e) =>
                  onChange("isBaseStationManager", e.target.checked)
                }
              />
            }
            label={USER_FORM.LABEL_IS_BS_MGR}
          />
          <FormControlLabel
            control={
              <Switch
                checked={form.isEndpointManager}
                onChange={(e) =>
                  onChange("isEndpointManager", e.target.checked)
                }
              />
            }
            label={USER_FORM.LABEL_IS_EP_MGR}
          />
        </Box>

        <Box display="flex" gap={2} mt={2}>
          <Button
            variant="contained"
            size="small"
            startIcon={<SaveIcon />}
            onClick={onSave}
            disabled={isSaving}
          >
            {USER_FORM.ACTION_SUBMIT}
          </Button>
          <Button
            variant="outlined"
            size="small"
            color="error"
            startIcon={<DeleteIcon />}
            onClick={onDeleteClick}
            disabled={isDeleting}
          >
            {USER_FORM.ACTION_DELETE}
          </Button>
        </Box>
      </Box>
    </CardContent>
  </Card>
);

export default UserProfileCard;
