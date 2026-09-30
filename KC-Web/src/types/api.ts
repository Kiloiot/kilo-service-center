/**
 * API Type Definitions
 *
 * Types derived from gRPC proto definitions. Separates backend API formats
 * from frontend UI formats.
 *
 * Backend types match gRPC response message shapes (kilocenter.proto).
 * UI types provide frontend-friendly formats for components.
 */

import type {
  BaseStationStatus,
  BsConnectionType,
  BsConnectionTypeLabel,
  EndpointActivity,
  EndpointAttachStatus,
  ErrorBucket,
  EventOperation,
  EventOutcome,
  LocationSource,
  TimeRange,
} from "@constants/app";
import { HTTP_STATUS } from "@constants/app";

import type { Tagged } from "./tagged";

// ============================================================================
// API Response Types (Backend format - matches gRPC proto messages)
// ============================================================================

// API Key Types
export interface ApiKeyAPI {
  id: string;
  orgId: string;
  userId: string;
  name: string;
  keyPrefix: string;
  keyType: string;
  isActive: boolean;
  expiresAt?: string;
  lastUsedAt?: string;
  createdAt: string;
}

export interface ApiKeyListResponse {
  apiKeys: ApiKeyAPI[];
  nextPageToken?: string;
  totalCount: number;
}

export interface ApiKeyCreateResponse {
  apiKey: ApiKeyAPI;
  rawKey: string;
}

// Base Station API Types
export interface BaseStationAPI {
  eui?: string; // Present in detail responses
  bsEui?: string; // Present in list responses (snake_case from backend)
  name: string;
  isOnline: boolean;
  lastSeen: string;
  firstSeen: string;
  // Detail fields (present on detail API responses)
  connectionType?: string;
  serviceCenterUrl?: string;
  // Legacy snake_case variants for backwards compatibility
  is_online?: boolean;
  first_seen?: string;
  last_seen?: string;
  last_seen_at?: string;
  created_at?: string;
  connection_type?: string;
  service_center_url?: string;
  // Location fields (available in both list and detail)
  latitude?: number;
  longitude?: number;
  altitude?: number;
  locationSource?: string;
  locationUpdatedAt?: string;
  certificateExpiresAt?: string;
  tlsCertFingerprint?: string;
  // Last completed BSSCI connect handshake
  sessionStartedAt?: string;
}

/** Outcome of an on-demand status request (RequestBaseStationStatus). */
export interface BaseStationStatusRequestResult {
  success: boolean;
  message: string;
  opId?: string;
}

/** A base station with coordinates, as ListAllBaseStationLocations reports it. */
export interface BaseStationLocationDTO {
  bsEui: string;
  name: string;
  latitude: number;
  longitude: number;
  altitude?: number;
  locationSource: string;
  isOnline: boolean;
  orgId: string;
}

export interface BaseStationDetailAPI extends BaseStationAPI {
  description?: string;
  connectionType: BsConnectionType;
  serviceCenterUrl?: string;
  version?: string;
  latitude?: number;
  longitude?: number;
  altitude?: number;
  locationSource?: string;
  locationUpdatedAt?: string;
  sessionUuid?: string;
  createdAt: string;
  updatedAt: string;
  // MIOTY status metrics per BSSCI v1.0.0 §3.5.2
  systemTime?: number;
  dutyCycle?: number;
  uptimeSeconds?: number;
  temperatureCelsius?: number;
  cpuLoad?: number;
  memoryLoad?: number;
  bsConfig?: Record<string, unknown>;
  lastStatusAt?: string;
}

// Base Station Reception info per SCACI §3.8.1
export interface BaseStationReceptionAPI {
  bsEui: string;
  rxTime: string; // Unix nanoseconds, carried as a string (int64 exceeds Number.MAX_SAFE_INTEGER)
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
}

export interface BaseStationMessagesFilter {
  startTime?: string;
  endTime?: string;
  messageTypes?: string[];
  direction?: "uplink" | "downlink";
  epEui?: string;
  minRssi?: number;
  maxRssi?: number;
  minSnr?: number;
  maxSnr?: number;
}

/** A date range narrowing a base station's or an endpoint's activity feed. */
export interface ActivityFilter {
  startTime?: string;
  endTime?: string;
}

