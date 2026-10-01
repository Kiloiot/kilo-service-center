import React from "react";

import { Chip, Tooltip } from "@mui/material";

import { ENDPOINT_DETAILS } from "@constants/messages";

interface ReattachPendingBadgeProps {
  pending?: boolean;
}

/** Marks an end point whose base stations still hold an earlier configuration. */
const ReattachPendingBadge: React.FC<ReattachPendingBadgeProps> = ({
  pending,
}) =>
  pending ? (
    <Tooltip title={ENDPOINT_DETAILS.TOOLTIP_REATTACH_PENDING}>
      <Chip
        label={ENDPOINT_DETAILS.BADGE_REATTACH_PENDING}
        color="warning"
        variant="outlined"
        size="small"
      />
    </Tooltip>
  ) : null;

export default ReattachPendingBadge;
