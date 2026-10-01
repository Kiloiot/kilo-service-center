/**
 * The End Points table: sortable columns, attach state and activity per row.
 */

import React from "react";

import type { EndpointUI } from "@api-types/api";
import {
  Chip,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  Typography,
} from "@mui/material";

import { endpointActivityChip } from "@utils/chipMappings";
import { formatRelativeDuration } from "@utils/date-format";
import { formatEui } from "@utils/eui";
import { getMonoBody2 } from "@utils/typography";
import type { SortDirection } from "@constants/app";
import { SORT_DIRECTION } from "@constants/app";
import { ENDPOINTS_PAGE } from "@constants/messages";

import AttachStatusChip from "./AttachStatusChip";

export type EndpointsOrderBy =
  | "name"
  | "epEui"
  | "attachState"
  | "status"
  | "lastSeen";

const COLUMNS: { field: EndpointsOrderBy; label: string }[] = [
  { field: "name", label: ENDPOINTS_PAGE.COL_NAME },
  { field: "epEui", label: ENDPOINTS_PAGE.COL_EUI },
  { field: "attachState", label: ENDPOINTS_PAGE.COL_ATTACH_STATE },
  { field: "status", label: ENDPOINTS_PAGE.COL_STATUS },
  { field: "lastSeen", label: ENDPOINTS_PAGE.COL_LAST_SEEN },
];

interface EndpointsTableProps {
  endpoints: EndpointUI[];
  emptyMessage: string;
  orderBy: string;
  orderDirection: SortDirection;
  onSort: (field: EndpointsOrderBy) => void;
  onRowClick: (epEui: string) => void;
}

function EndpointRow({
  endpoint,
  onClick,
}: {
  endpoint: EndpointUI;
  onClick: () => void;
}) {
  return (
    <TableRow hover onClick={onClick} sx={{ cursor: "pointer" }}>
      <TableCell>
        <Typography variant="body2">
          {endpoint.name || ENDPOINTS_PAGE.UNNAMED_DEVICE}
        </Typography>
      </TableCell>
      <TableCell>
        <Typography variant="body2" sx={(theme) => getMonoBody2(theme)}>
          {formatEui(endpoint.epEui)}
        </Typography>
      </TableCell>
      <TableCell>
        <AttachStatusChip status={endpoint.attachStatus} />
      </TableCell>
      <TableCell>
        <Chip {...endpointActivityChip(endpoint.status)} size="small" />
      </TableCell>
      <TableCell>
        <Typography
          variant="body2"
          color={endpoint.lastSeen ? "text.secondary" : "error"}
        >
          {formatRelativeDuration(endpoint.lastSeen)}
        </Typography>
      </TableCell>
    </TableRow>
  );
}

export const EndpointsTable: React.FC<EndpointsTableProps> = ({
  endpoints,
  emptyMessage,
  orderBy,
  orderDirection,
  onSort,
  onRowClick,
}) => (
  <TableContainer component={Paper} sx={{ overflowX: "auto" }}>
    <Table>
      <TableHead>
        <TableRow>
          {COLUMNS.map(({ field, label }) => (
            <TableCell key={field}>
              <TableSortLabel
                active={orderBy === field}
                direction={
                  orderBy === field ? orderDirection : SORT_DIRECTION.ASC
                }
                onClick={() => onSort(field)}
              >
                {label}
              </TableSortLabel>
            </TableCell>
          ))}
        </TableRow>
      </TableHead>
      <TableBody>
        {endpoints.length === 0 ? (
          <TableRow>
            <TableCell colSpan={COLUMNS.length} align="center" sx={{ py: 4 }}>
              <Typography color="text.secondary">{emptyMessage}</Typography>
            </TableCell>
          </TableRow>
        ) : (
          endpoints.map((endpoint) => (
            <EndpointRow
              key={endpoint.id}
              endpoint={endpoint}
              onClick={() => onRowClick(endpoint.epEui)}
            />
          ))
        )}
      </TableBody>
    </Table>
  </TableContainer>
);
