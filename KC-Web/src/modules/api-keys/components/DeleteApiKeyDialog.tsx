import React from "react";

import { Typography } from "@mui/material";
import { ConfirmDialog } from "@ui";

import {
  API_KEY_FORM,
  API_KEYS_PAGE,
  CONFIRM_DELETE_API_KEY,
  ERR_DELETE_API_KEY,
} from "@constants/messages";

import type { useKeyDeletion } from "../hooks";

/** Confirms the deletion of the key the admin picked. */
const DeleteApiKeyDialog: React.FC<{
  deletion: ReturnType<typeof useKeyDeletion>;
}> = ({ deletion }) => (
  <ConfirmDialog
    open={!!deletion.keyToDelete}
    title={API_KEYS_PAGE.DIALOG_DELETE_TITLE}
    message={CONFIRM_DELETE_API_KEY}
    confirmLabel={API_KEYS_PAGE.ACTION_DELETE}
    cancelLabel={API_KEY_FORM.ACTION_CANCEL}
    errorFallback={ERR_DELETE_API_KEY}
    onConfirm={deletion.confirm}
    onClose={deletion.cancel}
  >
    {deletion.keyToDelete && (
      <Typography variant="body2" sx={{ mt: 2, fontWeight: "bold" }}>
        {deletion.keyToDelete.name}
      </Typography>
    )}
  </ConfirmDialog>
);

export default DeleteApiKeyDialog;
