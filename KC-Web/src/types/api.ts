/**
 * API Type Definitions
 *
 * Types derived from gRPC proto definitions. Separates backend API formats
 * from frontend UI formats.
 *
 * Backend types match gRPC response message shapes (kilocenter.proto).
 * UI types provide frontend-friendly formats for components.
 */

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

export interface CreateApiKeyRequest {
  name: string;
  keyType: string;
  expiresAt?: string;
}

// Base Station API Types
export interface BaseStationAPI {
  eui?: string; // Present in detail responses
  bsEui?: string; // Present in list responses (snake_case from backend)
  name: string;
  isOnline: boolean;
  messageCount: number;
  lastSeen: string;
  firstSeen: string;
  uniqueEndpoints: number;
  avgRssi: number;
  avgSnr: number;
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
}

export interface BaseStationResponse {
  baseStations: BaseStationAPI[];
  page: number;
  pageSize: number;
  totalCount: number;
  totalPages: number;
}

export interface BaseStationDetailAPI extends BaseStationAPI {
  description?: string;
  connectionType: "bssci" | "mqtt";
  serviceCenterUrl?: string;
  version?: string;
  latitude?: number;
  longitude?: number;
  altitude?: number;
  locationSource?: string;
  locationUpdatedAt?: string;
  sessionUuid?: string;
  sessionStartedAt?: string;
  statusCode: number;
  statusMessage?: string;
  tlsCertExpiresAt?: string;
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
}

