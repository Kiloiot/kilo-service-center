import React from "react";

import { ConfirmDialog } from "@ui";

import { BASE_STATION_DETAILS } from "@constants/messages";

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
    <ConfirmDialog
      open={open}
      onClose={onClose}
      onConfirm={onConfirm}
      pending={isRegenerating}
      color="warning"
      title={BASE_STATION_DETAILS.REGENERATE_CERTS_CONFIRM_TITLE}
      message={BASE_STATION_DETAILS.REGENERATE_CERTS_CONFIRM_TEXT}
      confirmLabel={BASE_STATION_DETAILS.ACTION_REGENERATE}
      pendingLabel={BASE_STATION_DETAILS.ACTION_REGENERATE}
    />
  );
};

export default BaseStationCertRegenDialog;
