/**
 * Organizations Table Component
 *
 * Displays organizations in a sortable table.
 */

import React from "react";

import type { OrganizationUI } from "@api-types/api";
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

import { organizationStateChip } from "@utils/chipMappings";
import { formatRelativeDuration } from "@utils/date-format";
import type { SortDirection } from "@constants/app";
import { SORT_DIRECTION } from "@constants/app";
import { ORGANIZATIONS_PAGE } from "@constants/messages";
import { componentSpacing } from "@theme/index";

type OrderBy = "name" | "state" | "createdAt";
interface OrganizationsTableProps {
  organizations: OrganizationUI[];
  orderBy: OrderBy;
  orderDirection: SortDirection;
  onSort: (field: OrderBy) => void;
  onRowClick: (id: string) => void;
  emptyMessage?: string;
}

const OrganizationsTable: React.FC<OrganizationsTableProps> = ({
  organizations,
  orderBy,
  orderDirection,
  onSort,
  onRowClick,
  emptyMessage = ORGANIZATIONS_PAGE.NO_ORGANIZATIONS,
}) => {
  if (organizations.length === 0) {
    return (
      <Paper sx={{ p: 4, textAlign: "center" }}>
        <Typography color="text.secondary">{emptyMessage}</Typography>
      </Paper>
    );
  }

  return (
    <TableContainer component={Paper} sx={{ overflowX: "auto" }}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>
              <TableSortLabel
                active={orderBy === "name"}
                direction={
                  orderBy === "name" ? orderDirection : SORT_DIRECTION.ASC
                }
                onClick={() => onSort("name")}
              >
                {ORGANIZATIONS_PAGE.COL_NAME}
              </TableSortLabel>
            </TableCell>
            <TableCell>{ORGANIZATIONS_PAGE.COL_DESCRIPTION}</TableCell>
            <TableCell>
              <TableSortLabel
                active={orderBy === "state"}
                direction={
                  orderBy === "state" ? orderDirection : SORT_DIRECTION.ASC
                }
                onClick={() => onSort("state")}
              >
                {ORGANIZATIONS_PAGE.COL_STATE}
              </TableSortLabel>
            </TableCell>
            <TableCell>
              <TableSortLabel
                active={orderBy === "createdAt"}
                direction={
                  orderBy === "createdAt" ? orderDirection : SORT_DIRECTION.ASC
                }
                onClick={() => onSort("createdAt")}
              >
                {ORGANIZATIONS_PAGE.COL_CREATED}
              </TableSortLabel>
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {organizations.map((org) => (
            <TableRow
              key={org.id}
              hover
              sx={{ cursor: "pointer" }}
              onClick={() => onRowClick(org.id)}
            >
              <TableCell>
                <Typography fontWeight="medium">{org.name}</Typography>
              </TableCell>
              <TableCell>
                <Typography
                  variant="body2"
                  color="text.secondary"
                  sx={{
                    maxWidth: componentSpacing.truncatedCell.maxWidth,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {org.description || "-"}
                </Typography>
              </TableCell>
              <TableCell>
                <Chip
                  label={organizationStateChip(org.state).label}
                  color={organizationStateChip(org.state).color}
                  size="small"
                />
              </TableCell>
              <TableCell>
                <Typography variant="body2" color="text.secondary">
                  {formatRelativeDuration(org.createdAt)}
                </Typography>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default OrganizationsTable;
