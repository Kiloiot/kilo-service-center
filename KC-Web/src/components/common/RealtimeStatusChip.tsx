import React from "react";

import { useConnectionStatus } from "@hooks";
import { Chip } from "@mui/material";

import type { UIConnectionStatus } from "@hooks/useConnectionStatus";
import { DASHBOARD_PAGE } from "@constants/messages";
import { TimeIcon } from "@theme/icons";
import { livePulseAnimation } from "@theme/index";

const STATUS_CHIP: Record<
  UIConnectionStatus,
  { label: string; color: "success" | "warning" | "error" }
> = {
  connected: { label: DASHBOARD_PAGE.LIVE, color: "success" },
  reconnecting: {
    label: DASHBOARD_PAGE.REALTIME_RECONNECTING,
    color: "warning",
  },
  offline: { label: DASHBOARD_PAGE.REALTIME_OFFLINE, color: "error" },
};

/** Whether the page updates live: the state of the realtime connection. */
export const RealtimeStatusChip: React.FC = () => {
  const { status, isConnected } = useConnectionStatus();
  const chip = STATUS_CHIP[status];
  return (
    <Chip
      icon={<TimeIcon />}
      label={chip.label}
      color={chip.color}
      size="small"
      sx={isConnected ? { animation: livePulseAnimation } : undefined}
    />
  );
};
