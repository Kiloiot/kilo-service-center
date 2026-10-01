/**
 * Edit-device-model dialog.
 */

import React, { useState } from "react";

import type { DeviceModelUI, UpdateDeviceModelRequest } from "@api-types/api";
import {
  Alert,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
} from "@mui/material";

import { getErrorMessage } from "@utils/error-message";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { componentSpacing } from "@theme/index";

import { useUpdateDeviceModel } from "../hooks";

/**
 * Edit Device Model Dialog
 */
export interface EditDeviceModelDialogProps {
  model: DeviceModelUI | null;
  onClose: () => void;
  onSuccess: () => void;
}

export const EditDeviceModelDialog: React.FC<EditDeviceModelDialogProps> = ({
  model,
  onClose,
  onSuccess,
}) => {
  const [formData, setFormData] = useState<UpdateDeviceModelRequest>({});
  const [error, setError] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);

  // Reset form data when model changes
  React.useEffect(() => {
    if (model) {
      setFormData({
        name: model.name,
        description: model.description || undefined,
        datasheetUrl: model.datasheetUrl || undefined,
      });
      setError(null);
      setNameError(null);
    }
  }, [model]);

  const mutation = useUpdateDeviceModel();

  const handleSubmit = () => {
    if (!formData.name?.trim()) {
      setNameError(BLUEPRINT_LABELS.ERR_NAME_REQUIRED);
      return;
    }
    if (!model) {
      setError(BLUEPRINT_LABELS.ERR_NO_MODEL_SELECTED);
      return;
    }
    mutation.mutate(
      { id: model.id, data: formData },
      {
        onSuccess: () => {
          setError(null);
          onSuccess();
        },
        onError: (err: Error) => setError(getErrorMessage(err)),
      },
    );
  };

  return (
    <Dialog open={!!model} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{BLUEPRINT_LABELS.DIALOG_EDIT_MODEL}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <TextField
          autoFocus
          required
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_NAME}
          fullWidth
          value={formData.name || ""}
          onChange={(e) => {
            setFormData({ ...formData, name: e.target.value });
            setNameError(null);
          }}
          error={!!nameError}
          helperText={nameError}
        />
        <TextField
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_DESCRIPTION}
          fullWidth
          multiline
          rows={componentSpacing.textArea.compactRows}
          value={formData.description || ""}
          onChange={(e) =>
            setFormData({ ...formData, description: e.target.value })
          }
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{BLUEPRINT_LABELS.ACTION_CANCEL}</Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={mutation.isPending}
        >
          {mutation.isPending ? (
            <CircularProgress size={componentSpacing.spinner.button} />
          ) : (
            BLUEPRINT_LABELS.ACTION_SAVE
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
