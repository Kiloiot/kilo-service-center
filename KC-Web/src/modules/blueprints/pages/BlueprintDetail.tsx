/**
 * Blueprint Detail Page
 *
 * View and edit blueprint specification.
 * Includes decode preview panel for testing payload decoding.
 */

import React, { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type {
  BlueprintUI,
  DecodePreviewRequest,
  DecodePreviewResponse,
} from "@api-types/api";
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
} from "@mui/material";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@services/api";
import {
  formatDateTime,
  formatDecodedPayload,
  formatTypeEUI,
} from "@utils/formatters";
import { ROUTES } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { queryKeys } from "@config/query-keys";
import {
  ArrowBackIcon,
  CheckCircleIcon,
  EditIcon,
  PublishIcon,
  SaveIcon,
} from "@theme/icons";

import { RegistrySubmitDialog } from "../components/RegistrySubmitDialog";

/**
 * Returns parsed JSON when text is a valid blueprint spec, or throws an Error
 * with the localized invalid-JSON message otherwise.
 */
function validateBlueprintSpecJson(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    throw new Error(BLUEPRINT_LABELS.ERR_INVALID_JSON);
  }
}

interface BlueprintDetailHeaderProps {
  blueprint: BlueprintUI;
  isEditing: boolean;
  isSavePending: boolean;
  isSetDefaultPending: boolean;
  onStartEdit: () => void;
  onSave: () => void;
  onCancelEdit: () => void;
  onSetDefault: () => void;
  onSubmitToRegistry: () => void;
}

const BlueprintDetailHeader: React.FC<BlueprintDetailHeaderProps> = ({
  blueprint,
  isEditing,
  isSavePending,
  isSetDefaultPending,
  onStartEdit,
  onSave,
  onCancelEdit,
  onSetDefault,
  onSubmitToRegistry,
}) => {
  const navigate = useNavigate();
  return (
    <Box sx={{ display: "flex", alignItems: "center", gap: 2, mb: 3 }}>
      <IconButton onClick={() => navigate(ROUTES.BLUEPRINTS)}>
        <ArrowBackIcon />
      </IconButton>
      <Typography variant="h4">
        {BLUEPRINT_LABELS.BLUEPRINT_VERSION_PREFIX}
        {blueprint.version}
      </Typography>
      {blueprint.isDefault && (
        <Chip label={BLUEPRINT_LABELS.BADGE_DEFAULT} color="primary" />
      )}
      {blueprint.registryVerified && (
        <Chip label={BLUEPRINT_LABELS.BADGE_VERIFIED} color="success" />
      )}
      <Box sx={{ flex: 1 }} />
      {!isEditing && (
        <>
          <Button startIcon={<EditIcon />} onClick={onStartEdit}>
            {BLUEPRINT_LABELS.ACTION_EDIT}
          </Button>
          {!blueprint.isDefault && (
            <Button
              variant="outlined"
              onClick={onSetDefault}
              disabled={isSetDefaultPending}
            >
              {BLUEPRINT_LABELS.SET_DEFAULT}
            </Button>
          )}
          {!blueprint.registryVerified && (
            <Button
              variant="outlined"
              startIcon={<PublishIcon />}
              onClick={onSubmitToRegistry}
            >
              {BLUEPRINT_LABELS.SUBMIT_TO_REGISTRY}
            </Button>
          )}
        </>
      )}
      {isEditing && (
        <>
          <Button onClick={onCancelEdit}>
            {BLUEPRINT_LABELS.ACTION_CANCEL}
          </Button>
          <Button
            variant="contained"
            startIcon={<SaveIcon />}
            onClick={onSave}
            disabled={isSavePending}
          >
            {BLUEPRINT_LABELS.ACTION_SAVE}
          </Button>
        </>
      )}
    </Box>
  );
};

interface BlueprintInfoCardProps {
  blueprint: BlueprintUI;
  isEditing: boolean;
  editedVersion: string;
  onEditedVersionChange: (value: string) => void;
}

