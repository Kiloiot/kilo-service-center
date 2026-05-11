/**
 * gRPC-web Client Wrapper (INTERNAL ONLY)
 *
 * This module is INTERNAL to the services layer. External code should
 * import from '@services/api' which delegates to this client internally.
 *
 * Provides Promise-based APIs wrapping the generated gRPC-web client.
 * Handles metadata injection (auth token, organization ID) and error conversion.
 *
 * Primary API client implementation.
 */

import { grpc } from '@improbable-eng/grpc-web';
import { Empty } from 'google-protobuf/google/protobuf/empty_pb';
import { FieldMask } from 'google-protobuf/google/protobuf/field_mask_pb';
import * as google_protobuf_timestamp_pb from 'google-protobuf/google/protobuf/timestamp_pb';
import { DoubleValue } from 'google-protobuf/google/protobuf/wrappers_pb';

import * as pb from '@services/grpc/kilocenter_pb';
import type { ServiceError } from '@services/grpc/kilocenter_pb_service';
import * as ServiceModule from '@services/grpc/kilocenter_pb_service';
const { KiloCenterServiceClient } = ServiceModule;
import { bytesToHex, hexToBytes } from '@utils/formatters';
import { storageService } from '@utils/storage';
import type { CertificateDownloadType } from '@constants/app';
import {
  ENDPOINT_UPDATE_MASK_PATHS,
  HEADER_VALUES,
  HEADERS,
  ORG_USER_UPDATE_MASK_PATHS,
  STORAGE_KEYS,
  USER_UPDATE_MASK_PATHS,
} from '@constants/app';
import { ERR_AUTH_ORG_REQUIRED, ERR_UNAUTHORIZED, GRPC_CLIENT_ERRORS } from '@constants/messages';
import { grpcUrl } from '@config/env';

// gRPC error codes mapping to HTTP-like status codes
const GRPC_TO_HTTP_STATUS: Record<number, number> = {
  [grpc.Code.OK]: 200,
  [grpc.Code.Canceled]: 499,
  [grpc.Code.Unknown]: 500,
  [grpc.Code.InvalidArgument]: 400,
  [grpc.Code.DeadlineExceeded]: 504,
  [grpc.Code.NotFound]: 404,
  [grpc.Code.AlreadyExists]: 409,
  [grpc.Code.PermissionDenied]: 403,
  [grpc.Code.ResourceExhausted]: 429,
  [grpc.Code.FailedPrecondition]: 412,
  [grpc.Code.Aborted]: 409,
  [grpc.Code.OutOfRange]: 400,
  [grpc.Code.Unimplemented]: 501,
  [grpc.Code.Internal]: 500,
  [grpc.Code.Unavailable]: 503,
  [grpc.Code.DataLoss]: 500,
  [grpc.Code.Unauthenticated]: 401,
};

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();

function stringToBytes(value: string): Uint8Array {
  return textEncoder.encode(value);
}

function bytesToString(value: Uint8Array): string {
  return textDecoder.decode(value);
}

/** gRPC-layer API key (timestamps as Date from protobuf Timestamp). */
export interface GrpcApiKey {
  id: string;
  orgId: string;
  userId: string;
  name: string;
  keyPrefix: string;
  keyType: string;
  isActive: boolean;
  expiresAt?: Date;
  lastUsedAt?: Date;
  createdAt: Date;
}

/**
 * Blueprint transport DTO — maps protobuf fields to a plain transport shape.
 * grpc/client.ts knows only proto shapes; api.ts maps to UI types.
 */
export interface BlueprintTransportDTO {
  id: string;
  deviceModelId: string;
  name: string;
  version: string;
  description?: string;
  decoderScript: string;
  typeEui: string;
  specJson: string;
  isDefault: boolean;
  registryRepo?: string;
  registryCommitSha?: string;
  registryVerified: boolean;
  registryPrUrl?: string;
  createdAt?: Date;
  updatedAt?: Date;
}

/**
 * Maps a protobuf Blueprint message to the transport DTO.
 * Canonical spec_json (field 11) takes precedence; falls back to decoder_script (field 6).
 */
function mapProtoBlueprintToTransport(b: pb.Blueprint): BlueprintTransportDTO {
  const specJsonRaw = bytesToString(b.getSpecJson_asU8());
  const decoderScriptRaw = bytesToString(b.getDecoderScript_asU8());
  const specJson = specJsonRaw || decoderScriptRaw;

  return {
    id: b.getId(),
    deviceModelId: b.getDeviceModelId(),
    name: b.getName(),
    version: b.getVersion(),
    description: b.getDescription() || undefined,
    decoderScript: decoderScriptRaw,
    typeEui: b.getTypeEui(),
    specJson,
    isDefault: b.getIsDefault(),
    registryRepo: b.getRegistryRepo() || undefined,
    registryCommitSha: b.getRegistryCommitSha() || undefined,
    registryVerified: b.getRegistryVerified(),
    registryPrUrl: b.getRegistryPrUrl() || undefined,
    createdAt: b.getCreatedAt()?.toDate(),
    updatedAt: b.getUpdatedAt()?.toDate(),
  };
}

/**
 * gRPC API Error compatible with existing ApiError interface
 */
export class GrpcApiError extends Error {
  readonly status: number;
  readonly details?: unknown;
  readonly code?: string;
  readonly token?: string;
  readonly grpcCode?: number;

  constructor(
    status: number,
    message: string,
    details?: unknown,
    code?: string,
    token?: string,
    grpcCode?: number
  ) {
    super(message);
    this.name = 'GrpcApiError';
    this.status = status;
    this.details = details;
    this.code = code;
    this.token = token;
    this.grpcCode = grpcCode;
  }

  isNotFound(): boolean {
    return this.status === 404 || this.grpcCode === grpc.Code.NotFound;
  }

  isUnauthorized(): boolean {
    return this.status === 401 || this.grpcCode === grpc.Code.Unauthenticated;
  }

  isForbidden(): boolean {
    return this.status === 403 || this.grpcCode === grpc.Code.PermissionDenied;
  }

  isAlreadyExists(): boolean {
    return this.grpcCode === grpc.Code.AlreadyExists;
  }

  isInvalidArgument(): boolean {
    return this.status === 400 || this.grpcCode === grpc.Code.InvalidArgument;
  }
}

/**
 * Callback type for auth failure handling (401 responses)
 */
type AuthFailureCallback = () => void;

/**
 * Shared return type for all gRPC endpoint methods.
 * Single source of truth — prevents field drift across list/get/create/update.
 */
export interface GrpcEndpointResult {
  epEui: string;
  tenantId: string;
  name: string;
  description?: string;
  epClass: string;
  status: string;
  attachStatus?: string;
  shAddr?: number;
  attachCnt?: number;
  lastPacketCnt?: number;
  preAttach?: boolean;
  carrierOffset?: number;
  dualChan?: boolean;
  repetition?: boolean;
  wideCarrOff?: boolean;
  longBlkDist?: boolean;
  deviceModelId?: string;
  typeEui?: string;
  nwkSnKey?: string;
  appKey?: string;
  tags: Record<string, string>;
  createdAt?: Date;
  updatedAt?: Date;
  lastSeenAt?: Date;
}

/**
 * Maps a protobuf EndPoint message to the shared GrpcEndpointResult shape.
 */
function mapEndpointProto(ep: pb.EndPoint): GrpcEndpointResult {
  return {
    epEui: ep.getEpeui(),
    tenantId: String(ep.getTenantId()),
    name: ep.getName(),
    description: ep.getDescription() || undefined,
    epClass: ep.getEpClass(),
    status: ep.getStatus(),
    attachStatus: ep.getAttachStatus() || undefined,
    shAddr: ep.getShAddr(),
    attachCnt: ep.getAttachCnt(),
    lastPacketCnt: ep.getLastPacketCnt(),
    preAttach: ep.getPreAttach(),
    carrierOffset: ep.getCarrierOffset(),
    dualChan: ep.getDualChan(),
    repetition: ep.getRepetition(),
    wideCarrOff: ep.getWideCarrOff(),
    longBlkDist: ep.getLongBlkDist(),
    deviceModelId: ep.getDeviceModelId() || undefined,
    typeEui: (() => {
      const t = ep.getTypeEui_asU8();
      return t.length > 0 ? bytesToHex(t) : undefined;
    })(),
    nwkSnKey: (() => {
      const k = ep.getNwkSnKey_asU8();
      return k.length > 0 ? bytesToHex(k) : undefined;
    })(),
    appKey: (() => {
      const k = ep.getAppKey_asU8();
      return k.length > 0 ? bytesToHex(k) : undefined;
    })(),
    tags: ep
      .getTagsMap()
      .toObject()
      .reduce(
        (acc, [k, v]) => {
          acc[k] = v;
          return acc;
        },
        {} as Record<string, string>
      ),
    createdAt: ep.getCreatedAt()?.toDate(),
    updatedAt: ep.getUpdatedAt()?.toDate(),
    lastSeenAt: ep.getLastSeenAt()?.toDate(),
  };
}

// Module-private input shape for createEndpoint; isolates the long field
// list from the public method signature.
interface CreateEndpointInput {
  epEui: string;
  name?: string;
  description?: string;
  epClass?: string;
  nwkSnKey?: string;
  appSnKey?: string;
  shAddr?: number;
  dualChan?: boolean;
  repetition?: boolean;
  wideCarrOff?: boolean;
  longBlkDist?: boolean;
  attachCnt?: number;
  preAttach?: boolean;
  lastPacketCnt?: number;
  typeEui?: string;
  carrierOffset?: number;
  deviceModelId?: string;
}

// String fields with truthy check (empty string skipped, no conversion).
function applyEndpointCreateStringFields(endpoint: pb.EndPoint, data: CreateEndpointInput): void {
  if (data.name) endpoint.setName(data.name);
  if (data.description) endpoint.setDescription(data.description);
  if (data.epClass) endpoint.setEpClass(data.epClass);
  if (data.deviceModelId) endpoint.setDeviceModelId(data.deviceModelId);
}

// Hex-string fields with truthy check + hex→bytes conversion.
function applyEndpointCreateHexFields(endpoint: pb.EndPoint, data: CreateEndpointInput): void {
  if (data.nwkSnKey) endpoint.setNwkSnKey(hexToBytes(data.nwkSnKey));
  if (data.appSnKey) endpoint.setAppKey(hexToBytes(data.appSnKey));
  if (data.typeEui) endpoint.setTypeEui(hexToBytes(data.typeEui));
}

// Numeric fields with `!== undefined` check (preserves zero values).
function applyEndpointCreateNumericFields(endpoint: pb.EndPoint, data: CreateEndpointInput): void {
  if (data.shAddr !== undefined) endpoint.setShAddr(data.shAddr);
  if (data.attachCnt !== undefined) endpoint.setAttachCnt(data.attachCnt);
  if (data.lastPacketCnt !== undefined) endpoint.setLastPacketCnt(data.lastPacketCnt);
  if (data.carrierOffset !== undefined) endpoint.setCarrierOffset(data.carrierOffset);
}

// Boolean fields with `!== undefined` check (preserves explicit false values).
function applyEndpointCreateBooleanFields(endpoint: pb.EndPoint, data: CreateEndpointInput): void {
  if (data.preAttach !== undefined) endpoint.setPreAttach(data.preAttach);
  if (data.dualChan !== undefined) endpoint.setDualChan(data.dualChan);
  if (data.repetition !== undefined) endpoint.setRepetition(data.repetition);
  if (data.wideCarrOff !== undefined) endpoint.setWideCarrOff(data.wideCarrOff);
  if (data.longBlkDist !== undefined) endpoint.setLongBlkDist(data.longBlkDist);
}

// Builds a fully-populated EndPoint message from a CreateEndpointInput.
// Field predicates differ by type: strings use truthy checks (empty skipped),
// numerics/booleans use `!== undefined` (preserves zero / false).
function buildEndpointMessage(data: CreateEndpointInput): pb.EndPoint {
  const endpoint = new pb.EndPoint();
  endpoint.setEpeui(data.epEui);
  applyEndpointCreateStringFields(endpoint, data);
  applyEndpointCreateHexFields(endpoint, data);
  applyEndpointCreateNumericFields(endpoint, data);
  applyEndpointCreateBooleanFields(endpoint, data);
  return endpoint;
}

// Module-private input shape for updateEndpoint. typeEui and deviceModelId
// accept `null` for explicit clear; see buildEndpointUpdate for semantics.
// newEpEui is for cascade rename and lives on the request, not the EndPoint
// message — buildEndpointUpdate ignores it.
interface UpdateEndpointInput {
  name?: string;
  description?: string;
  epClass?: string;
  status?: string;
  shAddr?: number;
  nwkSnKey?: string;
  appKey?: string;
  preAttach?: boolean;
  dualChan?: boolean;
  repetition?: boolean;
  wideCarrOff?: boolean;
  longBlkDist?: boolean;
  attachCnt?: number;
  lastPacketCnt?: number;
  carrierOffset?: number;
  typeEui?: string | null;
  deviceModelId?: string | null;
  newEpEui?: string;
}

// Result of building an update EndPoint message: the populated message plus
// the FieldMask paths covering exactly the fields that were touched.
interface EndpointUpdateDraft {
  endpoint: pb.EndPoint;
  paths: string[];
}

// String fields: set + push path on `!== undefined` (preserves empty string writes).
function applyEndpointUpdateStringFields(
  endpoint: pb.EndPoint,
  paths: string[],
  data: UpdateEndpointInput
): void {
  if (data.name !== undefined) {
    endpoint.setName(data.name);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.NAME);
  }
  if (data.description !== undefined) {
    endpoint.setDescription(data.description);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.DESCRIPTION);
  }
  if (data.epClass !== undefined) {
    endpoint.setEpClass(data.epClass);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.EP_CLASS);
  }
  if (data.status !== undefined) {
    endpoint.setStatus(data.status);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.STATUS);
  }
}

// Hex-string fields: set + push path on `!== undefined` with hex→bytes conversion.
function applyEndpointUpdateHexFields(
  endpoint: pb.EndPoint,
  paths: string[],
  data: UpdateEndpointInput
): void {
  if (data.nwkSnKey !== undefined) {
    endpoint.setNwkSnKey(hexToBytes(data.nwkSnKey));
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.NWK_SN_KEY);
  }
  if (data.appKey !== undefined) {
    endpoint.setAppKey(hexToBytes(data.appKey));
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.APP_KEY);
  }
}

// Numeric fields: set + push path on `!== undefined`.
function applyEndpointUpdateNumericFields(
  endpoint: pb.EndPoint,
  paths: string[],
  data: UpdateEndpointInput
): void {
  if (data.shAddr !== undefined) {
    endpoint.setShAddr(data.shAddr);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.SH_ADDR);
  }
  if (data.attachCnt !== undefined) {
    endpoint.setAttachCnt(data.attachCnt);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.ATTACH_CNT);
  }
  if (data.lastPacketCnt !== undefined) {
    endpoint.setLastPacketCnt(data.lastPacketCnt);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.LAST_PACKET_CNT);
  }
  if (data.carrierOffset !== undefined) {
    endpoint.setCarrierOffset(data.carrierOffset);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.CARRIER_OFFSET);
  }
}

// Boolean fields: set + push path on `!== undefined` (preserves explicit false).
function applyEndpointUpdateBooleanFields(
  endpoint: pb.EndPoint,
  paths: string[],
  data: UpdateEndpointInput
): void {
  if (data.preAttach !== undefined) {
    endpoint.setPreAttach(data.preAttach);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.PRE_ATTACH);
  }
  if (data.dualChan !== undefined) {
    endpoint.setDualChan(data.dualChan);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.DUAL_CHAN);
  }
  if (data.repetition !== undefined) {
    endpoint.setRepetition(data.repetition);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.REPETITION);
  }
  if (data.wideCarrOff !== undefined) {
    endpoint.setWideCarrOff(data.wideCarrOff);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.WIDE_CARR_OFF);
  }
  if (data.longBlkDist !== undefined) {
    endpoint.setLongBlkDist(data.longBlkDist);
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.LONG_BLK_DIST);
  }
}

// Clearable fields: typeEui (skip-set on null, still push path) and
// deviceModelId (setDeviceModelId('') on null + push path).
function applyEndpointUpdateClearableFields(
  endpoint: pb.EndPoint,
  paths: string[],
  data: UpdateEndpointInput
): void {
  if (data.typeEui !== undefined) {
    if (data.typeEui) {
      endpoint.setTypeEui(hexToBytes(data.typeEui));
    }
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.TYPE_EUI);
  }
  if (data.deviceModelId !== undefined) {
    endpoint.setDeviceModelId(data.deviceModelId ?? '');
    paths.push(ENDPOINT_UPDATE_MASK_PATHS.DEVICE_MODEL_ID);
  }
}

