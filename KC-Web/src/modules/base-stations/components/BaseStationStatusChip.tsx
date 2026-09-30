/**
 * A base station's online/offline state as a chip.
 */

import type { BaseStationUI } from "@api-types/api";
import { Chip } from "@mui/material";

import { BASE_STATION_STATUS } from "@constants/app";
import { BASE_STATIONS_PAGE } from "@constants/messages";

interface BaseStationStatusChipProps {
  status: BaseStationUI["status"];
}

export function BaseStationStatusChip({ status }: BaseStationStatusChipProps) {
  return (
    <Chip
      label={
        status === BASE_STATION_STATUS.ONLINE
          ? BASE_STATIONS_PAGE.ONLINE
          : BASE_STATIONS_PAGE.OFFLINE
      }
      color={status === BASE_STATION_STATUS.ONLINE ? "success" : "default"}
      size="small"
    />
  );
}
