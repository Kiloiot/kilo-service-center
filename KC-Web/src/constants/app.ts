/**
 * Application Constants
 * Centralized display strings and configuration values
 * Follow centralization rules: no hardcoded strings in components
 */

// Application Identity
export const APP_NAME = 'KiloCenter';
export const APP_TAGLINE = 'Measurable IoT';
export const APP_TITLE = 'KiloCenter MIOTY Service Center';
export const APP_EDITION = __APP_EDITION__;
export const APP_POWERED_BY_PREFIX = 'Powered by KiloCenter';

export function formatPoweredByLabel(edition: string): string {
  return `${APP_POWERED_BY_PREFIX} ${edition}`;
}

// Navigation
export const NAV_ITEMS = {
  DASHBOARD: 'Dashboard',
  BASE_STATIONS: 'Base Stations',
  ENDPOINTS: 'End Points',
  BLUEPRINTS: 'Blueprints', // Blueprint feature: Device catalog and payload decoding
  CERTIFICATES: 'Certificates',
  USERS: 'Users & Roles',
  API_KEYS: 'API Keys',
  ORGANIZATIONS: 'Organizations',
} as const;

// Routes
export const ROUTES = {
  HOME: '/',
  BASE_STATIONS: '/base-stations',
  BASE_STATION_DETAIL: '/base-stations/:id',
  ENDPOINTS: '/endpoints',
  ENDPOINT_DETAIL: '/endpoints/:id',
  BLUEPRINTS: '/blueprints', // Blueprint feature: Device catalog
  BLUEPRINT_MANUFACTURER: '/blueprints/manufacturer/:mfrId', // Manufacturer details
  BLUEPRINT_MODEL: '/blueprints/model/:modelId', // Model details + blueprints list
  BLUEPRINT_DETAIL: '/blueprints/:id', // Blueprint editor/viewer
  BLUEPRINT_MODEL_NEW: '/blueprints/models/new', // Add device model with blueprint
  CERTIFICATES: '/settings/certificates',
  LOGIN: '/login',
  REGISTER: '/register',
  AUTH_CALLBACK: '/auth/callback',
  USERS: '/users',
  USER_DETAIL: '/users/:id',
  USER_PASSWORD: '/users/:id/password',
  MY_PASSWORD: '/auth/my-password', // Self-service password change
  API_KEYS: '/api-keys',
  ORGANIZATIONS: '/organizations',
  ORGANIZATION_DETAIL: '/organizations/:id',
  ORGANIZATION_USERS: '/organizations/:id/users',
} as const;

// Route Titles (replaces all literal titles in routes.ts)
export const ROUTE_TITLES = {
  DASHBOARD: 'Dashboard',
  BASE_STATIONS: 'Base Stations',
  BASE_STATION_DETAIL: 'Base Station Details',
  ENDPOINTS: 'Endpoints',
  ENDPOINT_DETAIL: 'Endpoint Details',
  BLUEPRINTS: 'Blueprints', // Blueprint feature
  BLUEPRINT_MANUFACTURER: 'Manufacturer Details',
  BLUEPRINT_MODEL: 'Model Details',
  BLUEPRINT_DETAIL: 'Blueprint Details',
  BLUEPRINT_MODEL_NEW: 'Add Device Model',
  CERTIFICATES: 'Certificates',
  LOGIN: 'Login',
  REGISTER: 'Create Account',
  AUTH_CALLBACK: 'Authentication',
  USERS: 'Users & Roles',
  USER_DETAIL: 'User Details',
  MY_PASSWORD: 'Change Password', // Self-service password change
  API_KEYS: 'API Keys',
  ORGANIZATIONS: 'Organizations',
  ORGANIZATION_DETAIL: 'Organization Details',
  ORGANIZATION_USERS: 'Organization Users',
  DEFAULT: APP_TITLE, // Browser title fallback
} as const;

// UI Configuration
export const DRAWER_WIDTH = 240;

/**
 * Logo configuration for navigation drawer
 * Path is relative to public folder (served at root)
 */
export const LOGO = {
  LIGHT_PATH: '/Kilo_Service_Center.svg',
  DARK_PATH: '/kilo_logo_white.svg',
  WIDTH: 120,
  ALT: APP_NAME,
} as const;