// Builds an EndPoint message + field-mask paths for an update RPC.
// All predicates are `!== undefined` so partial updates don't reset
// unspecified booleans/numbers. Clearable fields (typeEui, deviceModelId)
// have special null semantics — see applyEndpointUpdateClearableFields.
function buildEndpointUpdate(epEui: string, data: UpdateEndpointInput): EndpointUpdateDraft {
  const endpoint = new pb.EndPoint();
  endpoint.setEpeui(epEui);
  const paths: string[] = [];
  applyEndpointUpdateStringFields(endpoint, paths, data);
  applyEndpointUpdateHexFields(endpoint, paths, data);
  applyEndpointUpdateNumericFields(endpoint, paths, data);
  applyEndpointUpdateBooleanFields(endpoint, paths, data);
  applyEndpointUpdateClearableFields(endpoint, paths, data);
  return { endpoint, paths };
}

// Module-private shape returned by getBaseStation. Mirrored by the
// mapBaseStationProto helper below so the method body stays compact.
interface GrpcBaseStationDetail {
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
}

function mapBaseStationProto(response: pb.BaseStation): GrpcBaseStationDetail {
  return {
    bsEui: response.getBseui(),
    tenantId: response.getTenantId(),
    name: response.getName(),
    description: response.getDescription() || undefined,
    latitude: response.getLatitude()?.getValue() ?? undefined,
    longitude: response.getLongitude()?.getValue() ?? undefined,
    altitude: response.getAltitude()?.getValue() ?? undefined,
    status: response.getStatus(),
    tags: response
      .getTagsMap()
      .toObject()
      .reduce(
        (acc, [k, v]) => {
          acc[k] = v;
          return acc;
        },
        {} as Record<string, string>
      ),
    createdAt: response.getCreatedAt()?.toDate(),
    updatedAt: response.getUpdatedAt()?.toDate(),
    lastSeenAt: response.getLastSeenAt()?.toDate(),
    systemTime: response.getSystemTime()?.getValue() ?? undefined,
    dutyCycle: response.getDutyCycle()?.getValue() ?? undefined,
    uptimeSeconds: response.getUptimeSeconds()?.getValue() ?? undefined,
    temperatureCelsius: response.getTemperatureCelsius()?.getValue() ?? undefined,
    cpuLoad: response.getCpuLoad()?.getValue() ?? undefined,
    memoryLoad: response.getMemoryLoad()?.getValue() ?? undefined,
    bsConfig: response.getBsConfig()?.toJavaScript() ?? undefined,
    lastStatusAt: response.getLastStatusAt()?.toDate() ?? undefined,
    serviceCenterUrl: response.getServiceCenterUrl() || undefined,
    locationSource: response.getLocationSource() || undefined,
    locationUpdatedAt: response.getLocationUpdatedAt()?.toDate() ?? undefined,
  };
}

/**
 * gRPC-web Client Service
 *
 * Wraps the generated KiloCenterServiceClient with Promise-based APIs,
 * metadata injection, and error handling.
 */
class GrpcClientService {
  private client: InstanceType<typeof KiloCenterServiceClient>;
  private organizationId: string | null = null;
  private userId: string | null = null;
  private authFailureCallback?: AuthFailureCallback;
  private refreshPromise: Promise<boolean> | null = null;

  constructor() {
    // Default to gRPC-web endpoint on KC-Gateway port 9090
    // In development, Vite proxy handles routing
    const host = this.getGrpcHost();
    this.client = new KiloCenterServiceClient(host);
  }

  /**
   * Get gRPC host URL from environment config
   * Uses @config/env (no direct import.meta.env access)
   */
  private getGrpcHost(): string {
    // grpcUrl from @config/env (empty string uses Vite proxy in development)
    return grpcUrl;
  }

  /**
   * Clear organization context (logout/session reset)
   */
  clearOrganization(): void {
    this.organizationId = null;
    this.userId = null;
  }

  /**
   * Set organization context for API requests
   */
  setOrganization(orgId: string | null | undefined, userId?: string | null | undefined): void {
    if (!orgId) {
      this.clearOrganization();
      return;
    }
    this.organizationId = orgId;
    this.userId = userId || null;
  }

  /**
   * Set auth failure callback for 401 response handling
   */
  setAuthFailureCallback(callback: AuthFailureCallback | undefined): void {
    this.authFailureCallback = callback;
  }

  // =========================================================================
  // Public gRPC Methods - Skip org/user requirement
  // Canonical source: KC-Core/pkg/grpc/public_methods.go (PublicMethods, OrgExemptMethods)
  // =========================================================================

  /**
   * Build gRPC metadata with auth token and organization ID
   * Uses HEADERS constants for consistency with api.ts
   *
   * @param options.requireOrgUser - If true (default), throws if org/user not set.
   *                                  Set to false for public methods.
   */
  private buildMetadata(
    options: { requireOrgUser: boolean } = { requireOrgUser: true }
  ): grpc.Metadata {
    const metadata = new grpc.Metadata();

    // Add auth token using centralized header constants (always, if present)
    const token = storageService.getItem(STORAGE_KEYS.AUTH_TOKEN);
    if (token) {
      metadata.set(
        HEADERS.AUTHORIZATION.toLowerCase(),
        `${HEADER_VALUES.AUTH_BEARER_PREFIX}${token}`
      );
    }

    // For authenticated calls, require org/user context (fail closed)
    if (options.requireOrgUser) {
      if (!this.organizationId) {
        throw new GrpcApiError(401, ERR_AUTH_ORG_REQUIRED);
      }
      metadata.set(HEADERS.X_ORGANIZATION_ID.toLowerCase(), this.organizationId);

      if (!this.userId) {
        throw new GrpcApiError(401, ERR_UNAUTHORIZED);
      }
      metadata.set(HEADERS.X_USER_ID.toLowerCase(), this.userId);
    } else {
      // Public methods: add org/user only if available (no throw)
      if (this.organizationId) {
        metadata.set(HEADERS.X_ORGANIZATION_ID.toLowerCase(), this.organizationId);
      }
      if (this.userId) {
        metadata.set(HEADERS.X_USER_ID.toLowerCase(), this.userId);
      }
    }

    return metadata;
  }

  /**
   * Attempt to refresh the access token using the stored refresh token.
   * Deduplicates concurrent refresh attempts so only one RPC is in flight.
   * Returns true if the token was refreshed successfully.
   */
  private attemptTokenRefresh(): Promise<boolean> {
    if (this.refreshPromise) return this.refreshPromise;

    this.refreshPromise = (async () => {
      const refreshToken = storageService.getItem(STORAGE_KEYS.REFRESH_TOKEN);
      if (!refreshToken) return false;

      try {
        const result = await this.refreshTokens(refreshToken);
        storageService.setItem(STORAGE_KEYS.AUTH_TOKEN, result.accessToken);
        if (result.refreshToken) {
          storageService.setItem(STORAGE_KEYS.REFRESH_TOKEN, result.refreshToken);
        }
        return true;
      } catch {
        return false;
      } finally {
        this.refreshPromise = null;
      }
    })();

    return this.refreshPromise;
  }

  /**
   * Convert ServiceError to GrpcApiError
   */
  private handleError(error: ServiceError): GrpcApiError {
    const httpStatus = GRPC_TO_HTTP_STATUS[error.code] || 500;

    // Extract error catalog token from message if present (supplementary signal)
    const tokenMatch = error.message?.match(/KC-GRPC-ERR-\w+/);
    const token = tokenMatch ? tokenMatch[0] : undefined;

    return new GrpcApiError(httpStatus, error.message, undefined, undefined, token, error.code);
  }

  /**
   * Execute a single gRPC call returning a Promise.
   */
  private executeCall<TRequest, TResponse>(
    method: (
      request: TRequest,
      metadata: grpc.Metadata,
      callback: (error: ServiceError | null, response: TResponse | null) => void
    ) => void,
    request: TRequest,
    options: { requireOrgUser: boolean }
  ): Promise<TResponse> {
    return new Promise((resolve, reject) => {
      const metadata = this.buildMetadata(options);
      method.call(this.client, request, metadata, (error, response) => {
        if (error) {
          reject(this.handleError(error));
        } else if (response) {
          resolve(response);
        } else {
          reject(new GrpcApiError(500, GRPC_CLIENT_ERRORS.EMPTY_RESPONSE));
        }
      });
    });
  }

