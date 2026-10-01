/**
 * Application Constants
 * Centralized display strings and configuration values
 * Follow centralization rules: no hardcoded strings in components
 */

// Application Identity
export const APP_NAME = "KiloCenter";
export const APP_TITLE = "KiloCenter MIOTY Service Center";
export const APP_EDITION = __APP_EDITION__;

/** Runtime edition code GetVersion reports for enterprise builds (KC-Core/pkg/config EditionECE). */
export const EDITION_CODE = {
  ENTERPRISE: "ece",
} as const;

const APP_POWERED_BY_PREFIX = "Powered by KiloCenter";

export function formatPoweredByLabel(edition: string): string {
  return `${APP_POWERED_BY_PREFIX} ${edition}`;
}

// Navigation
export const NAV_ITEMS = {
  DASHBOARD: "Dashboard",
  BASE_STATIONS: "Base Stations",
  ENDPOINTS: "End Points",
  TRAFFIC: "Traffic",
  LOGS: "Logs",
  BLUEPRINTS: "Blueprints", // Blueprint feature: Device catalog and payload decoding
  CERTIFICATES: "Certificates",
  USERS: "Users & Roles",
  API_KEYS: "API Keys",
  ORGANIZATIONS: "Organizations",
} as const;

/**
 * What a page or action requires of the signed-in user's roles. An
 * administrator satisfies every requirement; ANY_ROLE is met by any role.
 */
export const ROLE_REQUIREMENT = {
  ANY_ROLE: "anyRole",
  /** A base station or endpoint manager: the roles that read device events. */
  DEVICE_MANAGER: "deviceManager",
  BASE_STATION_MANAGER: "baseStationManager",
  ENDPOINT_MANAGER: "endpointManager",
  TENANT_MANAGER: "tenantManager",
  ADMIN: "admin",
} as const;

export type RoleRequirement =
  (typeof ROLE_REQUIREMENT)[keyof typeof ROLE_REQUIREMENT];

// Routes
export const ROUTES = {
  HOME: "/",
  BASE_STATIONS: "/base-stations",
  BASE_STATION_DETAIL: "/base-stations/:id",
  ENDPOINTS: "/endpoints",
  ENDPOINT_DETAIL: "/endpoints/:id",
  TRAFFIC: "/traffic",
  LOGS: "/logs",
  BLUEPRINTS: "/blueprints", // Blueprint feature: Device catalog
  BLUEPRINT_MANUFACTURER: "/blueprints/manufacturer/:mfrId", // Manufacturer details
  BLUEPRINT_MODEL: "/blueprints/model/:modelId", // Model details + blueprints list
  BLUEPRINT_DETAIL: "/blueprints/:id", // Blueprint editor/viewer
  BLUEPRINT_MODEL_NEW: "/blueprints/models/new", // Add device model with blueprint
  CERTIFICATES: "/settings/certificates",
  LOGIN: "/login",
  REGISTER: "/register",
  AUTH_CALLBACK: "/auth/callback",
  USERS: "/users",
  USER_DETAIL: "/users/:id",
  USER_PASSWORD: "/users/:id/password",
  MY_PASSWORD: "/auth/my-password", // Self-service password change
  API_KEYS: "/api-keys",
  ORGANIZATIONS: "/organizations",
  ORGANIZATION_DETAIL: "/organizations/:id",
  ORGANIZATION_USERS: "/organizations/:id/users",
} as const;

/** Query and fragment parameters the identity provider returns to the auth callback (RFC 6749). */
export const OAUTH_CALLBACK_PARAMS = {
  CODE: "code",
  STATE: "state",
  ERROR: "error",
  ACCESS_TOKEN: "access_token",
} as const;

/** Path pattern that matches every route no other route claims. */
export const ROUTE_CATCH_ALL = "*";

// Route Titles (replaces all literal titles in routes.ts)
export const ROUTE_TITLES = {
  DASHBOARD: "Dashboard",
  BASE_STATIONS: "Base Stations",
  BASE_STATION_DETAIL: "Base Station Details",
  ENDPOINTS: "Endpoints",
  ENDPOINT_DETAIL: "Endpoint Details",
  TRAFFIC: "Traffic",
  LOGS: "Logs",
  BLUEPRINTS: "Blueprints", // Blueprint feature
  BLUEPRINT_MANUFACTURER: "Manufacturer Details",
  BLUEPRINT_MODEL: "Model Details",
  BLUEPRINT_DETAIL: "Blueprint Details",
  BLUEPRINT_MODEL_NEW: "Add Device Model",
  CERTIFICATES: "Certificates",
  LOGIN: "Login",
  REGISTER: "Create Account",
  AUTH_CALLBACK: "Authentication",
  USERS: "Users & Roles",
  USER_DETAIL: "User Details",
  MY_PASSWORD: "Change Password", // Self-service password change
  API_KEYS: "API Keys",
  ORGANIZATIONS: "Organizations",
  ORGANIZATION_DETAIL: "Organization Details",
  ORGANIZATION_USERS: "Organization Users",
} as const;

/** Infixes of the ids that pair each tab with its panel for assistive technology. */
export const TAB_A11Y_ID = {
  TAB: "-tab-",
  PANEL: "-tabpanel-",
} as const;

/** Id of the index.html element the app mounts into. */
export const APP_ROOT_ELEMENT_ID = "root";

// UI Configuration
export const DRAWER_WIDTH = 240;

/**
 * The Service Center product logo (served from the public folder) and the
 * widths it is drawn at; the aspect ratio is the artwork's viewBox.
 */
export const LOGO = {
  PATH: "/Kilo_Service_Center.svg",
  ASPECT_RATIO: "512 / 213",
  NAV_WIDTH: 120,
  AUTH_WIDTH: 200,
  ALT: APP_NAME,
} as const;

// Time units
export const MS_PER_SECOND = 1000;
export const SECONDS_PER_MINUTE = 60;
export const MINUTES_PER_HOUR = 60;
export const HOURS_PER_DAY = 24;
export const DAYS_PER_WEEK = 7;
/** Length of the "last month" time window. */
export const DAYS_PER_MONTH_WINDOW = 30;
export const SECONDS_PER_HOUR = SECONDS_PER_MINUTE * MINUTES_PER_HOUR;
export const SECONDS_PER_DAY = SECONDS_PER_HOUR * HOURS_PER_DAY;
export const MS_PER_MINUTE = MS_PER_SECOND * SECONDS_PER_MINUTE;
export const MS_PER_HOUR = MS_PER_MINUTE * MINUTES_PER_HOUR;
export const MS_PER_DAY = MS_PER_HOUR * HOURS_PER_DAY;
export const NANOSECONDS_PER_MILLISECOND = 1_000_000;

/** The single MIOTY display locale: day-first dates, 24-hour time. */
export const MIOTY_LOCALE = "en-GB";

/** Time-of-day rendering on MIOTY_LOCALE (HH:mm:ss). */
export const TIME_24H_FORMAT: Intl.DateTimeFormatOptions = {
  hour12: false,
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
};

/** Growth factor of every exponential retry and reconnect backoff. */
export const BACKOFF_FACTOR = 2;

export const TIMING_STATUS_REFRESH = 30000; // System status: 30s
export const TIMING_ROLES_REFRESH = 30000; // Signed-in user roles: as long as the service caches them
/**
 * Base station detail re-read interval, the service center's default
 * protocol.status_request_interval (30 s). The periodic statusRsp is a sample,
 * not an event: streaming it would add a system event per station every
 * interval to the activity and event logs, so the detail samples instead.
 * Only the answer to an operator's status request is announced.
 */
