/**
 * UserDetail Page
 *
 * View and edit user details, including organization memberships management.
 */

import React, { useEffect, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";

import type {
  OrganizationUI,
  SystemUserUI,
  UpdateUserRequest,
} from "@api-types/api";
import {
  useDeleteUser,
  useUpdateUser,
  useUser,
  useUserOrganizations,
} from "@hooks";
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
} from "@mui/material";

import { useSession } from "@contexts/SessionContext";
import {
  useAddOrgUser,
  useOrganizations,
  useRemoveOrgUser,
} from "@hooks/useOrganizations";
import { formatRelativeDuration } from "@utils/formatters";
import { ORG_MEMBER_STATUS, ORG_ROLE, ROUTES, UI_TIMING } from "@constants/app";
import {
  MSG_USER_DELETED,
  MSG_USER_UPDATED,
  ORG_USERS_PAGE,
  USER_FORM,
  USERS_PAGE,
} from "@constants/messages";
import { AddIcon, DeleteIcon, SaveIcon } from "@theme/icons";

interface UserOrgMembership {
  orgId: string;
  orgName: string;
  role: string;
  status: string;
}

interface UserProfileForm {
  email: string;
  note: string;
  isUserAdmin: boolean;
  isActive: boolean;
  isTenantManager: boolean;
  isBaseStationManager: boolean;
  isEndpointManager: boolean;
}

/**
 * Builds an UpdateUserRequest containing only the fields that differ from the
 * loaded user record. Returns an empty object when nothing changed.
 */
function buildUpdateUserPayload(
  form: UserProfileForm,
  original: SystemUserUI,
): UpdateUserRequest {
  const updates: UpdateUserRequest = {};
  if (form.email !== original.email) updates.email = form.email;
  if (form.note !== (original.note || "")) {
    updates.note = form.note || undefined;
  }
  if (form.isUserAdmin !== original.isAdmin)
    updates.is_admin = form.isUserAdmin;
  if (form.isActive !== original.isActive) updates.is_active = form.isActive;
  if (form.isTenantManager !== original.isTenantManager) {
    updates.is_tenant_manager = form.isTenantManager;
  }
  if (form.isBaseStationManager !== original.isBaseStationManager) {
    updates.is_base_station_manager = form.isBaseStationManager;
  }
  if (form.isEndpointManager !== original.isEndpointManager) {
    updates.is_endpoint_manager = form.isEndpointManager;
  }
  return updates;
}

interface UserDetailHeaderProps {
  onBack: () => void;
}

const UserDetailHeader: React.FC<UserDetailHeaderProps> = ({ onBack }) => (
  <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
    <Typography variant="h4" component="h1">
      {USERS_PAGE.DETAILS_TITLE}
    </Typography>
    <Button variant="outlined" onClick={onBack}>
      {USERS_PAGE.BACK_TO_LIST}
    </Button>
  </Box>
);

interface UserProfileCardProps {
  form: UserProfileForm;
  isSaving: boolean;
  isDeleting: boolean;
  onChange: <K extends keyof UserProfileForm>(
    key: K,
    value: UserProfileForm[K],
  ) => void;
  onSave: () => void;
  onDeleteClick: () => void;
}

