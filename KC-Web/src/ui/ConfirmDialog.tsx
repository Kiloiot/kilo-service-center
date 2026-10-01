import React, { useEffect, useState } from "react";

import {
  Alert,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from "@mui/material";

import { getErrorMessage } from "@utils/error-message";
import { ACTION_CANCEL } from "@constants/messages";
import { componentSpacing } from "@theme/index";

export type ConfirmDialogColor = "primary" | "error" | "warning";

export interface ConfirmDialogProps {
  open: boolean;
  title: React.ReactNode;
  message: React.ReactNode;
  confirmLabel: string;
  /** Label shown while the action is pending; falls back to confirmLabel. */
  pendingLabel?: string;
  cancelLabel?: string;
  color?: ConfirmDialogColor;
  /** Externally tracked pending state (e.g. a mutation's isPending). */
  pending?: boolean;
  /** Externally tracked error shown above the message. */
  error?: string | null;
  /** Fallback shown when a rejected onConfirm carries no message. */
  errorFallback?: string;
  onConfirm: () => void | Promise<void>;
  onClose: () => void;
  children?: React.ReactNode;
}

/**
 * One confirmation dialog for every destructive or irreversible action.
 * A promise-returning onConfirm drives the pending and error state itself;
 * a synchronous onConfirm relies on the pending and error props.
 */
export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  open,
  title,
  message,
  confirmLabel,
  pendingLabel,
  cancelLabel = ACTION_CANCEL,
  color = "error",
  pending = false,
  error = null,
  errorFallback,
  onConfirm,
  onClose,
  children,
}) => {
  const [running, setRunning] = useState(false);
  const [runError, setRunError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setRunning(false);
      setRunError(null);
    }
  }, [open]);

  const handleConfirm = async () => {
    const result = onConfirm();
    if (!(result instanceof Promise)) return;
    setRunning(true);
    setRunError(null);
    try {
      await result;
    } catch (err) {
      setRunError(getErrorMessage(err, errorFallback ?? confirmLabel));
      setRunning(false);
    }
  };

  const busy = pending || running;
  const shownError = error ?? runError;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        {shownError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {shownError}
          </Alert>
        )}
        <DialogContentText component="div">{message}</DialogContentText>
        {children}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          {cancelLabel}
        </Button>
        <Button
          onClick={() => void handleConfirm()}
          color={color}
          variant="contained"
          disabled={busy}
        >
          {busy && !pendingLabel ? (
            <CircularProgress size={componentSpacing.spinner.button} />
          ) : busy ? (
            pendingLabel
          ) : (
            confirmLabel
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
