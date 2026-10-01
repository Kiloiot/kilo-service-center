/**
 * Explains, on the System catalog, that the blueprint registry is not configured.
 */

import React from "react";

import { Alert } from "@mui/material";

import { useServerCapability } from "@hooks/useServerCapability";
import { SERVER_CAPABILITY } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";

const RegistryStatusNotice: React.FC = () => {
  const registryConfigured = useServerCapability(
    SERVER_CAPABILITY.BLUEPRINT_REGISTRY_SUBMISSION,
  );
  if (registryConfigured !== false) return null;
  return (
    <Alert severity="info" sx={{ mb: 2 }}>
      {BLUEPRINT_LABELS.REGISTRY_NOT_CONFIGURED}
    </Alert>
  );
};

export default RegistryStatusNotice;
