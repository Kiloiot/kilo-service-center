import React from "react";

import { ConfirmDialog } from "@ui";

import { formatEui } from "@utils/eui";
import {
  BASE_STATION_DETAILS,
  ERR_DELETE_BASE_STATION,
} from "@constants/messages";

interface BaseStationDeleteDialogProps {
  open: boolean;
  onClose: () => void;
  baseStationName: string | undefined;
  eui: string;
  onConfirm: () => Promise<void>;
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
    <ConfirmDialog
      open={open}
      onClose={onClose}
      onConfirm={onConfirm}
      pending={isPending}
      title={BASE_STATION_DETAILS.DIALOG_DELETE_TITLE}
      message={
        <>
          {BASE_STATION_DETAILS.DIALOG_DELETE_CONFIRM_PREFIX} &quot;
          {baseStationName || formatEui(eui)}&quot;?{" "}
          {BASE_STATION_DETAILS.DIALOG_DELETE_WARNING}
        </>
      }
      confirmLabel={BASE_STATION_DETAILS.DELETE}
      pendingLabel={BASE_STATION_DETAILS.DELETING}
      errorFallback={ERR_DELETE_BASE_STATION}
    />
  );
};

export default BaseStationDeleteDialog;