const BlueprintInfoCard: React.FC<BlueprintInfoCardProps> = ({
  blueprint,
  isEditing,
  editedVersion,
  onEditedVersionChange,
}) => (
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
          onChange={(e) => onEditedVersionChange(e.target.value)}
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
        <Typography fontFamily="monospace">
          {formatTypeEUI(blueprint.typeEui)}
        </Typography>
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
);

interface BlueprintDecodePreviewCardProps {
  testPayload: string;
  testFormatId: number;
  decodeResult: DecodePreviewResponse | null;
  isPending: boolean;
  onTestPayloadChange: (value: string) => void;
  onTestFormatIdChange: (value: number) => void;
  onRun: () => void;
}

const BlueprintDecodePreviewCard: React.FC<BlueprintDecodePreviewCardProps> = ({
  testPayload,
  testFormatId,
  decodeResult,
  isPending,
  onTestPayloadChange,
  onTestFormatIdChange,
  onRun,
}) => (
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
        onChange={(e) => onTestPayloadChange(e.target.value)}
        placeholder={BLUEPRINT_LABELS.PLACEHOLDER_HEX}
        helperText={BLUEPRINT_LABELS.HELPER_TEST_DATA}
        sx={{ mb: 2 }}
      />

      <TextField
        label={BLUEPRINT_LABELS.LABEL_FORMAT_ID}
        type="number"
        value={testFormatId}
        onChange={(e) => onTestFormatIdChange(parseInt(e.target.value) || 0)}
        sx={{ mb: 2, width: 120 }}
      />

      <Button
        variant="contained"
        onClick={onRun}
        disabled={!testPayload || isPending}
        fullWidth
      >
        {isPending ? (
          <CircularProgress size={20} />
        ) : (
          BLUEPRINT_LABELS.TEST_DECODE
        )}
      </Button>

      {decodeResult && (
        <Paper
          sx={(theme) => ({
            mt: 2,
            p: 2,
            bgcolor: decodeResult.success ? "success.dark" : "error.dark",
            color: "common.white",
            fontFamily: theme.typography.monoFontFamily,
            fontSize: "0.875rem",
          })}
        >
          {decodeResult.success ? (
            <>
              <Box
                sx={{ display: "flex", alignItems: "center", gap: 1, mb: 1 }}
              >
                <CheckCircleIcon fontSize="small" />
                <Typography variant="body2">
                  {BLUEPRINT_LABELS.DECODE_SUCCESS}
                </Typography>
              </Box>
              <Typography component="pre" sx={{ whiteSpace: "pre-wrap" }}>
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
                <Typography variant="body2">
                  {decodeResult.errorDetail}
                </Typography>
              )}
            </>
          )}
        </Paper>
      )}
    </CardContent>
  </Card>
);

interface BlueprintSpecCardProps {
  specJson: unknown;
  isEditing: boolean;
  editedSpec: string;
  onEditedSpecChange: (value: string) => void;
}

const BlueprintSpecCard: React.FC<BlueprintSpecCardProps> = ({
  specJson,
  isEditing,
  editedSpec,
  onEditedSpecChange,
}) => (
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
          onChange={(e) => onEditedSpecChange(e.target.value)}
          inputProps={{
            style: { fontFamily: "monospace", fontSize: "0.875rem" },
          }}
          helperText={BLUEPRINT_LABELS.HELPER_SPEC_JSON}
        />
      ) : (
        <Paper
          sx={(theme) => ({
            p: 2,
            bgcolor: "grey.900",
            color: "grey.100",
            fontFamily: theme.typography.monoFontFamily,
            fontSize: "0.875rem",
            overflow: "auto",
            maxHeight: 500,
          })}
        >
          <pre style={{ margin: 0 }}>{JSON.stringify(specJson, null, 2)}</pre>
        </Paper>
      )}
    </CardContent>
  </Card>
);

