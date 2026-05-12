/**
 * Add Organization Dialog Component
 *
 * Dialog for creating new organizations with full field support.
 */

import React, { useState } from "react";

import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from "@mui/material";

import { useCreateOrganization } from "@hooks/useOrganizations";
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
  const [canHaveBaseStations, setCanHaveBaseStations] = useState(true);
  const [maxBsCount, setMaxBsCount] = useState("");
  const [maxEpCount, setMaxEpCount] = useState("");
  const [tags, setTags] = useState<Record<string, string>>({});

  const { mutate: createOrg, isPending } = useCreateOrganization();

  const handleReset = () => {
    setName("");
    setDescription("");
    setCanHaveBaseStations(true);
    setMaxBsCount("");
    setMaxEpCount("");
    setTags({});
  };

  const handleClose = () => {
    handleReset();
    onClose();
  };

  const handleSubmit = () => {
    // Convert 0 or empty to undefined (NULL in DB = unlimited)
    const parsedMaxBs = maxBsCount ? parseInt(maxBsCount, 10) : undefined;
    const parsedMaxEp = maxEpCount ? parseInt(maxEpCount, 10) : undefined;

    createOrg(
      {
        name,
        description: description || undefined,
        can_have_base_stations: canHaveBaseStations,
        max_base_station_count: parsedMaxBs === 0 ? undefined : parsedMaxBs,
        max_endpoint_count: parsedMaxEp === 0 ? undefined : parsedMaxEp,
        tags: Object.keys(tags).length > 0 ? tags : undefined,
      },
      {
        onSuccess: () => {
          handleClose();
        },
      },
    );
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{ORGANIZATION_FORM.DIALOG_TITLE_ADD}</DialogTitle>
      <DialogContent>
        <OrganizationFormFields
          name={name}
          description={description}
          canHaveBaseStations={canHaveBaseStations}
          maxBsCount={maxBsCount}
          maxEpCount={maxEpCount}
          tags={tags}
          onNameChange={setName}
          onDescriptionChange={setDescription}
          onCanHaveBaseStationsChange={setCanHaveBaseStations}
          onMaxBsCountChange={setMaxBsCount}
          onMaxEpCountChange={setMaxEpCount}
          onTagsChange={setTags}
          autoFocus
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>{ORGANIZATION_FORM.ACTION_CANCEL}</Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={!name || isPending}
        >
          {ORGANIZATION_FORM.ACTION_SUBMIT}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default AddOrganizationDialog;
