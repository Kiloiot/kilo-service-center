/**
 * Add Organization Dialog Component
 *
 * Dialog for creating new organizations with full field support.
 */

import React, { useState } from "react";

import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Switch,
  TextField,
  Typography,
} from "@mui/material";

import { useCreateOrganization } from "@hooks/useOrganizations";
import { ORGANIZATION_FORM } from "@constants/messages";

import TagsEditor from "./TagsEditor";

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
        <TextField
          label={ORGANIZATION_FORM.LABEL_NAME}
          value={name}
          onChange={(e) => setName(e.target.value)}
          helperText={ORGANIZATION_FORM.HELPER_NAME}
          fullWidth
          required
          margin="normal"
          autoFocus
        />
        <TextField
          label={ORGANIZATION_FORM.LABEL_DESCRIPTION}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          helperText={ORGANIZATION_FORM.HELPER_DESCRIPTION}
          fullWidth
          multiline
          rows={3}
          margin="normal"
        />
        <FormControlLabel
          control={
            <Switch
              checked={canHaveBaseStations}
              onChange={(e) => setCanHaveBaseStations(e.target.checked)}
            />
          }
          label={ORGANIZATION_FORM.LABEL_CAN_HAVE_BS}
          sx={{ mt: 1 }}
        />
        <Box sx={{ display: "flex", gap: 2, mt: 2 }}>
          <TextField
            label={ORGANIZATION_FORM.LABEL_MAX_BS_COUNT}
            value={maxBsCount}
            onChange={(e) => setMaxBsCount(e.target.value.replace(/\D/g, ""))}
            helperText={ORGANIZATION_FORM.HELPER_MAX_BS_COUNT}
            type="number"
            slotProps={{ htmlInput: { min: 0 } }}
            fullWidth
            disabled={!canHaveBaseStations}
          />
          <TextField
            label={ORGANIZATION_FORM.LABEL_MAX_EP_COUNT}
            value={maxEpCount}
            onChange={(e) => setMaxEpCount(e.target.value.replace(/\D/g, ""))}
            helperText={ORGANIZATION_FORM.HELPER_MAX_EP_COUNT}
            type="number"
            slotProps={{ htmlInput: { min: 0 } }}
            fullWidth
          />
        </Box>
        <Box sx={{ mt: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            {ORGANIZATION_FORM.LABEL_TAGS}
          </Typography>
          <TagsEditor tags={tags} onChange={setTags} />
        </Box>
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
