/**
 * AddUserDialog Component
 *
 * Dialog for creating a new system user.
 * Optionally adds the user to selected organizations after creation.
 * Supports a multi-org picker when no explicit orgId is provided.
 */

import React, { useState } from 'react';

import type { CreateUserRequest, OrganizationUI } from '@api-types/api';
import { useCreateUser } from '@hooks';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Switch,
  TextField,
} from '@mui/material';

import { useOrganization as useOrganizationContext } from '@contexts/OrganizationContext';
import { useAddOrgUser, useOrganizations } from '@hooks/useOrganizations';
import { ORG_ROLE } from '@constants/app';
import { USER_FORM } from '@constants/messages';

interface AddUserDialogProps {
  open: boolean;
  onClose: () => void;
  /** Optional org ID - if provided, user will be auto-added to this org after creation */
  orgId?: string;
}

const AddUserDialog: React.FC<AddUserDialogProps> = ({ open, onClose, orgId }) => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [note, setNote] = useState('');
  const [isAdmin, setIsAdmin] = useState(false);
  const [isActive, setIsActive] = useState(true);
  const [isTenantManager, setIsTenantManager] = useState(false);
  const [isBaseStationManager, setIsBaseStationManager] = useState(false);
  const [isEndpointManager, setIsEndpointManager] = useState(false);
  const [selectedOrgs, setSelectedOrgs] = useState<OrganizationUI[]>([]);
  const [partialError, setPartialError] = useState('');

  const [emailError, setEmailError] = useState('');
  const [passwordError, setPasswordError] = useState('');
  const [confirmPasswordError, setConfirmPasswordError] = useState('');

  const createUser = useCreateUser();
  const addOrgUser = useAddOrgUser();
  const { organizationId } = useOrganizationContext();

  // Fetch organizations for the multi-select picker (only when dialog is open and no fixed orgId)
  const { data: orgsData } = useOrganizations(200, 0, undefined, {
    enabled: open && !orgId,
  });
  const organizations = orgsData?.organizations ?? [];

  // Pre-select current org context if available and no explicit orgId
  const handleOpen = () => {
    if (!orgId && organizationId && organizations.length > 0 && selectedOrgs.length === 0) {
      const currentOrg = organizations.find((o) => o.id === organizationId);
      if (currentOrg) {
        setSelectedOrgs([currentOrg]);
      }
    }
  };

  const resetForm = () => {
    setEmail('');
    setPassword('');
    setConfirmPassword('');
    setNote('');
    setIsAdmin(false);
    setIsActive(true);
    setIsTenantManager(false);
    setIsBaseStationManager(false);
    setIsEndpointManager(false);
    setSelectedOrgs([]);
    setPartialError('');
    setEmailError('');
    setPasswordError('');
    setConfirmPasswordError('');
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const validate = (): boolean => {
    let isValid = true;

    if (!email.trim()) {
      setEmailError(USER_FORM.ERR_EMAIL_REQUIRED);
      isValid = false;
    } else {
      setEmailError('');
    }

    if (!password) {
      setPasswordError(USER_FORM.ERR_PASSWORD_REQUIRED);
      isValid = false;
    } else {
      setPasswordError('');
    }

    if (password !== confirmPassword) {
      setConfirmPasswordError(USER_FORM.ERR_PASSWORD_MISMATCH);
      isValid = false;
    } else {
      setConfirmPasswordError('');
    }

    return isValid;
  };

  const handleSubmit = async () => {
    if (!validate()) {
      return;
    }

    const request: CreateUserRequest = {
      email: email.trim(),
      password,
      is_admin: isAdmin,
      is_active: isActive,
      is_tenant_manager: isTenantManager,
      is_base_station_manager: isBaseStationManager,
      is_endpoint_manager: isEndpointManager,
      ...(note.trim() && { note: note.trim() }),
    };

    try {
      const newUser = await createUser.mutateAsync(request);

      if (!newUser?.id) {
        handleClose();
        return;
      }

      // Determine which orgs to add the user to
      const orgsToAdd: string[] = [];
      if (orgId) {
        // Explicit single org
        orgsToAdd.push(orgId);
      } else {
        // Multi-select orgs
        orgsToAdd.push(...selectedOrgs.map((o) => o.id));
      }

      // Add user to each selected organization
      const failures: string[] = [];
      for (const targetOrgId of orgsToAdd) {
        try {
          await addOrgUser.mutateAsync({
            orgId: targetOrgId,
            data: {
              user_id: newUser.id,
              role: ORG_ROLE.MEMBER,
              is_org_admin: false,
              is_base_station_admin: isBaseStationManager,
              is_endpoint_admin: isEndpointManager,
            },
          });
        } catch {
          failures.push(targetOrgId);
        }
      }

      if (failures.length > 0) {
        setPartialError(USER_FORM.ERR_ADD_TO_ORG_PARTIAL);
      } else {
        handleClose();
      }
    } catch {
      // Error handled by mutation hook
    }
  };

  // Effect-like: auto-select current org when organizations load
  React.useEffect(() => {
    handleOpen();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [organizations.length, organizationId]);

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{USER_FORM.DIALOG_TITLE_ADD}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          {createUser.isError && (
            <Alert severity="error">
              {createUser.error instanceof Error
                ? createUser.error.message
                : USER_FORM.ERR_CREATE_FAILED}
            </Alert>
          )}

          {partialError && <Alert severity="warning">{partialError}</Alert>}

          <TextField
            label={USER_FORM.LABEL_EMAIL}
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            error={!!emailError}
            helperText={emailError}
            fullWidth
            required
            autoFocus
          />

          <TextField
            label={USER_FORM.LABEL_PASSWORD}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={!!passwordError}
            helperText={passwordError}
            fullWidth
            required
          />

          <TextField
            label={USER_FORM.LABEL_CONFIRM_PASSWORD}
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            error={!!confirmPasswordError}
            helperText={confirmPasswordError}
            fullWidth
            required
          />

          <TextField
            label={USER_FORM.LABEL_NOTE}
            value={note}
            onChange={(e) => setNote(e.target.value)}
            multiline
            rows={2}
            fullWidth
          />

          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <FormControlLabel
              control={
                <Switch checked={isActive} onChange={(e) => setIsActive(e.target.checked)} />
              }
              label={USER_FORM.LABEL_IS_ACTIVE}
            />
            <FormControlLabel
              control={<Switch checked={isAdmin} onChange={(e) => setIsAdmin(e.target.checked)} />}
              label={USER_FORM.LABEL_IS_ADMIN}
            />
            <FormControlLabel
              control={
                <Switch
                  checked={isTenantManager}
                  onChange={(e) => setIsTenantManager(e.target.checked)}
                />
              }
              label={USER_FORM.LABEL_IS_TENANT_MGR}
            />
            <FormControlLabel
              control={
                <Switch
                  checked={isBaseStationManager}
                  onChange={(e) => setIsBaseStationManager(e.target.checked)}
                />
              }
              label={USER_FORM.LABEL_IS_BS_MGR}
            />
            <FormControlLabel
              control={
                <Switch
                  checked={isEndpointManager}
                  onChange={(e) => setIsEndpointManager(e.target.checked)}
                />
              }
              label={USER_FORM.LABEL_IS_EP_MGR}
            />
          </Box>

          {/* Multi-org picker: shown when no explicit orgId is provided */}
          {!orgId && (
            <Autocomplete
              multiple
              options={organizations}
              getOptionLabel={(option) => option.name}
              value={selectedOrgs}
              onChange={(_, newValue) => setSelectedOrgs(newValue)}
              isOptionEqualToValue={(option, value) => option.id === value.id}
              renderTags={(value, getTagProps) =>
                value.map((option, index) => {
                  const { key, ...chipProps } = getTagProps({ index });
                  return <Chip key={key} label={option.name} size="small" {...chipProps} />;
                })
              }
              renderInput={(params) => (
                <TextField
                  {...params}
                  label={USER_FORM.LABEL_ORGANIZATIONS}
                  helperText={USER_FORM.HELPER_ORGANIZATIONS}
                />
              )}
            />
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose} disabled={createUser.isPending}>
          {USER_FORM.ACTION_CANCEL}
        </Button>
        <Button variant="contained" onClick={handleSubmit} disabled={createUser.isPending}>
          {USER_FORM.ACTION_SUBMIT}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default AddUserDialog;
