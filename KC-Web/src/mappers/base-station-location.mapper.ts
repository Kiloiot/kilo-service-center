/**
 * Base station location mapper: a located station as a dashboard map pin.
 */

import type {
  BaseStationLocationDTO,
  BaseStationLocationUI,
} from "@api-types/api";

import { BASE_STATION_STATUS } from "@constants/app";

export function mapBaseStationLocation(
  dto: BaseStationLocationDTO,
): BaseStationLocationUI {
  return {
    eui: dto.bsEui,
    name: dto.name || undefined,
    latitude: dto.latitude,
    longitude: dto.longitude,
    status: dto.isOnline
      ? BASE_STATION_STATUS.ONLINE
      : BASE_STATION_STATUS.OFFLINE,
  };
}
