/**
 * API Keys Page
 *
 * Management page for API keys used for programmatic access.
 * Allows users to list, create, and delete API keys.
 */

import React, { useState } from "react";
import { Navigate } from "react-router-dom";

import type { ApiKeyAPI } from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";

import { useSession } from "@contexts/SessionContext";
import {
  useApiKeys,
  useCreateApiKey,
  useDeleteApiKey,
} from "@hooks/useApiKeys";
import { formatRelativeDate } from "@utils/formatters";
import { logger } from "@utils/logger";
import {
  API_KEY_PAGE_SIZE,
  API_KEY_TYPES,
  ROUTES,
  TIMING_COPY_FEEDBACK,
} from "@constants/app";
import {
  API_KEY_CREATED_DIALOG,
  API_KEY_FORM,
  API_KEYS_PAGE,
  CONFIRM_DELETE_API_KEY,
  ERR_CREATE_API_KEY,
  ERR_DELETE_API_KEY,
  ERR_LOAD_API_KEYS,
  MSG_API_KEY_COPIED,
  MSG_API_KEY_CREATED,
  MSG_API_KEY_DELETED,
} from "@constants/messages";
import {
  AddIcon,
  ContentCopyIcon,
  DeleteIcon,
  RefreshIcon,
} from "@theme/icons";

const ApiKeys: React.FC = () => {
  const { isAdmin, isHydrated } = useSession();

  // Runtime admin guard (deep link protection)
  if (isHydrated && !isAdmin) {
    return <Navigate to={ROUTES.HOME} replace />;
  }

  if (!isHydrated) {
    return null;
  }

  return <ApiKeysContent />;
};