export const TIMING_BASE_STATION_STATUS_REFRESH_MS = 30000;
export const TIMING_COPY_FEEDBACK = 2000; // "Copied" indicator: 2s
export const TIMING_NOTICE_AUTO_HIDE = 6000; // Action feedback snackbar: 6s
// A saved file's object URL outlives the click that starts its download.
export const TIMING_DOWNLOAD_URL_REVOKE_MS = 1000;
// The access token is renewed this long before its exp, and never sooner than the min delay from now.
export const TIMING_TOKEN_REFRESH_BUFFER_MS = 60_000;
export const TIMING_TOKEN_REFRESH_MIN_DELAY_MS = 5_000;
// An access token this close to its exp is renewed before a call or stream sends it.
export const TIMING_ACCESS_TOKEN_EXPIRY_SKEW_MS = 10000;
// A proactive refresh the service could not answer is retried with backoff between these bounds.
export const TIMING_TOKEN_REFRESH_RETRY_BASE_MS = 5_000;
export const TIMING_TOKEN_REFRESH_RETRY_MAX_MS = 60_000;
// Realtime reconnection timing (ms)
export const TIMING_RECONNECT_BASE_DELAY = 1000; // Base delay for reconnection
export const TIMING_RECONNECT_MAX_DELAY = 30000; // Max delay cap for exponential backoff

// Catch-up invalidation cadence after a stream reconnects. The debounce
// collapses any rapid reconnect burst into a single invalidation; the min
// interval is a hard rate limit even under pathological reconnect cycling.
export const TIMING_REALTIME_CATCHUP_DEBOUNCE_MS = 500;
export const TIMING_REALTIME_CATCHUP_MIN_INTERVAL_MS = 5000;

// While the streams a user holds are not all connected, the open views re-read at this cadence.
export const TIMING_FALLBACK_POLL_MS = 15000;

// Realtime invalidations are coalesced into at most one flush per this window,
// so a busy tenant's event stream can't drive a refetch-per-event storm.
export const TIMING_REALTIME_INVALIDATION_WINDOW_MS = 2500;
// The first change after a quiet window waits only this long, so the events
// that arrive together with it share one flush.
export const TIMING_REALTIME_INVALIDATION_BATCH_MS = 250;

/**
 * TIMING object for React Query staleTime patterns.
 * Values are in SECONDS - multiply by 1000 for millisecond APIs (setTimeout, setInterval).
 * @see TIMING_* constants above for direct millisecond values
 */
export const TIMING = {
  LIST_REFRESH: 30, // Default staleTime for lists (seconds)
} as const;

/** React Query defaults: how long unused data stays cached and how a failed read is retried. */
export const QUERY_DEFAULTS = {
  GC_TIME_MS: 5 * MS_PER_MINUTE,
  RETRY_COUNT: 2,
  RETRY_BASE_DELAY_MS: 1000,
  RETRY_MAX_DELAY_MS: 30000,
} as const;

/** Days a dashboard card looks back when counting newly registered devices. */
export const DASHBOARD_ADDED_WINDOW_DAYS = 7;

// Downlink queue lifecycle statuses (canonical from bssci/constants.go:216-226)
export const DOWNLINK_QUEUE_STATUS = {
  PENDING: "pending",
  SCHEDULED: "scheduled",
  RESERVED: "reserved",
  QUEUED: "queued",
  REVOKING: "revoking",
  TRANSMITTED: "transmitted",
  DELIVERED: "delivered",
  FAILED: "failed",
  EXPIRED: "expired",
  REVOKED: "revoked",
  ACKED: "acked",
} as const;

// Downlink transmission result values (maps msg.Result at core_service.go:1790)
export const DOWNLINK_RESULT = {
  SENT: "sent",
  EXPIRED: "expired",
  INVALID: "invalid",
  REVOKED: "revoked",
} as const;

// Queue states eligible for revocation (non-terminal)
/** DL RX status query states KC-Core reports (GetDLRXStatusQueries). */
export const DL_RX_QUERY_STATUS = {
  PENDING: "pending",
  RECEIVED: "received",
  TIMEOUT: "timeout",
} as const;

/** Query-key segments of end point views kept under the end point's detail key. */
export const ENDPOINT_DETAIL_QUERY_SEGMENT = {
  DL_RX_STATUS: "dlRxStatus",
  DL_RX_STATUS_QUERIES: "dlRxStatusQueries",
} as const;

export const REVOCABLE_QUEUE_STATUSES: Set<string> = new Set([
  DOWNLINK_QUEUE_STATUS.PENDING,
  DOWNLINK_QUEUE_STATUS.SCHEDULED,
  DOWNLINK_QUEUE_STATUS.RESERVED,
  DOWNLINK_QUEUE_STATUS.QUEUED,
  DOWNLINK_QUEUE_STATUS.REVOKING,
]);

/** Views of the Traffic section (NAV_STRUCTURE §4). */
export const TRAFFIC_VIEW = {
  UPLINKS: "uplinks",
  DOWNLINK_QUEUE: "downlinkQueue",
  DOWNLINK_RESULTS: "downlinkResults",
} as const;

export type TrafficView = (typeof TRAFFIC_VIEW)[keyof typeof TRAFFIC_VIEW];

export const TRAFFIC_VIEWS: readonly TrafficView[] =
  Object.values(TRAFFIC_VIEW);

/** The endpoint keys an endpoint manager can reveal (GetEndPoint reveal_keys). */
export const ENDPOINT_KEY = {
  NETWORK: "nwkSnKey",
  APPLICATION: "appKey",
} as const;

export type EndpointKeyName = (typeof ENDPOINT_KEY)[keyof typeof ENDPOINT_KEY];

/**
 * Where the uplink view reads: the tenant listing (ListMessages, endpoint
 * managers) or one base station's listing (ListBaseStationMessages, base
 * station managers).
 */
export const UPLINK_SOURCE = {
  TENANT: "tenant",
  STATION: "station",
} as const;

export type UplinkSource = (typeof UPLINK_SOURCE)[keyof typeof UPLINK_SOURCE];

/** Views of the Logs section (NAV_STRUCTURE §5). */
export const LOG_VIEW = {
  EVENTS: "events",
  ERRORS: "errors",
  AUDIT: "audit",
} as const;

export type LogView = (typeof LOG_VIEW)[keyof typeof LOG_VIEW];

export const LOG_VIEWS: readonly LogView[] = Object.values(LOG_VIEW);

/** Tabs of the base station detail page. */
export const BASE_STATION_DETAIL_VIEW = {
  ACTIVITY: "activity",
  TRAFFIC: "traffic",
} as const;

export type BaseStationDetailView =
  (typeof BASE_STATION_DETAIL_VIEW)[keyof typeof BASE_STATION_DETAIL_VIEW];

export const BASE_STATION_DETAIL_VIEWS: readonly BaseStationDetailView[] =
  Object.values(BASE_STATION_DETAIL_VIEW);

/** Tabs of the end point detail page. */
export const ENDPOINT_DETAIL_VIEW = {
  ACTIVITY: "activity",
  DOWNLINK: "downlink",
  TRAFFIC: "traffic",
  CONFIGURATION: "configuration",
} as const;

export type EndpointDetailView =
  (typeof ENDPOINT_DETAIL_VIEW)[keyof typeof ENDPOINT_DETAIL_VIEW];

export const ENDPOINT_DETAIL_VIEWS: readonly EndpointDetailView[] =
  Object.values(ENDPOINT_DETAIL_VIEW);

