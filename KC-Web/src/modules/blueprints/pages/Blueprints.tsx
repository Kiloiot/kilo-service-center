/**
 * Blueprints Page
 *
 * Main page for device catalog management (manufacturers, models, blueprints).
 * Uses MUI List + Collapse for tree navigation per plan constraints.
 */

import React, { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type {
  CreateManufacturerRequest,
  DeviceModelUI,
  ManufacturerUI,
  UpdateDeviceModelRequest,
  UpdateManufacturerRequest,
} from "@api-types/api";
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Paper,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@services/api";
import { formatTypeEUI } from "@utils/formatters";
import { ROUTES } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { queryKeys } from "@config/query-keys";
import {
  AddIcon,
  BlueprintIcon,
  CategoryIcon,
  DeleteIcon,
  EditIcon,
  ExpandLess,
  ExpandMore,
} from "@theme/icons";

/**
 * Blueprints page component
 */
export const Blueprints: React.FC = () => {
  const navigate = useNavigate();
  const { mfrId, modelId } = useParams<{ mfrId?: string; modelId?: string }>();
  const queryClient = useQueryClient();

  // State for expanded manufacturers
  const [expandedMfrs, setExpandedMfrs] = useState<Set<string>>(new Set());
  // State for expanded models
  const [expandedModels, setExpandedModels] = useState<Set<string>>(new Set());
  // Dialog states - Add
  const [showMfrDialog, setShowMfrDialog] = useState(false);
  // Dialog states - Edit
  const [editMfr, setEditMfr] = useState<ManufacturerUI | null>(null);
  const [editModel, setEditModel] = useState<{
    model: DeviceModelUI;
    mfrId: string;
  } | null>(null);
  // Dialog states - Delete
  const [deleteMfr, setDeleteMfr] = useState<ManufacturerUI | null>(null);
  const [deleteModel, setDeleteModel] = useState<{
    model: DeviceModelUI;
    mfrId: string;
  } | null>(null);
  const [addBlueprintModel, setAddBlueprintModel] =
    useState<DeviceModelUI | null>(null);

  // Fetch manufacturers
  const {
    data: manufacturers,
    isLoading: mfrsLoading,
    error: mfrsError,
  } = useQuery({
    queryKey: queryKeys.blueprints.manufacturers(),
    queryFn: () => api.getManufacturers(),
  });

  const { data: routeModel } = useQuery({
    queryKey: queryKeys.blueprints.deviceModelDetail(modelId!),
    queryFn: () => api.getDeviceModel(modelId!),
    enabled: !!modelId,
  });

  React.useEffect(() => {
    if (!mfrId) {
      return;
    }
    setExpandedMfrs((prev) => {
      const next = new Set(prev);
      next.add(mfrId);
      return next;
    });
  }, [mfrId]);

  React.useEffect(() => {
    if (!routeModel) {
      return;
    }
    setExpandedMfrs((prev) => {
      const next = new Set(prev);
      next.add(routeModel.manufacturerId);
      return next;
    });
    setExpandedModels((prev) => {
      const next = new Set(prev);
      next.add(routeModel.id);
      return next;
    });
  }, [routeModel]);

  // Toggle manufacturer expansion
  const toggleMfr = (id: string) => {
    setExpandedMfrs((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // Toggle model expansion
  const toggleModel = (id: string) => {
    setExpandedModels((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // Navigate to blueprint detail
  const handleBlueprintClick = (blueprintId: string) => {
    navigate(ROUTES.BLUEPRINT_DETAIL.replace(":id", blueprintId));
  };

  // Navigate to add model page
  const handleAddModel = (_manufacturerId: string) => {
    navigate(ROUTES.BLUEPRINT_MODEL_NEW);
  };

  // Navigate to model detail to add a decoder (blueprint) to an existing model
  const handleAddDecoder = (model: DeviceModelUI) => {
    setAddBlueprintModel(model);
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
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setShowMfrDialog(true)}
        >
          {BLUEPRINT_LABELS.ADD_MANUFACTURER}
        </Button>
      </Box>

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
                isExpanded={expandedMfrs.has(mfr.id)}
                onToggle={() => toggleMfr(mfr.id)}
                expandedModels={expandedModels}
                onToggleModel={toggleModel}
                onBlueprintClick={handleBlueprintClick}
                onAddModel={() => handleAddModel(mfr.id)}
                onEdit={() => setEditMfr(mfr)}
                onDelete={() => setDeleteMfr(mfr)}
                onEditModel={(model) => setEditModel({ model, mfrId: mfr.id })}
                onDeleteModel={(model) =>
                  setDeleteModel({ model, mfrId: mfr.id })
                }
                onAddDecoder={handleAddDecoder}
              />
            ))}
          </List>
        </Paper>
      )}

      {/* Add Manufacturer Dialog */}
      <AddManufacturerDialog
        open={showMfrDialog}
        onClose={() => setShowMfrDialog(false)}
        onSuccess={() => {
          setShowMfrDialog(false);
          queryClient.invalidateQueries({
            queryKey: queryKeys.blueprints.manufacturers(),
          });
        }}
      />

      {/* Edit Manufacturer Dialog */}
      <EditManufacturerDialog
        manufacturer={editMfr}
        onClose={() => setEditMfr(null)}
        onSuccess={() => {
          setEditMfr(null);
          queryClient.invalidateQueries({
            queryKey: queryKeys.blueprints.manufacturers(),
          });
        }}
      />

      {/* Edit Device Model Dialog */}
      <EditDeviceModelDialog
        model={editModel?.model ?? null}
        manufacturerId={editModel?.mfrId ?? null}
        onClose={() => setEditModel(null)}
        onSuccess={() => {
          if (editModel) {
            queryClient.invalidateQueries({
              queryKey: queryKeys.blueprints.deviceModels(editModel.mfrId),
            });
          }
          setEditModel(null);
        }}
      />

      {/* Delete Manufacturer Confirmation */}
      <DeleteConfirmDialog
        open={!!deleteMfr}
        title={BLUEPRINT_LABELS.ACTION_DELETE}
        message={BLUEPRINT_LABELS.CONFIRM_DELETE_MANUFACTURER}
        onClose={() => setDeleteMfr(null)}
        onConfirm={async () => {
          if (deleteMfr) {
            await api.deleteManufacturer(deleteMfr.id);
            queryClient.invalidateQueries({
              queryKey: queryKeys.blueprints.manufacturers(),
            });
            setDeleteMfr(null);
          }
        }}
      />

      {/* Delete Device Model Confirmation */}
      <DeleteConfirmDialog
        open={!!deleteModel}
        title={BLUEPRINT_LABELS.ACTION_DELETE}
        message={BLUEPRINT_LABELS.CONFIRM_DELETE_MODEL}
        onClose={() => setDeleteModel(null)}
        onConfirm={async () => {
          if (deleteModel) {
            await api.deleteDeviceModel(deleteModel.model.id);
            queryClient.invalidateQueries({
              queryKey: queryKeys.blueprints.deviceModels(deleteModel.mfrId),
            });
            setDeleteModel(null);
          }
        }}
      />

      {/* Add Blueprint (Decoder) Dialog */}
      <AddBlueprintDialog
        model={addBlueprintModel}
        onClose={() => setAddBlueprintModel(null)}
        onSuccess={() => {
          if (addBlueprintModel) {
            queryClient.invalidateQueries({
              queryKey: queryKeys.blueprints.list(addBlueprintModel.id),
            });
            queryClient.invalidateQueries({
              queryKey: queryKeys.blueprints.deviceModels(
                addBlueprintModel.manufacturerId,
              ),
            });
          }
          setAddBlueprintModel(null);
        }}
      />
    </Box>
  );
};

/**
 * Manufacturer list item with collapsible models
 */
interface ManufacturerItemProps {
  manufacturer: ManufacturerUI;
  isExpanded: boolean;
  onToggle: () => void;
  expandedModels: Set<string>;
  onToggleModel: (id: string) => void;
  onBlueprintClick: (id: string) => void;
  onAddModel: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onEditModel: (model: DeviceModelUI) => void;
  onDeleteModel: (model: DeviceModelUI) => void;
  onAddDecoder: (model: DeviceModelUI) => void;
}

const ManufacturerItem: React.FC<ManufacturerItemProps> = ({
  manufacturer,
  isExpanded,
  onToggle,
  expandedModels,
  onToggleModel,
  onBlueprintClick,
  onAddModel,
  onEdit,
  onDelete,
  onEditModel,
  onDeleteModel,
  onAddDecoder,
}) => {
  // Fetch models when expanded
  const { data: models, isLoading: modelsLoading } = useQuery({
    queryKey: queryKeys.blueprints.deviceModels(manufacturer.id),
    queryFn: () => api.getDeviceModels(manufacturer.id),
    enabled: isExpanded,
  });

  return (
    <>
      <ListItem
        disablePadding
        secondaryAction={
          <Box sx={{ display: "flex", gap: 1, alignItems: "center" }}>
            <IconButton size="small" onClick={onToggle}>
              {isExpanded ? <ExpandLess /> : <ExpandMore />}
            </IconButton>
            <Tooltip title={BLUEPRINT_LABELS.ACTION_EDIT}>
              <IconButton
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  onEdit();
                }}
              >
                <EditIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title={BLUEPRINT_LABELS.ACTION_DELETE}>
              <IconButton
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete();
                }}
              >
                <DeleteIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title={BLUEPRINT_LABELS.ADD_MODEL}>
              <IconButton
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  onAddModel();
                }}
              >
                <AddIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            {manufacturer.isVerified && (
              <Chip
                label={BLUEPRINT_LABELS.BADGE_VERIFIED}
                size="small"
                color="success"
                sx={{
                  textDecoration: "none",
                  "& .MuiChip-label": { fontFamily: "inherit" },
                }}
              />
            )}
            <Chip
              label={`${models?.length ?? manufacturer.modelCount} ${BLUEPRINT_LABELS.COL_MODELS}`}
              size="small"
              sx={{
                textDecoration: "none",
                "& .MuiChip-label": { fontFamily: "inherit" },
              }}
            />
          </Box>
        }
      >
        <ListItemButton onClick={onToggle}>
          <ListItemIcon>
            <CategoryIcon />
          </ListItemIcon>
          <ListItemText
            primary={manufacturer.name}
            secondary={manufacturer.website || undefined}
          />
        </ListItemButton>
      </ListItem>

      <Collapse in={isExpanded} timeout="auto" unmountOnExit>
        <List component="div" disablePadding>
          {modelsLoading && (
            <ListItem sx={{ pl: 4 }}>
              <CircularProgress size={20} />
            </ListItem>
          )}
          {models?.map((model) => (
            <DeviceModelItem
              key={model.id}
              model={model}
              isExpanded={expandedModels.has(model.id)}
              onToggle={() => onToggleModel(model.id)}
              onBlueprintClick={onBlueprintClick}
              onEdit={() => onEditModel(model)}
              onDelete={() => onDeleteModel(model)}
              onAddDecoder={() => onAddDecoder(model)}
            />
          ))}
          {!modelsLoading && models?.length === 0 && (
            <ListItem sx={{ pl: 4 }}>
              <ListItemText
                secondary={BLUEPRINT_LABELS.NO_MODELS}
                sx={{ color: "text.secondary" }}
              />
            </ListItem>
          )}
        </List>
      </Collapse>
    </>
  );
};

