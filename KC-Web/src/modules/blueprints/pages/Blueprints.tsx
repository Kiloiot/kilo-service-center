/**
 * Blueprints Page
 *
 * Main page for device catalog management (manufacturers, models, blueprints).
 * Uses MUI List for tree navigation.
 */

import React, { useId } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import type { BlueprintScope } from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  List,
  Paper,
  Typography,
} from "@mui/material";
import { ConfirmDialog } from "@ui";

import { TabBar, type TabBarItem } from "@components/common/TabBar";
import { TabPanel } from "@components/common/TabPanel";
import { useFeedback } from "@contexts/feedback";
import { useCapabilities } from "@hooks/useCapabilities";
import { BLUEPRINT_SCOPE } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { blueprintDetailPath } from "@router/paths";
import { AddIcon } from "@theme/icons";

import { AddBlueprintDialog } from "../components/AddBlueprintDialog";
import { AddManufacturerDialog } from "../components/AddManufacturerDialog";
import { EditDeviceModelDialog } from "../components/EditDeviceModelDialog";
import { EditManufacturerDialog } from "../components/EditManufacturerDialog";
import { ManufacturerItem } from "../components/ManufacturerItem";
import RegistryStatusNotice from "../components/RegistryStatusNotice";
import {
  useCatalogDialogs,
  useCatalogTree,
  useDeleteDeviceModel,
  useDeleteManufacturer,
  useManufacturers,
} from "../hooks";
import {
  addModelPath,
  catalogPath,
  scopeFromQuery,
} from "../utils/catalog-routes";

const SCOPE_TABS: readonly TabBarItem<BlueprintScope>[] = [
  { value: BLUEPRINT_SCOPE.CUSTOM, label: BLUEPRINT_LABELS.SCOPE_CUSTOM },
  { value: BLUEPRINT_SCOPE.SYSTEM, label: BLUEPRINT_LABELS.SCOPE_SYSTEM },
];

/**
 * Blueprints page component
 */
