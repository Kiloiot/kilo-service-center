import React from "react";

import { Chip, type ChipProps } from "@mui/material";

import { DataCard } from "@components/common/DataCard";
import { InfoGrid } from "@components/common/InfoGrid";
import { useScaciSessions, useScaciStatus } from "@hooks/useScaci";
import { formatDateTime } from "@utils/date-format";
import { formatEui } from "@utils/eui";
import {
  formatMilliseconds,
  formatOpIdRange,
  formatOptional,
  formatYesNo,
} from "@utils/formatters";
import { CONTROL_PLANE_STRIP } from "@constants/messages";

import {
  CONTROL_PLANE_STATE,
  type ControlPlaneState,
  controlPlaneState,
} from "../utils/control-plane";

const LATEST_SESSION = { page: 0, pageSize: 1 };

const STATE_CHIP: Record<
  ControlPlaneState,
  { label: string; color: ChipProps["color"] }
> = {
  [CONTROL_PLANE_STATE.CONNECTED]: {
    label: CONTROL_PLANE_STRIP.CONNECTED,
    color: "success",
  },
  [CONTROL_PLANE_STATE.DISCONNECTED]: {
    label: CONTROL_PLANE_STRIP.DISCONNECTED,
    color: "warning",
  },
  [CONTROL_PLANE_STATE.LISTENER_OFFLINE]: {
    label: CONTROL_PLANE_STRIP.LISTENER_OFFLINE,
    color: "error",
  },
};

/** NAV 1 control-plane strip: the SC-AC session at a glance (GetScaciStatus, ListScaciSessions). */
export const ControlPlaneStrip: React.FC = () => {
  const status = useScaciStatus();
  const sessions = useScaciSessions(LATEST_SESSION);
  const latest = sessions.data?.items[0];

  return (
    <DataCard
      title={CONTROL_PLANE_STRIP.TITLE}
      isLoading={status.isLoading || sessions.isLoading}
      error={status.error ?? sessions.error}
      errorFallback={CONTROL_PLANE_STRIP.ERR_LOAD}
    >
      {status.data && (
        <InfoGrid
          items={[
            {
              label: CONTROL_PLANE_STRIP.SESSION_STATE,
              value: (
                <Chip
                  size="small"
                  {...STATE_CHIP[controlPlaneState(status.data, latest)]}
                />
              ),
            },
            {
              label: CONTROL_PLANE_STRIP.VERSION,
              value: formatOptional(
                latest?.version ?? status.data.protocolVersion,
              ),
            },
            {
              label: CONTROL_PLANE_STRIP.SC_EUI,
              value: status.data.scEui
                ? formatEui(status.data.scEui)
                : formatOptional(),
            },
            {
              label: CONTROL_PLANE_STRIP.AC_EUI,
              value: latest?.acEui ? formatEui(latest.acEui) : formatOptional(),
            },
            {
              label: CONTROL_PLANE_STRIP.SN_RESUME,
              value: latest ? formatYesNo(latest.snResume) : formatOptional(),
            },
            {
              label: CONTROL_PLANE_STRIP.SN_AC_UUID,
              value: formatOptional(latest?.snAcUuid),
            },
            {
              label: CONTROL_PLANE_STRIP.SN_SC_UUID,
              value: formatOptional(latest?.snScUuid),
            },
            {
              label: CONTROL_PLANE_STRIP.LAST_CON,
              value: formatOptional(status.data.lastConnectResult),
            },
            {
              label: CONTROL_PLANE_STRIP.LAST_PING,
              value: formatDateTime(status.data.lastPingAt),
            },
            {
              label: CONTROL_PLANE_STRIP.RTT,
              value: formatMilliseconds(status.data.lastPingRttMs),
            },
            {
              label: CONTROL_PLANE_STRIP.MISSED_HEARTBEATS,
              value: status.data.missedPings,
            },
            {
              label: CONTROL_PLANE_STRIP.OP_ID_RANGE,
              value: latest
                ? formatOpIdRange(latest.maxAcOpId, latest.minScOpId)
                : formatOptional(),
            },
          ]}
        />
      )}
    </DataCard>
  );
};
