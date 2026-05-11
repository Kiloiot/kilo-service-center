/**
 * Organization Users Page
 *
 * Organization members management with support for:
 *   - Route-based orgId (from /organizations/:id/users)
 *   - Context-based orgId (for /users Organization Users tab)
 *   - Edit/remove member functionality with confirmation dialogs
 *
 * Admin-only page with runtime guard for deep link protection.
 */

import React, { useEffect, useMemo, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";

import type { ApiError, OrganizationUserUI } from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Grid,
  InputAdornment,
  TextField,
  Typography,
} from "@mui/material";

import { useOrganization as useOrganizationContext } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import { useCapabilities } from "@hooks/useCapabilities";
import {
  useOrganization as useOrganizationQuery,
  useOrgUsers,
  useRemoveOrgUser,
} from "@hooks/useOrganizations";
import { ROUTES } from "@constants/app";
import {
  ERR_LOAD_ORG_USERS,
  ORG_USER_FORM,
  ORG_USERS_PAGE,
} from "@constants/messages";
import { AddIcon, ArrowBackIcon, PeopleIcon, SearchIcon } from "@theme/icons";

import AddUserDialog from "../components/AddUserDialog";
import OrganizationUserDialog from "../components/OrganizationUserDialog";
import OrganizationUsersTable from "../components/OrganizationUsersTable";

type OrderBy = "email" | "role" | "status" | "createdAt";
type OrderDirection = "asc" | "desc";

export interface OrganizationUsersProps {
  /** Optional orgId prop - if not provided, falls back to route param then context */
  orgId?: string;
  /** When true, hide the back button and "Add User" button (for use in tabs) */
  embedded?: boolean;
  /** Controlled add-dialog state from parent (used when embedded) */
  addDialogOpen?: boolean;
  /** Callback to update add-dialog state in parent (used when embedded) */
  onAddDialogOpenChange?: (open: boolean) => void;
}

