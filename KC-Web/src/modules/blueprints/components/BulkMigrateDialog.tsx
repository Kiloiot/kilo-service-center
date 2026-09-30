/**
 * Re-materializes a device model's snapshot endpoints onto a chosen blueprint via KC-Core.
 */

import React, { useEffect, useState } from "react";

import type { BlueprintScope } from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  Checkbox,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControlLabel,
  TextField,
} from "@mui/material";

import { useFeedback } from "@contexts/feedback";
import { getErrorMessage } from "@utils/error-message";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { componentSpacing } from "@theme/index";

import {
  useBlueprints,
  useBulkAssignBlueprint,
  useModelSnapshotCount,
} from "../hooks";

interface BulkMigrateDialogProps {
  open: boolean;
  deviceModelId: string;
  scope: BlueprintScope;
  modelIsSystem: boolean;
  initialBlueprintId?: string;
  onClose: () => void;
}

const BulkMigrateDialog: React.FC<BulkMigrateDialogProps> = ({
  open,
  deviceModelId,
  scope,
  modelIsSystem,
  initialBlueprintId,
  onClose,
}) => {
  const [selectedBlueprintId, setSelectedBlueprintId] = useState("");
  const [setAsDefault, setSetAsDefault] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const feedback = useFeedback();

  const { data: blueprints, isLoading: blueprintsLoading } = useBlueprints(
    deviceModelId,
    scope,
    { enabled: open },
  );

  const { data: affectedCount, isLoading: countLoading } =
    useModelSnapshotCount(deviceModelId, { enabled: open });

  useEffect(() => {
    if (!open) return;
    setError(null);
    setSetAsDefault(!modelIsSystem);
  }, [open, modelIsSystem]);

  // Preselect the requested blueprint, else the model default, else the first.
  useEffect(() => {
    if (!open || !blueprints) return;
    const fallback =
      blueprints.find((b) => b.isDefault)?.id ?? blueprints[0]?.id ?? "";
    setSelectedBlueprintId(initialBlueprintId ?? fallback);
  }, [open, blueprints, initialBlueprintId]);

  const mutation = useBulkAssignBlueprint();

  const migrate = () => {
    mutation.mutate(
      {
        blueprintId: selectedBlueprintId,
        deviceModelId,
        setAsDefault: modelIsSystem ? false : setAsDefault,
      },
      {
        onSuccess: (result) => {
          onClose();
          feedback.success(
            BLUEPRINT_LABELS.MSG_MIGRATE_SUCCESS.replace(
              "{affected}",
              String(result.affectedCount),
            ),
          );
        },
        onError: (err: Error) =>
          setError(getErrorMessage(err, BLUEPRINT_LABELS.ERR_MIGRATE_FAILED)),
      },
    );
  };

  const isBusy = blueprintsLoading || countLoading;
  const nothingToDo = !isBusy && !affectedCount && !setAsDefault;
  const canConfirm =
    !!selectedBlueprintId && !isBusy && !nothingToDo && !mutation.isPending;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{BLUEPRINT_LABELS.MIGRATE_DIALOG_TITLE}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {isBusy ? (
          <Box sx={{ display: "flex", justifyContent: "center", p: 2 }}>
            <CircularProgress size={componentSpacing.spinner.section} />
          </Box>
        ) : (
          <DialogContentText sx={{ mb: 2 }}>
            {affectedCount ? (
              <>
                {BLUEPRINT_LABELS.MIGRATE_AFFECTED_PREFIX}
                {affectedCount}
                {BLUEPRINT_LABELS.MIGRATE_AFFECTED_SUFFIX}
              </>
            ) : (
              BLUEPRINT_LABELS.MIGRATE_NO_DEVICES
            )}
          </DialogContentText>
        )}

        <TextField
          select
          fullWidth
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_VERSION}
          value={selectedBlueprintId}
          onChange={(e) => setSelectedBlueprintId(e.target.value)}
          disabled={blueprintsLoading}
          slotProps={{ select: { native: true } }}
        >
          <option value="" />
          {blueprints?.map((b) => (
            <option key={b.id} value={b.id}>
              {b.version}
              {b.isDefault ? ` (${BLUEPRINT_LABELS.BADGE_DEFAULT})` : ""}
            </option>
          ))}
        </TextField>

        {!modelIsSystem && (
          <FormControlLabel
            control={
              <Checkbox
                checked={setAsDefault}
                onChange={(e) => setSetAsDefault(e.target.checked)}
              />
            }
            label={BLUEPRINT_LABELS.MIGRATE_SET_AS_DEFAULT}
          />
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={mutation.isPending}>
          {BLUEPRINT_LABELS.ACTION_CANCEL}
        </Button>
        <Button onClick={migrate} variant="contained" disabled={!canConfirm}>
          {mutation.isPending ? (
            <CircularProgress size={componentSpacing.spinner.button} />
          ) : (
            BLUEPRINT_LABELS.MIGRATE_CONFIRM
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default BulkMigrateDialog;
