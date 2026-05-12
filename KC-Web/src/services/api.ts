/**
 * API Service - Canonical API Layer (gRPC-backed)
 *
 * GOVERNANCE: This is the SINGLE canonical API layer for KC-Web.
 * All components and hooks MUST import from '@services/api' - never from grpc/client directly.
 * The grpc/client module is INTERNAL ONLY and should only be imported here.
 *
 * This service provides Promise-based APIs with consistent error handling and
 * automatic mapping from gRPC responses to UI types via the @mappers module.
 */

import type {
  ActivityItem,
  AlertSummaryAPI,
  AnalyticsOverviewAPI,
  ApiKeyAPI,
  ApiKeyCreateResponse,
  ApiKeyListResponse,
  AuthSettingsAPI,
  BaseStationActivityFilter,
  BaseStationActivityPage,
  BaseStationAPI,
  BaseStationDetailAPI,
  BaseStationMessageAPI,
  BaseStationMessagesFilter,
  BaseStationMessagesPage,
  BaseStationMessagesStats,
  BaseStationReceptionAPI,
  BaseStationUI,
  BlueprintListItemAPI,
  BlueprintUI,
  CertificateUI,
  CreateBlueprintRequest,
  CreateDeviceModelRequest,
  CreateDeviceModelWithBlueprintRequest,
  CreateEndpointRequest,
  CreateManufacturerRequest,
  DecodePreviewRequest,
  DecodePreviewResponse,
  DeviceModelUI,
  EndpointActivityPage,
  EndpointAPI,
  EndpointUI,
  EventUI,
  ExchangeRequest,
  GenerateCertificateRequest,
  GenerateCertificateResponse,
  ListAllBaseStationLocationsResponse,
  LoginRequest,
  LoginResponseAPI,
  ManufacturerUI,
  RegisterAccountRequest,
  RegistrySubmitRequest,
  RegistrySubmitResponse,
  SystemStatusResponse,
  UpdateBlueprintRequest,
  UpdateDeviceModelRequest,
  UpdateEndpointRequest,
  UpdateManufacturerRequest,
  UserProfileAPI,
} from "@api-types/api";
import {
  mapBaseStation,
  mapCertificate,
  mapEndpoint,
  mapEvent,
  mapLoginResponse,
  mapOrganization,
  mapOrganizationList,
  mapOrgUser,
  mapOrgUserList,
  mapUser,
  mapUserList,
  mapUserProfile,
} from "@mappers";
import { deriveAttachState } from "@mappers/endpoint.mapper";

import { hexToBytes } from "@utils/formatters";
import type { CertificateDownloadType } from "@constants/app";
import {
  CERTIFICATE_TYPES,
  ENDPOINT_EP_CLASS,
  REGISTRY_DEFAULTS,
} from "@constants/app";
import {
  CERTIFICATE_LABELS,
  VAL_INVALID_HEX_STRING,
} from "@constants/messages";

// INTERNAL: grpcClient is only imported here - nowhere else in the codebase
/* eslint-disable import/no-internal-modules */
import {
  type BlueprintTransportDTO,
  GrpcApiError,
  grpcClient,
} from "./grpc/client";
/* eslint-enable import/no-internal-modules */

// Re-export GrpcApiError as ApiError for backward compatibility
export { GrpcApiError as ApiError };

/** Shape returned by grpcClient endpoint methods (uses centralized GrpcEndpointResult) */
type GrpcEndpoint = Awaited<
  ReturnType<typeof grpcClient.listEndpoints>
>[number];

/** Maps a gRPC endpoint response to EndpointAPI and then to EndpointUI */
function mapGrpcEndpointToUI(ep: GrpcEndpoint): EndpointUI {
  return mapEndpoint({
    epEui: ep.epEui,
    name: ep.name,
    status: ep.status as EndpointAPI["status"],
    bidi: ep.epClass === "A",
    createdAt: ep.createdAt?.toISOString() || "",
    lastSeen: ep.lastSeenAt?.toISOString() || "",
    attachStatus:
      (ep.attachStatus as EndpointAPI["attachStatus"]) ??
      deriveAttachState({ shAddr: ep.shAddr } as EndpointAPI),
    shAddr: ep.shAddr,
    attachCnt: ep.attachCnt,
    lastPacketCnt: ep.lastPacketCnt,
    preAttach: ep.preAttach,
    carrierOffset: ep.carrierOffset,
    dualChan: ep.dualChan,
    repetition: ep.repetition,
    wideCarrOff: ep.wideCarrOff,
    longBlkDist: ep.longBlkDist,
    nwkSnKey: ep.nwkSnKey,
    appKey: ep.appKey,
    typeEui: ep.typeEui,
    deviceModelId: ep.deviceModelId,
  });
}

/** Maps a BlueprintTransportDTO from the gRPC layer to the UI-facing BlueprintUI type */
function mapTransportBlueprintToUI(b: BlueprintTransportDTO): BlueprintUI {
  let specJson: Record<string, unknown> = {};
  if (b.specJson) {
    try {
      specJson = JSON.parse(b.specJson);
    } catch {
      specJson = { script: b.specJson };
    }
  }

  return {
    id: b.id,
    deviceModelId: b.deviceModelId,
    tenantId: 0,
    version: b.version,
    typeEui: b.typeEui,
    specJson,
    isDefault: b.isDefault,
    registryRepo: b.registryRepo,
    registryCommitSha: b.registryCommitSha,
    registryVerified: b.registryVerified,
    registryPrUrl: b.registryPrUrl,
    createdAt: b.createdAt?.toISOString() || "",
    updatedAt: b.updatedAt?.toISOString() || "",
  };
}

/** Parse proto tenant IDs (string) safely for UI models. */
function parseTenantIdOrZero(tenantId: string): number {
  const parsed = Number.parseInt(tenantId, 10);
  return Number.isNaN(parsed) ? 0 : parsed;
}

interface BaseStationListFilters {
  search?: string;
  status?: string[];
  sort?: { field: string; direction: "asc" | "desc" };
}

interface EndpointListFilters {
  search?: string;
  attachState?: string[];
  bidirectional?: boolean;
  sort?: { field: string; direction: "asc" | "desc" };
}

function compareNullableString(
  a: string | undefined | null,
  b: string | undefined | null,
  direction: "asc" | "desc",
): number {
  const aMissing = a === undefined || a === null || a === "";
  const bMissing = b === undefined || b === null || b === "";
  if (aMissing && bMissing) return 0;
  if (aMissing) return 1; // null/empty sort last
  if (bMissing) return -1;
  const cmp = (a as string).localeCompare(b as string);
  return direction === "asc" ? cmp : -cmp;
}

function compareNullableNumber(
  a: number | undefined | null,
  b: number | undefined | null,
  direction: "asc" | "desc",
): number {
  const aMissing = a === undefined || a === null;
  const bMissing = b === undefined || b === null;
  if (aMissing && bMissing) return 0;
  if (aMissing) return 1;
  if (bMissing) return -1;
  const cmp = (a as number) - (b as number);
  return direction === "asc" ? cmp : -cmp;
}

function filterAndSortBaseStations(
  items: BaseStationUI[],
  filters?: BaseStationListFilters,
): BaseStationUI[] {
  if (!filters) return items;
  let out = items;
  if (filters.search) {
    const needle = filters.search.toLowerCase();
    out = out.filter(
      (bs) =>
        (bs.name?.toLowerCase().includes(needle) ?? false) ||
        bs.eui.toLowerCase().includes(needle),
    );
  }
  if (filters.status && filters.status.length > 0) {
    const allowed = new Set(filters.status);
    out = out.filter((bs) => allowed.has(bs.status));
  }
  if (filters.sort) {
    const { field, direction } = filters.sort;
    out = [...out].sort((a, b) => {
      switch (field) {
        case "name":
          return compareNullableString(a.name, b.name, direction);
        case "eui":
          return compareNullableString(a.eui, b.eui, direction);
        case "status":
          return compareNullableString(a.status, b.status, direction);
        case "connectionType":
          return compareNullableString(
            a.connectionType,
            b.connectionType,
            direction,
          );
        case "lastSeen":
          return compareNullableString(a.lastSeen, b.lastSeen, direction);
        default:
          return 0;
      }
    });
  }
  return out;
}

function filterAndSortEndpoints(
  items: EndpointUI[],
  filters?: EndpointListFilters,
): EndpointUI[] {
  if (!filters) return items;
  let out = items;
  if (filters.search) {
    const needle = filters.search.toLowerCase();
    out = out.filter(
      (ep) =>
        (ep.name?.toLowerCase().includes(needle) ?? false) ||
        ep.epEui.toLowerCase().includes(needle),
    );
  }
  if (filters.attachState && filters.attachState.length > 0) {
    const allowed = new Set(filters.attachState);
    out = out.filter((ep) => allowed.has(ep.attachStatus));
  }
  if (filters.bidirectional !== undefined) {
    out = out.filter((ep) => (ep.bidi ?? false) === filters.bidirectional);
  }
  if (filters.sort) {
    const { field, direction } = filters.sort;
    out = [...out].sort((a, b) => {
      switch (field) {
        case "name":
          return compareNullableString(a.name, b.name, direction);
        case "epEui":
          return compareNullableString(a.epEui, b.epEui, direction);
        case "attachState":
          return compareNullableString(
            a.attachStatus,
            b.attachStatus,
            direction,
          );
        case "lastSeen":
          return compareNullableString(a.lastSeen, b.lastSeen, direction);
        case "packetCnt":
          return compareNullableNumber(
            a.lastPacketCnt,
            b.lastPacketCnt,
            direction,
          );
        default:
          return 0;
      }
    });
  }
  return out;
}

