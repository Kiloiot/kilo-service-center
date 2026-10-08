/**
 * Location rules of the base station edit form: validation, the update
 * payload and change detection. A GPS fix is read-only; a station whose GPS
 * reports no fix takes a manual position.
 */

import { isApiError } from "@api-types/api";

import { formatEui } from "@utils/eui";
import { GEO_BOUNDS, STATION_LOCATION_STATE } from "@constants/app";
import {
  ERR_BS_EUI_EXISTS,
  ERR_BS_NOT_FOUND,
  ERR_UPDATE_BS,
  ERR_UPDATE_BS_EUI,
  VAL_BS_EUI_FORMAT,
  VAL_LAT_LON_PAIR,
  VAL_LATITUDE_RANGE,
  VAL_LONGITUDE_RANGE,
} from "@constants/messages";

import { stationLocationState } from "./location-state";

export interface BaseStationEditFormData {
  name: string;
  eui: string;
  latitude: string;
  longitude: string;
  altitude: string;
}

export interface BaseStationDetailsSnapshot {
  locationSource?: string;
  latitude?: number | null;
  longitude?: number | null;
  altitude?: number | null;
}

/**
 * Validates the manual-location fields. GPS-sourced rows are read-only so
 * an empty errors map is returned. Returns sparse errors map keyed by
 * "latitude" / "longitude".
 */
export function validateLocationForm(
  form: BaseStationEditFormData,
  isGps: boolean,
): Record<string, string> {
  if (isGps) return {};
  const errs: Record<string, string> = {};
  const hasLat = form.latitude.trim() !== "";
  const hasLng = form.longitude.trim() !== "";
  if (hasLat !== hasLng) {
    errs.latitude = VAL_LAT_LON_PAIR;
    errs.longitude = VAL_LAT_LON_PAIR;
  }
  if (hasLat) {
    const lat = parseFloat(form.latitude);
    if (
      isNaN(lat) ||
      lat < GEO_BOUNDS.LATITUDE_MIN ||
      lat > GEO_BOUNDS.LATITUDE_MAX
    ) {
      errs.latitude = VAL_LATITUDE_RANGE;
    }
  }
  if (hasLng) {
    const lng = parseFloat(form.longitude);
    if (
      isNaN(lng) ||
      lng < GEO_BOUNDS.LONGITUDE_MIN ||
      lng > GEO_BOUNDS.LONGITUDE_MAX
    ) {
      errs.longitude = VAL_LONGITUDE_RANGE;
    }
  }
  return errs;
}

/**
 * Builds the location update payload. number = set, null = clear,
 * undefined-key = omit. A GPS fix always returns {} (do not modify).
 */
export function buildLocationUpdateData(
  form: BaseStationEditFormData,
  details: BaseStationDetailsSnapshot | null | undefined,
): {
  latitude?: number | null;
  longitude?: number | null;
  altitude?: number | null;
} {
  if (stationLocationState(details) === STATION_LOCATION_STATE.GPS_FIX)
    return {};
  const hasLat = form.latitude.trim() !== "";
  const hasLng = form.longitude.trim() !== "";
  const hadLat = details?.latitude != null;
  if (!hasLat && !hasLng && hadLat) {
    return { latitude: null, longitude: null, altitude: null };
  }
  if (hasLat && hasLng) {
    const data: {
      latitude: number;
      longitude: number;
      altitude?: number | null;
    } = {
      latitude: parseFloat(form.latitude),
      longitude: parseFloat(form.longitude),
    };
    if (form.altitude.trim()) {
      data.altitude = parseFloat(form.altitude);
    } else if (details?.altitude != null) {
      data.altitude = null;
    }
    return data;
  }
  return {};
}

/**
 * Returns true when the form's lat/lng/altitude differ from the fetched
 * details. A GPS fix always returns false (no manual edits propagate).
 */
export function hasLocationChanged(
  form: BaseStationEditFormData,
  details: BaseStationDetailsSnapshot | null | undefined,
): boolean {
  if (stationLocationState(details) === STATION_LOCATION_STATE.GPS_FIX)
    return false;
  const detailLat = details?.latitude != null ? String(details.latitude) : "";
  const detailLng = details?.longitude != null ? String(details.longitude) : "";
  const detailAlt = details?.altitude != null ? String(details.altitude) : "";
  return (
    form.latitude.trim() !== detailLat ||
    form.longitude.trim() !== detailLng ||
    form.altitude.trim() !== detailAlt
  );
}

const coordinate = (value: number | null | undefined): string =>
  value != null ? String(value) : "";

/** The edit form as the dialog opens: the station's name, EUI and location. */
export function initialEditForm(
  name: string | undefined,
  eui: string,
  details: BaseStationDetailsSnapshot | null | undefined,
): BaseStationEditFormData {
  return {
    name: name || "",
    eui: formatEui(eui),
    latitude: coordinate(details?.latitude),
    longitude: coordinate(details?.longitude),
    altitude: coordinate(details?.altitude),
  };
}

/** The message for a failed save of the name or location. */
export function updateFailure(error: unknown): string {
  return isApiError(error) && error.isNotFound()
    ? ERR_BS_NOT_FOUND
    : ERR_UPDATE_BS;
}

/** The message for a failed EUI change, and whether it belongs on the EUI field. */
export function euiUpdateFailure(error: unknown): {
  message: string;
  onEui: boolean;
} {
  if (!isApiError(error)) return { message: ERR_UPDATE_BS_EUI, onEui: false };
  if (error.isAlreadyExists())
    return { message: ERR_BS_EUI_EXISTS, onEui: true };
  if (error.isInvalidArgument())
    return { message: VAL_BS_EUI_FORMAT, onEui: true };
  if (error.isNotFound()) return { message: ERR_BS_NOT_FOUND, onEui: false };
  return { message: ERR_UPDATE_BS_EUI, onEui: false };
}
