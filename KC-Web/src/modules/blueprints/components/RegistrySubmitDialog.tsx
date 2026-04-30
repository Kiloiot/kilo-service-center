/**
 * Registry Submit Dialog
 *
 * Dialog for submitting a blueprint to the registry.
 * Shows form for description and displays success state with PR URL.
 */

import React, { useState } from 'react';

import type { RegistrySubmitResponse } from '@api-types/api';
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
} from '@mui/material';
import { useMutation } from '@tanstack/react-query';

import { api } from '@services/api';
import { BLUEPRINT_LABELS } from '@constants/messages';
import { CheckCircleIcon, OpenInNewIcon } from '@theme/icons';

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
  const [contributorName, setContributorName] = useState('');
  const [contributorEmail, setContributorEmail] = useState('');
  const [description, setDescription] = useState('');
  const [result, setResult] = useState<RegistrySubmitResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: async () => {
      if (!blueprintId) throw new Error(BLUEPRINT_LABELS.ERR_BLUEPRINT_ID_REQUIRED);
      return api.submitToRegistry(blueprintId, {
        contributorName: contributorName || undefined,
        contributorEmail: contributorEmail || undefined,
        description: description || undefined,
      });
    },
    onSuccess: (data) => {
      setResult(data);
      setError(null);
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  const handleSubmit = () => {
    if (!contributorName.trim()) {
      setError(BLUEPRINT_LABELS.ERR_CONTRIBUTOR_NAME_REQUIRED);
      return;
    }
    if (!contributorEmail.trim()) {
      setError(BLUEPRINT_LABELS.ERR_CONTRIBUTOR_EMAIL_REQUIRED);
      return;
    }
    setError(null);
    mutation.mutate();
  };

  const handleClose = () => {
    // If we have a result, also trigger the success callback
    if (result) {
      onSuccess();
    }
    // Reset state
    setContributorName('');
    setContributorEmail('');
    setDescription('');
    setResult(null);
    setError(null);
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
          <Box sx={{ textAlign: 'center', py: 2 }}>
            <CheckCircleIcon color="success" sx={{ fontSize: 48, mb: 2 }} />
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
              sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
            >
              {BLUEPRINT_LABELS.REGISTRY_VIEW_PR}
              <OpenInNewIcon fontSize="small" />
            </Link>
            <Typography variant="caption" display="block" sx={{ mt: 2 }} color="text.secondary">
              {BLUEPRINT_LABELS.REGISTRY_BRANCH} {result.branch}
            </Typography>
            <Typography variant="caption" display="block" color="text.secondary">
              {BLUEPRINT_LABELS.REGISTRY_COMMIT} {result.commitSha.substring(0, 7)}
            </Typography>
          </Box>
        ) : (
          // Form state
          <>
            <TextField
              autoFocus
              margin="dense"
              label={BLUEPRINT_LABELS.LABEL_CONTRIBUTOR_NAME}
              fullWidth
              value={contributorName}
              onChange={(e) => setContributorName(e.target.value)}
              sx={{ mb: 1 }}
            />
            <TextField
              margin="dense"
              label={BLUEPRINT_LABELS.LABEL_CONTRIBUTOR_EMAIL}
              type="email"
              fullWidth
              value={contributorEmail}
              onChange={(e) => setContributorEmail(e.target.value)}
              sx={{ mb: 1 }}
            />
            <TextField
              margin="dense"
              label={BLUEPRINT_LABELS.LABEL_DESCRIPTION}
              fullWidth
              multiline
              rows={4}
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
            <Button onClick={handleSubmit} variant="contained" disabled={mutation.isPending}>
              {mutation.isPending ? (
                <CircularProgress size={20} />
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

export default RegistrySubmitDialog;
