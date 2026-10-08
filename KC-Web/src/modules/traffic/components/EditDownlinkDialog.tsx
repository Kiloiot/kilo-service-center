/**
 * Edit pending downlink (UpdatePendingDownlink): the dlDataQue fields of a
 * downlink no base station has taken yet, prefilled from the queue entry.
 */

import { useState } from "react";

import type { SCACIDownlinkQueueDTO } from "@api-types/api";
import { useUpdatePendingDownlink } from "@hooks";
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from "@mui/material";

import { getErrorMessage } from "@utils/error-message";
import { ACTION_CANCEL, EDIT_DOWNLINK_DIALOG } from "@constants/messages";

import { useDownlinkForm } from "../hooks";
import { downlinkFormInit } from "../utils/downlink-form";
import { DownlinkFormFields } from "./DownlinkFormFields";

interface EditDownlinkDialogProps {
  downlink: SCACIDownlinkQueueDTO;
  onClose: () => void;
  onSaved: () => void;
}

export function EditDownlinkDialog({
  downlink,
  onClose,
  onSaved,
}: EditDownlinkDialogProps) {
  const form = useDownlinkForm(downlinkFormInit(downlink));
  const update = useUpdatePendingDownlink();
  const [error, setError] = useState<string | null>(null);

  const handleSave = async () => {
    setError(null);
    try {
      await update.mutateAsync({
        epEui: downlink.epEui,
        queId: downlink.queId,
        ...form.buildContent(),
      });
      onSaved();
    } catch (err) {
      setError(getErrorMessage(err, EDIT_DOWNLINK_DIALOG.TITLE));
    }
  };

  return (
    <Dialog open onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        {EDIT_DOWNLINK_DIALOG.TITLE} {downlink.queId}
      </DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <DownlinkFormFields form={form} />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={update.isPending}>
          {ACTION_CANCEL}
        </Button>
        <Button
          variant="contained"
          onClick={() => void handleSave()}
          disabled={update.isPending || !form.isValid}
        >
          {update.isPending
            ? EDIT_DOWNLINK_DIALOG.ACTION_SAVING
            : EDIT_DOWNLINK_DIALOG.ACTION_SAVE}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
