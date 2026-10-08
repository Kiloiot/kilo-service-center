import React from "react";

import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  TextField,
} from "@mui/material";

import { API_KEY_TYPES } from "@constants/app";
import { API_KEY_FORM, API_KEYS_PAGE } from "@constants/messages";

import { useApiKeyForm } from "../hooks";

interface CreateApiKeyDialogProps {
  open: boolean;
  onClose: () => void;
  onCreated: (rawKey: string) => void;
}

type ApiKeyForm = ReturnType<typeof useApiKeyForm>;

const ApiKeyFields: React.FC<{ form: ApiKeyForm }> = ({ form }) => (
  <Box display="flex" flexDirection="column" gap={2} mt={1}>
    <TextField
      required
      label={API_KEY_FORM.LABEL_NAME}
      placeholder={API_KEY_FORM.PLACEHOLDER_NAME}
      value={form.fields.name}
      onChange={(e) => form.setName(e.target.value)}
      fullWidth
      autoFocus
    />
    <FormControl fullWidth>
      <InputLabel>{API_KEY_FORM.LABEL_TYPE}</InputLabel>
      <Select
        value={form.fields.keyType}
        label={API_KEY_FORM.LABEL_TYPE}
        onChange={(e) => form.setKeyType(e.target.value)}
      >
        <MenuItem value={API_KEY_TYPES.USER}>
          {API_KEYS_PAGE.TYPE_USER}
        </MenuItem>
        <MenuItem value={API_KEY_TYPES.SERVICE_ACCOUNT}>
          {API_KEYS_PAGE.TYPE_SERVICE_ACCOUNT}
        </MenuItem>
      </Select>
    </FormControl>
    <TextField
      label={API_KEY_FORM.LABEL_EXPIRES_ON}
      placeholder={API_KEY_FORM.PLACEHOLDER_EXPIRES_ON}
      value={form.fields.expiresOn}
      onChange={(e) => form.setExpiresOn(e.target.value)}
      error={form.fields.expiresOnInvalid}
      helperText={
        form.fields.expiresOnInvalid
          ? API_KEY_FORM.ERR_EXPIRES_ON_INVALID
          : API_KEY_FORM.HELPER_EXPIRES_ON
      }
      fullWidth
      slotProps={{ inputLabel: { shrink: true } }}
    />
  </Box>
);

/** Creates an API key; a refusal shows the server's reason in the dialog. */
const CreateApiKeyDialog: React.FC<CreateApiKeyDialogProps> = ({
  open,
  onClose,
  onCreated,
}) => {
  const form = useApiKeyForm(onCreated);

  const handleClose = () => {
    form.reset();
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{API_KEY_FORM.DIALOG_TITLE_CREATE}</DialogTitle>
      <DialogContent>
        {form.error && (
          <Alert severity="error" sx={{ mt: 1 }}>
            {form.error}
          </Alert>
        )}
        <ApiKeyFields form={form} />
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={form.pending}>
          {API_KEY_FORM.ACTION_CANCEL}
        </Button>
        <Button
          onClick={form.submit}
          variant="contained"
          disabled={!form.canSubmit}
        >
          {API_KEY_FORM.ACTION_CREATE}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default CreateApiKeyDialog;
