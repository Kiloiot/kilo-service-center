import React from "react";
import { useNavigate } from "react-router-dom";

import type { BlueprintUI } from "@api-types/api";
import { Box, Button, Chip, Typography } from "@mui/material";

import { BackButton } from "@components/common/BackButton";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { EditIcon, SaveIcon } from "@theme/icons";

import { catalogPath } from "../utils/catalog-routes";
import { catalogScopeOf } from "../utils/scope";
import RegistrySubmitButton from "./RegistrySubmitButton";

interface BlueprintDetailHeaderProps {
  blueprint: BlueprintUI;
  canMutate: boolean;
  isEditing: boolean;
  isSavePending: boolean;
  isSetDefaultPending: boolean;
  onStartEdit: () => void;
  onSave: () => void;
  onCancelEdit: () => void;
  onSetDefault: () => void;
  onSubmitToRegistry: () => void;
  onMigrate: () => void;
}

const BlueprintDetailHeader: React.FC<BlueprintDetailHeaderProps> = ({
  blueprint,
  canMutate,
  isEditing,
  isSavePending,
  isSetDefaultPending,
  onStartEdit,
  onSave,
  onCancelEdit,
  onSetDefault,
  onSubmitToRegistry,
  onMigrate,
}) => {
  const navigate = useNavigate();
  return (
    <Box sx={{ display: "flex", alignItems: "center", gap: 2, mb: 3 }}>
      <BackButton
        label={BLUEPRINT_LABELS.BACK_TO_BLUEPRINTS}
        onClick={() => navigate(catalogPath(catalogScopeOf(blueprint)))}
      />
      <Typography variant="h4">
        {BLUEPRINT_LABELS.BLUEPRINT_VERSION_PREFIX}
        {blueprint.version}
      </Typography>
      {blueprint.isSystem && (
        <Chip label={BLUEPRINT_LABELS.BADGE_SYSTEM} color="info" />
      )}
      {blueprint.isDefault && (
        <Chip label={BLUEPRINT_LABELS.BADGE_DEFAULT} color="primary" />
      )}
      {blueprint.registryVerified && (
        <Chip label={BLUEPRINT_LABELS.BADGE_VERIFIED} color="success" />
      )}
      <Box sx={{ flex: 1 }} />
      {!isEditing && (
        <>
          <Button onClick={onMigrate}>
            {BLUEPRINT_LABELS.MIGRATE_DEVICES}
          </Button>
          {canMutate && (
            <Button startIcon={<EditIcon />} onClick={onStartEdit}>
              {BLUEPRINT_LABELS.ACTION_EDIT}
            </Button>
          )}
          {canMutate && !blueprint.isDefault && (
            <Button
              variant="outlined"
              onClick={onSetDefault}
              disabled={isSetDefaultPending}
            >
              {BLUEPRINT_LABELS.SET_DEFAULT}
            </Button>
          )}
          {!blueprint.registryVerified && (
            <RegistrySubmitButton onSubmit={onSubmitToRegistry} />
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

export default BlueprintDetailHeader;