// Timing Constants (milliseconds)
export const TIMING_DASHBOARD_REFRESH = 10000; // Dashboard: 10s
export const TIMING_LIST_REFRESH = 30000; // Base stations/endpoints lists: 30s
export const TIMING_MESSAGE_REFRESH = 5000; // Messages page: 5s
export const TIMING_STATUS_REFRESH = 30000; // System status: 30s
export const TIMING_COPY_FEEDBACK = 2000; // "Copied" indicator: 2s
export const TIMING_REFRESH_DEBOUNCE = 100; // Post-action refresh: 100ms

// Realtime reconnection timing (ms)
export const TIMING_RECONNECT_BASE_DELAY = 1000; // Base delay for reconnection
export const TIMING_RECONNECT_MAX_DELAY = 30000; // Max delay cap for exponential backoff

/**
 * TIMING object for React Query staleTime patterns.
 * Values are in SECONDS - multiply by 1000 for millisecond APIs (setTimeout, setInterval).
 * @see TIMING_* constants above for direct millisecond values
 */
export const TIMING = {
  DASHBOARD_REFRESH: 10, // Dashboard stats staleTime (seconds)
  LIST_REFRESH: 30, // Default staleTime for lists (seconds)
  MESSAGE_REFRESH: 5, // Fallback polling for messages (seconds)
} as const;

// Status Labels
export const STATUS = {
  ONLINE: 'online',
  OFFLINE: 'offline',
  ACTIVE: 'active',
  INACTIVE: 'inactive',
  PENDING: 'pending',
  ERROR: 'error',
} as const;

/**
 * Event categories for dashboard server activity feed.
 * Excludes 'message' category (endpoint uplink/downlink traffic).
 * Operator-level events only. Excludes per-device traffic (endpoint, basestation, message, protocol).
 */
export const SERVER_ACTIVITY_CATEGORIES = [
  'system', // Service lifecycle (started, stopped)
  'audit', // Admin actions (user/org CRUD)
  'security', // Auth failures, access violations
  'error', // System errors (including SCACI errors)
] as const;

/**
 * Event types for the Platform Activity feed.
 * Filters dashboard to operator-relevant events only (lifecycle, admin, security).
 * Must match EventType* constants in KC-DB/storage/models/system_event.go.
 */
export const SERVER_ACTIVITY_EVENT_TYPES = [
  'service.started', 'service.stopped',
  'migration.applied', 'migration.failed',
  'user.created', 'user.updated', 'user.deleted',
  'organization.created', 'organization.updated', 'organization.deleted',
  'api_key.created', 'api_key.deleted',
  'integration.created', 'integration.updated', 'integration.deleted',
  'certificate.generated', 'certificate.server_generated', 'certificate.server_renewed',
  'auth.invalid_token', 'auth.api_key_rejected',
  'auth.organization_context_missing', 'auth.organization_resolution_failed',
  'auth.internal_trust_violation', 'auth.permission_denied',
  'auth.login_failed', 'auth.oidc_exchange_failed', 'auth.oauth2_exchange_failed',
] as const;

// Message Types
export const MESSAGE_TYPES = {
  UPLINK: 'uplink',
  DOWNLINK: 'downlink',
} as const;

// Default message type for UL data
export const MESSAGE_TYPE_DEFAULT = 'ulData';

// Downlink queue lifecycle statuses (canonical from bssci/constants.go:216-226)
export const DOWNLINK_QUEUE_STATUS = {
  PENDING: 'pending',
  SCHEDULED: 'scheduled',
  RESERVED: 'reserved',
  QUEUED: 'queued',
  TRANSMITTED: 'transmitted',
  DELIVERED: 'delivered',
  FAILED: 'failed',
  EXPIRED: 'expired',
  REVOKED: 'revoked',
  ACKED: 'acked',
} as const;

// Downlink transmission result values (maps msg.Result at core_service.go:1790)
export const DOWNLINK_RESULT = {
  SENT: 'sent',
  EXPIRED: 'expired',
  INVALID: 'invalid',
} as const;

// Queue states eligible for revocation (non-terminal)
export const REVOCABLE_QUEUE_STATUSES: Set<string> = new Set([
  DOWNLINK_QUEUE_STATUS.PENDING,
  DOWNLINK_QUEUE_STATUS.SCHEDULED,
  DOWNLINK_QUEUE_STATUS.RESERVED,
  DOWNLINK_QUEUE_STATUS.QUEUED,
]);