// Base Station Messages API Types
export interface BaseStationMessageAPI {
  id: string;
  receivedAt: string;
  messageType: string; // "ulData", "attach", "detach", etc.
  direction: "uplink" | "downlink";
  epEui: string;
  bsEui: string;
  packetCnt: number;
  snr: number;
  rssi: number;
  userData?: string; // Base64 encoded
  dlOpen?: boolean;
  responseExp?: boolean;
  // Radio metrics
  eqSnr?: number;
  rxDuration?: number;
  frequency?: number;
  // Multi-BS support per SCACI §3.8.1
  baseStations?: BaseStationReceptionAPI[];
  duplicate?: boolean;
  // Endpoint info
  endpointName?: string;
  /** Set when the uplink was received by a base station belonging to a foreign tenant. */
  ownerTenantId?: number;
  /** True when the endpoint is currently served by a non-home tenant's infrastructure. */
  isRoaming?: boolean;
  // Blueprint decode results
  decodedPayload?: Record<string, unknown>;
  decodeStatus?: string;
  decodeErrorCode?: string;
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

export interface BaseStationMessagesPage {
  messages: BaseStationMessageAPI[];
  totalCount: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface BaseStationMessagesStats {
  totalMessages: number;
  uplinkMessages: number;
  downlinkMessages: number;
  uniqueEndpoints: number;
  averageRssi: number;
  averageSnr: number;
  messagesLastHour: number;
  messagesToday: number;
  messagesThisWeek: number;
  messagesThisMonth: number;
  messagesByType: Record<string, number>;
  topEndpoints: Array<{
    epEui: string;
    endpointName?: string;
    messageCount: number;
    lastSeen: string;
    averageRssi: number;
    averageSnr: number;
  }>;
}

// Unified Activity Feed Types
export interface ActivityEventItem {
  id: string;
  eventType: string;
  category: string;
  severity: string;
  title: string;
  description: string;
  timestamp: Date;
  sourceName?: string;
}

export interface ActivityItem {
  type: "event" | "message";
  occurredAt: Date;
  event?: ActivityEventItem;
  message?: BaseStationMessageAPI;
}

export interface BaseStationActivityFilter {
  startTime?: string;
  endTime?: string;
}

export interface BaseStationActivityPage {
  items: ActivityItem[];
  nextPageToken?: string;
  totalCount: number;
}

export interface EndpointActivityPage {
  items: ActivityItem[];
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
  status: "active" | "inactive";
  batteryLevel?: number;
  shAddr?: number;
  bidi?: boolean; // Derived from ep_class: 'A' = true, 'Z' = false
  // MIOTY configuration fields per BSSCI v1.0.0 §3.8.1
  preAttach?: boolean;
  carrierOffset?: number;
  nwkSnKey?: string;
  appKey?: string;
  dualChan?: boolean;
  repetition?: boolean;
  wideCarrOff?: boolean;
  longBlkDist?: boolean;
  // SCACI §3.6.1 fields
  attachCnt?: number; // Attachment counter
  lastPacketCnt?: number; // Packet counter (last received)
  // Attach status from ep_status column (replaces propagated/propagatedAt/propagationCount)
  attachStatus?: "attached" | "detached" | "attaching" | "pending" | "unknown";
  // Type EUI (8-byte device type identifier)
  typeEui?: string;
  // Blueprint device model association
  deviceModelId?: string;
}

// Endpoint V2 API Types (includes roaming support)
export interface EndpointAPIV2 extends EndpointAPI {
  ownerTenantId?: number;
  isRoaming: boolean;
}

export interface EndpointsResponse {
  endpoints: EndpointAPI[];
  page: number;
  pageSize: number;
  totalCount: number;
  totalPages: number;
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
  appKey?: string; // 32-char hex if provided
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

// Event API Types
export interface EventAPI {
  id: string;
  timestamp: string;
  createdAt: string;
  category: string;
  event_type: string; // Snake case to match backend
  title: string;
  message?: string;
  description?: string;
  source_name?: string;
  severity: "success" | "warning" | "error" | "info"; // Backend uses severity, not status
  metadata?: Record<string, unknown>;
}

export interface EventsResponse {
  events: EventAPI[];
  page?: number;
  pageSize?: number;
  totalCount?: number;
}

// BSSCI Event Types (matches system_events table structure)
export interface BSSCIEventAPI {
  id: string;
  timestamp: string;
  event_type: string; // Snake case to match backend
  category: string; // Category like 'bssci', 'system', etc.
  severity: string; // 'info', 'warning', 'error', 'critical'
  source_name?: string; // Source identifier
  title: string; // Event title
  description?: string; // Event description
  data?: Record<string, unknown>; // Additional event data
}

export interface BSSCIEventsResponse {
  events: BSSCIEventAPI[];
  page: number;
  page_size: number;
  total_count: number;
  total_pages: number;
}

// Certificate API Types
export interface CertificateAPI {
  type: string;
  name: string;
  path: string;
  expiryDate: string;
  issuer: string;
  subject: string;
  serialNumber: string;
}

export interface CertificateStatusResponse {
  certificates: CertificateAPI[];
  status: "ok" | "warning" | "error";
  message?: string;
}

export interface GenerateCertificateRequest {
  bsEui: string;
  validityDays: number;
  organizationName?: string;
  countryCode?: string;
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

// Alert API Types
export interface AlertSummaryAPI {
  critical: number;
  warning: number;
  info: number;
  recent: Array<{
    id: string;
    level: "critical" | "warning" | "info";
    message: string;
    timestamp: string;
  }>;
}

// ============================================================================
// Frontend UI Types (Consistent format for components)
// ============================================================================

export interface BaseStationUI {
  id: string;
  eui: string;
  name?: string;
  status: "online" | "offline";
  connectionType: "BSSCI" | "MQTT";
  createdAt: string;
  lastSeen: string;
  serviceCenterUrl: string;
  certificateExpiryDate?: string;
  version?: string;
  // Geolocation
  latitude?: number;
  longitude?: number;
  altitude?: number;
  locationSource?: "gps" | "manual";
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

// Uses epEui as primary identifier per gRPC contract
export interface EndpointUI {
  id: string; // Backward compat alias for epEui (used as key in lists/grids)
  epEui: string; // Primary identifier (was numeric id)
  name?: string;
  status: "active" | "inactive";
  batteryLevel?: number;
  lastSeen?: string;
  createdAt: string; // Required for weekly addition tracking (matches EndpointAPI)
  // SCACI §3.6.1 fields for display
  attachCnt?: number;
  lastPacketCnt?: number;
  // Attach status from ep_status column (replaces propagated/propagatedAt/propagationCount)
  attachStatus: "attached" | "detached" | "attaching" | "pending" | "unknown";
  // MIOTY configuration fields per BSSCI v1.0.0 §3.8.1
  shAddr?: number;
  bidi?: boolean; // Derived from ep_class: 'A' = true, 'Z' = false
  preAttach?: boolean;
  carrierOffset?: number;
  nwkSnKey?: string;
  appKey?: string;
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
}

export interface EventUI {
  id: string;
  type: "success" | "warning" | "error" | "info";
  severity: "success" | "warning" | "error" | "info";
  message: string;
  time: string; // Relative time like "2 hours ago"
  timestamp: string; // ISO timestamp for date grouping
  title?: string; // Event title for display
  category?: string;
  eventType?: string; // Backend event_type (e.g. service.started)
  sourceName?: string; // Service or component that emitted the event
  metadata?: Record<string, unknown>; // Extra data from backend
}

export interface CertificateUI {
  name: string;
  status: "valid" | "expiring" | "expired";
  expiryDate: string;
  daysUntilExpiry: number;
  issuer: string;
  subject: string;
}

// ============================================================================
// Error Types
// ============================================================================

export class ApiError extends Error {
  public status: number;
  public code?: string;
  public token?: string;
  public details?: unknown;

