/**
 * Users Page
 *
 * System user management list page with edit/delete action buttons.
 */

import React, { useMemo, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";

import type { OrganizationUserUI, SystemUserUI } from "@api-types/api";
import { useDeleteUser, useUsers } from "@hooks";
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

import { useSession } from "@contexts/SessionContext";
import { ORG_MEMBER_STATUS, ORG_ROLE, ROUTES } from "@constants/app";
import { ERR_LOAD_USERS, USERS_PAGE } from "@constants/messages";
import {
  AddIcon,
  AdminIcon,
  ErrorIcon,
  PeopleIcon,
  SuccessIcon,
} from "@theme/icons";

import SearchField from "@components/common/SearchField";

import AddUserDialog from "../components/AddUserDialog";
import UsersTableBase, { type OrderBy } from "../components/UsersTableBase";

type OrderDirection = "asc" | "desc";

export interface UsersProps {
  /** When true, reduces padding for use inside tabs */
  embedded?: boolean;
  addDialogOpen?: boolean;
  onAddDialogOpenChange?: (open: boolean) => void;
}

const Users: React.FC<UsersProps> = ({
  embedded,
  addDialogOpen,
  onAddDialogOpenChange,
}) => {
  const navigate = useNavigate();
  const { isAdmin, isHydrated } = useSession();

  // Local UI state
  const [search, setSearch] = useState("");
  const [orderBy, setOrderBy] = useState<OrderBy>("email");
  const [orderDirection, setOrderDirection] = useState<OrderDirection>("asc");
  const [localAddDialogOpen, setLocalAddDialogOpen] = useState(false);
  const isAddDialogOpen = addDialogOpen ?? localAddDialogOpen;
  const setAddDialogOpen = onAddDialogOpenChange ?? setLocalAddDialogOpen;

  // Delete confirmation state
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<SystemUserUI | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  // React Query hooks
  const { data, isLoading, isError, error } = useUsers(50, 0, {
    enabled: isHydrated && isAdmin,
  });
  const deleteUser = useDeleteUser();

  // Memoize users array to prevent unnecessary re-renders
  const users = useMemo(() => data?.users ?? [], [data?.users]);

  // Filter users by search
  const filteredUsers = useMemo(() => {
    const searchLower = search.toLowerCase();
    return users.filter(
      (user) =>
        user.email.toLowerCase().includes(searchLower) ||
        user.note?.toLowerCase().includes(searchLower),
    );
  }, [users, search]);

  // Sort users using unified role/status values
  const sortedUsers = useMemo(() => {
    return [...filteredUsers].sort((a, b) => {
      let aValue = "";
      let bValue = "";

      if (orderBy === "email") {
        aValue = a.email;
        bValue = b.email;
      } else if (orderBy === "role") {
        aValue = a.isAdmin ? ORG_ROLE.ADMIN : ORG_ROLE.MEMBER;
        bValue = b.isAdmin ? ORG_ROLE.ADMIN : ORG_ROLE.MEMBER;
      } else if (orderBy === "status") {
        aValue = a.isActive
          ? ORG_MEMBER_STATUS.ACTIVE
          : ORG_MEMBER_STATUS.REMOVED;
        bValue = b.isActive
          ? ORG_MEMBER_STATUS.ACTIVE
          : ORG_MEMBER_STATUS.REMOVED;
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

  const handleEdit = (user: SystemUserUI | OrganizationUserUI) => {
    const id = "id" in user ? (user as SystemUserUI).id : "";
    navigate(`${ROUTES.USERS}/${id}`);
  };

  const handleRemove = (user: SystemUserUI | OrganizationUserUI) => {
    setSelectedUser(user as SystemUserUI);
    setDeleteError(null);
    setDeleteConfirmOpen(true);
  };

  const confirmDelete = () => {
    if (!selectedUser) return;

    deleteUser.mutate(selectedUser.id, {
      onSuccess: () => {
        setDeleteConfirmOpen(false);
        setSelectedUser(null);
        setDeleteError(null);
      },
      onError: (err: unknown) => {
        setDeleteError(
          err instanceof Error ? err.message : USERS_PAGE.ERR_DELETE_FAILED,
        );
      },
    });
  };

  const handleDeleteDialogClose = () => {
    setDeleteConfirmOpen(false);
    setSelectedUser(null);
    setDeleteError(null);
  };

  // Calculate statistics
  const activeCount = users.filter((u: SystemUserUI) => u.isActive).length;
  const inactiveCount = users.length - activeCount;
  const adminCount = users.filter((u: SystemUserUI) => u.isAdmin).length;

  if (!isHydrated) {
    return null;
  }

  if (!isAdmin) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  return (
    <Box
      data-testid="users-page"
      sx={{ p: embedded ? 0 : 3, pt: embedded ? 0 : 4 }}
    >
      {!embedded && (
        <Box
          display="flex"
          justifyContent="space-between"
          alignItems="center"
          mb={3}
        >
          <Typography variant="h4" component="h1">
            {USERS_PAGE.TITLE}
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setAddDialogOpen(true)}
          >
            {USERS_PAGE.ADD_USER}
          </Button>
        </Box>
      )}

      {/* Statistics Cards */}
      <Grid container spacing={3} mb={3}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <PeopleIcon
                  sx={{ fontSize: 40, color: "primary.main", mr: 2 }}
                />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {USERS_PAGE.TOTAL_USERS}
                  </Typography>
                  <Typography variant="h4">{users.length}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <SuccessIcon
                  sx={{ fontSize: 40, color: "success.main", mr: 2 }}
                />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {USERS_PAGE.ACTIVE_USERS}
                  </Typography>
                  <Typography variant="h4">{activeCount}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <ErrorIcon sx={{ fontSize: 40, color: "error.main", mr: 2 }} />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {USERS_PAGE.INACTIVE_USERS}
                  </Typography>
                  <Typography variant="h4">{inactiveCount}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <AdminIcon
                  sx={{ fontSize: 40, color: "warning.main", mr: 2 }}
                />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {USERS_PAGE.ADMIN_USERS}
                  </Typography>
                  <Typography variant="h4">{adminCount}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Search */}
      <Box display="flex" gap={2} mb={3}>
        <SearchField
          placeholder={USERS_PAGE.SEARCH_PLACEHOLDER}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
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
          {error instanceof Error ? error.message : ERR_LOAD_USERS}
        </Alert>
      )}

      {/* Users Table */}
      {!isLoading && !isError && (
        <UsersTableBase
          users={sortedUsers}
          orderBy={orderBy}
          orderDirection={orderDirection}
          onSort={handleSort}
          onEdit={handleEdit}
          onRemove={handleRemove}
          emptyMessage={search ? USERS_PAGE.NO_MATCH : USERS_PAGE.NO_USERS}
        />
      )}

      {/* Add User Dialog */}
      <AddUserDialog
        open={isAddDialogOpen}
        onClose={() => setAddDialogOpen(false)}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteConfirmOpen} onClose={handleDeleteDialogClose}>
        <DialogTitle>{USERS_PAGE.CONFIRM_DELETE_TITLE}</DialogTitle>
        <DialogContent>
          {deleteError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {deleteError}
            </Alert>
          )}
          <DialogContentText>
            {USERS_PAGE.CONFIRM_DELETE_MESSAGE}
          </DialogContentText>
          {selectedUser && (
            <Typography variant="body2" sx={{ mt: 1, fontWeight: "medium" }}>
              {selectedUser.email}
            </Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={handleDeleteDialogClose}>
            {USERS_PAGE.ACTION_CANCEL}
          </Button>
          <Button
            onClick={confirmDelete}
            color="error"
            variant="contained"
            disabled={deleteUser.isPending}
          >
            {USERS_PAGE.ACTION_DELETE}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Users;