/** ListErrorGroups buckets (KC-Core internal/services/errorgroups/service.go). */
export const ERROR_BUCKET = {
  CONTROL_PLANE: "control_plane",
  BASE_STATION: "base_station",
  ENDPOINT: "endpoint",
  DOWNLINK: "downlink",
} as const;

export type ErrorBucket = (typeof ERROR_BUCKET)[keyof typeof ERROR_BUCKET];

export const ERROR_BUCKETS: readonly ErrorBucket[] =
  Object.values(ERROR_BUCKET);

/** ListEvents outcome values (KC-Core internal/services/grpcservices EventOutcome*). */
export const EVENT_OUTCOME = {
  SUCCESS: "success",
  FAILURE: "failure",
} as const;

export type EventOutcome = (typeof EVENT_OUTCOME)[keyof typeof EVENT_OUTCOME];

/** system_events severities (KC-DB storage/models/system_event.go EventSeverity*). */
export const EVENT_SEVERITY = {
  CRITICAL: "critical",
  ERROR: "error",
  WARNING: "warning",
  INFO: "info",
} as const;

/** system_events categories (KC-DB storage/models/system_event.go EventCategory*). */
export const EVENT_CATEGORY = {
  ENDPOINT: "endpoint",
  BASE_STATION: "basestation",
  MESSAGE: "message",
  SYSTEM: "system",
  SECURITY: "security",
  ROAMING: "roaming",
  ERROR: "error",
  AUDIT: "audit",
  PROTOCOL: "protocol",
  SCACI: "scaci",
  BSSCI: "bssci",
  SESSION: "session",
} as const;
export type EventCategory =
  (typeof EVENT_CATEGORY)[keyof typeof EVENT_CATEGORY];

/**
 * Protocol operations the Events Log filters by, each mapped to the
 * system_events types the Service Center records for it (KC-DB
 * storage/models/system_event.go, KC-Core pkg/bssci/constants.go).
 */
export const EVENT_OPERATION_TYPES = {
  con: ["basestation_online", "basestation_offline", "connection_error"],
  ping: ["basestation_ping_sent", "basestation_ping_answered"],
  reg: ["endpoint.created", "endpoint.updated"],
  dereg: ["endpoint.deleted"],
  epStat: [
    "endpoint_attached",
    "endpoint_detached",
    "endpoint_attach_failed",
    "endpoint_detach_failed",
    "attach_operation_failed",
    "detach_operation_failed",
  ],
  propagate: [
    "attach_propagate_initiated",
    "attach_propagate_completed",
    "attach_propagate_failed",
    "detach_propagate_initiated",
    "detach_propagate_completed",
    "detach_propagate_failed",
  ],
  dlDataQue: [
    "downlink.queued",
    "downlink.updated",
    "dl_data_enqueued",
    "dl_data_updated",
    "dl_data_requeued",
    "dl_data_queue_acknowledged",
  ],
  dlDataRev: [
    "downlink.revoke_requested",
    "dl_data_revoke_initiated",
    "dl_data_revoked",
  ],
  dlDataRes: [
    "dl_data_sent",
    "dl_data_sent_after_expiry",
    "dl_data_expired",
    "dl_data_invalid",
    "dl_data_unknown_result",
    "dl_data_acknowledged",
  ],
  dlRxStat: ["dl_rx_status_received"],
  vm: [
    "vm_activate_success",
    "vm_activate_failed",
    "vm_deactivate_success",
    "vm_status_received",
  ],
  error: ["scaciErr", "session.pending_operation_dropped"],
} as const satisfies Record<string, readonly string[]>;

export type EventOperation = keyof typeof EVENT_OPERATION_TYPES;

/** SCACI command names shown in raw message previews (SCACI §3). */
export const MIOTY_COMMAND = {
  UL_DATA: "ulData",
} as const;

/** Event details key carrying the BSSCI/SCACI opId (KC-DB models.EventDetailKeyOpID). */
export const EVENT_DETAIL_KEY_OP_ID = "opId";

/** Relative time windows the Traffic and Logs filters offer. */
export const TIME_RANGE = {
  ALL: "all",
  LAST_HOUR: "1h",
  LAST_DAY: "24h",
  LAST_WEEK: "7d",
  LAST_MONTH: "30d",
} as const;

export type TimeRange = (typeof TIME_RANGE)[keyof typeof TIME_RANGE];

export const TIME_RANGES: readonly TimeRange[] = Object.values(TIME_RANGE);

/** Lookback of each time window; ALL has no lower bound. */
export const TIME_RANGE_LOOKBACK_MS: Record<TimeRange, number | undefined> = {
  [TIME_RANGE.ALL]: undefined,
  [TIME_RANGE.LAST_HOUR]: MS_PER_HOUR,
  [TIME_RANGE.LAST_DAY]: HOURS_PER_DAY * MS_PER_HOUR,
  [TIME_RANGE.LAST_WEEK]: DAYS_PER_WEEK * HOURS_PER_DAY * MS_PER_HOUR,
  [TIME_RANGE.LAST_MONTH]: DAYS_PER_MONTH_WINDOW * HOURS_PER_DAY * MS_PER_HOUR,
};

/** Direction of a sorted table column. */
export const SORT_DIRECTION = {
  ASC: "asc",
  DESC: "desc",
} as const;

export type SortDirection =
  (typeof SORT_DIRECTION)[keyof typeof SORT_DIRECTION];

/** KeyboardEvent.key values the UI reacts to. */
export const KEYBOARD_KEY = {
  ENTER: "Enter",
} as const;

/** Layout of the shared filter bar (theme.spacing units, px for widths). */
export const FILTER_CONTROL = {
  COMMIT_KEY: KEYBOARD_KEY.ENTER,
  MIN_WIDTH: 160,
  GAP: 2,
} as const;

/** Joins the fields that together identify a table row without an id of its own. */
export const ROW_KEY_SEPARATOR = "|";

/** Layout of the shared data table (theme.spacing units). */
export const DATA_TABLE_LAYOUT = {
  DETAILS_PADDING: 2,
  STATE_PADDING: 3,
  ERROR_MIN_HEIGHT: 160,
  SECTION_GAP: 2,
} as const;

/** Layout of a device's Activity panel (theme.spacing units). */
export const ACTIVITY_PANEL_LAYOUT = {
  TOOLBAR_GAP: 2,
} as const;

/** Layout of raw payload previews (theme.spacing units, px for the height cap). */
export const JSON_PREVIEW = {
  INDENT: 2,
  PADDING: 1.5,
  MAX_HEIGHT: 320,
} as const;

/** Layout of a device's traffic summary card (theme.spacing units, px for the skeleton). */
export const TRAFFIC_SUMMARY_LAYOUT = {
  PADDING: 2,
  ITEM_GAP: 4,
  SKELETON_HEIGHT: 48,
} as const;

/** Layout of protocol flag chips (theme.spacing units). */
export const FLAG_CHIPS_LAYOUT = {
  GAP: 0.5,
} as const;

/** Decimal places shown for radio measurements (snr, rssi, eqSnr). */
export const MEASUREMENT_FRACTION_DIGITS = 1;

/** Decimal places of every other displayed number. */
export const FRACTION_DIGITS = {
  PERCENT: 1,
  DUTY_CYCLE_PERCENT: 2,
  ALTITUDE: 1,
  TEMPERATURE: 1,
  COORDINATE: 6,
  COMPACT_COUNT: 1,
  DOWNLINK_PRIORITY: 2,
  DECODED_VALUE: 2,
} as const;

