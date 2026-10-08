import React, { useEffect, useState } from "react";

import type { EndpointUI } from "@api-types/api";
import { useAttachEndpoint, useUpdateEndpoint } from "@hooks";
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from "@mui/material";

import { useFeedback } from "@contexts/feedback";
import { getErrorMessage } from "@utils/error-message";
import { ENDPOINT_FORM } from "@constants/messages";

import { useEndpointForm } from "../hooks";
import AttachStatusChip from "./AttachStatusChip";
import { EndpointFormSections } from "./EndpointFormSections";
import ReattachPendingBadge from "./ReattachPendingBadge";
import { ReattachPromptDialog } from "./ReattachPromptDialog";

interface EditEndPointDialogProps {
  open: boolean;
  onClose: () => void;
  endpoint: EndpointUI | null;
}

/**
 * Edits a registered end point with the same form as Add End Point; only
 * changed fields are sent, and an EUI change cascades to dependent tables.
 */
const EditEndPointDialog: React.FC<EditEndPointDialogProps> = ({
  open,
  onClose,
  endpoint,
}) => {
  const updateEndpointMutation = useUpdateEndpoint();
  const attachEndpointMutation = useAttachEndpoint();

  const form = useEndpointForm();
  const { reset: resetForm } = form;
  const [reattachFields, setReattachFields] = useState<string[]>([]);
  const showReattachPrompt = reattachFields.length > 0;
  const feedback = useFeedback();

  useEffect(() => {
    if (open && endpoint) resetForm(endpoint);
  }, [open, endpoint, resetForm]);

  const handleClose = () => {
    form.reset();
    setReattachFields([]);
    onClose();
  };

  const handleSubmit = () => {
    if (!endpoint) return;
    if (!form.validate()) return;

    const changedFields = form.buildUpdateRequest();
    if (Object.keys(changedFields).length === 0) {
      handleClose();
      return;
    }

    updateEndpointMutation.mutate(
      { epEui: endpoint.epEui, data: changedFields },
      {
        onSuccess: (updated) => {
          feedback.success(ENDPOINT_FORM.MSG_ENDPOINT_UPDATED);
          const changedProfile = form.changedProfileFields();
          if (updated.reattachPending && changedProfile.length > 0) {
            setReattachFields(changedProfile);
          } else {
            handleClose();
          }
        },
        onError: (error) => {
          const message = getErrorMessage(error, "");
          form.setGeneralError(
            message
              ? `${ENDPOINT_FORM.ERROR_UPDATE_PREFIX}${message}`
              : ENDPOINT_FORM.ERROR_UPDATE_GENERIC,
          );
        },
      },
    );
  };

  const handleReattach = () => {
    if (!endpoint) return;
    attachEndpointMutation.mutate(form.values.epEui, {
      onSettled: handleClose,
    });
  };

  const hasChanges = form.isDirty;

  if (!endpoint) return null;

  return (
    <>
      <Dialog
        open={open && !showReattachPrompt}
        onClose={handleClose}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          {ENDPOINT_FORM.EDIT_DIALOG_TITLE}
          <Typography variant="body2" color="text.secondary">
            {ENDPOINT_FORM.EDIT_DIALOG_SUBTITLE}
          </Typography>
          <Box mt={1} display="flex" gap={1} flexWrap="wrap">
            <AttachStatusChip status={endpoint.attachStatus} />
            <ReattachPendingBadge pending={endpoint.reattachPending} />
          </Box>
        </DialogTitle>
        <DialogContent>
          {form.errors.general && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {form.errors.general}
            </Alert>
          )}

          <EndpointFormSections form={form} />
        </DialogContent>
        <DialogActions>
          {!hasChanges && (
            <Typography variant="body2" color="text.secondary" sx={{ mr: 1 }}>
              {ENDPOINT_FORM.MSG_NO_CHANGES}
            </Typography>
          )}
          <Button onClick={handleClose}>{ENDPOINT_FORM.BUTTON_CANCEL}</Button>
          <Button
            onClick={handleSubmit}
            variant="contained"
            disabled={updateEndpointMutation.isPending || !hasChanges}
          >
            {updateEndpointMutation.isPending
              ? ENDPOINT_FORM.BUTTON_UPDATING
              : ENDPOINT_FORM.BUTTON_UPDATE}
          </Button>
        </DialogActions>
      </Dialog>

      <ReattachPromptDialog
        open={showReattachPrompt}
        pending={attachEndpointMutation.isPending}
        changedFields={reattachFields}
        onReattach={handleReattach}
        onLater={handleClose}
      />
    </>
  );
};

export default EditEndPointDialog;
