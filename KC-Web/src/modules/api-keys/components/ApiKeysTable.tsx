import React from "react";

import type { ApiKeyAPI } from "@api-types/api";
import {
  Box,
  Chip,
  IconButton,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from "@mui/material";

import { formatDate, formatDateTime } from "@utils/date-format";
import { getMonoBody2 } from "@utils/typography";
import { API_KEY_TYPES, TRUNCATION } from "@constants/app";
import { API_KEYS_PAGE } from "@constants/messages";
import { DeleteIcon } from "@theme/icons";

const COLUMNS = [
  API_KEYS_PAGE.COL_NAME,
  API_KEYS_PAGE.COL_KEY_PREFIX,
  API_KEYS_PAGE.COL_TYPE,
  API_KEYS_PAGE.COL_STATUS,
  API_KEYS_PAGE.COL_LAST_USED,
  API_KEYS_PAGE.COL_CREATED,
  API_KEYS_PAGE.COL_EXPIRES,
] as const;
const COLUMN_COUNT = COLUMNS.length + 1;

const StatusChip: React.FC<{ apiKey: ApiKeyAPI }> = ({ apiKey }) => {
  if (apiKey.expiresAt && new Date(apiKey.expiresAt) < new Date()) {
    return (
      <Chip label={API_KEYS_PAGE.STATUS_EXPIRED} color="error" size="small" />
    );
  }
  if (!apiKey.isActive) {
    return (
      <Chip
        label={API_KEYS_PAGE.STATUS_INACTIVE}
        color="default"
        size="small"
      />
    );
  }
  return (
    <Chip label={API_KEYS_PAGE.STATUS_ACTIVE} color="success" size="small" />
  );
};

const ApiKeyRow: React.FC<{
  apiKey: ApiKeyAPI;
  onDelete: (key: ApiKeyAPI) => void;
}> = ({ apiKey, onDelete }) => (
  <TableRow>
    <TableCell>{apiKey.name}</TableCell>
    <TableCell>
      <Typography variant="body2" sx={getMonoBody2}>
        {apiKey.keyPrefix}
        {TRUNCATION.ELLIPSIS}
      </Typography>
    </TableCell>
    <TableCell>
      {apiKey.keyType === API_KEY_TYPES.USER
        ? API_KEYS_PAGE.TYPE_USER
        : API_KEYS_PAGE.TYPE_SERVICE_ACCOUNT}
    </TableCell>
    <TableCell>
      <StatusChip apiKey={apiKey} />
    </TableCell>
    <TableCell>
      {apiKey.lastUsedAt
        ? formatDateTime(apiKey.lastUsedAt)
        : API_KEYS_PAGE.NEVER}
    </TableCell>
    <TableCell>{formatDate(apiKey.createdAt)}</TableCell>
    <TableCell>
      {apiKey.expiresAt
        ? formatDate(apiKey.expiresAt)
        : API_KEYS_PAGE.NO_EXPIRY}
    </TableCell>
    <TableCell align="right">
      <Tooltip title={API_KEYS_PAGE.ACTION_DELETE}>
        <IconButton color="error" onClick={() => onDelete(apiKey)}>
          <DeleteIcon />
        </IconButton>
      </Tooltip>
    </TableCell>
  </TableRow>
);

const EmptyRow: React.FC<{ isLoading: boolean }> = ({ isLoading }) => (
  <TableRow>
    <TableCell colSpan={COLUMN_COUNT} align="center">
      {isLoading ? (
        API_KEYS_PAGE.LOADING
      ) : (
        <Box py={4}>
          <Typography variant="body1" color="text.secondary">
            {API_KEYS_PAGE.NO_API_KEYS}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {API_KEYS_PAGE.NO_API_KEYS_DESCRIPTION}
          </Typography>
        </Box>
      )}
    </TableCell>
  </TableRow>
);

interface ApiKeysTableProps {
  apiKeys: ApiKeyAPI[];
  isLoading: boolean;
  onDelete: (key: ApiKeyAPI) => void;
}

/** The organization's API keys with their status and a delete action. */
const ApiKeysTable: React.FC<ApiKeysTableProps> = ({
  apiKeys,
  isLoading,
  onDelete,
}) => (
  <TableContainer component={Paper} sx={{ overflowX: "auto" }}>
    <Table>
      <TableHead>
        <TableRow>
          {COLUMNS.map((column) => (
            <TableCell key={column}>{column}</TableCell>
          ))}
          <TableCell align="right">{API_KEYS_PAGE.COL_ACTIONS}</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {isLoading || apiKeys.length === 0 ? (
          <EmptyRow isLoading={isLoading} />
        ) : (
          apiKeys.map((key) => (
            <ApiKeyRow key={key.id} apiKey={key} onDelete={onDelete} />
          ))
        )}
      </TableBody>
    </Table>
  </TableContainer>
);

export default ApiKeysTable;
