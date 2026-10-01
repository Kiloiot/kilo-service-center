/**
 * UsersTableBase Component
 *
 * Shared sortable table used by both System Users and Organization Users views.
 * Renders a single unified layout: Email, Role, Status, Tenant Manager, Base
 * Station Manager, Endpoint Manager, Joined, Actions. System users are mapped
 * to the same columns.
 */

import React from "react";

import type { OrganizationUserUI, SystemUserUI } from "@api-types/api";
import {
  Chip,
  IconButton,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  Tooltip,
  Typography,
} from "@mui/material";

import { memberStatusChip } from "@utils/chipMappings";
import { formatRelativeDuration } from "@utils/date-format";
import type { SortDirection } from "@constants/app";
import { ORG_ROLE, SORT_DIRECTION } from "@constants/app";
import { DATA_TABLE, ORG_USERS_PAGE } from "@constants/messages";
import { DeleteIcon, EditIcon } from "@theme/icons";

import {
  isOrgUser,
  roleLabel,
  unifiedRole,
  unifiedStatus,
} from "../utils/membership-labels";

export type OrderBy = "email" | "role" | "status" | "createdAt";
export interface UsersTableBaseProps {
  users: SystemUserUI[] | OrganizationUserUI[];
  orderBy: OrderBy;
  orderDirection: SortDirection;
  onSort: (field: OrderBy) => void;
  onEdit?: (user: SystemUserUI | OrganizationUserUI) => void;
  onRemove?: (user: SystemUserUI | OrganizationUserUI) => void;
  emptyMessage?: string;
}

function getUserId(user: SystemUserUI | OrganizationUserUI): string {
  return isOrgUser(user) ? user.userId : user.id;
}

interface UsersTableRowProps {
  user: SystemUserUI | OrganizationUserUI;
  onEdit?: (user: SystemUserUI | OrganizationUserUI) => void;
  onRemove?: (user: SystemUserUI | OrganizationUserUI) => void;
}

const UsersTableRow: React.FC<UsersTableRowProps> = ({
  user,
  onEdit,
  onRemove,
}) => {
  const org = isOrgUser(user);
  const role = unifiedRole(user);
  const status = unifiedStatus(user);

  const effectiveOrgAdmin = org
    ? user.isOrgAdmin
    : user.isAdmin || user.isTenantManager;
  const effectiveBs = org
    ? user.isOrgAdmin || user.isBaseStationAdmin
    : user.isAdmin || user.isBaseStationManager;
  const effectiveEp = org
    ? user.isOrgAdmin || user.isEndpointAdmin
    : user.isAdmin || user.isEndpointManager;

  return (
    <TableRow hover>
      <TableCell>
        <Typography variant="body2" fontWeight="medium">
          {user.email}
        </Typography>
      </TableCell>

      <TableCell>
        {roleLabel(user) ? (
          <Chip
            label={roleLabel(user)}
            size="small"
            color={role === ORG_ROLE.OWNER ? "primary" : "default"}
          />
        ) : (
          DATA_TABLE.NO_VALUE
        )}
      </TableCell>

      <TableCell>
        <Chip
          label={memberStatusChip(status).label}
          color={memberStatusChip(status).color}
          size="small"
        />
      </TableCell>

      <TableCell>
        {effectiveOrgAdmin ? ORG_USERS_PAGE.YES : ORG_USERS_PAGE.NO}
      </TableCell>
      <TableCell>
        {effectiveBs ? ORG_USERS_PAGE.YES : ORG_USERS_PAGE.NO}
      </TableCell>
      <TableCell>
        {effectiveEp ? ORG_USERS_PAGE.YES : ORG_USERS_PAGE.NO}
      </TableCell>

      <TableCell>
        <Typography variant="body2" color="text.secondary">
          {formatRelativeDuration(user.createdAt)}
        </Typography>
      </TableCell>

      {(onEdit || onRemove) && (
        <TableCell>
          {onEdit && (
            <Tooltip title={ORG_USERS_PAGE.TOOLTIP_EDIT}>
              <IconButton
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  onEdit(user);
                }}
              >
                <EditIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
          {onRemove && (
            <Tooltip title={ORG_USERS_PAGE.TOOLTIP_REMOVE}>
              <IconButton
                size="small"
                color="error"
                onClick={(e) => {
                  e.stopPropagation();
                  onRemove(user);
                }}
              >
                <DeleteIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          )}
        </TableCell>
      )}
    </TableRow>
  );
};

const UsersTableBase: React.FC<UsersTableBaseProps> = ({
  users,
  orderBy,
  orderDirection,
  onSort,
  onEdit,
  onRemove,
  emptyMessage,
}) => {
  if (users.length === 0) {
    return (
      <Paper sx={{ p: 4, textAlign: "center" }}>
        <Typography color="text.secondary">
          {emptyMessage ?? ORG_USERS_PAGE.NO_USERS}
        </Typography>
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
                active={orderBy === "email"}
                direction={
                  orderBy === "email" ? orderDirection : SORT_DIRECTION.ASC
                }
                onClick={() => onSort("email")}
              >
                {ORG_USERS_PAGE.COL_EMAIL}
              </TableSortLabel>
            </TableCell>

            <TableCell>
              <TableSortLabel
                active={orderBy === "role"}
                direction={
                  orderBy === "role" ? orderDirection : SORT_DIRECTION.ASC
                }
                onClick={() => onSort("role")}
              >
                {ORG_USERS_PAGE.COL_ROLE}
              </TableSortLabel>
            </TableCell>

            <TableCell>
              <TableSortLabel
                active={orderBy === "status"}
                direction={
                  orderBy === "status" ? orderDirection : SORT_DIRECTION.ASC
                }
                onClick={() => onSort("status")}
              >
                {ORG_USERS_PAGE.COL_STATUS}
              </TableSortLabel>
            </TableCell>

            <TableCell>{ORG_USERS_PAGE.COL_TENANT_MANAGER}</TableCell>
            <TableCell>{ORG_USERS_PAGE.COL_BASE_STATION_MANAGER}</TableCell>
            <TableCell>{ORG_USERS_PAGE.COL_ENDPOINT_MANAGER}</TableCell>

            <TableCell>
              <TableSortLabel
                active={orderBy === "createdAt"}
                direction={
                  orderBy === "createdAt" ? orderDirection : SORT_DIRECTION.ASC
                }
                onClick={() => onSort("createdAt")}
              >
                {ORG_USERS_PAGE.COL_JOINED}
              </TableSortLabel>
            </TableCell>

            {(onEdit || onRemove) && (
              <TableCell>{ORG_USERS_PAGE.COL_ACTIONS}</TableCell>
            )}
          </TableRow>
        </TableHead>
        <TableBody>
          {users.map((user) => (
            <UsersTableRow
              key={getUserId(user)}
              user={user}
              onEdit={onEdit}
              onRemove={onRemove}
            />
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default UsersTableBase;
