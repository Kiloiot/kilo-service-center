/**
 * Add Organization Dialog Component
 *
 * Dialog for creating new organizations with full field support.
 */

import React, { useState } from "react";

import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from "@mui/material";

import { useFeedback } from "@contexts/feedback";
import { useCreateOrganization } from "@hooks/useOrganizations";
import { getErrorMessage } from "@utils/error-message";
import { ORGANIZATION_FORM } from "@constants/messages";

import OrganizationFormFields from "./OrganizationFormFields";

interface AddOrganizationDialogProps {
  open: boolean;
  onClose: () => void;
}

const AddOrganizationDialog: React.FC<AddOrganizationDialogProps> = ({
  open,
  onClose,
}) => {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState<Record<string, string>>({});

  const createOrganization = useCreateOrganization();
  const feedback = useFeedback();

  const handleReset = () => {
    setName("");
    setDescription("");
    setTags({});
    createOrganization.reset();
  };

  const handleClose = () => {
    handleReset();
    onClose();
  };

  const handleSubmit = () => {
    createOrganization.mutate(
      {
        name,
        description: description || undefined,
        tags: Object.keys(tags).length > 0 ? tags : undefined,
      },
      {
        onSuccess: () => {
          handleClose();
          feedback.success(ORGANIZATION_FORM.SUCCESS_CREATE);
        },
      },
    );
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{ORGANIZATION_FORM.DIALOG_TITLE_ADD}</DialogTitle>
      <DialogContent>
        {createOrganization.isError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {getErrorMessage(
              createOrganization.error,
              ORGANIZATION_FORM.ERR_CREATE_FAILED,
            )}
          </Alert>
        )}
        <OrganizationFormFields
          name={name}
          description={description}
          tags={tags}
          onNameChange={setName}
          onDescriptionChange={setDescription}
          onTagsChange={setTags}
          autoFocus
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>{ORGANIZATION_FORM.ACTION_CANCEL}</Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={!name || createOrganization.isPending}
        >
          {ORGANIZATION_FORM.ACTION_SUBMIT}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default AddOrganizationDialog;
