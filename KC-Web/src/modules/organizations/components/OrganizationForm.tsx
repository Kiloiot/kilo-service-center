/**
 * Organization Form Component
 *
 * Editable organization configuration form with delete action.
 */

import React, { useEffect, useState } from "react";

import type { OrganizationUI, UpdateOrganizationRequest } from "@api-types/api";
import { Box, Button } from "@mui/material";
import { ConfirmDialog } from "@ui";

import { useFeedback } from "@contexts/feedback";
import {
  useDeleteOrganization,
  useUpdateOrganization,
} from "@hooks/useOrganizations";
import { getErrorMessage } from "@utils/error-message";
import {
  ERR_DELETE_ORGANIZATION,
  ORGANIZATION_FORM,
} from "@constants/messages";

import OrganizationFormFields from "./OrganizationFormFields";

interface OrganizationFormProps {
  organization: OrganizationUI;
  onDeleted?: () => void;
}

const OrganizationForm: React.FC<OrganizationFormProps> = ({
  organization,
  onDeleted,
}) => {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState<Record<string, string>>({});
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const feedback = useFeedback();

  const updateOrganization = useUpdateOrganization();
  const deleteOrganization = useDeleteOrganization();

  useEffect(() => {
    setName(organization.name);
    setDescription(organization.description ?? "");
    setTags(organization.tags ?? {});
  }, [organization]);

  const buildUpdateRequest = (): UpdateOrganizationRequest => {
    return {
      name: name.trim(),
      description: description.trim() || undefined,
      tags: Object.keys(tags).length > 0 ? tags : undefined,
    };
  };

  const handleSave = async () => {
    const payload = buildUpdateRequest();
    try {
      await updateOrganization.mutateAsync({
        id: organization.id,
        data: payload,
      });
      feedback.success(ORGANIZATION_FORM.SUCCESS_UPDATE);
    } catch (error) {
      feedback.error(
        getErrorMessage(error, ORGANIZATION_FORM.ERR_UPDATE_FAILED),
      );
    }
  };

  const handleDelete = async () => {
    await deleteOrganization.mutateAsync(organization.id);
    setConfirmDeleteOpen(false);
    onDeleted?.();
    feedback.success(ORGANIZATION_FORM.SUCCESS_DELETE);
  };

  return (
    <Box>
      <OrganizationFormFields
        name={name}
        description={description}
        tags={tags}
        onNameChange={setName}
        onDescriptionChange={setDescription}
        onTagsChange={setTags}
      />
      <Box sx={{ display: "flex", gap: 2, mt: 3 }}>
        <Button
          variant="contained"
          onClick={handleSave}
          disabled={!name.trim() || updateOrganization.isPending}
        >
          {ORGANIZATION_FORM.ACTION_SUBMIT}
        </Button>
        <Button
          variant="outlined"
          color="error"
          onClick={() => setConfirmDeleteOpen(true)}
          disabled={deleteOrganization.isPending}
        >
          {ORGANIZATION_FORM.ACTION_DELETE}
        </Button>
      </Box>

      <ConfirmDialog
        open={confirmDeleteOpen}
        title={ORGANIZATION_FORM.DIALOG_DELETE_TITLE}
        message={ORGANIZATION_FORM.CONFIRM_DELETE}
        confirmLabel={ORGANIZATION_FORM.ACTION_DELETE}
        cancelLabel={ORGANIZATION_FORM.ACTION_CANCEL}
        errorFallback={ERR_DELETE_ORGANIZATION}
        onConfirm={handleDelete}
        onClose={() => setConfirmDeleteOpen(false)}
      />
    </Box>
  );
};

export default OrganizationForm;