/** The payload of each kind of Activity row (ACTIVITY_KIND). */
export interface ActivityRowPayload {
  event: { entry: EventLogEntryUI };
  uplink: { uplink: UplinkUI };
}

export type ActivityKind = keyof ActivityRowPayload;

/** One row of a device's Activity table: a system event or a stored uplink. */
export type ActivityRow<K extends ActivityKind = ActivityKind> = Tagged<
  ActivityRowPayload,
  K
>;

/** One page of a device's activity feed and the cursor of the next page. */
export interface ActivityPage {
  items: ActivityRow[];
  nextPageToken?: string;
  totalCount: number;
}

// Endpoint API Types
// Uses epEui as primary identifier per gRPC contract
export interface EndpointAPI {
  epEui: string; // Primary identifier (was numeric id)
  name?: string;
  lastSeen?: string;
  createdAt: string;
  status: EndpointActivity;
  batteryLevel?: number;
  shAddr?: number;
  bidi?: boolean; // Derived from ep_class: 'A' = true, 'Z' = false
  // MIOTY configuration fields per BSSCI v1.0.0 §3.8.1
  preAttach?: boolean;
  carrierOffset?: number;
  /** A network session key is stored; reads never carry the key. */
  nwkSnKeySet?: boolean;
  /** An application key is stored; reads never carry the key. */
  appKeySet?: boolean;
  dualChan?: boolean;
  repetition?: boolean;
  wideCarrOff?: boolean;
  longBlkDist?: boolean;
  // SCACI §3.6.1 fields
  attachCnt?: number; // Attachment counter
  lastPacketCnt?: number; // Packet counter (last received)
  // Attach status from ep_status column (replaces propagated/propagatedAt/propagationCount)
  attachStatus?: EndpointAttachStatus;
  // Base stations hold an earlier attach propagate profile (BSSCI §3.8.1)
  reattachPending?: boolean;
  // Type EUI (8-byte device type identifier)
  typeEui?: string;
  // Blueprint device model association
  deviceModelId?: string;
  // Latest reception and the station serving the endpoint (detail read only)
  lastRssi?: number;
  lastSnr?: number;
  lastEqSnr?: number;
  servingBsEui?: string;
}

// CreateEndpointRequest - all MIOTY protocol fields required per BSSCI §3.8.1, SCACI §3.6.1
// No auto-derivation - values must be provided from device provisioning.
export interface CreateEndpointRequest {
  epEui: string; // Required: 16-char hex
  name: string; // Required
  shAddr: number; // Required: 1-65535 (4 hex chars, nonzero)
  nwkSnKey: string; // Required: 32-char hex
  // Radio flags: explicit boolean (unchecked=false, not omitted)
  bidi: boolean;
  preAttach: boolean;
  dualChan: boolean;
  repetition: boolean;
  wideCarrOff: boolean;
  longBlkDist: boolean;
  // Counter fields (required per BSSCI §3.8.1, SCACI §3.6.1)
  lastPacketCnt: number; // Required: 0-4294967295
  attachCnt: number; // Required: 0-4294967295
  // Optional fields (validated when provided)
  appKey?: string; // 32-char hex if provided
  typeEui?: string; // 16-char hex if provided
  carrierOffset?: number; // Hz
  deviceModelId?: string; // UUID reference to device_models.id
}

// UpdateEndpointRequest - partial update (PATCH semantics)
// All fields optional - only provided fields are updated
export interface UpdateEndpointRequest {
  name?: string;
  shAddr?: number; // Must be nonzero if provided (1-65535)
  nwkSnKey?: string; // 32-char hex, non-zero bytes
  appKey?: string | null; // 32-char hex sets it, null removes the stored key
  bidi?: boolean;
  preAttach?: boolean;
  dualChan?: boolean;
  repetition?: boolean;
  wideCarrOff?: boolean;
  longBlkDist?: boolean;
  lastPacketCnt?: number; // 0-4294967295
  attachCnt?: number; // 0-4294967295
  carrierOffset?: number; // Hz
  typeEui?: string | null; // null = explicit clear, string = set, undefined = no change
  deviceModelId?: string | null; // null = explicit clear, string = set, undefined = no change
  newEpEui?: string; // 16-char hex — triggers EUI cascade if different from current
}

export interface GenerateCertificateRequest {
  bsEui: string;
  validityDays: number;
  organizationName?: string;
  countryCode?: string;
}

