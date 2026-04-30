import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { useBaseStationFilters, useBaseStations } from '@hooks';
import {
  Alert,
  alpha,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Grid,
  IconButton,
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
  useTheme,
} from '@mui/material';

import { PaginationControls } from '@components/common/PaginationControls';
import { calculateDaysUntilExpiry, formatDate, formatDateTime, paginate } from '@utils/formatters';
import { getMonoBody2 } from '@utils/typography';
import { BASE_STATIONS_PAGE, ERR_LOAD_BASE_STATIONS } from '@constants/messages';
import {
  CheckCircleIcon,
  ErrorIcon,
  RefreshIcon,
  SearchIcon,
  SecurityIcon,
  WarningIcon,
} from '@theme/icons';

import BaseStationCommissioningDialog from '../components/BaseStationCommissioningDialog';
import BaseStationDetails from '../components/BaseStationDetails';

type SortField = 'name' | 'eui' | 'status' | 'connectionType' | 'lastSeen';

const BaseStations: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams();
  const theme = useTheme();

  // Filter context for persistent search/sort/pagination state
  const { filters, setSearch, setSort, pagination, setPage, setPageSize } = useBaseStationFilters();

  // Local UI state (not persisted - card click sorting overlays)
  const [selectedBaseStation, setSelectedBaseStation] = useState<string | null>(null);
  const [showDetails, setShowDetails] = useState(false);
  const [commissioningDialogOpen, setCommissioningDialogOpen] = useState(false);
  const [sortByCertExpiry, setSortByCertExpiry] = useState(false);
  const [sortByStatus, setSortByStatus] = useState<'online' | 'offline' | null>(null);

  // React Query hooks for data fetching (passes filters for TypeScript alignment)
  const {
    data: baseStations = [],
    isLoading: loading,
    isError,
    error,
    refetch,
  } = useBaseStations(filters);

  // Handle URL parameter for direct navigation to base station details
  useEffect(() => {
    if (id && baseStations.length > 0) {
      // id is actually the EUI from the URL
      const baseStation = baseStations.find((bs) => bs.eui === id);
      if (baseStation) {
        setSelectedBaseStation(baseStation.id);
        setShowDetails(true);
      }
    } else if (!id) {
      setShowDetails(false);
      setSelectedBaseStation(null);
    }
  }, [id, baseStations]);

  const handleSort = (field: SortField) => {
    const isAsc = filters.sort.field === field && filters.sort.direction === 'asc';
    setSort({ field, direction: isAsc ? 'desc' : 'asc' });
    // Clear card-click sorts when using table sort
    setSortByCertExpiry(false);
    setSortByStatus(null);
  };

  const handleExpiringCertificatesClick = () => {
    setSortByCertExpiry(!sortByCertExpiry);
    setSortByStatus(null); // Clear status sort when sorting by cert expiry
  };

  const handleOnlineClick = () => {
    setSortByStatus(sortByStatus === 'online' ? null : 'online');
    setSortByCertExpiry(false); // Clear cert expiry sort
  };

  const handleOfflineClick = () => {
    setSortByStatus(sortByStatus === 'offline' ? null : 'offline');
    setSortByCertExpiry(false); // Clear cert expiry sort
  };

  const sortedBaseStations = useMemo(() => {
    let sorted = [...baseStations];
    const sortField = filters.sort.field as SortField;
    const sortDirection = filters.sort.direction;

    if (sortByCertExpiry) {
      sorted = sorted.sort((a, b) => {
        if (!a.certificateExpiryDate) return 1;
        if (!b.certificateExpiryDate) return 1;
        return (
          new Date(a.certificateExpiryDate).getTime() - new Date(b.certificateExpiryDate).getTime()
        );
      });
    } else if (sortByStatus === 'online') {
      sorted = sorted.sort((a, b) => {
        if (a.status === 'online' && b.status !== 'online') return -1;
        if (a.status !== 'online' && b.status === 'online') return 1;
        return 0;
      });
    } else if (sortByStatus === 'offline') {
      sorted = sorted.sort((a, b) => {
        if (a.status === 'offline' && b.status !== 'offline') return -1;
        if (a.status !== 'offline' && b.status === 'offline') return 1;
        return 0;
      });
    } else {
      sorted = sorted.sort((a, b) => {
        let aValue: string | number | boolean | undefined | null = a[sortField];
        let bValue: string | number | boolean | undefined | null = b[sortField];

        // Handle name field - use EUI if name is not present
        if (sortField === 'name') {
          aValue = a.name || a.eui;
          bValue = b.name || b.eui;
        }

        // Handle null/undefined values
        if (aValue == null) return 1;
        if (bValue == null) return -1;

        // Compare values
        if (aValue < bValue) {
          return sortDirection === 'asc' ? -1 : 1;
        }
        if (aValue > bValue) {
          return sortDirection === 'asc' ? 1 : -1;
        }
        return 0;
      });
    }
    return sorted;
  }, [baseStations, filters.sort.field, filters.sort.direction, sortByCertExpiry, sortByStatus]);

  const filteredBaseStations = sortedBaseStations.filter(
    (bs) =>
      bs.eui?.toLowerCase().includes(filters.search.toLowerCase()) ||
      bs.name?.toLowerCase().includes(filters.search.toLowerCase())
  );

  // Paginate the filtered list for display
  const paginatedStations = useMemo(
    () => paginate(filteredBaseStations, pagination.page, pagination.pageSize),
    [filteredBaseStations, pagination.page, pagination.pageSize]
  );

  const handleBaseStationClick = (id: string) => {
    const baseStation = baseStations.find((bs) => bs.id === id);
    if (baseStation) {
      navigate(`/base-stations/${baseStation.eui}`);
    }
  };

  const handleBackToList = () => {
    navigate('/base-stations');
  };

  const selectedBaseStationData = selectedBaseStation
    ? baseStations.find((bs) => bs.id === selectedBaseStation)
    : null;

  // Calculate statistics
  const totalBaseStations = baseStations.length;
  const onlineBaseStations = baseStations.filter((bs) => bs.status === 'online').length;
  const offlineBaseStations = baseStations.length - onlineBaseStations;

  // Calculate expiring certificates
  const expiringCertificates = baseStations.filter((bs) => {
    const days = calculateDaysUntilExpiry(bs.certificateExpiryDate);
    return days !== null && days < 60; // Less than 60 days
  }).length;

  // Determine icon and color for expiring certificates widget
  const getExpiringCertificatesIcon = () => {
    const criticalCount = baseStations.filter((bs) => {
      const days = calculateDaysUntilExpiry(bs.certificateExpiryDate);
      return days !== null && days < 30; // Less than 30 days
    }).length;

    if (criticalCount > 0) {
      return <ErrorIcon sx={{ fontSize: 40, color: 'error.main' }} />;
    } else if (expiringCertificates > 0) {
      return <WarningIcon sx={{ fontSize: 40, color: 'warning.main' }} />;
    } else {
      return <CheckCircleIcon sx={{ fontSize: 40, color: 'success.main' }} />;
    }
  };

  // If showing details, render only the details view
  if (showDetails && selectedBaseStationData) {
    return (
      <Box sx={{ pt: theme.spacing(4) }}>
        <Box
          display="flex"
          justifyContent="space-between"
          alignItems="center"
          mb={theme.spacing(3)}
        >
          <Typography variant="h4" component="h1">
            {BASE_STATIONS_PAGE.DETAILS_TITLE}
          </Typography>
          <Button variant="outlined" onClick={handleBackToList}>
            {BASE_STATIONS_PAGE.BACK_TO_LIST}
          </Button>
        </Box>
        <BaseStationDetails
          baseStation={selectedBaseStationData}
          onDelete={() => {
            // Navigate back to list - cache invalidation handles refresh
            handleBackToList();
          }}
        />
      </Box>
    );
  }

  // Otherwise render the table view
  return (
    <Box data-testid="base-stations-page" sx={{ p: theme.spacing(3), pt: theme.spacing(4) }}>
      {/* Header */}
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          mb: theme.spacing(3),
        }}
      >
        <Typography variant="h4" component="h1">
          {BASE_STATIONS_PAGE.TITLE}
        </Typography>
        <Box sx={{ display: 'flex', gap: theme.spacing(2) }}>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={() => refetch()}
            disabled={loading}
          >
            {BASE_STATIONS_PAGE.REFRESH}
          </Button>
          <Button
            variant="contained"
            startIcon={<SecurityIcon />}
            onClick={() => setCommissioningDialogOpen(true)}
            sx={{
              background: `linear-gradient(45deg, ${theme.palette.primary.main} 30%, ${theme.palette.primary.light} 90%)`,
              boxShadow: `0 3px 5px 2px ${alpha(theme.palette.primary.main, 0.3)}`,
            }}
          >
            {BASE_STATIONS_PAGE.ADD_BASESTATION}
          </Button>
        </Box>
      </Box>

      {/* Loading and Error States */}
      {loading && (
        <Box display="flex" justifyContent="center" alignItems="center" minHeight="200px">
          <CircularProgress />
        </Box>
      )}

      {isError && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error instanceof Error ? error.message : ERR_LOAD_BASE_STATIONS}
        </Alert>
      )}

      {!loading && !isError && (
        <>
          {/* Status Cards */}
          <Grid container spacing={theme.spacing(3)} sx={{ mb: theme.spacing(4) }}>
            <Grid size={{ xs: 12, sm: 6, md: 3 }}>
              <Card>
                <CardContent>
                  <Box display="flex" alignItems="center" justifyContent="space-between">
                    <Box>
                      <Typography color="text.secondary" variant="body2">
                        {BASE_STATIONS_PAGE.TOTAL_BASE_STATIONS}
                      </Typography>
                      <Typography variant="h4">{totalBaseStations}</Typography>
                    </Box>
                  </Box>
                </CardContent>
              </Card>
            </Grid>
            <Grid size={{ xs: 12, sm: 6, md: 3 }}>
              <Card sx={{ cursor: 'pointer' }} onClick={handleOnlineClick}>
                <CardContent>
                  <Box display="flex" alignItems="center" justifyContent="space-between">
                    <Box>
                      <Typography color="text.secondary" variant="body2">
                        {BASE_STATIONS_PAGE.ONLINE}
                      </Typography>
                      <Typography variant="h4" color="success.main">
                        {onlineBaseStations}
                      </Typography>
                      {sortByStatus === 'online' && (
                        <Typography variant="caption" color="primary">
                          {BASE_STATIONS_PAGE.SORTED_BY_ONLINE}
                        </Typography>
                      )}
                    </Box>
                    <CheckCircleIcon
                      sx={{ fontSize: theme.typography.h1.fontSize, color: 'success.main' }}
                    />
                  </Box>
                </CardContent>
              </Card>
            </Grid>
            <Grid size={{ xs: 12, sm: 6, md: 3 }}>
              <Card sx={{ cursor: 'pointer' }} onClick={handleOfflineClick}>
                <CardContent>
                  <Box display="flex" alignItems="center" justifyContent="space-between">
                    <Box>
                      <Typography color="text.secondary" variant="body2">
                        {BASE_STATIONS_PAGE.OFFLINE}
                      </Typography>
                      <Typography variant="h4" color="text.secondary">
                        {offlineBaseStations}
                      </Typography>
                      {sortByStatus === 'offline' && (
                        <Typography variant="caption" color="primary">
                          {BASE_STATIONS_PAGE.SORTED_BY_OFFLINE}
                        </Typography>
                      )}
                    </Box>
                    <ErrorIcon
                      sx={{ fontSize: theme.typography.h1.fontSize, color: 'text.secondary' }}
                    />
                  </Box>
                </CardContent>
              </Card>
            </Grid>
            <Grid size={{ xs: 12, sm: 6, md: 3 }}>
              <Card sx={{ cursor: 'pointer' }} onClick={handleExpiringCertificatesClick}>
                <CardContent>
                  <Box display="flex" alignItems="center" justifyContent="space-between">
                    <Box>
                      <Typography color="text.secondary" variant="body2">
                        {BASE_STATIONS_PAGE.EXPIRING_CERTIFICATES}
                      </Typography>
                      <Typography variant="h4">{expiringCertificates}</Typography>
                      {sortByCertExpiry && (
                        <Typography variant="caption" color="primary">
                          {BASE_STATIONS_PAGE.SORTED_BY_EXPIRY}
                        </Typography>
                      )}
                    </Box>
                    {getExpiringCertificatesIcon()}
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          </Grid>

          <Box mb={3} display="flex" gap={2}>
            <TextField
              fullWidth
              variant="outlined"
              placeholder={BASE_STATIONS_PAGE.SEARCH_PLACEHOLDER}
              value={filters.search}
              onChange={(e) => setSearch(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon />
                  </InputAdornment>
                ),
              }}
            />
            <IconButton>
              <SearchIcon />
            </IconButton>
          </Box>

          <TableContainer component={Paper} sx={{ mb: 3, overflowX: 'auto' }}>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>
                    <TableSortLabel
                      active={filters.sort.field === 'name' && !sortByCertExpiry && !sortByStatus}
                      direction={filters.sort.field === 'name' ? filters.sort.direction : 'asc'}
                      onClick={() => handleSort('name')}
                    >
                      {BASE_STATIONS_PAGE.COL_NAME}
                    </TableSortLabel>
                  </TableCell>
                  <TableCell>
                    <TableSortLabel
                      active={filters.sort.field === 'eui' && !sortByCertExpiry && !sortByStatus}
                      direction={filters.sort.field === 'eui' ? filters.sort.direction : 'asc'}
                      onClick={() => handleSort('eui')}
                    >
                      {BASE_STATIONS_PAGE.COL_EUI}
                    </TableSortLabel>
                  </TableCell>
                  <TableCell>
                    <TableSortLabel
                      active={
                        filters.sort.field === 'connectionType' &&
                        !sortByCertExpiry &&
                        !sortByStatus
                      }
                      direction={
                        filters.sort.field === 'connectionType' ? filters.sort.direction : 'asc'
                      }
                      onClick={() => handleSort('connectionType')}
                    >
                      {BASE_STATIONS_PAGE.COL_CONNECTION}
                    </TableSortLabel>
                  </TableCell>
                  <TableCell>
                    <TableSortLabel
                      active={filters.sort.field === 'status' && !sortByCertExpiry && !sortByStatus}
                      direction={filters.sort.field === 'status' ? filters.sort.direction : 'asc'}
                      onClick={() => handleSort('status')}
                    >
                      {BASE_STATIONS_PAGE.COL_STATUS}
                    </TableSortLabel>
                  </TableCell>
                  <TableCell>{BASE_STATIONS_PAGE.COL_DATE_ADDED}</TableCell>
                  <TableCell>
                    <TableSortLabel
                      active={
                        filters.sort.field === 'lastSeen' && !sortByCertExpiry && !sortByStatus
                      }
                      direction={filters.sort.field === 'lastSeen' ? filters.sort.direction : 'asc'}
                      onClick={() => handleSort('lastSeen')}
                    >
                      {BASE_STATIONS_PAGE.COL_LAST_SEEN}
                    </TableSortLabel>
                  </TableCell>
                  <TableCell>{BASE_STATIONS_PAGE.COL_CERTIFICATE_EXPIRY}</TableCell>
                  <TableCell align="right">{BASE_STATIONS_PAGE.COL_ACTIONS}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {paginatedStations.map((baseStation) => {
                  const daysUntilExpiry = calculateDaysUntilExpiry(
                    baseStation.certificateExpiryDate
                  );
                  const isExpiringSoon = daysUntilExpiry !== null && daysUntilExpiry <= 30;
                  const isExpired = daysUntilExpiry !== null && daysUntilExpiry < 0;

                  return (
                    <TableRow
                      key={baseStation.id}
                      hover
                      onClick={() => handleBaseStationClick(baseStation.id)}
                      sx={{
                        cursor: 'pointer',
                        backgroundColor:
                          selectedBaseStation === baseStation.id ? 'action.selected' : 'inherit',
                      }}
                    >
                      <TableCell>{baseStation.name || baseStation.eui}</TableCell>
                      <TableCell>
                        <Typography variant="body2" sx={(theme) => getMonoBody2(theme)}>
                          {baseStation.eui}
                        </Typography>
                      </TableCell>
                      <TableCell>{baseStation.connectionType}</TableCell>
                      <TableCell>
                        <Chip
                          label={baseStation.status}
                          color={baseStation.status === 'online' ? 'success' : 'default'}
                          size="small"
                        />
                      </TableCell>
                      <TableCell>
                        {baseStation.createdAt ? formatDate(baseStation.createdAt) : '-'}
                      </TableCell>
                      <TableCell>{formatDateTime(baseStation.lastSeen)}</TableCell>
                      <TableCell>
                        {baseStation.certificateExpiryDate ? (
                          <Box>
                            <Typography
                              variant="body2"
                              sx={{
                                color: isExpired
                                  ? 'error.main'
                                  : isExpiringSoon
                                    ? 'warning.main'
                                    : 'text.primary',
                                fontWeight: isExpiringSoon || isExpired ? 'bold' : 'normal',
                              }}
                            >
                              {formatDate(baseStation.certificateExpiryDate)}
                            </Typography>
                            <Typography
                              variant="caption"
                              sx={{
                                color: isExpired
                                  ? 'error.main'
                                  : isExpiringSoon
                                    ? 'warning.main'
                                    : 'text.secondary',
                              }}
                            >
                              {isExpired
                                ? BASE_STATIONS_PAGE.EXPIRED
                                : `${daysUntilExpiry} ${BASE_STATIONS_PAGE.DAYS}`}
                            </Typography>
                          </Box>
                        ) : (
                          <Typography variant="body2" color="text.secondary">
                            -
                          </Typography>
                        )}
                      </TableCell>
                      <TableCell align="right">
                        <IconButton
                          size="small"
                          onClick={(e) => {
                            e.stopPropagation();
                            // Handle actions
                          }}
                        >
                          <SearchIcon />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>

          <PaginationControls
            page={pagination.page}
            rowsPerPage={pagination.pageSize}
            totalCount={filteredBaseStations.length}
            onPageChange={setPage}
            onRowsPerPageChange={setPageSize}
          />

          <BaseStationCommissioningDialog
            open={commissioningDialogOpen}
            onClose={() => setCommissioningDialogOpen(false)}
          />
        </>
      )}
    </Box>
  );
};

export default BaseStations;
