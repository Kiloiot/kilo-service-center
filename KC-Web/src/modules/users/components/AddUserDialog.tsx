/**
 * AddUserDialog Component
 *
 * Dialog for creating a new system user and adding it to organizations:
 * the given orgId, or the ones picked (enterprise edition).
 */

import React from "react";

import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
} from "@mui/material";

import { NewPasswordFields } from "@components/common/NewPasswordFields";
import { USER_FORM } from "@constants/messages";
import { componentSpacing } from "@theme/index";

import { useAddUserForm } from "../hooks";
import NewUserFlags from "./NewUserFlags";
import NewUserOrganizations from "./NewUserOrganizations";

interface AddUserDialogProps {
  open: boolean;
  onClose: () => void;
  /** Optional org ID - if provided, user will be auto-added to this org after creation */
  orgId?: string;
}

type AddUserForm = ReturnType<typeof useAddUserForm>;

const CredentialFields: React.FC<{ state: AddUserForm }> = ({ state }) => (
  <>
    <TextField
      label={USER_FORM.LABEL_EMAIL}
      type="email"
      value={state.form.email}
      onChange={(e) => state.update("email", e.target.value)}
      error={!!state.errors.email}
      helperText={state.errors.email}
      fullWidth
      required
      autoFocus
    />
    <NewPasswordFields
      password={state.form.password}
      confirmation={state.form.confirmPassword}
      onPasswordChange={(value) => state.update("password", value)}
      onConfirmationChange={(value) => state.update("confirmPassword", value)}
      passwordLabel={USER_FORM.LABEL_PASSWORD}
      confirmationLabel={USER_FORM.LABEL_CONFIRM_PASSWORD}
      passwordError={state.errors.password}
      confirmationError={state.errors.confirmPassword}
      required
    />
    <TextField
      label={USER_FORM.LABEL_NOTE}
      value={state.form.note}
      onChange={(e) => state.update("note", e.target.value)}
      multiline
      rows={componentSpacing.textArea.compactRows}
      fullWidth
    />
  </>
);

const AddUserDialog: React.FC<AddUserDialogProps> = ({
  open,
  onClose,
  orgId,
}) => {
  const state = useAddUserForm(orgId, onClose);

  const handleClose = () => {
    state.reset();
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{USER_FORM.DIALOG_TITLE_ADD}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1 }}>
          {state.createError && (
            <Alert severity="error">{state.createError}</Alert>
          )}
          {state.partialError && (
            <Alert severity="warning">{state.partialError}</Alert>
          )}
          <CredentialFields state={state} />
          <NewUserFlags form={state.form} onChange={state.update} />
          {!orgId && (
            <NewUserOrganizations
              open={open}
              value={state.form.organizations}
              onChange={(orgs) => state.update("organizations", orgs)}
            />
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={state.pending}>
          {USER_FORM.ACTION_CANCEL}
        </Button>
        <Button
          variant="contained"
          onClick={state.submit}
          disabled={state.pending}
        >
          {USER_FORM.ACTION_SUBMIT}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default AddUserDialog;
