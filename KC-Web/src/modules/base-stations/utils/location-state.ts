/**
 * What a base station's stored location says: a GPS fix, GPS reporting no
 * fix, a manual position, or nothing.
 */

import {
  LOCATION_SOURCE,
  STATION_LOCATION_STATE,
  type StationLocationState,
} from "@constants/app";

export interface StationLocation {
  locationSource?: string | null;
  latitude?: number | null;
  longitude?: number | null;
}

export function stationLocationState(
  location: StationLocation | null | undefined,
): StationLocationState {
  const hasPosition = location?.latitude != null && location?.longitude != null;
  if (location?.locationSource === LOCATION_SOURCE.GPS) {
    return hasPosition
      ? STATION_LOCATION_STATE.GPS_FIX
      : STATION_LOCATION_STATE.GPS_NO_FIX;
  }
  return hasPosition
    ? STATION_LOCATION_STATE.MANUAL
    : STATION_LOCATION_STATE.NONE;
}
