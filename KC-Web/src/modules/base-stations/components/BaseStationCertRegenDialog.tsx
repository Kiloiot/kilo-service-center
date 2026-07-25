import React from "react";

import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from "@mui/material";

import { ACTION_CANCEL, BASE_STATION_DETAILS } from "@constants/messages";

interface BaseStationCertRegenDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  isRegenerating: boolean;
}

/** Confirmation dialog for certificate regeneration. */
const BaseStationCertRegenDialog: React.FC<BaseStationCertRegenDialogProps> = ({
  open,
  onClose,
  onConfirm,
  isRegenerating,
}) => {
  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>
        {BASE_STATION_DETAILS.REGENERATE_CERTS_CONFIRM_TITLE}
      </DialogTitle>
      <DialogContent>
        <DialogContentText>
          {BASE_STATION_DETAILS.REGENERATE_CERTS_CONFIRM_TEXT}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{ACTION_CANCEL}</Button>
        <Button onClick={onConfirm} color="warning" disabled={isRegenerating}>
          {BASE_STATION_DETAILS.ACTION_REGENERATE}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default BaseStationCertRegenDialog;