// Priority presets for downlink compose form
export const DOWNLINK_PRIORITY_PRESETS = [
  { label: 'Low', value: 0.0 },
  { label: 'Normal', value: 0.5 },
  { label: 'High', value: 1.0 },
] as const;

// Realtime event types (must match RealtimeEventType union in types.ts)
export const REALTIME_EVENT_TYPE = {
  UPLINK_RECEIVED: 'uplink.received',
  DOWNLINK_QUEUED: 'downlink.queued',
  DOWNLINK_SENT: 'downlink.sent',
  DOWNLINK_ACKNOWLEDGED: 'downlink.acknowledged',
  DOWNLINK_FAILED: 'downlink.failed',
  DOWNLINK_REVOKED: 'downlink.revoked',
  ENDPOINT_ATTACHED: 'endpoint.attached',
  ENDPOINT_DETACHED: 'endpoint.detached',
  BASESTATION_ONLINE: 'basestation.online',
  BASESTATION_OFFLINE: 'basestation.offline',
  SCACI_SESSION_OPENED: 'scaci.session.opened',
  SCACI_SESSION_CLOSED: 'scaci.session.closed',
  SCACI_ERROR: 'scaci.error',
  EVENT_RECEIVED: 'event.received',
} as const;

/**
 * Backend event_type → UI realtime event type mapping
 * Backend uses underscore/dotted format (DB-compatible), UI uses dotted format
 * CRUD events map to generic EVENT_RECEIVED to preserve semantics
 */
export const BACKEND_EVENT_TO_REALTIME: Record<
  string,
  (typeof REALTIME_EVENT_TYPE)[keyof typeof REALTIME_EVENT_TYPE]
> = {
  // Status mappings
  endpoint_attached: REALTIME_EVENT_TYPE.ENDPOINT_ATTACHED,
  endpoint_detached: REALTIME_EVENT_TYPE.ENDPOINT_DETACHED,
  basestation_online: REALTIME_EVENT_TYPE.BASESTATION_ONLINE,
  basestation_offline: REALTIME_EVENT_TYPE.BASESTATION_OFFLINE,
  uldata_completed: REALTIME_EVENT_TYPE.UPLINK_RECEIVED,
  // CRUD events → generic event.received (preserves semantics)
  'basestation.registered': REALTIME_EVENT_TYPE.EVENT_RECEIVED,
  'basestation.updated': REALTIME_EVENT_TYPE.EVENT_RECEIVED,
  'basestation.deregistered': REALTIME_EVENT_TYPE.EVENT_RECEIVED,
  'endpoint.created': REALTIME_EVENT_TYPE.EVENT_RECEIVED,
  'endpoint.updated': REALTIME_EVENT_TYPE.EVENT_RECEIVED,
  'endpoint.deleted': REALTIME_EVENT_TYPE.EVENT_RECEIVED,
  'user.created': REALTIME_EVENT_TYPE.EVENT_RECEIVED,
  'user.deleted': REALTIME_EVENT_TYPE.EVENT_RECEIVED,
  'service.started': REALTIME_EVENT_TYPE.EVENT_RECEIVED,
  'service.stopped': REALTIME_EVENT_TYPE.EVENT_RECEIVED,
  // Downlink events (audit_logger.go + bssci/constants.go)
  dl_data_queue_acknowledged: REALTIME_EVENT_TYPE.DOWNLINK_QUEUED,
  dl_data_sent: REALTIME_EVENT_TYPE.DOWNLINK_SENT,
  dl_data_expired: REALTIME_EVENT_TYPE.DOWNLINK_FAILED,
  dl_data_invalid: REALTIME_EVENT_TYPE.DOWNLINK_FAILED,
  dl_data_revoke_initiated: REALTIME_EVENT_TYPE.DOWNLINK_REVOKED,
  dl_data_revoked: REALTIME_EVENT_TYPE.DOWNLINK_REVOKED,
} as const;

// Certificate type identifiers (config values, not user-facing)
export const CERTIFICATE_TYPES = {
  SERVER: 'server',
  CA: 'ca',
} as const;

// Certificate download types (used by DownloadCertificate RPC)
export const CERTIFICATE_DOWNLOAD_TYPES = {
  CA: 'ca',
  CLIENT: 'client',
  KEY: 'key',
} as const;