  constructor(
    status: number,
    message: string,
    details?: unknown,
    code?: string,
    token?: string,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.token = token;
    this.details = details;
  }

  isUnauthorized(): boolean {
    return this.status === 401;
  }

  isForbidden(): boolean {
    return this.status === 403;
  }

  isNotFound(): boolean {
    return this.status === 404;
  }

  isServerError(): boolean {
    return this.status >= 500;
  }

  isNetworkError(): boolean {
    return this.status === 0;
  }
}

// ============================================================================
// SCACI Types (Service Center - Application Center Interface)
// ============================================================================

/**
 * SCACI Session Summary per SCACI v1.0.0 §3.3
 * Matches gRPC ScaciSession message in kilocenter.proto
 */
export interface SCACISessionSummary {
  id: number;
  acEui: string; // Hex-encoded Application Center EUI
  snAcUuid: string; // Hex-encoded AC session UUID
  snScUuid: string; // Hex-encoded SC session UUID
  status: string; // active, disconnected, terminated
  lastOpIdAc: number; // Last AC operation ID
  lastOpIdSc: number; // Last SC operation ID
  canResume: boolean; // Whether session can be resumed
  connectedAt: string; // ISO timestamp
  lastHeartbeat?: string; // ISO timestamp
  sessionDuration: number; // Duration in seconds
  tlsVersion: string; // TLS version used
  cipherSuite: string; // TLS cipher suite
}

/**
 * SCACI Operation DTO per SCACI v1.0.0 §3.4-3.14
 * Matches gRPC ScaciOperation message in kilocenter.proto
 */
export interface SCACIOperationDTO {
  opId: number;
  command: string; // ulData, dlDataQue, etc.
  direction: string; // inbound or outbound
  state: string; // pending, acknowledged, completed, failed
  initiatedAt: string; // ISO timestamp
  acknowledgedAt?: string; // ISO timestamp
  completedAt?: string; // ISO timestamp
  duration: number; // Duration in seconds
  errorMessage?: string;
}

/**
 * SCACI Error DTO per SCACI v1.0.0 §3.14
 * Matches gRPC ScaciError message in kilocenter.proto
 */
export interface SCACIErrorDTO {
  id: string;
  eventType: string;
  category: string;
  severity: string;
  title: string;
  description: string;
  sessionId?: number;
  opId?: number;
  command?: string;
  errorCode?: string;
  errorMsg: string;
  occurredAt: string; // ISO timestamp
}

/**
 * SCACI Downlink Queue DTO per SCACI v1.0.0 §3.12
 * Matches gRPC ScaciDownlinkQueue message in kilocenter.proto
 */
export interface SCACIDownlinkQueueDTO {
  queId: number;
  epEui: string; // Hex-encoded endpoint EUI
  payload: string; // Base64-encoded payload
  cntDepend: boolean;
  packetCnt?: number[];
  format: number;
  priority: number;
  responseExp: boolean;
  responsePrio: boolean;
  dlWindReq: boolean;
  expOnly: boolean;
  result?: string;
  txTime?: number; // Unix nanoseconds
  bsEui?: string; // Hex-encoded base station EUI
  createdAt: string; // ISO timestamp
  id?: string; // DB row ID (only in results)
  status?: string; // Queue lifecycle status (pending/queued/transmitted/etc.)
  dlRxStatQry?: boolean;
  scheduledAt?: string;
  transmittedAt?: string;
  transmissionPacketCnt?: number;
}

/**
 * Downlink send request DTO
 * Matches gRPC SendDownlinkRequest fields
 */
export interface SendDownlinkRequest {
  epEui: string;
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

export interface SendDownlinkResponse {
  id: string;
  status: string;
}

export interface RevokeDownlinkResponse {
  status: string;
  message: string;
}

export interface DownlinkQueueResponse {
  messages: SCACIDownlinkQueueDTO[];
  nextPageToken?: string;
  totalCount: number;
}

export interface DownlinkResultsResponse {
  results: SCACIDownlinkQueueDTO[];
  nextPageToken?: string;
  totalCount: number;
}

/**
 * SCACI Statistics per SCACI v1.0.0 aggregated metrics
 * Matches gRPC ScaciStatistics message in kilocenter.proto
 */
export interface SCACIStatistics {
  activeSessions: number;
  resumableSessions: number;
  totalOperations: number;
  ulDataForwarded: number;
  dlQueued: number;
  errorRate: number;
  operationSummary?: {
    pending: number;
    acknowledged: number;
    completed: number;
    failed: number;
  };
}

/**
 * SCACI Session Detail Response per SCACI v1.0.0 §3.3
 * Matches gRPC GetScaciSessionResponse message in kilocenter.proto
 */
export interface SCACISessionDetailResponse {
  session: SCACISessionSummary;
  recentOperations: SCACIOperationDTO[];
  pendingOperations: SCACIOperationDTO[];
  recentErrors: SCACIErrorDTO[];
}

/**
 * SCACI Session List Response (paginated)
 * Matches gRPC ListScaciSessionsResponse message in kilocenter.proto
 */
export interface SCACISessionListResponse {
  sessions: SCACISessionSummary[];
  total: number;
  limit: number;
  offset: number;
}

/**
 * SCACI Error List Response (paginated)
 * Matches gRPC ListScaciErrorsResponse message in kilocenter.proto
 */
export interface SCACIErrorListResponse {
  errors: SCACIErrorDTO[];
  total: number;
  limit: number;
  offset: number;
}

/**
 * SCACI Queue List Response (paginated)
 * Matches gRPC ListScaciQueuesResponse message in kilocenter.proto
 */
export interface SCACIQueueListResponse {
  queue: SCACIDownlinkQueueDTO[];
  total: number;
  limit: number;
  offset: number;
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
 * RefreshRequest represents the request body for token refresh.
 */
export interface RefreshRequest {
  refreshToken: string;
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

/**
 * ChangePasswordRequest represents the request body for changing a user's password.
 */
export interface ChangePasswordRequest {
  password: string;
}

/**
 * UsersListResponse represents the API response for listing users.
 */
export interface UsersListResponse {
  users: SystemUserAPI[];
  total: number;
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
  can_have_base_stations: boolean;
  max_base_station_count?: number;
  max_endpoint_count?: number;
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
  canHaveBaseStations: boolean;
  maxBaseStationCount?: number;
  maxEndpointCount?: number;
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
  can_have_base_stations?: boolean;
  max_base_station_count?: number;
  max_endpoint_count?: number;
  tags?: Record<string, string>;
}

/**
 * UpdateOrganizationRequest represents the request body for updating an organization.
 */
export interface UpdateOrganizationRequest {
  name?: string;
  description?: string;
  state?: string;
  can_have_base_stations?: boolean;
  max_base_station_count?: number;
  max_endpoint_count?: number;
  tags?: Record<string, string>;
}

/**
 * OrganizationsListResponse represents the API response for listing organizations.
 */
export interface OrganizationsListResponse {
  organizations: OrganizationAPI[];
  total: number;
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

/**
 * OrganizationUsersListResponse represents the API response for listing organization members.
 */
export interface OrganizationUsersListResponse {
  users: OrganizationUserAPI[];
  total: number;
}

// ============================================================================
// Blueprint Feature: Device Catalog and Payload Decoding Types
// ============================================================================

/**
 * ManufacturerAPI represents the API response format for manufacturers.
 * Uses camelCase per backend JSON conventions.
 */
export interface ManufacturerAPI {
  id: string;
  tenantId: number;
  name: string;
  website?: string;
  isVerified: boolean;
  modelCount?: number; // Populated on list queries
  createdAt: string;
  updatedAt: string;
}

/**
 * ManufacturerUI represents the UI-friendly format for manufacturers.
 */
export interface ManufacturerUI {
  id: string;
  tenantId: number;
  name: string;
  website?: string;
  isVerified: boolean;
  modelCount: number;
  createdAt: string;
  updatedAt: string;
}

/**
 * CreateManufacturerRequest represents the request body for creating a manufacturer.
 */
export interface CreateManufacturerRequest {
  name: string;
  website?: string;
}

/**
 * UpdateManufacturerRequest represents the request body for updating a manufacturer.
 */
export interface UpdateManufacturerRequest {
  name?: string;
  website?: string;
}

/**
 * ManufacturersListResponse represents the API response for listing manufacturers.
 */
export interface ManufacturersListResponse {
  manufacturers: ManufacturerAPI[];
  total: number;
}

/**
 * DeviceModelAPI represents the API response format for device models.
 * Uses camelCase per backend JSON conventions.
 */
export interface DeviceModelAPI {
  id: string;
  manufacturerId: string;
  tenantId: number;
  name: string;
  code: string;
  typeEui?: string; // 16-char hex (8 bytes)
  description?: string;
  datasheetUrl?: string;
  blueprintCount?: number; // Populated on list queries
  createdAt: string;
  updatedAt: string;
  // Joined data
  manufacturer?: ManufacturerAPI;
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
  blueprintCount: number;
  createdAt: string;
  updatedAt: string;
  manufacturerName?: string;
}

/**
 * CreateDeviceModelRequest represents the request body for creating a device model.
 */
export interface CreateDeviceModelRequest {
  name: string;
  code: string;
  typeEui?: string; // 16-char hex (8 bytes)
  description?: string;
  datasheetUrl?: string;
}

/**
 * CreateDeviceModelWithBlueprintRequest creates a model + default blueprint atomically.
 * Code and type_eui are auto-generated server-side.
 */
export interface CreateDeviceModelWithBlueprintRequest {
  manufacturerId: string;
  name: string;
  version: string;
  specJson: object;
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
 * DeviceModelsListResponse represents the API response for listing device models.
 */
export interface DeviceModelsListResponse {
  models: DeviceModelAPI[];
  total: number;
}

/**
 * BlueprintAPI represents the API response format for blueprints.
 * Uses camelCase per backend JSON conventions.
 */
export interface BlueprintAPI {
  id: string;
  deviceModelId: string;
  tenantId: number;
  version: string;
  typeEui: string; // 16-char hex (8 bytes)
  specJson: object; // Blueprint specification JSON
  isDefault: boolean;
  registryRepo?: string;
  registryCommitSha?: string;
  registryVerified: boolean;
  registryPrUrl?: string;
  createdAt: string;
  updatedAt: string;
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
  createdAt: string;
}

/**
 * CreateBlueprintRequest represents the request body for creating a blueprint.
 */
export interface CreateBlueprintRequest {
  version: string;
  typeEui?: string; // 16-char hex (8 bytes), optional for models without Type EUI
  specJson: object; // Blueprint specification JSON
}

/**
 * UpdateBlueprintRequest represents the request body for updating a blueprint.
 */
export interface UpdateBlueprintRequest {
  version?: string;
  specJson?: object;
}

/**
 * BlueprintsListResponse represents the API response for listing blueprints.
 */
export interface BlueprintsListResponse {
  blueprints: BlueprintListItemAPI[];
  total: number;
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

/**
 * BaseStationLocation represents a base station's location for global coverage display.
 */
export interface BaseStationLocation {
  bsEui: string;
  name: string;
  latitude: number;
  longitude: number;
  altitude?: number;
  locationSource: string;
  isOnline: boolean;
  orgId: string;
}

/**
 * ListAllBaseStationLocationsResponse contains all located base stations across tenants.
 */
export interface ListAllBaseStationLocationsResponse {
  locations: BaseStationLocation[];
  totalCount: number;
}