/** One server-side certificate as the Certificates page shows it. */
export interface CertificateSummary {
  subject: string;
  issuer: string;
  notBefore: Date;
  notAfter: Date;
  daysUntilExpiry: number;
  isValid: boolean;
}

/** The service center's certificates and the names a renewal issues. */
export interface ServerCertificateStatus {
  serverCert?: CertificateSummary;
  caCert?: CertificateSummary;
  renewalNames: string[];
}

export interface GenerateCertificateResponse {
  bsEui: string;
  serviceCenterUrl: string;
  downloadUrls: {
    caCert: string;
    clientCert: string;
    privateKey: string; // Backend returns privateKey, not clientKey
  };
  expiresAt: string; // Backend returns expiresAt, not expiryDate
}

// Analytics API Types
export interface AnalyticsOverviewAPI {
  totalMessages: number;
  activeEndpoints: number;
  activeBaseStations: number;
  messagesLast24h: number;
  messagesLastWeek: number;
  topEndpoints?: Array<{
    epEui: string;
    name?: string;
    messageCount: number;
  }>;
}

// ============================================================================
// Frontend UI Types (Consistent format for components)
// ============================================================================

export interface BaseStationUI {
  id: string;
  eui: string;
  name?: string;
  status: BaseStationStatus;
  connectionType: BsConnectionTypeLabel;
  createdAt: string;
  lastSeen: string;
  lastHandshake?: string;
  serviceCenterUrl: string;
  certificateExpiryDate?: string;
  certificateFingerprint?: string;
  version?: string;
  // Geolocation
  latitude?: number;
  longitude?: number;
  altitude?: number;
  locationSource?: LocationSource;
  locationUpdatedAt?: string;
  // MIOTY status metrics (available in detail view) per BSSCI v1.0.0 §3.5.2
  systemTime?: number;
  dutyCycle?: number;
  uptimeSeconds?: number;
  temperatureCelsius?: number;
  cpuLoad?: number;
  memoryLoad?: number;
  bsConfig?: Record<string, unknown>;
  lastStatusAt?: string;
}

/** A base station pin on the dashboard map. */
export interface BaseStationLocationUI {
  eui: string;
  name?: string;
  latitude: number;
  longitude: number;
  status: BaseStationUI["status"];
}

// Uses epEui as primary identifier per gRPC contract
export interface EndpointUI {
  id: string; // Backward compat alias for epEui (used as key in lists/grids)
  epEui: string; // Primary identifier (was numeric id)
  name?: string;
  status: EndpointActivity;
  batteryLevel?: number;
  lastSeen?: string;
  createdAt: string; // Required for weekly addition tracking (matches EndpointAPI)
  // SCACI §3.6.1 fields for display
  attachCnt?: number;
  lastPacketCnt?: number;
  // Attach status from ep_status column (replaces propagated/propagatedAt/propagationCount)
  attachStatus: EndpointAttachStatus;
  reattachPending?: boolean;
  // MIOTY configuration fields per BSSCI v1.0.0 §3.8.1
  shAddr?: number;
  bidi?: boolean; // Derived from ep_class: 'A' = true, 'Z' = false
  preAttach?: boolean;
  carrierOffset?: number;
  /** A network session key is stored; reads never carry the key. */
  nwkSnKeySet?: boolean;
  /** An application key is stored; reads never carry the key. */
  appKeySet?: boolean;
  dualChan?: boolean;
  repetition?: boolean;
  wideCarrOff?: boolean;
  longBlkDist?: boolean;
  /** Set when the uplink was received by a base station belonging to a foreign tenant. */
  ownerTenantId?: number;
  /** True when the endpoint is currently served by a non-home tenant's infrastructure. */
  isRoaming?: boolean;
  // Type EUI (8-byte device type identifier)
  typeEui?: string;
  // Blueprint device model association
  deviceModelId?: string;
  // Latest reception and the station serving the endpoint (detail read only)
  lastRssi?: number;
  lastSnr?: number;
  lastEqSnr?: number;
  servingBsEui?: string;
}

// ============================================================================
// Error Types
// ============================================================================

/**
 * Predicate for 401-shaped errors across transports.
 * Accepts ApiError, GrpcApiError, or any duck-typed error exposing
 * status === 401 or an isUnauthorized() method. Keeps callers free
 * of transport-specific imports.
 */
