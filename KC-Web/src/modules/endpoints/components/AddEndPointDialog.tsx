import React from "react";
import { useNavigate } from "react-router-dom";

import { useCreateEndpoint } from "@hooks";
import {
  Alert,
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
import { endpointDetailPath } from "@router/paths";

import { useEndpointForm } from "../hooks";
import { EndpointFormSections } from "./EndpointFormSections";

interface AddEndPointDialogProps {
  open: boolean;
  onClose: () => void;
}

/** Registers an end point and opens it, so a list filter cannot hide the new entry. */
const AddEndPointDialog: React.FC<AddEndPointDialogProps> = ({
  open,
  onClose,
}) => {
  const createEndpointMutation = useCreateEndpoint();
  const form = useEndpointForm();
  const navigate = useNavigate();
  const feedback = useFeedback();

  const handleClose = () => {
    form.reset();
    onClose();
  };

  const handleSubmit = () => {
    if (!form.validate()) return;
    createEndpointMutation.mutate(form.buildCreateRequest(), {
      onSuccess: (created) => {
        handleClose();
        navigate(endpointDetailPath(created.epEui));
        feedback.success(ENDPOINT_FORM.MSG_ENDPOINT_CREATED);
      },
      onError: (error) => {
        const message = getErrorMessage(error, "");
        form.setGeneralError(
          message
            ? ENDPOINT_FORM.ERROR_CREATE_PREFIX + message
            : ENDPOINT_FORM.ERROR_CREATE_GENERIC,
        );
      },
    });
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>
        {ENDPOINT_FORM.DIALOG_TITLE}
        <Typography variant="body2" color="text.secondary">
          {ENDPOINT_FORM.DIALOG_SUBTITLE}
        </Typography>
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
        <Button onClick={handleClose}>{ENDPOINT_FORM.BUTTON_CANCEL}</Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={createEndpointMutation.isPending}
        >
          {createEndpointMutation.isPending
            ? ENDPOINT_FORM.BUTTON_CREATING
            : ENDPOINT_FORM.BUTTON_CREATE}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default AddEndPointDialog;
