/**
 * Add-blueprint (decoder) dialog for a device model.
 */

import React, { useState } from "react";

import type { DeviceModelUI } from "@api-types/api";
import {
  Alert,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  TextField,
} from "@mui/material";

import { getErrorMessage } from "@utils/error-message";
import { BLUEPRINT_SPEC_ROWS } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { componentSpacing } from "@theme/index";

import { useCreateBlueprint } from "../hooks";
import { specJsonError, validateBlueprintSpecJson } from "../utils/spec";
import SpecJsonField from "./SpecJsonField";

/**
 * Add Blueprint (Decoder) Dialog
 */
export interface AddBlueprintDialogProps {
  model: DeviceModelUI | null;
  onClose: () => void;
  onSuccess: () => void;
}

export const AddBlueprintDialog: React.FC<AddBlueprintDialogProps> = ({
  model,
  onClose,
  onSuccess,
}) => {
  const [version, setVersion] = useState("");
  const [specJson, setSpecJson] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [versionError, setVersionError] = useState<string | null>(null);
  const [specError, setSpecError] = useState<string | null>(null);

  React.useEffect(() => {
    if (model) {
      setVersion("");
      setSpecJson("");
      setError(null);
      setVersionError(null);
      setSpecError(null);
    }
  }, [model]);

  const mutation = useCreateBlueprint();

  const handleSubmit = () => {
    if (!model) {
      setError(BLUEPRINT_LABELS.ERR_CREATE_BLUEPRINT);
      return;
    }
    const missingVersion = version.trim()
      ? null
      : BLUEPRINT_LABELS.ERR_VERSION_REQUIRED;
    const invalidSpec = specJsonError(specJson);
    setVersionError(missingVersion);
    setSpecError(invalidSpec);
    if (missingVersion || invalidSpec) return;
    const parsedSpec = validateBlueprintSpecJson(specJson);
    mutation.mutate(
      {
        deviceModelId: model.id,
        data: {
          version: version.trim(),
          specJson: parsedSpec,
          // Child ownership mirrors the parent model (System vs Custom).
          isSystem: model.isSystem,
        },
      },
      {
        onSuccess: () => {
          setError(null);
          onSuccess();
        },
        onError: (err: Error) =>
          setError(getErrorMessage(err, BLUEPRINT_LABELS.ERR_CREATE_BLUEPRINT)),
      },
    );
  };

  return (
    <Dialog open={!!model} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{BLUEPRINT_LABELS.ADD_DECODER}</DialogTitle>
      <DialogContent>
        <DialogContentText sx={{ mb: 2 }}>
          {model ? `${BLUEPRINT_LABELS.LABEL_NAME}: ${model.name}` : ""}
        </DialogContentText>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <TextField
          autoFocus
          required
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_VERSION}
          fullWidth
          value={version}
          onChange={(e) => {
            setVersion(e.target.value);
            setVersionError(null);
          }}
          error={!!versionError}
          helperText={versionError}
        />
        <SpecJsonField
          label={BLUEPRINT_LABELS.LABEL_SPEC_JSON}
          value={specJson}
          onChange={(value) => {
            setSpecJson(value);
            setSpecError(null);
          }}
          rows={BLUEPRINT_SPEC_ROWS.DIALOG}
          error={specError}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={mutation.isPending}>
          {BLUEPRINT_LABELS.ACTION_CANCEL}
        </Button>
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