/** Errors the transport raises: an HTTP-like status plus classifier methods. */
export interface ApiErrorLike extends Error {
  readonly status: number;
  readonly code?: string;
  readonly token?: string;
  isNotFound(): boolean;
  isUnauthorized(): boolean;
  isForbidden(): boolean;
  isAlreadyExists(): boolean;
  isInvalidArgument(): boolean;
}

export function isApiError(err: unknown): err is ApiErrorLike {
  if (!(err instanceof Error)) return false;
  const candidate = err as Partial<ApiErrorLike>;
  return (
    typeof candidate.status === "number" &&
    typeof candidate.isNotFound === "function" &&
    typeof candidate.isForbidden === "function"
  );
}

export function isUnauthorizedError(err: unknown): boolean {
  if (err === null || typeof err !== "object") {
    return false;
  }
  const candidate = err as {
    status?: unknown;
    isUnauthorized?: () => boolean;
  };
  if (typeof candidate.isUnauthorized === "function") {
    try {
      if (candidate.isUnauthorized() === true) {
        return true;
      }
    } catch {
      // fall through to status check
    }
  }
  return candidate.status === HTTP_STATUS.UNAUTHORIZED;
}

// ============================================================================
// SCACI Types (Service Center - Application Center Interface)
// ============================================================================

/**
 * SCACI Downlink Queue DTO per SCACI v1.0.0 §3.12
 * Matches gRPC ScaciDownlinkQueue message in kilocenter.proto
 */
export interface SCACIDownlinkQueueDTO {
  queId: string; // int64 queue id carried as a string (exceeds Number.MAX_SAFE_INTEGER)
  epEui: string; // Hex-encoded endpoint EUI
  payloads: string[]; // Hex-encoded userData, one entry per packetCnt when cntDepend
  cntDepend: boolean;
  packetCnt?: number[];
  format: number;
  priority: number;
  responseExp: boolean;
  responsePrio: boolean;
  dlWindReq: boolean;
  expOnly: boolean;
  result?: string;
  txTime?: string; // Unix nanoseconds, carried as a string
  bsEui?: string; // Hex-encoded base station EUI
  createdAt: string; // ISO timestamp
  id?: string; // DB row ID (only in results)
  status?: string; // Queue lifecycle status (pending/queued/transmitted/etc.)
  dlRxStatQry?: boolean;
  scheduledAt?: string;
  transmittedAt?: string;
  transmissionPacketCnt?: number;
  endpointAckedAt?: string; // ISO time the device acknowledged the transmitted downlink (BSSCI §3.10.1 dlAck)
  acceptedAt?: string; // ISO time bsEui accepted the downlink (BSSCI §3.12 dlDataQueRsp)
}

/** The SCACI §3.10.1 dlDataQue fields an operator composes or edits. */
export interface DownlinkContent {
  payloads: string[]; // hex-encoded
  priority: number;
  cntDepend: boolean;
  packetCnt?: number[];
  format: number;
  responseExp: boolean;
  responsePrio: boolean;
  dlWindReq: boolean;
  expOnly: boolean;
  dlRxStatQry: boolean;
}

/** Matches gRPC SendDownlinkRequest fields. */
export interface SendDownlinkRequest extends DownlinkContent {
  epEui: string;
}

/** The fields a pending downlink is rewritten with (UpdatePendingDownlink). */
export interface UpdatePendingDownlinkRequest extends SendDownlinkRequest {
  queId: string;
}

// ============================================================================
// Traffic and Logs Types
// ============================================================================

/** The device a Traffic or Logs view is narrowed to; empty for the tenant-wide view. */
export interface DeviceScope {
  epEui?: string;
  bsEui?: string;
}

/** One server-side page: the rows and the total the filter matches. */
export interface Page<T> {
  items: T[];
  totalCount: number;
}

/** A listing RPC result before it is cut into pages. */
export interface ListResult<T> {
  items: T[];
  nextPageToken?: string;
  totalCount: number;
}

/** ListMessages predicates (SCACI §3.8.1 ulData). */
export interface UplinkFilter extends DeviceScope {
  timeRange: TimeRange;
  duplicate?: boolean;
  dlOpen?: boolean;
  profile?: string;
  mode?: string;
}