export type CertificateDownloadType =
  (typeof CERTIFICATE_DOWNLOAD_TYPES)[keyof typeof CERTIFICATE_DOWNLOAD_TYPES];

/**
 * API key type identifiers.
 * Matches backend api_key key_type column values.
 */
export const API_KEY_TYPES = {
  USER: 'user',
  SERVICE_ACCOUNT: 'service_account',
} as const;

/**
 * API key list page size.
 * Larger than default pagination since API keys are lightweight.
 */
export const API_KEY_PAGE_SIZE = 100;

// Registry defaults
export const REGISTRY_DEFAULTS = {
  BRANCH: 'main',
} as const;

/**
 * Local storage keys used across the application.
 * Centralized to prevent typos and enable refactoring.
 */
export const STORAGE_KEYS = {
  AUTH_TOKEN: 'auth_token',
  REFRESH_TOKEN: 'kc_refresh_token',
  USER_PROFILE: 'kc_user_profile', // Persisted user profile for session hydration
  TENANT: 'tenant', // Reserved - see BLOCKERS.txt: "Tenant ID type mismatch"
  FILTERS: 'kc-filters',
  SAVED_VIEWS: 'kc-saved-views',
} as const;

/**
 * HTTP header names used for gRPC-web metadata (matches KC-Core interceptors).
 * Ensures frontend-backend header consistency for tenant isolation.
 */
export const HEADERS = {
  X_TENANT_ID: 'X-Tenant-ID', // Reserved - see BLOCKERS.txt: "Tenant ID type mismatch"
  CONTENT_TYPE: 'Content-Type',
  AUTHORIZATION: 'Authorization',
  API_VERSION: 'X-API-Version',
  X_ORGANIZATION_ID: 'X-Organization-ID',
  X_USER_ID: 'X-User-ID',
} as const;

/**
 * HTTP header values - centralized to avoid literals in api.ts
 */
export const HEADER_VALUES = {
  CONTENT_TYPE_JSON: 'application/json',
  API_VERSION: 'v2',
  AUTH_BEARER_PREFIX: 'Bearer ',
} as const;

// Protocol URLs (BSSCI Section 1 specification references)
export const PROTOCOL_URLS = {
  BSSCI_DEFAULT: 'tls://localhost:5000',
  SCACI_DEFAULT: 'tls://localhost:5001',
  BSSCI_DOCS: 'https://www.ets.org/mioty',
  SCACI_DOCS: 'https://www.ets.org/mioty',
} as const;

// Default Values
export const DEFAULT_ORG_NAME = 'Default Organization';

/**
 * Environment badge labels.
 * Centralized to avoid inline string literals in EnvBadge.tsx.
 */
export const ENV_LABELS = {
  development: 'DEV',
  staging: 'STG',
  production: 'PROD',
} as const;

/**
 * Environment badge tooltips.
 * Centralized to avoid inline string literals in EnvBadge.tsx.
 */
export const ENV_TOOLTIPS = {
  development: 'Development Environment',
  staging: 'Staging Environment',
  production: 'Production Environment',
} as const;

/**
 * External IdP claim path for organization ID extraction from JWT.
 * Default matches external IdP claim structure; override via VITE_EXTERNAL_ORG_CLAIM_PATH for other IdPs.
 */
export const DEFAULT_EXTERNAL_ORG_CLAIM_PATH = 'urn:zitadel:iam:org:project:id';

/**
 * TIMING_MS - Millisecond timing constants for direct usage
 * These are new additions with explicit _MS suffix for clarity
 * @see TIMING for second-based constants used with staleTime
 */
export const TIMING_MS = {
  DEBOUNCE_SEARCH: 300, // Search input debounce
  POLL_INTERVAL_FALLBACK: 5000, // Fallback polling when realtime disconnected
} as const;

/**
 * UI timing constants for user feedback.
 * Used for success message auto-dismiss and navigation delays.
 */
export const UI_TIMING = {
  SUCCESS_MESSAGE_DISMISS_MS: 3000, // Auto-dismiss success alerts after 3s
  NAVIGATION_DELAY_MS: 1500, // Delay before navigation after successful action
} as const;

/**
 * Certificate validity options in days
 * Matches BSSCI certificate generation requirements
 */
