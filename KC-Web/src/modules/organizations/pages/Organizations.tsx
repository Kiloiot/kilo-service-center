/**
 * Organizations Page
 *
 * Organization management list page with runtime admin guard for deep link protection.
 */

import React, { useMemo, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";

import type { OrganizationUI } from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Typography,
} from "@mui/material";

import SearchField from "@components/common/SearchField";
import { StatCard, StatCardRow } from "@components/common/StatCard";
import { useSession } from "@contexts/SessionContext";
import { useCapabilities } from "@hooks/useCapabilities";
import { useOrganizations } from "@hooks/useOrganizations";
import { getErrorMessage } from "@utils/error-message";
import { toggleSortDirection } from "@utils/list-query";
import type { SortDirection } from "@constants/app";
import { ORG_STATE, PAGINATION, ROUTES, SORT_DIRECTION } from "@constants/app";
import {
  ERR_LOAD_ORGANIZATIONS,
  ORGANIZATIONS_PAGE,
} from "@constants/messages";
import { organizationDetailPath } from "@router/paths";
import {
  AddIcon,
  ArchiveIcon,
  BusinessIcon,
  ErrorIcon,
  SuccessIcon,
} from "@theme/icons";
import { componentSpacing } from "@theme/index";

import AddOrganizationDialog from "../components/AddOrganizationDialog";
import OrganizationsTable from "../components/OrganizationsTable";

type OrderBy = "name" | "state" | "createdAt";
const Organizations: React.FC = () => {
  const navigate = useNavigate();
  const { isHydrated } = useSession();
  const { isServerAdmin: isAdmin } = useCapabilities();

  // Local UI state
  const [search, setSearch] = useState("");
  const [orderBy, setOrderBy] = useState<OrderBy>("name");
  const [orderDirection, setOrderDirection] = useState<SortDirection>(
    SORT_DIRECTION.ASC,
  );
  const [dialogOpen, setDialogOpen] = useState(false);

  // React Query hook for data fetching
  const { data, isLoading, isError, error } = useOrganizations(
    PAGINATION.ADMIN_LIST_PAGE_SIZE,
    0,
    undefined,
    {
      enabled: isHydrated && isAdmin,
    },
  );

  // Memoize organizations array
  const organizations = useMemo(
    () => data?.organizations ?? [],
    [data?.organizations],
  );

  // Filter organizations by search
  const filteredOrganizations = useMemo(() => {
    const searchLower = search.toLowerCase();
    return organizations.filter(
      (org) =>
        org.name.toLowerCase().includes(searchLower) ||
        org.description?.toLowerCase().includes(searchLower),
    );
  }, [organizations, search]);

  // Sort organizations
  const sortedOrganizations = useMemo(() => {
    return [...filteredOrganizations].sort((a, b) => {
      let aValue: string = "";
      let bValue: string = "";

      if (orderBy === "name") {
        aValue = a.name;
        bValue = b.name;
      } else if (orderBy === "state") {
        aValue = a.state;
        bValue = b.state;
      } else if (orderBy === "createdAt") {
        aValue = a.createdAt;
        bValue = b.createdAt;
      }

      if (orderDirection === SORT_DIRECTION.ASC) {
        return aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
      }
      return aValue > bValue ? -1 : aValue < bValue ? 1 : 0;
    });
  }, [filteredOrganizations, orderBy, orderDirection]);

  const handleSort = (field: OrderBy) => {
    if (orderBy === field) {
      setOrderDirection(toggleSortDirection(orderDirection));
    } else {
      setOrderBy(field);
      setOrderDirection(SORT_DIRECTION.ASC);
    }
  };

  const handleRowClick = (id: string) => {
    navigate(organizationDetailPath(id));
  };

  // Calculate statistics
  const activeCount = organizations.filter(
    (o: OrganizationUI) => o.state === ORG_STATE.ACTIVE,
  ).length;
  const suspendedCount = organizations.filter(
    (o: OrganizationUI) => o.state === ORG_STATE.SUSPENDED,
  ).length;
  const archivedCount = organizations.filter(
    (o: OrganizationUI) => o.state === ORG_STATE.ARCHIVED,
  ).length;

  if (!isHydrated) {
    return null;
  }

  // Runtime admin guard (deep link protection) - placed after hooks per React rules
  if (!isAdmin) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  return (
    <Box data-testid="organizations-page" sx={{ p: 3, pt: 4 }}>
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Typography variant="h4" component="h1">
          {ORGANIZATIONS_PAGE.TITLE}
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setDialogOpen(true)}
        >
          {ORGANIZATIONS_PAGE.ADD_ORGANIZATION}
        </Button>
      </Box>

      <StatCardRow>
        <StatCard
          label={ORGANIZATIONS_PAGE.TOTAL_ORGANIZATIONS}
          value={organizations.length}
          icon={<BusinessIcon />}
          color="primary"
        />
        <StatCard
          label={ORGANIZATIONS_PAGE.ACTIVE_ORGANIZATIONS}
          value={activeCount}
          icon={<SuccessIcon />}
          color="success"
        />
        <StatCard
          label={ORGANIZATIONS_PAGE.SUSPENDED_ORGANIZATIONS}
          value={suspendedCount}
          icon={<ErrorIcon />}
          color="warning"
        />
        <StatCard
          label={ORGANIZATIONS_PAGE.ARCHIVED_ORGANIZATIONS}
          value={archivedCount}
          icon={<ArchiveIcon />}
          color="secondary"
        />
      </StatCardRow>

      {/* Search */}
      <Box display="flex" gap={2} mb={3}>
        <SearchField
          placeholder={ORGANIZATIONS_PAGE.SEARCH_PLACEHOLDER}
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
          {getErrorMessage(error, ERR_LOAD_ORGANIZATIONS)}
        </Alert>
      )}

      {/* Organizations Table */}
      {!isLoading && !isError && (
        <OrganizationsTable
          organizations={sortedOrganizations}
          orderBy={orderBy}
          orderDirection={orderDirection}
          onSort={handleSort}
          onRowClick={handleRowClick}
          emptyMessage={
            search
              ? ORGANIZATIONS_PAGE.NO_MATCH
              : ORGANIZATIONS_PAGE.NO_ORGANIZATIONS
          }
        />
      )}

      {/* Add Organization Dialog */}
      <AddOrganizationDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
      />
    </Box>
  );
};

export default Organizations;