/** A stored ulData as ListMessages returns it; userData is hex. */
export interface UplinkAPI {
  id: string;
  opId?: string;
  epEui: string;
  bsEui: string;
  packetCnt: number;
  snr: number;
  rssi: number;
  dlOpen: boolean;
  responseExp: boolean;
  dlAck: boolean;
  duplicate: boolean;
  userData: string;
  format?: number; // SCACI §3.8.1 userData format identifier
  receptions: BaseStationReceptionAPI[];
  /** DECODE_STATUS of the blueprint decode of the payload. */
  decodeStatus: string;
  /** The blueprint error token of a failed decode. */
  decodeErrorCode?: string;
  /** The blueprint's output when the Service Center decoded the payload. */
  decodedPayload?: Record<string, unknown>;
}

/** An uplink row: radio fields come from the reception of the base station in view. */
export interface UplinkUI {
  id: string;
  opId?: string;
  epEui: string;
  bsEui: string;
  rxTime?: string; // Unix nanoseconds, carried as a string
  packetCnt: number;
  snr: number;
  rssi: number;
  eqSnr?: number;
  dlOpen: boolean;
  responseExp: boolean;
  dlAck: boolean;
  duplicate: boolean;
  userData: string;
  format?: number; // SCACI §3.8.1 userData format identifier
  receptions: BaseStationReceptionAPI[];
  decodeStatus: string;
  decodeErrorCode?: string;
  decodedPayload?: Record<string, unknown>;
}

/** ListDownlinkQueue predicates (SCACI §3.10.1 dlDataQue); bsEui names the station holding the downlink. */
export interface DownlinkQueueFilter extends DeviceScope {
  status?: string;
  priority?: string;
  queId?: string;
}

/** GetDownlinkResults predicates (SCACI §3.12.1 dlDataRes). */
export interface DownlinkResultFilter extends DeviceScope {
  timeRange: TimeRange;
  result?: string;
  queId?: string;
}

/** ListEvents predicates behind the Events and Audit logs. */
export interface EventLogFilter extends DeviceScope {
  timeRange: TimeRange;
  category?: string;
  operation?: EventOperation;
  outcome?: EventOutcome;
  opId?: string;
  search?: string;
}

/** A system event as ListEvents returns it; data is the projected event details. */
export interface EventRecordAPI {
  id: string;
  eventType: string;
  category: string;
  severity: string;
  title: string;
  description: string;
  sourceName: string;
  userId: string;
  userEmail: string;
  timestamp: Date;
  data?: unknown;
}

export interface EventLogEntryUI {
  id: string;
  timestamp: string;
  eventType: string;
  category: string;
  severity: string;
  title: string;
  description: string;
  /** What the Scope column names: the event's source, else the device its data records. */
  scope?: string;
  userId?: string;
  userEmail?: string;
  opId?: string;
  data?: Record<string, unknown>;
}

/** ListErrorGroups predicates: one bucket over a time range. */
export interface ErrorGroupFilter {
  bucket: ErrorBucket;
  timeRange: TimeRange;
}

export interface ErrorGroupUI {
  eventType: string;
  code: string;
  message: string;
  sourceName: string;
  firstSeen: string;
  lastSeen: string;
  count: number;
  lastOpId?: string;
}

/** Uplink totals of one base station (GetBaseStationMessageStats) or endpoint (GetEndPointStats). */
export interface TrafficSummaryUI {
  totalMessages: number;
  avgRssi: number;
  avgSnr: number;
  firstSeen?: string;
  lastSeen?: string;
  uniqueEndpoints?: number;
  messagesToday?: number;
  messagesThisWeek?: number;
  messagesThisMonth?: number;
  activeDays?: number;
}

/** A DL RX status an end point reported through a base station (BSSCI §3.15). */
export interface DlRxStatusDTO {
  bsEui: string;
  rxTime?: string; // Unix UTC nanoseconds, carried as a string
  packetCnt: number;
  dlRxSnr: number; // dB
  dlRxRssi: number; // dBm
}

export interface DlRxStatusResponse {
  statuses: DlRxStatusDTO[];
  totalCount: number;
}

/** A DL RX status query the service center sent to a base station (BSSCI §3.15). */
export interface DlRxStatusQueryDTO {
  bsEui: string;
  opId?: string;
  status: string;
  requestedAt?: string; // ISO timestamp
}

export interface DlRxStatusQueriesResponse {
  queries: DlRxStatusQueryDTO[];
  pendingCount: number;
}

