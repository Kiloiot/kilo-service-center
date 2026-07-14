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
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@services/api";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { queryKeys } from "@config/query-keys";

interface BulkMigrateDialogProps {
  open: boolean;
  deviceModelId: string;
  scope: BlueprintScope;
  modelIsSystem: boolean;
  initialBlueprintId?: string;
  onClose: () => void;
  onSuccess?: () => void;
}

export const BulkMigrateDialog: React.FC<BulkMigrateDialogProps> = ({
  open,
  deviceModelId,
  scope,
  modelIsSystem,
  initialBlueprintId,
  onClose,
  onSuccess,
}) => {
  const queryClient = useQueryClient();

  const [selectedBlueprintId, setSelectedBlueprintId] = useState("");
  const [setAsDefault, setSetAsDefault] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [affectedDone, setAffectedDone] = useState<number | null>(null);

  const { data: blueprints, isLoading: blueprintsLoading } = useQuery({
    queryKey: queryKeys.blueprints.list(deviceModelId, scope),
    queryFn: () => api.getBlueprints(deviceModelId, scope),
    enabled: open && !!deviceModelId,
  });

  const { data: affectedCount, isLoading: countLoading } = useQuery({
    queryKey: queryKeys.blueprints.modelSnapshotCount(deviceModelId),
    queryFn: () => api.countModelSnapshotEndpoints(deviceModelId),
    enabled: open && !!deviceModelId,
  });

  useEffect(() => {
    if (!open) return;
    setError(null);
    setAffectedDone(null);
    setSetAsDefault(!modelIsSystem);
  }, [open, modelIsSystem]);

  // Preselect the requested blueprint, else the model default, else the first.
  useEffect(() => {
    if (!open || !blueprints) return;
    const fallback =
      blueprints.find((b) => b.isDefault)?.id ?? blueprints[0]?.id ?? "";
    setSelectedBlueprintId(initialBlueprintId ?? fallback);
  }, [open, blueprints, initialBlueprintId]);

  const mutation = useMutation({
    mutationFn: () =>
      api.bulkAssignBlueprint({
        blueprintId: selectedBlueprintId,
        deviceModelId,
        setAsDefault: modelIsSystem ? false : setAsDefault,
      }),
    onSuccess: (result) => {
      setAffectedDone(result.affectedCount);
      queryClient.invalidateQueries({ queryKey: queryKeys.blueprints.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.all });
      queryClient.invalidateQueries({
        queryKey: queryKeys.blueprints.modelSnapshotCount(deviceModelId),
      });
      onSuccess?.();
    },
    onError: (err: Error) => {
      setError(err.message || BLUEPRINT_LABELS.ERR_MIGRATE_FAILED);
    },
  });

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

        {affectedDone !== null ? (
          <Alert severity="success">
            {BLUEPRINT_LABELS.MSG_MIGRATE_SUCCESS}
            {BLUEPRINT_LABELS.MIGRATE_AFFECTED_PREFIX}
            {affectedDone}
            {BLUEPRINT_LABELS.MIGRATE_AFFECTED_SUFFIX}
          </Alert>
        ) : (
          <>
            {isBusy ? (
              <Box sx={{ display: "flex", justifyContent: "center", p: 2 }}>
                <CircularProgress size={24} />
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
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={mutation.isPending}>
          {affectedDone !== null
            ? BLUEPRINT_LABELS.ACTION_CLOSE
            : BLUEPRINT_LABELS.ACTION_CANCEL}
        </Button>
        {affectedDone === null && (
          <Button
            onClick={() => mutation.mutate()}
            variant="contained"
            disabled={!canConfirm}
          >
            {mutation.isPending ? (
              <CircularProgress size={20} />
            ) : (
              BLUEPRINT_LABELS.MIGRATE_CONFIRM
            )}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};

export default BulkMigrateDialog;
