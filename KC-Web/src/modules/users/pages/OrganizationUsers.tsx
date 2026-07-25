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
  Typography,
} from "@mui/material";

import SearchField from "@components/common/SearchField";
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
import { AddIcon, ArrowBackIcon, PeopleIcon } from "@theme/icons";

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

/** Classifies a removeOrgUser failure to the message shown in the dialog. */
function classifyRemoveError(err: unknown): string {
  const apiError = err as ApiError | undefined;

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
      apiError.message.toLowerCase().includes("cannot remove yourself"));

  if (isLastOwnerError) return ORG_USERS_PAGE.ERR_CANNOT_REMOVE_LAST_OWNER;
  if (isSelfRemovalError) return ORG_USERS_PAGE.ERR_CANNOT_REMOVE_SELF;
  return ORG_USER_FORM.ERR_REMOVE_FAILED;
}

/** Pure sort over the OrderBy-enumerated string fields. */
function sortUsers(
  users: OrganizationUserUI[],
  orderBy: OrderBy,
  direction: OrderDirection,
): OrganizationUserUI[] {
  return [...users].sort((a, b) => {
    const aValue = a[orderBy] ?? "";
    const bValue = b[orderBy] ?? "";
    if (direction === "asc") {
      return aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
    }
    return aValue > bValue ? -1 : aValue < bValue ? 1 : 0;
  });
}

interface OrgUsersHeaderProps {
  embedded: boolean;
  orgName?: string;
  orgId: string;
  onAddClick: () => void;
}

const OrgUsersHeader: React.FC<OrgUsersHeaderProps> = ({
  embedded,
  orgName,
  orgId,
  onAddClick,
}) => {
  const navigate = useNavigate();
  return (
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
        {orgName && (
          <Typography variant="h6" color="text.secondary">
            - {orgName}
          </Typography>
        )}
      </Box>
      {!embedded && (
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={onAddClick}
        >
          {ORG_USERS_PAGE.ADD_USER}
        </Button>
      )}
    </Box>
  );
};

interface OrgUsersRemoveDialogProps {
  open: boolean;
  selectedUser: OrganizationUserUI | null;
  removeError: string | null;
  isPending: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

const OrgUsersRemoveDialog: React.FC<OrgUsersRemoveDialogProps> = ({
  open,
  selectedUser,
  removeError,
  isPending,
  onClose,
  onConfirm,
}) => (
  <Dialog open={open} onClose={onClose}>
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
      <Button onClick={onClose}>{ORG_USER_FORM.ACTION_CANCEL}</Button>
      <Button
        onClick={onConfirm}
        color="error"
        variant="contained"
        disabled={isPending}
      >
        {ORG_USER_FORM.ACTION_REMOVE}
      </Button>
    </DialogActions>
  </Dialog>
);

/** Search + sort state over the loaded member list. */
function useOrgUsersFiltering(users: OrganizationUserUI[]) {
  const [search, setSearch] = useState("");
  const [orderBy, setOrderBy] = useState<OrderBy>("email");
  const [orderDirection, setOrderDirection] = useState<OrderDirection>("asc");

  const filteredUsers = useMemo(() => {
    const searchLower = search.toLowerCase();
    return users.filter((user) =>
      user.email.toLowerCase().includes(searchLower),
    );
  }, [users, search]);

  const sortedUsers = useMemo(
    () => sortUsers(filteredUsers, orderBy, orderDirection),
    [filteredUsers, orderBy, orderDirection],
  );

  const handleSort = (field: OrderBy) => {
    if (orderBy === field) {
      setOrderDirection(orderDirection === "asc" ? "desc" : "asc");
    } else {
      setOrderBy(field);
      setOrderDirection("asc");
    }
  };

  return {
    search,
    setSearch,
    orderBy,
    orderDirection,
    sortedUsers,
    handleSort,
  };
}

/** Add/edit/remove dialog state and handlers, including the remove mutation. */
function useOrgUsersDialogs(
  orgId: string,
  externalAddDialogOpen?: boolean,
  onAddDialogOpenChange?: (open: boolean) => void,
) {
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

  const removeOrgUser = useRemoveOrgUser();

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
    setRemoveError(null);
    setRemoveConfirmOpen(true);
  };

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
          setRemoveError(classifyRemoveError(err));
        },
      },
    );
  };

  const handleRemoveDialogClose = () => {
    setRemoveConfirmOpen(false);
    setSelectedUser(null);
    setRemoveError(null);
  };

  const handleAddDialogClose = () => {
    setAddDialogOpen(false);
  };

  const handleEditDialogClose = () => {
    setEditDialogOpen(false);
    setSelectedUser(null);
  };

  return {
    isAddDialogOpen,
    editDialogOpen,
    selectedUser,
    removeConfirmOpen,
    removeError,
    isRemovePending: removeOrgUser.isPending,
    handleAddClick,
    handleEdit,
    handleRemove,
    confirmRemove,
    handleRemoveDialogClose,
    handleAddDialogClose,
    handleEditDialogClose,
  };
}

