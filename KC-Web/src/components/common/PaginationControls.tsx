import React from 'react';

import { Box, FormControl, MenuItem, Pagination, Select, Typography } from '@mui/material';

import { PAGINATION } from '@constants/app';
import { PAGINATION_LABELS } from '@constants/messages';
import { componentSpacing } from '@theme/index';

interface PaginationControlsProps {
  page: number; // 0-based
  rowsPerPage: number;
  totalCount: number;
  pageSizeOptions?: readonly number[];
  onPageChange: (newPage: number) => void; // 0-based output
  onRowsPerPageChange: (newPageSize: number) => void;
}

export const PaginationControls: React.FC<PaginationControlsProps> = ({
  page,
  rowsPerPage,
  totalCount,
  pageSizeOptions = PAGINATION.PAGE_SIZE_OPTIONS,
  onPageChange,
  onRowsPerPageChange,
}) => {
  // Always render pagination controls, but disable when no data
  const hasData = totalCount > 0;
  const pageCount = hasData ? Math.ceil(totalCount / rowsPerPage) : 1;
  const startItem = hasData ? page * rowsPerPage + 1 : 0;
  const endItem = hasData ? Math.min((page + 1) * rowsPerPage, totalCount) : 0;

  return (
    <Box
      sx={{
        mt: componentSpacing.pagination.containerMt,
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        flexWrap: 'wrap',
        gap: componentSpacing.pagination.containerGap,
      }}
    >
      {/* Left: Rows per page */}
      <Box display="flex" alignItems="center" gap={componentSpacing.pagination.controlGap}>
        <Typography variant="body2" color="text.secondary">
          {PAGINATION_LABELS.ROWS_PER_PAGE}
        </Typography>
        <FormControl size="small" sx={{ minWidth: componentSpacing.pagination.selectMinWidth }}>
          <Select
            value={rowsPerPage}
            onChange={(e) => onRowsPerPageChange(Number(e.target.value))}
            size="small"
          >
            {pageSizeOptions.map((option) => (
              <MenuItem key={option} value={option}>
                {option}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>

      {/* Center: Page numbers */}
      <Pagination
        count={pageCount}
        page={page + 1}
        onChange={(_, newPage) => onPageChange(newPage - 1)}
        color="primary"
        showFirstButton
        showLastButton
        disabled={!hasData}
      />

      {/* Right: Item count */}
      <Typography variant="body2" color="text.secondary">
        {hasData
          ? `${startItem}–${endItem} ${PAGINATION_LABELS.OF} ${totalCount}`
          : `0 ${PAGINATION_LABELS.OF} 0`}
      </Typography>
    </Box>
  );
};

export default PaginationControls;