export interface QueryDlRxStatusResponse {
  queryInitiated: boolean;
  message: string;
}

// ============================================================================
// System Status Types (Service Health Monitoring)
// ============================================================================

/**
 * ServiceStatusAPI represents the health state of a single monitored service.
 * Matches gRPC ServiceStatus message in kilocenter.proto
 */
export interface ServiceStatusAPI {
  name: string;
  url: string;
  healthy: boolean;
  latencyMs: number;
  error?: string;
  checkedAt: string; // ISO timestamp
}

/**
 * SystemStatusResponse aggregates all service health statuses.
 * Matches gRPC SystemStatus message in kilocenter.proto
 */
export interface SystemStatusResponse {
  services: ServiceStatusAPI[];
  healthy: boolean; // Overall system health (all services must be healthy)
  timestamp: string; // ISO timestamp
  error?: string; // Set when status check fails (e.g., no endpoints configured)
  startedAt?: string; // Service Center process start (last restart)
}

// ============================================================================
// Authentication Types (OIDC/OAuth2)
// ============================================================================

/**
 * ProviderSettingsAPI represents a single external auth provider configuration.
 * Matches gRPC ProviderSettings message in kilocenter.proto
 */
export interface ProviderSettingsAPI {
  enabled: boolean;
  login_url: string;
  login_label: string;
  login_redirect: boolean;
  logout_url?: string;
}

/** The rules a new password must meet (gRPC PasswordPolicy in identity.proto). */
export interface PasswordPolicyAPI {
  min_length: number;
  max_length: number;
  requires_letter: boolean;
  requires_digit: boolean;
}

/**
 * AuthSettingsAPI represents combined local and external auth settings.
 * Matches gRPC AuthSettings message in kilocenter.proto
 */
export interface AuthSettingsAPI {
  enabled: boolean;
  local_login_enabled: boolean;
  refresh_token_enabled: boolean;
  registration_enabled: boolean;
  login_url?: string;
  login_label?: string;
  login_redirect?: boolean;
  logout_url?: string;
  // Nested provider settings
  oidc?: ProviderSettingsAPI;
  oauth2?: ProviderSettingsAPI;
  password_policy?: PasswordPolicyAPI;
}

/**
 * UserMembershipAPI represents a user's organization membership.
 * Matches gRPC UserMembership message in kilocenter.proto
 */
export interface UserMembershipAPI {
  orgId: string;
  orgName: string;
  role: string;
  status: string;
  isOrgAdmin: boolean;
  isBaseStationAdmin: boolean;
  isEndpointAdmin: boolean;
}

/**
 * UserRolesAPI are the roles the signed-in user holds in the current
 * organization. Matches gRPC UserRoles message in identity.proto
 */
export interface UserRolesAPI {
  admin: boolean;
  tenantManager: boolean;
  baseStationManager: boolean;
  endpointManager: boolean;
}

/**
 * UserProfileAPI represents the authenticated user's profile.
 * Matches gRPC UserProfile message in kilocenter.proto
 */
export interface UserProfileAPI {
  id: string;
  email: string;
  isAdmin: boolean;
  hasPassword?: boolean;
  memberships: UserMembershipAPI[];
  defaultOrgId?: string;
  firstName?: string;
  lastName?: string;
}

/**
 * AuthTokensAPI represents JWT tokens returned on login.
 * Matches gRPC AuthTokens message in kilocenter.proto
 */
export interface AuthTokensAPI {
  accessToken: string;
  accessExpiresIn: number; // seconds
  refreshToken?: string;
  refreshExpiresIn?: number; // seconds
}

/**
 * LoginResponseAPI represents the response from login endpoints.
 * Matches gRPC LoginResponse message in kilocenter.proto
 */
export interface LoginResponseAPI {
  tokens: AuthTokensAPI;
  user: UserProfileAPI;
}

/**
 * LoginRequest represents the request body for local login.
 */
export interface LoginRequest {
  email: string;
  password: string;
}

/**
 * ExchangeRequest represents the request body for external auth code exchange.
 */
export interface ExchangeRequest {
  code: string;
  state: string;
}

// ============================================================================
// System User Types
// ============================================================================

/**
 * SystemUserAPI represents the API response format for system users.
 * Uses snake_case per backend JSON conventions.
 */