  /**
   * Wrap callback-based gRPC call in a Promise with automatic token refresh on 401.
   *
   * @param method - The gRPC method to wrap
   * @param options.requireOrgUser - If true (default), throws if org/user not set
   * @param options.skipRefreshRetry - If true, skip the refresh-and-retry on 401
   */
  private promisify<TRequest, TResponse>(
    method: (
      request: TRequest,
      metadata: grpc.Metadata,
      callback: (error: ServiceError | null, response: TResponse | null) => void
    ) => void,
    options: { requireOrgUser: boolean; skipRefreshRetry?: boolean } = { requireOrgUser: true }
  ): (request: TRequest) => Promise<TResponse> {
    return async (request: TRequest): Promise<TResponse> => {
      try {
        return await this.executeCall(method, request, options);
      } catch (err) {
        if (
          !options.skipRefreshRetry &&
          err instanceof GrpcApiError &&
          err.isUnauthorized()
        ) {
          const refreshed = await this.attemptTokenRefresh();
          if (refreshed) {
            return this.executeCall(method, request, options);
          }
          // Refresh failed — clear session and redirect to login
          this.authFailureCallback?.();
          if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
            window.location.href = '/login';
          }
          // Return a never-resolving promise to suppress error toasts
          // while the redirect is in progress
          return new Promise<TResponse>(() => {});
        }
        throw err;
      }
    };
  }

  // ============================================================================
  // AUTH METHODS
  // ============================================================================

  /**
   * Login with email and password
   */
  async login(
    email: string,
    password: string
  ): Promise<{
    tokens: {
      accessToken: string;
      refreshToken: string;
      accessExpiresIn: number;
      refreshExpiresIn: number;
    };
    user: {
      id: string;
      email: string;
      isAdmin: boolean;
      hasPassword: boolean;
      memberships: Array<{
        orgId: string;
        orgName: string;
        role: string;
        status: string;
        isOrgAdmin: boolean;
        isBaseStationAdmin: boolean;
        isEndpointAdmin: boolean;
      }>;
      defaultOrgId?: string;
      firstName?: string;
      lastName?: string;
    };
  }> {
    const request = new pb.LoginRequest();
    request.setEmail(email);
    request.setPassword(password);

    // Public method: skip org/user requirement (pre-auth).
    // skipRefreshRetry prevents the 401-refresh handler from swallowing
    // invalid-credential errors into a never-resolving redirect promise.
    const response = await this.promisify<pb.LoginRequest, pb.LoginResponse>(this.client.login, {
      requireOrgUser: false,
      skipRefreshRetry: true,
    })(request);

    const tokens = response.getTokens();
    const user = response.getUser();

    if (!tokens || !user) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_LOGIN_RESPONSE);
    }

    return {
      tokens: {
        accessToken: tokens.getAccessToken(),
        refreshToken: tokens.getRefreshToken(),
        accessExpiresIn: tokens.getAccessExpiresIn(),
        refreshExpiresIn: tokens.getRefreshExpiresIn(),
      },
      user: {
        id: user.getId(),
        email: user.getEmail(),
        isAdmin: user.getIsAdmin(),
        hasPassword: user.getHasPassword(),
        memberships: user.getMembershipsList().map((m) => ({
          orgId: m.getOrgId(),
          orgName: m.getOrgName(),
          role: m.getRole(),
          status: m.getStatus(),
          isOrgAdmin: m.getIsOrgAdmin(),
          isBaseStationAdmin: m.getIsBaseStationAdmin(),
          isEndpointAdmin: m.getIsEndpointAdmin(),
        })),
        defaultOrgId: user.getDefaultOrgId() || undefined,
        firstName: user.getFirstName() || undefined,
        lastName: user.getLastName() || undefined,
      },
    };
  }

  /**
   * Register a new account (public endpoint)
   */
  async registerAccount(
    email: string,
    password: string,
    firstName: string,
    lastName: string,
    companyName: string
  ): Promise<{
    tokens: {
      accessToken: string;
      refreshToken: string;
      accessExpiresIn: number;
      refreshExpiresIn: number;
    };
    user: {
      id: string;
      email: string;
      isAdmin: boolean;
      hasPassword: boolean;
      memberships: Array<{
        orgId: string;
        orgName: string;
        role: string;
        status: string;
        isOrgAdmin: boolean;
        isBaseStationAdmin: boolean;
        isEndpointAdmin: boolean;
      }>;
      defaultOrgId?: string;
      firstName?: string;
      lastName?: string;
    };
  }> {
    const request = new pb.RegisterAccountRequest();
    request.setEmail(email);
    request.setPassword(password);
    request.setFirstName(firstName);
    request.setLastName(lastName);
    request.setCompanyName(companyName);

    const response = await this.promisify<pb.RegisterAccountRequest, pb.LoginResponse>(
      this.client.registerAccount,
      { requireOrgUser: false }
    )(request);

    const tokens = response.getTokens();
    const user = response.getUser();

    if (!tokens || !user) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_LOGIN_RESPONSE);
    }

    return {
      tokens: {
        accessToken: tokens.getAccessToken(),
        refreshToken: tokens.getRefreshToken(),
        accessExpiresIn: tokens.getAccessExpiresIn(),
        refreshExpiresIn: tokens.getRefreshExpiresIn(),
      },
      user: {
        id: user.getId(),
        email: user.getEmail(),
        isAdmin: user.getIsAdmin(),
        hasPassword: user.getHasPassword(),
        memberships: user.getMembershipsList().map((m) => ({
          orgId: m.getOrgId(),
          orgName: m.getOrgName(),
          role: m.getRole(),
          status: m.getStatus(),
          isOrgAdmin: m.getIsOrgAdmin(),
          isBaseStationAdmin: m.getIsBaseStationAdmin(),
          isEndpointAdmin: m.getIsEndpointAdmin(),
        })),
        defaultOrgId: user.getDefaultOrgId() || undefined,
        firstName: user.getFirstName() || undefined,
        lastName: user.getLastName() || undefined,
      },
    };
  }

  /**
   * Get authentication settings (public endpoint)
   */
  async getAuthSettings(): Promise<{
    enabled: boolean;
    localLoginEnabled: boolean;
    loginUrl?: string;
    loginLabel?: string;
    loginRedirect: boolean;
    logoutUrl?: string;
    refreshTokenEnabled: boolean;
    registrationEnabled: boolean;
    oidc?: {
      enabled: boolean;
      loginUrl?: string;
      loginLabel?: string;
      loginRedirect: boolean;
      logoutUrl?: string;
    };
    oauth2?: {
      enabled: boolean;
      loginUrl?: string;
      loginLabel?: string;
      loginRedirect: boolean;
      logoutUrl?: string;
    };
  }> {
    const request = new pb.GetAuthSettingsRequest();
    // Public method: skip org/user requirement (pre-auth)
    const response = await this.promisify<pb.GetAuthSettingsRequest, pb.GetAuthSettingsResponse>(
      this.client.getAuthSettings,
      { requireOrgUser: false }
    )(request);

    const settings = response.getSettings();
    if (!settings) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_AUTH_SETTINGS_RESPONSE);
    }

    const oidc = settings.getOidc();
    const oauth2 = settings.getOauth2();

    return {
      enabled: settings.getEnabled(),
      localLoginEnabled: settings.getLocalLoginEnabled(),
      loginUrl: settings.getLoginUrl() || undefined,
      loginLabel: settings.getLoginLabel() || undefined,
      loginRedirect: settings.getLoginRedirect(),
      logoutUrl: settings.getLogoutUrl() || undefined,
      refreshTokenEnabled: settings.getRefreshTokenEnabled(),
      registrationEnabled: settings.getRegistrationEnabled(),
      oidc: oidc
        ? {
            enabled: oidc.getEnabled(),
            loginUrl: oidc.getLoginUrl() || undefined,
            loginLabel: oidc.getLoginLabel() || undefined,
            loginRedirect: oidc.getLoginRedirect(),
            logoutUrl: oidc.getLogoutUrl() || undefined,
          }
        : undefined,
      oauth2: oauth2
        ? {
            enabled: oauth2.getEnabled(),
            loginUrl: oauth2.getLoginUrl() || undefined,
            loginLabel: oauth2.getLoginLabel() || undefined,
            loginRedirect: oauth2.getLoginRedirect(),
            logoutUrl: oauth2.getLogoutUrl() || undefined,
          }
        : undefined,
    };
  }

  /**
   * Logout and revoke refresh token
   */
  async logout(refreshToken: string): Promise<void> {
    const request = new pb.LogoutRequest();
    request.setRefreshToken(refreshToken);

    await this.promisify<pb.LogoutRequest, pb.LogoutResponse>(this.client.logout)(request);
  }

  // ============================================================================
  // BASE STATION METHODS
  // ============================================================================

  /**
   * List all base stations
   */
  async listBaseStations(): Promise<
    Array<{
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
      locationSource?: string;
      locationUpdatedAt?: Date;
    }>
  > {
    const request = new pb.ListBaseStationsRequest();

    const response = await this.promisify<pb.ListBaseStationsRequest, pb.ListBaseStationsResponse>(
      this.client.listBaseStations
    )(request);

    return response.getBasestationsList().map((bs) => ({
      bsEui: bs.getBseui(),
      tenantId: bs.getTenantId(),
      name: bs.getName(),
      description: bs.getDescription() || undefined,
      latitude: bs.getLatitude()?.getValue() ?? undefined,
      longitude: bs.getLongitude()?.getValue() ?? undefined,
      altitude: bs.getAltitude()?.getValue() ?? undefined,
      status: bs.getStatus(),
      tags: bs
        .getTagsMap()
        .toObject()
        .reduce(
          (acc, [k, v]) => {
            acc[k] = v;
            return acc;
          },
          {} as Record<string, string>
        ),
      createdAt: bs.getCreatedAt()?.toDate(),
      updatedAt: bs.getUpdatedAt()?.toDate(),
      lastSeenAt: bs.getLastSeenAt()?.toDate(),
      locationSource: bs.getLocationSource() || undefined,
      locationUpdatedAt: bs.getLocationUpdatedAt()?.toDate(),
    }));
  }

  /**
   * Get base station by EUI
   */
  async getBaseStation(bsEui: string): Promise<GrpcBaseStationDetail | null> {
    const request = new pb.GetBaseStationRequest();
    request.setBseui(bsEui);

    try {
      const response = await this.promisify<pb.GetBaseStationRequest, pb.BaseStation>(
        this.client.getBaseStation
      )(request);
      return mapBaseStationProto(response);
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Delete base station
   */
  async deleteBaseStation(bsEui: string): Promise<void> {
    const request = new pb.DeleteBaseStationRequest();
    request.setBseui(bsEui);

    await this.promisify<pb.DeleteBaseStationRequest, Empty>(this.client.deleteBaseStation)(
      request
    );
  }

  // ============================================================================
  // ENDPOINT METHODS
  // ============================================================================

  /**
   * List all endpoints
   */
  async listEndpoints(): Promise<GrpcEndpointResult[]> {
    const request = new pb.ListEndPointsRequest();

    const response = await this.promisify<pb.ListEndPointsRequest, pb.ListEndPointsResponse>(
      this.client.listEndPoints
    )(request);

    return response.getEndpointsList().map(mapEndpointProto);
  }

  /**
   * Get endpoint by EUI
   */
  async getEndpoint(epEui: string): Promise<GrpcEndpointResult | null> {
    const request = new pb.GetEndPointRequest();
    request.setEpeui(epEui);

    try {
      const response = await this.promisify<pb.GetEndPointRequest, pb.EndPoint>(
        this.client.getEndPoint
      )(request);

      return mapEndpointProto(response);
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Attach endpoint to network
   */
  async attachEndpoint(epEui: string): Promise<{ operationId: string; status: string }> {
    const request = new pb.AttachEndPointRequest();
    request.setEpEui(epEui);

    const response = await this.promisify<pb.AttachEndPointRequest, pb.AttachEndPointResponse>(
      this.client.attachEndPoint
    )(request);

    return {
      operationId: response.getOperationId(),
      status: response.getStatus(),
    };
  }

  /**
   * Detach endpoint from network
   */
  async detachEndpoint(epEui: string): Promise<{ operationId: string; status: string }> {
    const request = new pb.DetachEndPointRequest();
    request.setEpEui(epEui);

    const response = await this.promisify<pb.DetachEndPointRequest, pb.DetachEndPointResponse>(
      this.client.detachEndPoint
    )(request);

    return {
      operationId: response.getOperationId(),
      status: response.getStatus(),
    };
  }

  /**
   * Delete endpoint
   */
  async deleteEndpoint(epEui: string): Promise<void> {
    const request = new pb.DeleteEndPointRequest();
    request.setEpeui(epEui);

    await this.promisify<pb.DeleteEndPointRequest, Empty>(this.client.deleteEndPoint)(request);
  }

  // ============================================================================
  // EVENTS METHODS
  // ============================================================================

  /**
   * List system events
   */
  async listEvents(params?: {
    pageSize?: number;
    pageToken?: string;
    categories?: readonly string[];
    eventTypes?: readonly string[];
    severity?: string;
    startTime?: Date;
    endTime?: Date;
  }): Promise<{
    events: Array<{
      id: string;
      eventType: string;
      category: string;
      severity: string;
      title: string;
      description: string;
      sourceName: string;
      timestamp: Date;
      data?: unknown;
    }>;
    nextPageToken?: string;
    totalCount: number;
  }> {
    const request = new pb.ListEventsRequest();
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.pageToken) request.setPageToken(params.pageToken);
    if (params?.categories?.length) {
      request.setCategoriesList([...params.categories]);
    }
    if (params?.severity) request.setSeverity(params.severity);
    if (params?.eventTypes?.length) {
      request.setEventTypesList([...params.eventTypes]);
    }

    const response = await this.promisify<pb.ListEventsRequest, pb.ListEventsResponse>(
      this.client.listEvents
    )(request);

    return {
      events: response.getEventsList().map((e) => ({
        id: e.getId(),
        eventType: e.getEventType(),
        category: e.getCategory(),
        severity: e.getSeverity(),
        title: e.getTitle(),
        description: e.getDescription(),
        sourceName: e.getSourceName(),
        timestamp: e.getTimestamp()?.toDate() || new Date(),
        data:
          e.getData_asU8().length > 0
            ? JSON.parse(new TextDecoder().decode(e.getData_asU8()))
            : undefined,
      })),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  // ============================================================================
  // SYSTEM METHODS
  // ============================================================================

  /**
   * Get system status with service health information
   */
  async getSystemStatus(): Promise<{
    version: string;
    status: string;
    uptime?: Date;
    activeEndpoints: number;
    activeBasestations: number;
    messagesProcessed: number;
    services: Array<{
      name: string;
      url: string;
      healthy: boolean;
      latencyMs: number;
      error: string;
      checkedAt: Date;
    }>;
  }> {
    const request = new Empty();

    const response = await this.promisify<Empty, pb.SystemStatus>(this.client.getSystemStatus, {
      requireOrgUser: false,
    })(request);

    // Fallback timestamp only for legacy servers that omit checked_at
    const fallbackTime = response.getUptime()?.toDate() ?? new Date();

    return {
      version: response.getVersion(),
      status: response.getStatus(),
      uptime: response.getUptime()?.toDate(),
      activeEndpoints: response.getActiveEndpoints(),
      activeBasestations: response.getActiveBasestations(),
      messagesProcessed: response.getMessagesProcessed(),
      services: response.getServicesList().map((s) => ({
        name: s.getName(),
        url: s.getUrl(),
        healthy: s.getHealthy(),
        latencyMs: s.getLatencyMs(),
        error: s.getError(),
        // Use proto checked_at if present; fallback only for legacy servers
        checkedAt: s.getCheckedAt()?.toDate() ?? fallbackTime,
      })),
    };
  }

  /**
   * Get release info / version
   */
  async getReleaseInfo(): Promise<{
    version: string;
    buildTime: string;
    gitCommit: string;
    gitBranch: string;
    buildUser: string;
    goVersion: string;
    schemaVersion: number;
    artifacts: Record<string, string>;
    scEui?: string;
    scVendor?: string;
    scModel?: string;
    scName?: string;
    scSwVersion?: string;
    edition?: string;
    editionCode?: string;
    licenseId?: string;
    licenseUrl?: string;
    sourceUrl?: string;
    documentationUrl?: string;
    homepageUrl?: string;
    trademarkNotice?: string;
  }> {
    const request = new Empty();

    const response = await this.promisify<Empty, pb.ReleaseInfo>(this.client.getReleaseInfo, {
      requireOrgUser: false,
    })(request);

    return {
      version: response.getVersion(),
      buildTime: response.getBuildTime(),
      gitCommit: response.getGitCommit(),
      gitBranch: response.getGitBranch(),
      buildUser: response.getBuildUser(),
      goVersion: response.getGoVersion(),
      schemaVersion: response.getSchemaVersion(),
      artifacts: response
        .getArtifactsMap()
        .toObject()
        .reduce(
          (acc, [k, v]) => {
            acc[k] = v;
            return acc;
          },
          {} as Record<string, string>
        ),
      scEui: response.getScEui() ? response.getScEui().toString() : undefined,
      scVendor: response.getScVendor() || undefined,
      scModel: response.getScModel() || undefined,
      scName: response.getScName() || undefined,
      scSwVersion: response.getScSwVersion() || undefined,
      edition: response.getEdition() || undefined,
      editionCode: response.getEditionCode() || undefined,
      licenseId: response.getLicenseId() || undefined,
      licenseUrl: response.getLicenseUrl() || undefined,
      sourceUrl: response.getSourceUrl() || undefined,
      documentationUrl: response.getDocumentationUrl() || undefined,
      homepageUrl: response.getHomepageUrl() || undefined,
      trademarkNotice: response.getTrademarkNotice() || undefined,
    };
  }

  // ============================================================================
  // ANALYTICS METHODS
  // ============================================================================

  /**
   * Get analytics overview
   */
  async getAnalyticsOverview(): Promise<{
    totalMessages: number;
    activeEndpoints: number;
    activeBaseStations: number;
    avgRssi: number;
    avgSnr: number;
    firstMessage?: Date;
    lastMessage?: Date;
  }> {
    const request = new pb.GetAnalyticsOverviewRequest();

    const response = await this.promisify<
      pb.GetAnalyticsOverviewRequest,
      pb.GetAnalyticsOverviewResponse
    >(this.client.getAnalyticsOverview)(request);

    const overview = response.getOverview();
    if (!overview) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_ANALYTICS_RESPONSE);
    }

    return {
      totalMessages: overview.getTotalMessages(),
      activeEndpoints: overview.getActiveEndpoints(),
      activeBaseStations: overview.getActiveBaseStations(),
      avgRssi: overview.getAvgRssi(),
      avgSnr: overview.getAvgSnr(),
      firstMessage: overview.getFirstMessage()?.toDate(),
      lastMessage: overview.getLastMessage()?.toDate(),
    };
  }

  /**
   * Get alert summary
   */
  async getAlertSummary(): Promise<{
    critical: number;
    warning: number;
    info: number;
    recent: Array<{
      id: string;
      severity: string;
      category: string;
      title: string;
      description: string;
      sourceName: string;
      timestamp: Date;
      status: string;
    }>;
  }> {
    const request = new pb.GetAlertSummaryRequest();

    const response = await this.promisify<pb.GetAlertSummaryRequest, pb.GetAlertSummaryResponse>(
      this.client.getAlertSummary
    )(request);

    const summary = response.getSummary();
    if (!summary) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_ALERT_SUMMARY_RESPONSE);
    }

    return {
      critical: summary.getCritical(),
      warning: summary.getWarning(),
      info: summary.getInfo(),
      recent: summary.getRecentList().map((a) => ({
        id: a.getId(),
        severity: a.getSeverity(),
        category: a.getCategory(),
        title: a.getTitle(),
        description: a.getDescription(),
        sourceName: a.getSourceName(),
        timestamp: a.getTimestamp()?.toDate() || new Date(),
        status: a.getStatus(),
      })),
    };
  }

  // ============================================================================
  // AUTH METHODS (Extended)
  // ============================================================================

  /**
   * Get authenticated user profile
   */
  async getProfile(): Promise<{
    id: string;
    email: string;
    isAdmin: boolean;
    hasPassword: boolean;
    memberships: Array<{
      orgId: string;
      orgName: string;
      role: string;
      status: string;
      isOrgAdmin: boolean;
      isBaseStationAdmin: boolean;
      isEndpointAdmin: boolean;
    }>;
    defaultOrgId?: string;
    externalId?: string;
    externalIdp?: string;
    firstName?: string;
    lastName?: string;
  }> {
    const request = new pb.GetProfileRequest();
    const response = await this.promisify<pb.GetProfileRequest, pb.GetProfileResponse>(
      this.client.getProfile
    )(request);

    const user = response.getUser();
    if (!user) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_PROFILE_RESPONSE);
    }

    return {
      id: user.getId(),
      email: user.getEmail(),
      isAdmin: user.getIsAdmin(),
      hasPassword: user.getHasPassword(),
      memberships: user.getMembershipsList().map((m) => ({
        orgId: m.getOrgId(),
        orgName: m.getOrgName(),
        role: m.getRole(),
        status: m.getStatus(),
        isOrgAdmin: m.getIsOrgAdmin(),
        isBaseStationAdmin: m.getIsBaseStationAdmin(),
        isEndpointAdmin: m.getIsEndpointAdmin(),
      })),
      defaultOrgId: user.getDefaultOrgId() || undefined,
      firstName: user.getFirstName() || undefined,
      lastName: user.getLastName() || undefined,
    };
  }

  /**
   * Refresh tokens
   */
  async refreshTokens(refreshToken: string): Promise<{
    accessToken: string;
    refreshToken: string;
    accessExpiresIn: number;
    refreshExpiresIn: number;
  }> {
    const request = new pb.RefreshTokensRequest();
    request.setRefreshToken(refreshToken);

    const response = await this.promisify<pb.RefreshTokensRequest, pb.RefreshTokensResponse>(
      this.client.refreshTokens,
      { requireOrgUser: false, skipRefreshRetry: true }
    )(request);

    const tokens = response.getTokens();
    if (!tokens) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_REFRESH_TOKENS_RESPONSE);
    }

    return {
      accessToken: tokens.getAccessToken(),
      refreshToken: tokens.getRefreshToken(),
      accessExpiresIn: tokens.getAccessExpiresIn(),
      refreshExpiresIn: tokens.getRefreshExpiresIn(),
    };
  }

  /**
   * Change own password
   */
  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    const request = new pb.ChangePasswordRequest();
    request.setCurrentPassword(currentPassword);
    request.setNewPassword(newPassword);

    await this.promisify<pb.ChangePasswordRequest, pb.ChangePasswordResponse>(
      this.client.changePassword
    )(request);
  }

  /**
   * Exchange OIDC authorization code for tokens.
   * Backend resolves the redirect URI from server-side config, so the
   * client doesn't send one over the wire.
   */
  async exchangeOIDC(
    code: string,
    state: string
  ): Promise<{
    tokens: {
      accessToken: string;
      refreshToken: string;
      accessExpiresIn: number;
      refreshExpiresIn: number;
    };
    user: {
      id: string;
      email: string;
      isAdmin: boolean;
      hasPassword: boolean;
      memberships: Array<{
        orgId: string;
        orgName: string;
        role: string;
        status: string;
        isOrgAdmin: boolean;
        isBaseStationAdmin: boolean;
        isEndpointAdmin: boolean;
      }>;
      defaultOrgId?: string;
      firstName?: string;
      lastName?: string;
    };
  }> {
    const request = new pb.ExchangeOIDCRequest();
    request.setCode(code);
    request.setState(state);

    const response = await this.promisify<pb.ExchangeOIDCRequest, pb.LoginResponse>(
      this.client.exchangeOIDC,
      { requireOrgUser: false, skipRefreshRetry: true }
    )(request);

    const tokens = response.getTokens();
    const user = response.getUser();

    if (!tokens || !user) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_OIDC_EXCHANGE_RESPONSE);
    }

    return {
      tokens: {
        accessToken: tokens.getAccessToken(),
        refreshToken: tokens.getRefreshToken(),
        accessExpiresIn: tokens.getAccessExpiresIn(),
        refreshExpiresIn: tokens.getRefreshExpiresIn(),
      },
      user: {
        id: user.getId(),
        email: user.getEmail(),
        isAdmin: user.getIsAdmin(),
        hasPassword: user.getHasPassword(),
        memberships: user.getMembershipsList().map((m) => ({
          orgId: m.getOrgId(),
          orgName: m.getOrgName(),
          role: m.getRole(),
          status: m.getStatus(),
          isOrgAdmin: m.getIsOrgAdmin(),
          isBaseStationAdmin: m.getIsBaseStationAdmin(),
          isEndpointAdmin: m.getIsEndpointAdmin(),
        })),
        defaultOrgId: user.getDefaultOrgId() || undefined,
        firstName: user.getFirstName() || undefined,
        lastName: user.getLastName() || undefined,
      },
    };
  }

  /**
   * Exchange OAuth2 authorization code for tokens.
   * Backend resolves the redirect URI from server-side config, so the
   * client doesn't send one over the wire.
   */
  async exchangeOAuth2(
    code: string,
    state: string
  ): Promise<{
    tokens: {
      accessToken: string;
      refreshToken: string;
      accessExpiresIn: number;
      refreshExpiresIn: number;
    };
    user: {
      id: string;
      email: string;
      isAdmin: boolean;
      hasPassword: boolean;
      memberships: Array<{
        orgId: string;
        orgName: string;
        role: string;
        status: string;
        isOrgAdmin: boolean;
        isBaseStationAdmin: boolean;
        isEndpointAdmin: boolean;
      }>;
      defaultOrgId?: string;
      firstName?: string;
      lastName?: string;
    };
  }> {
    const request = new pb.ExchangeOAuth2Request();
    request.setCode(code);
    request.setState(state);

    const response = await this.promisify<pb.ExchangeOAuth2Request, pb.LoginResponse>(
      this.client.exchangeOAuth2,
      { requireOrgUser: false, skipRefreshRetry: true }
    )(request);

    const tokens = response.getTokens();
    const user = response.getUser();

    if (!tokens || !user) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_OAUTH2_EXCHANGE_RESPONSE);
    }

    return {
      tokens: {
        accessToken: tokens.getAccessToken(),
        refreshToken: tokens.getRefreshToken(),
        accessExpiresIn: tokens.getAccessExpiresIn(),
        refreshExpiresIn: tokens.getRefreshExpiresIn(),
      },
      user: {
        id: user.getId(),
        email: user.getEmail(),
        isAdmin: user.getIsAdmin(),
        hasPassword: user.getHasPassword(),
        memberships: user.getMembershipsList().map((m) => ({
          orgId: m.getOrgId(),
          orgName: m.getOrgName(),
          role: m.getRole(),
          status: m.getStatus(),
          isOrgAdmin: m.getIsOrgAdmin(),
          isBaseStationAdmin: m.getIsBaseStationAdmin(),
          isEndpointAdmin: m.getIsEndpointAdmin(),
        })),
        defaultOrgId: user.getDefaultOrgId() || undefined,
        firstName: user.getFirstName() || undefined,
        lastName: user.getLastName() || undefined,
      },
    };
  }

  // ============================================================================
  // USER ADMIN METHODS
  // ============================================================================

  /**
   * List users
   */
  async listUsers(params?: { pageSize?: number; pageToken?: string }): Promise<{
    users: Array<{
      id: string;
      email: string;
      isAdmin: boolean;
      isActive: boolean;
      isTenantManager: boolean;
      isBaseStationManager: boolean;
      isEndpointManager: boolean;
      note?: string;
      createdAt?: Date;
      updatedAt?: Date;
    }>;
    nextPageToken?: string;
    totalCount: number;
  }> {
    const request = new pb.ListUsersRequest();
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.pageToken) request.setPageToken(params.pageToken);

    const response = await this.promisify<pb.ListUsersRequest, pb.ListUsersResponse>(
      this.client.listUsers,
      { requireOrgUser: false }
    )(request);

    return {
      users: response.getUsersList().map((u) => ({
        id: u.getId(),
        email: u.getEmail(),
        isAdmin: u.getIsAdmin(),
        isActive: u.getIsActive(),
        isTenantManager: u.getIsTenantManager(),
        isBaseStationManager: u.getIsBaseStationManager(),
        isEndpointManager: u.getIsEndpointManager(),
        note: u.getNote() || undefined,
        createdAt: u.getCreatedAt()?.toDate(),
        updatedAt: u.getUpdatedAt()?.toDate(),
      })),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  /**
   * Get user by ID
   */
  async getUser(userId: string): Promise<{
    id: string;
    email: string;
    isAdmin: boolean;
    isActive: boolean;
    isTenantManager: boolean;
    isBaseStationManager: boolean;
    isEndpointManager: boolean;
    note?: string;
    createdAt?: Date;
    updatedAt?: Date;
  } | null> {
    const request = new pb.GetUserRequest();
    request.setId(userId);

    try {
      const response = await this.promisify<pb.GetUserRequest, pb.GetUserResponse>(
        this.client.getUser,
        { requireOrgUser: false }
      )(request);

      const user = response.getUser();
      if (!user) return null;

      return {
        id: user.getId(),
        email: user.getEmail(),
        isAdmin: user.getIsAdmin(),
        isActive: user.getIsActive(),
        isTenantManager: user.getIsTenantManager(),
        isBaseStationManager: user.getIsBaseStationManager(),
        isEndpointManager: user.getIsEndpointManager(),
        note: user.getNote() || undefined,
        createdAt: user.getCreatedAt()?.toDate(),
        updatedAt: user.getUpdatedAt()?.toDate(),
      };
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Create user
   */
  async createUser(data: {
    email: string;
    password?: string;
    isAdmin?: boolean;
    isActive?: boolean;
    isTenantManager?: boolean;
    isBaseStationManager?: boolean;
    isEndpointManager?: boolean;
    note?: string;
  }): Promise<{
    id: string;
    email: string;
    isAdmin: boolean;
    isActive: boolean;
    isTenantManager: boolean;
    isBaseStationManager: boolean;
    isEndpointManager: boolean;
    note?: string;
    createdAt?: Date;
    updatedAt?: Date;
  }> {
    const request = new pb.CreateUserRequest();
    request.setEmail(data.email);
    if (data.password) request.setPassword(data.password);
    if (data.isAdmin !== undefined) request.setIsAdmin(data.isAdmin);
    if (data.isActive !== undefined) request.setIsActive(data.isActive);
    if (data.isTenantManager !== undefined) request.setIsTenantManager(data.isTenantManager);
    if (data.isBaseStationManager !== undefined)
      request.setIsBaseStationManager(data.isBaseStationManager);
    if (data.isEndpointManager !== undefined) request.setIsEndpointManager(data.isEndpointManager);
    if (data.note) request.setNote(data.note);

    const response = await this.promisify<pb.CreateUserRequest, pb.CreateUserResponse>(
      this.client.createUser,
      { requireOrgUser: false }
    )(request);

    const user = response.getUser();
    if (!user) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_CREATE_USER_RESPONSE);
    }

    return {
      id: user.getId(),
      email: user.getEmail(),
      isAdmin: user.getIsAdmin(),
      isActive: user.getIsActive(),
      isTenantManager: user.getIsTenantManager(),
      isBaseStationManager: user.getIsBaseStationManager(),
      isEndpointManager: user.getIsEndpointManager(),
      note: user.getNote() || undefined,
      createdAt: user.getCreatedAt()?.toDate(),
      updatedAt: user.getUpdatedAt()?.toDate(),
    };
  }

  /**
   * Update user with FieldMask for partial updates
   */
  async updateUser(
    userId: string,
    data: {
      email?: string;
      isAdmin?: boolean;
      isActive?: boolean;
      isTenantManager?: boolean;
      isBaseStationManager?: boolean;
      isEndpointManager?: boolean;
      note?: string;
    }
  ): Promise<{
    id: string;
    email: string;
    isAdmin: boolean;
    isActive: boolean;
    isTenantManager: boolean;
    isBaseStationManager: boolean;
    isEndpointManager: boolean;
    note?: string;
    createdAt?: Date;
    updatedAt?: Date;
  }> {
    const request = new pb.UpdateUserRequest();
    request.setId(userId);

    const paths: string[] = [];
    if (data.email !== undefined) {
      request.setEmail(data.email);
      paths.push(USER_UPDATE_MASK_PATHS.EMAIL);
    }
    if (data.isAdmin !== undefined) {
      request.setIsAdmin(data.isAdmin);
      paths.push(USER_UPDATE_MASK_PATHS.IS_ADMIN);
    }
    if (data.isActive !== undefined) {
      request.setIsActive(data.isActive);
      paths.push(USER_UPDATE_MASK_PATHS.IS_ACTIVE);
    }
    if (data.isTenantManager !== undefined) {
      request.setIsTenantManager(data.isTenantManager);
      paths.push(USER_UPDATE_MASK_PATHS.IS_TENANT_MANAGER);
    }
    if (data.isBaseStationManager !== undefined) {
      request.setIsBaseStationManager(data.isBaseStationManager);
      paths.push(USER_UPDATE_MASK_PATHS.IS_BASE_STATION_MANAGER);
    }
    if (data.isEndpointManager !== undefined) {
      request.setIsEndpointManager(data.isEndpointManager);
      paths.push(USER_UPDATE_MASK_PATHS.IS_ENDPOINT_MANAGER);
    }
    if (data.note !== undefined) {
      request.setNote(data.note);
      paths.push(USER_UPDATE_MASK_PATHS.NOTE);
    }

    const mask = new FieldMask();
    mask.setPathsList(paths);
    request.setUpdateMask(mask);

    const response = await this.promisify<pb.UpdateUserRequest, pb.UpdateUserResponse>(
      this.client.updateUser,
      { requireOrgUser: false }
    )(request);

    const user = response.getUser();
    if (!user) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_UPDATE_USER_RESPONSE);
    }

    return {
      id: user.getId(),
      email: user.getEmail(),
      isAdmin: user.getIsAdmin(),
      isActive: user.getIsActive(),
      isTenantManager: user.getIsTenantManager(),
      isBaseStationManager: user.getIsBaseStationManager(),
      isEndpointManager: user.getIsEndpointManager(),
      note: user.getNote() || undefined,
      createdAt: user.getCreatedAt()?.toDate(),
      updatedAt: user.getUpdatedAt()?.toDate(),
    };
  }

  /**
   * Delete user
   */
  async deleteUser(userId: string): Promise<void> {
    const request = new pb.DeleteUserRequest();
    request.setId(userId);

    await this.promisify<pb.DeleteUserRequest, pb.DeleteUserResponse>(this.client.deleteUser, {
      requireOrgUser: false,
    })(request);
  }

  /**
   * Update user password (admin)
   */
  async updateUserPassword(userId: string, newPassword: string): Promise<void> {
    const request = new pb.UpdateUserPasswordRequest();
    request.setId(userId);
    request.setNewPassword(newPassword);

    await this.promisify<pb.UpdateUserPasswordRequest, pb.UpdateUserPasswordResponse>(
      this.client.updateUserPassword,
      { requireOrgUser: false }
    )(request);
  }

  // ============================================================================
  // ORGANIZATION METHODS
  // ============================================================================

  /**
   * List organizations
   */
  async listOrganizations(params?: { pageSize?: number; pageToken?: string }): Promise<{
    organizations: Array<{
      id: string;
      name: string;
      description?: string;
      tenantId: string;
      state?: string;
      canHaveBaseStations?: boolean;
      createdAt?: Date;
      updatedAt?: Date;
    }>;
    nextPageToken?: string;
    totalCount: number;
  }> {
    const request = new pb.ListOrganizationsRequest();
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.pageToken) request.setPageToken(params.pageToken);

    const response = await this.promisify<
      pb.ListOrganizationsRequest,
      pb.ListOrganizationsResponse
    >(this.client.listOrganizations, { requireOrgUser: false })(request);

    return {
      organizations: response.getOrganizationsList().map((o) => ({
        id: o.getId(),
        name: o.getName(),
        description: o.getDescription() || undefined,
        tenantId: String(o.getTenantId()),
        state: o.getState(),
        canHaveBaseStations: o.getCanHaveBaseStations(),
        createdAt: o.getCreatedAt()?.toDate(),
        updatedAt: o.getUpdatedAt()?.toDate(),
      })),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  /**
   * Get organization by ID
   */
  async getOrganization(orgId: string): Promise<{
    id: string;
    name: string;
    description?: string;
    tenantId: string;
    state?: string;
    canHaveBaseStations?: boolean;
    createdAt?: Date;
    updatedAt?: Date;
  } | null> {
    const request = new pb.GetOrganizationRequest();
    request.setId(orgId);

    try {
      const response = await this.promisify<pb.GetOrganizationRequest, pb.GetOrganizationResponse>(
        this.client.getOrganization,
        { requireOrgUser: false }
      )(request);

      const org = response.getOrganization();
      if (!org) return null;

      return {
        id: org.getId(),
        name: org.getName(),
        description: org.getDescription() || undefined,
        tenantId: String(org.getTenantId()),
        state: org.getState(),
        canHaveBaseStations: org.getCanHaveBaseStations(),
        createdAt: org.getCreatedAt()?.toDate(),
        updatedAt: org.getUpdatedAt()?.toDate(),
      };
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Create organization
   */
  async createOrganization(data: { name: string; description?: string }): Promise<{
    id: string;
    name: string;
    description?: string;
    tenantId: string;
  }> {
    const request = new pb.CreateOrganizationRequest();
    request.setName(data.name);
    if (data.description) request.setDescription(data.description);

    const response = await this.promisify<
      pb.CreateOrganizationRequest,
      pb.CreateOrganizationResponse
    >(this.client.createOrganization, { requireOrgUser: false })(request);

    const org = response.getOrganization();
    if (!org) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_CREATE_ORGANIZATION_RESPONSE);
    }

    return {
      id: org.getId(),
      name: org.getName(),
      description: org.getDescription() || undefined,
      tenantId: String(org.getTenantId()),
    };
  }

  /**
   * Update organization
   */
  async updateOrganization(
    orgId: string,
    data: {
      name?: string;
      description?: string;
    }
  ): Promise<{
    id: string;
    name: string;
    description?: string;
    tenantId: string;
  }> {
    const request = new pb.UpdateOrganizationRequest();
    request.setId(orgId);
    if (data.name) request.setName(data.name);
    if (data.description !== undefined) request.setDescription(data.description);

    const response = await this.promisify<
      pb.UpdateOrganizationRequest,
      pb.UpdateOrganizationResponse
    >(this.client.updateOrganization, { requireOrgUser: false })(request);

    const org = response.getOrganization();
    if (!org) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_UPDATE_ORGANIZATION_RESPONSE);
    }

    return {
      id: org.getId(),
      name: org.getName(),
      description: org.getDescription() || undefined,
      tenantId: String(org.getTenantId()),
    };
  }

  /**
   * Delete organization
   */
  async deleteOrganization(orgId: string): Promise<void> {
    const request = new pb.DeleteOrganizationRequest();
    request.setId(orgId);

    await this.promisify<pb.DeleteOrganizationRequest, pb.DeleteOrganizationResponse>(
      this.client.deleteOrganization,
      { requireOrgUser: false }
    )(request);
  }

  // ============================================================================
  // ORGANIZATION USER METHODS
  // ============================================================================

  /**
   * List organization users
   */
  async listOrganizationUsers(
    orgId: string,
    params?: {
      pageSize?: number;
      pageToken?: string;
      status?: string;
    }
  ): Promise<{
    users: Array<{
      userId: string;
      orgId: string;
      role: string;
      status: string;
      email: string;
      isOrgAdmin?: boolean;
      isBaseStationAdmin?: boolean;
      isEndpointAdmin?: boolean;
      createdAt?: Date;
      updatedAt?: Date;
    }>;
    nextPageToken?: string;
    totalCount: number;
  }> {
    const request = new pb.ListOrganizationUsersRequest();
    request.setOrgId(orgId);
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.pageToken) request.setPageToken(params.pageToken);
    if (params?.status) request.setStatus(params.status);

    const response = await this.promisify<
      pb.ListOrganizationUsersRequest,
      pb.ListOrganizationUsersResponse
    >(this.client.listOrganizationUsers, { requireOrgUser: false })(request);

    return {
      users: response.getMembersList().map((u) => ({
        userId: u.getUserId(),
        orgId: u.getOrgId(),
        role: u.getRole(),
        status: u.getStatus(),
        email: u.getEmail(),
        isOrgAdmin: u.getIsOrgAdmin(),
        isBaseStationAdmin: u.getIsBaseStationAdmin(),
        isEndpointAdmin: u.getIsEndpointAdmin(),
        createdAt: u.getCreatedAt()?.toDate(),
        updatedAt: u.getUpdatedAt()?.toDate(),
      })),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  /**
   * Get organization user
   */
  async getOrganizationUser(
    orgId: string,
    userId: string
  ): Promise<{
    userId: string;
    orgId: string;
    role: string;
    status: string;
    email: string;
    isOrgAdmin?: boolean;
    isBaseStationAdmin?: boolean;
    isEndpointAdmin?: boolean;
    createdAt?: Date;
    updatedAt?: Date;
  } | null> {
    const request = new pb.GetOrganizationUserRequest();
    request.setOrgId(orgId);
    request.setUserId(userId);

    try {
      const response = await this.promisify<
        pb.GetOrganizationUserRequest,
        pb.GetOrganizationUserResponse
      >(this.client.getOrganizationUser, { requireOrgUser: false })(request);

      const member = response.getMember();
      if (!member) return null;

      return {
        userId: member.getUserId(),
        orgId: member.getOrgId(),
        role: member.getRole(),
        status: member.getStatus(),
        email: member.getEmail(),
        isOrgAdmin: member.getIsOrgAdmin(),
        isBaseStationAdmin: member.getIsBaseStationAdmin(),
        isEndpointAdmin: member.getIsEndpointAdmin(),
        createdAt: member.getCreatedAt()?.toDate(),
        updatedAt: member.getUpdatedAt()?.toDate(),
      };
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Add user to organization
   */
  async addOrganizationUser(
    orgId: string,
    data: {
      userId?: string;
      email?: string;
      role: string;
      isOrgAdmin?: boolean;
      isBaseStationAdmin?: boolean;
      isEndpointAdmin?: boolean;
    }
  ): Promise<{
    userId: string;
    orgId: string;
    email: string;
    role: string;
    status: string;
    isOrgAdmin?: boolean;
    isBaseStationAdmin?: boolean;
    isEndpointAdmin?: boolean;
    createdAt?: Date;
    updatedAt?: Date;
  }> {
    const request = new pb.AddOrganizationUserRequest();
    request.setOrgId(orgId);
    if (data.userId) request.setUserId(data.userId);
    if (data.email) request.setEmail(data.email);
    request.setRole(data.role);
    if (data.isOrgAdmin !== undefined) request.setIsOrgAdmin(data.isOrgAdmin);
    if (data.isBaseStationAdmin !== undefined)
      request.setIsBaseStationAdmin(data.isBaseStationAdmin);
    if (data.isEndpointAdmin !== undefined) request.setIsEndpointAdmin(data.isEndpointAdmin);

    const response = await this.promisify<
      pb.AddOrganizationUserRequest,
      pb.AddOrganizationUserResponse
    >(this.client.addOrganizationUser, { requireOrgUser: false })(request);

    const member = response.getMember();
    if (!member) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_ADD_ORG_USER_RESPONSE);
    }

    return {
      userId: member.getUserId(),
      orgId: member.getOrgId(),
      email: member.getEmail(),
      role: member.getRole(),
      status: member.getStatus(),
      isOrgAdmin: member.getIsOrgAdmin(),
      isBaseStationAdmin: member.getIsBaseStationAdmin(),
      isEndpointAdmin: member.getIsEndpointAdmin(),
      createdAt: member.getCreatedAt()?.toDate(),
      updatedAt: member.getUpdatedAt()?.toDate(),
    };
  }

  /**
   * Update organization user with FieldMask for partial updates
   */
  async updateOrganizationUser(
    orgId: string,
    userId: string,
    data: {
      role?: string;
      isOrgAdmin?: boolean;
      isBaseStationAdmin?: boolean;
      isEndpointAdmin?: boolean;
    }
  ): Promise<{
    userId: string;
    orgId: string;
    email: string;
    role: string;
    status: string;
    isOrgAdmin?: boolean;
    isBaseStationAdmin?: boolean;
    isEndpointAdmin?: boolean;
    createdAt?: Date;
    updatedAt?: Date;
  }> {
    const request = new pb.UpdateOrganizationUserRequest();
    request.setOrgId(orgId);
    request.setUserId(userId);

    const paths: string[] = [];
    if (data.role !== undefined) {
      request.setRole(data.role);
      paths.push(ORG_USER_UPDATE_MASK_PATHS.ROLE);
    }
    if (data.isOrgAdmin !== undefined) {
      request.setIsOrgAdmin(data.isOrgAdmin);
      paths.push(ORG_USER_UPDATE_MASK_PATHS.IS_ORG_ADMIN);
    }
    if (data.isBaseStationAdmin !== undefined) {
      request.setIsBaseStationAdmin(data.isBaseStationAdmin);
      paths.push(ORG_USER_UPDATE_MASK_PATHS.IS_BASE_STATION_ADMIN);
    }
    if (data.isEndpointAdmin !== undefined) {
      request.setIsEndpointAdmin(data.isEndpointAdmin);
      paths.push(ORG_USER_UPDATE_MASK_PATHS.IS_ENDPOINT_ADMIN);
    }

    const mask = new FieldMask();
    mask.setPathsList(paths);
    request.setUpdateMask(mask);

    const response = await this.promisify<
      pb.UpdateOrganizationUserRequest,
      pb.UpdateOrganizationUserResponse
    >(this.client.updateOrganizationUser, { requireOrgUser: false })(request);

    const member = response.getMember();
    if (!member) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_UPDATE_ORG_USER_RESPONSE);
    }

    return {
      userId: member.getUserId(),
      orgId: member.getOrgId(),
      email: member.getEmail(),
      role: member.getRole(),
      status: member.getStatus(),
      isOrgAdmin: member.getIsOrgAdmin(),
      isBaseStationAdmin: member.getIsBaseStationAdmin(),
      isEndpointAdmin: member.getIsEndpointAdmin(),
      createdAt: member.getCreatedAt()?.toDate(),
      updatedAt: member.getUpdatedAt()?.toDate(),
    };
  }

  /**
   * Remove user from organization
   */
  async removeOrganizationUser(orgId: string, userId: string): Promise<void> {
    const request = new pb.RemoveOrganizationUserRequest();
    request.setOrgId(orgId);
    request.setUserId(userId);

    await this.promisify<pb.RemoveOrganizationUserRequest, pb.RemoveOrganizationUserResponse>(
      this.client.removeOrganizationUser,
      { requireOrgUser: false }
    )(request);
  }

  /**
   * List organizations a user belongs to
   */
  async listUserOrganizations(userId: string): Promise<{
    memberships: Array<{ orgId: string; orgName: string; role: string; status: string }>;
  }> {
    const request = new pb.ListUserOrganizationsRequest();
    request.setUserId(userId);
    const response = await this.promisify<
      pb.ListUserOrganizationsRequest,
      pb.ListUserOrganizationsResponse
    >(this.client.listUserOrganizations, { requireOrgUser: false })(request);
    return {
      memberships: response.getMembershipsList().map((m) => ({
        orgId: m.getOrgId(),
        orgName: m.getOrgName(),
        role: m.getRole(),
        status: m.getStatus(),
      })),
    };
  }

  // ============================================================================
  // CERTIFICATE METHODS
  // ============================================================================

  /**
   * Generate certificate for base station
   */
  async generateCertificate(data: {
    bsEui: string;
    baseStationName?: string;
    validityDays?: number;
  }): Promise<{
    bsEui: string;
    serviceCenterUrl: string;
    downloadUrls: Record<string, string>;
    expiresAt: Date;
  }> {
    const request = new pb.GenerateCertificateRequest();
    request.setBsEui(data.bsEui);
    if (data.baseStationName) request.setBaseStationName(data.baseStationName);
    if (data.validityDays) request.setValidityDays(data.validityDays);

    const response = await this.promisify<
      pb.GenerateCertificateRequest,
      pb.GenerateCertificateResponse
    >(this.client.generateCertificate)(request);

    // Convert downloadUrlsMap to plain object
    const downloadUrls: Record<string, string> = {};
    response.getDownloadUrlsMap().forEach((value, key) => {
      downloadUrls[key] = value;
    });

    return {
      bsEui: response.getBsEui(),
      serviceCenterUrl: response.getServiceCenterUrl(),
      downloadUrls,
      expiresAt: response.getExpiresAt()?.toDate() || new Date(),
    };
  }

  /**
   * Get server certificate status
   */
  async getServerCertificateStatus(): Promise<{
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
    const request = new pb.GetServerCertificateStatusRequest();

    const response = await this.promisify<
      pb.GetServerCertificateStatusRequest,
      pb.GetServerCertificateStatusResponse
    >(this.client.getServerCertificateStatus)(request);

    const serverCert = response.getServerCert();
    const caCert = response.getCaCert();

    return {
      serverCert: serverCert
        ? {
            subject: serverCert.getSubject(),
            issuer: serverCert.getIssuer(),
            notBefore: serverCert.getNotBefore()?.toDate() || new Date(),
            notAfter: serverCert.getNotAfter()?.toDate() || new Date(),
            daysUntilExpiry: serverCert.getDaysUntilExpiry(),
            isValid: serverCert.getIsValid(),
          }
        : undefined,
      caCert: caCert
        ? {
            subject: caCert.getSubject(),
            issuer: caCert.getIssuer(),
            notBefore: caCert.getNotBefore()?.toDate() || new Date(),
            notAfter: caCert.getNotAfter()?.toDate() || new Date(),
            daysUntilExpiry: caCert.getDaysUntilExpiry(),
            isValid: caCert.getIsValid(),
          }
        : undefined,
    };
  }

  /**
   * Generate server certificates
   */
  async generateServerCertificates(): Promise<{ success: boolean; message: string }> {
    const request = new pb.GenerateServerCertificatesRequest();

    const response = await this.promisify<
      pb.GenerateServerCertificatesRequest,
      pb.GenerateServerCertificatesResponse
    >(this.client.generateServerCertificates)(request);

    return {
      success: response.getSuccess(),
      message: response.getMessage(),
    };
  }

  /**
   * Renew server certificates
   */
  async renewServerCertificates(): Promise<{ success: boolean; message: string }> {
    const request = new pb.RenewServerCertificatesRequest();

    const response = await this.promisify<
      pb.RenewServerCertificatesRequest,
      pb.RenewServerCertificatesResponse
    >(this.client.renewServerCertificates)(request);

    return {
      success: response.getSuccess(),
      message: response.getMessage(),
    };
  }

  /**
   * Download certificate by ID and type
   */
  async downloadCertificate(
    certId: string,
    certType: CertificateDownloadType
  ): Promise<{
    content: Uint8Array;
    filename: string;
    contentType: string;
  }> {
    const request = new pb.DownloadCertificateRequest();
    request.setId(certId);
    request.setCertType(certType);

    const response = await this.promisify<
      pb.DownloadCertificateRequest,
      pb.DownloadCertificateResponse
    >(this.client.downloadCertificate)(request);

    return {
      content: response.getContent_asU8(),
      filename: response.getFilename(),
      contentType: response.getContentType(),
    };
  }

  // ============================================================================
  // ENDPOINT METHODS (Extended)
  // ============================================================================

  /**
   * Create endpoint
   */
  async createEndpoint(data: CreateEndpointInput): Promise<GrpcEndpointResult> {
    const request = new pb.CreateEndPointRequest();
    request.setEndpoint(buildEndpointMessage(data));

    // Service returns EndPoint directly
    const ep = await this.promisify<pb.CreateEndPointRequest, pb.EndPoint>(
      this.client.createEndPoint
    )(request);

    return mapEndpointProto(ep);
  }

  /**
   * Update endpoint
   * Uses FieldMask to track which fields are being updated, enabling partial updates
   * for boolean fields without resetting unspecified values to false.
   */
  async updateEndpoint(
    epEui: string,
    data: UpdateEndpointInput
  ): Promise<GrpcEndpointResult> {
    const { endpoint, paths } = buildEndpointUpdate(epEui, data);

    const request = new pb.UpdateEndPointRequest();
    request.setEndpoint(endpoint);

    // Set new EUI for cascade rename if provided
    if (data.newEpEui !== undefined) {
      request.setNewEpEui(data.newEpEui);
    }

    // Set update mask if any fields are being updated
    if (paths.length > 0) {
      const mask = new FieldMask();
      mask.setPathsList(paths);
      request.setUpdateMask(mask);
    }

    // Service returns EndPoint directly
    const ep = await this.promisify<pb.UpdateEndPointRequest, pb.EndPoint>(
      this.client.updateEndPoint
    )(request);

    return mapEndpointProto(ep);
  }

  // ============================================================================
  // BASE STATION METHODS (Extended)
  // ============================================================================

  /**
   * Create base station
   */
  async createBaseStation(data: {
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
    const basestation = new pb.BaseStation();
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

    const request = new pb.CreateBaseStationRequest();
    request.setBasestation(basestation);

    const response = await this.promisify<pb.CreateBaseStationRequest, pb.BaseStation>(
      this.client.createBaseStation
    )(request);

    return {
      bsEui: response.getBseui(),
      tenantId: response.getTenantId(),
      name: response.getName(),
      description: response.getDescription() || undefined,
      latitude: response.getLatitude()?.getValue() ?? undefined,
      longitude: response.getLongitude()?.getValue() ?? undefined,
      altitude: response.getAltitude()?.getValue() ?? undefined,
      locationSource: response.getLocationSource() || undefined,
      locationUpdatedAt: response.getLocationUpdatedAt()?.toDate(),
      status: response.getStatus(),
    };
  }

  /**
   * Update base station
   */
  async updateBaseStation(
    bsEui: string,
    data: {
      name?: string;
      description?: string;
      latitude?: number | null;
      longitude?: number | null;
      altitude?: number | null;
    }
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
    const basestation = new pb.BaseStation();
    basestation.setBseui(bsEui);

    const paths: string[] = [];

    if (data.name !== undefined) {
      basestation.setName(data.name);
      paths.push('name');
    }
    if (data.description !== undefined) {
      basestation.setDescription(data.description);
      paths.push('description');
    }
    if (data.latitude !== undefined) {
      if (data.latitude !== null) {
        const w = new DoubleValue();
        w.setValue(data.latitude);
        basestation.setLatitude(w);
      }
      paths.push('latitude');
    }
    if (data.longitude !== undefined) {
      if (data.longitude !== null) {
        const w = new DoubleValue();
        w.setValue(data.longitude);
        basestation.setLongitude(w);
      }
      paths.push('longitude');
    }
    if (data.altitude !== undefined) {
      if (data.altitude !== null) {
        const w = new DoubleValue();
        w.setValue(data.altitude);
        basestation.setAltitude(w);
      }
      paths.push('altitude');
    }

    const request = new pb.UpdateBaseStationRequest();
    request.setBasestation(basestation);

    if (paths.length > 0) {
      const mask = new FieldMask();
      mask.setPathsList(paths);
      request.setUpdateMask(mask);
    }

    const response = await this.promisify<pb.UpdateBaseStationRequest, pb.BaseStation>(
      this.client.updateBaseStation
    )(request);

    return {
      bsEui: response.getBseui(),
      tenantId: response.getTenantId(),
      name: response.getName(),
      description: response.getDescription() || undefined,
      latitude: response.getLatitude()?.getValue() ?? undefined,
      longitude: response.getLongitude()?.getValue() ?? undefined,
      altitude: response.getAltitude()?.getValue() ?? undefined,
      locationSource: response.getLocationSource() || undefined,
      locationUpdatedAt: response.getLocationUpdatedAt()?.toDate(),
      status: response.getStatus(),
    };
  }

  /**
   * Update base station EUI with cascade to all dependent tables.
   */
  async updateBaseStationEui(
    bsEui: string,
    newBsEui: string
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
    const request = new pb.UpdateBaseStationEuiRequest();
    request.setBsEui(bsEui);
    request.setNewBsEui(newBsEui);

    const response = await this.promisify<pb.UpdateBaseStationEuiRequest, pb.BaseStation>(
      this.client.updateBaseStationEui
    )(request);

    return {
      bsEui: response.getBseui(),
      tenantId: response.getTenantId(),
      name: response.getName(),
      description: response.getDescription() || undefined,
      latitude: response.getLatitude()?.getValue() ?? undefined,
      longitude: response.getLongitude()?.getValue() ?? undefined,
      altitude: response.getAltitude()?.getValue() ?? undefined,
      locationSource: response.getLocationSource() || undefined,
      locationUpdatedAt: response.getLocationUpdatedAt()?.toDate(),
      status: response.getStatus(),
    };
  }

  // ============================================================================
  // BASE STATION MESSAGE METHODS
  // ============================================================================

  /**
   * List base station messages
   * Added direction and epEui filters.
   */
  async listBaseStationMessages(
    bsEui: string,
    params?: {
      pageSize?: number;
      pageToken?: string;
      startTime?: Date;
      endTime?: Date;
      direction?: 'uplink' | 'downlink';
      epEui?: string;
    }
  ): Promise<{
    messages: Array<{
      id: string;
      bsEui: string;
      epEui: string;
      payload: string;
      rssi: number;
      snr: number;
      eqSnr: number;
      uplinkMode: string;
      packetCounter: number;
      receivedAt: Date;
      direction: 'uplink' | 'downlink';
    }>;
    nextPageToken?: string;
    totalCount: number;
  }> {
    const request = new pb.ListBaseStationMessagesRequest();
    request.setBsEui(bsEui);
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.pageToken) request.setPageToken(params.pageToken);
    if (params?.direction) request.setDirection(params.direction);
    if (params?.epEui) request.setEpEui(params.epEui);
    if (params?.startTime) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(params.startTime);
      request.setStartTime(ts);
    }
    if (params?.endTime) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(params.endTime);
      request.setEndTime(ts);
    }

    const response = await this.promisify<
      pb.ListBaseStationMessagesRequest,
      pb.ListBaseStationMessagesResponse
    >(this.client.listBaseStationMessages)(request);

    return {
      messages: response.getMessagesList().map((m) => ({
        id: m.getId(),
        bsEui: m.getBsEui(),
        epEui: m.getEpEui(),
        payload: m.getPayload_asB64(),
        rssi: m.getRssi(),
        snr: m.getSnr(),
        eqSnr: m.getEqSnr(),
        uplinkMode: m.getUplinkMode(),
        packetCounter: m.getPacketCounter(),
        receivedAt: m.getReceivedAt()?.toDate() || new Date(),
        direction: m.getDirection() as 'uplink' | 'downlink',
      })),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  /**
   * Get base station message
   */
  async getBaseStationMessage(
    bsEui: string,
    messageId: string
  ): Promise<{
    id: string;
    bsEui: string;
    epEui: string;
    payload: string;
    rssi: number;
    snr: number;
    eqSnr: number;
    uplinkMode: string;
    packetCounter: number;
    receivedAt: Date;
    direction: 'uplink' | 'downlink';
  } | null> {
    const request = new pb.GetBaseStationMessageRequest();
    request.setBsEui(bsEui);
    request.setMessageId(messageId);

    try {
      const response = await this.promisify<
        pb.GetBaseStationMessageRequest,
        pb.GetBaseStationMessageResponse
      >(this.client.getBaseStationMessage)(request);

      const msg = response.getMessage();
      if (!msg) return null;

      return {
        id: msg.getId(),
        bsEui: msg.getBsEui(),
        epEui: msg.getEpEui(),
        payload: msg.getPayload_asB64(),
        rssi: msg.getRssi(),
        snr: msg.getSnr(),
        eqSnr: msg.getEqSnr(),
        uplinkMode: msg.getUplinkMode(),
        packetCounter: msg.getPacketCounter(),
        receivedAt: msg.getReceivedAt()?.toDate() || new Date(),
        direction: msg.getDirection() as 'uplink' | 'downlink',
      };
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Get base station message stats
   * Returns null when no stats are available (let caller handle absence)
   */
  async getBaseStationMessageStats(
    bsEui: string
  ): Promise<{
    bsEui: string;
    totalMessages: number;
    uniqueEndpoints: number;
    avgRssi: number;
    avgSnr: number;
    messagesToday: number;
    messagesThisWeek: number;
    messagesThisMonth: number;
    firstMessageAt?: Date;
    lastMessageAt?: Date;
  } | null> {
    const request = new pb.GetBaseStationMessageStatsRequest();
    request.setBsEui(bsEui);

    const response = await this.promisify<
      pb.GetBaseStationMessageStatsRequest,
      pb.GetBaseStationMessageStatsResponse
    >(this.client.getBaseStationMessageStats)(request);

    const stats = response.getStats();
    if (!stats) {
      return null; // Let caller handle absence
    }

    return {
      bsEui: stats.getBsEui(),
      totalMessages: stats.getTotalMessages(),
      uniqueEndpoints: stats.getUniqueEndpoints(),
      avgRssi: stats.getAvgRssi(),
      avgSnr: stats.getAvgSnr(),
      messagesToday: stats.getMessagesToday(),
      messagesThisWeek: stats.getMessagesThisWeek(),
      messagesThisMonth: stats.getMessagesThisMonth(),
      firstMessageAt: stats.getFirstMessageAt()?.toDate(),
      lastMessageAt: stats.getLastMessageAt()?.toDate(),
    };
  }

  /**
   * Search base station messages
   * Added direction and epEui filters.
   */
  async searchBaseStationMessages(
    bsEui: string,
    query: string,
    params?: {
      pageSize?: number;
      pageToken?: string;
      startTime?: Date;
      endTime?: Date;
      direction?: 'uplink' | 'downlink';
      epEui?: string;
    }
  ): Promise<{
    messages: Array<{
      id: string;
      bsEui: string;
      epEui: string;
      payload: string;
      rssi: number;
      snr: number;
      eqSnr: number;
      uplinkMode: string;
      packetCounter: number;
      receivedAt: Date;
      direction: 'uplink' | 'downlink';
    }>;
    nextPageToken?: string;
    totalCount: number;
  }> {
    const request = new pb.SearchBaseStationMessagesRequest();
    request.setBsEui(bsEui);
    request.setQuery(query);
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.pageToken) request.setPageToken(params.pageToken);
    if (params?.direction) request.setDirection(params.direction);
    if (params?.epEui) request.setEpEui(params.epEui);
    if (params?.startTime) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(params.startTime);
      request.setStartTime(ts);
    }
    if (params?.endTime) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(params.endTime);
      request.setEndTime(ts);
    }

    const response = await this.promisify<
      pb.SearchBaseStationMessagesRequest,
      pb.SearchBaseStationMessagesResponse
    >(this.client.searchBaseStationMessages)(request);

    return {
      messages: response.getMessagesList().map((m) => ({
        id: m.getId(),
        bsEui: m.getBsEui(),
        epEui: m.getEpEui(),
        payload: m.getPayload_asB64(),
        rssi: m.getRssi(),
        snr: m.getSnr(),
        eqSnr: m.getEqSnr(),
        uplinkMode: m.getUplinkMode(),
        packetCounter: m.getPacketCounter(),
        receivedAt: m.getReceivedAt()?.toDate() || new Date(),
        direction: m.getDirection() as 'uplink' | 'downlink',
      })),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  /**
   * Export base station messages
   * Added direction and epEui filters.
   */
  async exportBaseStationMessages(
    bsEui: string,
    format: 'csv' | 'json',
    params?: {
      startTime?: Date;
      endTime?: Date;
      direction?: 'uplink' | 'downlink';
      epEui?: string;
    }
  ): Promise<{
    content: Uint8Array;
    filename: string;
    contentType: string;
  }> {
    const request = new pb.ExportBaseStationMessagesRequest();
    request.setBsEui(bsEui);
    request.setFormat(format);
    if (params?.direction) request.setDirection(params.direction);
    if (params?.epEui) request.setEpEui(params.epEui);
    if (params?.startTime) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(params.startTime);
      request.setStartTime(ts);
    }
    if (params?.endTime) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(params.endTime);
      request.setEndTime(ts);
    }

    const response = await this.promisify<
      pb.ExportBaseStationMessagesRequest,
      pb.ExportBaseStationMessagesResponse
    >(this.client.exportBaseStationMessages)(request);

    return {
      content: response.getContent_asU8(),
      filename: response.getFilename(),
      contentType: response.getContentType(),
    };
  }

  // ============================================================================
  // BASE STATION ACTIVITY METHODS
  // ============================================================================

  /** Shared activity event mapping for both BS and endpoint activity feeds. */
  private mapActivityEvent(event: pb.Event) {
    return {
      id: event.getId(),
      eventType: event.getEventType(),
      category: event.getCategory(),
      severity: event.getSeverity(),
      title: event.getTitle(),
      description: event.getDescription(),
      timestamp: event.getTimestamp()?.toDate() || new Date(),
    };
  }

  /** Map BaseStationReceptionInfo list to transport DTO. */
  private mapBaseStationReceptions(bsList: pb.BaseStationReceptionInfo[]) {
    if (bsList.length === 0) return undefined;
    return bsList.map((bs) => {
      const sp = bs.getSubpackets();
      return {
        bsEui: bs.getBsEui(),
        rxTime: bs.getRxTime(),
        snr: bs.getSnr(),
        rssi: bs.getRssi(),
        eqSnr: bs.getEqSnr()?.getValue(),
        rxDuration: bs.getRxDuration()?.getValue(),
        profile: bs.getProfile()?.getValue(),
        mode: bs.getMode()?.getValue(),
        dlRxSnr: bs.getDlRxSnr()?.getValue(),
        dlRxRssi: bs.getDlRxRssi()?.getValue(),
        subpackets: sp ? {
          snr: sp.getSnrList(),
          rssi: sp.getRssiList(),
          frequency: sp.getFrequencyList(),
          phase: sp.getPhaseList().length > 0 ? sp.getPhaseList() : undefined,
        } : undefined,
      };
    });
  }

  /**
   * Map a BaseStationMessage proto to a rich transport DTO.
   * Used by activity feed methods.
   */
  private mapBsActivityMessage(message: pb.BaseStationMessage) {
    const bsList = message.getBaseStationsList();

    let decodedPayload: Record<string, unknown> | undefined;
    const rawDecoded = message.getDecodedPayload_asU8();
    if (rawDecoded && rawDecoded.length > 0) {
      try {
        decodedPayload = JSON.parse(new TextDecoder().decode(rawDecoded));
      } catch {
        // Malformed JSON — preserve status/error fields, skip payload
      }
    }

    return {
      id: message.getId(),
      bsEui: message.getBsEui(),
      epEui: message.getEpEui(),
      userData: message.getPayload_asB64(),
      rssi: message.getRssi(),
      snr: message.getSnr(),
      eqSnr: message.getEqSnr(),
      messageType: message.getUplinkMode() || 'ulData',
      packetCnt: message.getPacketCounter(),
      receivedAt: message.getReceivedAt()?.toDate() || new Date(),
      direction: message.getDirection() as 'uplink' | 'downlink',
      baseStations: this.mapBaseStationReceptions(bsList),
      duplicate: message.getDuplicate() || undefined,
      decodedPayload,
      decodeStatus: message.getDecodeStatus() || undefined,
      decodeErrorCode: message.getDecodeErrorCode() || undefined,
    };
  }

  /**
   * Map a Message proto (used for endpoint activity) to a rich transport DTO.
   */
  private mapEpActivityMessage(message: pb.Message) {
    let decodedPayload: Record<string, unknown> | undefined;
    const rawDecoded = message.getDecodedPayload_asU8();
    if (rawDecoded && rawDecoded.length > 0) {
      try {
        decodedPayload = JSON.parse(new TextDecoder().decode(rawDecoded));
      } catch {
        // Malformed JSON — preserve status/error fields, skip payload
      }
    }

    const bsList = message.getBaseStationsList();
    return {
      id: message.getId(),
      bsEui: message.getBseui(),
      epEui: message.getEpeui(),
      userData: message.getPayload_asB64(),
      rssi: message.getRssi(),
      snr: message.getSnr(),
      eqSnr: message.getEqSnr(),
      messageType: message.getUplinkMode() || 'ulData',
      packetCnt: message.getPacketCounter(),
      receivedAt: message.getReceivedAt()?.toDate() || new Date(),
      direction: 'uplink' as const,
      baseStations: this.mapBaseStationReceptions(bsList),
      duplicate: message.getDuplicate() || undefined,
      decodedPayload,
      decodeStatus: message.getDecodeStatus() || undefined,
      decodeErrorCode: message.getDecodeErrorCode() || undefined,
    };
  }

  /**
   * List base station activity (unified events and messages)
   */
  async listBaseStationActivity(
    bsEui: string,
    params?: {
      pageSize?: number;
      pageToken?: string;
      startTime?: Date;
      endTime?: Date;
    }
  ): Promise<{
    items: Array<{
      type: 'event' | 'message';
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
        direction: 'uplink' | 'downlink';
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
      };
    }>;
    nextPageToken?: string;
    totalCount: number;
  }> {
    const request = new pb.ListBaseStationActivityRequest();
    request.setBsEui(bsEui);
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.pageToken) request.setPageToken(params.pageToken);
    if (params?.startTime) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(params.startTime);
      request.setStartTime(ts);
    }
    if (params?.endTime) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(params.endTime);
      request.setEndTime(ts);
    }

    const response = await this.promisify<
      pb.ListBaseStationActivityRequest,
      pb.ListBaseStationActivityResponse
    >(this.client.listBaseStationActivity)(request);

    return {
      items: response.getItemsList().map((item) => {
        const occurredAt = item.getOccurredAt()?.toDate() || new Date();
        const event = item.getEvent();
        const message = item.getMessage();

        if (event) {
          return {
            type: 'event' as const,
            occurredAt,
            event: this.mapActivityEvent(event),
          };
        } else if (message) {
          return {
            type: 'message' as const,
            occurredAt,
            message: this.mapBsActivityMessage(message),
          };
        }

        return {
          type: 'event' as const,
          occurredAt,
        };
      }),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  /**
   * List endpoint activity (unified events and messages)
   */
  async listEndpointActivity(
    epEui: string,
    params?: {
      pageSize?: number;
      pageToken?: string;
      startTime?: Date;
      endTime?: Date;
    }
  ): Promise<{
    items: Array<{
      type: 'event' | 'message';
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
        direction: 'uplink' | 'downlink';
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
  }> {
    const request = new pb.ListEndpointActivityRequest();
    request.setEpEui(epEui);
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.pageToken) request.setPageToken(params.pageToken);
    if (params?.startTime) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(params.startTime);
      request.setStartTime(ts);
    }
    if (params?.endTime) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(params.endTime);
      request.setEndTime(ts);
    }

    const response = await this.promisify<
      pb.ListEndpointActivityRequest,
      pb.ListEndpointActivityResponse
    >(this.client.listEndpointActivity)(request);

    return {
      items: response.getItemsList().map((item) => {
        const occurredAt = item.getOccurredAt()?.toDate() || new Date();
        const event = item.getEvent();
        const message = item.getMessage();

        if (event) {
          return {
            type: 'event' as const,
            occurredAt,
            event: this.mapActivityEvent(event),
          };
        } else if (message) {
          return {
            type: 'message' as const,
            occurredAt,
            message: this.mapEpActivityMessage(message),
          };
        }

        return {
          type: 'event' as const,
          occurredAt,
        };
      }),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  // ============================================================================
  // DOWNLINK METHODS
  // ============================================================================

  /**
   * Map proto DownlinkMessage to SCACIDownlinkQueueDTO
   */
  private mapDownlinkMessage(
    m: pb.DownlinkMessage
  ): import('@api-types/api').SCACIDownlinkQueueDTO {
    return {
      id: m.getId(),
      queId: m.getQueId(),
      epEui: m.getEpeui(),
      payload: m.getPayload_asB64(),
      priority: m.getPriority(),
      status: m.getStatus(),
      cntDepend: m.getCntDepend(),
      packetCnt: m.getPacketCntList(),
      format: m.getFormat(),
      responseExp: m.getResponseExp(),
      responsePrio: m.getResponsePrio(),
      dlWindReq: m.getDlWindReq(),
      expOnly: m.getExpOnly(),
      dlRxStatQry: m.getDlRxStatQry(),
      result: m.getResult() || undefined,
      txTime: m.getTxTime() || undefined,
      bsEui: m.getBsEui() || undefined,
      createdAt: m.getCreatedAt()?.toDate().toISOString() ?? '',
      scheduledAt: m.getScheduledAt()?.toDate().toISOString(),
      transmittedAt: m.getTransmittedAt()?.toDate().toISOString(),
      transmissionPacketCnt: m.getTransmissionPacketCnt(),
    };
  }

  /**
   * Send a downlink message to an endpoint
   */
  async sendDownlink(
    data: import('@api-types/api').SendDownlinkRequest
  ): Promise<import('@api-types/api').SendDownlinkResponse> {
    const request = new pb.SendDownlinkRequest();
    request.setEpeui(data.epEui);
    request.setPriority(data.priority);
    request.setCntDepend(data.cntDepend);
    request.setFormat(data.format);
    request.setResponseExp(data.responseExp);
    request.setResponsePrio(data.responsePrio);
    request.setDlWindReq(data.dlWindReq);
    request.setExpOnly(data.expOnly);
    request.setDlRxStatQry(data.dlRxStatQry);

    // Convert hex payloads to bytes (skip empty strings for ACK-only path)
    const payloadBytes = data.payloads.filter((p) => p.length > 0).map((p) => hexToBytes(p));
    request.setPayloadsList(payloadBytes);

    if (data.packetCnt) {
      request.setPacketCntList(data.packetCnt);
    }

    const response = await this.promisify<pb.SendDownlinkRequest, pb.SendDownlinkResponse>(
      this.client.sendDownlink
    )(request);

    return {
      id: response.getId(),
      status: response.getStatus(),
    };
  }

  /**
   * Revoke a queued downlink message
   */
  async revokeDownlink(data: {
    epEui: string;
    queueId: string;
  }): Promise<import('@api-types/api').RevokeDownlinkResponse> {
    const request = new pb.RevokeDownlinkRequest();
    request.setEpeui(data.epEui);
    request.setQueueId(data.queueId);

    const response = await this.promisify<pb.RevokeDownlinkRequest, pb.RevokeDownlinkResponse>(
      this.client.revokeDownlink
    )(request);

    return {
      status: response.getStatus(),
      message: response.getMessage(),
    };
  }

  /**
   * List queued downlink messages for an endpoint
   */
  async listDownlinkQueue(data: {
    epEui: string;
    pageSize?: number;
    pageToken?: string;
  }): Promise<import('@api-types/api').DownlinkQueueResponse> {
    const request = new pb.ListDownlinkQueueRequest();
    request.setEpeui(data.epEui);
    if (data.pageSize) request.setPageSize(data.pageSize);
    if (data.pageToken) request.setPageToken(data.pageToken);

    const response = await this.promisify<
      pb.ListDownlinkQueueRequest,
      pb.ListDownlinkQueueResponse
    >(this.client.listDownlinkQueue)(request);

    return {
      messages: response.getMessagesList().map((m) => this.mapDownlinkMessage(m)),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  /**
   * Get downlink transmission results for an endpoint
   */
  async getDownlinkResults(data: {
    epEui: string;
    statusFilter?: string;
    pageSize?: number;
    pageToken?: string;
  }): Promise<import('@api-types/api').DownlinkResultsResponse> {
    const request = new pb.GetDownlinkResultsRequest();
    request.setEpeui(data.epEui);
    if (data.statusFilter) request.setStatusFilter(data.statusFilter);
    if (data.pageSize) request.setPageSize(data.pageSize);
    if (data.pageToken) request.setPageToken(data.pageToken);

    const response = await this.promisify<
      pb.GetDownlinkResultsRequest,
      pb.GetDownlinkResultsResponse
    >(this.client.getDownlinkResults)(request);

    return {
      results: response.getResultsList().map((m) => this.mapDownlinkMessage(m)),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  // ============================================================================
  // DL RX STATUS METHODS
  // ============================================================================

  // ============================================================================
  // UL TRANSMIT METHOD
  // ============================================================================

  /**
   * Send UL data transmit
   */
  async sendULTransmit(data: {
    epEui: string;
    userData: string; // hex encoded
    packetCnt: number;
    nwkSnKey?: string; // hex encoded
    shAddr?: number;
    format?: number;
    profile?: string;
  }): Promise<{
    id: string;
    status: string;
    message: string;
  }> {
    const request = new pb.SendULTransmitRequest();
    request.setEpeui(data.epEui);
    // Convert hex string to Uint8Array for bytes field
    request.setUserData(hexToBytes(data.userData));
    request.setPacketCnt(data.packetCnt);
    if (data.nwkSnKey) {
      // Convert hex string to Uint8Array for bytes field
      request.setNwkSnKey(hexToBytes(data.nwkSnKey));
    }
    if (data.shAddr !== undefined) request.setShAddr(data.shAddr);
    if (data.format !== undefined) request.setFormat(data.format);
    if (data.profile) request.setProfile(data.profile);

    const response = await this.promisify<pb.SendULTransmitRequest, pb.SendULTransmitResponse>(
      this.client.sendULTransmit
    )(request);

    return {
      id: response.getId(),
      status: response.getStatus(),
      message: response.getMessage(),
    };
  }

  // ============================================================================
  // BLUEPRINT METHODS: MANUFACTURERS
  // ============================================================================

  /**
   * List manufacturers
   */
  async listManufacturers(): Promise<
    Array<{
      id: string;
      name: string;
      code: string;
      description?: string;
      website?: string;
      contactEmail?: string;
      tenantId: string;
      isVerified: boolean;
      modelCount: number;
      createdAt?: Date;
      updatedAt?: Date;
    }>
  > {
    const request = new pb.ListManufacturersRequest();

    const response = await this.promisify<
      pb.ListManufacturersRequest,
      pb.ListManufacturersResponse
    >(this.client.listManufacturers)(request);

    return response.getManufacturersList().map((m) => ({
      id: m.getId(),
      name: m.getName(),
      code: m.getCode(),
      description: m.getDescription() || undefined,
      website: m.getWebsite() || undefined,
      contactEmail: m.getContactEmail() || undefined,
      tenantId: m.getTenantId(),
      isVerified: m.getIsVerified(),
      modelCount: m.getModelCount(),
      createdAt: m.getCreatedAt()?.toDate(),
      updatedAt: m.getUpdatedAt()?.toDate(),
    }));
  }

  /**
   * Get manufacturer by ID
   */
  async getManufacturer(manufacturerId: string): Promise<{
    id: string;
    name: string;
    code: string;
    description?: string;
    website?: string;
    contactEmail?: string;
    tenantId: string;
    isVerified: boolean;
    modelCount: number;
    createdAt?: Date;
    updatedAt?: Date;
  } | null> {
    const request = new pb.GetManufacturerRequest();
    request.setId(manufacturerId);

    try {
      const response = await this.promisify<pb.GetManufacturerRequest, pb.GetManufacturerResponse>(
        this.client.getManufacturer
      )(request);

      const m = response.getManufacturer();
      if (!m) return null;

      return {
        id: m.getId(),
        name: m.getName(),
        code: m.getCode(),
        description: m.getDescription() || undefined,
        website: m.getWebsite() || undefined,
        contactEmail: m.getContactEmail() || undefined,
        tenantId: m.getTenantId(),
        isVerified: m.getIsVerified(),
        modelCount: m.getModelCount(),
        createdAt: m.getCreatedAt()?.toDate(),
        updatedAt: m.getUpdatedAt()?.toDate(),
      };
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Create manufacturer
   */
  async createManufacturer(data: {
    name: string;
    code?: string;
    description?: string;
    website?: string;
    contactEmail?: string;
  }): Promise<{
    id: string;
    name: string;
    code: string;
    description?: string;
    website?: string;
    contactEmail?: string;
    tenantId: string;
    isVerified: boolean;
    modelCount: number;
    createdAt?: Date;
    updatedAt?: Date;
  }> {
    const request = new pb.CreateManufacturerRequest();
    request.setName(data.name);
    if (data.code) request.setCode(data.code);
    if (data.description) request.setDescription(data.description);
    if (data.website) request.setWebsite(data.website);
    if (data.contactEmail) request.setContactEmail(data.contactEmail);

    const response = await this.promisify<
      pb.CreateManufacturerRequest,
      pb.CreateManufacturerResponse
    >(this.client.createManufacturer)(request);

    const m = response.getManufacturer();
    if (!m) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_CREATE_MANUFACTURER_RESPONSE);
    }

    return {
      id: m.getId(),
      name: m.getName(),
      code: m.getCode(),
      description: m.getDescription() || undefined,
      website: m.getWebsite() || undefined,
      contactEmail: m.getContactEmail() || undefined,
      tenantId: m.getTenantId(),
      isVerified: m.getIsVerified(),
      modelCount: m.getModelCount(),
      createdAt: m.getCreatedAt()?.toDate(),
      updatedAt: m.getUpdatedAt()?.toDate(),
    };
  }

  /**
   * Update manufacturer
   */
  async updateManufacturer(
    manufacturerId: string,
    data: { name?: string; description?: string; website?: string; contactEmail?: string }
  ): Promise<void> {
    const request = new pb.UpdateManufacturerRequest();
    request.setId(manufacturerId);
    if (data.name) request.setName(data.name);
    if (data.description !== undefined) request.setDescription(data.description);
    if (data.website !== undefined) request.setWebsite(data.website);
    if (data.contactEmail !== undefined) request.setContactEmail(data.contactEmail);

    await this.promisify<pb.UpdateManufacturerRequest, pb.UpdateManufacturerResponse>(
      this.client.updateManufacturer
    )(request);
  }

  /**
   * Delete manufacturer
   */
  async deleteManufacturer(manufacturerId: string): Promise<void> {
    const request = new pb.DeleteManufacturerRequest();
    request.setId(manufacturerId);

    await this.promisify<pb.DeleteManufacturerRequest, pb.DeleteManufacturerResponse>(
      this.client.deleteManufacturer
    )(request);
  }

  // ============================================================================
  // BLUEPRINT METHODS: DEVICE MODELS
  // ============================================================================

  /**
   * List device models for a manufacturer
   */
  async listDeviceModels(manufacturerId: string): Promise<
    Array<{
      id: string;
      manufacturerId: string;
      tenantId: string;
      name: string;
      code: string;
      typeEui?: string;
      description?: string;
      datasheetUrl?: string;
      blueprintCount: number;
      createdAt?: Date;
      updatedAt?: Date;
    }>
  > {
    const request = new pb.ListDeviceModelsRequest();
    request.setManufacturerId(manufacturerId);

    const response = await this.promisify<pb.ListDeviceModelsRequest, pb.ListDeviceModelsResponse>(
      this.client.listDeviceModels
    )(request);

    return response.getDeviceModelsList().map((m) => ({
      id: m.getId(),
      manufacturerId: m.getManufacturerId(),
      tenantId: m.getTenantId(),
      name: m.getName(),
      code: m.getCode(),
      typeEui: m.getTypeEui() || undefined,
      description: m.getDescription() || undefined,
      datasheetUrl: m.getDatasheetUrl() || undefined,
      blueprintCount: m.getBlueprintCount(),
      createdAt: m.getCreatedAt()?.toDate(),
      updatedAt: m.getUpdatedAt()?.toDate(),
    }));
  }

  /**
   * Get device model by ID
   */
  async getDeviceModel(modelId: string): Promise<{
    id: string;
    manufacturerId: string;
    tenantId: string;
    name: string;
    code: string;
    typeEui?: string;
    description?: string;
    datasheetUrl?: string;
    blueprintCount: number;
    createdAt?: Date;
    updatedAt?: Date;
  } | null> {
    const request = new pb.GetDeviceModelRequest();
    request.setId(modelId);

    try {
      const response = await this.promisify<pb.GetDeviceModelRequest, pb.GetDeviceModelResponse>(
        this.client.getDeviceModel
      )(request);

      const m = response.getDeviceModel();
      if (!m) return null;

      return {
        id: m.getId(),
        manufacturerId: m.getManufacturerId(),
        tenantId: m.getTenantId(),
        name: m.getName(),
        code: m.getCode(),
        typeEui: m.getTypeEui() || undefined,
        description: m.getDescription() || undefined,
        datasheetUrl: m.getDatasheetUrl() || undefined,
        blueprintCount: m.getBlueprintCount(),
        createdAt: m.getCreatedAt()?.toDate(),
        updatedAt: m.getUpdatedAt()?.toDate(),
      };
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Create device model
   */
  async createDeviceModel(
    manufacturerId: string,
    data: {
      name: string;
      code: string;
      typeEui?: string;
      description?: string;
    }
  ): Promise<{
    id: string;
    manufacturerId: string;
    tenantId: string;
    name: string;
    code: string;
    typeEui?: string;
    description?: string;
    datasheetUrl?: string;
    blueprintCount: number;
    createdAt?: Date;
    updatedAt?: Date;
  }> {
    const request = new pb.CreateDeviceModelRequest();
    request.setManufacturerId(manufacturerId);
    request.setName(data.name);
    request.setCode(data.code);
    if (data.typeEui) request.setTypeEui(data.typeEui);
    if (data.description) request.setDescription(data.description);

    const response = await this.promisify<
      pb.CreateDeviceModelRequest,
      pb.CreateDeviceModelResponse
    >(this.client.createDeviceModel)(request);

    const m = response.getDeviceModel();
    if (!m) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_CREATE_DEVICE_MODEL_RESPONSE);
    }

    return {
      id: m.getId(),
      manufacturerId: m.getManufacturerId(),
      tenantId: m.getTenantId(),
      name: m.getName(),
      code: m.getCode(),
      typeEui: m.getTypeEui() || undefined,
      description: m.getDescription() || undefined,
      datasheetUrl: m.getDatasheetUrl() || undefined,
      blueprintCount: m.getBlueprintCount(),
      createdAt: m.getCreatedAt()?.toDate(),
      updatedAt: m.getUpdatedAt()?.toDate(),
    };
  }

  /**
   * Update device model
   */
  async updateDeviceModel(
    modelId: string,
    data: {
      name?: string;
      description?: string;
    }
  ): Promise<void> {
    const request = new pb.UpdateDeviceModelRequest();
    request.setId(modelId);
    if (data.name) request.setName(data.name);
    if (data.description !== undefined) request.setDescription(data.description);

    await this.promisify<pb.UpdateDeviceModelRequest, pb.UpdateDeviceModelResponse>(
      this.client.updateDeviceModel
    )(request);
  }

  /**
   * Delete device model
   */
  async deleteDeviceModel(modelId: string): Promise<void> {
    const request = new pb.DeleteDeviceModelRequest();
    request.setId(modelId);

    await this.promisify<pb.DeleteDeviceModelRequest, pb.DeleteDeviceModelResponse>(
      this.client.deleteDeviceModel
    )(request);
  }

  // ============================================================================
  // BLUEPRINT METHODS: BLUEPRINTS
  // ============================================================================

  /**
   * List blueprints for a device model
   */
  async listBlueprints(modelId: string): Promise<Array<BlueprintTransportDTO>> {
    const request = new pb.ListBlueprintsRequest();
    request.setDeviceModelId(modelId);

    const response = await this.promisify<pb.ListBlueprintsRequest, pb.ListBlueprintsResponse>(
      this.client.listBlueprints
    )(request);

    return response.getBlueprintsList().map((b) => mapProtoBlueprintToTransport(b));
  }

  /**
   * Get blueprint by ID
   */
  async getBlueprint(blueprintId: string): Promise<BlueprintTransportDTO | null> {
    const request = new pb.GetBlueprintRequest();
    request.setId(blueprintId);

    try {
      const response = await this.promisify<pb.GetBlueprintRequest, pb.GetBlueprintResponse>(
        this.client.getBlueprint
      )(request);

      const b = response.getBlueprint();
      if (!b) return null;

      return mapProtoBlueprintToTransport(b);
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Create blueprint
   */
  async createBlueprint(
    modelId: string,
    data: {
      name: string;
      version: string;
      description?: string;
      decoderScript: string;
      isDefault?: boolean;
    }
  ): Promise<BlueprintTransportDTO> {
    const request = new pb.CreateBlueprintRequest();
    request.setDeviceModelId(modelId);
    request.setName(data.name);
    request.setVersion(data.version);
    request.setDecoderScript(stringToBytes(data.decoderScript));
    if (data.description) request.setDescription(data.description);
    if (data.isDefault !== undefined) request.setIsDefault(data.isDefault);

    const response = await this.promisify<pb.CreateBlueprintRequest, pb.CreateBlueprintResponse>(
      this.client.createBlueprint
    )(request);

    const b = response.getBlueprint();
    if (!b) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_CREATE_BLUEPRINT_RESPONSE);
    }

    return mapProtoBlueprintToTransport(b);
  }

  /**
   * Update blueprint
   */
  async updateBlueprint(
    blueprintId: string,
    data: {
      name?: string;
      version?: string;
      description?: string;
      decoderScript?: string;
    }
  ): Promise<void> {
    const request = new pb.UpdateBlueprintRequest();
    request.setId(blueprintId);
    if (data.name) request.setName(data.name);
    if (data.version) request.setVersion(data.version);
    if (data.description !== undefined) request.setDescription(data.description);
    if (data.decoderScript) request.setDecoderScript(stringToBytes(data.decoderScript));

    await this.promisify<pb.UpdateBlueprintRequest, pb.UpdateBlueprintResponse>(
      this.client.updateBlueprint
    )(request);
  }

  /**
   * Delete blueprint
   */
  async deleteBlueprint(blueprintId: string): Promise<void> {
    const request = new pb.DeleteBlueprintRequest();
    request.setId(blueprintId);

    await this.promisify<pb.DeleteBlueprintRequest, pb.DeleteBlueprintResponse>(
      this.client.deleteBlueprint
    )(request);
  }

  /**
   * Set blueprint as default
   */
  async setDefaultBlueprint(blueprintId: string): Promise<void> {
    const request = new pb.SetDefaultBlueprintRequest();
    request.setId(blueprintId);

    await this.promisify<pb.SetDefaultBlueprintRequest, pb.SetDefaultBlueprintResponse>(
      this.client.setDefaultBlueprint
    )(request);
  }

  /**
   * Submit blueprint to registry
   */
  async submitBlueprintToRegistry(
    blueprintId: string,
    data: {
      contributorName?: string;
      contributorEmail?: string;
      notes?: string;
    }
  ): Promise<{
    success: boolean;
    prUrl: string;
    commitSha: string;
    branchName: string;
  }> {
    const request = new pb.SubmitBlueprintToRegistryRequest();
    request.setId(blueprintId);
    if (data.contributorName) request.setContributorName(data.contributorName);
    if (data.contributorEmail) request.setContributorEmail(data.contributorEmail);
    if (data.notes) request.setNotes(data.notes);

    const response = await this.promisify<
      pb.SubmitBlueprintToRegistryRequest,
      pb.SubmitBlueprintToRegistryResponse
    >(this.client.submitBlueprintToRegistry)(request);

    return {
      success: response.getSuccess(),
      prUrl: response.getPrUrl(),
      commitSha: response.getCommitSha(),
      branchName: response.getBranchName(),
    };
  }

  /**
   * Create device model with default blueprint atomically
   */
  async createDeviceModelWithBlueprint(data: {
    manufacturerId: string;
    name: string;
    version: string;
    decoderScript: string;
  }): Promise<{
    deviceModel: {
      id: string;
      manufacturerId: string;
      tenantId: string;
      name: string;
      code: string;
      typeEui: string;
      description: string;
      datasheetUrl: string;
      blueprintCount: number;
      createdAt?: Date;
      updatedAt?: Date;
    };
    blueprint: BlueprintTransportDTO;
  }> {
    const request = new pb.CreateDeviceModelWithBlueprintRequest();
    request.setManufacturerId(data.manufacturerId);
    request.setName(data.name);
    request.setVersion(data.version);
    request.setDecoderScript(stringToBytes(data.decoderScript));

    const response = await this.promisify<
      pb.CreateDeviceModelWithBlueprintRequest,
      pb.CreateDeviceModelWithBlueprintResponse
    >(this.client.createDeviceModelWithBlueprint)(request);

    const model = response.getDeviceModel();
    const blueprint = response.getBlueprint();
    if (!model || !blueprint) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.INVALID_CREATE_BLUEPRINT_RESPONSE);
    }

    return {
      deviceModel: {
        id: model.getId(),
        manufacturerId: model.getManufacturerId(),
        tenantId: model.getTenantId(),
        name: model.getName(),
        code: model.getCode(),
        typeEui: model.getTypeEui(),
        description: model.getDescription(),
        datasheetUrl: model.getDatasheetUrl(),
        blueprintCount: model.getBlueprintCount(),
        createdAt: model.getCreatedAt()?.toDate(),
        updatedAt: model.getUpdatedAt()?.toDate(),
      },
      blueprint: mapProtoBlueprintToTransport(blueprint),
    };
  }

  /**
   * Preview blueprint decoding against a raw payload
   */
  async decodePreview(
    blueprintId: string,
    payload: Uint8Array,
    formatId: number
  ): Promise<{
    success: boolean;
    decodedPayload?: Record<string, unknown>;
    errorCode?: string;
    errorDetail?: string;
    formatId: number;
    blueprintVersion?: string;
  }> {
    const request = new pb.DecodePreviewRequest();
    request.setBlueprintId(blueprintId);
    request.setPayload(payload);
    request.setFormatId(formatId);

    const response = await this.promisify<pb.DecodePreviewRequest, pb.DecodePreviewResponse>(
      this.client.decodePreview
    )(request);

    let decodedPayload: Record<string, unknown> | undefined;
    const rawDecoded = response.getDecodedPayload_asU8();
    if (rawDecoded && rawDecoded.length > 0) {
      try {
        decodedPayload = JSON.parse(new TextDecoder().decode(rawDecoded));
      } catch {
        // Malformed JSON from decoder
      }
    }

    return {
      success: response.getSuccess(),
      decodedPayload,
      errorCode: response.getErrorCode() || undefined,
      errorDetail: response.getErrorDetail() || undefined,
      formatId: response.getFormatId(),
      blueprintVersion: response.getBlueprintVersion() || undefined,
    };
  }

  // ============================================================================
  // ENDPOINT MESSAGES & EVENTS (Stub methods for legacy compatibility)
  // ============================================================================

  /**
   * List messages for a specific endpoint
   */
  async listEndpointMessages(
    epEui: string,
    params?: { pageSize?: number; pageToken?: string }
  ): Promise<{
    messages: Array<{
      id: string;
      epEui: string;
      bsEui: string;
      payload: string;
      packetCounter: number;
      rssi: number;
      snr: number;
      eqSnr: number;
      receivedAt: Date;
      decodedPayload?: Record<string, unknown>;
      decodeStatus?: string;
      decodeErrorCode?: string;
      duplicate?: boolean;
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
    }>;
    nextPageToken?: string;
    totalCount: number;
  }> {
    const request = new pb.ListEndpointMessagesRequest();
    request.setEpEui(epEui);
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.pageToken) request.setPageToken(params.pageToken);

    const response = await this.promisify<
      pb.ListEndpointMessagesRequest,
      pb.ListEndpointMessagesResponse
    >(this.client.listEndpointMessages)(request);

    return {
      messages: response.getMessagesList().map((m) => {
        // Parse decoded payload JSON safely
        let decodedPayload: Record<string, unknown> | undefined;
        const rawDecoded = m.getDecodedPayload_asU8();
        if (rawDecoded && rawDecoded.length > 0) {
          try {
            decodedPayload = JSON.parse(new TextDecoder().decode(rawDecoded));
          } catch {
            // Malformed JSON — preserve status/error fields, skip payload
          }
        }

        // Map full BaseStationReceptionInfo per SCACI §3.8.1
        const bsList = m.getBaseStationsList();
        const baseStations = bsList.length > 0
          ? bsList.map((bs) => {
              const sp = bs.getSubpackets();
              return {
                bsEui: bs.getBsEui(),
                rxTime: bs.getRxTime(),
                snr: bs.getSnr(),
                rssi: bs.getRssi(),
                eqSnr: bs.getEqSnr()?.getValue(),
                rxDuration: bs.getRxDuration()?.getValue(),
                profile: bs.getProfile()?.getValue(),
                mode: bs.getMode()?.getValue(),
                dlRxSnr: bs.getDlRxSnr()?.getValue(),
                dlRxRssi: bs.getDlRxRssi()?.getValue(),
                subpackets: sp ? {
                  snr: sp.getSnrList(),
                  rssi: sp.getRssiList(),
                  frequency: sp.getFrequencyList(),
                  phase: sp.getPhaseList().length > 0 ? sp.getPhaseList() : undefined,
                } : undefined,
              };
            })
          : undefined;

        return {
          id: m.getId(),
          epEui: m.getEpeui(),
          bsEui: m.getBseui(),
          payload: m.getPayload_asB64(),
          packetCounter: m.getPacketCounter(),
          rssi: m.getRssi(),
          snr: m.getSnr(),
          eqSnr: m.getEqSnr(),
          receivedAt: m.getReceivedAt()?.toDate() || new Date(),
          decodedPayload,
          decodeStatus: m.getDecodeStatus() || undefined,
          decodeErrorCode: m.getDecodeErrorCode() || undefined,
          duplicate: m.getDuplicate() || undefined,
          baseStations,
        };
      }),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  // ============================================================================
  // API KEY METHODS
  // ============================================================================

  /**
   * List API keys for the current organization
   */
  async listApiKeys(params?: { pageSize?: number; pageToken?: string; userId?: string }): Promise<{
    apiKeys: GrpcApiKey[];
    nextPageToken?: string;
    totalCount: number;
  }> {
    const request = new pb.ListApiKeysRequest();
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.pageToken) request.setPageToken(params.pageToken);
    if (params?.userId) request.setUserId(params.userId);

    const response = await this.promisify<pb.ListApiKeysRequest, pb.ListApiKeysResponse>(
      this.client.listApiKeys
    )(request);

    return {
      apiKeys: response.getApiKeysList().map((k) => ({
        id: k.getId(),
        orgId: k.getOrgId(),
        userId: k.getUserId(),
        name: k.getName(),
        keyPrefix: k.getKeyPrefix(),
        keyType: k.getKeyType(),
        isActive: k.getIsActive(),
        expiresAt: k.getExpiresAt()?.toDate(),
        lastUsedAt: k.getLastUsedAt()?.toDate(),
        createdAt: k.getCreatedAt()?.toDate() || new Date(),
      })),
      nextPageToken: response.getNextPageToken() || undefined,
      totalCount: response.getTotalCount(),
    };
  }

  /**
   * Create a new API key
   * Returns the raw key only on creation (never stored)
   */
  async createApiKey(data: { name: string; keyType: string; expiresAt?: Date }): Promise<{
    apiKey: GrpcApiKey;
    rawKey: string;
  }> {
    const request = new pb.CreateApiKeyRequest();
    request.setName(data.name);
    request.setKeyType(data.keyType);
    if (data.expiresAt) {
      const ts = new google_protobuf_timestamp_pb.Timestamp();
      ts.fromDate(data.expiresAt);
      request.setExpiresAt(ts);
    }

    const response = await this.promisify<pb.CreateApiKeyRequest, pb.CreateApiKeyResponse>(
      this.client.createApiKey
    )(request);

    const apiKey = response.getApiKey();
    if (!apiKey) {
      throw new GrpcApiError(500, GRPC_CLIENT_ERRORS.API_KEY_CREATION_FAILED);
    }

    return {
      apiKey: {
        id: apiKey.getId(),
        orgId: apiKey.getOrgId(),
        userId: apiKey.getUserId(),
        name: apiKey.getName(),
        keyPrefix: apiKey.getKeyPrefix(),
        keyType: apiKey.getKeyType(),
        isActive: apiKey.getIsActive(),
        expiresAt: apiKey.getExpiresAt()?.toDate(),
        createdAt: apiKey.getCreatedAt()?.toDate() || new Date(),
      },
      rawKey: response.getRawKey(),
    };
  }

  /**
   * Delete an API key
   */
  async deleteApiKey(id: string): Promise<boolean> {
    const request = new pb.DeleteApiKeyRequest();
    request.setId(id);

    const response = await this.promisify<pb.DeleteApiKeyRequest, pb.DeleteApiKeyResponse>(
      this.client.deleteApiKey
    )(request);

    return response.getSuccess();
  }

  // ============================================================================
  // ENDPOINT STATS & OPERATIONS
  // ============================================================================

  /**
   * Get endpoint message statistics
   */
  async getEndpointStats(epEui: string): Promise<{
    epEui: string;
    totalCount: number;
    uniqueEndpoints: number;
    avgRssi: number;
    avgSnr: number;
    firstSeen?: Date;
    lastSeen?: Date;
    activeDays: number;
    attachStatus: string;
  } | null> {
    const request = new pb.GetEndPointStatsRequest();
    request.setEpEui(epEui);

    try {
      const response = await this.promisify<
        pb.GetEndPointStatsRequest,
        pb.GetEndPointStatsResponse
      >(this.client.getEndPointStats)(request);

      return {
        epEui: response.getEpEui(),
        totalCount: response.getTotalCount(),
        uniqueEndpoints: response.getUniqueEndpoints(),
        avgRssi: response.getAvgRssi(),
        avgSnr: response.getAvgSnr(),
        firstSeen: response.getFirstSeen()?.toDate(),
        lastSeen: response.getLastSeen()?.toDate(),
        activeDays: response.getActiveDays(),
        attachStatus: response.getAttachStatus(),
      };
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Get endpoint operations history
   */
  async getEndpointOperations(
    epEui: string,
    params?: { pageSize?: number; offset?: number }
  ): Promise<{
    operations: Array<{
      id: string;
      eventType: string;
      category: string;
      severity: string;
      title: string;
      createdAt: Date;
    }>;
  }> {
    const request = new pb.GetEndPointOperationsRequest();
    request.setEpEui(epEui);
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.offset) request.setOffset(params.offset);

    const response = await this.promisify<
      pb.GetEndPointOperationsRequest,
      pb.GetEndPointOperationsResponse
    >(this.client.getEndPointOperations)(request);

    return {
      operations: response.getOperationsList().map((op) => ({
        id: op.getId(),
        eventType: op.getEventType(),
        category: op.getCategory(),
        severity: op.getSeverity(),
        title: op.getTitle(),
        createdAt: op.getCreatedAt()?.toDate() || new Date(),
      })),
    };
  }

  // ============================================================================
  // INTEGRATION METHODS
  // ============================================================================

  /**
   * Create integration
   * Integration CRUD for event sink management.
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
    createdAt: Date;
    updatedAt: Date;
  }> {
    const request = new pb.CreateIntegrationRequest();
    request.setName(data.name);
    if (data.description) request.setDescription(data.description);
    request.setType(data.type);
    // Config and eventFilter are google.protobuf.Struct - need special handling
    // For now these will be handled when proto is regenerated
    if (data.deliveryFormat) request.setDeliveryFormat(data.deliveryFormat);

    const response = await this.promisify<pb.CreateIntegrationRequest, pb.Integration>(
      this.client.createIntegration
    )(request);

    return {
      id: response.getId(),
      name: response.getName(),
      description: response.getDescription() || undefined,
      type: response.getType(),
      deliveryFormat: response.getDeliveryFormat(),
      status: response.getStatus(),
      createdAt: response.getCreatedAt()?.toDate() || new Date(),
      updatedAt: response.getUpdatedAt()?.toDate() || new Date(),
    };
  }

  /**
   * Get integration by ID
   * Integration CRUD for event sink management.
   */
  async getIntegration(id: number): Promise<{
    id: number;
    name: string;
    description?: string;
    type: string;
    deliveryFormat: string;
    status: string;
    createdAt: Date;
    updatedAt: Date;
  } | null> {
    const request = new pb.GetIntegrationRequest();
    request.setId(id);

    try {
      const response = await this.promisify<pb.GetIntegrationRequest, pb.Integration>(
        this.client.getIntegration
      )(request);

      return {
        id: response.getId(),
        name: response.getName(),
        description: response.getDescription() || undefined,
        type: response.getType(),
        deliveryFormat: response.getDeliveryFormat(),
        status: response.getStatus(),
        createdAt: response.getCreatedAt()?.toDate() || new Date(),
        updatedAt: response.getUpdatedAt()?.toDate() || new Date(),
      };
    } catch (error) {
      if (error instanceof GrpcApiError && error.isNotFound()) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Update integration
   * Integration CRUD for event sink management.
   */
  async updateIntegration(
    id: number,
    data: {
      name?: string;
      description?: string;
      status?: string;
    }
  ): Promise<{
    id: number;
    name: string;
    description?: string;
    type: string;
    deliveryFormat: string;
    status: string;
    createdAt: Date;
    updatedAt: Date;
  }> {
    const request = new pb.UpdateIntegrationRequest();
    request.setId(id);
    if (data.name) request.setName(data.name);
    if (data.description !== undefined) request.setDescription(data.description);
    if (data.status) request.setStatus(data.status);

    const response = await this.promisify<pb.UpdateIntegrationRequest, pb.Integration>(
      this.client.updateIntegration
    )(request);

    return {
      id: response.getId(),
      name: response.getName(),
      description: response.getDescription() || undefined,
      type: response.getType(),
      deliveryFormat: response.getDeliveryFormat(),
      status: response.getStatus(),
      createdAt: response.getCreatedAt()?.toDate() || new Date(),
      updatedAt: response.getUpdatedAt()?.toDate() || new Date(),
    };
  }

  /**
   * Delete integration
   * Integration CRUD for event sink management.
   */
  async deleteIntegration(id: number): Promise<void> {
    const request = new pb.DeleteIntegrationRequest();
    request.setId(id);

    await this.promisify<pb.DeleteIntegrationRequest, Empty>(this.client.deleteIntegration)(
      request
    );
  }

  /**
   * List integrations
   * Integration CRUD for event sink management.
   */
  async listIntegrations(params?: { pageSize?: number; offset?: number }): Promise<{
    integrations: Array<{
      id: number;
      name: string;
      description?: string;
      type: string;
      deliveryFormat: string;
      status: string;
      createdAt: Date;
      updatedAt: Date;
    }>;
    totalCount: number;
  }> {
    const request = new pb.ListIntegrationsRequest();
    if (params?.pageSize) request.setPageSize(params.pageSize);
    if (params?.offset) request.setOffset(params.offset);

    const response = await this.promisify<pb.ListIntegrationsRequest, pb.ListIntegrationsResponse>(
      this.client.listIntegrations
    )(request);

    return {
      integrations: response.getIntegrationsList().map((i) => ({
        id: i.getId(),
        name: i.getName(),
        description: i.getDescription() || undefined,
        type: i.getType(),
        deliveryFormat: i.getDeliveryFormat(),
        status: i.getStatus(),
        createdAt: i.getCreatedAt()?.toDate() || new Date(),
        updatedAt: i.getUpdatedAt()?.toDate() || new Date(),
      })),
      totalCount: response.getTotalCount(),
    };
  }

  // ---------------------------------------------------------------------------
  // Global coverage (server admin only)
  // ---------------------------------------------------------------------------

  async listAllBaseStationLocations(): Promise<{
    locations: Array<{
      bsEui: string;
      name: string;
      latitude: number;
      longitude: number;
      altitude?: number;
      locationSource: string;
      isOnline: boolean;
      orgId: string;
    }>;
    totalCount: number;
  }> {
    const request = new pb.ListAllBaseStationLocationsRequest();

    const response = await this.promisify<
      pb.ListAllBaseStationLocationsRequest,
      pb.ListAllBaseStationLocationsResponse
    >(this.client.listAllBaseStationLocations, { requireOrgUser: false })(request);

    return {
      locations: response.getLocationsList().map((loc) => ({
        bsEui: loc.getBsEui(),
        name: loc.getName(),
        latitude: loc.getLatitude(),
        longitude: loc.getLongitude(),
        altitude: loc.getAltitude()?.getValue() ?? undefined,
        locationSource: loc.getLocationSource(),
        isOnline: loc.getIsOnline(),
        orgId: loc.getOrgId(),
      })),
      totalCount: response.getTotalCount(),
    };
  }
}

// Export singleton instance
export const grpcClient = new GrpcClientService();