interface OrgUsersToolbarProps {
  memberCount: number;
  search: string;
  onSearchChange: (value: string) => void;
}

/** Member-count stat card and search field shown above the table. */
const OrgUsersToolbar: React.FC<OrgUsersToolbarProps> = ({
  memberCount,
  search,
  onSearchChange,
}) => (
  <>
    <Grid container spacing={3} mb={3}>
      <Grid size={{ xs: 12, sm: 6, md: 4 }}>
        <Card>
          <CardContent>
            <Box display="flex" alignItems="center">
              <PeopleIcon sx={{ fontSize: 40, color: "primary.main", mr: 2 }} />
              <Box>
                <Typography color="text.secondary" variant="body2">
                  {ORG_USERS_PAGE.TOTAL_MEMBERS}
                </Typography>
                <Typography variant="h4">{memberCount}</Typography>
              </Box>
            </Box>
          </CardContent>
        </Card>
      </Grid>
    </Grid>

    <Box display="flex" gap={2} mb={3}>
      <SearchField
        placeholder={ORG_USERS_PAGE.SEARCH_PLACEHOLDER}
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
      />
    </Box>
  </>
);

const OrganizationUsers: React.FC<OrganizationUsersProps> = ({
  orgId: propOrgId,
  embedded,
  addDialogOpen: externalAddDialogOpen,
  onAddDialogOpenChange,
}) => {
  const { id: routeOrgId } = useParams<{ id: string }>();
  const { organizationId: contextOrgId, setOrganization } =
    useOrganizationContext();
  const { isHydrated } = useSession();
  const { isServerAdmin, isOrgAdmin } = useCapabilities();

  // Resolve orgId: prop > route param > context
  const orgId = propOrgId || routeOrgId || contextOrgId || "";

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

  // Set organization context when org data loads
  useEffect(() => {
    if (org) {
      setOrganization(org.id, org.name);
    }
  }, [org, setOrganization]);

  const users = useMemo(() => usersData?.users ?? [], [usersData?.users]);
  const {
    search,
    setSearch,
    orderBy,
    orderDirection,
    sortedUsers,
    handleSort,
  } = useOrgUsersFiltering(users);
  const dialogs = useOrgUsersDialogs(
    orgId,
    externalAddDialogOpen,
    onAddDialogOpenChange,
  );

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
      <OrgUsersHeader
        embedded={Boolean(embedded)}
        orgName={org?.name}
        orgId={orgId}
        onAddClick={dialogs.handleAddClick}
      />

      <OrgUsersToolbar
        memberCount={users.length}
        search={search}
        onSearchChange={setSearch}
      />

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
          onEdit={dialogs.handleEdit}
          onRemove={dialogs.handleRemove}
          emptyMessage={
            search ? ORG_USERS_PAGE.NO_MATCH : ORG_USERS_PAGE.NO_USERS
          }
        />
      )}

      {/* Add User Dialog - server admin creates new system user; org admin invites by email */}
      {isServerAdmin ? (
        <AddUserDialog
          open={dialogs.isAddDialogOpen}
          onClose={dialogs.handleAddDialogClose}
          orgId={orgId}
        />
      ) : (
        <OrganizationUserDialog
          open={dialogs.isAddDialogOpen}
          onClose={dialogs.handleAddDialogClose}
          orgId={orgId}
          mode="add"
        />
      )}

      {/* Edit Organization User Dialog - edits role/permissions of existing member */}
      <OrganizationUserDialog
        open={dialogs.editDialogOpen}
        onClose={dialogs.handleEditDialogClose}
        orgId={orgId}
        mode="edit"
        initialUser={dialogs.selectedUser ?? undefined}
      />

      <OrgUsersRemoveDialog
        open={dialogs.removeConfirmOpen}
        selectedUser={dialogs.selectedUser}
        removeError={dialogs.removeError}
        isPending={dialogs.isRemovePending}
        onClose={dialogs.handleRemoveDialogClose}
        onConfirm={dialogs.confirmRemove}
      />
    </Box>
  );
};

export default OrganizationUsers;
