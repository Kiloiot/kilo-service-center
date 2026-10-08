/**
 * Base station RPCs (CoreService): inventory, activity feed and message export.
 */

import { Empty } from "google-protobuf/google/protobuf/empty_pb";
import { FieldMask } from "google-protobuf/google/protobuf/field_mask_pb";
import { DoubleValue } from "google-protobuf/google/protobuf/wrappers_pb";

import * as corePb from "@services/grpc/core_pb";
import type { UplinkExportFormat } from "@constants/app";

import {
  applyPaginationAndTimeRange,
  dateToTimestamp,
  tagsMapToRecord,
  wrapperValue,
} from "../codec";
import { GrpcApiError } from "../errors";
import { grpcTransport } from "../transport";
import { type ActivityPageDTO, mapActivityResponse } from "./activity";
import { mapStationUplinkMessage } from "./uplinks";

/** gRPC-layer base station detail shape returned by getBaseStation. */
export interface GrpcBaseStationDetail {
  bsEui: string;
  tenantId: string;
  name: string;
  description?: string;
  latitude?: number;
  longitude?: number;
  altitude?: number;
  status: string;
  tags: Record<string, string>;
  createdAt?: Date;
  updatedAt?: Date;
  lastSeenAt?: Date;
  sessionStartedAt?: Date;
  // MIOTY status fields (BSSCI v1.0.0 §5.5.2)
  systemTime?: number;
  dutyCycle?: number;
  uptimeSeconds?: number;
  temperatureCelsius?: number;
  cpuLoad?: number;
  memoryLoad?: number;
  bsConfig?: Record<string, unknown>;
  lastStatusAt?: Date;
  serviceCenterUrl?: string;
  locationSource?: string;
  locationUpdatedAt?: Date;
  certificateExpiresAt?: Date;
  tlsCertFingerprint?: string;
}

/** Maps a protobuf BaseStation message to the shared GrpcBaseStationDetail shape. */
export function mapBaseStationProto(
  response: corePb.BaseStation,
): GrpcBaseStationDetail {
  return {
    bsEui: response.getBseui(),
    tenantId: response.getTenantId(),
    name: response.getName(),
    description: response.getDescription() || undefined,
    latitude: wrapperValue(response.getLatitude()),
    longitude: wrapperValue(response.getLongitude()),
    altitude: wrapperValue(response.getAltitude()),
    status: response.getStatus(),
    tags: tagsMapToRecord(response.getTagsMap()),
    createdAt: response.getCreatedAt()?.toDate(),
    updatedAt: response.getUpdatedAt()?.toDate(),
    lastSeenAt: response.getLastSeenAt()?.toDate(),
    sessionStartedAt: response.getSessionStartedAt()?.toDate(),
    systemTime: wrapperValue(response.getSystemTime()),
    dutyCycle: wrapperValue(response.getDutyCycle()),
    uptimeSeconds: wrapperValue(response.getUptimeSeconds()),
    temperatureCelsius: wrapperValue(response.getTemperatureCelsius()),
    cpuLoad: wrapperValue(response.getCpuLoad()),
    memoryLoad: wrapperValue(response.getMemoryLoad()),
    bsConfig: response.getBsConfig()?.toJavaScript() ?? undefined,
    lastStatusAt: response.getLastStatusAt()?.toDate() ?? undefined,
    serviceCenterUrl: response.getServiceCenterUrl() || undefined,
    locationSource: response.getLocationSource() || undefined,
    locationUpdatedAt: response.getLocationUpdatedAt()?.toDate() ?? undefined,
    certificateExpiresAt:
      response.getCertificateExpiresAt()?.toDate() ?? undefined,
    tlsCertFingerprint: response.getTlsCertFingerprint() || undefined,
  };
}

/**
 * List all base stations
 */
export async function listBaseStations(): Promise<GrpcBaseStationDetail[]> {
  const request = new corePb.ListBaseStationsRequest();

  const response = await grpcTransport.callCore<
    corePb.ListBaseStationsRequest,
    corePb.ListBaseStationsResponse
  >((c) => c.listBaseStations, request);

  return response.getBasestationsList().map(mapBaseStationProto);
}

