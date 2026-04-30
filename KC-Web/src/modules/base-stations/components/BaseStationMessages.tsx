import React, { useCallback, useEffect, useState } from 'react';

import type { BaseStationActivityFilter } from '@api-types/api';
import { useBaseStationActivity } from '@hooks';
import {
  Alert,
  Box,
  Button,
  Paper,
  Snackbar,
  TextField,
  Typography,
} from '@mui/material';

import { ActivityTimeline } from '@components/common/ActivityTimeline';
import { PaginationControls } from '@components/common/PaginationControls';
import { apiService } from '@services/api';
import {
  formatDate,
  parseDateInputEU,
} from '@utils/formatters';
import { PAGINATION } from '@constants/app';
import { BASE_STATION_MESSAGES } from '@constants/messages';
import { DownloadIcon } from '@theme/icons';

interface BaseStationMessagesProps {
  bsEui: string;
  basestationName?: string;
}

export const BaseStationMessages: React.FC<BaseStationMessagesProps> = ({
  bsEui,
  basestationName,
}) => {
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState<number>(PAGINATION.DEFAULT_PAGE_SIZE);
  const [filter, setFilter] = useState<BaseStationActivityFilter>({});
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // Local state for date inputs (allows typing before blur validation)
  const [startDateText, setStartDateText] = useState('');
  const [endDateText, setEndDateText] = useState('');
  const [startDateError, setStartDateError] = useState<string | null>(null);
  const [endDateError, setEndDateError] = useState<string | null>(null);

  // Sync local date text from filter (for initial load and clear)
  useEffect(() => {
    setStartDateText(filter.startTime ? formatDate(filter.startTime) : '');
    setEndDateText(filter.endTime ? formatDate(filter.endTime) : '');
    setStartDateError(null);
    setEndDateError(null);
  }, [filter.startTime, filter.endTime]);

  // Track page tokens for cursor-based pagination
  const [pageTokens, setPageTokens] = useState<string[]>(['']); // First page has empty token
  const currentPageToken = pageTokens[page] || '';

  // Use unified activity feed hook
  const { data: activityData, isLoading } = useBaseStationActivity(
    bsEui,
    filter,
    currentPageToken,
    rowsPerPage
  );

  // Update page tokens when we get a new nextPageToken
  const handlePageChange = useCallback(
    (newPage: number) => {
      if (newPage > page && activityData?.nextPageToken) {
        // Going forward - store the next page token
        setPageTokens((prev) => {
          const updated = [...prev];
          updated[newPage] = activityData.nextPageToken!;
          return updated;
        });
      }
      setPage(newPage);
    },
    [page, activityData?.nextPageToken]
  );

  const handleExport = async (format: 'csv' | 'json') => {
    try {
      const blob = await apiService.exportBaseStationMessages(bsEui, filter, format);
      const text = await blob.text();

      // CSV: detect header-only export
      if (format === 'csv' && text.trim() === BASE_STATION_MESSAGES.EXPORT_CSV_HEADER) {
        setErrorMessage(BASE_STATION_MESSAGES.ERR_EXPORT_NO_DATA);
        return;
      }
      // JSON: detect empty array
      if (format === 'json' && text.trim() === BASE_STATION_MESSAGES.EXPORT_JSON_EMPTY) {
        setErrorMessage(BASE_STATION_MESSAGES.ERR_EXPORT_NO_DATA);
        return;
      }

      // Re-create blob from text for download
      const downloadBlob = new Blob([text], { type: blob.type });
      const url = URL.createObjectURL(downloadBlob);
      const a = document.createElement('a');
      a.href = url;
      const timestamp = new Date().toISOString();
      a.download = `${BASE_STATION_MESSAGES.EXPORT_FILENAME_PREFIX}${bsEui}${BASE_STATION_MESSAGES.EXPORT_FILENAME_SUFFIX}${timestamp}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch {
      setErrorMessage(BASE_STATION_MESSAGES.ERR_EXPORT_FAILED);
    }
  };

  const handleClearFilters = () => {
    setFilter({});
    setPage(0);
    setPageTokens(['']);
    // Reset local date input state
    setStartDateText('');
    setEndDateText('');
    setStartDateError(null);
    setEndDateError(null);
  };

  // Apply filter changes
  const handleFilterChange = (newFilter: BaseStationActivityFilter) => {
    setFilter(newFilter);
    setPage(0);
    setPageTokens(['']);
  };

  const totalCount = activityData?.totalCount ?? 0;

  return (
    <Box>
      <Paper sx={{ p: 2, mb: 2 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6">
            {BASE_STATION_MESSAGES.TITLE_PREFIX}
            {basestationName || `${BASE_STATION_MESSAGES.TITLE_FALLBACK_PREFIX}${bsEui}`}
          </Typography>
          <Box>
            <Button startIcon={<DownloadIcon />} onClick={() => handleExport('csv')} sx={{ ml: 1 }}>
              {BASE_STATION_MESSAGES.EXPORT_CSV}
            </Button>
            <Button
              startIcon={<DownloadIcon />}
              onClick={() => handleExport('json')}
              sx={{ ml: 1 }}
            >
              {BASE_STATION_MESSAGES.EXPORT_JSON}
            </Button>
          </Box>
        </Box>

        {/* Filters */}
        <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
          <TextField
            label={BASE_STATION_MESSAGES.LABEL_START_DATE}
            type="text"
            inputMode="numeric"
            placeholder={BASE_STATION_MESSAGES.DATE_PLACEHOLDER_EU}
            value={startDateText}
            onChange={(e) => {
              setStartDateText(e.target.value);
              if (startDateError) setStartDateError(null);
            }}
            onBlur={() => {
              if (!startDateText) {
                handleFilterChange({ ...filter, startTime: undefined });
                return;
              }
              const isoDate = parseDateInputEU(startDateText);
              if (!isoDate) {
                setStartDateError(BASE_STATION_MESSAGES.ERR_DATE_INVALID);
                return;
              }
              handleFilterChange({ ...filter, startTime: `${isoDate}T00:00:00Z` });
            }}
            error={!!startDateError}
            helperText={startDateError || BASE_STATION_MESSAGES.DATE_HELPER_EU}
            size="small"
            slotProps={{ inputLabel: { shrink: true } }}
            sx={{ minWidth: 150 }}
          />

          <TextField
            label={BASE_STATION_MESSAGES.LABEL_END_DATE}
            type="text"
            inputMode="numeric"
            placeholder={BASE_STATION_MESSAGES.DATE_PLACEHOLDER_EU}
            value={endDateText}
            onChange={(e) => {
              setEndDateText(e.target.value);
              if (endDateError) setEndDateError(null);
            }}
            onBlur={() => {
              if (!endDateText) {
                handleFilterChange({ ...filter, endTime: undefined });
                return;
              }
              const isoDate = parseDateInputEU(endDateText);
              if (!isoDate) {
                setEndDateError(BASE_STATION_MESSAGES.ERR_DATE_INVALID);
                return;
              }
              handleFilterChange({ ...filter, endTime: `${isoDate}T23:59:59Z` });
            }}
            error={!!endDateError}
            helperText={endDateError || BASE_STATION_MESSAGES.DATE_HELPER_EU}
            size="small"
            slotProps={{ inputLabel: { shrink: true } }}
            sx={{ minWidth: 150 }}
          />

          <Button variant="outlined" onClick={handleClearFilters}>
            {BASE_STATION_MESSAGES.CLEAR_FILTERS}
          </Button>
        </Box>

        {/* Activity timeline */}
        <ActivityTimeline
          variant="detailed"
          items={activityData?.items ?? []}
          loading={isLoading}
          emptyMessage={BASE_STATION_MESSAGES.NO_ACTIVITY}
        />

        {/* Pagination - show even when totalCount is 0 but data loaded */}
        <PaginationControls
          page={page}
          rowsPerPage={rowsPerPage}
          totalCount={totalCount}
          onPageChange={handlePageChange}
          onRowsPerPageChange={(newSize) => {
            setRowsPerPage(newSize);
            setPage(0);
            setPageTokens(['']);
          }}
        />
      </Paper>

      {/* Error Snackbar */}
      <Snackbar
        open={!!errorMessage}
        autoHideDuration={6000}
        onClose={() => setErrorMessage(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity="error" onClose={() => setErrorMessage(null)}>
          {errorMessage}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default BaseStationMessages;