const payloadWhitespaceRegex = /\s+/g;
const payloadHexPrefixRegex = /^0x/i;
const payloadBase64Regex = /^[A-Za-z0-9+/]+={0,2}$/;

function normalizePayloadInput(payload: string): string {
  return payload
    .replace(payloadWhitespaceRegex, "")
    .replace(payloadHexPrefixRegex, "");
}

function base64ToBytes(base64Payload: string): Uint8Array {
  const decoded = atob(base64Payload);
  return Uint8Array.from(decoded, (char) => char.charCodeAt(0));
}

/**
 * Decode preview payload parser:
 * 1) Accepts canonical hex (with optional spaces / 0x prefix)
 * 2) Falls back to base64 for copied payloads from endpoint views
 */
function parseDecodePreviewPayload(payload: string): Uint8Array {
  const normalized = normalizePayloadInput(payload);

  try {
    return hexToBytes(normalized);
  } catch {
    const looksBase64 =
      normalized.length > 0 &&
      normalized.length % 4 === 0 &&
      payloadBase64Regex.test(normalized);
    if (looksBase64) {
      return base64ToBytes(normalized);
    }
    throw new Error(VAL_INVALID_HEX_STRING);
  }
}

/**
 * Callback type for auth failure handling (401 responses)
 */
type AuthFailureCallback = () => void;

/**
 * API Service - gRPC-backed implementation
 *
 * All operations delegate to grpcClient internally.
 * REST shim removed, all methods now use gRPC directly.
 */
class ApiService {
  /**
   * Clear organization context (logout/session reset)
   */
  clearOrganization(): void {
    grpcClient.clearOrganization();
  }

  /**
   * Set organization context for API requests
   */
  setOrganization(
    orgId: string | null | undefined,
    userId?: string | null | undefined,
  ): void {
    grpcClient.setOrganization(orgId, userId);
  }

  /**
   * Set auth failure callback for 401 response handling
   */
  setAuthFailureCallback(callback: AuthFailureCallback | undefined): void {
    grpcClient.setAuthFailureCallback(callback);
  }

  // ============================================================================
  // BASE STATIONS
  // ============================================================================

  async getBaseStations(filters?: {
    search?: string;
    status?: string[];
    sort?: { field: string; direction: "asc" | "desc" };
  }): Promise<BaseStationUI[]> {
    const response = await grpcClient.listBaseStations();

    // Map gRPC response to API format expected by mappers
    const items = response.map((bs) => {
      const apiFormat: BaseStationAPI = {
        bsEui: bs.bsEui,
        name: bs.name,
        isOnline: bs.status === "online",
        messageCount: 0,
        lastSeen: bs.lastSeenAt?.toISOString() || "",
        firstSeen: bs.createdAt?.toISOString() || "",
        uniqueEndpoints: 0,
        avgRssi: 0,
        avgSnr: 0,
        latitude: bs.latitude,
        longitude: bs.longitude,
        altitude: bs.altitude,
        locationSource: bs.locationSource,
        locationUpdatedAt: bs.locationUpdatedAt?.toISOString(),
      };
      return mapBaseStation(apiFormat);
    });

    return filterAndSortBaseStations(items, filters);
  }