/**
 * Get base station by EUI
 */
export async function getBaseStation(
  bsEui: string,
): Promise<GrpcBaseStationDetail | null> {
  const request = new corePb.GetBaseStationRequest();
  request.setBseui(bsEui);

  try {
    const response = await grpcTransport.callCore<
      corePb.GetBaseStationRequest,
      corePb.BaseStation
    >((c) => c.getBaseStation, request);
    return mapBaseStationProto(response);
  } catch (error) {
    if (error instanceof GrpcApiError && error.isNotFound()) {
      return null;
    }
    throw error;
  }
}

/**
 * Create base station
 */
export async function createBaseStation(data: {
  bsEui: string;
  name?: string;
  description?: string;
  latitude?: number;
  longitude?: number;
  altitude?: number;
}): Promise<{
  bsEui: string;
  tenantId: string;
  name: string;
  description?: string;
  latitude?: number;
  longitude?: number;
  altitude?: number;
  locationSource?: string;
  locationUpdatedAt?: Date;
  status: string;
}> {
  // Build nested BaseStation message
  const basestation = new corePb.BaseStation();
  basestation.setBseui(data.bsEui);
  if (data.name) basestation.setName(data.name);
  if (data.description) basestation.setDescription(data.description);
  if (data.latitude !== undefined) {
    const w = new DoubleValue();
    w.setValue(data.latitude);
    basestation.setLatitude(w);
  }
  if (data.longitude !== undefined) {
    const w = new DoubleValue();
    w.setValue(data.longitude);
    basestation.setLongitude(w);
  }
  if (data.altitude !== undefined) {
    const w = new DoubleValue();
    w.setValue(data.altitude);
    basestation.setAltitude(w);
  }

  const request = new corePb.CreateBaseStationRequest();
  request.setBasestation(basestation);

  const response = await grpcTransport.callCore<
    corePb.CreateBaseStationRequest,
    corePb.BaseStation
  >((c) => c.createBaseStation, request);

  return {
    bsEui: response.getBseui(),
    tenantId: response.getTenantId(),
    name: response.getName(),
    description: response.getDescription() || undefined,
    latitude: wrapperValue(response.getLatitude()),
    longitude: wrapperValue(response.getLongitude()),
    altitude: wrapperValue(response.getAltitude()),
    locationSource: response.getLocationSource() || undefined,
    locationUpdatedAt: response.getLocationUpdatedAt()?.toDate(),
    status: response.getStatus(),
  };
}

/**
 * Update base station
 */
export async function updateBaseStation(
  bsEui: string,
  data: {
    name?: string;
    description?: string;
    latitude?: number | null;
    longitude?: number | null;
    altitude?: number | null;
  },
): Promise<{
  bsEui: string;
  tenantId: string;
  name: string;
  description?: string;
  latitude?: number;
  longitude?: number;
  altitude?: number;
  locationSource?: string;
  locationUpdatedAt?: Date;
  status: string;
}> {
  const basestation = new corePb.BaseStation();
  basestation.setBseui(bsEui);

  const paths: string[] = [];

  if (data.name !== undefined) {
    basestation.setName(data.name);
    paths.push("name");
  }
  if (data.description !== undefined) {
    basestation.setDescription(data.description);
    paths.push("description");
  }
  if (data.latitude !== undefined) {
    if (data.latitude !== null) {
      const w = new DoubleValue();
      w.setValue(data.latitude);
      basestation.setLatitude(w);
    }
    paths.push("latitude");
  }
  if (data.longitude !== undefined) {
    if (data.longitude !== null) {
      const w = new DoubleValue();
      w.setValue(data.longitude);
      basestation.setLongitude(w);
    }
    paths.push("longitude");
  }
  if (data.altitude !== undefined) {
    if (data.altitude !== null) {
      const w = new DoubleValue();
      w.setValue(data.altitude);
      basestation.setAltitude(w);
    }
    paths.push("altitude");
  }

  const request = new corePb.UpdateBaseStationRequest();
  request.setBasestation(basestation);

  if (paths.length > 0) {
    const mask = new FieldMask();
    mask.setPathsList(paths);
    request.setUpdateMask(mask);
  }

  const response = await grpcTransport.callCore<
    corePb.UpdateBaseStationRequest,
    corePb.BaseStation
  >((c) => c.updateBaseStation, request);

  return {
    bsEui: response.getBseui(),
    tenantId: response.getTenantId(),
    name: response.getName(),
    description: response.getDescription() || undefined,
    latitude: wrapperValue(response.getLatitude()),
    longitude: wrapperValue(response.getLongitude()),
    altitude: wrapperValue(response.getAltitude()),
    locationSource: response.getLocationSource() || undefined,
    locationUpdatedAt: response.getLocationUpdatedAt()?.toDate(),
    status: response.getStatus(),
  };
}

