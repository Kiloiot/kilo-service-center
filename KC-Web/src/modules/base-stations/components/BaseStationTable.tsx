import type { BaseStationUI } from "@api-types/api";
import {
  Box,
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

import type { SortState } from "@contexts/filters";
import { certificateExpiry } from "@utils/certificate-expiry";
import { formatDate, formatDateTime } from "@utils/date-format";
import { formatEui } from "@utils/eui";
import { getMonoBody2 } from "@utils/typography";
import { CERTIFICATE_EXPIRY_STATE, SORT_DIRECTION } from "@constants/app";
import { BASE_STATIONS_PAGE, DATA_TABLE } from "@constants/messages";

import type { BaseStationSortField } from "../utils/base-station-list";
import { BaseStationStatusChip } from "./BaseStationStatusChip";

interface BaseStationTableProps {
  rows: BaseStationUI[];
  sort: SortState;
  columnSortActive: boolean;
  onSort: (field: BaseStationSortField) => void;
  onRowClick: (id: string) => void;
}

const COLUMNS: { label: string; sortField?: BaseStationSortField }[] = [
  { label: BASE_STATIONS_PAGE.COL_NAME, sortField: "name" },
  { label: BASE_STATIONS_PAGE.COL_EUI, sortField: "eui" },
  { label: BASE_STATIONS_PAGE.COL_CONNECTION, sortField: "connectionType" },
  { label: BASE_STATIONS_PAGE.COL_STATUS, sortField: "status" },
  { label: BASE_STATIONS_PAGE.COL_DATE_ADDED },
  { label: BASE_STATIONS_PAGE.COL_LAST_SEEN, sortField: "lastSeen" },
  { label: BASE_STATIONS_PAGE.COL_CERTIFICATE_EXPIRY },
];

function CertificateExpiryCell({ expiryDate }: { expiryDate?: string }) {
  if (!expiryDate) {
    return (
      <Typography variant="body2" color="text.secondary">
        {DATA_TABLE.NO_VALUE}
      </Typography>
    );
  }
  const { daysUntilExpiry, state } = certificateExpiry(expiryDate);
  const isExpired = state === CERTIFICATE_EXPIRY_STATE.EXPIRED;
  const needsAttention =
    isExpired || state === CERTIFICATE_EXPIRY_STATE.CRITICAL;
  const color = isExpired
    ? "error.main"
    : needsAttention
      ? "warning.main"
      : undefined;
  return (
    <Box>
      <Typography
        variant="body2"
        sx={{
          color: color ?? "text.primary",
          fontWeight: needsAttention ? "bold" : "normal",
        }}
      >
        {formatDate(expiryDate)}
      </Typography>
      <Typography variant="caption" sx={{ color: color ?? "text.secondary" }}>
        {isExpired
          ? BASE_STATIONS_PAGE.EXPIRED
          : `${daysUntilExpiry} ${BASE_STATIONS_PAGE.DAYS}`}
      </Typography>
    </Box>
  );
}

export default function BaseStationTable({
  rows,
  sort,
  columnSortActive,
  onSort,
  onRowClick,
}: BaseStationTableProps) {
  return (
    <TableContainer component={Paper} sx={{ mb: 3, overflowX: "auto" }}>
      <Table>
        <TableHead>
          <TableRow>
            {COLUMNS.map((column) => (
              <TableCell key={column.label}>
                {column.sortField ? (
                  <TableSortLabel
                    active={columnSortActive && sort.field === column.sortField}
                    direction={
                      sort.field === column.sortField
                        ? sort.direction
                        : SORT_DIRECTION.ASC
                    }
                    onClick={() =>
                      onSort(column.sortField as BaseStationSortField)
                    }
                  >
                    {column.label}
                  </TableSortLabel>
                ) : (
                  column.label
                )}
              </TableCell>
            ))}
          </TableRow>
        </TableHead>
        <TableBody>
          {rows.map((baseStation) => (
            <TableRow
              key={baseStation.id}
              hover
              onClick={() => onRowClick(baseStation.id)}
              sx={{ cursor: "pointer" }}
            >
              <TableCell>
                {baseStation.name || formatEui(baseStation.eui)}
              </TableCell>
              <TableCell>
                <Typography variant="body2" sx={(theme) => getMonoBody2(theme)}>
                  {formatEui(baseStation.eui)}
                </Typography>
              </TableCell>
              <TableCell>{baseStation.connectionType}</TableCell>
              <TableCell>
                <BaseStationStatusChip status={baseStation.status} />
              </TableCell>
              <TableCell>
                {baseStation.createdAt
                  ? formatDate(baseStation.createdAt)
                  : "-"}
              </TableCell>
              <TableCell>{formatDateTime(baseStation.lastSeen)}</TableCell>
              <TableCell>
                <CertificateExpiryCell
                  expiryDate={baseStation.certificateExpiryDate}
                />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