export const Blueprints: React.FC = () => {
  const navigate = useNavigate();
  const tabsId = useId();
  const { mfrId, modelId } = useParams<{ mfrId?: string; modelId?: string }>();
  const { isServerAdmin } = useCapabilities();

  const [searchParams] = useSearchParams();
  const scope = scopeFromQuery(searchParams);
  // Creating in the System catalog is admin-only.
  const canCreateInScope = scope === BLUEPRINT_SCOPE.CUSTOM || isServerAdmin;

  const {
    data: manufacturers,
    isLoading: mfrsLoading,
    error: mfrsError,
  } = useManufacturers(scope);
  const tree = useCatalogTree({ mfrId, modelId });
  const { dialog, open: openDialog, close: closeDialog } = useCatalogDialogs();
  const deleteManufacturerMutation = useDeleteManufacturer();
  const deleteDeviceModelMutation = useDeleteDeviceModel();
  const feedback = useFeedback();

  const finishDialog = (message: string) => {
    closeDialog();
    feedback.success(message);
  };

  const handleBlueprintClick = (blueprintId: string) => {
    navigate(blueprintDetailPath(blueprintId));
  };

  // The form creates in the open catalog, for the manufacturer whose "+" was clicked.
  const handleAddModel = (manufacturerId: string) => {
    navigate(addModelPath(scope, manufacturerId));
  };

  const confirmDeleteManufacturer = async () => {
    if (dialog.kind !== "deleteManufacturer") return;
    await deleteManufacturerMutation.mutateAsync(dialog.manufacturer.id);
    finishDialog(BLUEPRINT_LABELS.MSG_MANUFACTURER_DELETED);
  };

  const confirmDeleteModel = async () => {
    if (dialog.kind !== "deleteModel") return;
    await deleteDeviceModelMutation.mutateAsync(dialog.model.id);
    finishDialog(BLUEPRINT_LABELS.MSG_MODEL_DELETED);
  };

  return (
    <Box sx={{ p: 3, pt: 4 }}>
      {/* Header */}
      <Box
        sx={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          mb: 3,
        }}
      >
        <Typography variant="h4">{BLUEPRINT_LABELS.PAGE_TITLE}</Typography>
        {canCreateInScope && (
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => openDialog({ kind: "addManufacturer" })}
          >
            {BLUEPRINT_LABELS.ADD_MANUFACTURER}
          </Button>
        )}
      </Box>

      <TabBar
        value={scope}
        onChange={(value) => navigate(catalogPath(value))}
        items={SCOPE_TABS}
        ariaLabel={BLUEPRINT_LABELS.SCOPE_TABS_ARIA}
        idPrefix={tabsId}
      />

      <TabPanel idPrefix={tabsId} value={scope}>
        {scope === BLUEPRINT_SCOPE.SYSTEM && <RegistryStatusNotice />}

        {/* Error display */}
        {mfrsError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {BLUEPRINT_LABELS.ERR_LOAD_MANUFACTURERS}
          </Alert>
        )}

        {/* Loading state */}
        {mfrsLoading && (
          <Box sx={{ display: "flex", justifyContent: "center", p: 4 }}>
            <CircularProgress />
          </Box>
        )}

        {/* Empty state */}
        {!mfrsLoading && manufacturers?.length === 0 && (
          <Paper sx={{ p: 4, textAlign: "center" }}>
            <Typography color="text.secondary">
              {BLUEPRINT_LABELS.NO_MANUFACTURERS}
            </Typography>
          </Paper>
        )}

        {/* Manufacturer list */}
        {manufacturers && manufacturers.length > 0 && (
          <Paper>
            <List>
              {manufacturers.map((mfr) => (
                <ManufacturerItem
                  key={mfr.id}
                  manufacturer={mfr}
                  scope={scope}
                  isServerAdmin={isServerAdmin}
                  isExpanded={tree.expandedMfrs.has(mfr.id)}
                  onToggle={() => tree.toggleMfr(mfr.id)}
                  expandedModels={tree.expandedModels}
                  onToggleModel={tree.toggleModel}
                  onBlueprintClick={handleBlueprintClick}
                  onAddModel={() => handleAddModel(mfr.id)}
                  onEdit={() =>
                    openDialog({ kind: "editManufacturer", manufacturer: mfr })
                  }
                  onDelete={() =>
                    openDialog({
                      kind: "deleteManufacturer",
                      manufacturer: mfr,
                    })
                  }
                  onEditModel={(model) =>
                    openDialog({ kind: "editModel", model })
                  }
                  onDeleteModel={(model) =>
                    openDialog({ kind: "deleteModel", model })
                  }
                  onAddDecoder={(model) =>
                    openDialog({ kind: "addBlueprint", model })
                  }
                />
              ))}
            </List>
          </Paper>
        )}
      </TabPanel>

      <AddManufacturerDialog
        open={dialog.kind === "addManufacturer"}
        scope={scope}
        onClose={closeDialog}
        onSuccess={() =>
          finishDialog(BLUEPRINT_LABELS.MSG_MANUFACTURER_CREATED)
        }
      />

      <EditManufacturerDialog
        manufacturer={
          dialog.kind === "editManufacturer" ? dialog.manufacturer : null
        }
        onClose={closeDialog}
        onSuccess={() =>
          finishDialog(BLUEPRINT_LABELS.MSG_MANUFACTURER_UPDATED)
        }
      />

      <EditDeviceModelDialog
        model={dialog.kind === "editModel" ? dialog.model : null}
        onClose={closeDialog}
        onSuccess={() => finishDialog(BLUEPRINT_LABELS.MSG_MODEL_UPDATED)}
      />

      <ConfirmDialog
        confirmLabel={BLUEPRINT_LABELS.ACTION_DELETE}
        cancelLabel={BLUEPRINT_LABELS.ACTION_CANCEL}
        errorFallback={BLUEPRINT_LABELS.ERR_DELETE_FAILED}
        open={dialog.kind === "deleteManufacturer"}
        title={BLUEPRINT_LABELS.DIALOG_DELETE_MANUFACTURER}
        message={BLUEPRINT_LABELS.CONFIRM_DELETE_MANUFACTURER}
        onClose={closeDialog}
        onConfirm={confirmDeleteManufacturer}
      />

      <ConfirmDialog
        confirmLabel={BLUEPRINT_LABELS.ACTION_DELETE}
        cancelLabel={BLUEPRINT_LABELS.ACTION_CANCEL}
        errorFallback={BLUEPRINT_LABELS.ERR_DELETE_FAILED}
        open={dialog.kind === "deleteModel"}
        title={BLUEPRINT_LABELS.DIALOG_DELETE_MODEL}
        message={BLUEPRINT_LABELS.CONFIRM_DELETE_MODEL}
        onClose={closeDialog}
        onConfirm={confirmDeleteModel}
      />

      <AddBlueprintDialog
        model={dialog.kind === "addBlueprint" ? dialog.model : null}
        onClose={closeDialog}
        onSuccess={() => finishDialog(BLUEPRINT_LABELS.MSG_BLUEPRINT_CREATED)}
      />
    </Box>
  );
};