const ApiKeysContent: React.FC = () => {
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Create dialog state
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyType, setNewKeyType] = useState(API_KEY_TYPES.USER);
  const [newKeyExpiresAt, setNewKeyExpiresAt] = useState("");

  // Key reveal dialog state
  const [revealDialogOpen, setRevealDialogOpen] = useState(false);
  const [newRawKey, setNewRawKey] = useState("");
  const [keyCopied, setKeyCopied] = useState(false);

  // Delete confirmation state
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [keyToDelete, setKeyToDelete] = useState<ApiKeyAPI | null>(null);

  // React Query hooks
  const { data, isLoading, isError, refetch } = useApiKeys({
    pageSize: API_KEY_PAGE_SIZE,
  });
  const createMutation = useCreateApiKey();
  const deleteMutation = useDeleteApiKey();

  const apiKeys = data?.apiKeys ?? [];

  const handleCreateKey = () => {
    if (!newKeyName.trim()) {
      return;
    }

    setError(null);
    createMutation.mutate(
      {
        name: newKeyName.trim(),
        keyType: newKeyType,
        expiresAt: newKeyExpiresAt
          ? (() => {
              const [y, m, d] = newKeyExpiresAt.split("-").map(Number);
              return new Date(y, m - 1, d, 23, 59, 59, 999);
            })()
          : undefined,
      },
      {
        onSuccess: (response) => {
          setNewRawKey(response.rawKey);
          setRevealDialogOpen(true);
          setCreateDialogOpen(false);
          setNewKeyName("");
          setNewKeyType(API_KEY_TYPES.USER);
          setNewKeyExpiresAt("");
          setSuccessMessage(MSG_API_KEY_CREATED);
        },
        onError: (err) => {
          logger.error("Failed to create API key:", err);
          setError(ERR_CREATE_API_KEY);
        },
      },
    );
  };

  const handleDeleteKey = () => {
    if (!keyToDelete) return;

    setError(null);
    deleteMutation.mutate(keyToDelete.id, {
      onSuccess: () => {
        setSuccessMessage(MSG_API_KEY_DELETED);
        setDeleteDialogOpen(false);
        setKeyToDelete(null);
      },
      onError: (err) => {
        logger.error("Failed to delete API key:", err);
        setError(ERR_DELETE_API_KEY);
      },
    });
  };

  const handleCopyKey = async () => {
    try {
      await navigator.clipboard.writeText(newRawKey);
      setKeyCopied(true);
      setSuccessMessage(MSG_API_KEY_COPIED);
      setTimeout(() => setKeyCopied(false), TIMING_COPY_FEEDBACK);
    } catch (err) {
      logger.error("Failed to copy to clipboard:", err);
    }
  };

  const handleCloseRevealDialog = () => {
    setRevealDialogOpen(false);
    setNewRawKey("");
    setKeyCopied(false);
  };

  const getStatusChip = (key: ApiKeyAPI) => {
    if (key.expiresAt && new Date(key.expiresAt) < new Date()) {
      return (
        <Chip label={API_KEYS_PAGE.STATUS_EXPIRED} color="error" size="small" />
      );
    }
    if (!key.isActive) {
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

  return (
    <Box sx={{ p: 3, pt: 4 }}>
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Box>
          <Typography variant="h4" component="h1" gutterBottom>
            {API_KEYS_PAGE.TITLE}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {API_KEYS_PAGE.DESCRIPTION}
          </Typography>
        </Box>
        <Box display="flex" gap={1}>
          <Tooltip title={API_KEYS_PAGE.TOOLTIP_REFRESH}>
            <span>
              <IconButton onClick={() => refetch()} disabled={isLoading}>
                <RefreshIcon />
              </IconButton>
            </span>
          </Tooltip>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setCreateDialogOpen(true)}
          >
            {API_KEYS_PAGE.ACTION_CREATE}
          </Button>
        </Box>
      </Box>

      {isError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {ERR_LOAD_API_KEYS}
        </Alert>
      )}

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {successMessage && (
        <Alert
          severity="success"
          sx={{ mb: 2 }}
          onClose={() => setSuccessMessage(null)}
        >
          {successMessage}
        </Alert>
      )}

      <TableContainer component={Paper} sx={{ overflowX: "auto" }}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>{API_KEYS_PAGE.COL_NAME}</TableCell>
              <TableCell>{API_KEYS_PAGE.COL_KEY_PREFIX}</TableCell>
              <TableCell>{API_KEYS_PAGE.COL_TYPE}</TableCell>
              <TableCell>{API_KEYS_PAGE.COL_STATUS}</TableCell>
              <TableCell>{API_KEYS_PAGE.COL_LAST_USED}</TableCell>
              <TableCell>{API_KEYS_PAGE.COL_CREATED}</TableCell>
              <TableCell>{API_KEYS_PAGE.COL_EXPIRES}</TableCell>
              <TableCell align="right">{API_KEYS_PAGE.COL_ACTIONS}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={8} align="center">
                  {API_KEYS_PAGE.LOADING}
                </TableCell>
              </TableRow>
            ) : apiKeys.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} align="center">
                  <Box py={4}>
                    <Typography variant="body1" color="text.secondary">
                      {API_KEYS_PAGE.NO_API_KEYS}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {API_KEYS_PAGE.NO_API_KEYS_DESCRIPTION}
                    </Typography>
                  </Box>
                </TableCell>
              </TableRow>
            ) : (
              apiKeys.map((key) => (
                <TableRow key={key.id}>
                  <TableCell>{key.name}</TableCell>
                  <TableCell>
                    <Typography variant="body2" fontFamily="monospace">
                      {key.keyPrefix}...
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {key.keyType === API_KEY_TYPES.USER
                      ? API_KEYS_PAGE.TYPE_USER
                      : API_KEYS_PAGE.TYPE_SERVICE_ACCOUNT}
                  </TableCell>
                  <TableCell>{getStatusChip(key)}</TableCell>
                  <TableCell>
                    {key.lastUsedAt
                      ? formatRelativeDate(key.lastUsedAt)
                      : API_KEYS_PAGE.NEVER}
                  </TableCell>
                  <TableCell>{formatRelativeDate(key.createdAt)}</TableCell>
                  <TableCell>
                    {key.expiresAt
                      ? formatRelativeDate(key.expiresAt)
                      : API_KEYS_PAGE.NO_EXPIRY}
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title={API_KEYS_PAGE.ACTION_DELETE}>
                      <IconButton
                        color="error"
                        onClick={() => {
                          setKeyToDelete(key);
                          setDeleteDialogOpen(true);
                        }}
                      >
                        <DeleteIcon />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Create API Key Dialog */}
      <Dialog
        open={createDialogOpen}
        onClose={() => setCreateDialogOpen(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>{API_KEY_FORM.DIALOG_TITLE_CREATE}</DialogTitle>
        <DialogContent>
          <Box display="flex" flexDirection="column" gap={2} mt={1}>
            <TextField
              label={API_KEY_FORM.LABEL_NAME}
              placeholder={API_KEY_FORM.PLACEHOLDER_NAME}
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
              fullWidth
              autoFocus
              error={!newKeyName.trim() && createMutation.isPending}
              helperText={
                !newKeyName.trim() && createMutation.isPending
                  ? API_KEY_FORM.ERR_NAME_REQUIRED
                  : ""
              }
            />
            <FormControl fullWidth>
              <InputLabel>{API_KEY_FORM.LABEL_TYPE}</InputLabel>
              <Select
                value={newKeyType}
                label={API_KEY_FORM.LABEL_TYPE}
                onChange={(e) => setNewKeyType(e.target.value)}
              >
                <MenuItem value={API_KEY_TYPES.USER}>
                  {API_KEYS_PAGE.TYPE_USER}
                </MenuItem>
                <MenuItem value={API_KEY_TYPES.SERVICE_ACCOUNT}>
                  {API_KEYS_PAGE.TYPE_SERVICE_ACCOUNT}
                </MenuItem>
              </Select>
            </FormControl>
            <TextField
              label={API_KEY_FORM.LABEL_EXPIRES_AT}
              type="date"
              value={newKeyExpiresAt}
              onChange={(e) => setNewKeyExpiresAt(e.target.value)}
              fullWidth
              slotProps={{ inputLabel: { shrink: true } }}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setCreateDialogOpen(false)}
            disabled={createMutation.isPending}
          >
            {API_KEY_FORM.ACTION_CANCEL}
          </Button>
          <Button
            onClick={handleCreateKey}
            variant="contained"
            disabled={createMutation.isPending || !newKeyName.trim()}
          >
            {API_KEY_FORM.ACTION_CREATE}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Key Reveal Dialog */}
      <Dialog
        open={revealDialogOpen}
        onClose={handleCloseRevealDialog}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>{API_KEY_CREATED_DIALOG.TITLE}</DialogTitle>
        <DialogContent>
          <Alert severity="warning" sx={{ mb: 2 }}>
            {API_KEY_CREATED_DIALOG.WARNING}
          </Alert>
          <Typography variant="body2" color="text.secondary" gutterBottom>
            {API_KEY_CREATED_DIALOG.LABEL_API_KEY}
          </Typography>
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              gap: 1,
              p: 2,
              bgcolor: "grey.100",
              borderRadius: 1,
              fontFamily: "monospace",
              wordBreak: "break-all",
            }}
          >
            <Typography
              variant="body2"
              sx={{ flex: 1, fontFamily: "monospace" }}
            >
              {newRawKey}
            </Typography>
            <IconButton
              onClick={handleCopyKey}
              color={keyCopied ? "success" : "default"}
            >
              <ContentCopyIcon />
            </IconButton>
          </Box>
          {keyCopied && (
            <Typography variant="body2" color="success.main" sx={{ mt: 1 }}>
              {API_KEYS_PAGE.ACTION_COPIED}
            </Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseRevealDialog} variant="contained">
            {API_KEY_CREATED_DIALOG.ACTION_CLOSE}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
      >
        <DialogTitle>{API_KEYS_PAGE.ACTION_DELETE}</DialogTitle>
        <DialogContent>
          <DialogContentText>{CONFIRM_DELETE_API_KEY}</DialogContentText>
          {keyToDelete && (
            <Typography variant="body2" sx={{ mt: 2, fontWeight: "bold" }}>
              {keyToDelete.name}
            </Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setDeleteDialogOpen(false)}
            disabled={deleteMutation.isPending}
          >
            {API_KEY_FORM.ACTION_CANCEL}
          </Button>
          <Button
            onClick={handleDeleteKey}
            color="error"
            variant="contained"
            disabled={deleteMutation.isPending}
          >
            {API_KEYS_PAGE.ACTION_DELETE}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ApiKeys;