/**
 * Device model list item with collapsible blueprints
 */
interface DeviceModelItemProps {
  model: DeviceModelUI;
  isExpanded: boolean;
  onToggle: () => void;
  onBlueprintClick: (id: string) => void;
  onEdit: () => void;
  onDelete: () => void;
  onAddDecoder: () => void;
}

const DeviceModelItem: React.FC<DeviceModelItemProps> = ({
  model,
  isExpanded,
  onToggle,
  onBlueprintClick,
  onEdit,
  onDelete,
  onAddDecoder,
}) => {
  // Fetch blueprints when expanded
  const { data: blueprints, isLoading: blueprintsLoading } = useQuery({
    queryKey: queryKeys.blueprints.list(model.id),
    queryFn: () => api.getBlueprints(model.id),
    enabled: isExpanded,
  });

  return (
    <>
      <ListItem
        disablePadding
        sx={{ pl: 4 }}
        secondaryAction={
          <Box sx={{ display: "flex", gap: 1, alignItems: "center" }}>
            <IconButton size="small" onClick={onToggle}>
              {isExpanded ? <ExpandLess /> : <ExpandMore />}
            </IconButton>
            <Tooltip title={BLUEPRINT_LABELS.ADD_DECODER}>
              <IconButton
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  onAddDecoder();
                }}
              >
                <AddIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title={BLUEPRINT_LABELS.ACTION_EDIT}>
              <IconButton
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  onEdit();
                }}
              >
                <EditIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title={BLUEPRINT_LABELS.ACTION_DELETE}>
              <IconButton
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete();
                }}
              >
                <DeleteIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Chip
              label={`${blueprints?.length ?? model.blueprintCount} ${BLUEPRINT_LABELS.COL_BLUEPRINTS}`}
              size="small"
              sx={{
                textDecoration: "none",
                "& .MuiChip-label": { fontFamily: "inherit" },
              }}
            />
          </Box>
        }
      >
        <ListItemButton onClick={onToggle}>
          <ListItemIcon>
            <BlueprintIcon />
          </ListItemIcon>
          <ListItemText
            primary={model.name}
            secondary={
              model.typeEui ? formatTypeEUI(model.typeEui) : model.code
            }
          />
        </ListItemButton>
      </ListItem>

      <Collapse in={isExpanded} timeout="auto" unmountOnExit>
        <List component="div" disablePadding>
          {blueprintsLoading && (
            <ListItem sx={{ pl: 8 }}>
              <CircularProgress size={16} />
            </ListItem>
          )}
          {blueprints?.map((bp) => (
            <ListItem key={bp.id} disablePadding sx={{ pl: 8 }}>
              <ListItemButton onClick={() => onBlueprintClick(bp.id)}>
                <ListItemText
                  primary={`v${bp.version}`}
                  secondary={formatTypeEUI(bp.typeEui)}
                />
                {bp.isDefault && (
                  <Chip
                    label={BLUEPRINT_LABELS.BADGE_DEFAULT}
                    size="small"
                    color="primary"
                  />
                )}
              </ListItemButton>
            </ListItem>
          ))}
          {!blueprintsLoading && blueprints?.length === 0 && (
            <ListItem sx={{ pl: 8 }}>
              <ListItemText
                secondary={BLUEPRINT_LABELS.NO_BLUEPRINTS}
                sx={{ color: "text.secondary" }}
              />
            </ListItem>
          )}
        </List>
      </Collapse>
    </>
  );
};