/** Thresholds and divisors of compact counts (1.2K, 3.4M). */
export const COMPACT_COUNT = {
  THOUSAND: 1_000,
  MILLION: 1_000_000,
} as const;

/** ListErrorGroups always applies a lookback, so it offers bounded windows only. */
export const BOUNDED_TIME_RANGES: readonly TimeRange[] = [
  TIME_RANGE.LAST_HOUR,
  TIME_RANGE.LAST_DAY,
  TIME_RANGE.LAST_WEEK,
  TIME_RANGE.LAST_MONTH,
];

// A new downlink form's text fields (format and prio as SCACI §3.10.1 numbers)
export const DOWNLINK_FORM_DEFAULTS = {
  FORMAT: "0",
  PRIORITY: "0.5",
  PACKET_CNT: "0",
} as const;

/** Named downlink priorities; labels live in DOWNLINK_PRIORITY_LABELS. */
export const DOWNLINK_PRIORITY_PRESET = {
  LOW: "low",
  NORMAL: "normal",
  HIGH: "high",
} as const;
export type DownlinkPriorityPreset =
  (typeof DOWNLINK_PRIORITY_PRESET)[keyof typeof DOWNLINK_PRIORITY_PRESET];

export const DOWNLINK_PRIORITY_PRESETS: readonly {
  preset: DownlinkPriorityPreset;
  value: number;
}[] = [
  { preset: DOWNLINK_PRIORITY_PRESET.LOW, value: 0.0 },
  { preset: DOWNLINK_PRIORITY_PRESET.NORMAL, value: 0.5 },
  { preset: DOWNLINK_PRIORITY_PRESET.HIGH, value: 1.0 },
];

// Realtime event types (must match RealtimeEventType union in types.ts)
export const REALTIME_EVENT_TYPE = {
  UPLINK_RECEIVED: "uplink.received",
  DOWNLINK_QUEUED: "downlink.queued",
  DOWNLINK_UPDATED: "downlink.updated",
  DOWNLINK_SENT: "downlink.sent",
  DOWNLINK_ACKNOWLEDGED: "downlink.acknowledged",
  DOWNLINK_FAILED: "downlink.failed",
  DOWNLINK_REVOKED: "downlink.revoked",
  ENDPOINT_ATTACHED: "endpoint.attached",
  ENDPOINT_DETACHED: "endpoint.detached",
  ENDPOINT_CHANGED: "endpoint.changed",
  BASESTATION_ONLINE: "basestation.online",
  BASESTATION_OFFLINE: "basestation.offline",
  BASESTATION_CHANGED: "basestation.changed",
  SCACI_SESSION_OPENED: "scaci.session.opened",
  SCACI_SESSION_CLOSED: "scaci.session.closed",
  SCACI_ERROR: "scaci.error",
  USER_CHANGED: "user.changed",
  ORGANIZATION_CHANGED: "organization.changed",
  MEMBERSHIP_CHANGED: "membership.changed",
  API_KEY_CHANGED: "api_key.changed",
  SERVER_CERTIFICATE_CHANGED: "server_certificate.changed",
  CATALOG_CHANGED: "catalog.changed",
  EVENT_RECEIVED: "event.received",
} as const;

/** What an event is about: the end point or the base station its source names. */
export const EVENT_SOURCE_KIND = {
  ENDPOINT: "endpoint",
  STATION: "station",
} as const;

export type SourceKind =
  (typeof EVENT_SOURCE_KIND)[keyof typeof EVENT_SOURCE_KIND];

/** States of a realtime stream connection and of the streams a user holds together. */
export const CONNECTION_STATE = {
  DISCONNECTED: "disconnected",
  CONNECTING: "connecting",
  CONNECTED: "connected",
  RECONNECTING: "reconnecting",
} as const;

/** Realtime connection status as the UI presents it. */
export const UI_CONNECTION_STATUS = {
  CONNECTED: "connected",
  RECONNECTING: "reconnecting",
  OFFLINE: "offline",
} as const;

/** Lifecycle notices a realtime stream emits about its own connection. */
export const REALTIME_LIFECYCLE_EVENT = {
  CONNECT: "realtime_connect",
  CONNECTED: "realtime_connected",
  ERROR: "realtime_error",
  RECONNECT: "realtime_reconnect",
  DISCONNECTED: "realtime_disconnected",
} as const;

/**
 * Identifies which underlying gRPC stream a realtime connection event belongs
 * to. The catch-up hook refreshes what a stream carries when that stream
 * reconnects after a drop.
 */
