/**
 * UserDetail Page
 *
 * View and edit user details, including organization memberships management.
 */

import React, { useEffect, useState } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router-dom';

import type { OrganizationUI, UpdateUserRequest } from '@api-types/api';
import { useDeleteUser, useUpdateUser, useUser, useUserOrganizations } from '@hooks';
import {
  Alert,
  Autocomplete,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControl,
  FormControlLabel,
  Grid,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';

import { useSession } from '@contexts/SessionContext';
import { useAddOrgUser, useOrganizations, useRemoveOrgUser } from '@hooks/useOrganizations';
import { formatRelativeDuration } from '@utils/formatters';
import { ORG_MEMBER_STATUS, ORG_ROLE, ROUTES, UI_TIMING } from '@constants/app';
import {
  MSG_USER_DELETED,
  MSG_USER_UPDATED,
  ORG_USERS_PAGE,
  USER_FORM,
  USERS_PAGE,
} from '@constants/messages';
import { AddIcon, DeleteIcon, SaveIcon } from '@theme/icons';

const UserDetail: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { isAdmin, isHydrated } = useSession();

  // Hooks
  const {
    data: user,
    isLoading,
    isError,
    error,
  } = useUser(id || '', {
    enabled: isHydrated && isAdmin && Boolean(id),
  });
  const updateUser = useUpdateUser();
  const deleteUser = useDeleteUser();
  const canNavigateToPassword = Boolean(id);

  // Organization memberships
  const { data: userOrgsData, isLoading: isOrgsLoading } = useUserOrganizations(id || '', {
    enabled: isHydrated && isAdmin && Boolean(id),
  });
  const memberships = userOrgsData?.memberships ?? [];

  // Available organizations for the "Add to Organization" picker
  const { data: allOrgsData } = useOrganizations(200, 0, undefined, {
    enabled: isHydrated && isAdmin && Boolean(id),
  });
  const allOrganizations = allOrgsData?.organizations ?? [];

  // Filter out orgs the user is already a member of
  const availableOrgs = allOrganizations.filter(
    (org) => !memberships.some((m) => m.orgId === org.id)
  );

  // Add/remove org mutation hooks
  const addOrgUser = useAddOrgUser();
  const removeOrgUser = useRemoveOrgUser();

  // Form state (use isUserAdmin to avoid conflict with session isAdmin)
  const [email, setEmail] = useState('');
  const [note, setNote] = useState('');
  const [isUserAdmin, setIsUserAdmin] = useState(false);
  const [isActive, setIsActive] = useState(true);
  const [isTenantManager, setIsTenantManager] = useState(false);
  const [isBaseStationManager, setIsBaseStationManager] = useState(false);
  const [isEndpointManager, setIsEndpointManager] = useState(false);

  // Dialog state
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  // Success messages
  const [successMessage, setSuccessMessage] = useState('');

  // Add-to-org form state
  const [selectedOrg, setSelectedOrg] = useState<OrganizationUI | null>(null);
  const [selectedRole, setSelectedRole] = useState(ORG_ROLE.MEMBER);
  const [membershipError, setMembershipError] = useState('');
  const [membershipSuccess, setMembershipSuccess] = useState('');

  // Populate form when user data loads
  useEffect(() => {
    if (user) {
      setEmail(user.email);
      setNote(user.note || '');
      setIsUserAdmin(user.isAdmin);
      setIsActive(user.isActive);
      setIsTenantManager(user.isTenantManager);
      setIsBaseStationManager(user.isBaseStationManager);
      setIsEndpointManager(user.isEndpointManager);
    }
  }, [user]);

  const handleSave = async () => {
    if (!id || !user) return;

    const updates: UpdateUserRequest = {};

    if (email !== user.email) updates.email = email;
    if (note !== (user.note || '')) updates.note = note || undefined;
    if (isUserAdmin !== user.isAdmin) updates.is_admin = isUserAdmin;
    if (isActive !== user.isActive) updates.is_active = isActive;
    if (isTenantManager !== user.isTenantManager) updates.is_tenant_manager = isTenantManager;
    if (isBaseStationManager !== user.isBaseStationManager)
      updates.is_base_station_manager = isBaseStationManager;
    if (isEndpointManager !== user.isEndpointManager)
      updates.is_endpoint_manager = isEndpointManager;

    // Only call API if there are changes
    if (Object.keys(updates).length > 0) {
      try {
        await updateUser.mutateAsync({ id, data: updates });
        setSuccessMessage(MSG_USER_UPDATED);
        setTimeout(() => setSuccessMessage(''), UI_TIMING.SUCCESS_MESSAGE_DISMISS_MS);
      } catch {
        // Error handled by mutation hook
      }
    }
  };

  const handleDelete = async () => {
    if (!id) return;

    try {
      await deleteUser.mutateAsync(id);
      setSuccessMessage(MSG_USER_DELETED);
      navigate(ROUTES.USERS);
    } catch {
      // Error handled by mutation hook
    }
    setDeleteDialogOpen(false);
  };

  const handleBackToList = () => {
    navigate(ROUTES.USERS);
  };

  const handleAddMembership = async () => {
    if (!id || !selectedOrg) return;

    setMembershipError('');
    setMembershipSuccess('');

    try {
      await addOrgUser.mutateAsync({
        orgId: selectedOrg.id,
        data: {
          user_id: id,
          role: selectedRole,
          is_org_admin: false,
          is_base_station_admin: false,
          is_endpoint_admin: false,
        },
      });
      setSelectedOrg(null);
      setSelectedRole(ORG_ROLE.MEMBER);
      setMembershipSuccess(USER_FORM.MSG_MEMBERSHIP_ADDED);
      setTimeout(() => setMembershipSuccess(''), UI_TIMING.SUCCESS_MESSAGE_DISMISS_MS);
    } catch {
      setMembershipError(USER_FORM.ERR_MEMBERSHIP_ADD_FAILED);
    }
  };

  const handleRemoveMembership = async (orgId: string) => {
    if (!id) return;

    setMembershipError('');
    setMembershipSuccess('');

    try {
      await removeOrgUser.mutateAsync({ orgId, userId: id });
      setMembershipSuccess(USER_FORM.MSG_MEMBERSHIP_REMOVED);
      setTimeout(() => setMembershipSuccess(''), UI_TIMING.SUCCESS_MESSAGE_DISMISS_MS);
    } catch {
      setMembershipError(USER_FORM.ERR_MEMBERSHIP_REMOVE_FAILED);
    }
  };

  if (!isHydrated) {
    return null;
  }

  if (!isAdmin) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  if (isLoading) {
    return (
      <Box
        sx={{
          pt: 4,
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          minHeight: '400px',
        }}
      >
        <CircularProgress />
      </Box>
    );
  }

  if (isError || !user) {
    return (
      <Box sx={{ pt: 4, p: 3 }}>
        <Alert severity="error">
          {error instanceof Error ? error.message : USERS_PAGE.ERR_NOT_FOUND}
        </Alert>
        <Button sx={{ mt: 2 }} onClick={handleBackToList}>
          {USERS_PAGE.BACK_TO_LIST}
        </Button>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3, pt: 4 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" component="h1">
          {USERS_PAGE.DETAILS_TITLE}
        </Typography>
        <Button variant="outlined" onClick={handleBackToList}>
          {USERS_PAGE.BACK_TO_LIST}
        </Button>
      </Box>

      {/* Success Message */}
      {successMessage && (
        <Alert severity="success" sx={{ mb: 3 }}>
          {successMessage}
        </Alert>
      )}

      {/* Update Error */}
      {updateUser.isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {updateUser.error instanceof Error
            ? updateUser.error.message
            : USER_FORM.ERR_UPDATE_FAILED}
        </Alert>
      )}

      <Grid container spacing={3}>
        {/* User Details Card */}
        <Grid size={{ xs: 12, md: 8 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                {USER_FORM.DIALOG_TITLE_EDIT}
              </Typography>

              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 2 }}>
                <TextField
                  label={USER_FORM.LABEL_EMAIL}
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  fullWidth
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
                    control={
                      <Switch
                        checked={isUserAdmin}
                        onChange={(e) => setIsUserAdmin(e.target.checked)}
                      />
                    }
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

                <Box display="flex" gap={2} mt={2}>
                  <Button
                    variant="contained"
                    startIcon={<SaveIcon />}
                    onClick={handleSave}
                    disabled={updateUser.isPending}
                  >
                    {USER_FORM.ACTION_SUBMIT}
                  </Button>
                  <Button
                    variant="outlined"
                    color="error"
                    startIcon={<DeleteIcon />}
                    onClick={() => setDeleteDialogOpen(true)}
                    disabled={deleteUser.isPending}
                  >
                    {USER_FORM.ACTION_DELETE}
                  </Button>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* Info Card */}
        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                {USER_FORM.INFO_TITLE}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {USER_FORM.INFO_CREATED}: {formatRelativeDuration(user.createdAt)}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {USER_FORM.INFO_UPDATED}: {formatRelativeDuration(user.updatedAt)}
              </Typography>
              <Button
                variant="outlined"
                onClick={() => {
                  if (!id) return;
                  navigate(ROUTES.USER_PASSWORD.replace(':id', id));
                }}
                sx={{ mt: 2 }}
                fullWidth
                disabled={!canNavigateToPassword}
              >
                {USER_FORM.ACTION_CHANGE_PASSWORD}
              </Button>
            </CardContent>
          </Card>
        </Grid>

        {/* Organization Memberships Card */}
        <Grid size={{ xs: 12 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                {USER_FORM.SECTION_ORG_MEMBERSHIPS}
              </Typography>

              {membershipSuccess && (
                <Alert severity="success" sx={{ mb: 2 }}>
                  {membershipSuccess}
                </Alert>
              )}

              {membershipError && (
                <Alert severity="error" sx={{ mb: 2 }}>
                  {membershipError}
                </Alert>
              )}

              {/* Memberships table */}
              {isOrgsLoading ? (
                <Box display="flex" justifyContent="center" py={2}>
                  <CircularProgress size={24} />
                </Box>
              ) : memberships.length === 0 ? (
                <Typography color="text.secondary" sx={{ py: 2 }}>
                  {USER_FORM.EMPTY_NO_MEMBERSHIPS}
                </Typography>
              ) : (
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>{USER_FORM.COL_ORG_NAME}</TableCell>
                        <TableCell>{ORG_USERS_PAGE.COL_ROLE}</TableCell>
                        <TableCell>{ORG_USERS_PAGE.COL_STATUS}</TableCell>
                        <TableCell>{ORG_USERS_PAGE.COL_ACTIONS}</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {memberships.map((m) => (
                        <TableRow key={m.orgId}>
                          <TableCell>
                            <Typography variant="body2">{m.orgName}</Typography>
                          </TableCell>
                          <TableCell>
                            <Chip label={m.role} size="small" />
                          </TableCell>
                          <TableCell>
                            <Chip
                              label={m.status}
                              size="small"
                              color={m.status === ORG_MEMBER_STATUS.ACTIVE ? 'success' : 'default'}
                            />
                          </TableCell>
                          <TableCell>
                            <Tooltip title={ORG_USERS_PAGE.TOOLTIP_REMOVE}>
                              <IconButton
                                size="small"
                                color="error"
                                onClick={() => handleRemoveMembership(m.orgId)}
                                disabled={removeOrgUser.isPending}
                              >
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}

              {/* Add to Organization section */}
              <Box sx={{ mt: 3, pt: 2, borderTop: 1, borderColor: 'divider' }}>
                <Typography variant="subtitle2" gutterBottom>
                  {USER_FORM.LABEL_ADD_TO_ORG}
                </Typography>
                <Box display="flex" gap={2} alignItems="flex-start">
                  <Autocomplete
                    options={availableOrgs}
                    getOptionLabel={(option) => option.name}
                    value={selectedOrg}
                    onChange={(_, newValue) => setSelectedOrg(newValue)}
                    isOptionEqualToValue={(option, value) => option.id === value.id}
                    sx={{ minWidth: 250, flex: 1 }}
                    renderInput={(params) => (
                      <TextField {...params} label={USER_FORM.LABEL_SELECT_ORG} size="small" />
                    )}
                  />
                  <FormControl size="small" sx={{ minWidth: 120 }}>
                    <InputLabel>{USER_FORM.LABEL_SELECT_ROLE}</InputLabel>
                    <Select
                      value={selectedRole}
                      onChange={(e) => setSelectedRole(e.target.value)}
                      label={USER_FORM.LABEL_SELECT_ROLE}
                    >
                      <MenuItem value={ORG_ROLE.MEMBER}>{ORG_USERS_PAGE.ROLE_MEMBER}</MenuItem>
                      <MenuItem value={ORG_ROLE.ADMIN}>{ORG_USERS_PAGE.ROLE_ADMIN}</MenuItem>
                      <MenuItem value={ORG_ROLE.OWNER}>{ORG_USERS_PAGE.ROLE_OWNER}</MenuItem>
                    </Select>
                  </FormControl>
                  <Button
                    variant="contained"
                    size="small"
                    startIcon={<AddIcon />}
                    onClick={handleAddMembership}
                    disabled={!selectedOrg || addOrgUser.isPending}
                    sx={{ mt: 0.5 }}
                  >
                    {USER_FORM.ACTION_ADD_MEMBERSHIP}
                  </Button>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>{USER_FORM.ACTION_DELETE}</DialogTitle>
        <DialogContent>
          <DialogContentText>{USER_FORM.CONFIRM_DELETE}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>{USER_FORM.ACTION_CANCEL}</Button>
          <Button color="error" onClick={handleDelete} disabled={deleteUser.isPending}>
            {USER_FORM.ACTION_DELETE}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default UserDetail;
