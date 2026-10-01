/**
 * Base Station Mappers
 *
 * Transforms base station data from API format to UI format.
 * Aligns with KC-DB/storage/mioty/types.go field names.
 */

import type {
  BaseStationAPI,
  BaseStationDetailAPI,
  BaseStationUI,
} from "@api-types/api";

import type { LocationSource } from "@constants/app";
import {
  BASE_STATION_STATUS,
  BS_CONNECTION_TYPE,
  BS_CONNECTION_TYPE_LABEL,
  LOCATION_SOURCE,
} from "@constants/app";

type ConnectionFields = Pick<BaseStationUI, "status" | "connectionType">;
type LocationFields = Pick<
  BaseStationUI,
  "latitude" | "longitude" | "altitude" | "locationSource" | "locationUpdatedAt"
>;
type HealthFields = Pick<
  BaseStationUI,
  | "systemTime"
  | "dutyCycle"
  | "uptimeSeconds"
  | "temperatureCelsius"
  | "cpuLoad"
  | "memoryLoad"
  | "bsConfig"
  | "lastStatusAt"
>;

/** The stored location source when it is one the UI knows, else none. */
const toLocationSource = (
  value: string | undefined,
): LocationSource | undefined =>
  Object.values(LOCATION_SOURCE).find((source) => source === value);

/** Resolves online/offline + BSSCI/MQTT from list- and detail-shape payloads. */
function mapConnectionStatus(
  bs: BaseStationAPI,
  isDetail: boolean,
): ConnectionFields {
  const isOnline = bs.isOnline ?? bs.is_online ?? false;
  const wireType = isDetail
    ? bs.connectionType || bs.connection_type
    : undefined;
  const connectionType: BaseStationUI["connectionType"] =
    wireType === BS_CONNECTION_TYPE.MQTT
      ? BS_CONNECTION_TYPE_LABEL.MQTT
      : BS_CONNECTION_TYPE_LABEL.BSSCI;
  return {
    status: isOnline ? BASE_STATION_STATUS.ONLINE : BASE_STATION_STATUS.OFFLINE,
    connectionType,
  };
}

/** Resolves GPS coordinates + location source/timestamp (list + detail shapes). */
function mapLocationFields(bs: BaseStationAPI): LocationFields {
  const detail = bs as BaseStationDetailAPI;
  return {
    latitude: bs.latitude,
    longitude: bs.longitude,
    altitude: bs.altitude,
    locationSource: toLocationSource(
      bs.locationSource || detail.locationSource,
    ),
    locationUpdatedAt: detail.locationUpdatedAt,
  };
}

/** Resolves the detail-only MIOTY health metrics (BSSCI v1.0.0 §3.5.2). */
function mapHealthStatus(bs: BaseStationAPI, isDetail: boolean): HealthFields {
  if (!isDetail) {
    return {
      systemTime: undefined,
      dutyCycle: undefined,
      uptimeSeconds: undefined,
      temperatureCelsius: undefined,
      cpuLoad: undefined,
      memoryLoad: undefined,
      bsConfig: undefined,
      lastStatusAt: undefined,
    };
  }
  const detail = bs as BaseStationDetailAPI;
  return {
    systemTime: detail.systemTime,
    dutyCycle: detail.dutyCycle,
    uptimeSeconds: detail.uptimeSeconds,
    temperatureCelsius: detail.temperatureCelsius,
    cpuLoad: detail.cpuLoad,
    memoryLoad: detail.memoryLoad,
    bsConfig: detail.bsConfig,
    lastStatusAt: detail.lastStatusAt,
  };
}

/**
 * Transform a single base station from API to UI format
 * Handles both list responses and detail responses
 */
export function mapBaseStation(bs: BaseStationAPI): BaseStationUI {
  const isDetail = "connectionType" in bs || "connection_type" in bs;
  const detail = bs as BaseStationDetailAPI;
  const eui = bs.eui || bs.bsEui || "";

  return {
    id: eui, // backend doesn't provide a numeric ID; EUI is the stable key
    eui,
    name: bs.name,
    ...mapConnectionStatus(bs, isDetail),
    createdAt: bs.firstSeen || bs.first_seen || bs.created_at || "",
    lastSeen: bs.lastSeen || bs.last_seen || bs.last_seen_at || "",
    lastHandshake: bs.sessionStartedAt,
    serviceCenterUrl: isDetail
      ? bs.serviceCenterUrl || bs.service_center_url || ""
      : "",
    ...mapLocationFields(bs),
    certificateExpiryDate: bs.certificateExpiresAt,
    certificateFingerprint: bs.tlsCertFingerprint,
    version: isDetail ? detail.version : undefined,
    ...mapHealthStatus(bs, isDetail),
  };
}