/**
 * Update base station EUI with cascade to all dependent tables.
 */
export async function updateBaseStationEui(
  bsEui: string,
  newBsEui: string,
): Promise<{
  bsEui: string;
  tenantId: string;
  name: string;
  description?: string;
  latitude?: number;
  longitude?: number;
  altitude?: number;
  locationSource?: string;
  locationUpdatedAt?: Date;
  status: string;
}> {
  const request = new corePb.UpdateBaseStationEuiRequest();
  request.setBsEui(bsEui);
  request.setNewBsEui(newBsEui);

  const response = await grpcTransport.callCore<
    corePb.UpdateBaseStationEuiRequest,
    corePb.BaseStation
  >((c) => c.updateBaseStationEui, request);

  return {
    bsEui: response.getBseui(),
    tenantId: response.getTenantId(),
    name: response.getName(),
    description: response.getDescription() || undefined,
    latitude: wrapperValue(response.getLatitude()),
    longitude: wrapperValue(response.getLongitude()),
    altitude: wrapperValue(response.getAltitude()),
    locationSource: response.getLocationSource() || undefined,
    locationUpdatedAt: response.getLocationUpdatedAt()?.toDate(),
    status: response.getStatus(),
  };
}

/**
 * Delete base station
 */
export async function deleteBaseStation(bsEui: string): Promise<void> {
  const request = new corePb.DeleteBaseStationRequest();
  request.setBseui(bsEui);

  await grpcTransport.callCore<corePb.DeleteBaseStationRequest, Empty>(
    (c) => c.deleteBaseStation,
    request,
  );
}

/**
 * List base station activity (unified events and messages)
 */
export async function listBaseStationActivity(
  bsEui: string,
  params?: {
    pageSize?: number;
    pageToken?: string;
    startTime?: Date;
    endTime?: Date;
  },
): Promise<ActivityPageDTO> {
  const request = new corePb.ListBaseStationActivityRequest();
  request.setBsEui(bsEui);
  applyPaginationAndTimeRange(request, params);

  const response = await grpcTransport.callCore<
    corePb.ListBaseStationActivityRequest,
    corePb.ListBaseStationActivityResponse
  >((c) => c.listBaseStationActivity, request);

  return mapActivityResponse(response, mapStationUplinkMessage);
}

/**
 * Export base station messages
 * Added direction and epEui filters.
 */
export async function exportBaseStationMessages(
  bsEui: string,
  format: UplinkExportFormat,
  params?: {
    startTime?: Date;
    endTime?: Date;
    direction?: "uplink" | "downlink";
    epEui?: string;
  },
): Promise<{
  content: Uint8Array;
  filename: string;
  contentType: string;
}> {
  const request = new corePb.ExportBaseStationMessagesRequest();
  request.setBsEui(bsEui);
  request.setFormat(format);
  if (params?.direction) request.setDirection(params.direction);
  if (params?.epEui) request.setEpEui(params.epEui);
  if (params?.startTime)
    request.setStartTime(dateToTimestamp(params.startTime));
  if (params?.endTime) request.setEndTime(dateToTimestamp(params.endTime));

  const response = await grpcTransport.callCore<
    corePb.ExportBaseStationMessagesRequest,
    corePb.ExportBaseStationMessagesResponse
  >((c) => c.exportBaseStationMessages, request);

  return {
    content: response.getContent_asU8(),
    filename: response.getFilename(),
    contentType: response.getContentType(),
  };
}