const UserProfileCard: React.FC<UserProfileCardProps> = ({
  form,
  isSaving,
  isDeleting,
  onChange,
  onSave,
  onDeleteClick,
}) => (
  <Card>
    <CardContent>
      <Typography variant="h6" gutterBottom>
        {USER_FORM.DIALOG_TITLE_EDIT}
      </Typography>

      <Box sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 2 }}>
        <TextField
          label={USER_FORM.LABEL_EMAIL}
          type="email"
          value={form.email}
          onChange={(e) => onChange("email", e.target.value)}
          fullWidth
        />

        <TextField
          label={USER_FORM.LABEL_NOTE}
          value={form.note}
          onChange={(e) => onChange("note", e.target.value)}
          multiline
          rows={2}
          fullWidth
        />

        <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
          <FormControlLabel
            control={
              <Switch
                checked={form.isActive}
                onChange={(e) => onChange("isActive", e.target.checked)}
              />
            }
            label={USER_FORM.LABEL_IS_ACTIVE}
          />
          <FormControlLabel
            control={
              <Switch
                checked={form.isUserAdmin}
                onChange={(e) => onChange("isUserAdmin", e.target.checked)}
              />
            }
            label={USER_FORM.LABEL_IS_ADMIN}
          />
          <FormControlLabel
            control={
              <Switch
                checked={form.isTenantManager}
                onChange={(e) => onChange("isTenantManager", e.target.checked)}
              />
            }
            label={USER_FORM.LABEL_IS_TENANT_MGR}
          />
          <FormControlLabel
            control={
              <Switch
                checked={form.isBaseStationManager}
                onChange={(e) =>
                  onChange("isBaseStationManager", e.target.checked)
                }
              />
            }
            label={USER_FORM.LABEL_IS_BS_MGR}
          />
          <FormControlLabel
            control={
              <Switch
                checked={form.isEndpointManager}
                onChange={(e) =>
                  onChange("isEndpointManager", e.target.checked)
                }
              />
            }
            label={USER_FORM.LABEL_IS_EP_MGR}
          />
        </Box>

        <Box display="flex" gap={2} mt={2}>
          <Button
            variant="contained"
            startIcon={<SaveIcon />}
            onClick={onSave}
            disabled={isSaving}
          >
            {USER_FORM.ACTION_SUBMIT}
          </Button>
          <Button
            variant="outlined"
            color="error"
            startIcon={<DeleteIcon />}
            onClick={onDeleteClick}
            disabled={isDeleting}
          >
            {USER_FORM.ACTION_DELETE}
          </Button>
        </Box>
      </Box>
    </CardContent>
  </Card>
);

interface UserMembershipsCardProps {
  memberships: UserOrgMembership[];
  availableOrgs: OrganizationUI[];
  isOrgsLoading: boolean;
  isRemovePending: boolean;
  isAddPending: boolean;
  selectedOrg: OrganizationUI | null;
  selectedRole: string;
  membershipSuccess: string;
  membershipError: string;
  onSelectedOrgChange: (org: OrganizationUI | null) => void;
  onSelectedRoleChange: (role: string) => void;
  onAddMembership: () => void;
  onRemoveMembership: (orgId: string) => void;
}

