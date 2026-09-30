/**
 * Manufacturer row of the catalog tree with its device models.
 */

import React from "react";

import type {
  BlueprintScope,
  DeviceModelUI,
  ManufacturerUI,
} from "@api-types/api";
import {
  Box,
  Chip,
  CircularProgress,
  Collapse,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Tooltip,
} from "@mui/material";

import { BLUEPRINT_LABELS } from "@constants/messages";
import {
  AddIcon,
  CategoryIcon,
  DeleteIcon,
  EditIcon,
  ExpandLess,
  ExpandMore,
} from "@theme/icons";
import { componentSpacing } from "@theme/index";

import { useDeviceModels } from "../hooks";
import { DeviceModelItem } from "./DeviceModelItem";

/**
 * Manufacturer list item with collapsible models
 */
export interface ManufacturerItemProps {
  manufacturer: ManufacturerUI;
  scope: BlueprintScope;
  isServerAdmin: boolean;
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

export const ManufacturerItem: React.FC<ManufacturerItemProps> = ({
  manufacturer,
  scope,
  isServerAdmin,
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
  // System rows are mutable only by server admins; Custom rows by their tenant.
  const canMutate = manufacturer.isSystem ? isServerAdmin : true;

  // Fetch models when expanded
  const { data: models, isLoading: modelsLoading } = useDeviceModels(
    manufacturer.id,
    scope,
    { enabled: isExpanded },
  );

  return (
    <>
      <ListItem
        disablePadding
        secondaryAction={
          <Box sx={{ display: "flex", gap: 1, alignItems: "center" }}>
            <IconButton size="small" onClick={onToggle}>
              {isExpanded ? <ExpandLess /> : <ExpandMore />}
            </IconButton>
            {canMutate && (
              <>
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
              </>
            )}
            {manufacturer.isSystem && (
              <Chip
                label={BLUEPRINT_LABELS.BADGE_SYSTEM}
                size="small"
                color="info"
                sx={{
                  textDecoration: "none",
                  "& .MuiChip-label": { fontFamily: "inherit" },
                }}
              />
            )}
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
              <CircularProgress size={componentSpacing.spinner.button} />
            </ListItem>
          )}
          {models?.map((model) => (
            <DeviceModelItem
              key={model.id}
              model={model}
              scope={scope}
              isServerAdmin={isServerAdmin}
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