export interface SystemUserAPI {
  id: string;
  email: string;
  is_admin: boolean;
  is_active: boolean;
  is_tenant_manager: boolean;
  is_base_station_manager: boolean;
  is_endpoint_manager: boolean;
  note?: string;
  created_at: string;
  updated_at: string;
}

/**
 * SystemUserUI represents the UI-friendly format for system users.
 * Uses camelCase per frontend conventions.
 * Timestamps kept as ISO strings for consistency with other UI types.
 */
export interface SystemUserUI {
  id: string;
  email: string;
  isAdmin: boolean;
  isActive: boolean;
  isTenantManager: boolean;
  isBaseStationManager: boolean;
  isEndpointManager: boolean;
  note?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * CreateUserRequest represents the request body for creating a new user.
 */
export interface CreateUserRequest {
  email: string;
  password: string;
  note?: string;
  is_admin: boolean;
  is_active: boolean;
  is_tenant_manager: boolean;
  is_base_station_manager: boolean;
  is_endpoint_manager: boolean;
}

/**
 * UpdateUserRequest represents the request body for updating a user.
 */
export interface UpdateUserRequest {
  email?: string;
  note?: string;
  is_admin?: boolean;
  is_active?: boolean;
  is_tenant_manager?: boolean;
  is_base_station_manager?: boolean;
  is_endpoint_manager?: boolean;
}

// ============================================================================
// Organization Types
// ============================================================================

/**
 * OrganizationAPI represents the API response format for organizations.
 * Uses snake_case per backend JSON conventions.
 */
export interface OrganizationAPI {
  id: string;
  tenant_id: number;
  name: string;
  description?: string; // nullable
  state: string;
  tags?: Record<string, string>;
  created_at: string;
  updated_at: string;
}

/**
 * OrganizationUI represents the UI-friendly format for organizations.
 * Uses camelCase per frontend conventions.
 */
export interface OrganizationUI {
  id: string;
  tenantId: number;
  name: string;
  description?: string;
  state: string;
  tags?: Record<string, string>;
  createdAt: string;
  updatedAt: string;
}

/**
 * CreateOrganizationRequest represents the request body for creating an organization.
 * NO tenant_id - backend auto-creates tenant.
 */
export interface CreateOrganizationRequest {
  name: string;
  description?: string;
  tags?: Record<string, string>;
}

/**
 * UpdateOrganizationRequest represents the request body for updating an organization.
 */
export interface UpdateOrganizationRequest {
  name?: string;
  description?: string;
  state?: string;
  tags?: Record<string, string>;
}

/**
 * OrganizationUserAPI represents the API response format for organization members.
 * Uses snake_case per backend JSON conventions.
 * email comes from users table via JOIN query.
 */
export interface OrganizationUserAPI {
  org_id: string;
  user_id: string;
  email: string; // from users table via JOIN
  role: string;
  status: string;
  is_org_admin: boolean;
  is_base_station_admin: boolean;
  is_endpoint_admin: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * OrganizationUserUI represents the UI-friendly format for organization members.
 * Uses camelCase per frontend conventions.
 */
export interface OrganizationUserUI {
  orgId: string;
  userId: string;
  email: string;
  role: string;
  status: string;
  isOrgAdmin: boolean;
  isBaseStationAdmin: boolean;
  isEndpointAdmin: boolean;
  createdAt: string;
  updatedAt: string;
}

/**
 * AddOrgUserRequest represents the request body for adding a user to an organization.
 */
export interface AddOrgUserRequest {
  user_id?: string;
  email?: string;
  role: string;
  is_org_admin?: boolean;
  is_base_station_admin?: boolean;
  is_endpoint_admin?: boolean;
}

/**
 * RegisterAccountRequest represents the request body for self-service registration.
 */
export interface RegisterAccountRequest {
  email: string;
  password: string;
  firstName: string;
  lastName: string;
  companyName: string;
}

/**
 * UpdateOrgUserRequest represents the request body for updating an organization member.
 */
export interface UpdateOrgUserRequest {
  role?: string;
  status?: string;
  is_org_admin?: boolean;
  is_base_station_admin?: boolean;
  is_endpoint_admin?: boolean;
}

// ============================================================================
// Blueprint Feature: Device Catalog and Payload Decoding Types
// ============================================================================

// Which catalog a list query targets: "system" (shared) vs "custom" (tenant-owned).
export type BlueprintScope = "system" | "custom";

/**
 * ManufacturerUI represents the UI-friendly format for manufacturers.
 */
export interface ManufacturerUI {
  id: string;
  tenantId: number;
  name: string;
  website?: string;
  isVerified: boolean;
  isSystem: boolean;
  modelCount: number;
  createdAt: string;
  updatedAt: string;
}

/**
 * CreateManufacturerRequest represents the request body for creating a manufacturer.
 * isSystem is admin-only; the server rejects it for non-admin callers.
 */
export interface CreateManufacturerRequest {
  name: string;
  website?: string;
  isSystem?: boolean;
}

/**
 * UpdateManufacturerRequest represents the request body for updating a manufacturer.
 */
export interface UpdateManufacturerRequest {
  name?: string;
  website?: string;
}

/**
 * DeviceModelUI represents the UI-friendly format for device models.
 */
export interface DeviceModelUI {
  id: string;
  manufacturerId: string;
  tenantId: number;
  name: string;
  code: string;
  typeEui?: string;
  description?: string;
  datasheetUrl?: string;
  isSystem: boolean;
  blueprintCount: number;
  createdAt: string;
  updatedAt: string;
}

/**
 * CreateDeviceModelWithBlueprintRequest creates a model + default blueprint atomically.
 * Code and type_eui are auto-generated server-side.
 * isSystem is admin-only; the server rejects it for non-admin callers.
 */
export interface CreateDeviceModelWithBlueprintRequest {
  manufacturerId: string;
  name: string;
  version: string;
  specJson: object;
  isSystem?: boolean;
}

/**
 * UpdateDeviceModelRequest represents the request body for updating a device model.
 */
export interface UpdateDeviceModelRequest {
  name?: string;
  description?: string;
  datasheetUrl?: string;
}

/**
 * BlueprintUI represents the UI-friendly format for blueprints.
 */
export interface BlueprintUI {
  id: string;
  deviceModelId: string;
  tenantId: number;
  version: string;
  typeEui: string;
  specJson: object;
  isDefault: boolean;
  isSystem: boolean;
  registryRepo?: string;
  registryCommitSha?: string;
  registryVerified: boolean;
  registryPrUrl?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * BlueprintListItemAPI represents a summary item in blueprint lists.
 */
export interface BlueprintListItemAPI {
  id: string;
  deviceModelId: string;
  version: string;
  typeEui: string;
  isDefault: boolean;
  isSystem: boolean;
  createdAt: string;
}

/**
 * CreateBlueprintRequest represents the request body for creating a blueprint.
 * isSystem is admin-only; the server rejects it for non-admin callers.
 */
export interface CreateBlueprintRequest {
  version: string;
  typeEui?: string; // 16-char hex (8 bytes), optional for models without Type EUI
  specJson: object; // Blueprint specification JSON
  isSystem?: boolean;
}

/**
 * UpdateBlueprintRequest represents the request body for updating a blueprint.
 */
export interface UpdateBlueprintRequest {
  version?: string;
  specJson?: object;
}

export interface BulkAssignBlueprintResponse {
  affectedCount: number;
}

/**
 * DecodePreviewRequest represents the request body for decode preview.
 */
export interface DecodePreviewRequest {
  userData: string; // Hex-encoded payload (or base64 for copied endpoint payloads)
  formatId?: number;
}

/**
 * DecodePreviewResponse represents the response from decode preview.
 */
export interface DecodePreviewResponse {
  success: boolean;
  decodedData?: Record<string, unknown>;
  errorCode?: string;
  errorDetail?: string;
  formatId: number;
}

/**
 * RegistrySubmitRequest represents the request body for submitting to registry.
 */
export interface RegistrySubmitRequest {
  contributorName?: string;
  contributorEmail?: string;
  description?: string;
}

/**
 * RegistrySubmitResponse represents the response from registry submission.
 */
export interface RegistrySubmitResponse {
  prUrl: string;
  commitSha: string;
  branch: string;
}

export interface RevokeDownlinkResponse {
  status: string;
  message: string;
}

/** Outcome of revoking every revocable downlink a queue filter matches. */
export interface DownlinkFlushResult {
  revoked: number;
  failed: number;
}

export interface SendDownlinkResponse {
  id: string;
  status: string;
}
