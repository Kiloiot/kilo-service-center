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
  Card,
  CardContent,
  CircularProgress,
  Grid,
  InputAdornment,
  TextField,
  Typography,
} from "@mui/material";

import { useSession } from "@contexts/SessionContext";
import { useOrganizations } from "@hooks/useOrganizations";
import { ORG_STATE, ROUTES } from "@constants/app";
import {
  ERR_LOAD_ORGANIZATIONS,
  ORGANIZATIONS_PAGE,
} from "@constants/messages";
import {
  AddIcon,
  ArchiveIcon,
  BusinessIcon,
  ErrorIcon,
  SearchIcon,
  SuccessIcon,
} from "@theme/icons";

import AddOrganizationDialog from "../components/AddOrganizationDialog";
import OrganizationsTable from "../components/OrganizationsTable";

type OrderBy = "name" | "state" | "createdAt";
type OrderDirection = "asc" | "desc";

const Organizations: React.FC = () => {
  const navigate = useNavigate();
  const { isAdmin, isHydrated } = useSession();

  // Local UI state
  const [search, setSearch] = useState("");
  const [orderBy, setOrderBy] = useState<OrderBy>("name");
  const [orderDirection, setOrderDirection] = useState<OrderDirection>("asc");
  const [dialogOpen, setDialogOpen] = useState(false);

  // React Query hook for data fetching
  const { data, isLoading, isError, error } = useOrganizations(
    50,
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

      if (orderDirection === "asc") {
        return aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
      }
      return aValue > bValue ? -1 : aValue < bValue ? 1 : 0;
    });
  }, [filteredOrganizations, orderBy, orderDirection]);

  const handleSort = (field: OrderBy) => {
    if (orderBy === field) {
      setOrderDirection(orderDirection === "asc" ? "desc" : "asc");
    } else {
      setOrderBy(field);
      setOrderDirection("asc");
    }
  };

  const handleRowClick = (id: string) => {
    navigate(`${ROUTES.ORGANIZATIONS}/${id}`);
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

      {/* Statistics Cards */}
      <Grid container spacing={3} mb={3}>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <BusinessIcon
                  sx={{ fontSize: 40, color: "primary.main", mr: 2 }}
                />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {ORGANIZATIONS_PAGE.TOTAL_ORGANIZATIONS}
                  </Typography>
                  <Typography variant="h4">{organizations.length}</Typography>
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
                    {ORGANIZATIONS_PAGE.ACTIVE_ORGANIZATIONS}
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
                <ErrorIcon
                  sx={{ fontSize: 40, color: "warning.main", mr: 2 }}
                />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {ORGANIZATIONS_PAGE.SUSPENDED_ORGANIZATIONS}
                  </Typography>
                  <Typography variant="h4">{suspendedCount}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 3 }}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <ArchiveIcon
                  sx={{ fontSize: 40, color: "text.secondary", mr: 2 }}
                />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {ORGANIZATIONS_PAGE.ARCHIVED_ORGANIZATIONS}
                  </Typography>
                  <Typography variant="h4">{archivedCount}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Search */}
      <Box display="flex" gap={2} mb={3}>
        <TextField
          placeholder={ORGANIZATIONS_PAGE.SEARCH_PLACEHOLDER}
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
          {error instanceof Error ? error.message : ERR_LOAD_ORGANIZATIONS}
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