/** Blueprint detail page component */
export const BlueprintDetail: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  const [isEditing, setIsEditing] = useState(false);
  const [editedSpec, setEditedSpec] = useState<string>("");
  const [editedVersion, setEditedVersion] = useState<string>("");
  const [editError, setEditError] = useState<string | null>(null);

  const [testPayload, setTestPayload] = useState<string>("");
  const [testFormatId, setTestFormatId] = useState<number>(0);
  const [decodeResult, setDecodeResult] =
    useState<DecodePreviewResponse | null>(null);

  const [showRegistryDialog, setShowRegistryDialog] = useState(false);

  const {
    data: blueprint,
    isLoading,
    error,
  } = useQuery({
    queryKey: queryKeys.blueprints.detail(id!),
    queryFn: () => api.getBlueprint(id!),
    enabled: !!id,
  });

  const updateMutation = useMutation({
    mutationFn: async () => {
      if (!id) throw new Error("No blueprint ID");
      const specJson = validateBlueprintSpecJson(editedSpec);
      await api.updateBlueprint(id, {
        version: editedVersion || undefined,
        specJson,
      });
    },
    onSuccess: () => {
      setIsEditing(false);
      setEditError(null);
      if (id) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.blueprints.detail(id),
        });
      }
    },
    onError: (err: Error) => {
      setEditError(err.message);
    },
  });

  const setDefaultMutation = useMutation({
    mutationFn: () => api.setBlueprintDefault(id!),
    onSuccess: () => {
      if (id) {
        queryClient.invalidateQueries({
          queryKey: queryKeys.blueprints.detail(id),
        });
      }
      // Default status affects every list, not just the detail entry.
      queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.all });
    },
  });

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

  const handleStartEdit = () => {
    if (blueprint) {
      setEditedVersion(blueprint.version);
      setEditedSpec(JSON.stringify(blueprint.specJson, null, 2));
      setIsEditing(true);
      setEditError(null);
    }
  };

  const handleSave = () => {
    updateMutation.mutate();
  };

  const handleCancelEdit = () => {
    setIsEditing(false);
    setEditError(null);
  };

  const handleDecodePreview = () => {
    if (!testPayload) return;
    decodeMutation.mutate({
      userData: testPayload,
      formatId: testFormatId,
    });
  };

  if (isLoading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !blueprint) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">
          {error instanceof Error
            ? error.message
            : BLUEPRINT_LABELS.ERR_BLUEPRINT_NOT_FOUND}
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
      <BlueprintDetailHeader
        blueprint={blueprint}
        isEditing={isEditing}
        isSavePending={updateMutation.isPending}
        isSetDefaultPending={setDefaultMutation.isPending}
        onStartEdit={handleStartEdit}
        onSave={handleSave}
        onCancelEdit={handleCancelEdit}
        onSetDefault={() => setDefaultMutation.mutate()}
        onSubmitToRegistry={() => setShowRegistryDialog(true)}
      />

      {editError && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {editError}
        </Alert>
      )}

      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 6 }}>
          <BlueprintInfoCard
            blueprint={blueprint}
            isEditing={isEditing}
            editedVersion={editedVersion}
            onEditedVersionChange={setEditedVersion}
          />
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <BlueprintDecodePreviewCard
            testPayload={testPayload}
            testFormatId={testFormatId}
            decodeResult={decodeResult}
            isPending={decodeMutation.isPending}
            onTestPayloadChange={setTestPayload}
            onTestFormatIdChange={setTestFormatId}
            onRun={handleDecodePreview}
          />
        </Grid>

        <Grid size={{ xs: 12 }}>
          <BlueprintSpecCard
            specJson={blueprint.specJson}
            isEditing={isEditing}
            editedSpec={editedSpec}
            onEditedSpecChange={setEditedSpec}
          />
        </Grid>
      </Grid>

      <RegistrySubmitDialog
        open={showRegistryDialog}
        blueprintId={id ?? null}
        onClose={() => setShowRegistryDialog(false)}
        onSuccess={() => {
          setShowRegistryDialog(false);
          if (id) {
            queryClient.invalidateQueries({
              queryKey: queryKeys.blueprints.detail(id),
            });
          }
        }}
      />
    </Box>
  );
};

export default BlueprintDetail;
