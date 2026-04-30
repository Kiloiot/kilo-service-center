/**
 * Organization User Dialog Component
 *
 * Unified dialog for adding and editing organization members.
 * - Add mode: Search and select user, configure role and permissions
 * - Edit mode: Shows user email (read-only), allows role/permission changes
 */

import React, { useEffect, useMemo, useState } from 'react';

import type { OrganizationUserUI } from '@api-types/api';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  MenuItem,
  Switch,
  TextField,
} from '@mui/material';

import { useCapabilities } from '@hooks/useCapabilities';
import { useAddOrgUser, useUpdateOrgUser } from '@hooks/useOrganizations';
import { useUsersForLookup } from '@hooks/useUsers';
import { ORG_ROLE, ORG_USER_PLACEHOLDER_EMAIL } from '@constants/app';
import { ORG_USER_FORM, ORG_USERS_PAGE } from '@constants/messages';

export interface OrganizationUserDialogProps {
  open: boolean;
  onClose: () => void;
  orgId: string;
  mode: 'add' | 'edit';
  initialUser?: OrganizationUserUI;
}

interface UserOption {
  id: string;
  email: string;
}

const OrganizationUserDialog: React.FC<OrganizationUserDialogProps> = ({
  open,
  onClose,
  orgId,
  mode,
  initialUser,
}) => {
  const isEditMode = mode === 'edit';
  const { isServerAdmin } = useCapabilities();

  // In add mode: server admin picks existing user via autocomplete,
  // org admin enters an email address directly
  const showUserPicker = !isEditMode && isServerAdmin;

  // Form state
  const [email, setEmail] = useState('');
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [selectedUserOption, setSelectedUserOption] = useState<UserOption | null>(null);
  const [role, setRole] = useState<string>(ORG_ROLE.MEMBER);
  const [isOrgAdmin, setIsOrgAdmin] = useState(false);
  const [isBsAdmin, setIsBsAdmin] = useState(false);
  const [isEpAdmin, setIsEpAdmin] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  // Mutations
  const { mutate: addUser, isPending: isAddPending } = useAddOrgUser();
  const { mutate: updateUser, isPending: isUpdatePending } = useUpdateOrgUser();
  const isPending = isAddPending || isUpdatePending;

  // Fetch users for autocomplete (only in add mode for server admins)
  const { data: usersData, isLoading: isLoadingUsers } = useUsersForLookup({
    enabled: open && showUserPicker,
  });

  // Initialize form when dialog opens or initialUser changes
  useEffect(() => {
    if (open && isEditMode && initialUser) {
      setEmail(initialUser.email);
      setSelectedUserId(initialUser.userId);
      setSelectedUserOption({ id: initialUser.userId, email: initialUser.email });
      setRole(initialUser.role);
      setIsOrgAdmin(initialUser.isOrgAdmin);
      setIsBsAdmin(initialUser.isBaseStationAdmin);
      setIsEpAdmin(initialUser.isEndpointAdmin);
      setSubmitError(null);
    } else if (open && !isEditMode) {
      // Reset form for add mode
      handleReset();
    }
  }, [open, isEditMode, initialUser]);

  // Filter users by email input (client-side) - only used in add mode
  const filteredUsers = useMemo(() => {
    if (!usersData?.users || !email) return [];
    const searchLower = email.toLowerCase();
    return usersData.users.filter((u) => u.email.toLowerCase().includes(searchLower));
  }, [usersData?.users, email]);

  const handleReset = () => {
    setEmail('');
    setSelectedUserId(null);
    setSelectedUserOption(null);
    setRole(ORG_ROLE.MEMBER);
    setIsOrgAdmin(false);
    setIsBsAdmin(false);
    setIsEpAdmin(false);
    setSubmitError(null);
  };

  const handleClose = () => {
    handleReset();
    onClose();
  };

  // When isOrgAdmin is true, clear and disable BS/EP admin flags
  const handleOrgAdminChange = (checked: boolean) => {
    setIsOrgAdmin(checked);
    if (checked) {
      setIsBsAdmin(false);
      setIsEpAdmin(false);
    }
  };

  const handleSubmit = () => {
    setSubmitError(null);

    // In add mode: server admin needs selectedUserId, org admin needs email
    if (!isEditMode && showUserPicker && !selectedUserId) {
      setSubmitError(ORG_USER_FORM.ERR_USER_REQUIRED);
      return;
    }
    if (!isEditMode && !showUserPicker && !email.trim()) {
      setSubmitError(ORG_USER_FORM.ERR_USER_REQUIRED);
      return;
    }
    if (isEditMode && !selectedUserId) {
      setSubmitError(ORG_USER_FORM.ERR_USER_REQUIRED);
      return;
    }

    if (isEditMode) {
      // Update existing member
      updateUser(
        {
          orgId,
          userId: selectedUserId!,
          data: {
            role,
            is_org_admin: isOrgAdmin,
            is_base_station_admin: isOrgAdmin ? false : isBsAdmin,
            is_endpoint_admin: isOrgAdmin ? false : isEpAdmin,
          },
        },
        {
          onSuccess: () => {
            handleClose();
          },
          onError: () => {
            setSubmitError(ORG_USER_FORM.ERR_UPDATE_FAILED);
          },
        }
      );
    } else {
      // Add new member - server admin uses userId, org admin uses email
      addUser(
        {
          orgId,
          data: {
            user_id: showUserPicker ? selectedUserId! : undefined,
            email: showUserPicker ? undefined : email.trim(),
            role,
            is_org_admin: isOrgAdmin,
            is_base_station_admin: isOrgAdmin ? false : isBsAdmin,
            is_endpoint_admin: isOrgAdmin ? false : isEpAdmin,
          },
        },
        {
          onSuccess: () => {
            handleClose();
          },
          onError: () => {
            setSubmitError(ORG_USER_FORM.ERR_ADD_FAILED);
          },
        }
      );
    }
  };

  const dialogTitle = isEditMode ? ORG_USER_FORM.DIALOG_TITLE_EDIT : ORG_USER_FORM.DIALOG_TITLE_ADD;

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>{dialogTitle}</DialogTitle>
      <DialogContent>
        {submitError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {submitError}
          </Alert>
        )}
        {/* Server admin: Autocomplete user picker; Org admin: simple email field */}
        {showUserPicker || isEditMode ? (
          <Autocomplete
            freeSolo={!isEditMode}
            disabled={isEditMode}
            options={isEditMode && selectedUserOption ? [selectedUserOption] : filteredUsers}
            getOptionLabel={(option) => (typeof option === 'string' ? option : option.email)}
            value={isEditMode ? selectedUserOption : undefined}
            inputValue={email}
            onInputChange={(_, newValue) => {
              if (!isEditMode) {
                setEmail(newValue);
              }
            }}
            onChange={(_, newValue) => {
              if (!isEditMode && newValue && typeof newValue !== 'string') {
                setSelectedUserId(newValue.id);
                setEmail(newValue.email);
                setSelectedUserOption(newValue);
              } else if (!isEditMode) {
                setSelectedUserId(null);
                setSelectedUserOption(null);
              }
            }}
            loading={isLoadingUsers}
            renderInput={(params) => (
              <TextField
                {...params}
                label={ORG_USER_FORM.LABEL_USER_ID}
                placeholder={isEditMode ? undefined : ORG_USER_PLACEHOLDER_EMAIL}
                required
                margin="normal"
                type="email"
                slotProps={{
                  input: {
                    ...params.InputProps,
                    endAdornment: (
                      <>
                        {isLoadingUsers ? <CircularProgress color="inherit" size={20} /> : null}
                        {params.InputProps.endAdornment}
                      </>
                    ),
                  },
                }}
              />
            )}
          />
        ) : (
          <TextField
            label={ORG_USER_FORM.LABEL_USER_ID}
            placeholder={ORG_USER_PLACEHOLDER_EMAIL}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            fullWidth
            margin="normal"
            type="email"
          />
        )}
        <TextField
          select
          label={ORG_USERS_PAGE.COL_ROLE}
          value={role}
          onChange={(e) => setRole(e.target.value)}
          fullWidth
          margin="normal"
        >
          <MenuItem value={ORG_ROLE.MEMBER}>{ORG_USERS_PAGE.ROLE_MEMBER}</MenuItem>
          <MenuItem value={ORG_ROLE.ADMIN}>{ORG_USERS_PAGE.ROLE_ADMIN}</MenuItem>
        </TextField>
        <Box sx={{ mt: 2 }}>
          <FormControlLabel
            control={
              <Switch
                checked={isOrgAdmin}
                onChange={(e) => handleOrgAdminChange(e.target.checked)}
              />
            }
            label={ORG_USER_FORM.LABEL_IS_ORG_ADMIN}
          />
          <Box sx={{ color: 'text.secondary', fontSize: '0.75rem', ml: 4, mb: 1 }}>
            {ORG_USER_FORM.HELPER_ORG_ADMIN}
          </Box>
        </Box>
        {/* Hide BS/EP admin toggles when isOrgAdmin is true */}
        {!isOrgAdmin && (
          <>
            <FormControlLabel
              control={
                <Switch checked={isBsAdmin} onChange={(e) => setIsBsAdmin(e.target.checked)} />
              }
              label={ORG_USER_FORM.LABEL_IS_BS_ADMIN}
            />
            <Box sx={{ color: 'text.secondary', fontSize: '0.75rem', ml: 4, mb: 1 }}>
              {ORG_USER_FORM.HELPER_BS_ADMIN}
            </Box>
            <FormControlLabel
              control={
                <Switch checked={isEpAdmin} onChange={(e) => setIsEpAdmin(e.target.checked)} />
              }
              label={ORG_USER_FORM.LABEL_IS_EP_ADMIN}
            />
            <Box sx={{ color: 'text.secondary', fontSize: '0.75rem', ml: 4 }}>
              {ORG_USER_FORM.HELPER_EP_ADMIN}
            </Box>
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>{ORG_USER_FORM.ACTION_CANCEL}</Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={
            isPending ||
            (isEditMode && !selectedUserId) ||
            (!isEditMode && showUserPicker && !selectedUserId) ||
            (!isEditMode && !showUserPicker && !email.trim())
          }
        >
          {ORG_USER_FORM.ACTION_SUBMIT}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default OrganizationUserDialog;
