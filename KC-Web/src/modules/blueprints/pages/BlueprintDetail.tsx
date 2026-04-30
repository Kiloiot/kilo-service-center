/**
 * Blueprint Detail Page
 *
 * View and edit blueprint specification.
 * Includes decode preview panel for testing payload decoding.
 */

import React, { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import type { DecodePreviewRequest, DecodePreviewResponse } from '@api-types/api';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  Grid,
  IconButton,
  Paper,
  TextField,
  Typography,
} from '@mui/material';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { api } from '@services/api';
import { formatDateTime, formatDecodedPayload, formatTypeEUI } from '@utils/formatters';
import { ROUTES } from '@constants/app';
import { BLUEPRINT_LABELS } from '@constants/messages';
import { queryKeys } from '@config/query-keys';
import { ArrowBackIcon, CheckCircleIcon, EditIcon, PublishIcon, SaveIcon } from '@theme/icons';

import { RegistrySubmitDialog } from '../components/RegistrySubmitDialog';

/**
 * Blueprint detail page component
 */
export const BlueprintDetail: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  // Edit state
  const [isEditing, setIsEditing] = useState(false);
  const [editedSpec, setEditedSpec] = useState<string>('');
  const [editedVersion, setEditedVersion] = useState<string>('');
  const [editError, setEditError] = useState<string | null>(null);

  // Decode preview state
  const [testPayload, setTestPayload] = useState<string>('');
  const [testFormatId, setTestFormatId] = useState<number>(0);
  const [decodeResult, setDecodeResult] = useState<DecodePreviewResponse | null>(null);

  // Registry submit state
  const [showRegistryDialog, setShowRegistryDialog] = useState(false);

  // Fetch blueprint
  const {
    data: blueprint,
    isLoading,
    error,
  } = useQuery({
    queryKey: queryKeys.blueprints.detail(id!),
    queryFn: () => api.getBlueprint(id!),
    enabled: !!id,
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: async () => {
      if (!id) throw new Error('No blueprint ID');

      // Validate JSON
      try {
        JSON.parse(editedSpec);
      } catch {
        throw new Error(BLUEPRINT_LABELS.ERR_INVALID_JSON);
      }

      await api.updateBlueprint(id, {
        version: editedVersion || undefined,
        specJson: JSON.parse(editedSpec),
      });
    },
    onSuccess: () => {
      setIsEditing(false);
      setEditError(null);
      if (id) {
        queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.detail(id) });
      }
    },
    onError: (err: Error) => {
      setEditError(err.message);
    },
  });

  // Set default mutation
  const setDefaultMutation = useMutation({
    mutationFn: () => api.setBlueprintDefault(id!),
    onSuccess: () => {
      if (id) {
        queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.detail(id) });
      }
      // Invalidate all blueprint lists since default status affects list display
      queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.all });
    },
  });

  // Decode preview mutation
  const decodeMutation = useMutation({
    mutationFn: (data: DecodePreviewRequest) => api.decodePreview(id!, data),
    onSuccess: (result) => {
      setDecodeResult(result);
    },
    onError: (err: Error) => {
      setDecodeResult({
        success: false,
        errorDetail: err.message,
        formatId: testFormatId,
      });
    },
  });

  // Start editing
  const handleStartEdit = () => {
    if (blueprint) {
      setEditedVersion(blueprint.version);
      setEditedSpec(JSON.stringify(blueprint.specJson, null, 2));
      setIsEditing(true);
      setEditError(null);
    }
  };

  // Save edits
  const handleSave = () => {
    updateMutation.mutate();
  };

  // Cancel editing
  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditError(null);
  };

  // Run decode preview
  const handleDecodePreview = () => {
    if (!testPayload) return;
    decodeMutation.mutate({
      userData: testPayload,
      formatId: testFormatId,
    });
  };

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !blueprint) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">
          {error instanceof Error ? error.message : BLUEPRINT_LABELS.ERR_BLUEPRINT_NOT_FOUND}
        </Alert>
        <Button
          startIcon={<ArrowBackIcon />}
          onClick={() => navigate(ROUTES.BLUEPRINTS)}
          sx={{ mt: 2 }}
        >
          {BLUEPRINT_LABELS.BACK_TO_BLUEPRINTS}
        </Button>
      </Box>
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 3 }}>
        <IconButton onClick={() => navigate(ROUTES.BLUEPRINTS)}>
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="h4">
          {BLUEPRINT_LABELS.BLUEPRINT_VERSION_PREFIX}
          {blueprint.version}
        </Typography>
        {blueprint.isDefault && <Chip label={BLUEPRINT_LABELS.BADGE_DEFAULT} color="primary" />}
        {blueprint.registryVerified && (
          <Chip label={BLUEPRINT_LABELS.BADGE_VERIFIED} color="success" />
        )}
        <Box sx={{ flex: 1 }} />
        {!isEditing && (
          <>
            <Button startIcon={<EditIcon />} onClick={handleStartEdit}>
              {BLUEPRINT_LABELS.ACTION_EDIT}
            </Button>
            {!blueprint.isDefault && (
              <Button
                variant="outlined"
                onClick={() => setDefaultMutation.mutate()}
                disabled={setDefaultMutation.isPending}
              >
                {BLUEPRINT_LABELS.SET_DEFAULT}
              </Button>
            )}
            {!blueprint.registryVerified && (
              <Button
                variant="outlined"
                startIcon={<PublishIcon />}
                onClick={() => setShowRegistryDialog(true)}
              >
                {BLUEPRINT_LABELS.SUBMIT_TO_REGISTRY}
              </Button>
            )}
          </>
        )}
        {isEditing && (
          <>
            <Button onClick={handleCancelEdit}>{BLUEPRINT_LABELS.ACTION_CANCEL}</Button>
            <Button
              variant="contained"
              startIcon={<SaveIcon />}
              onClick={handleSave}
              disabled={updateMutation.isPending}
            >
              {BLUEPRINT_LABELS.ACTION_SAVE}
            </Button>
          </>
        )}
      </Box>

      {editError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {editError}
        </Alert>
      )}

      <Grid container spacing={3}>
        {/* Blueprint Info */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                {BLUEPRINT_LABELS.BLUEPRINT_INFORMATION}
              </Typography>
              <Divider sx={{ mb: 2 }} />

              {isEditing ? (
                <TextField
                  label={BLUEPRINT_LABELS.LABEL_VERSION}
                  fullWidth
                  value={editedVersion}
                  onChange={(e) => setEditedVersion(e.target.value)}
                  sx={{ mb: 2 }}
                />
              ) : (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    {BLUEPRINT_LABELS.LABEL_VERSION}
                  </Typography>
                  <Typography>{blueprint.version}</Typography>
                </Box>
              )}

              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  {BLUEPRINT_LABELS.LABEL_TYPE_EUI}
                </Typography>
                <Typography fontFamily="monospace">{formatTypeEUI(blueprint.typeEui)}</Typography>
              </Box>

              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  {BLUEPRINT_LABELS.LABEL_CREATED}
                </Typography>
                <Typography>{formatDateTime(blueprint.createdAt)}</Typography>
              </Box>

              <Box>
                <Typography variant="subtitle2" color="text.secondary">
                  {BLUEPRINT_LABELS.LABEL_UPDATED}
                </Typography>
                <Typography>{formatDateTime(blueprint.updatedAt)}</Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* Decode Preview */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                {BLUEPRINT_LABELS.TEST_DECODE}
              </Typography>
              <Divider sx={{ mb: 2 }} />

              <TextField
                label={BLUEPRINT_LABELS.LABEL_TEST_DATA}
                fullWidth
                value={testPayload}
                onChange={(e) => setTestPayload(e.target.value)}
                placeholder={BLUEPRINT_LABELS.PLACEHOLDER_HEX}
                helperText={BLUEPRINT_LABELS.HELPER_TEST_DATA}
                sx={{ mb: 2 }}
              />

              <TextField
                label={BLUEPRINT_LABELS.LABEL_FORMAT_ID}
                type="number"
                value={testFormatId}
                onChange={(e) => setTestFormatId(parseInt(e.target.value) || 0)}
                sx={{ mb: 2, width: 120 }}
              />

              <Button
                variant="contained"
                onClick={handleDecodePreview}
                disabled={!testPayload || decodeMutation.isPending}
                fullWidth
              >
                {decodeMutation.isPending ? (
                  <CircularProgress size={20} />
                ) : (
                  BLUEPRINT_LABELS.TEST_DECODE
                )}
              </Button>

              {/* Decode Result */}
              {decodeResult && (
                <Paper
                  sx={(theme) => ({
                    mt: 2,
                    p: 2,
                    bgcolor: decodeResult.success ? 'success.dark' : 'error.dark',
                    color: 'common.white',
                    fontFamily: theme.typography.monoFontFamily,
                    fontSize: '0.875rem',
                  })}
                >
                  {decodeResult.success ? (
                    <>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                        <CheckCircleIcon fontSize="small" />
                        <Typography variant="body2">{BLUEPRINT_LABELS.DECODE_SUCCESS}</Typography>
                      </Box>
                      <Typography component="pre" sx={{ whiteSpace: 'pre-wrap' }}>
                        {decodeResult.decodedData
                          ? formatDecodedPayload(decodeResult.decodedData)
                          : BLUEPRINT_LABELS.NO_DECODE_RESULT}
                      </Typography>
                    </>
                  ) : (
                    <>
                      <Typography variant="body2" gutterBottom>
                        {BLUEPRINT_LABELS.DECODE_FAILED}
                      </Typography>
                      {decodeResult.errorCode && (
                        <Typography variant="body2">
                          {BLUEPRINT_LABELS.ERROR_CODE_PREFIX} {decodeResult.errorCode}
                        </Typography>
                      )}
                      {decodeResult.errorDetail && (
                        <Typography variant="body2">{decodeResult.errorDetail}</Typography>
                      )}
                    </>
                  )}
                </Paper>
              )}
            </CardContent>
          </Card>
        </Grid>

        {/* Blueprint Specification */}
        <Grid size={{ xs: 12 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                {BLUEPRINT_LABELS.LABEL_SPEC_JSON}
              </Typography>
              <Divider sx={{ mb: 2 }} />

              {isEditing ? (
                <TextField
                  fullWidth
                  multiline
                  rows={20}
                  value={editedSpec}
                  onChange={(e) => setEditedSpec(e.target.value)}
                  inputProps={{
                    style: { fontFamily: 'monospace', fontSize: '0.875rem' },
                  }}
                  helperText={BLUEPRINT_LABELS.HELPER_SPEC_JSON}
                />
              ) : (
                <Paper
                  sx={(theme) => ({
                    p: 2,
                    bgcolor: 'grey.900',
                    color: 'grey.100',
                    fontFamily: theme.typography.monoFontFamily,
                    fontSize: '0.875rem',
                    overflow: 'auto',
                    maxHeight: 500,
                  })}
                >
                  <pre style={{ margin: 0 }}>{JSON.stringify(blueprint.specJson, null, 2)}</pre>
                </Paper>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Registry Submit Dialog */}
      <RegistrySubmitDialog
        open={showRegistryDialog}
        blueprintId={id ?? null}
        onClose={() => setShowRegistryDialog(false)}
        onSuccess={() => {
          setShowRegistryDialog(false);
          if (id) {
            queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.detail(id) });
          }
        }}
      />
    </Box>
  );
};

export default BlueprintDetail;
