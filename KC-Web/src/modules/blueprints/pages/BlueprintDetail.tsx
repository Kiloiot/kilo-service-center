/**
 * Blueprint Detail Page
 *
 * View and edit blueprint specification.
 * Includes decode preview panel for testing payload decoding.
 */
import React, { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { Alert, Box, CircularProgress, Grid } from "@mui/material";

import { DetailNotFound } from "@components/common/DetailNotFound";
import { useFeedback } from "@contexts/feedback";
import { useCapabilities } from "@hooks/useCapabilities";
import { getErrorMessage } from "@utils/error-message";
import { ROUTES } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { componentSpacing } from "@theme/index";

import BlueprintDecodePreviewCard from "../components/BlueprintDecodePreviewCard";
import BlueprintDetailHeader from "../components/BlueprintDetailHeader";
import BlueprintInfoCard from "../components/BlueprintInfoCard";
import BlueprintSpecCard from "../components/BlueprintSpecCard";
import BulkMigrateDialog from "../components/BulkMigrateDialog";
import { RegistrySubmitDialog } from "../components/RegistrySubmitDialog";
import {
  useBlueprint,
  useBlueprintSpecEditor,
  useDecodePlayground,
  useSetBlueprintDefault,
} from "../hooks";

/** Blueprint detail page component */
export const BlueprintDetail: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { isServerAdmin } = useCapabilities();
  const [showMigrate, setShowMigrate] = useState(false);
  const [showRegistryDialog, setShowRegistryDialog] = useState(false);
  const feedback = useFeedback();

  const { data: blueprint, isLoading, error } = useBlueprint(id);
  const editor = useBlueprintSpecEditor(blueprint);
  const playground = useDecodePlayground(id);
  const setDefaultMutation = useSetBlueprintDefault();

  if (isLoading) {
    return (
      <Box sx={{ display: "flex", justifyContent: "center", p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !blueprint) {
    return (
      <DetailNotFound
        message={getErrorMessage(
          error,
          BLUEPRINT_LABELS.ERR_BLUEPRINT_NOT_FOUND,
        )}
        backLabel={BLUEPRINT_LABELS.BACK_TO_BLUEPRINTS}
        onBack={() => navigate(ROUTES.BLUEPRINTS)}
      />
    );
  }

  return (
    <Box sx={{ p: 3 }}>
      <BlueprintDetailHeader
        blueprint={blueprint}
        canMutate={blueprint.isSystem ? isServerAdmin : true}
        isEditing={editor.isEditing}
        isSavePending={editor.isSaving}
        isSetDefaultPending={setDefaultMutation.isPending}
        onStartEdit={editor.start}
        onSave={editor.save}
        onCancelEdit={editor.cancel}
        onSetDefault={() =>
          setDefaultMutation.mutate(
            { id: blueprint.id },
            {
              onSuccess: () =>
                feedback.success(BLUEPRINT_LABELS.MSG_BLUEPRINT_DEFAULT_SET),
            },
          )
        }
        onSubmitToRegistry={() => setShowRegistryDialog(true)}
        onMigrate={() => setShowMigrate(true)}
      />

      {editor.error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {editor.error}
        </Alert>
      )}

      <Grid container spacing={3}>
        <Grid size={componentSpacing.gridSpan.half}>
          <BlueprintInfoCard
            blueprint={blueprint}
            isEditing={editor.isEditing}
            editedVersion={editor.version}
            onEditedVersionChange={editor.setVersion}
          />
        </Grid>
        <Grid size={componentSpacing.gridSpan.half}>
          <BlueprintDecodePreviewCard
            testPayload={playground.payload}
            testFormatId={playground.formatId}
            decodeResult={playground.result}
            isPending={playground.isPending}
            onTestPayloadChange={playground.setPayload}
            onTestFormatIdChange={playground.setFormatId}
            onRun={playground.run}
          />
        </Grid>
        <Grid size={componentSpacing.gridSpan.full}>
          <BlueprintSpecCard
            specJson={blueprint.specJson}
            isEditing={editor.isEditing}
            editedSpec={editor.spec}
            onEditedSpecChange={editor.setSpec}
          />
        </Grid>
      </Grid>

      <RegistrySubmitDialog
        open={showRegistryDialog}
        blueprintId={id ?? null}
        onClose={() => setShowRegistryDialog(false)}
        onSuccess={() => setShowRegistryDialog(false)}
      />
      <BulkMigrateDialog
        open={showMigrate}
        deviceModelId={blueprint.deviceModelId}
        scope={blueprint.isSystem ? "system" : "custom"}
        modelIsSystem={blueprint.isSystem}
        initialBlueprintId={blueprint.id}
        onClose={() => setShowMigrate(false)}
      />
    </Box>
  );
};
