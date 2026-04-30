/**
 * UsersTableBase Component
 *
 * Shared sortable table used by both System Users and Organization Users views.
 * Renders a single unified layout: Email, Role, Status, Org Admin, BS Admin,
 * EP Admin, Joined, Actions. System users are mapped to the same columns.
 */

import React from 'react';

import type { OrganizationUserUI, SystemUserUI } from '@api-types/api';
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
} from '@mui/material';

import { formatRelativeDuration } from '@utils/formatters';
import { ORG_MEMBER_STATUS, ORG_ROLE } from '@constants/app';
import { ORG_USERS_PAGE } from '@constants/messages';
import { DeleteIcon, EditIcon } from '@theme/icons';

export type OrderBy = 'email' | 'role' | 'status' | 'createdAt';
export type OrderDirection = 'asc' | 'desc';

export interface UsersTableBaseProps {
  users: SystemUserUI[] | OrganizationUserUI[];
  orderBy: OrderBy;
  orderDirection: OrderDirection;
  onSort: (field: OrderBy) => void;
  onEdit?: (user: SystemUserUI | OrganizationUserUI) => void;
  onRemove?: (user: SystemUserUI | OrganizationUserUI) => void;
  emptyMessage?: string;
}

const getRoleLabel = (role: string): string => {
  switch (role) {
    case ORG_ROLE.OWNER:
      return ORG_USERS_PAGE.ROLE_OWNER;
    case ORG_ROLE.ADMIN:
      return ORG_USERS_PAGE.ROLE_ADMIN;
    case ORG_ROLE.MEMBER:
      return ORG_USERS_PAGE.ROLE_MEMBER;
    default:
      return role;
  }
};

const getStatusColor = (status: string): 'success' | 'warning' | 'error' | 'default' => {
  switch (status) {
    case ORG_MEMBER_STATUS.ACTIVE:
      return 'success';
    case ORG_MEMBER_STATUS.INVITED:
      return 'warning';
    case ORG_MEMBER_STATUS.REMOVED:
      return 'error';
    default:
      return 'default';
  }
};

const getStatusLabel = (status: string): string => {
  switch (status) {
    case ORG_MEMBER_STATUS.ACTIVE:
      return ORG_USERS_PAGE.STATUS_ACTIVE;
    case ORG_MEMBER_STATUS.INVITED:
      return ORG_USERS_PAGE.STATUS_INVITED;
    case ORG_MEMBER_STATUS.REMOVED:
      return ORG_USERS_PAGE.STATUS_REMOVED;
    default:
      return status;
  }
};

function isOrgUser(user: SystemUserUI | OrganizationUserUI): user is OrganizationUserUI {
  return 'orgId' in user;
}

function getUserId(user: SystemUserUI | OrganizationUserUI): string {
  return isOrgUser(user) ? user.userId : user.id;
}

/** Map a system user to unified role/status values for display */
function getUnifiedRole(user: SystemUserUI | OrganizationUserUI): string {
  if (isOrgUser(user)) return user.role;
  return user.isAdmin ? ORG_ROLE.ADMIN : ORG_ROLE.MEMBER;
}

function getUnifiedStatus(user: SystemUserUI | OrganizationUserUI): string {
  if (isOrgUser(user)) return user.status;
  return user.isActive ? ORG_MEMBER_STATUS.ACTIVE : ORG_MEMBER_STATUS.REMOVED;
}

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
      <Paper sx={{ p: 4, textAlign: 'center' }}>
        <Typography color="text.secondary">{emptyMessage ?? ORG_USERS_PAGE.NO_USERS}</Typography>
      </Paper>
    );
  }

  return (
    <TableContainer component={Paper} sx={{ overflowX: 'auto' }}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>
              <TableSortLabel
                active={orderBy === 'email'}
                direction={orderBy === 'email' ? orderDirection : 'asc'}
                onClick={() => onSort('email')}
              >
                {ORG_USERS_PAGE.COL_EMAIL}
              </TableSortLabel>
            </TableCell>

            <TableCell>
              <TableSortLabel
                active={orderBy === 'role'}
                direction={orderBy === 'role' ? orderDirection : 'asc'}
                onClick={() => onSort('role')}
              >
                {ORG_USERS_PAGE.COL_ROLE}
              </TableSortLabel>
            </TableCell>

            <TableCell>
              <TableSortLabel
                active={orderBy === 'status'}
                direction={orderBy === 'status' ? orderDirection : 'asc'}
                onClick={() => onSort('status')}
              >
                {ORG_USERS_PAGE.COL_STATUS}
              </TableSortLabel>
            </TableCell>

            <TableCell>{ORG_USERS_PAGE.COL_ORG_ADMIN}</TableCell>
            <TableCell>{ORG_USERS_PAGE.COL_BS_ADMIN}</TableCell>
            <TableCell>{ORG_USERS_PAGE.COL_EP_ADMIN}</TableCell>

            <TableCell>
              <TableSortLabel
                active={orderBy === 'createdAt'}
                direction={orderBy === 'createdAt' ? orderDirection : 'asc'}
                onClick={() => onSort('createdAt')}
              >
                {ORG_USERS_PAGE.COL_JOINED}
              </TableSortLabel>
            </TableCell>

            {(onEdit || onRemove) && <TableCell>{ORG_USERS_PAGE.COL_ACTIONS}</TableCell>}
          </TableRow>
        </TableHead>
        <TableBody>
          {users.map((user) => {
            const id = getUserId(user);
            const org = isOrgUser(user);

            const role = getUnifiedRole(user);
            const status = getUnifiedStatus(user);

            const effectiveOrgAdmin = org ? user.isOrgAdmin : user.isAdmin || user.isTenantManager;
            const effectiveBs = org
              ? user.isOrgAdmin || user.isBaseStationAdmin
              : user.isAdmin || user.isBaseStationManager;
            const effectiveEp = org
              ? user.isOrgAdmin || user.isEndpointAdmin
              : user.isAdmin || user.isEndpointManager;

            return (
              <TableRow key={id} hover>
                <TableCell>
                  <Typography variant="body2" fontWeight="medium">
                    {user.email}
                  </Typography>
                </TableCell>

                <TableCell>
                  <Chip
                    label={getRoleLabel(role)}
                    size="small"
                    color={role === ORG_ROLE.OWNER ? 'primary' : 'default'}
                  />
                </TableCell>

                <TableCell>
                  <Chip
                    label={getStatusLabel(status)}
                    color={getStatusColor(status)}
                    size="small"
                  />
                </TableCell>

                <TableCell>{effectiveOrgAdmin ? ORG_USERS_PAGE.YES : ORG_USERS_PAGE.NO}</TableCell>
                <TableCell>{effectiveBs ? ORG_USERS_PAGE.YES : ORG_USERS_PAGE.NO}</TableCell>
                <TableCell>{effectiveEp ? ORG_USERS_PAGE.YES : ORG_USERS_PAGE.NO}</TableCell>

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
          })}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default UsersTableBase;
