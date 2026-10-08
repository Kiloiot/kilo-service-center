/**
 * Edit-manufacturer dialog.
 */

import React, { useState } from "react";

import type { ManufacturerUI, UpdateManufacturerRequest } from "@api-types/api";
import {
  Alert,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from "@mui/material";

import { getErrorMessage } from "@utils/error-message";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { componentSpacing } from "@theme/index";

import { useUpdateManufacturer } from "../hooks";
import ManufacturerFormFields from "./ManufacturerFormFields";

/**
 * Edit Manufacturer Dialog
 */
export interface EditManufacturerDialogProps {
  manufacturer: ManufacturerUI | null;
  onClose: () => void;
  onSuccess: () => void;
}

export const EditManufacturerDialog: React.FC<EditManufacturerDialogProps> = ({
  manufacturer,
  onClose,
  onSuccess,
}) => {
  const [formData, setFormData] = useState<UpdateManufacturerRequest>({});
  const [error, setError] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);

  // Reset form data when manufacturer changes
  React.useEffect(() => {
    if (manufacturer) {
      setFormData({
        name: manufacturer.name,
        website: manufacturer.website || undefined,
      });
      setError(null);
      setNameError(null);
    }
  }, [manufacturer]);

  const mutation = useUpdateManufacturer();

  const handleSubmit = () => {
    if (!formData.name?.trim()) {
      setNameError(BLUEPRINT_LABELS.ERR_NAME_REQUIRED);
      return;
    }
    if (!manufacturer) {
      setError(BLUEPRINT_LABELS.ERR_NO_MANUFACTURER_SELECTED);
      return;
    }
    mutation.mutate(
      { id: manufacturer.id, data: formData },
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
    <Dialog open={!!manufacturer} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{BLUEPRINT_LABELS.DIALOG_EDIT_MANUFACTURER}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <ManufacturerFormFields
          name={formData.name || ""}
          nameError={nameError}
          website={formData.website || ""}
          onNameChange={(value) => {
            setFormData({ ...formData, name: value });
            setNameError(null);
          }}
          onWebsiteChange={(value) =>
            setFormData({ ...formData, website: value })
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
