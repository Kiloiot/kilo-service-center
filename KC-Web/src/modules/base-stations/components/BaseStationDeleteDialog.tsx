import React from "react";

import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from "@mui/material";

import { formatEUIWithDashes } from "@utils/formatters";
import { ACTION_CANCEL, BASE_STATION_DETAILS } from "@constants/messages";

interface BaseStationDeleteDialogProps {
  open: boolean;
  onClose: () => void;
  baseStationName: string | undefined;
  eui: string;
  onConfirm: () => void;
  isPending: boolean;
}

/** Confirmation dialog for deleting a base station. */
const BaseStationDeleteDialog: React.FC<BaseStationDeleteDialogProps> = ({
  open,
  onClose,
  baseStationName,
  eui,
  onConfirm,
  isPending,
}) => {
  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>{BASE_STATION_DETAILS.DIALOG_DELETE_TITLE}</DialogTitle>
      <DialogContent>
        <DialogContentText>
          {BASE_STATION_DETAILS.DIALOG_DELETE_CONFIRM_PREFIX} &quot;
          {baseStationName || formatEUIWithDashes(eui)}&quot;?{" "}
          {BASE_STATION_DETAILS.DIALOG_DELETE_WARNING}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{ACTION_CANCEL}</Button>
        <Button
          onClick={onConfirm}
          color="error"
          variant="contained"
          disabled={isPending}
        >
          {isPending
            ? BASE_STATION_DETAILS.DELETING
            : BASE_STATION_DETAILS.DELETE}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default BaseStationDeleteDialog;