const UserMembershipsCard: React.FC<UserMembershipsCardProps> = ({
  memberships,
  availableOrgs,
  isOrgsLoading,
  isRemovePending,
  isAddPending,
  selectedOrg,
  selectedRole,
  membershipSuccess,
  membershipError,
  onSelectedOrgChange,
  onSelectedRoleChange,
  onAddMembership,
  onRemoveMembership,
}) => (
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
                      color={
                        m.status === ORG_MEMBER_STATUS.ACTIVE
                          ? "success"
                          : "default"
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <Tooltip title={ORG_USERS_PAGE.TOOLTIP_REMOVE}>
                      <IconButton
                        size="small"
                        color="error"
                        onClick={() => onRemoveMembership(m.orgId)}
                        disabled={isRemovePending}
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

      <Box sx={{ mt: 3, pt: 2, borderTop: 1, borderColor: "divider" }}>
        <Typography variant="subtitle2" gutterBottom>
          {USER_FORM.LABEL_ADD_TO_ORG}
        </Typography>
        <Box display="flex" gap={2} alignItems="flex-start">
          <Autocomplete
            options={availableOrgs}
            getOptionLabel={(option) => option.name}
            value={selectedOrg}
            onChange={(_, newValue) => onSelectedOrgChange(newValue)}
            isOptionEqualToValue={(option, value) => option.id === value.id}
            sx={{ minWidth: 250, flex: 1 }}
            renderInput={(params) => (
              <TextField
                {...params}
                label={USER_FORM.LABEL_SELECT_ORG}
                size="small"
              />
            )}
          />
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>{USER_FORM.LABEL_SELECT_ROLE}</InputLabel>
            <Select
              value={selectedRole}
              onChange={(e) => onSelectedRoleChange(e.target.value)}
              label={USER_FORM.LABEL_SELECT_ROLE}
            >
              <MenuItem value={ORG_ROLE.MEMBER}>
                {ORG_USERS_PAGE.ROLE_MEMBER}
              </MenuItem>
              <MenuItem value={ORG_ROLE.ADMIN}>
                {ORG_USERS_PAGE.ROLE_ADMIN}
              </MenuItem>
              <MenuItem value={ORG_ROLE.OWNER}>
                {ORG_USERS_PAGE.ROLE_OWNER}
              </MenuItem>
            </Select>
          </FormControl>
          <Button
            variant="contained"
            size="small"
            startIcon={<AddIcon />}
            onClick={onAddMembership}
            disabled={!selectedOrg || isAddPending}
            sx={{ mt: 0.5 }}
          >
            {USER_FORM.ACTION_ADD_MEMBERSHIP}
          </Button>
        </Box>
      </Box>
    </CardContent>
  </Card>
);

const UserDetail: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { isAdmin, isHydrated } = useSession();

  const {
    data: user,
    isLoading,
    isError,
    error,
  } = useUser(id || "", {
    enabled: isHydrated && isAdmin && Boolean(id),
  });
  const updateUser = useUpdateUser();
  const deleteUser = useDeleteUser();
  const canNavigateToPassword = Boolean(id);

  const { data: userOrgsData, isLoading: isOrgsLoading } = useUserOrganizations(
    id || "",
    {
      enabled: isHydrated && isAdmin && Boolean(id),
    },
  );
  const memberships = userOrgsData?.memberships ?? [];

  const { data: allOrgsData } = useOrganizations(200, 0, undefined, {
    enabled: isHydrated && isAdmin && Boolean(id),
  });
  const allOrganizations = allOrgsData?.organizations ?? [];

  const availableOrgs = allOrganizations.filter(
    (org) => !memberships.some((m) => m.orgId === org.id),
  );

  const addOrgUser = useAddOrgUser();
  const removeOrgUser = useRemoveOrgUser();

  const [form, setForm] = useState<UserProfileForm>({
    email: "",
    note: "",
    isUserAdmin: false,
    isActive: true,
    isTenantManager: false,
    isBaseStationManager: false,
    isEndpointManager: false,
  });
  const updateForm = <K extends keyof UserProfileForm>(
    key: K,
    value: UserProfileForm[K],
  ) => setForm((prev) => ({ ...prev, [key]: value }));

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [successMessage, setSuccessMessage] = useState("");

  const [selectedOrg, setSelectedOrg] = useState<OrganizationUI | null>(null);
  const [selectedRole, setSelectedRole] = useState<string>(ORG_ROLE.MEMBER);
  const [membershipError, setMembershipError] = useState("");
  const [membershipSuccess, setMembershipSuccess] = useState("");

  useEffect(() => {
    if (user) {
      setForm({
        email: user.email,
        note: user.note || "",
        isUserAdmin: user.isAdmin,
        isActive: user.isActive,
        isTenantManager: user.isTenantManager,
        isBaseStationManager: user.isBaseStationManager,
        isEndpointManager: user.isEndpointManager,
      });
    }
  }, [user]);

  const handleSave = async () => {
    if (!id || !user) return;
    const updates = buildUpdateUserPayload(form, user);
    if (Object.keys(updates).length === 0) return;
    try {
      await updateUser.mutateAsync({ id, data: updates });
      setSuccessMessage(MSG_USER_UPDATED);
      setTimeout(
        () => setSuccessMessage(""),
        UI_TIMING.SUCCESS_MESSAGE_DISMISS_MS,
      );
    } catch {
      // Surfaced via updateUser.isError
    }
  };

  const handleDelete = async () => {
    if (!id) return;
    try {
      await deleteUser.mutateAsync(id);
      setSuccessMessage(MSG_USER_DELETED);
      navigate(ROUTES.USERS);
    } catch {
      // Surfaced via deleteUser.isError
    }
    setDeleteDialogOpen(false);
  };

  const handleBackToList = () => {
    navigate(ROUTES.USERS);
  };

  const handleAddMembership = async () => {
    if (!id || !selectedOrg) return;
    setMembershipError("");
    setMembershipSuccess("");
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
      setTimeout(
        () => setMembershipSuccess(""),
        UI_TIMING.SUCCESS_MESSAGE_DISMISS_MS,
      );
    } catch {
      setMembershipError(USER_FORM.ERR_MEMBERSHIP_ADD_FAILED);
    }
  };

  const handleRemoveMembership = async (orgId: string) => {
    if (!id) return;
    setMembershipError("");
    setMembershipSuccess("");
    try {
      await removeOrgUser.mutateAsync({ orgId, userId: id });
      setMembershipSuccess(USER_FORM.MSG_MEMBERSHIP_REMOVED);
      setTimeout(
        () => setMembershipSuccess(""),
        UI_TIMING.SUCCESS_MESSAGE_DISMISS_MS,
      );
    } catch {
      setMembershipError(USER_FORM.ERR_MEMBERSHIP_REMOVE_FAILED);
    }
  };

  if (!isHydrated) return null;
  if (!isAdmin) return <Navigate to={ROUTES.HOME} replace />;

  if (isLoading) {
    return (
      <Box
        sx={{
          pt: 4,
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          minHeight: "400px",
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
      <UserDetailHeader onBack={handleBackToList} />

      {successMessage && (
        <Alert severity="success" sx={{ mb: 3 }}>
          {successMessage}
        </Alert>
      )}

      {updateUser.isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {updateUser.error instanceof Error
            ? updateUser.error.message
            : USER_FORM.ERR_UPDATE_FAILED}
        </Alert>
      )}

      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 8 }}>
          <UserProfileCard
            form={form}
            isSaving={updateUser.isPending}
            isDeleting={deleteUser.isPending}
            onChange={updateForm}
            onSave={handleSave}
            onDeleteClick={() => setDeleteDialogOpen(true)}
          />
        </Grid>

        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                {USER_FORM.INFO_TITLE}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {USER_FORM.INFO_CREATED}:{" "}
                {formatRelativeDuration(user.createdAt)}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {USER_FORM.INFO_UPDATED}:{" "}
                {formatRelativeDuration(user.updatedAt)}
              </Typography>
              <Button
                variant="outlined"
                onClick={() => {
                  if (!id) return;
                  navigate(ROUTES.USER_PASSWORD.replace(":id", id));
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

        <Grid size={{ xs: 12 }}>
          <UserMembershipsCard
            memberships={memberships}
            availableOrgs={availableOrgs}
            isOrgsLoading={isOrgsLoading}
            isRemovePending={removeOrgUser.isPending}
            isAddPending={addOrgUser.isPending}
            selectedOrg={selectedOrg}
            selectedRole={selectedRole}
            membershipSuccess={membershipSuccess}
            membershipError={membershipError}
            onSelectedOrgChange={setSelectedOrg}
            onSelectedRoleChange={setSelectedRole}
            onAddMembership={handleAddMembership}
            onRemoveMembership={handleRemoveMembership}
          />
        </Grid>
      </Grid>

      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
      >
        <DialogTitle>{USER_FORM.ACTION_DELETE}</DialogTitle>
        <DialogContent>
          <DialogContentText>{USER_FORM.CONFIRM_DELETE}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>
            {USER_FORM.ACTION_CANCEL}
          </Button>
          <Button
            color="error"
            onClick={handleDelete}
            disabled={deleteUser.isPending}
          >
            {USER_FORM.ACTION_DELETE}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default UserDetail;
