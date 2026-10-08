/**
 * Submit to Registry, offered only while the service center has a blueprint
 * registry configured; otherwise disabled with the reason.
 */

import React from "react";

import { Box, Button, Tooltip } from "@mui/material";

import { useServerCapability } from "@hooks/useServerCapability";
import { SERVER_CAPABILITY } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { PublishIcon } from "@theme/icons";

const RegistrySubmitButton: React.FC<{ onSubmit: () => void }> = ({
  onSubmit,
}) => {
  const registryConfigured = useServerCapability(
    SERVER_CAPABILITY.BLUEPRINT_REGISTRY_SUBMISSION,
  );
  const unavailable = registryConfigured === false;
  return (
    <Tooltip
      title={unavailable ? BLUEPRINT_LABELS.REGISTRY_SUBMIT_UNAVAILABLE : ""}
    >
      {/* A disabled button fires no events, so the tooltip hangs on its wrapper. */}
      <Box component="span">
        <Button
          variant="outlined"
          startIcon={<PublishIcon />}
          onClick={onSubmit}
          disabled={registryConfigured !== true}
        >
          {BLUEPRINT_LABELS.SUBMIT_TO_REGISTRY}
        </Button>
      </Box>
    </Tooltip>
  );
};

export default RegistrySubmitButton;