export const REALTIME_STREAM_KIND = {
  MESSAGE: "message",
  BASE_STATION: "base_station",
  EVENT: "event",
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
  // Attachment (endpoint_attachment + BSSCI attach/detach propagate)
  endpoint_attached: REALTIME_EVENT_TYPE.ENDPOINT_ATTACHED,
  endpoint_detached: REALTIME_EVENT_TYPE.ENDPOINT_DETACHED,
  attPrp: REALTIME_EVENT_TYPE.ENDPOINT_ATTACHED,
  attach_propagate_completed: REALTIME_EVENT_TYPE.ENDPOINT_ATTACHED,
  attach_propagate_failed: REALTIME_EVENT_TYPE.ENDPOINT_CHANGED,
  detach_propagate_completed: REALTIME_EVENT_TYPE.ENDPOINT_DETACHED,
  detach_propagate_failed: REALTIME_EVENT_TYPE.ENDPOINT_CHANGED,
  // Base station connectivity
  basestation_online: REALTIME_EVENT_TYPE.BASESTATION_ONLINE,
  basestation_offline: REALTIME_EVENT_TYPE.BASESTATION_OFFLINE,
  connection_error: REALTIME_EVENT_TYPE.BASESTATION_CHANGED,
  // Service center pings and the station's answers (BSSCI §5.4)
  basestation_ping_sent: REALTIME_EVENT_TYPE.BASESTATION_CHANGED,
  basestation_ping_answered: REALTIME_EVENT_TYPE.BASESTATION_CHANGED,
  // The station's answer to an operator's status request (BSSCI §5.5)
  basestation_status_answered: REALTIME_EVENT_TYPE.BASESTATION_CHANGED,
  // Registration changes
  "basestation.registered": REALTIME_EVENT_TYPE.BASESTATION_CHANGED,
  "basestation.updated": REALTIME_EVENT_TYPE.BASESTATION_CHANGED,
  "basestation.deregistered": REALTIME_EVENT_TYPE.BASESTATION_CHANGED,
  "certificate.generated": REALTIME_EVENT_TYPE.BASESTATION_CHANGED,
  "endpoint.created": REALTIME_EVENT_TYPE.ENDPOINT_CHANGED,
  "endpoint.updated": REALTIME_EVENT_TYPE.ENDPOINT_CHANGED,
  "endpoint.deleted": REALTIME_EVENT_TYPE.ENDPOINT_CHANGED,
  // Downlink lifecycle (KC-Core internal/services/bssci/downlink_events.go)
  dl_data_enqueued: REALTIME_EVENT_TYPE.DOWNLINK_QUEUED,
  dl_data_updated: REALTIME_EVENT_TYPE.DOWNLINK_UPDATED,
  dl_data_requeued: REALTIME_EVENT_TYPE.DOWNLINK_UPDATED,
  dl_data_queue_acknowledged: REALTIME_EVENT_TYPE.DOWNLINK_QUEUED,
  dl_data_sent: REALTIME_EVENT_TYPE.DOWNLINK_SENT,
  dl_data_acknowledged: REALTIME_EVENT_TYPE.DOWNLINK_ACKNOWLEDGED,
  dl_data_expired: REALTIME_EVENT_TYPE.DOWNLINK_FAILED,
  dl_data_invalid: REALTIME_EVENT_TYPE.DOWNLINK_FAILED,
  dl_data_unknown_result: REALTIME_EVENT_TYPE.DOWNLINK_FAILED,
  dl_data_revoke_initiated: REALTIME_EVENT_TYPE.DOWNLINK_REVOKED,
  dl_data_revoked: REALTIME_EVENT_TYPE.DOWNLINK_REVOKED,
  dl_rx_status_received: REALTIME_EVENT_TYPE.DOWNLINK_UPDATED,
  // Operator downlink actions (downlink API handlers)
  "downlink.queued": REALTIME_EVENT_TYPE.DOWNLINK_QUEUED,
  "downlink.updated": REALTIME_EVENT_TYPE.DOWNLINK_UPDATED,
  "downlink.revoke_requested": REALTIME_EVENT_TYPE.DOWNLINK_REVOKED,
  // SCACI protocol errors (models.EventTypeSCACIError)
  scaciErr: REALTIME_EVENT_TYPE.SCACI_ERROR,
  // Application center session lifecycle (models.EventTypeSCACISession*)
  "scaci.session_opened": REALTIME_EVENT_TYPE.SCACI_SESSION_OPENED,
  "scaci.session_resumed": REALTIME_EVENT_TYPE.SCACI_SESSION_OPENED,
  "scaci.session_closed": REALTIME_EVENT_TYPE.SCACI_SESSION_CLOSED,
  "scaci.connect_refused": REALTIME_EVENT_TYPE.SCACI_ERROR,
  // Administration (audit category; KC-Identity and the certificate handlers)
  "user.created": REALTIME_EVENT_TYPE.USER_CHANGED,
  "user.updated": REALTIME_EVENT_TYPE.USER_CHANGED,
  "user.deleted": REALTIME_EVENT_TYPE.USER_CHANGED,
  "user.registered": REALTIME_EVENT_TYPE.USER_CHANGED,
  "organization.created": REALTIME_EVENT_TYPE.ORGANIZATION_CHANGED,
  "organization.updated": REALTIME_EVENT_TYPE.ORGANIZATION_CHANGED,
  "organization.deleted": REALTIME_EVENT_TYPE.ORGANIZATION_CHANGED,
  "organization.member_added": REALTIME_EVENT_TYPE.MEMBERSHIP_CHANGED,
  "organization.member_updated": REALTIME_EVENT_TYPE.MEMBERSHIP_CHANGED,
  "organization.member_removed": REALTIME_EVENT_TYPE.MEMBERSHIP_CHANGED,
  "api_key.created": REALTIME_EVENT_TYPE.API_KEY_CHANGED,
  "api_key.deleted": REALTIME_EVENT_TYPE.API_KEY_CHANGED,
  "certificate.server_generated":
    REALTIME_EVENT_TYPE.SERVER_CERTIFICATE_CHANGED,
  "certificate.server_renewed": REALTIME_EVENT_TYPE.SERVER_CERTIFICATE_CHANGED,
  // Device catalog (endpoint category; KC-Core catalog handlers)
  "manufacturer.created": REALTIME_EVENT_TYPE.CATALOG_CHANGED,
  "manufacturer.updated": REALTIME_EVENT_TYPE.CATALOG_CHANGED,
  "manufacturer.deleted": REALTIME_EVENT_TYPE.CATALOG_CHANGED,
  "device_model.created": REALTIME_EVENT_TYPE.CATALOG_CHANGED,
  "device_model.updated": REALTIME_EVENT_TYPE.CATALOG_CHANGED,
  "device_model.deleted": REALTIME_EVENT_TYPE.CATALOG_CHANGED,
  "blueprint.created": REALTIME_EVENT_TYPE.CATALOG_CHANGED,
  "blueprint.updated": REALTIME_EVENT_TYPE.CATALOG_CHANGED,
  "blueprint.deleted": REALTIME_EVENT_TYPE.CATALOG_CHANGED,
} as const;

/** Blueprint decode status of an uplink (messages.decode_status, KC-Core pkg/blueprint); only a success carries a payload to show. */
export const DECODE_STATUS = {
  PENDING: "pending",
  SUCCESS: "success",
  FAILED: "failed",
  SKIPPED: "skipped",
} as const;

/** Hex characters of an uplink payload an Activity caption shows before it is cut short. */
export const UPLINK_CAPTION_PAYLOAD_HEX_CHARS = 64;

/** Kinds of a row of a device's Activity table: a system event or a stored uplink. */
export const ACTIVITY_KIND = {
  EVENT: "event",
  UPLINK: "uplink",
} as const;

/** Page token of the first page of a forward-only cursor listing. */
export const CURSOR_FIRST_PAGE_TOKEN = "";

/** Detail keys of a system event's JSON data (KC-DB models EventDetailKey*). */
export const EVENT_DETAIL_KEY = {
  BS_EUI: "bsEui",
  EP_EUI: "epEui",
} as const;

// Certificate download types (used by DownloadCertificate RPC)
/** Keys of the download URLs IssueCertificate returns (KC-Core certificates issuance). */
export const CERTIFICATE_BUNDLE_KEY = {
  CA_CERT: "ca_cert",
  CLIENT_CERT: "client_cert",
  PRIVATE_KEY: "private_key",
} as const;

export const CERTIFICATE_DOWNLOAD_TYPES = {
  CA: "ca",
  CLIENT: "client",
  KEY: "key",
} as const;

export type CertificateDownloadType =
  (typeof CERTIFICATE_DOWNLOAD_TYPES)[keyof typeof CERTIFICATE_DOWNLOAD_TYPES];

// Formats of the base station uplink export (ExportBaseStationMessages).
export const UPLINK_EXPORT_FORMAT = {
  CSV: "csv",
  JSON: "json",
} as const;

export type UplinkExportFormat =
  (typeof UPLINK_EXPORT_FORMAT)[keyof typeof UPLINK_EXPORT_FORMAT];

/** KC-Core settings holding the BSSCI URL base stations are given (pkg/config ProtocolConfig.BSCIExternalURL). */
export const SERVICE_CENTER_URL_SETTING = {
  CONFIG_KEY: "protocol.bsci_external_url",
  ENV_VAR: "KILOCENTER_PROTOCOL_BSCI_EXTERNAL_URL",
} as const;

// Base station fields the copy buttons report as copied (useClipboard).
export const BS_COPY_FIELDS = {
  EUI: "eui",
  SC_URL: "url",
} as const;

/**
 * API key type identifiers.
 * Matches backend api_key key_type column values.
 */
export const API_KEY_TYPES = {
  USER: "user",
  SERVICE_ACCOUNT: "service_account",
} as const;

/**
 * API key list page size.
 * Larger than default pagination since API keys are lightweight.
 */
export const API_KEY_PAGE_SIZE = 100;

// Registry defaults
export const REGISTRY_DEFAULTS = {
  BRANCH: "main",
} as const;

/** Capability names ListCapabilities publishes (KC-Core internal/services/capabilities). */
export const SERVER_CAPABILITY = {
  BLUEPRINT_REGISTRY_SUBMISSION: "blueprint_registry_submission",
} as const;

/** Blueprint catalogs: the server-wide System catalog and the tenant's Custom one. */
export const BLUEPRINT_SCOPE = {
  SYSTEM: "system",
  CUSTOM: "custom",
} as const;

