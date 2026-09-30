import React from "react";

import { Chip } from "@mui/material";

import { attachStatusChip } from "@utils/chipMappings";

interface AttachStatusChipProps {
  status?: string;
}

/** An end point's attach state, labeled and colored by the shared mapping. */
const AttachStatusChip: React.FC<AttachStatusChipProps> = ({ status }) => {
  const { label, color } = attachStatusChip(status);
  return <Chip label={label} color={color} size="small" />;
};

export default AttachStatusChip;
