import React, { useEffect, useMemo, useState } from 'react';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { useEndpoint, useEndpointFilters, useEndpoints } from '@hooks';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Grid,
  InputAdornment,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  TextField,
  Typography,
} from '@mui/material';

import { PaginationControls } from '@components/common/PaginationControls';
import { formatRelativeDuration, paginate } from '@utils/formatters';
import { getMonoBody2 } from '@utils/typography';
import { ROUTES } from '@constants/app';
import { ENDPOINTS_PAGE, ERR_LOAD_ENDPOINTS } from '@constants/messages';
import {
  AddIcon,
  EndPointIcon,
  ErrorIcon,
  FilterListIcon,
  SearchIcon,
  SuccessIcon,
} from '@theme/icons';

import AddEndPointDialog from '../components/AddEndPointDialog';
import EndPointDetails from '../components/EndPointDetails';

type OrderBy = 'name' | 'epEui' | 'status' | 'lastSeen';

const EndPoints: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams();

  // Filter context for persistent search/sort/pagination state
  const { filters, setSearch, setSort, pagination, setPage, setPageSize } = useEndpointFilters();

  // Local UI state (not persisted)
  const [selectedEndPoint, setSelectedEndPoint] = useState<string | null>(null);
  const [showDetails, setShowDetails] = useState(false);
  const [addDialogOpen, setAddDialogOpen] = useState(false);

  // Convert filter state to API-compatible filters (null -> undefined)
  const apiFilters = {
    ...filters,
    bidirectional: filters.bidirectional ?? undefined,
  };

  // React Query hooks for data fetching (passes filters for TypeScript alignment)
  const { data: endPoints = [], isLoading: loading, isError, error } = useEndpoints(apiFilters);

  // Fetch full endpoint data when viewing details
  const { data: fullEndpointData, isLoading: detailsLoading } = useEndpoint(selectedEndPoint || '');

  // Handle URL parameter for direct navigation to endpoint details
  useEffect(() => {
    if (id && endPoints.length > 0) {
      // id is the EUI from the URL
      const endPoint = endPoints.find((ep) => ep.epEui === id);
      if (endPoint) {
        setSelectedEndPoint(endPoint.epEui);
        setShowDetails(true);
      }
    } else if (!id) {
      setShowDetails(false);
      setSelectedEndPoint(null);
    }
  }, [id, endPoints]);

  const filteredEndPoints = useMemo(() => {
    const searchLower = filters.search.toLowerCase();
    return endPoints.filter(
      (ep) =>
        ep.name?.toLowerCase().includes(searchLower) || ep.epEui.toLowerCase().includes(searchLower)
    );
  }, [endPoints, filters.search]);

  const sortedEndPoints = useMemo(() => {
    const orderBy = filters.sort.field as OrderBy;
    const orderDirection = filters.sort.direction;
    return [...filteredEndPoints].sort((a, b) => {
      const aValue = a[orderBy] || '';
      const bValue = b[orderBy] || '';

      if (orderDirection === 'asc') {
        return aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
      } else {
        return aValue > bValue ? -1 : aValue < bValue ? 1 : 0;
      }
    });
  }, [filteredEndPoints, filters.sort.field, filters.sort.direction]);

  // Paginate the sorted list for display
  const paginatedEndpoints = useMemo(
    () => paginate(sortedEndPoints, pagination.page, pagination.pageSize),
    [sortedEndPoints, pagination.page, pagination.pageSize]
  );

  const handleSort = (property: OrderBy) => {
    const isAsc = filters.sort.field === property && filters.sort.direction === 'asc';
    setSort({ field: property, direction: isAsc ? 'desc' : 'asc' });
  };

  const handleAddDialogClose = () => {
    setAddDialogOpen(false);
  };

  const handleEndPointClick = (id: string) => {
    const endPoint = endPoints.find((ep) => ep.epEui === id);
    if (endPoint) {
      navigate(generatePath(ROUTES.ENDPOINT_DETAIL, { id: endPoint.epEui }));
    }
  };

  const selectedEndPointData = endPoints.find((ep) => ep.epEui === selectedEndPoint);

  // Calculate statistics
  const activeCount = endPoints.filter((ep) => ep.status === 'active').length;

  const handleBackToList = () => {
    navigate(ROUTES.ENDPOINTS);
  };

  // If showing details, render only the details view
  if (showDetails && selectedEndPointData) {
    // Use full endpoint data if available, otherwise fall back to basic data
    const endpointToDisplay = fullEndpointData || selectedEndPointData;

    if (detailsLoading) {
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

    return (
      <Box sx={{ pt: 4 }}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
          <Typography variant="h4" component="h1">
            {ENDPOINTS_PAGE.DETAILS_TITLE}
          </Typography>
          <Button variant="outlined" onClick={handleBackToList}>
            {ENDPOINTS_PAGE.BACK_TO_LIST}
          </Button>
        </Box>
        <EndPointDetails
          endPoint={endpointToDisplay}
          onDelete={() => {
            // Navigate back to list - cache invalidation handles refresh
            handleBackToList();
          }}
        />
      </Box>
    );
  }

  return (
    <Box data-testid="endpoints-page" sx={{ p: 3, pt: 4 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Typography variant="h4" component="h1">
          {ENDPOINTS_PAGE.TITLE}
        </Typography>
        <Button variant="contained" startIcon={<AddIcon />} onClick={() => setAddDialogOpen(true)}>
          {ENDPOINTS_PAGE.ADD_ENDPOINT}
        </Button>
      </Box>

      {/* Statistics Cards */}
      <Grid container spacing={3} mb={3}>
        <Grid size={{ xs: 12, sm: 6, md: 4 }}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <EndPointIcon sx={{ fontSize: 40, color: 'primary.main', mr: 2 }} />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {ENDPOINTS_PAGE.TOTAL_ENDPOINTS}
                  </Typography>
                  <Typography variant="h4">{endPoints.length}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 4 }}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <SuccessIcon sx={{ fontSize: 40, color: 'success.main', mr: 2 }} />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {ENDPOINTS_PAGE.ACTIVE}
                  </Typography>
                  <Typography variant="h4">{activeCount}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, sm: 6, md: 4 }}>
          <Card>
            <CardContent>
              <Box display="flex" alignItems="center">
                <ErrorIcon sx={{ fontSize: 40, color: 'error.main', mr: 2 }} />
                <Box>
                  <Typography color="text.secondary" variant="body2">
                    {ENDPOINTS_PAGE.INACTIVE}
                  </Typography>
                  <Typography variant="h4">{endPoints.length - activeCount}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Search and Filter */}
      <Box display="flex" gap={2} mb={3}>
        <TextField
          placeholder={ENDPOINTS_PAGE.SEARCH_PLACEHOLDER}
          value={filters.search}
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
        <Button variant="outlined" startIcon={<FilterListIcon />}>
          {ENDPOINTS_PAGE.FILTERS}
        </Button>
      </Box>

      {/* Loading State */}
      {loading && (
        <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
          <CircularProgress />
        </Box>
      )}

      {/* Error Alert */}
      {isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error instanceof Error ? error.message : ERR_LOAD_ENDPOINTS}
        </Alert>
      )}

      {/* End Points Table */}
      {!loading && !isError && (
        <TableContainer component={Paper} sx={{ overflowX: 'auto' }}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>
                  <TableSortLabel
                    active={filters.sort.field === 'name'}
                    direction={filters.sort.field === 'name' ? filters.sort.direction : 'asc'}
                    onClick={() => handleSort('name')}
                  >
                    {ENDPOINTS_PAGE.COL_NAME}
                  </TableSortLabel>
                </TableCell>
                <TableCell>
                  <TableSortLabel
                    active={filters.sort.field === 'epEui'}
                    direction={filters.sort.field === 'epEui' ? filters.sort.direction : 'asc'}
                    onClick={() => handleSort('epEui')}
                  >
                    {ENDPOINTS_PAGE.COL_EUI}
                  </TableSortLabel>
                </TableCell>
                <TableCell>
                  <TableSortLabel
                    active={filters.sort.field === 'status'}
                    direction={filters.sort.field === 'status' ? filters.sort.direction : 'asc'}
                    onClick={() => handleSort('status')}
                  >
                    {ENDPOINTS_PAGE.COL_STATUS}
                  </TableSortLabel>
                </TableCell>
                <TableCell>
                  <TableSortLabel
                    active={filters.sort.field === 'lastSeen'}
                    direction={filters.sort.field === 'lastSeen' ? filters.sort.direction : 'asc'}
                    onClick={() => handleSort('lastSeen')}
                  >
                    {ENDPOINTS_PAGE.COL_LAST_SEEN}
                  </TableSortLabel>
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {sortedEndPoints.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} align="center" sx={{ py: 4 }}>
                    <Typography color="text.secondary">
                      {filters.search ? ENDPOINTS_PAGE.NO_MATCH : ENDPOINTS_PAGE.NO_ENDPOINTS}
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                paginatedEndpoints.map((endPoint) => (
                  <TableRow
                    key={endPoint.id}
                    hover
                    onClick={() => handleEndPointClick(endPoint.epEui)}
                    sx={{ cursor: 'pointer' }}
                  >
                    <TableCell>
                      <Typography variant="body2">
                        {endPoint.name || ENDPOINTS_PAGE.UNNAMED_DEVICE}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={(theme) => getMonoBody2(theme)}>
                        {endPoint.epEui}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={endPoint.status}
                        color={endPoint.status === 'active' ? 'success' : 'default'}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Typography
                        variant="body2"
                        color={endPoint.lastSeen ? 'text.secondary' : 'error'}
                      >
                        {formatRelativeDuration(endPoint.lastSeen)}
                      </Typography>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Pagination */}
      {!loading && !isError && (
        <PaginationControls
          page={pagination.page}
          rowsPerPage={pagination.pageSize}
          totalCount={filteredEndPoints.length}
          onPageChange={setPage}
          onRowsPerPageChange={setPageSize}
        />
      )}

      {/* Add End Point Dialog */}
      <AddEndPointDialog open={addDialogOpen} onClose={handleAddDialogClose} />
    </Box>
  );
};

export default EndPoints;