/** Query parameters of the catalog pages: the open catalog and the manufacturer a new model is for. */
export const BLUEPRINT_QUERY_PARAMS = {
  SCOPE: "scope",
  MANUFACTURER: "manufacturer",
} as const;

/** Views of the Users & Roles page; the query parameter selects one. */
export const USERS_VIEW = {
  SYSTEM: "system",
  ORGANIZATION: "organization",
} as const;

export type UsersView = (typeof USERS_VIEW)[keyof typeof USERS_VIEW];

export const USERS_VIEW_QUERY_PARAM = "tab";

/** Views of an organization's detail page. */
export const ORGANIZATION_VIEW = {
  OVERVIEW: "overview",
  CONFIGURATION: "configuration",
} as const;

export type OrganizationView =
  (typeof ORGANIZATION_VIEW)[keyof typeof ORGANIZATION_VIEW];

export const ORGANIZATION_VIEWS: readonly OrganizationView[] =
  Object.values(ORGANIZATION_VIEW);

/** Rows shown by the blueprint specification editors. */
export const BLUEPRINT_SPEC_ROWS = {
  PAGE: 12,
  DIALOG: 10,
  DETAIL: 20,
} as const;

/**
 * Local storage keys used across the application.
 * Centralized to prevent typos and enable refactoring.
 */
export const STORAGE_KEYS = {
  AUTH_TOKEN: "auth_token",
  REFRESH_TOKEN: "kc_refresh_token",
  USER_PROFILE: "kc_user_profile", // Persisted user profile for session hydration
  FILTERS: "kc-filters",
} as const;

/** Scope suffix of the filters storage key while no organization is selected. */
export const FILTERS_STORAGE_DEFAULT_SCOPE = "default";

/** Feature flag names, the keys of config/flags/default.json. */
export const FEATURE_FLAG = {
  ENTERPRISE_ORGANIZATIONS: "enterprise_organizations",
} as const;

/**
 * Web Locks API names. A lock is shared by every tab of the origin, so holding
 * one excludes the other tabs.
 */
export const LOCK_NAMES = {
  TOKEN_REFRESH: "kc_token_refresh",
  // Held for its life by the one tab that owns the realtime streams.
  REALTIME_LEADER: "kc_realtime_leader",
  // Each tab holds this prefix plus its id for its life, so the leader learns when it closes.
  REALTIME_TAB_PREFIX: "kc_realtime_tab_",
} as const;

/** BroadcastChannel names; a channel reaches every tab of the origin. */
export const BROADCAST_CHANNEL_NAMES = {
  REALTIME: "kc_realtime",
} as const;

/**
 * Messages between the tab that owns the realtime streams and the other tabs:
 * a tab's demand for streams, the new leader's census of those demands, and
 * the streamed events, connection notices, errors and states it relays.
 */
export const REALTIME_RELAY_MESSAGE = {
  DEMAND: "demand",
  CENSUS: "census",
  EVENT: "event",
  CONNECTION: "connection",
  ERROR: "error",
  STATES: "states",
} as const;

/** Random bytes of a tab's id in the realtime relay. */
export const REALTIME_TAB_ID_BYTES = 16;

/**
 * How one access-token refresh ended. Only REFUSED, the server rejecting the
 * refresh token, ends the session; UNAVAILABLE keeps the tokens for a retry.
 */
export const TOKEN_REFRESH_OUTCOME = {
  RENEWED: "renewed",
  REFUSED: "refused",
  UNAVAILABLE: "unavailable",
} as const;

/**
 * HTTP header names used for gRPC-web metadata (matches KC-Core interceptors).
 * Ensures frontend-backend header consistency for tenant isolation.
 */
export const HEADERS = {
  AUTHORIZATION: "Authorization",
  X_ORGANIZATION_ID: "X-Organization-ID",
  X_USER_ID: "X-User-ID",
  // Present in response headers only when the call ended before any message.
  GRPC_STATUS: "grpc-status",
} as const;

/**
 * HTTP header values - centralized to avoid literals in api.ts
 */
export const HEADER_VALUES = {
  AUTH_BEARER_PREFIX: "Bearer ",
} as const;

// Default Values
export const DEFAULT_ORG_NAME = "Default Organization";

/**
 * Environment badge labels.
 * Centralized to avoid inline string literals in EnvBadge.tsx.
 */
export const ENVIRONMENT = {
  DEVELOPMENT: "development",
  STAGING: "staging",
  PRODUCTION: "production",
} as const;

/** Substrings of the gRPC URL that identify a staging deployment. */
export const STAGING_URL_MARKERS = ["staging", "stage", "stg"] as const;

export const ENV_LABELS = {
  development: "DEV",
  staging: "STG",
  production: "PROD",
} as const;

/**
 * Environment badge tooltips.
 * Centralized to avoid inline string literals in EnvBadge.tsx.
 */
export const ENV_TOOLTIPS = {
  development: "Development Environment",
  staging: "Staging Environment",
  production: "Production Environment",
} as const;

/**
 * External IdP claim path for organization ID extraction from JWT.
 * Default matches external IdP claim structure; override via VITE_EXTERNAL_ORG_CLAIM_PATH for other IdPs.
 */
export const DEFAULT_EXTERNAL_ORG_CLAIM_PATH = "urn:zitadel:iam:org:project:id";

/**
 * Certificate validity options in days
 * Matches BSSCI certificate generation requirements
 */
export const CERT_VALIDITY_DAYS = {
  THREE_YEARS: 1095,
} as const;

/** Days-until-expiry thresholds that flag a base station certificate in lists. */
export const CERT_EXPIRY_THRESHOLD_DAYS = {
  WARNING: 60,
  CRITICAL: 30,
} as const;

/**
 * Auth layout constants for login/callback pages.
 * Centralizes spacing and sizing to avoid inline literals.
 */