export const CERT_VALIDITY_DAYS = {
  ONE_YEAR: 365,
  TWO_YEARS: 730,
  THREE_YEARS: 1095,
} as const;

/**
 * Auth layout constants for login/callback pages.
 * Centralizes spacing and sizing to avoid inline literals.
 */
export const AUTH_LAYOUT = {
  FULL_HEIGHT: '100vh',
  CARD_PADDING: 4,
  CARD_MAX_WIDTH: 400,
  SPACING_MT: 2,
  SPACING_MB: 2,
  SPACING_ML: 2,
  SPINNER_SIZE: 24,
} as const;

/**
 * Base station detail page layout constants.
 * Centralizes spacing values to avoid inline literals.
 */
export const BS_DETAIL_LAYOUT = {
  GRID_SPACING: 10,
  DIALOG_ALERT_MX: 3,
  DIALOG_ALERT_MB: 1,
} as const;

/**
 * Organization role values.
 * Matches backend organization_constants.go.
 */
export const ORG_ROLE = {
  OWNER: 'owner',
  ADMIN: 'admin',
  MEMBER: 'member',
} as const;

/**
 * Organization member status values.
 * Matches backend organization_constants.go.
 */
export const ORG_MEMBER_STATUS = {
  ACTIVE: 'active',
  INVITED: 'invited',
  REMOVED: 'removed',
} as const;

/**
 * Organization state values.
 * Matches backend organization_constants.go.
 */
export const ORG_STATE = {
  ACTIVE: 'active',
  SUSPENDED: 'suspended',
  ARCHIVED: 'archived',
} as const;

/**
 * User lookup limit for autocomplete.
 * No search API available - client-side filtering.
 */
export const USER_LOOKUP_LIMIT = 1000;

/**
 * Placeholder for email input in organization user forms.
 */
export const ORG_USER_PLACEHOLDER_EMAIL = 'user@example.com';

/**
 * Pagination defaults and options
 * Used for consistent table pagination across all list views
 * UI spacing tokens are in src/theme/index.ts:componentSpacing.pagination
 */
export const PAGINATION = {
  DEFAULT_PAGE_SIZE: 25,
  PAGE_SIZE_OPTIONS: [10, 25, 50, 100] as const,
  DOWNLINK_QUEUE_PAGE_SIZE: 20,
  DOWNLINK_RESULTS_PAGE_SIZE: 20,
  DOWNLINK_FLUSH_PAGE_SIZE: 100,
} as const;

/**
 * Decode status constants (mirrors KC-Core/pkg/blueprint/constants.go)
 * Used for message display and filtering
 */
export const DECODE_STATUS = {
  PENDING: 'pending',
  SUCCESS: 'success',
  FAILED: 'failed',
  SKIPPED: 'skipped',
} as const;

/**
 * UI truncation constants for data display
 * Used for payload preview, config display, etc.
 */
export const TRUNCATION = {
  PAYLOAD_PREVIEW_LENGTH: 32,
  CONFIG_PREVIEW_LENGTH: 100,
  ELLIPSIS: '...',
} as const;

/**
 * Endpoint update field mask paths for gRPC partial updates.
 * Mirrors backend fieldMask* constants in kilocenter_service.go.
 */
export const ENDPOINT_UPDATE_MASK_PATHS = {
  NAME: 'name',
  DESCRIPTION: 'description',
  EP_CLASS: 'ep_class',
  STATUS: 'status',
  SH_ADDR: 'sh_addr',
  ATTACH_CNT: 'attach_cnt',
  DUAL_CHAN: 'dual_chan',
  REPETITION: 'repetition',
  WIDE_CARR_OFF: 'wide_carr_off',
  LONG_BLK_DIST: 'long_blk_dist',
  PRE_ATTACH: 'pre_attach',
  LAST_PACKET_CNT: 'last_packet_cnt',
  CARRIER_OFFSET: 'carrier_offset',
  NWK_SN_KEY: 'nwk_sn_key',
  APP_KEY: 'app_key',
  DEVICE_MODEL_ID: 'device_model_id',
  TYPE_EUI: 'type_eui',
} as const;

/**
 * User update field mask paths for gRPC partial updates.
 * Mirrors backend userField* constants in kilocenter_service_auth.go.
 */