  async getBaseStationById(id: string): Promise<BaseStationUI | null> {
    try {
      const bs = await grpcClient.getBaseStation(id);
      if (!bs) return null;

      const apiFormat: BaseStationAPI = {
        bsEui: bs.bsEui,
        name: bs.name,
        isOnline: bs.status === "online",
        messageCount: 0,
        lastSeen: bs.lastSeenAt?.toISOString() || "",
        firstSeen: bs.createdAt?.toISOString() || "",
        uniqueEndpoints: 0,
        avgRssi: 0,
        avgSnr: 0,
        latitude: bs.latitude,
        longitude: bs.longitude,
        altitude: bs.altitude,
        locationSource: bs.locationSource,
        locationUpdatedAt: bs.locationUpdatedAt?.toISOString(),
      };
      return mapBaseStation(apiFormat);
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) return null;
      throw error;
    }
  }

  async getBaseStationDetails(eui: string): Promise<BaseStationUI | null> {
    try {
      const bs = await grpcClient.getBaseStation(eui);
      if (!bs) return null;

      const apiFormat: BaseStationDetailAPI = {
        eui: bs.bsEui,
        bsEui: bs.bsEui,
        name: bs.name,
        isOnline: bs.status === "online",
        messageCount: 0,
        lastSeen: bs.lastSeenAt?.toISOString() || "",
        firstSeen: bs.createdAt?.toISOString() || "",
        uniqueEndpoints: 0,
        avgRssi: 0,
        avgSnr: 0,
        connectionType: "bssci",
        statusCode: 0,
        createdAt: bs.createdAt?.toISOString() || "",
        updatedAt: bs.updatedAt?.toISOString() || "",
        // MIOTY status fields (BSSCI v1.0.0 §5.5.2)
        systemTime: bs.systemTime,
        dutyCycle: bs.dutyCycle,
        uptimeSeconds: bs.uptimeSeconds,
        temperatureCelsius: bs.temperatureCelsius,
        cpuLoad: bs.cpuLoad,
        memoryLoad: bs.memoryLoad,
        bsConfig: bs.bsConfig,
        lastStatusAt: bs.lastStatusAt?.toISOString(),
        serviceCenterUrl: bs.serviceCenterUrl,
        latitude: bs.latitude,
        longitude: bs.longitude,
        altitude: bs.altitude,
        locationSource: bs.locationSource,
        locationUpdatedAt: bs.locationUpdatedAt?.toISOString(),
      };
      return mapBaseStation(apiFormat);
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  async createBaseStation(data: {
    eui: string;
    name?: string;
    description?: string;
    connection_type: "bssci" | "mqtt";
    service_center_url?: string;
    tls_auth_required?: boolean;
    tls_hostname_verification?: boolean;
    tls_ca_certificate?: string;
    tls_certificate?: string;
    tls_key?: string;
    latitude?: number;
    longitude?: number;
    altitude?: number;
    mqtt_broker_url?: string;
    mqtt_username?: string;
    mqtt_password?: string;
    config_file_content?: string;
  }): Promise<BaseStationUI> {
    const response = await grpcClient.createBaseStation({
      bsEui: data.eui,
      name: data.name,
      description: data.description,
      latitude: data.latitude,
      longitude: data.longitude,
      altitude: data.altitude,
    });

    const apiFormat: BaseStationAPI = {
      bsEui: response.bsEui,
      name: response.name,
      isOnline: response.status === "online",
      messageCount: 0,
      lastSeen: "",
      firstSeen: "",
      uniqueEndpoints: 0,
      avgRssi: 0,
      avgSnr: 0,
      latitude: response.latitude,
      longitude: response.longitude,
      altitude: response.altitude,
      locationSource: response.locationSource,
      locationUpdatedAt: response.locationUpdatedAt?.toISOString(),
    };
    return mapBaseStation(apiFormat);
  }

  async deleteBaseStation(eui: string): Promise<void> {
    await grpcClient.deleteBaseStation(eui);
  }

  /**
   * Update base station
   * Enables editing from detail page dialog.
   */
  async updateBaseStation(
    eui: string,
    data: {
      name?: string;
      latitude?: number | null;
      longitude?: number | null;
      altitude?: number | null;
    },
  ): Promise<BaseStationUI> {
    const response = await grpcClient.updateBaseStation(eui, {
      name: data.name,
      latitude: data.latitude,
      longitude: data.longitude,
      altitude: data.altitude,
    });

    const apiFormat: BaseStationAPI = {
      bsEui: response.bsEui,
      name: response.name,
      isOnline: response.status === "online",
      messageCount: 0,
      lastSeen: "",
      firstSeen: "",
      uniqueEndpoints: 0,
      avgRssi: 0,
      avgSnr: 0,
      latitude: response.latitude,
      longitude: response.longitude,
      altitude: response.altitude,
      locationSource: response.locationSource,
      locationUpdatedAt: response.locationUpdatedAt?.toISOString(),
    };
    return mapBaseStation(apiFormat);
  }

  /**
   * Update base station EUI with cascade to all dependent tables.
   */
  async updateBaseStationEui(
    eui: string,
    newEui: string,
  ): Promise<BaseStationUI> {
    const response = await grpcClient.updateBaseStationEui(eui, newEui);

    const apiFormat: BaseStationAPI = {
      bsEui: response.bsEui,
      name: response.name,
      isOnline: response.status === "online",
      messageCount: 0,
      lastSeen: "",
      firstSeen: "",
      uniqueEndpoints: 0,
      avgRssi: 0,
      avgSnr: 0,
      latitude: response.latitude,
      longitude: response.longitude,
      altitude: response.altitude,
      locationSource: response.locationSource,
      locationUpdatedAt: response.locationUpdatedAt?.toISOString(),
    };
    return mapBaseStation(apiFormat);
  }

  // ============================================================================
  // BASE STATION MESSAGES
  // ============================================================================

  async getBaseStationMessages(
    eui: string,
    filter?: BaseStationMessagesFilter,
    page = 1,
    pageSize = 50,
  ): Promise<BaseStationMessagesPage> {
    const response = await grpcClient.listBaseStationMessages(eui, {
      pageSize,
      direction: filter?.direction as "uplink" | "downlink" | undefined,
      epEui: filter?.epEui,
      startTime: filter?.startTime ? new Date(filter.startTime) : undefined,
      endTime: filter?.endTime ? new Date(filter.endTime) : undefined,
    });

    return {
      messages: response.messages.map((m) => ({
        id: m.id,
        bsEui: m.bsEui,
        epEui: m.epEui,
        direction: m.direction,
        messageType: m.uplinkMode || "ulData",
        packetCnt: m.packetCounter || 0,
        userData: m.payload,
        rssi: m.rssi,
        snr: m.snr,
        receivedAt: m.receivedAt.toISOString(),
      })),
      page,
      pageSize,
      totalCount: response.totalCount,
      totalPages: Math.ceil(response.totalCount / pageSize),
    };
  }

  async getBaseStationMessage(
    eui: string,
    messageId: string,
  ): Promise<{
    id: string;
    bsEui: string;
    epEui: string;
    direction: "uplink" | "downlink";
    userData: string;
    rssi: number;
    snr: number;
    timestamp: string;
    receivedAt: string;
  } | null> {
    const msg = await grpcClient.getBaseStationMessage(eui, messageId);
    if (!msg) return null;

    return {
      id: msg.id,
      bsEui: msg.bsEui,
      epEui: msg.epEui,
      direction: msg.direction,
      userData: msg.payload,
      rssi: msg.rssi,
      snr: msg.snr,
      timestamp: msg.receivedAt.toISOString(),
      receivedAt: msg.receivedAt.toISOString(),
    };
  }

  async getBaseStationMessageStats(
    eui: string,
    startTime?: Date,
    endTime?: Date,
  ): Promise<BaseStationMessagesStats | null> {
    const response = await grpcClient.getBaseStationMessageStats(eui, {
      startTime,
      endTime,
    });
    if (!response) {
      return null; // Caller handles absence
    }
    return {
      totalMessages: response.totalMessages,
      uplinkMessages: response.totalMessages, // All messages are uplinks
      downlinkMessages: 0,
      uniqueEndpoints: response.uniqueEndpoints,
      averageRssi: response.avgRssi,
      averageSnr: response.avgSnr,
      messagesLastHour: 0,
      messagesToday: response.messagesToday,
      messagesThisWeek: response.messagesThisWeek,
      messagesThisMonth: response.messagesThisMonth,
      messagesByType: {},
      topEndpoints: [],
    };
  }

  /**
   * Map a transport activity response to a typed activity page.
   * Shared by base station and endpoint activity methods.
   */
  private mapActivityPage(response: {
    items: Array<{
      type: string;
      occurredAt: Date;
      event?: {
        id: string;
        eventType: string;
        category: string;
        severity: string;
        title: string;
        description: string;
        timestamp: Date;
      };
      message?: {
        id: string;
        bsEui: string;
        epEui: string;
        userData: string;
        rssi: number;
        snr: number;
        eqSnr: number;
        messageType: string;
        packetCnt: number;
        receivedAt: Date;
        direction: "uplink" | "downlink";
        baseStations?: Array<{
          bsEui: string;
          rxTime: number;
          snr: number;
          rssi: number;
          eqSnr?: number;
          rxDuration?: number;
          profile?: string;
          mode?: string;
          dlRxSnr?: number;
          dlRxRssi?: number;
          subpackets?: {
            snr: number[];
            rssi: number[];
            frequency: number[];
            phase?: number[];
          };
        }>;
        duplicate?: boolean;
        decodedPayload?: Record<string, unknown>;
        decodeStatus?: string;
        decodeErrorCode?: string;
      };
    }>;
    nextPageToken?: string;
    totalCount: number;
  }): BaseStationActivityPage {
    return {
      items: response.items.map(
        (item): ActivityItem => ({
          type: item.type as "event" | "message",
          occurredAt: item.occurredAt,
          event: item.event,
          message: item.message
            ? {
                ...item.message,
                receivedAt: item.message.receivedAt.toISOString(),
              }
            : undefined,
        }),
      ),
      nextPageToken: response.nextPageToken,
      totalCount: response.totalCount,
    };
  }

  /**
   * Get unified base station activity (events + messages)
   */
  async getBaseStationActivity(
    eui: string,
    filter?: BaseStationActivityFilter,
    pageToken?: string,
    pageSize = 50,
  ): Promise<BaseStationActivityPage> {
    const response = await grpcClient.listBaseStationActivity(eui, {
      pageSize,
      pageToken,
      startTime: filter?.startTime ? new Date(filter.startTime) : undefined,
      endTime: filter?.endTime ? new Date(filter.endTime) : undefined,
    });

    return this.mapActivityPage(response);
  }

  /**
   * Get unified endpoint activity (events + messages)
   */
  async getEndpointActivity(
    epEui: string,
    pageToken?: string,
    pageSize = 50,
  ): Promise<EndpointActivityPage> {
    const response = await grpcClient.listEndpointActivity(epEui, {
      pageSize,
      pageToken,
    });

    return this.mapActivityPage(response);
  }

  async getLatestBaseStationMessages(
    eui: string,
    limit = 10,
  ): Promise<
    Array<{
      id: string;
      bsEui: string;
      epEui: string;
      direction: "uplink" | "downlink";
      userData: string;
      rssi: number;
      snr: number;
      timestamp: string;
      receivedAt: string;
    }>
  > {
    const response = await grpcClient.listBaseStationMessages(eui, {
      pageSize: limit,
    });
    return response.messages.map((m) => ({
      id: m.id,
      bsEui: m.bsEui,
      epEui: m.epEui,
      direction: m.direction,
      userData: m.payload,
      rssi: m.rssi,
      snr: m.snr,
      timestamp: m.receivedAt.toISOString(),
      receivedAt: m.receivedAt.toISOString(),
    }));
  }

  async searchBaseStationMessages(
    eui: string,
    searchTerm: string,
    filter?: BaseStationMessagesFilter,
    page = 1,
    pageSize = 50,
  ): Promise<BaseStationMessagesPage> {
    const response = await grpcClient.searchBaseStationMessages(
      eui,
      searchTerm,
      {
        pageSize,
        direction: filter?.direction as "uplink" | "downlink" | undefined,
        epEui: filter?.epEui,
        startTime: filter?.startTime ? new Date(filter.startTime) : undefined,
        endTime: filter?.endTime ? new Date(filter.endTime) : undefined,
      },
    );

    return {
      messages: response.messages.map((m) => ({
        id: m.id,
        bsEui: m.bsEui,
        epEui: m.epEui,
        direction: m.direction,
        messageType: m.uplinkMode || "ulData",
        packetCnt: m.packetCounter || 0,
        userData: m.payload,
        rssi: m.rssi,
        snr: m.snr,
        receivedAt: m.receivedAt.toISOString(),
      })),
      page,
      pageSize,
      totalCount: response.totalCount,
      totalPages: Math.ceil(response.totalCount / pageSize),
    };
  }

  async exportBaseStationMessages(
    eui: string,
    filter?: BaseStationMessagesFilter,
    format: "csv" | "json" = "csv",
  ): Promise<Blob> {
    const response = await grpcClient.exportBaseStationMessages(eui, format, {
      direction: filter?.direction as "uplink" | "downlink" | undefined,
      epEui: filter?.epEui,
      startTime: filter?.startTime ? new Date(filter.startTime) : undefined,
      endTime: filter?.endTime ? new Date(filter.endTime) : undefined,
    });
    return new Blob([new Uint8Array(response.content)], {
      type: response.contentType,
    });
  }

  // ============================================================================
  // ENDPOINTS
  // ============================================================================

  async getEndpoints(filters?: {
    search?: string;
    attachState?: string[];
    bidirectional?: boolean;
    sort?: { field: string; direction: "asc" | "desc" };
  }): Promise<EndpointUI[]> {
    const response = await grpcClient.listEndpoints();
    const items = response.map(mapGrpcEndpointToUI);
    return filterAndSortEndpoints(items, filters);
  }

  async getEndpointById(id: string): Promise<EndpointUI | null> {
    try {
      const ep = await grpcClient.getEndpoint(id);
      if (!ep) return null;

      return mapGrpcEndpointToUI(ep);
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  async createEndpoint(data: CreateEndpointRequest): Promise<EndpointUI> {
    const response = await grpcClient.createEndpoint({
      epEui: data.epEui,
      name: data.name,
      epClass: data.bidi
        ? ENDPOINT_EP_CLASS.BIDIRECTIONAL
        : ENDPOINT_EP_CLASS.UNIDIRECTIONAL,
      nwkSnKey: data.nwkSnKey,
      appSnKey: data.appKey,
      shAddr: data.shAddr,
      dualChan: data.dualChan,
      repetition: data.repetition,
      wideCarrOff: data.wideCarrOff,
      longBlkDist: data.longBlkDist,
      attachCnt: data.attachCnt,
      preAttach: data.preAttach,
      lastPacketCnt: data.lastPacketCnt,
      carrierOffset: data.carrierOffset,
      typeEui: data.typeEui,
      deviceModelId: data.deviceModelId,
    });

    return mapGrpcEndpointToUI(response);
  }

  async updateEndpoint(
    epEui: string,
    data: UpdateEndpointRequest,
  ): Promise<EndpointUI> {
    const response = await grpcClient.updateEndpoint(epEui, {
      name: data.name,
      epClass:
        data.bidi !== undefined
          ? data.bidi
            ? ENDPOINT_EP_CLASS.BIDIRECTIONAL
            : ENDPOINT_EP_CLASS.UNIDIRECTIONAL
          : undefined,
      shAddr: data.shAddr,
      nwkSnKey: data.nwkSnKey,
      appKey: data.appKey,
      preAttach: data.preAttach,
      dualChan: data.dualChan,
      repetition: data.repetition,
      wideCarrOff: data.wideCarrOff,
      longBlkDist: data.longBlkDist,
      lastPacketCnt: data.lastPacketCnt,
      attachCnt: data.attachCnt,
      carrierOffset: data.carrierOffset,
      typeEui: data.typeEui,
      deviceModelId: data.deviceModelId,
      newEpEui: data.newEpEui,
    });

    return mapGrpcEndpointToUI(response);
  }

  async deleteEndpoint(id: string): Promise<void> {
    await grpcClient.deleteEndpoint(id);
  }

  // ============================================================================
  // EVENTS
  // ============================================================================

  async getEvents(
    limit = 5,
    categories?: readonly string[],
    pageToken?: string,
    eventTypes?: readonly string[],
  ): Promise<{
    events: EventUI[];
    nextPageToken?: string;
    totalCount: number;
  }> {
    const response = await grpcClient.listEvents({
      pageSize: limit,
      categories,
      eventTypes,
      pageToken,
    });

    return {
      events: response.events.map((e) =>
        mapEvent({
          id: e.id,
          event_type: e.eventType,
          category: e.category,
          severity: e.severity as "error" | "info" | "success" | "warning",
          title: e.title,
          description: e.description,
          source_name: e.sourceName,
          message: e.description || e.title,
          timestamp: e.timestamp.toISOString(),
          createdAt: e.timestamp.toISOString(),
          metadata: e.data as Record<string, unknown> | undefined,
        }),
      ),
      nextPageToken: response.nextPageToken || undefined,
      totalCount: response.totalCount,
    };
  }

  // ============================================================================
  // CERTIFICATES
  // ============================================================================

  async getCertificates(): Promise<CertificateUI[]> {
    const response = await grpcClient.getServerCertificateStatus();
    const certs: CertificateUI[] = [];

    // Build certificates from serverCert and caCert objects
    if (response.serverCert) {
      certs.push(
        mapCertificate({
          type: CERTIFICATE_TYPES.SERVER,
          name: CERTIFICATE_LABELS.SERVER,
          path: "",
          subject: response.serverCert.subject,
          issuer: response.serverCert.issuer,
          expiryDate: response.serverCert.notAfter?.toISOString() || "",
          serialNumber: "", // Not available in gRPC response
        }),
      );
    }
    if (response.caCert) {
      certs.push(
        mapCertificate({
          type: CERTIFICATE_TYPES.CA,
          name: CERTIFICATE_LABELS.CA,
          path: "",
          subject: response.caCert.subject,
          issuer: response.caCert.issuer,
          expiryDate: response.caCert.notAfter?.toISOString() || "",
          serialNumber: "", // Not available in gRPC response
        }),
      );
    }
    return certs;
  }

  async generateCertificate(
    data: GenerateCertificateRequest,
  ): Promise<GenerateCertificateResponse> {
    const response = await grpcClient.generateCertificate({
      bsEui: data.bsEui,
      validityDays: data.validityDays,
    });

    return {
      bsEui: data.bsEui,
      serviceCenterUrl: response.serviceCenterUrl || "",
      downloadUrls: {
        caCert: response.downloadUrls?.["ca_cert"] || "",
        clientCert: response.downloadUrls?.["client_cert"] || "",
        privateKey: response.downloadUrls?.["private_key"] || "",
      },
      expiresAt: response.expiresAt?.toISOString() || "",
    };
  }

  /**
   * Download certificate as Blob for browser download
   */
  async downloadCertificate(
    certId: string,
    certType: CertificateDownloadType,
  ): Promise<{
    blob: Blob;
    filename: string;
  }> {
    const response = await grpcClient.downloadCertificate(certId, certType);
    return {
      blob: new Blob([new Uint8Array(response.content)], {
        type: response.contentType,
      }),
      filename: response.filename,
    };
  }

  async getCertificateStatus(): Promise<{
    serverCert?: {
      subject: string;
      issuer: string;
      notBefore: Date;
      notAfter: Date;
      daysUntilExpiry: number;
      isValid: boolean;
    };
    caCert?: {
      subject: string;
      issuer: string;
      notBefore: Date;
      notAfter: Date;
      daysUntilExpiry: number;
      isValid: boolean;
    };
  }> {
    return grpcClient.getServerCertificateStatus();
  }

  async generateServerCertificates(): Promise<{
    success: boolean;
    message: string;
  }> {
    return grpcClient.generateServerCertificates();
  }

  async renewServerCertificates(): Promise<{
    success: boolean;
    message: string;
  }> {
    return grpcClient.renewServerCertificates();
  }

  // ============================================================================
  // ANALYTICS & ALERTS
  // ============================================================================

  async getAnalyticsOverview(): Promise<AnalyticsOverviewAPI> {
    const response = await grpcClient.getAnalyticsOverview();
    return {
      totalMessages: response.totalMessages,
      activeEndpoints: response.activeEndpoints,
      activeBaseStations: response.activeBaseStations,
      messagesLast24h: 0,
      messagesLastWeek: 0,
    };
  }

  async getAlertSummary(): Promise<AlertSummaryAPI> {
    const response = await grpcClient.getAlertSummary();
    return {
      critical: response.critical,
      warning: response.warning,
      info: response.info,
      recent: response.recent.map((a) => ({
        id: a.id,
        level: a.severity as "critical" | "warning" | "info",
        message: a.title || a.description || "",
        timestamp: a.timestamp.toISOString(),
      })),
    };
  }

  async getSystemStatus(): Promise<SystemStatusResponse> {
    const response = await grpcClient.getSystemStatus();

    // Compute healthy from services when present
    const healthy =
      response.services.length > 0
        ? response.services.every((s) => s.healthy)
        : response.status === "healthy";

    // Derive timestamp from latest checkedAt (already has fallback from client)
    const timestamp =
      response.services.length > 0
        ? response.services
            .reduce(
              (latest, s) => (s.checkedAt > latest ? s.checkedAt : latest),
              response.services[0].checkedAt,
            )
            .toISOString()
        : response.uptime
          ? response.uptime.toISOString()
          : "";

    return {
      services: response.services.map((s) => ({
        name: s.name,
        url: s.url,
        healthy: s.healthy,
        latencyMs: s.latencyMs,
        error: s.error,
        checkedAt: s.checkedAt.toISOString(),
      })),
      healthy,
      timestamp,
    };
  }

  async sendULTransmit(params: {
    epEui: string;
    userData: string;
    packetCnt: number;
    nwkSnKey?: string;
    shAddr?: number;
    format?: number;
    profile?: string;
  }): Promise<{ id: string; status: string; message: string }> {
    const response = await grpcClient.sendULTransmit(params);
    return {
      id: response.id,
      status: response.status,
      message: response.message,
    };
  }

  async attachEndpoint(
    id: string,
  ): Promise<{ status: string; message?: string }> {
    const response = await grpcClient.attachEndpoint(id);
    return {
      status: response.status,
      message: undefined,
    };
  }

  async detachEndpoint(
    id: string,
  ): Promise<{ status: string; message?: string }> {
    const response = await grpcClient.detachEndpoint(id);
    return {
      status: response.status,
      message: undefined,
    };
  }

  // ============================================================================
  // DOWNLINK MANAGEMENT
  // ============================================================================

  async sendDownlink(
    params: import("@api-types/api").SendDownlinkRequest,
  ): Promise<import("@api-types/api").SendDownlinkResponse> {
    return grpcClient.sendDownlink(params);
  }

  async revokeDownlink(
    epEui: string,
    queueId: string,
  ): Promise<import("@api-types/api").RevokeDownlinkResponse> {
    return grpcClient.revokeDownlink({ epEui, queueId });
  }

  async listDownlinkQueue(
    epEui: string,
    pageSize: number,
    pageToken?: string,
  ): Promise<import("@api-types/api").DownlinkQueueResponse> {
    return grpcClient.listDownlinkQueue({ epEui, pageSize, pageToken });
  }

  async getDownlinkResults(
    epEui: string,
    pageSize: number,
    pageToken?: string,
    statusFilter?: string,
  ): Promise<import("@api-types/api").DownlinkResultsResponse> {
    return grpcClient.getDownlinkResults({
      epEui,
      statusFilter,
      pageSize,
      pageToken,
    });
  }

  // ============================================================================
  // SYSTEM & VERSION
  // ============================================================================

  async getVersion(): Promise<{
    version: string;
    buildTime: string;
    gitCommit: string;
    gitBranch: string;
    buildUser: string;
    goVersion: string;
    schemaVersion: number;
    artifacts: Record<string, string>;
    isProduction: boolean;
    edition?: string;
    editionCode?: string;
    licenseId?: string;
    licenseUrl?: string;
    sourceUrl?: string;
    documentationUrl?: string;
    homepageUrl?: string;
    trademarkNotice?: string;
  }> {
    const response = await grpcClient.getReleaseInfo();
    return {
      version: response.version,
      buildTime: response.buildTime,
      gitCommit: response.gitCommit,
      gitBranch: response.gitBranch,
      buildUser: response.buildUser,
      goVersion: response.goVersion,
      schemaVersion: response.schemaVersion,
      artifacts: response.artifacts,
      isProduction: false,
      edition: response.edition,
      editionCode: response.editionCode,
      licenseId: response.licenseId,
      licenseUrl: response.licenseUrl,
      sourceUrl: response.sourceUrl,
      documentationUrl: response.documentationUrl,
      homepageUrl: response.homepageUrl,
      trademarkNotice: response.trademarkNotice,
    };
  }

  // ============================================================================
  // Auth API Methods
  // ============================================================================

  async getAuthSettings(): Promise<AuthSettingsAPI> {
    const response = await grpcClient.getAuthSettings();
    return {
      enabled: response.enabled,
      local_login_enabled: response.localLoginEnabled,
      refresh_token_enabled: response.refreshTokenEnabled,
      registration_enabled: response.registrationEnabled,
      login_url: response.loginUrl,
      login_label: response.loginLabel,
      login_redirect: response.loginRedirect,
      logout_url: response.logoutUrl,
      oidc: response.oidc
        ? {
            enabled: response.oidc.enabled,
            login_url: response.oidc.loginUrl || "",
            login_label: response.oidc.loginLabel || "",
            login_redirect: response.oidc.loginRedirect,
            logout_url: response.oidc.logoutUrl,
          }
        : undefined,
      oauth2: response.oauth2
        ? {
            enabled: response.oauth2.enabled,
            login_url: response.oauth2.loginUrl || "",
            login_label: response.oauth2.loginLabel || "",
            login_redirect: response.oauth2.loginRedirect,
            logout_url: response.oauth2.logoutUrl,
          }
        : undefined,
    };
  }

  async getAuthProfile(): Promise<UserProfileAPI> {
    const response = await grpcClient.getProfile();
    return mapUserProfile({
      id: response.id,
      email: response.email,
      is_admin: response.isAdmin,
      has_password: response.hasPassword,
      memberships: response.memberships.map((m) => ({
        org_id: m.orgId,
        org_name: m.orgName,
        role: m.role,
        status: m.status,
        is_org_admin: m.isOrgAdmin,
        is_base_station_admin: m.isBaseStationAdmin,
        is_endpoint_admin: m.isEndpointAdmin,
      })),
      default_org_id: response.defaultOrgId,
      first_name: response.firstName,
      last_name: response.lastName,
    });
  }

  async login(payload: LoginRequest): Promise<LoginResponseAPI> {
    const response = await grpcClient.login(payload.email, payload.password);
    return mapLoginResponse({
      tokens: {
        access_token: response.tokens.accessToken,
        refresh_token: response.tokens.refreshToken,
        access_expires_in: response.tokens.accessExpiresIn,
        refresh_expires_in: response.tokens.refreshExpiresIn,
      },
      user: {
        id: response.user.id,
        email: response.user.email,
        is_admin: response.user.isAdmin,
        has_password: response.user.hasPassword,
        memberships: response.user.memberships.map((m) => ({
          org_id: m.orgId,
          org_name: m.orgName,
          role: m.role,
          status: m.status,
          is_org_admin: m.isOrgAdmin,
          is_base_station_admin: m.isBaseStationAdmin,
          is_endpoint_admin: m.isEndpointAdmin,
        })),
        default_org_id: response.user.defaultOrgId,
        first_name: response.user.firstName,
        last_name: response.user.lastName,
      },
    });
  }

  async registerAccount(
    payload: RegisterAccountRequest,
  ): Promise<LoginResponseAPI> {
    const response = await grpcClient.registerAccount(
      payload.email,
      payload.password,
      payload.firstName,
      payload.lastName,
      payload.companyName,
    );
    return mapLoginResponse({
      tokens: {
        access_token: response.tokens.accessToken,
        refresh_token: response.tokens.refreshToken,
        access_expires_in: response.tokens.accessExpiresIn,
        refresh_expires_in: response.tokens.refreshExpiresIn,
      },
      user: {
        id: response.user.id,
        email: response.user.email,
        is_admin: response.user.isAdmin,
        has_password: response.user.hasPassword,
        memberships: response.user.memberships.map((m) => ({
          org_id: m.orgId,
          org_name: m.orgName,
          role: m.role,
          status: m.status,
          is_org_admin: m.isOrgAdmin,
          is_base_station_admin: m.isBaseStationAdmin,
          is_endpoint_admin: m.isEndpointAdmin,
        })),
        default_org_id: response.user.defaultOrgId,
        first_name: response.user.firstName,
        last_name: response.user.lastName,
      },
    });
  }

  async exchangeOIDC(payload: ExchangeRequest): Promise<LoginResponseAPI> {
    const response = await grpcClient.exchangeOIDC(payload.code, payload.state);
    return mapLoginResponse({
      tokens: {
        access_token: response.tokens.accessToken,
        refresh_token: response.tokens.refreshToken,
        access_expires_in: response.tokens.accessExpiresIn,
        refresh_expires_in: response.tokens.refreshExpiresIn,
      },
      user: {
        id: response.user.id,
        email: response.user.email,
        is_admin: response.user.isAdmin,
        has_password: response.user.hasPassword,
        memberships: response.user.memberships.map((m) => ({
          org_id: m.orgId,
          org_name: m.orgName,
          role: m.role,
          status: m.status,
          is_org_admin: m.isOrgAdmin ?? false,
          is_base_station_admin: m.isBaseStationAdmin ?? false,
          is_endpoint_admin: m.isEndpointAdmin ?? false,
        })),
        default_org_id: response.user.defaultOrgId,
        first_name: response.user.firstName,
        last_name: response.user.lastName,
      },
    });
  }

  async exchangeOAuth2(payload: ExchangeRequest): Promise<LoginResponseAPI> {
    const response = await grpcClient.exchangeOAuth2(
      payload.code,
      payload.state,
    );
    return mapLoginResponse({
      tokens: {
        access_token: response.tokens.accessToken,
        refresh_token: response.tokens.refreshToken,
        access_expires_in: response.tokens.accessExpiresIn,
        refresh_expires_in: response.tokens.refreshExpiresIn,
      },
      user: {
        id: response.user.id,
        email: response.user.email,
        is_admin: response.user.isAdmin,
        has_password: response.user.hasPassword,
        memberships: response.user.memberships.map((m) => ({
          org_id: m.orgId,
          org_name: m.orgName,
          role: m.role,
          status: m.status,
          is_org_admin: m.isOrgAdmin ?? false,
          is_base_station_admin: m.isBaseStationAdmin ?? false,
          is_endpoint_admin: m.isEndpointAdmin ?? false,
        })),
        default_org_id: response.user.defaultOrgId,
        first_name: response.user.firstName,
        last_name: response.user.lastName,
      },
    });
  }

  async logout(refreshToken: string): Promise<void> {
    await grpcClient.logout(refreshToken);
  }

  async refreshTokens(refreshToken: string): Promise<{
    accessToken: string;
    refreshToken: string;
    accessExpiresIn: number;
    refreshExpiresIn: number;
  }> {
    return grpcClient.refreshTokens(refreshToken);
  }

  async changeOwnPassword(
    currentPassword: string,
    newPassword: string,
  ): Promise<void> {
    await grpcClient.changePassword(currentPassword, newPassword);
  }

  // ============================================================================
  // User Admin Methods
  // ============================================================================

  async getUsers(
    limit = 50,
    offset = 0,
  ): Promise<{
    users: import("@api-types/api").SystemUserUI[];
    total: number;
  }> {
    const response = await grpcClient.listUsers({ pageSize: limit + offset });
    const mapped = mapUserList(
      response.users.map((u) => ({
        id: u.id,
        email: u.email,
        is_admin: u.isAdmin,
        is_active: u.isActive,
        is_tenant_manager: u.isTenantManager,
        is_base_station_manager: u.isBaseStationManager,
        is_endpoint_manager: u.isEndpointManager,
        note: u.note,
        created_at: u.createdAt?.toISOString() || new Date().toISOString(),
        updated_at: u.updatedAt?.toISOString() || new Date().toISOString(),
      })),
    );
    return {
      users: mapped.slice(offset, offset + limit),
      total: response.totalCount,
    };
  }

  async getUser(
    id: string,
  ): Promise<import("@api-types/api").SystemUserUI | null> {
    const response = await grpcClient.getUser(id);
    if (!response) return null;

    return mapUser({
      id: response.id,
      email: response.email,
      is_admin: response.isAdmin,
      is_active: response.isActive,
      is_tenant_manager: response.isTenantManager,
      is_base_station_manager: response.isBaseStationManager,
      is_endpoint_manager: response.isEndpointManager,
      note: response.note,
      created_at: response.createdAt?.toISOString() || new Date().toISOString(),
      updated_at: response.updatedAt?.toISOString() || new Date().toISOString(),
    });
  }

  async createUser(
    data: import("@api-types/api").CreateUserRequest,
  ): Promise<import("@api-types/api").SystemUserUI> {
    const response = await grpcClient.createUser({
      email: data.email,
      password: data.password,
      isAdmin: data.is_admin,
      isActive: data.is_active,
      isTenantManager: data.is_tenant_manager,
      isBaseStationManager: data.is_base_station_manager,
      isEndpointManager: data.is_endpoint_manager,
      note: data.note,
    });

    return mapUser({
      id: response.id,
      email: response.email,
      is_admin: response.isAdmin ?? false,
      is_active: response.isActive ?? true,
      is_tenant_manager: response.isTenantManager ?? false,
      is_base_station_manager: response.isBaseStationManager ?? false,
      is_endpoint_manager: response.isEndpointManager ?? false,
      created_at: response.createdAt?.toISOString() || new Date().toISOString(),
      updated_at: response.updatedAt?.toISOString() || new Date().toISOString(),
    });
  }

  async updateUser(
    id: string,
    data: import("@api-types/api").UpdateUserRequest,
  ): Promise<import("@api-types/api").SystemUserUI> {
    const response = await grpcClient.updateUser(id, {
      email: data.email,
      note: data.note,
      isAdmin: data.is_admin,
      isActive: data.is_active,
      isTenantManager: data.is_tenant_manager,
      isBaseStationManager: data.is_base_station_manager,
      isEndpointManager: data.is_endpoint_manager,
    });

    return mapUser({
      id: response.id,
      email: response.email,
      note: response.note,
      is_admin: response.isAdmin ?? false,
      is_active: response.isActive ?? true,
      is_tenant_manager: response.isTenantManager ?? false,
      is_base_station_manager: response.isBaseStationManager ?? false,
      is_endpoint_manager: response.isEndpointManager ?? false,
      created_at: response.createdAt?.toISOString() || new Date().toISOString(),
      updated_at: response.updatedAt?.toISOString() || new Date().toISOString(),
    });
  }

  async deleteUser(id: string): Promise<void> {
    await grpcClient.deleteUser(id);
  }

  async changeUserPassword(id: string, password: string): Promise<void> {
    await grpcClient.updateUserPassword(id, password);
  }

  // ============================================================================
  // Organization Admin Methods
  // ============================================================================

  async getOrganizations(
    limit = 50,
    offset = 0,
  ): Promise<{
    organizations: import("@api-types/api").OrganizationUI[];
    total: number;
  }> {
    const pageSize = limit + offset;
    const response = await grpcClient.listOrganizations({ pageSize });
    const organizations = mapOrganizationList(
      response.organizations.map((o) => ({
        id: o.id,
        name: o.name,
        description: o.description,
        tenant_id: parseInt(o.tenantId, 10) || 0,
        state: o.state || "active",
        can_have_base_stations: o.canHaveBaseStations ?? true,
        created_at: o.createdAt?.toISOString() || new Date().toISOString(),
        updated_at: o.updatedAt?.toISOString() || new Date().toISOString(),
      })),
    );
    return {
      organizations: organizations.slice(offset, offset + limit),
      total: response.totalCount,
    };
  }

  async getOrganization(
    id: string,
  ): Promise<import("@api-types/api").OrganizationUI | null> {
    const response = await grpcClient.getOrganization(id);
    if (!response) return null;

    return mapOrganization({
      id: response.id,
      name: response.name,
      description: response.description,
      tenant_id: parseInt(response.tenantId, 10) || 0,
      state: response.state || "active",
      can_have_base_stations: response.canHaveBaseStations ?? true,
      created_at: response.createdAt?.toISOString() || new Date().toISOString(),
      updated_at: response.updatedAt?.toISOString() || new Date().toISOString(),
    });
  }

  async createOrganization(
    data: import("@api-types/api").CreateOrganizationRequest,
  ): Promise<import("@api-types/api").OrganizationUI> {
    const response = await grpcClient.createOrganization({
      name: data.name,
      description: data.description,
    });

    return mapOrganization({
      id: response.id,
      name: response.name,
      description: response.description,
      tenant_id: parseInt(response.tenantId, 10) || 0,
      state: "active",
      can_have_base_stations: true,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });
  }

  async updateOrganization(
    id: string,
    data: import("@api-types/api").UpdateOrganizationRequest,
  ): Promise<import("@api-types/api").OrganizationUI> {
    const response = await grpcClient.updateOrganization(id, {
      name: data.name,
      description: data.description,
    });

    return mapOrganization({
      id: response.id,
      name: response.name,
      description: response.description,
      tenant_id: parseInt(response.tenantId, 10) || 0,
      state: "active",
      can_have_base_stations: true,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });
  }

  async deleteOrganization(id: string): Promise<void> {
    await grpcClient.deleteOrganization(id);
  }

  // ============================================================================
  // Organization User Admin Methods
  // ============================================================================

  async getOrgUsers(
    orgId: string,
    status?: string,
  ): Promise<{
    users: import("@api-types/api").OrganizationUserUI[];
    total: number;
  }> {
    const response = await grpcClient.listOrganizationUsers(orgId, { status });
    return {
      users: mapOrgUserList(
        response.users.map((u) => ({
          user_id: u.userId,
          org_id: u.orgId,
          role: u.role,
          status: u.status,
          email: u.email,
          is_org_admin: u.isOrgAdmin ?? false,
          is_base_station_admin: u.isBaseStationAdmin ?? false,
          is_endpoint_admin: u.isEndpointAdmin ?? false,
          created_at: u.createdAt?.toISOString() || new Date().toISOString(),
          updated_at: u.updatedAt?.toISOString() || new Date().toISOString(),
        })),
      ),
      total: response.totalCount,
    };
  }

  async getOrgUser(
    orgId: string,
    userId: string,
  ): Promise<import("@api-types/api").OrganizationUserUI | null> {
    const response = await grpcClient.getOrganizationUser(orgId, userId);
    if (!response) return null;

    return mapOrgUser({
      user_id: response.userId,
      org_id: response.orgId,
      role: response.role,
      status: response.status,
      email: response.email,
      is_org_admin: response.isOrgAdmin ?? false,
      is_base_station_admin: response.isBaseStationAdmin ?? false,
      is_endpoint_admin: response.isEndpointAdmin ?? false,
      created_at: response.createdAt?.toISOString() || new Date().toISOString(),
      updated_at: response.updatedAt?.toISOString() || new Date().toISOString(),
    });
  }

  async addOrgUser(
    orgId: string,
    data: import("@api-types/api").AddOrgUserRequest,
  ): Promise<import("@api-types/api").OrganizationUserUI> {
    const response = await grpcClient.addOrganizationUser(orgId, {
      userId: data.user_id,
      email: data.email,
      role: data.role,
      isOrgAdmin: data.is_org_admin,
      isBaseStationAdmin: data.is_base_station_admin,
      isEndpointAdmin: data.is_endpoint_admin,
    });

    return mapOrgUser({
      user_id: response.userId,
      org_id: response.orgId,
      role: response.role,
      status: response.status,
      email: response.email || "",
      is_org_admin: response.isOrgAdmin ?? false,
      is_base_station_admin: response.isBaseStationAdmin ?? false,
      is_endpoint_admin: response.isEndpointAdmin ?? false,
      created_at: response.createdAt?.toISOString() || "",
      updated_at: response.updatedAt?.toISOString() || "",
    });
  }

  async updateOrgUser(
    orgId: string,
    userId: string,
    data: import("@api-types/api").UpdateOrgUserRequest,
  ): Promise<import("@api-types/api").OrganizationUserUI> {
    const response = await grpcClient.updateOrganizationUser(orgId, userId, {
      role: data.role,
      isOrgAdmin: data.is_org_admin,
      isBaseStationAdmin: data.is_base_station_admin,
      isEndpointAdmin: data.is_endpoint_admin,
    });

    return mapOrgUser({
      user_id: response.userId,
      org_id: response.orgId,
      role: response.role,
      status: response.status,
      email: response.email || "",
      is_org_admin: response.isOrgAdmin ?? false,
      is_base_station_admin: response.isBaseStationAdmin ?? false,
      is_endpoint_admin: response.isEndpointAdmin ?? false,
      created_at: response.createdAt?.toISOString() || "",
      updated_at: response.updatedAt?.toISOString() || "",
    });
  }

  async removeOrgUser(orgId: string, userId: string): Promise<void> {
    await grpcClient.removeOrganizationUser(orgId, userId);
  }

  async listUserOrganizations(userId: string): Promise<{
    memberships: Array<{
      orgId: string;
      orgName: string;
      role: string;
      status: string;
    }>;
  }> {
    return grpcClient.listUserOrganizations(userId);
  }

  // ============================================================================
  // Blueprint Feature: Manufacturers, Device Models, Blueprints
  // ============================================================================

  async getManufacturers(): Promise<ManufacturerUI[]> {
    const response = await grpcClient.listManufacturers();
    return response.map((m) => ({
      id: m.id,
      tenantId: parseTenantIdOrZero(m.tenantId),
      name: m.name,
      website: m.website,
      isVerified: m.isVerified,
      modelCount: m.modelCount,
      createdAt: m.createdAt?.toISOString() || "",
      updatedAt: m.updatedAt?.toISOString() || "",
    }));
  }

  async getManufacturer(id: string): Promise<ManufacturerUI | null> {
    const response = await grpcClient.getManufacturer(id);
    if (!response) return null;

    return {
      id: response.id,
      tenantId: parseTenantIdOrZero(response.tenantId),
      name: response.name,
      website: response.website,
      isVerified: response.isVerified,
      modelCount: response.modelCount,
      createdAt: response.createdAt?.toISOString() || "",
      updatedAt: response.updatedAt?.toISOString() || "",
    };
  }

  async createManufacturer(
    data: CreateManufacturerRequest,
  ): Promise<ManufacturerUI> {
    const response = await grpcClient.createManufacturer({
      name: data.name,
      website: data.website,
    });

    return {
      id: response.id,
      tenantId: parseTenantIdOrZero(response.tenantId),
      name: response.name,
      website: response.website,
      isVerified: response.isVerified,
      modelCount: response.modelCount,
      createdAt: response.createdAt?.toISOString() || "",
      updatedAt: response.updatedAt?.toISOString() || "",
    };
  }

  async updateManufacturer(
    id: string,
    data: UpdateManufacturerRequest,
  ): Promise<void> {
    await grpcClient.updateManufacturer(id, {
      name: data.name,
      website: data.website,
    });
  }

  async deleteManufacturer(id: string): Promise<void> {
    await grpcClient.deleteManufacturer(id);
  }

  // ============================================================================
  // Device Models
  // ============================================================================

  async getDeviceModels(manufacturerId: string): Promise<DeviceModelUI[]> {
    const response = await grpcClient.listDeviceModels(manufacturerId);
    return response.map((m) => ({
      id: m.id,
      manufacturerId: m.manufacturerId,
      tenantId: parseTenantIdOrZero(m.tenantId),
      name: m.name,
      code: m.code,
      typeEui: m.typeEui,
      description: m.description,
      datasheetUrl: m.datasheetUrl,
      blueprintCount: m.blueprintCount,
      createdAt: m.createdAt?.toISOString() || "",
      updatedAt: m.updatedAt?.toISOString() || "",
    }));
  }

  async getDeviceModel(id: string): Promise<DeviceModelUI | null> {
    const response = await grpcClient.getDeviceModel(id);
    if (!response) return null;

    return {
      id: response.id,
      manufacturerId: response.manufacturerId,
      tenantId: parseTenantIdOrZero(response.tenantId),
      name: response.name,
      code: response.code,
      typeEui: response.typeEui,
      description: response.description,
      datasheetUrl: response.datasheetUrl,
      blueprintCount: response.blueprintCount,
      createdAt: response.createdAt?.toISOString() || "",
      updatedAt: response.updatedAt?.toISOString() || "",
    };
  }

  async createDeviceModel(
    manufacturerId: string,
    data: CreateDeviceModelRequest,
  ): Promise<DeviceModelUI> {
    const response = await grpcClient.createDeviceModel(manufacturerId, {
      name: data.name,
      code: data.code,
      typeEui: data.typeEui,
      description: data.description,
    });

    return {
      id: response.id,
      manufacturerId: response.manufacturerId,
      tenantId: parseTenantIdOrZero(response.tenantId),
      name: response.name,
      code: response.code,
      typeEui: response.typeEui,
      description: response.description,
      datasheetUrl: response.datasheetUrl,
      blueprintCount: response.blueprintCount,
      createdAt: response.createdAt?.toISOString() || "",
      updatedAt: response.updatedAt?.toISOString() || "",
    };
  }

  async updateDeviceModel(
    id: string,
    data: UpdateDeviceModelRequest,
  ): Promise<void> {
    await grpcClient.updateDeviceModel(id, {
      name: data.name,
      description: data.description,
    });
  }

  async deleteDeviceModel(id: string): Promise<void> {
    await grpcClient.deleteDeviceModel(id);
  }

  /**
   * Create a device model with a default blueprint atomically.
   * The server generates the model code slug automatically.
   */
  async createDeviceModelWithBlueprint(
    data: CreateDeviceModelWithBlueprintRequest,
  ): Promise<{
    model: DeviceModelUI;
    blueprint: BlueprintUI;
  }> {
    const response = await grpcClient.createDeviceModelWithBlueprint({
      manufacturerId: data.manufacturerId,
      name: data.name,
      version: data.version,
      decoderScript: JSON.stringify(data.specJson),
    });

    return {
      model: {
        id: response.deviceModel.id,
        manufacturerId: response.deviceModel.manufacturerId,
        tenantId: parseTenantIdOrZero(response.deviceModel.tenantId),
        name: response.deviceModel.name,
        code: response.deviceModel.code,
        typeEui: response.deviceModel.typeEui,
        description: response.deviceModel.description,
        datasheetUrl: response.deviceModel.datasheetUrl,
        blueprintCount: response.deviceModel.blueprintCount || 1,
        createdAt: response.deviceModel.createdAt?.toISOString() || "",
        updatedAt: response.deviceModel.updatedAt?.toISOString() || "",
      },
      blueprint: mapTransportBlueprintToUI(response.blueprint),
    };
  }

  // ============================================================================
  // Blueprints
  // ============================================================================

  async getBlueprints(modelId: string): Promise<BlueprintListItemAPI[]> {
    const response = await grpcClient.listBlueprints(modelId);
    return response.map((b: BlueprintTransportDTO) => ({
      id: b.id,
      deviceModelId: b.deviceModelId,
      version: b.version,
      typeEui: b.typeEui,
      isDefault: b.isDefault,
      createdAt: b.createdAt?.toISOString() || "",
    }));
  }

  async getBlueprint(id: string): Promise<BlueprintUI | null> {
    const response = await grpcClient.getBlueprint(id);
    if (!response) return null;

    return mapTransportBlueprintToUI(response);
  }

  async createBlueprint(
    modelId: string,
    data: CreateBlueprintRequest,
  ): Promise<BlueprintUI> {
    const response = await grpcClient.createBlueprint(modelId, {
      name: data.version,
      version: data.version,
      decoderScript: JSON.stringify(data.specJson),
    });

    return mapTransportBlueprintToUI(response);
  }

  async updateBlueprint(
    id: string,
    data: UpdateBlueprintRequest,
  ): Promise<void> {
    await grpcClient.updateBlueprint(id, {
      version: data.version,
      decoderScript: data.specJson ? JSON.stringify(data.specJson) : undefined,
    });
  }

  async deleteBlueprint(id: string): Promise<void> {
    await grpcClient.deleteBlueprint(id);
  }

  async setBlueprintDefault(id: string): Promise<void> {
    await grpcClient.setDefaultBlueprint(id);
  }

  async decodePreview(
    id: string,
    data: DecodePreviewRequest,
  ): Promise<DecodePreviewResponse> {
    const payloadBytes = parseDecodePreviewPayload(data.userData);
    const result = await grpcClient.decodePreview(
      id,
      payloadBytes,
      data.formatId || 0,
    );

    return {
      success: result.success,
      decodedData: result.decodedPayload,
      errorCode: result.errorCode,
      errorDetail: result.errorDetail,
      formatId: result.formatId,
    };
  }

  async submitToRegistry(
    id: string,
    data: RegistrySubmitRequest,
  ): Promise<RegistrySubmitResponse> {
    const response = await grpcClient.submitBlueprintToRegistry(id, {
      contributorName: data.contributorName,
      contributorEmail: data.contributorEmail,
      notes: data.description,
    });

    return {
      prUrl: response.prUrl,
      commitSha: response.commitSha || "",
      branch: response.branchName || REGISTRY_DEFAULTS.BRANCH,
    };
  }

  // ============================================================================
  // API Keys
  // ============================================================================

  /**
   * List API keys for the current organization
   */
  async listApiKeys(params?: {
    pageSize?: number;
    pageToken?: string;
    userId?: string;
  }): Promise<ApiKeyListResponse> {
    const response = await grpcClient.listApiKeys(params);
    return {
      apiKeys: response.apiKeys.map(
        (k): ApiKeyAPI => ({
          id: k.id,
          orgId: k.orgId,
          userId: k.userId,
          name: k.name,
          keyPrefix: k.keyPrefix,
          keyType: k.keyType,
          isActive: k.isActive,
          expiresAt: k.expiresAt?.toISOString(),
          lastUsedAt: k.lastUsedAt?.toISOString(),
          createdAt: k.createdAt?.toISOString() || new Date().toISOString(),
        }),
      ),
      nextPageToken: response.nextPageToken,
      totalCount: response.totalCount,
    };
  }

  /**
   * Create a new API key.
   * Returns the raw key only on creation (never stored).
   */
  async createApiKey(data: {
    name: string;
    keyType: string;
    expiresAt?: Date;
  }): Promise<ApiKeyCreateResponse> {
    const response = await grpcClient.createApiKey(data);
    const k = response.apiKey;
    return {
      apiKey: {
        id: k.id,
        orgId: k.orgId,
        userId: k.userId,
        name: k.name,
        keyPrefix: k.keyPrefix,
        keyType: k.keyType,
        isActive: k.isActive,
        expiresAt: k.expiresAt?.toISOString(),
        createdAt: k.createdAt?.toISOString() || new Date().toISOString(),
      },
      rawKey: response.rawKey,
    };
  }

  /**
   * Delete an API key
   */
  async deleteApiKey(id: string): Promise<boolean> {
    return grpcClient.deleteApiKey(id);
  }

  // ============================================================================
  // Endpoint Messages & Events
  // ============================================================================

  /**
   * Get endpoint messages via gRPC
   */
  async getEndpointMessages(
    epEui: string,
    page: number,
    pageSize: number,
  ): Promise<{
    messages: BaseStationMessageAPI[];
    pagination: {
      page: number;
      pageSize: number;
      totalCount: number;
      totalPages: number;
    };
  }> {
    // Convert 1-based page number to offset-based pageToken for gRPC pagination
    const offset = (page - 1) * pageSize;
    const pageToken = offset > 0 ? String(offset) : undefined;

    const response = await grpcClient.listEndpointMessages(epEui, {
      pageSize,
      pageToken,
    });
    return {
      messages: response.messages.map(
        (m): BaseStationMessageAPI => ({
          id: m.id,
          epEui: m.epEui,
          bsEui: m.bsEui,
          packetCnt: m.packetCounter,
          userData: m.payload,
          rssi: m.rssi,
          snr: m.snr,
          receivedAt: m.receivedAt?.toISOString() || "",
          messageType: "ulData",
          direction: "uplink" as const,
          decodedPayload: m.decodedPayload,
          decodeStatus: m.decodeStatus,
          decodeErrorCode: m.decodeErrorCode,
          duplicate: m.duplicate,
          baseStations: m.baseStations?.map(
            (bs): BaseStationReceptionAPI => ({
              bsEui: bs.bsEui,
              rxTime: bs.rxTime,
              snr: bs.snr,
              rssi: bs.rssi,
              eqSnr: bs.eqSnr,
              rxDuration: bs.rxDuration,
              profile: bs.profile,
              mode: bs.mode,
              dlRxSnr: bs.dlRxSnr,
              dlRxRssi: bs.dlRxRssi,
              subpackets: bs.subpackets,
            }),
          ),
        }),
      ),
      pagination: {
        page,
        pageSize,
        totalCount: response.totalCount,
        totalPages: Math.ceil(response.totalCount / pageSize),
      },
    };
  }

  /**
   * Get endpoint message statistics
   */
  async getEndpointStats(epEui: string): Promise<{
    epEui: string;
    totalCount: number;
    uniqueEndpoints: number;
    avgRssi: number;
    avgSnr: number;
    firstSeen?: string;
    lastSeen?: string;
    activeDays: number;
    attachStatus: string;
  } | null> {
    const result = await grpcClient.getEndpointStats(epEui);
    if (!result) return null;
    return {
      epEui: result.epEui,
      totalCount: result.totalCount,
      uniqueEndpoints: result.uniqueEndpoints,
      avgRssi: result.avgRssi,
      avgSnr: result.avgSnr,
      firstSeen: result.firstSeen?.toISOString(),
      lastSeen: result.lastSeen?.toISOString(),
      activeDays: result.activeDays,
      attachStatus: result.attachStatus,
    };
  }

  /**
   * Get endpoint operations history
   */
  async getEndpointOperations(
    epEui: string,
    page = 1,
    pageSize = 50,
  ): Promise<{
    operations: Array<{
      id: string;
      eventType: string;
      category: string;
      severity: string;
      title: string;
      createdAt: string;
    }>;
  }> {
    const offset = (page - 1) * pageSize;
    const response = await grpcClient.getEndpointOperations(epEui, {
      pageSize,
      offset: offset > 0 ? offset : undefined,
    });
    return {
      operations: response.operations.map((op) => ({
        id: op.id,
        eventType: op.eventType,
        category: op.category,
        severity: op.severity,
        title: op.title,
        createdAt: op.createdAt?.toISOString() || "",
      })),
    };
  }

  // ============================================================================
  // INTEGRATIONS
  // ============================================================================

  /**
   * Create an integration
   */
  async createIntegration(data: {
    name: string;
    description?: string;
    type: string;
    config: Record<string, unknown>;
    eventFilter?: Record<string, unknown>;
    deliveryFormat?: string;
  }): Promise<{
    id: number;
    name: string;
    description?: string;
    type: string;
    config?: Record<string, unknown>;
    eventFilter?: Record<string, unknown>;
    deliveryFormat: string;
    status: string;
    createdAt: string;
    updatedAt: string;
  }> {
    const result = await grpcClient.createIntegration(data);
    return {
      id: result.id,
      name: result.name,
      description: result.description,
      type: result.type,
      config: result.config,
      eventFilter: result.eventFilter,
      deliveryFormat: result.deliveryFormat,
      status: result.status,
      createdAt: result.createdAt?.toISOString() || "",
      updatedAt: result.updatedAt?.toISOString() || "",
    };
  }

  /**
   * Get integration by ID
   */
  async getIntegration(id: number): Promise<{
    id: number;
    name: string;
    description?: string;
    type: string;
    deliveryFormat: string;
    status: string;
    createdAt: string;
    updatedAt: string;
  } | null> {
    const result = await grpcClient.getIntegration(id);
    if (!result) return null;
    return {
      id: result.id,
      name: result.name,
      description: result.description,
      type: result.type,
      deliveryFormat: result.deliveryFormat,
      status: result.status,
      createdAt: result.createdAt?.toISOString() || "",
      updatedAt: result.updatedAt?.toISOString() || "",
    };
  }

  /**
   * Update an integration
   */
  async updateIntegration(
    id: number,
    data: {
      name?: string;
      description?: string;
      status?: string;
    },
  ): Promise<{
    id: number;
    name: string;
    description?: string;
    type: string;
    deliveryFormat: string;
    status: string;
    createdAt: string;
    updatedAt: string;
  }> {
    const result = await grpcClient.updateIntegration(id, data);
    return {
      id: result.id,
      name: result.name,
      description: result.description,
      type: result.type,
      deliveryFormat: result.deliveryFormat,
      status: result.status,
      createdAt: result.createdAt?.toISOString() || "",
      updatedAt: result.updatedAt?.toISOString() || "",
    };
  }

  /**
   * Delete an integration
   */
  async deleteIntegration(id: number): Promise<void> {
    await grpcClient.deleteIntegration(id);
  }

  /**
   * List integrations with pagination
   */
  async listIntegrations(
    page = 1,
    pageSize = 50,
  ): Promise<{
    integrations: Array<{
      id: number;
      name: string;
      description?: string;
      type: string;
      deliveryFormat: string;
      status: string;
      createdAt: string;
      updatedAt: string;
    }>;
    totalCount: number;
  }> {
    const offset = (page - 1) * pageSize;
    const response = await grpcClient.listIntegrations({
      pageSize,
      offset: offset > 0 ? offset : undefined,
    });
    return {
      integrations: response.integrations.map((i) => ({
        id: i.id,
        name: i.name,
        description: i.description,
        type: i.type,
        deliveryFormat: i.deliveryFormat,
        status: i.status,
        createdAt: i.createdAt?.toISOString() || "",
        updatedAt: i.updatedAt?.toISOString() || "",
      })),
      totalCount: response.totalCount,
    };
  }

  // ---------------------------------------------------------------------------
  // Global Coverage (server admin only)
  // ---------------------------------------------------------------------------

  async listAllBaseStationLocations(): Promise<ListAllBaseStationLocationsResponse> {
    const response = await grpcClient.listAllBaseStationLocations();
    return {
      locations: response.locations,
      totalCount: response.totalCount,
    };
  }
}

// Export singleton instance
export const api = new ApiService();

// For backward compatibility
export const apiService = api;
