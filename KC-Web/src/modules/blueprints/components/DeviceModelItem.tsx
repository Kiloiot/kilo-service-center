/**
 * Device model row of the catalog tree with its blueprints.
 */

import React, { useState } from "react";

import type { BlueprintScope, DeviceModelUI } from "@api-types/api";
import {
  Box,
  Button,
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

import { formatEui } from "@utils/eui";
import { BLUEPRINT_LABELS } from "@constants/messages";
import {
  AddIcon,
  BlueprintIcon,
  DeleteIcon,
  EditIcon,
  ExpandLess,
  ExpandMore,
} from "@theme/icons";
import { componentSpacing } from "@theme/index";

import { useBlueprints } from "../hooks";
import BulkMigrateDialog from "./BulkMigrateDialog";

/**
 * Device model list item with collapsible blueprints
 */
export interface DeviceModelItemProps {
  model: DeviceModelUI;
  scope: BlueprintScope;
  isServerAdmin: boolean;
  isExpanded: boolean;
  onToggle: () => void;
  onBlueprintClick: (id: string) => void;
  onEdit: () => void;
  onDelete: () => void;
  onAddDecoder: () => void;
}

export const DeviceModelItem: React.FC<DeviceModelItemProps> = ({
  model,
  scope,
  isServerAdmin,
  isExpanded,
  onToggle,
  onBlueprintClick,
  onEdit,
  onDelete,
  onAddDecoder,
}) => {
  const canMutate = model.isSystem ? isServerAdmin : true;
  const [showMigrate, setShowMigrate] = useState(false);

  // Fetch blueprints when expanded
  const { data: blueprints, isLoading: blueprintsLoading } = useBlueprints(
    model.id,
    scope,
    { enabled: isExpanded },
  );

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
            <Button
              size="small"
              onClick={(e) => {
                e.stopPropagation();
                setShowMigrate(true);
              }}
            >
              {BLUEPRINT_LABELS.MIGRATE_DEVICES}
            </Button>
            {canMutate && (
              <>
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
              </>
            )}
            {model.isSystem && (
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
            secondary={model.typeEui ? formatEui(model.typeEui) : model.code}
          />
        </ListItemButton>
      </ListItem>

      <Collapse in={isExpanded} timeout="auto" unmountOnExit>
        <List component="div" disablePadding>
          {blueprintsLoading && (
            <ListItem sx={{ pl: 8 }}>
              <CircularProgress size={componentSpacing.spinner.inline} />
            </ListItem>
          )}
          {blueprints?.map((bp) => (
            <ListItem key={bp.id} disablePadding sx={{ pl: 8 }}>
              <ListItemButton onClick={() => onBlueprintClick(bp.id)}>
                <ListItemText
                  primary={bp.version}
                  secondary={formatEui(bp.typeEui)}
                />
                <Box sx={{ display: "flex", gap: 1 }}>
                  {bp.isSystem && (
                    <Chip
                      label={BLUEPRINT_LABELS.BADGE_SYSTEM}
                      size="small"
                      color="info"
                    />
                  )}
                  {bp.isDefault && (
                    <Chip
                      label={BLUEPRINT_LABELS.BADGE_DEFAULT}
                      size="small"
                      color="primary"
                    />
                  )}
                </Box>
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

      <BulkMigrateDialog
        open={showMigrate}
        deviceModelId={model.id}
        scope={scope}
        modelIsSystem={model.isSystem}
        onClose={() => setShowMigrate(false)}
      />
    </>
  );
};
