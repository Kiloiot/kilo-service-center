/**
 * The blueprint an endpoint's uplinks are decoded with: the manufacturer and
 * model it is bound to, or none with the action that assigns one.
 */

import React from "react";

import { Box, Button, Typography } from "@mui/material";

import { ENDPOINT_DETAILS } from "@constants/messages";
import { BlueprintIcon } from "@theme/icons";

export interface BlueprintInfo {
  manufacturerName: string;
  modelName: string;
}

function LabeledValue({ label, value }: { label: string; value: string }) {
  return (
    <Box>
      <Typography variant="body2" color="text.secondary">
        {label}
      </Typography>
      <Typography variant="body1">{value}</Typography>
    </Box>
  );
}

function BoundModel({ blueprintInfo }: { blueprintInfo: BlueprintInfo }) {
  return (
    <Box display="flex" gap={4}>
      <LabeledValue
        label={ENDPOINT_DETAILS.OPTION_SELECT_MANUFACTURER}
        value={blueprintInfo.manufacturerName}
      />
      <LabeledValue
        label={ENDPOINT_DETAILS.OPTION_SELECT_MODEL}
        value={blueprintInfo.modelName}
      />
    </Box>
  );
}

function NoModel({ onAssign }: { onAssign: () => void }) {
  return (
    <Box display="flex" alignItems="center" gap={2}>
      <Typography variant="body1">{ENDPOINT_DETAILS.BLUEPRINT_NONE}</Typography>
      <Button
        variant="outlined"
        size="small"
        startIcon={<BlueprintIcon />}
        onClick={onAssign}
      >
        {ENDPOINT_DETAILS.ACTION_ASSIGN_BLUEPRINT}
      </Button>
    </Box>
  );
}

export const EndpointBlueprintSection: React.FC<{
  blueprintInfo: BlueprintInfo | null;
  onAssign: () => void;
}> = ({ blueprintInfo, onAssign }) => (
  <>
    <Typography
      variant="subtitle2"
      color="text.secondary"
      gutterBottom
      sx={{ mt: 1, display: "flex", alignItems: "center", gap: 0.5 }}
    >
      <BlueprintIcon fontSize="small" />
      {ENDPOINT_DETAILS.SECTION_BLUEPRINT_INFO}
    </Typography>
    {blueprintInfo ? (
      <BoundModel blueprintInfo={blueprintInfo} />
    ) : (
      <NoModel onAssign={onAssign} />
    )}
  </>
);
