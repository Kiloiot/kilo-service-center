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
  CircularProgress,
  Typography,
} from "@mui/material";
import { ConfirmDialog } from "@ui";

import SearchField from "@components/common/SearchField";
import { StatCard, StatCardRow } from "@components/common/StatCard";
import { useFeedback } from "@contexts/feedback";
import { useSession } from "@contexts/SessionContext";
import { useCapabilities } from "@hooks/useCapabilities";
import { getErrorMessage } from "@utils/error-message";
import { toggleSortDirection } from "@utils/list-query";
import type { SortDirection } from "@constants/app";
import { PAGINATION, ROUTES, SORT_DIRECTION } from "@constants/app";
import {
  ERR_LOAD_USERS,
  MSG_USER_DELETED,
  USERS_PAGE,
} from "@constants/messages";
import { userDetailPath } from "@router/paths";
import {
  AddIcon,
  AdminIcon,
  ErrorIcon,
  PeopleIcon,
  SuccessIcon,
} from "@theme/icons";
import { componentSpacing } from "@theme/index";

import AddUserDialog from "../components/AddUserDialog";
import UsersTableBase, { type OrderBy } from "../components/UsersTableBase";
import { systemUserRole, systemUserStatus } from "../utils/membership-labels";

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
  const { isHydrated } = useSession();
  const { isServerAdmin: isAdmin } = useCapabilities();

  // Local UI state
  const [search, setSearch] = useState("");
  const [orderBy, setOrderBy] = useState<OrderBy>("email");
  const [orderDirection, setOrderDirection] = useState<SortDirection>(
    SORT_DIRECTION.ASC,
  );
  const [localAddDialogOpen, setLocalAddDialogOpen] = useState(false);
  const isAddDialogOpen = addDialogOpen ?? localAddDialogOpen;
  const setAddDialogOpen = onAddDialogOpenChange ?? setLocalAddDialogOpen;

  // Delete confirmation state
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<SystemUserUI | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  // React Query hooks
  const { data, isLoading, isError, error } = useUsers(
    PAGINATION.ADMIN_LIST_PAGE_SIZE,
    0,
    {
      enabled: isHydrated && isAdmin,
    },
  );
  const deleteUser = useDeleteUser();
  const feedback = useFeedback();

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
        aValue = systemUserRole(a);
        bValue = systemUserRole(b);
      } else if (orderBy === "status") {
        aValue = systemUserStatus(a);
        bValue = systemUserStatus(b);
      } else if (orderBy === "createdAt") {
        aValue = a.createdAt;
        bValue = b.createdAt;
      }

      if (orderDirection === SORT_DIRECTION.ASC) {
        return aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
      }
      return aValue > bValue ? -1 : aValue < bValue ? 1 : 0;
    });
  }, [filteredUsers, orderBy, orderDirection]);

  const handleSort = (field: OrderBy) => {
    if (orderBy === field) {
      setOrderDirection(toggleSortDirection(orderDirection));
    } else {
      setOrderBy(field);
      setOrderDirection(SORT_DIRECTION.ASC);
    }
  };

  const handleEdit = (user: SystemUserUI | OrganizationUserUI) => {
    const id = "id" in user ? (user as SystemUserUI).id : "";
    navigate(userDetailPath(id));
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
        feedback.success(MSG_USER_DELETED);
      },
      onError: (err: unknown) => {
        setDeleteError(getErrorMessage(err, USERS_PAGE.ERR_DELETE_FAILED));
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

      <StatCardRow>
        <StatCard
          label={USERS_PAGE.TOTAL_USERS}
          value={users.length}
          icon={<PeopleIcon />}
          color="primary"
        />
        <StatCard
          label={USERS_PAGE.ACTIVE_USERS}
          value={activeCount}
          icon={<SuccessIcon />}
          color="success"
        />
        <StatCard
          label={USERS_PAGE.INACTIVE_USERS}
          value={inactiveCount}
          icon={<ErrorIcon />}
          color="error"
        />
        <StatCard
          label={USERS_PAGE.ADMIN_USERS}
          value={adminCount}
          icon={<AdminIcon />}
          color="warning"
        />
      </StatCardRow>

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
          minHeight={componentSpacing.stateView.listMinHeight}
        >
          <CircularProgress />
        </Box>
      )}

      {/* Error Alert */}
      {isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {getErrorMessage(error, ERR_LOAD_USERS)}
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

      <ConfirmDialog
        open={deleteConfirmOpen}
        onClose={handleDeleteDialogClose}
        onConfirm={confirmDelete}
        pending={deleteUser.isPending}
        error={deleteError}
        title={USERS_PAGE.CONFIRM_DELETE_TITLE}
        message={USERS_PAGE.CONFIRM_DELETE_MESSAGE}
        confirmLabel={USERS_PAGE.ACTION_DELETE}
        cancelLabel={USERS_PAGE.ACTION_CANCEL}
      >
        {selectedUser && (
          <Typography variant="body2" sx={{ mt: 1, fontWeight: "medium" }}>
            {selectedUser.email}
          </Typography>
        )}
      </ConfirmDialog>
    </Box>
  );
};

export default Users;
