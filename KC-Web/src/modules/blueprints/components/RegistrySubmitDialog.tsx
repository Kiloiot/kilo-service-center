/**
 * Registry Submit Dialog
 *
 * Dialog for submitting a blueprint to the registry.
 * Shows form for description and displays success state with PR URL.
 */

import React, { useState } from "react";

import type { RegistrySubmitResponse } from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Link,
  TextField,
  Typography,
} from "@mui/material";

import { getErrorMessage } from "@utils/error-message";
import { SHORT_COMMIT_SHA_LENGTH } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { CheckCircleIcon, OpenInNewIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import { useSubmitToRegistry } from "../hooks";

interface RegistrySubmitDialogProps {
  open: boolean;
  blueprintId: string | null;
  onClose: () => void;
  onSuccess: () => void;
}

export const RegistrySubmitDialog: React.FC<RegistrySubmitDialogProps> = ({
  open,
  blueprintId,
  onClose,
  onSuccess,
}) => {
  const [contributorName, setContributorName] = useState("");
  const [contributorEmail, setContributorEmail] = useState("");
  const [description, setDescription] = useState("");
  const [result, setResult] = useState<RegistrySubmitResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);
  const [emailError, setEmailError] = useState<string | null>(null);

  const mutation = useSubmitToRegistry(blueprintId);

  const handleSubmit = () => {
    const missingName = contributorName.trim()
      ? null
      : BLUEPRINT_LABELS.ERR_CONTRIBUTOR_NAME_REQUIRED;
    const missingEmail = contributorEmail.trim()
      ? null
      : BLUEPRINT_LABELS.ERR_CONTRIBUTOR_EMAIL_REQUIRED;
    setNameError(missingName);
    setEmailError(missingEmail);
    if (missingName || missingEmail) return;
    setError(null);
    mutation.mutate(
      {
        contributorName: contributorName || undefined,
        contributorEmail: contributorEmail || undefined,
        description: description || undefined,
      },
      {
        onSuccess: (data) => {
          setResult(data);
          setError(null);
        },
        onError: (err: Error) => setError(getErrorMessage(err)),
      },
    );
  };

  const handleClose = () => {
    // If we have a result, also trigger the success callback
    if (result) {
      onSuccess();
    }
    // Reset state
    setContributorName("");
    setContributorEmail("");
    setDescription("");
    setResult(null);
    setError(null);
    setNameError(null);
    setEmailError(null);
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{BLUEPRINT_LABELS.DIALOG_SUBMIT_TO_REGISTRY}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {result ? (
          // Success state
          <Box sx={{ textAlign: "center", py: 2 }}>
            <CheckCircleIcon
              color="success"
              sx={{ fontSize: componentSpacing.resultIcon.size, mb: 2 }}
            />
            <Typography variant="h6" gutterBottom>
              {BLUEPRINT_LABELS.REGISTRY_SUBMIT_SUCCESS}
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              {BLUEPRINT_LABELS.REGISTRY_PR_CREATED}
            </Typography>
            <Link
              href={result.prUrl}
              target="_blank"
              rel="noopener noreferrer"
              sx={{ display: "inline-flex", alignItems: "center", gap: 0.5 }}
            >
              {BLUEPRINT_LABELS.REGISTRY_VIEW_PR}
              <OpenInNewIcon fontSize="small" />
            </Link>
            <Typography
              variant="caption"
              display="block"
              sx={{ mt: 2 }}
              color="text.secondary"
            >
              {BLUEPRINT_LABELS.REGISTRY_BRANCH} {result.branch}
            </Typography>
            <Typography
              variant="caption"
              display="block"
              color="text.secondary"
            >
              {BLUEPRINT_LABELS.REGISTRY_COMMIT}{" "}
              {result.commitSha.substring(0, SHORT_COMMIT_SHA_LENGTH)}
            </Typography>
          </Box>
        ) : (
          // Form state
          <>
            <TextField
              autoFocus
              required
              margin="dense"
              label={BLUEPRINT_LABELS.LABEL_CONTRIBUTOR_NAME}
              fullWidth
              value={contributorName}
              onChange={(e) => {
                setContributorName(e.target.value);
                setNameError(null);
              }}
              error={!!nameError}
              helperText={nameError}
              sx={{ mb: 1 }}
            />
            <TextField
              required
              margin="dense"
              label={BLUEPRINT_LABELS.LABEL_CONTRIBUTOR_EMAIL}
              type="email"
              fullWidth
              value={contributorEmail}
              onChange={(e) => {
                setContributorEmail(e.target.value);
                setEmailError(null);
              }}
              error={!!emailError}
              helperText={emailError}
              sx={{ mb: 1 }}
            />
            <TextField
              margin="dense"
              label={BLUEPRINT_LABELS.LABEL_DESCRIPTION}
              fullWidth
              multiline
              rows={componentSpacing.textArea.tallRows}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={BLUEPRINT_LABELS.REGISTRY_DESCRIPTION_PLACEHOLDER}
            />
          </>
        )}
      </DialogContent>
      <DialogActions>
        {result ? (
          <Button onClick={handleClose} variant="contained">
            {BLUEPRINT_LABELS.ACTION_CLOSE}
          </Button>
        ) : (
          <>
            <Button onClick={handleClose} disabled={mutation.isPending}>
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
                BLUEPRINT_LABELS.SUBMIT_TO_REGISTRY
              )}
            </Button>
          </>
        )}
      </DialogActions>
    </Dialog>
  );
};
