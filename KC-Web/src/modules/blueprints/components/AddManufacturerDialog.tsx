/**
 * Create-manufacturer dialog.
 */

import React, { useState } from "react";

import type { BlueprintScope, CreateManufacturerRequest } from "@api-types/api";
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
import { BLUEPRINT_SCOPE } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { componentSpacing } from "@theme/index";

import { useCreateManufacturer } from "../hooks";
import CatalogScopeNotice from "./CatalogScopeNotice";
import ManufacturerFormFields from "./ManufacturerFormFields";

/**
 * Add Manufacturer Dialog
 */
export interface AddManufacturerDialogProps {
  open: boolean;
  scope: BlueprintScope;
  onClose: () => void;
  onSuccess: () => void;
}

export const AddManufacturerDialog: React.FC<AddManufacturerDialogProps> = ({
  open,
  scope,
  onClose,
  onSuccess,
}) => {
  const [formData, setFormData] = useState<CreateManufacturerRequest>({
    name: "",
    isSystem: scope === BLUEPRINT_SCOPE.SYSTEM,
  });
  const [error, setError] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);

  // Default the System toggle to match the active scope tab.
  React.useEffect(() => {
    if (open) {
      setFormData({ name: "", isSystem: scope === BLUEPRINT_SCOPE.SYSTEM });
      setError(null);
      setNameError(null);
    }
  }, [open, scope]);

  const mutation = useCreateManufacturer();

  const handleSubmit = () => {
    if (!formData.name.trim()) {
      setNameError(BLUEPRINT_LABELS.ERR_NAME_REQUIRED);
      return;
    }
    mutation.mutate(formData, {
      onSuccess: () => {
        setFormData({ name: "", isSystem: scope === BLUEPRINT_SCOPE.SYSTEM });
        setError(null);
        onSuccess();
      },
      onError: (err: Error) => setError(getErrorMessage(err)),
    });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{BLUEPRINT_LABELS.ADD_MANUFACTURER}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <ManufacturerFormFields
          name={formData.name}
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
        <CatalogScopeNotice scope={scope} />
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
            BLUEPRINT_LABELS.ACTION_CREATE
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
