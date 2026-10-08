/**
 * Confirms deleting an end point; an attached end point is noted as being
 * detached from its base stations first.
 */

import { ConfirmDialog } from "@ui";

import { ACTION_DELETE, ENDPOINT_DETAILS } from "@constants/messages";

interface EndpointDeleteDialogProps {
  open: boolean;
  name: string;
  attached: boolean;
  pending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export function EndpointDeleteDialog({
  open,
  name,
  attached,
  pending,
  onClose,
  onConfirm,
}: EndpointDeleteDialogProps) {
  return (
    <ConfirmDialog
      open={open}
      onClose={onClose}
      onConfirm={onConfirm}
      pending={pending}
      title={ENDPOINT_DETAILS.DIALOG_DELETE_TITLE}
      message={
        <>
          {ENDPOINT_DETAILS.DIALOG_DELETE_CONFIRM_PREFIX} ({name})?
          {attached && (
            <>
              <br />
              <br />
              <strong>{ENDPOINT_DETAILS.DIALOG_DELETE_NOTE_PREFIX}</strong>{" "}
              {ENDPOINT_DETAILS.DIALOG_DELETE_NOTE}
            </>
          )}
          <br />
          <br />
          {ENDPOINT_DETAILS.DIALOG_DELETE_WARNING}
        </>
      }
      confirmLabel={ACTION_DELETE}
      pendingLabel={ENDPOINT_DETAILS.ACTION_DELETING}
    />
  );
}
