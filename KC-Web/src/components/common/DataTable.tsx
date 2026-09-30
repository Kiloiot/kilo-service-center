/**
 * Server-paged table: column definitions render the cells, an optional
 * details renderer makes rows expandable, and the pagination controls page
 * through the server's total.
 */

import { Fragment, type ReactNode, useState } from "react";

import { isApiError } from "@api-types/api";
import {
  Box,
  CircularProgress,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from "@mui/material";
import { ErrorState } from "@ui";

import NoAccessPanel from "@components/common/NoAccessPanel";
import { PaginationControls } from "@components/common/PaginationControls";
import { DATA_TABLE_LAYOUT } from "@constants/app";
import { DATA_TABLE } from "@constants/messages";
import { ExpandLess, ExpandMore } from "@theme/icons";
import { componentSpacing } from "@theme/index";

export interface DataTableColumn<T> {
  /** Stable column id; also names the scope key that hides the column. */
  id: string;
  header: string;
  align?: "left" | "right" | "center";
  render: (row: T) => ReactNode;
}

export interface DataTablePaging {
  page: number;
  pageSize: number;
  /** The last page a forward-only cursor can open; pages past it are disabled. */
  lastReachablePage?: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
}

interface DataTableProps<T> {
  columns: readonly DataTableColumn<T>[];
  rows: readonly T[];
  rowKey: (row: T) => string;
  emptyMessage: string;
  isLoading?: boolean;
  error?: Error | null;
  /** Server paging; a table of rows already in hand shows no pagination. */
  paging?: DataTablePaging;
  totalCount?: number;
  renderDetails?: (row: T) => ReactNode;
  /** Makes each row a button, e.g. to open its detail. */
  onRowClick?: (row: T) => void;
}

function ExpandToggle({
  open,
  onToggle,
}: {
  open: boolean;
  onToggle: () => void;
}) {
  const label = open ? DATA_TABLE.HIDE_DETAILS : DATA_TABLE.SHOW_DETAILS;
  return (
    <Tooltip title={label}>
      <IconButton size="small" aria-label={label} onClick={onToggle}>
        {open ? <ExpandLess /> : <ExpandMore />}
      </IconButton>
    </Tooltip>
  );
}

function TableState({
  isLoading,
  error,
  isEmpty,
  emptyMessage,
}: {
  isLoading: boolean;
  error: Error | null;
  isEmpty: boolean;
  emptyMessage: string;
}) {
  if (isApiError(error) && error.isForbidden()) {
    return <NoAccessPanel />;
  }
  if (error) {
    return (
      <ErrorState
        title={DATA_TABLE.LOAD_FAILED}
        error={error}
        minHeight={DATA_TABLE_LAYOUT.ERROR_MIN_HEIGHT}
      />
    );
  }
  if (isLoading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        p={DATA_TABLE_LAYOUT.STATE_PADDING}
      >
        <CircularProgress size={componentSpacing.spinner.section} />
      </Box>
    );
  }
  if (isEmpty) {
    return (
      <Typography
        variant="body2"
        color="text.secondary"
        sx={{ p: DATA_TABLE_LAYOUT.STATE_PADDING }}
      >
        {emptyMessage}
      </Typography>
    );
  }
  return null;
}

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  isLoading = false,
  error = null,
  emptyMessage,
  paging,
  totalCount = rows.length,
  renderDetails,
  onRowClick,
}: DataTableProps<T>) {
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());
  const expandable = renderDetails !== undefined;
  const columnCount = columns.length + (expandable ? 1 : 0);

  const toggle = (key: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });

  return (
    <Box>
      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              {expandable && <TableCell padding="checkbox" />}
              {columns.map((column) => (
                <TableCell key={column.id} align={column.align}>
                  {column.header}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((row) => {
              const key = rowKey(row);
              const open = expanded.has(key);
              return (
                <Fragment key={key}>
                  <TableRow
                    hover
                    onClick={onRowClick ? () => onRowClick(row) : undefined}
                    sx={onRowClick ? { cursor: "pointer" } : undefined}
                  >
                    {expandable && (
                      <TableCell padding="checkbox">
                        <ExpandToggle
                          open={open}
                          onToggle={() => toggle(key)}
                        />
                      </TableCell>
                    )}
                    {columns.map((column) => (
                      <TableCell key={column.id} align={column.align}>
                        {column.render(row)}
                      </TableCell>
                    ))}
                  </TableRow>
                  {expandable && open && (
                    <TableRow>
                      <TableCell colSpan={columnCount}>
                        <Box p={DATA_TABLE_LAYOUT.DETAILS_PADDING}>
                          {renderDetails(row)}
                        </Box>
                      </TableCell>
                    </TableRow>
                  )}
                </Fragment>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>
      <TableState
        isLoading={isLoading}
        error={error}
        isEmpty={rows.length === 0}
        emptyMessage={emptyMessage}
      />
      {paging && (
        <PaginationControls
          page={paging.page}
          rowsPerPage={paging.pageSize}
          totalCount={totalCount}
          lastReachablePage={paging.lastReachablePage}
          onPageChange={paging.onPageChange}
          onRowsPerPageChange={paging.onPageSizeChange}
        />
      )}
    </Box>
  );
}