const OrganizationUsers: React.FC<OrganizationUsersProps> = ({
  orgId: propOrgId,
  embedded,
  addDialogOpen: externalAddDialogOpen,
  onAddDialogOpenChange,
}) => {
  const navigate = useNavigate();
  const { id: routeOrgId } = useParams<{ id: string }>();
  const { organizationId: contextOrgId, setOrganization } =
    useOrganizationContext();
  const { isHydrated } = useSession();
  const { isServerAdmin, isOrgAdmin } = useCapabilities();

  // Resolve orgId: prop > route param > context
  const orgId = propOrgId || routeOrgId || contextOrgId || "";

  // Local UI state
  const [search, setSearch] = useState("");
  const [orderBy, setOrderBy] = useState<OrderBy>("email");
  const [orderDirection, setOrderDirection] = useState<OrderDirection>("asc");

  // Dialog state - separate dialogs for add (new user) vs edit (existing member)
  const [localAddDialogOpen, setLocalAddDialogOpen] = useState(false);
  const isAddDialogOpen = externalAddDialogOpen ?? localAddDialogOpen;
  const setAddDialogOpen = onAddDialogOpenChange ?? setLocalAddDialogOpen;
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<OrganizationUserUI | null>(
    null,
  );

  // Remove confirmation state
  const [removeConfirmOpen, setRemoveConfirmOpen] = useState(false);
  const [removeError, setRemoveError] = useState<string | null>(null);

  // React Query hooks
  const canQuery =
    isHydrated && (isServerAdmin || isOrgAdmin) && Boolean(orgId);
  const { data: org } = useOrganizationQuery(orgId, {
    enabled: canQuery && isServerAdmin,
  });
  const {
    data: usersData,
    isLoading,
    isError,
    error,
  } = useOrgUsers(orgId, undefined, {
    enabled: canQuery,
  });

  // Remove mutation
  const removeOrgUser = useRemoveOrgUser();

  // Set organization context when org data loads
  useEffect(() => {
    if (org) {
      setOrganization(org.id, org.name);
    }
  }, [org, setOrganization]);

  // Memoize users array
  const users = useMemo(() => usersData?.users ?? [], [usersData?.users]);

  // Filter users by search
  const filteredUsers = useMemo(() => {
    const searchLower = search.toLowerCase();
    return users.filter((user) =>
      user.email.toLowerCase().includes(searchLower),
    );
  }, [users, search]);

  // Sort users
  const sortedUsers = useMemo(() => {
    return [...filteredUsers].sort((a, b) => {
      let aValue: string = "";
      let bValue: string = "";

      if (orderBy === "email") {
        aValue = a.email;
        bValue = b.email;
      } else if (orderBy === "role") {
        aValue = a.role;
        bValue = b.role;
      } else if (orderBy === "status") {
        aValue = a.status;
        bValue = b.status;
      } else if (orderBy === "createdAt") {
        aValue = a.createdAt;
        bValue = b.createdAt;
      }

      if (orderDirection === "asc") {
        return aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
      }
      return aValue > bValue ? -1 : aValue < bValue ? 1 : 0;
    });
  }, [filteredUsers, orderBy, orderDirection]);

  const handleSort = (field: OrderBy) => {
    if (orderBy === field) {
      setOrderDirection(orderDirection === "asc" ? "desc" : "asc");
    } else {
      setOrderBy(field);
      setOrderDirection("asc");
    }
  };

  const handleAddClick = () => {
    setAddDialogOpen(true);
  };

  // Edit member handler - opens OrganizationUserDialog for editing
  const handleEdit = (user: OrganizationUserUI) => {
    setSelectedUser(user);
    setEditDialogOpen(true);
  };

  // Remove member handler - opens confirmation dialog
  const handleRemove = (user: OrganizationUserUI) => {
    setSelectedUser(user);
    setRemoveError(null); // Reset error when opening dialog
    setRemoveConfirmOpen(true);
  };

  // Confirm removal
  const confirmRemove = () => {
    if (!selectedUser) return;

    removeOrgUser.mutate(
      { orgId, userId: selectedUser.userId },
      {
        onSuccess: () => {
          setRemoveConfirmOpen(false);
          setSelectedUser(null);
          setRemoveError(null);
        },
        onError: (err: unknown) => {
          const apiError = err as ApiError;

          const isLastOwnerError =
            apiError?.status === 409 ||
            apiError?.code === "LAST_OWNER" ||
            apiError?.token === "LAST_OWNER" ||
            (typeof apiError?.message === "string" &&
              apiError.message.toLowerCase().includes("last owner"));

          const isSelfRemovalError =
            apiError?.status === 412 ||
            apiError?.token === "KC-GRPC-ERR-306" ||
            (typeof apiError?.message === "string" &&
              apiError.message
                .toLowerCase()
                .includes("cannot remove yourself"));

          if (isLastOwnerError) {
            setRemoveError(ORG_USERS_PAGE.ERR_CANNOT_REMOVE_LAST_OWNER);
          } else if (isSelfRemovalError) {
            setRemoveError(ORG_USERS_PAGE.ERR_CANNOT_REMOVE_SELF);
          } else {
            setRemoveError(ORG_USER_FORM.ERR_REMOVE_FAILED);
          }
        },
      },
    );
  };

  // Close remove confirmation dialog
  const handleRemoveDialogClose = () => {
    setRemoveConfirmOpen(false);
    setSelectedUser(null);
    setRemoveError(null); // Reset error when closing
  };

  const handleAddDialogClose = () => {
    setAddDialogOpen(false);
  };

  // Close edit dialog
  const handleEditDialogClose = () => {
    setEditDialogOpen(false);
    setSelectedUser(null);
  };

  if (!isHydrated) {
    return null;
  }

  // Runtime guard (deep link protection) - requires server admin or org admin
  if (!isServerAdmin && !isOrgAdmin) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  return (
    <Box
      data-testid="organization-users-page"
      sx={{ p: embedded ? 0 : 3, pt: embedded ? 0 : 4 }}
    >
      {/* Header */}
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Box display="flex" alignItems="center" gap={2}>
          {!embedded && (
            <Button
              startIcon={<ArrowBackIcon />}
              onClick={() => navigate(`${ROUTES.ORGANIZATIONS}/${orgId}`)}
            >
              {ORG_USERS_PAGE.BACK_TO_ORG}
            </Button>
          )}
          <Typography variant="h4" component="h1">
            {ORG_USERS_PAGE.TITLE}
          </Typography>
          {org && (
            <Typography variant="h6" color="text.secondary">
              - {org.name}
            </Typography>
          )}
        </Box>
        {!embedded && (
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleAddClick}
          >
            {ORG_USERS_PAGE.ADD_USER}
          </Button>
        )}
      </Box>

      {/* Statistics Cards */}
      <Grid container spacing={3} mb={3}>
        <Grid size={{ xs: 12, sm: 6, md: 4 }}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <PeopleIcon
                  sx={{ fontSize: 40, color: "primary.main", mr: 2 }}
                />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {ORG_USERS_PAGE.TOTAL_MEMBERS}
                  </Typography>
                  <Typography variant="h4">{users.length}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Search */}
      <Box display="flex" gap={2} mb={3}>
        <TextField
          placeholder={ORG_USERS_PAGE.SEARCH_PLACEHOLDER}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          sx={{ flexGrow: 1 }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
      </Box>

      {/* Loading State */}
      {isLoading && (
        <Box
          display="flex"
          justifyContent="center"
          alignItems="center"
          minHeight="200px"
        >
          <CircularProgress />
        </Box>
      )}

      {/* Error Alert */}
      {isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error instanceof Error ? error.message : ERR_LOAD_ORG_USERS}
        </Alert>
      )}

      {/* Users Table */}
      {!isLoading && !isError && (
        <OrganizationUsersTable
          users={sortedUsers}
          orderBy={orderBy}
          orderDirection={orderDirection}
          onSort={handleSort}
          onEdit={handleEdit}
          onRemove={handleRemove}
          emptyMessage={
            search ? ORG_USERS_PAGE.NO_MATCH : ORG_USERS_PAGE.NO_USERS
          }
        />
      )}

      {/* Add User Dialog - server admin creates new system user; org admin invites by email */}
      {isServerAdmin ? (
        <AddUserDialog
          open={isAddDialogOpen}
          onClose={handleAddDialogClose}
          orgId={orgId}
        />
      ) : (
        <OrganizationUserDialog
          open={isAddDialogOpen}
          onClose={handleAddDialogClose}
          orgId={orgId}
          mode="add"
        />
      )}

      {/* Edit Organization User Dialog - edits role/permissions of existing member */}
      <OrganizationUserDialog
        open={editDialogOpen}
        onClose={handleEditDialogClose}
        orgId={orgId}
        mode="edit"
        initialUser={selectedUser ?? undefined}
      />

      {/* Remove Confirmation Dialog */}
      <Dialog open={removeConfirmOpen} onClose={handleRemoveDialogClose}>
        <DialogTitle>{ORG_USER_FORM.ACTION_REMOVE}</DialogTitle>
        <DialogContent>
          {removeError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {removeError}
            </Alert>
          )}
          <DialogContentText>{ORG_USER_FORM.CONFIRM_REMOVE}</DialogContentText>
          {selectedUser && (
            <Typography variant="body2" sx={{ mt: 1, fontWeight: "medium" }}>
              {selectedUser.email}
            </Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={handleRemoveDialogClose}>
            {ORG_USER_FORM.ACTION_CANCEL}
          </Button>
          <Button
            onClick={confirmRemove}
            color="error"
            variant="contained"
            disabled={removeOrgUser.isPending}
          >
            {ORG_USER_FORM.ACTION_REMOVE}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default OrganizationUsers;
