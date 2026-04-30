/**
 * Organization Form Component
 *
 * Editable organization configuration form with delete action.
 */

import React, { useEffect, useState } from 'react';

import type { OrganizationUI, UpdateOrganizationRequest } from '@api-types/api';
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControlLabel,
  Snackbar,
  Switch,
  TextField,
  Typography,
} from '@mui/material';

import { useDeleteOrganization, useUpdateOrganization } from '@hooks/useOrganizations';
import { ERR_DELETE_ORGANIZATION, ORGANIZATION_FORM } from '@constants/messages';

import TagsEditor from './TagsEditor';

interface OrganizationFormProps {
  organization: OrganizationUI;
  onDeleted?: () => void;
}

const OrganizationForm: React.FC<OrganizationFormProps> = ({ organization, onDeleted }) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [canHaveBaseStations, setCanHaveBaseStations] = useState(true);
  const [maxBsCount, setMaxBsCount] = useState('');
  const [maxEpCount, setMaxEpCount] = useState('');
  const [tags, setTags] = useState<Record<string, string>>({});
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [successSnackbarOpen, setSuccessSnackbarOpen] = useState(false);

  const updateOrganization = useUpdateOrganization();
  const deleteOrganization = useDeleteOrganization();

  useEffect(() => {
    setName(organization.name);
    setDescription(organization.description ?? '');
    setCanHaveBaseStations(organization.canHaveBaseStations);
    setMaxBsCount(
      organization.maxBaseStationCount !== undefined ? String(organization.maxBaseStationCount) : ''
    );
    setMaxEpCount(
      organization.maxEndpointCount !== undefined ? String(organization.maxEndpointCount) : ''
    );
    setTags(organization.tags ?? {});
  }, [organization]);

  const buildUpdateRequest = (): UpdateOrganizationRequest => {
    const parsedMaxBs = maxBsCount ? parseInt(maxBsCount, 10) : undefined;
    const parsedMaxEp = maxEpCount ? parseInt(maxEpCount, 10) : undefined;

    return {
      name: name.trim(),
      description: description.trim() || undefined,
      can_have_base_stations: canHaveBaseStations,
      max_base_station_count: parsedMaxBs === 0 ? undefined : parsedMaxBs,
      max_endpoint_count: parsedMaxEp === 0 ? undefined : parsedMaxEp,
      tags: Object.keys(tags).length > 0 ? tags : undefined,
    };
  };

  const handleSave = async () => {
    const payload = buildUpdateRequest();
    try {
      await updateOrganization.mutateAsync({ id: organization.id, data: payload });
      setSuccessSnackbarOpen(true);
    } catch {
      // Error surfaced via updateOrganization.isError
    }
  };

  const handleDelete = async () => {
    try {
      await deleteOrganization.mutateAsync(organization.id);
      setConfirmDeleteOpen(false);
      if (onDeleted) onDeleted();
    } catch {
      // Error surfaced via deleteOrganization.isError
    }
  };

  return (
    <Box>
      {updateOrganization.isError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {updateOrganization.error instanceof Error
            ? updateOrganization.error.message
            : ORGANIZATION_FORM.ERR_UPDATE_FAILED}
        </Alert>
      )}

      {deleteOrganization.isError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {deleteOrganization.error instanceof Error
            ? deleteOrganization.error.message
            : ERR_DELETE_ORGANIZATION}
        </Alert>
      )}

      <TextField
        label={ORGANIZATION_FORM.LABEL_NAME}
        value={name}
        onChange={(e) => setName(e.target.value)}
        helperText={ORGANIZATION_FORM.HELPER_NAME}
        fullWidth
        required
        margin="normal"
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
      <Box sx={{ display: 'flex', gap: 2, mt: 2 }}>
        <TextField
          label={ORGANIZATION_FORM.LABEL_MAX_BS_COUNT}
          value={maxBsCount}
          onChange={(e) => setMaxBsCount(e.target.value.replace(/\D/g, ''))}
          helperText={ORGANIZATION_FORM.HELPER_MAX_BS_COUNT}
          type="number"
          slotProps={{ htmlInput: { min: 0 } }}
          fullWidth
          disabled={!canHaveBaseStations}
        />
        <TextField
          label={ORGANIZATION_FORM.LABEL_MAX_EP_COUNT}
          value={maxEpCount}
          onChange={(e) => setMaxEpCount(e.target.value.replace(/\D/g, ''))}
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
      <Box sx={{ display: 'flex', gap: 2, mt: 3 }}>
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

      <Dialog open={confirmDeleteOpen} onClose={() => setConfirmDeleteOpen(false)}>
        <DialogTitle>{ORGANIZATION_FORM.ACTION_DELETE}</DialogTitle>
        <DialogContent>
          <DialogContentText>{ORGANIZATION_FORM.CONFIRM_DELETE}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmDeleteOpen(false)}>
            {ORGANIZATION_FORM.ACTION_CANCEL}
          </Button>
          <Button color="error" onClick={handleDelete} disabled={deleteOrganization.isPending}>
            {ORGANIZATION_FORM.ACTION_DELETE}
          </Button>
        </DialogActions>
      </Dialog>

      <Snackbar
        open={successSnackbarOpen}
        autoHideDuration={3000}
        onClose={() => setSuccessSnackbarOpen(false)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity="success" onClose={() => setSuccessSnackbarOpen(false)}>
          {ORGANIZATION_FORM.SUCCESS_UPDATE}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default OrganizationForm;
