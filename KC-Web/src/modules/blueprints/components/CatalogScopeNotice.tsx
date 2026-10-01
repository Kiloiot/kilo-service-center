/**
 * States which catalog a create form adds to; the catalog is the open tab.
 */

import React from "react";

import type { BlueprintScope } from "@api-types/api";
import { Alert } from "@mui/material";

import { BLUEPRINT_SCOPE } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";

const CatalogScopeNotice: React.FC<{ scope: BlueprintScope }> = ({ scope }) => (
  <Alert severity="info" sx={{ mt: 2 }}>
    {scope === BLUEPRINT_SCOPE.SYSTEM
      ? BLUEPRINT_LABELS.CREATES_IN_SYSTEM
      : BLUEPRINT_LABELS.CREATES_IN_CUSTOM}
  </Alert>
);

export default CatalogScopeNotice;
