/**
 * Organizations Table Component
 *
 * Displays organizations in a sortable table.
 */

import React from 'react';

import type { OrganizationUI } from '@api-types/api';
import {
  Box,
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
} from '@mui/material';

import { formatRelativeDuration } from '@utils/formatters';
import { ORG_STATE } from '@constants/app';
import { ORGANIZATIONS_PAGE } from '@constants/messages';

type OrderBy = 'name' | 'state' | 'createdAt';
type OrderDirection = 'asc' | 'desc';

interface OrganizationsTableProps {
  organizations: OrganizationUI[];
  orderBy: OrderBy;
  orderDirection: OrderDirection;
  onSort: (field: OrderBy) => void;
  onRowClick: (id: string) => void;
  emptyMessage?: string;
}

const getStateColor = (state: string): 'success' | 'warning' | 'error' | 'default' => {
  switch (state) {
    case ORG_STATE.ACTIVE:
      return 'success';
    case ORG_STATE.SUSPENDED:
      return 'warning';
    case ORG_STATE.ARCHIVED:
      return 'error';
    default:
      return 'default';
  }
};

const getStateLabel = (state: string): string => {
  switch (state) {
    case ORG_STATE.ACTIVE:
      return ORGANIZATIONS_PAGE.STATE_ACTIVE;
    case ORG_STATE.SUSPENDED:
      return ORGANIZATIONS_PAGE.STATE_SUSPENDED;
    case ORG_STATE.ARCHIVED:
      return ORGANIZATIONS_PAGE.STATE_ARCHIVED;
    default:
      return state;
  }
};

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
      <Paper sx={{ p: 4, textAlign: 'center' }}>
        <Typography color="text.secondary">{emptyMessage}</Typography>
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
                active={orderBy === 'name'}
                direction={orderBy === 'name' ? orderDirection : 'asc'}
                onClick={() => onSort('name')}
              >
                {ORGANIZATIONS_PAGE.COL_NAME}
              </TableSortLabel>
            </TableCell>
            <TableCell>{ORGANIZATIONS_PAGE.COL_DESCRIPTION}</TableCell>
            <TableCell>
              <TableSortLabel
                active={orderBy === 'state'}
                direction={orderBy === 'state' ? orderDirection : 'asc'}
                onClick={() => onSort('state')}
              >
                {ORGANIZATIONS_PAGE.COL_STATE}
              </TableSortLabel>
            </TableCell>
            <TableCell>{ORGANIZATIONS_PAGE.COL_CAN_HAVE_BS}</TableCell>
            <TableCell>{ORGANIZATIONS_PAGE.COL_BS_QUOTA}</TableCell>
            <TableCell>{ORGANIZATIONS_PAGE.COL_EP_QUOTA}</TableCell>
            <TableCell>
              <TableSortLabel
                active={orderBy === 'createdAt'}
                direction={orderBy === 'createdAt' ? orderDirection : 'asc'}
                onClick={() => onSort('createdAt')}
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
              sx={{ cursor: 'pointer' }}
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
                    maxWidth: 200,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {org.description || '-'}
                </Typography>
              </TableCell>
              <TableCell>
                <Chip
                  label={getStateLabel(org.state)}
                  color={getStateColor(org.state)}
                  size="small"
                />
              </TableCell>
              <TableCell>
                <Chip
                  label={
                    org.canHaveBaseStations
                      ? ORGANIZATIONS_PAGE.QUOTA_ALLOWED
                      : ORGANIZATIONS_PAGE.QUOTA_NOT_ALLOWED
                  }
                  color={org.canHaveBaseStations ? 'success' : 'default'}
                  size="small"
                  variant="outlined"
                />
              </TableCell>
              <TableCell>
                <Box>{org.maxBaseStationCount ?? ORGANIZATIONS_PAGE.QUOTA_UNLIMITED}</Box>
              </TableCell>
              <TableCell>
                <Box>{org.maxEndpointCount ?? ORGANIZATIONS_PAGE.QUOTA_UNLIMITED}</Box>
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