/**
 * Add Manufacturer Dialog
 */
interface AddManufacturerDialogProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

const AddManufacturerDialog: React.FC<AddManufacturerDialogProps> = ({
  open,
  onClose,
  onSuccess,
}) => {
  const [formData, setFormData] = useState<CreateManufacturerRequest>({
    name: "",
  });
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: (data: CreateManufacturerRequest) =>
      api.createManufacturer(data),
    onSuccess: () => {
      setFormData({ name: "" });
      setError(null);
      onSuccess();
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  const handleSubmit = () => {
    if (!formData.name) {
      setError(BLUEPRINT_LABELS.ERR_NAME_REQUIRED);
      return;
    }
    mutation.mutate(formData);
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{BLUEPRINT_LABELS.ADD_MANUFACTURER}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <TextField
          autoFocus
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_NAME}
          fullWidth
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
        />
        <TextField
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_WEBSITE}
          fullWidth
          value={formData.website || ""}
          onChange={(e) =>
            setFormData({ ...formData, website: e.target.value })
          }
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{BLUEPRINT_LABELS.ACTION_CANCEL}</Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={mutation.isPending}
        >
          {mutation.isPending ? (
            <CircularProgress size={20} />
          ) : (
            BLUEPRINT_LABELS.ACTION_CREATE
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

/**
 * Edit Manufacturer Dialog
 */
interface EditManufacturerDialogProps {
  manufacturer: ManufacturerUI | null;
  onClose: () => void;
  onSuccess: () => void;
}

const EditManufacturerDialog: React.FC<EditManufacturerDialogProps> = ({
  manufacturer,
  onClose,
  onSuccess,
}) => {
  const [formData, setFormData] = useState<UpdateManufacturerRequest>({});
  const [error, setError] = useState<string | null>(null);

  // Reset form data when manufacturer changes
  React.useEffect(() => {
    if (manufacturer) {
      setFormData({
        name: manufacturer.name,
        website: manufacturer.website || undefined,
      });
      setError(null);
    }
  }, [manufacturer]);

  const mutation = useMutation({
    mutationFn: (data: UpdateManufacturerRequest) => {
      if (!manufacturer)
        throw new Error(BLUEPRINT_LABELS.ERR_NO_MANUFACTURER_SELECTED);
      return api.updateManufacturer(manufacturer.id, data);
    },
    onSuccess: () => {
      setError(null);
      onSuccess();
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  const handleSubmit = () => {
    if (!formData.name) {
      setError(BLUEPRINT_LABELS.ERR_NAME_REQUIRED);
      return;
    }
    mutation.mutate(formData);
  };

  return (
    <Dialog open={!!manufacturer} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{BLUEPRINT_LABELS.DIALOG_EDIT_MANUFACTURER}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <TextField
          autoFocus
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_NAME}
          fullWidth
          value={formData.name || ""}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
        />
        <TextField
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_WEBSITE}
          fullWidth
          value={formData.website || ""}
          onChange={(e) =>
            setFormData({ ...formData, website: e.target.value })
          }
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{BLUEPRINT_LABELS.ACTION_CANCEL}</Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={mutation.isPending}
        >
          {mutation.isPending ? (
            <CircularProgress size={20} />
          ) : (
            BLUEPRINT_LABELS.ACTION_SAVE
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

/**
 * Edit Device Model Dialog
 */
interface EditDeviceModelDialogProps {
  model: DeviceModelUI | null;
  manufacturerId: string | null;
  onClose: () => void;
  onSuccess: () => void;
}

const EditDeviceModelDialog: React.FC<EditDeviceModelDialogProps> = ({
  model,
  manufacturerId: _manufacturerId,
  onClose,
  onSuccess,
}) => {
  const [formData, setFormData] = useState<UpdateDeviceModelRequest>({});
  const [error, setError] = useState<string | null>(null);

  // Reset form data when model changes
  React.useEffect(() => {
    if (model) {
      setFormData({
        name: model.name,
        description: model.description || undefined,
        datasheetUrl: model.datasheetUrl || undefined,
      });
      setError(null);
    }
  }, [model]);

  const mutation = useMutation({
    mutationFn: (data: UpdateDeviceModelRequest) => {
      if (!model) throw new Error(BLUEPRINT_LABELS.ERR_NO_MODEL_SELECTED);
      return api.updateDeviceModel(model.id, data);
    },
    onSuccess: () => {
      setError(null);
      onSuccess();
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  const handleSubmit = () => {
    if (!formData.name) {
      setError(BLUEPRINT_LABELS.ERR_NAME_REQUIRED);
      return;
    }
    mutation.mutate(formData);
  };

  return (
    <Dialog open={!!model} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{BLUEPRINT_LABELS.DIALOG_EDIT_MODEL}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <TextField
          autoFocus
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_NAME}
          fullWidth
          value={formData.name || ""}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
        />
        <TextField
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_DESCRIPTION}
          fullWidth
          multiline
          rows={2}
          value={formData.description || ""}
          onChange={(e) =>
            setFormData({ ...formData, description: e.target.value })
          }
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{BLUEPRINT_LABELS.ACTION_CANCEL}</Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={mutation.isPending}
        >
          {mutation.isPending ? (
            <CircularProgress size={20} />
          ) : (
            BLUEPRINT_LABELS.ACTION_SAVE
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

/**
 * Add Blueprint (Decoder) Dialog
 */
interface AddBlueprintDialogProps {
  model: DeviceModelUI | null;
  onClose: () => void;
  onSuccess: () => void;
}

const AddBlueprintDialog: React.FC<AddBlueprintDialogProps> = ({
  model,
  onClose,
  onSuccess,
}) => {
  const [version, setVersion] = useState("");
  const [specJson, setSpecJson] = useState("");
  const [error, setError] = useState<string | null>(null);

  React.useEffect(() => {
    if (model) {
      setVersion("");
      setSpecJson("");
      setError(null);
    }
  }, [model]);

  const mutation = useMutation({
    mutationFn: async () => {
      if (!model) throw new Error(BLUEPRINT_LABELS.ERR_CREATE_BLUEPRINT);

      if (!version.trim()) {
        throw new Error(BLUEPRINT_LABELS.ERR_VERSION_REQUIRED);
      }
      if (!specJson.trim()) {
        throw new Error(BLUEPRINT_LABELS.ERR_SPEC_JSON_REQUIRED);
      }

      let parsedSpec: object;
      try {
        parsedSpec = JSON.parse(specJson);
      } catch {
        throw new Error(BLUEPRINT_LABELS.ERR_INVALID_JSON);
      }

      await api.createBlueprint(model.id, {
        version: version.trim(),
        specJson: parsedSpec,
      });
    },
    onSuccess: () => {
      setError(null);
      onSuccess();
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  return (
    <Dialog open={!!model} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{BLUEPRINT_LABELS.ADD_DECODER}</DialogTitle>
      <DialogContent>
        <DialogContentText sx={{ mb: 2 }}>
          {model ? `${BLUEPRINT_LABELS.LABEL_NAME}: ${model.name}` : ""}
        </DialogContentText>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <TextField
          autoFocus
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_VERSION}
          fullWidth
          value={version}
          onChange={(e) => setVersion(e.target.value)}
        />
        <TextField
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_SPEC_JSON}
          fullWidth
          multiline
          rows={10}
          value={specJson}
          onChange={(e) => setSpecJson(e.target.value)}
          helperText={BLUEPRINT_LABELS.HELPER_SPEC_JSON}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={mutation.isPending}>
          {BLUEPRINT_LABELS.ACTION_CANCEL}
        </Button>
        <Button
          onClick={() => mutation.mutate()}
          variant="contained"
          disabled={mutation.isPending}
        >
          {mutation.isPending ? (
            <CircularProgress size={20} />
          ) : (
            BLUEPRINT_LABELS.ACTION_CREATE
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

/**
 * Delete Confirmation Dialog
 */
interface DeleteConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  onClose: () => void;
  onConfirm: () => Promise<void>;
}

const DeleteConfirmDialog: React.FC<DeleteConfirmDialogProps> = ({
  open,
  title,
  message,
  onClose,
  onConfirm,
}) => {
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleConfirm = async () => {
    setIsDeleting(true);
    setError(null);
    try {
      await onConfirm();
    } catch (err) {
      setError(
        err instanceof Error ? err.message : BLUEPRINT_LABELS.ERR_DELETE_FAILED,
      );
      setIsDeleting(false);
    }
  };

  // Reset state when dialog closes
  React.useEffect(() => {
    if (!open) {
      setIsDeleting(false);
      setError(null);
    }
  }, [open]);

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        <DialogContentText>{message}</DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={isDeleting}>
          {BLUEPRINT_LABELS.ACTION_CANCEL}
        </Button>
        <Button
          onClick={handleConfirm}
          color="error"
          variant="contained"
          disabled={isDeleting}
        >
          {isDeleting ? (
            <CircularProgress size={20} />
          ) : (
            BLUEPRINT_LABELS.ACTION_DELETE
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default Blueprints;