export const USER_UPDATE_MASK_PATHS = {
  EMAIL: 'email',
  IS_ADMIN: 'is_admin',
  IS_ACTIVE: 'is_active',
  IS_TENANT_MANAGER: 'is_tenant_manager',
  IS_BASE_STATION_MANAGER: 'is_base_station_manager',
  IS_ENDPOINT_MANAGER: 'is_endpoint_manager',
  NOTE: 'note',
} as const;

/**
 * Organization user update field mask paths for gRPC partial updates.
 * Mirrors backend orgUserField* constants in kilocenter_service_admin.go.
 */
export const ORG_USER_UPDATE_MASK_PATHS = {
  ROLE: 'role',
  IS_ORG_ADMIN: 'is_org_admin',
  IS_BASE_STATION_ADMIN: 'is_base_station_admin',
  IS_ENDPOINT_ADMIN: 'is_endpoint_admin',
} as const;

/**
 * Endpoint class constants per BSSCI specification.
 * 'A' = Active (bidirectional), 'Z' = Zero-power (unidirectional).
 */
export const ENDPOINT_EP_CLASS = {
  BIDIRECTIONAL: 'A',
  UNIDIRECTIONAL: 'Z',
} as const;

/**
 * MIOTY protocol validation constants per BSSCI v1.0.0 §3.8.1 and SCACI §3.6.1.
 * Shared across Add/Edit endpoint dialogs.
 */
export const MIOTY_KEY_BYTE_LENGTH = 16;
export const MIOTY_KEY_HEX_LENGTH = 32;
export const MIOTY_EUI_HEX_LENGTH = 16;
export const MIOTY_SHORT_ADDR_MAX = 0xffff;
export const MIOTY_UINT32_MAX = 4294967295;
export const MIOTY_EUI_REGEX = /^[0-9A-Fa-f]{16}$/;
export const MIOTY_KEY_REGEX = /^[0-9A-Fa-f]{32}$/;
export const MIOTY_SHORT_ADDR_REGEX = /^[0-9A-Fa-f]{1,4}$/;

/** Activity threshold in hours — endpoints seen within this window are "active".
 * Matches backend DefaultEndpointActivityWindowHours. */
export const ENDPOINT_ACTIVITY_WINDOW_HOURS = 24;

/** Backend event_type values for endpoint/BS lifecycle events.
 * Must match canonical constants in KC-DB/storage/models/system_event.go. */
export const ENDPOINT_EVENT_TYPES = {
  ATTACH_PROPAGATE_INITIATED: 'attach_propagate_initiated',
  ATTACH_PROPAGATE_COMPLETED: 'attach_propagate_completed',
  DETACH_PROPAGATE_INITIATED: 'detach_propagate_initiated',
  DETACH_PROPAGATE_COMPLETED: 'detach_propagate_completed',
  ENDPOINT_ATTACHED: 'endpoint_attached',
  ENDPOINT_DETACHED: 'endpoint_detached',
} as const;

/** Terminal event timeline column widths (px) for EndPointDetails event rendering */
export const ENDPOINT_TERMINAL_LAYOUT = {
  ENDPOINT_COL_WIDTH: '260px',
  STATUS_COL_WIDTH: '90px',
  CONNECTOR_COL_WIDTH: '140px',
  BASESTATION_MARGIN_LEFT: '8px',
} as const;

/** Geographic coordinate validation bounds */
export const GEO_BOUNDS = {
  LATITUDE_MIN: -90,
  LATITUDE_MAX: 90,
  LONGITUDE_MIN: -180,
  LONGITUDE_MAX: 180,
} as const;

/** Map defaults for location picker and detail page map card.
 * TILE_URL / TILE_ATTRIBUTION are neutral defaults (CartoDB Voyager).
 * Deployments override via VITE_MAP_TILE_URL / VITE_MAP_TILE_ATTRIBUTION env vars
 * resolved through env.mapTileUrl / env.mapTileAttribution at render time. */
export const MAP_DEFAULTS = {
  CENTER_LAT: 0,
  CENTER_LNG: 0,
  ZOOM_DEFAULT: 2,
  ZOOM_LOCATION: 10,
  TILE_URL: 'https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png',
  TILE_ATTRIBUTION:
    '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
  PICKER_HEIGHT: 400,
} as const;