export const AUTH_LAYOUT = {
  FULL_HEIGHT: "100vh",
  CARD_PADDING: 4,
  CARD_MAX_WIDTH: 400,
  SPACING_MT: 2,
  SPACING_MB: 2,
  SPACING_ML: 2,
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

/** Layout of the end point detail page (theme.spacing units). */
export const ENDPOINT_DETAIL_LAYOUT = {
  DIVIDER_MARGIN_Y: 1,
} as const;

/**
 * Organization role values.
 * Matches backend organization_constants.go.
 */
export const ORG_ROLE = {
  OWNER: "owner",
  ADMIN: "admin",
  MEMBER: "member",
} as const;

/**
 * Organization member status values.
 * Matches backend organization_constants.go.
 */
export const ORG_MEMBER_STATUS = {
  ACTIVE: "active",
  INVITED: "invited",
  REMOVED: "removed",
} as const;

/**
 * Organization state values.
 * Matches backend organization_constants.go.
 */
export const ORG_STATE = {
  ACTIVE: "active",
  SUSPENDED: "suspended",
  ARCHIVED: "archived",
} as const;

/**
 * User lookup limit for autocomplete.
 * No search API available - client-side filtering.
 */
export const USER_LOOKUP_LIMIT = 1000;

/**
 * Placeholder for email input in organization user forms.
 */
export const ORG_USER_PLACEHOLDER_EMAIL = "user@example.com";

/**
 * Pagination defaults and options
 * Used for consistent table pagination across all list views
 * UI spacing tokens are in src/theme/index.ts:componentSpacing.pagination
 */
export const PAGINATION = {
  DEFAULT_PAGE_SIZE: 25,
  PAGE_SIZE_OPTIONS: [10, 25, 50, 100] as const,
  DOWNLINK_FLUSH_PAGE_SIZE: 100,
  DL_RX_STATUS_PAGE_SIZE: 20,
  ORG_PICKER_PAGE_SIZE: 200,
  ADMIN_LIST_PAGE_SIZE: 50,
} as const;

/**
 * UI truncation constants for data display
 * Used for payload preview, config display, etc.
 */
export const TRUNCATION = {
  CONFIG_PREVIEW_LENGTH: 100,
  ELLIPSIS: "...",
} as const;

/**
 * Endpoint update field mask paths for gRPC partial updates.
 * Mirrors backend fieldMask* constants in kilocenter_service.go.
 */
export const ENDPOINT_UPDATE_MASK_PATHS = {
  NAME: "name",
  DESCRIPTION: "description",
  EP_CLASS: "ep_class",
  STATUS: "status",
  SH_ADDR: "sh_addr",
  ATTACH_CNT: "attach_cnt",
  DUAL_CHAN: "dual_chan",
  REPETITION: "repetition",
  WIDE_CARR_OFF: "wide_carr_off",
  LONG_BLK_DIST: "long_blk_dist",
  PRE_ATTACH: "pre_attach",
  LAST_PACKET_CNT: "last_packet_cnt",
  CARRIER_OFFSET: "carrier_offset",
  NWK_SN_KEY: "nwk_sn_key",
  APP_KEY: "app_key",
  DEVICE_MODEL_ID: "device_model_id",
  TYPE_EUI: "type_eui",
} as const;

/**
 * User update field mask paths for gRPC partial updates.
 * Mirrors backend userField* constants in kilocenter_service_auth.go.
 */
export const USER_UPDATE_MASK_PATHS = {
  EMAIL: "email",
  IS_ADMIN: "is_admin",
  IS_ACTIVE: "is_active",
  IS_TENANT_MANAGER: "is_tenant_manager",
  IS_BASE_STATION_MANAGER: "is_base_station_manager",
  IS_ENDPOINT_MANAGER: "is_endpoint_manager",
  NOTE: "note",
} as const;

/**
 * Organization user update field mask paths for gRPC partial updates.
 * Mirrors backend orgUserField* constants in kilocenter_service_admin.go.
 */
export const ORG_USER_UPDATE_MASK_PATHS = {
  ROLE: "role",
  IS_ORG_ADMIN: "is_org_admin",
  IS_BASE_STATION_ADMIN: "is_base_station_admin",
  IS_ENDPOINT_ADMIN: "is_endpoint_admin",
} as const;

/**
 * Endpoint class constants per BSSCI specification.
 * 'A' = Active (bidirectional), 'Z' = Zero-power (unidirectional).
 */
export const ENDPOINT_EP_CLASS = {
  BIDIRECTIONAL: "A",
  UNIDIRECTIONAL: "Z",
} as const;

/**
 * End point attach states: the ep_status values KC-Core reports (attached,
 * detached, attaching) plus the states the web derives when none is reported.
 */
export const ENDPOINT_ATTACH_STATUS = {
  ATTACHED: "attached",
  DETACHED: "detached",
  ATTACHING: "attaching",
  PENDING: "pending",
  UNKNOWN: "unknown",
} as const;

export type EndpointAttachStatus =
  (typeof ENDPOINT_ATTACH_STATUS)[keyof typeof ENDPOINT_ATTACH_STATUS];

/**
 * MIOTY protocol validation constants per BSSCI v1.0.0 §3.8.1 and SCACI §3.6.1.
 * Shared across Add/Edit endpoint dialogs.
 */
/** Sizes of the text encodings the UI reads and writes. */
export const ENCODING = {
  HEX_RADIX: 16,
  HEX_DIGITS_PER_BYTE: 2,
  BASE64_BLOCK_LENGTH: 4,
  JWT_SEGMENT_COUNT: 3,
} as const;

/** Source text of a JSON number without fraction or exponent (RFC 8259 §6 int). */
export const JSON_INTEGER_SOURCE_REGEX = /^-?\d+$/;

export const MIOTY_KEY_BYTE_LENGTH = 16;
export const MIOTY_SHORT_ADDR_MAX = 0xffff;
/** Hex digits a short address is shown with. */
export const MIOTY_SHORT_ADDR_HEX_DIGITS = 4;
export const MIOTY_UINT32_MAX = 4294967295;

/** SCACI §3.10.1 dlDataQue bounds: format is uint8, packet counters uint32. */
export const DOWNLINK_LIMITS = {
  FORMAT_MAX: 255,
  COUNTER_MAX: MIOTY_UINT32_MAX,
  PRIORITY_STEP: 0.1,
} as const;
export const MIOTY_EUI_REGEX = /^[0-9A-Fa-f]{16}$/;
export const MIOTY_KEY_REGEX = /^[0-9A-Fa-f]{32}$/;
export const MIOTY_SHORT_ADDR_REGEX = /^[0-9A-Fa-f]{1,4}$/;

/** Range of EndPoint.carrier_offset (proto int32), an informational value in Hz. */
export const ENDPOINT_CARRIER_OFFSET_RANGE = {
  MIN: -2147483648,
  MAX: 2147483647,
} as const;
export const ENDPOINT_CARRIER_OFFSET_REGEX = /^-?\d+$/;
/** The stored carrier offset of an end point that records none. */
export const ENDPOINT_CARRIER_OFFSET_NONE = 0;

/** Activity threshold in hours — endpoints seen within this window are "active".
 * Matches backend DefaultEndpointActivityWindowHours. */
export const ENDPOINT_ACTIVITY_WINDOW_HOURS = 24;

/** Whether an end point was heard within ENDPOINT_ACTIVITY_WINDOW_HOURS. */
export const ENDPOINT_ACTIVITY = {
  ACTIVE: "active",
  INACTIVE: "inactive",
} as const;

export type EndpointActivity =
  (typeof ENDPOINT_ACTIVITY)[keyof typeof ENDPOINT_ACTIVITY];

/** Base station location sources (KC-DB models LocationSource*). */
export const LOCATION_SOURCE = {
  GPS: "gps",
  MANUAL: "manual",
} as const;

export type LocationSource =
  (typeof LOCATION_SOURCE)[keyof typeof LOCATION_SOURCE];

/** What a base station's stored location says: a GPS fix, GPS without a fix, a manual position, or nothing. */
export const STATION_LOCATION_STATE = {
  GPS_FIX: "gpsFix",
  GPS_NO_FIX: "gpsNoFix",
  MANUAL: "manual",
  NONE: "none",
} as const;

export type StationLocationState =
  (typeof STATION_LOCATION_STATE)[keyof typeof STATION_LOCATION_STATE];

/** Outcome of commissioning: the station and its certificates, or the station only. */
export const COMMISSION_RESULT_STATUS = {
  COMPLETE: "complete",
  PARTIAL: "partial",
} as const;

export type CommissionResultStatus =
  (typeof COMMISSION_RESULT_STATUS)[keyof typeof COMMISSION_RESULT_STATUS];

/** Steps of the add-base-station wizard. */
export const COMMISSIONING_PHASE = {
  INPUT: "input",
  PARTIAL: "partial",
  SUCCESS: "success",
} as const;

export type CommissioningPhase =
  (typeof COMMISSIONING_PHASE)[keyof typeof COMMISSIONING_PHASE];

/** Geographic coordinate validation bounds */
export const GEO_BOUNDS = {
  LATITUDE_MIN: -90,
  LATITUDE_MAX: 90,
  LONGITUDE_MIN: -180,
  LONGITUDE_MAX: 180,
} as const;

/** Map defaults for location picker and detail page map card.
 * TILE_URL / TILE_ATTRIBUTION are the keyless OpenStreetMap standard tiles
 * (light use under the OSMF tile policy); a deployment serving many users
 * sets VITE_MAP_TILE_URL / VITE_MAP_TILE_ATTRIBUTION to its own provider.
 * config/env.ts resolves env.mapTileUrl / env.mapTileAttribution with these. */
export const MAP_DEFAULTS = {
  CENTER_LAT: 0,
  CENTER_LNG: 0,
  ZOOM_DEFAULT: 2,
  ZOOM_LOCATION: 10,
  TILE_URL: "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
  TILE_ATTRIBUTION:
    '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
  PICKER_HEIGHT: 400,
  OVERVIEW_HEIGHT: 360,
  /** Pixels kept free around fitted pins so none sits on the map edge. */
  FIT_PADDING: 32,
  PIN_RADIUS: 8,
  PIN_STROKE_WEIGHT: 2,
  PIN_FILL_OPACITY: 0.9,
} as const;

/** Hex digits in an 8-byte EUI. */
export const MIOTY_EUI_HEX_LENGTH = 16;

/** Rendered in place of an absent EUI. */
export const EUI_PLACEHOLDER = "-";

/** Separator between the byte pairs of a displayed EUI: none, the compact form the service center writes into event text and scopes. */
export const EUI_DISPLAY_SEPARATOR = "";

/** A timestamp in a file name: the ISO characters file systems refuse, and their stand-in. */
export const FILE_TIMESTAMP = {
  UNSAFE: /[:.]/g,
  SEPARATOR: "-",
} as const;

/** Longest EUI an input accepts: byte pairs split by one-character separators (XX-XX-XX-XX-XX-XX-XX-XX). */
export const EUI_INPUT_MAX_LENGTH =
  MIOTY_EUI_HEX_LENGTH + MIOTY_EUI_HEX_LENGTH / 2 - 1;

/** A typed DD/MM/YYYY date: its digits and where the separators go. */
export const DATE_INPUT_EU = {
  DIGITS: 8,
  DAY_END: 2,
  MONTH_END: 4,
  SEPARATOR: "/",
} as const;

/** Characters of a git commit hash shown where the full hash would not fit. */
export const SHORT_COMMIT_SHA_LENGTH = 7;

/** Error catalog tokens the UI branches on (KC-Core/pkg/grpc/errors_catalog.go). */
export const GRPC_ERROR_TOKEN = {
  CANNOT_REMOVE_SELF: "KC-GRPC-ERR-306",
  CANNOT_REMOVE_LAST_OWNER: "KC-GRPC-ERR-308",
} as const;

/** Message browsers attach to a TypeError when the network layer fails. */
export const BROWSER_NETWORK_FAILURE_MESSAGE = "Failed to fetch";

/** HTTP-like statuses the transport maps gRPC codes onto (see GRPC_TO_HTTP_STATUS). */
export const HTTP_STATUS = {
  OK: 200,
  BAD_REQUEST: 400,
  UNAUTHORIZED: 401,
  FORBIDDEN: 403,
  NOT_FOUND: 404,
  CONFLICT: 409,
  PRECONDITION_FAILED: 412,
  TOO_MANY_REQUESTS: 429,
  CLIENT_CLOSED_REQUEST: 499,
  INTERNAL_SERVER_ERROR: 500,
  NOT_IMPLEMENTED: 501,
  SERVICE_UNAVAILABLE: 503,
  GATEWAY_TIMEOUT: 504,
} as const;

/** Refresh cadence (ms) of the SCACI ping times, which no realtime event carries. */
export const TIMING_LIVE_POLL = {
  CONTROL_PLANE_MS: 10000,
} as const;

/** The SCACI session states (models.SCACISessionStatus*) of a live session. */
export const SCACI_SESSION_STATUS = {
  ACTIVE: "active",
  RESUMED: "resumed",
} as const;

/** Severities ListAlerts accepts and the alert summary counts (alerts.AlertSeverities). */
export const ALERT_FILTER_SEVERITIES = [
  EVENT_SEVERITY.CRITICAL,
  EVENT_SEVERITY.ERROR,
  EVENT_SEVERITY.WARNING,
] as const;

/** Aggregate health GetSystemStatus reports when it lists no services. */
export const SYSTEM_HEALTH_STATUS = {
  HEALTHY: "healthy",
} as const;

/** Whether a form dialog creates a record or edits an existing one. */
export const DIALOG_MODE = {
  ADD: "add",
  EDIT: "edit",
} as const;

export type DialogMode = (typeof DIALOG_MODE)[keyof typeof DIALOG_MODE];

/** Color scheme of the UI, persisted in a cookie and defaulted from the OS preference. */
export const THEME_MODE = {
  LIGHT: "light",
  DARK: "dark",
} as const;

export type ThemeMode = (typeof THEME_MODE)[keyof typeof THEME_MODE];

export const THEME_PREFERENCE = {
  COOKIE_NAME: "kc-web-theme-mode",
  COOKIE_MAX_AGE_DAYS: 365,
  COOKIE_PATH: "/",
  PREFERS_DARK_QUERY: "(prefers-color-scheme: dark)",
} as const;

/** A base station's session state as ListBaseStations reports it. */
export const BASE_STATION_STATUS = {
  ONLINE: "online",
  OFFLINE: "offline",
} as const;

export type BaseStationStatus =
  (typeof BASE_STATION_STATUS)[keyof typeof BASE_STATION_STATUS];

/** How a base station reaches the service center: the wire value and its display label. */
export const BS_CONNECTION_TYPE = {
  BSSCI: "bssci",
  MQTT: "mqtt",
} as const;

export const BS_CONNECTION_TYPE_LABEL = {
  BSSCI: "BSSCI",
  MQTT: "MQTT",
} as const;

export type BsConnectionType =
  (typeof BS_CONNECTION_TYPE)[keyof typeof BS_CONNECTION_TYPE];

export type BsConnectionTypeLabel =
  (typeof BS_CONNECTION_TYPE_LABEL)[keyof typeof BS_CONNECTION_TYPE_LABEL];

/** Availability card: a day of hourly, UTC-aligned buckets. */
export const BS_AVAILABILITY = {
  WINDOW_HOURS: 24,
  BUCKET_SECONDS: 3600,
} as const;

/** Certificates a base station detail may download: public material only, never the key. */
export const BS_PUBLIC_CERTIFICATE_TYPES = [
  CERTIFICATE_DOWNLOAD_TYPES.CA,
  CERTIFICATE_DOWNLOAD_TYPES.CLIENT,
] as const;

export type PublicCertificateType =
  (typeof BS_PUBLIC_CERTIFICATE_TYPES)[number];

/**
 * Where a certificate stands against CERT_EXPIRY_THRESHOLD_DAYS; a station
 * pinned to a certificate whose expiry is not stored yet is EXPIRY_NOT_RECORDED.
 */
export const CERTIFICATE_EXPIRY_STATE = {
  NOT_ISSUED: "not_issued",
  EXPIRY_NOT_RECORDED: "expiry_not_recorded",
  VALID: "valid",
  EXPIRING: "expiring",
  CRITICAL: "critical",
  EXPIRED: "expired",
} as const;

export type CertificateExpiryState =
  (typeof CERTIFICATE_EXPIRY_STATE)[keyof typeof CERTIFICATE_EXPIRY_STATE];

/** Page size of the dashboard alert list. */
export const ALERT_TABLE_PAGE_SIZE = 10;

export const PERCENT_SCALE = 100;
