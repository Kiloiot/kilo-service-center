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

/**
 * Transform a single base station from API to UI format
 * Handles both list responses and detail responses
 */
export function mapBaseStation(bs: BaseStationAPI): BaseStationUI {
  // Check if this is a detail response by checking for detail-specific fields
  const isDetail = "connectionType" in bs || "connection_type" in bs;

  // Handle both camelCase and snake_case field names from the API
  const isOnline = bs.isOnline ?? bs.is_online ?? false;

  // Handle both 'eui' and 'bsEui' field names from backend (list vs detail endpoints)
  const eui = bs.eui || bs.bsEui || "";

  return {
    id: eui, // Using EUI as ID since backend doesn't provide numeric ID
    eui: eui,
    name: bs.name,
    status: isOnline ? "online" : "offline",
    connectionType: isDetail
      ? ((bs.connectionType || bs.connection_type || "bssci").toUpperCase() as
          | "BSSCI"
          | "MQTT")
      : "BSSCI",
    createdAt: bs.firstSeen || bs.first_seen || bs.created_at || "",
    lastSeen: bs.lastSeen || bs.last_seen || bs.last_seen_at || "",
    serviceCenterUrl: isDetail
      ? bs.serviceCenterUrl || bs.service_center_url || ""
      : "",
    latitude: bs.latitude,
    longitude: bs.longitude,
    altitude: bs.altitude,
    locationSource:
      (bs.locationSource as "gps" | "manual") ||
      ((bs as BaseStationDetailAPI).locationSource as
        | "gps"
        | "manual"
        | undefined),
    locationUpdatedAt: (bs as BaseStationDetailAPI).locationUpdatedAt,
    certificateExpiryDate: isDetail
      ? (bs as BaseStationDetailAPI).tlsCertExpiresAt
      : undefined,
    version: isDetail ? (bs as BaseStationDetailAPI).version : undefined,
    // MIOTY status metrics (only available in detail view) per BSSCI v1.0.0 §3.5.2
    systemTime: isDetail ? (bs as BaseStationDetailAPI).systemTime : undefined,
    dutyCycle: isDetail ? (bs as BaseStationDetailAPI).dutyCycle : undefined,
    uptimeSeconds: isDetail
      ? (bs as BaseStationDetailAPI).uptimeSeconds
      : undefined,
    temperatureCelsius: isDetail
      ? (bs as BaseStationDetailAPI).temperatureCelsius
      : undefined,
    cpuLoad: isDetail ? (bs as BaseStationDetailAPI).cpuLoad : undefined,
    memoryLoad: isDetail ? (bs as BaseStationDetailAPI).memoryLoad : undefined,
    bsConfig: isDetail ? (bs as BaseStationDetailAPI).bsConfig : undefined,
    lastStatusAt: isDetail
      ? (bs as BaseStationDetailAPI).lastStatusAt
      : undefined,
  };
}

/**
 * Transform an array of base stations
 */
export function mapBaseStationList(
  baseStations: BaseStationAPI[],
): BaseStationUI[] {
  return baseStations.map(mapBaseStation);
}
