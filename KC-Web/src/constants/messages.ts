/**
 * User-Facing Message Constants
 * Centralized error/validation/success messages for KC-Web
 *
 * IMPORTANT: All setError() calls must use these constants
 * Never hardcode user-facing strings in components
 *
 * Naming convention:
 * - ERR_* = Error messages
 * - VAL_* = Validation messages
 * - MSG_* = Informational messages
 *
 */

import type {
  BaseStationDetailView,
  DownlinkPriorityPreset,
  EndpointDetailView,
  ErrorBucket,
  EventCategory,
  EventOperation,
  EventOutcome,
  LogView,
  TimeRange,
  TrafficView,
} from "@constants/app";

// ============================================================================
// AUTH/LOGIN MESSAGES
// ============================================================================

// Labels
export const LABEL_EMAIL = "Email";
export const LABEL_PASSWORD = "Password";
export const LABEL_FIRST_NAME = "First Name";
export const LABEL_LAST_NAME = "Last Name";
export const LABEL_COMPANY_NAME = "Company Name";
export const LABEL_CONFIRM_PASSWORD = "Confirm Password";

// Auth label aliases for login/register forms
export const AUTH_EMAIL_LABEL = LABEL_EMAIL;
export const AUTH_PASSWORD_LABEL = LABEL_PASSWORD;

// Buttons
export const ACTION_LOGIN = "Log In";
export const ACTION_LOGIN_PROVIDER = "Sign in with";
export const ACTION_RETURN_TO_LOGIN = "Return to login";
export const ACTION_CREATE_ACCOUNT = "Create Account";
export const ACTION_HAVE_ACCOUNT = "Already have an account?";
export const ACTION_NO_ACCOUNT = "Don't have an account?";
export const ACTION_CREATE_ONE = "Create one";

// Status messages
export const MSG_AUTH_COMPLETING = "Completing authentication...";

// Error messages
export const ERR_AUTH_SETTINGS_LOAD = "Failed to load authentication settings.";
export const ERR_AUTH_LOGIN_FAILED = "Login failed. Please try again.";
export const ERR_AUTH_INVALID_CREDENTIALS = "Invalid email or password.";
export const ERR_AUTH_ORG_REQUIRED =
  "Authentication failed: no organization membership found.";
export const ERR_AUTH_CALLBACK_MISSING_CODE =
  "Authentication failed: missing authorization code.";
export const ERR_AUTH_CALLBACK_EXCHANGE_FAILED =
  "Authentication failed: could not complete login.";
export const ERR_AUTH_PROFILE_LOAD_FAILED =
  "Authentication failed: could not load user profile.";

// Validation messages
export const VAL_EMAIL_REQUIRED = "Email is required.";
export const VAL_PASSWORD_REQUIRED = "Password is required.";
export const VAL_FIRST_NAME_REQUIRED = "First name is required";
export const VAL_LAST_NAME_REQUIRED = "Last name is required";
export const VAL_COMPANY_NAME_REQUIRED = "Company name is required";
export const VAL_PASSWORD_CONFIRM_MISMATCH = "Passwords do not match";

// Registration error messages
export const ERR_REGISTRATION_FAILED = "Registration failed. Please try again.";

// ============================================================================
// Base Stations - Validation
// ============================================================================

export const VAL_BS_NAME_REQUIRED = "Base Station name is required.";
export const VAL_BS_EUI_FORMAT =
  "Invalid Base Station EUI format. Enter 16 hex digits.";
export const VAL_BS_EUI_REQUIRED = "Base Station EUI is required.";
// ============================================================================
// Base Stations - Errors
// ============================================================================

export const ERR_LOAD_BASE_STATIONS =
  "Failed to load base stations. Please try again.";
export const ERR_LOAD_BS_DETAILS = "Failed to load base station details";
export const ERR_UPDATE_BS_EUI =
  "Failed to update base station EUI. Please try again.";
export const ERR_BS_EUI_EXISTS = "A base station with this EUI already exists.";
export const ERR_BS_NOT_FOUND =
  "Base station not found. It may have been deleted.";
export const ERR_UPDATE_BS_NAME_PARTIAL =
  "Base station EUI updated but name update failed. Please try updating the name again.";
export const ERR_UPDATE_BS = "Failed to update base station. Please try again.";

// ============================================================================
// Base Stations - Success
// ============================================================================

export const MSG_BS_ADDED_WITH_CERTS =
  "Base station added successfully! Download the certificates below to complete setup.";
export const MSG_BS_ADDED_NO_CERTS =
  "Base station added successfully! You can now configure your base station.";
export const MSG_BS_EUI_UPDATED = "Base station EUI updated successfully";
export const MSG_BS_UPDATED = "Base station updated";

// ============================================================================
// Base Stations - Deletion
// ============================================================================

export const MSG_BS_DELETED = "Base station deleted";
export const ERR_DELETE_BASE_STATION =
  "Failed to delete base station. Please try again.";
export const ERR_CREATE_BASE_STATION =
  "Failed to create base station. Please try again.";
export const ERR_CERT_GENERATION_FAILED_RETRY =
  "Certificate generation failed. The base station was created - you can retry certificate generation.";
export const MSG_BS_PARTIAL_SUCCESS =
  "Base station created, but certificate generation failed. Use the retry button to generate certificates.";

// ============================================================================
// Endpoints - Errors
// ============================================================================

export const ERR_LOAD_ENDPOINTS = "Failed to load endpoints";
export const ERR_ATTACH_ENDPOINT =
  "Failed to attach endpoint. Please try again.";
export const ERR_DETACH_ENDPOINT =
  "Failed to detach endpoint. Please try again.";
export const ERR_DELETE_ENDPOINT =
  "Failed to delete endpoint. Please try again.";

// ============================================================================
// Endpoints - Validation
// ============================================================================

// ============================================================================
// Messages - Errors
// ============================================================================

// Endpoint Operation Log Messages (console logging)
export const LOG_ATTACH_FAILED = "Failed to attach endpoint:";
export const LOG_DETACH_FAILED = "Failed to detach endpoint:";
export const LOG_PRE_DELETE_DETACH_FAILED =
  "Pre-delete detach failed, attempting delete anyway:";
export const LOG_DELETE_FAILED = "Failed to delete endpoint:";

// ============================================================================
// Downlink - Errors
// ============================================================================

// ============================================================================
// Downlink - Success
// ============================================================================

export const MSG_DOWNLINK_QUEUED = "Downlink message queued successfully";
export const MSG_DOWNLINK_REVOKED = "Downlink message revoked successfully";
export const MSG_QUEUE_FLUSHED = "Revoked {revoked} queued downlink(s)";
export const MSG_QUEUE_FLUSH_EMPTY = "No queued downlink was left to revoke";
export const ERR_FLUSH_PARTIAL =
  "Revoked {revoked} queued downlink(s); {failed} could not be revoked";

// Downlink - Labels
export const LABEL_DL_CNT_DEPEND = "Counter Dependent";
export const LABEL_DL_RESPONSE_EXP = "Response Expected";
export const LABEL_DL_RESPONSE_PRIO = "Response Priority";
export const LABEL_DL_WIND_REQ = "DL Window Required";
export const LABEL_DL_EXP_ONLY = "Expedited Only";
export const LABEL_DL_RX_STAT_QRY = "DL RX Status Query";
export const LABEL_DL_ENDPOINT_ACK = "Acknowledged by device";
export const LABEL_DL_PAYLOAD_COUNTER = "Packet counter";
export const LABEL_DL_ACCEPTED = "Accepted";
export const LABEL_DL_LAST_PACKET_CNT =
  "Last packet counter reported by the endpoint:";
export const MSG_DL_AWAITING_ACCEPTANCE = "Awaiting acceptance";
export const MSG_DL_NO_PACKET_CNT = "none yet";
// Downlink - Helper text
export const HELPER_DL_FORMAT = "Payload format identifier (0-255)";
export const HELPER_DL_CNT_DEPEND =
  "Payloads are tied to specific packet counters";
export const HELPER_DL_PACKET_CNT = "Uplink packet counter value (uint32)";

// Downlink - Actions
export const ACTION_SEND_DOWNLINK = "Send Downlink";
export const ACTION_SENDING_DOWNLINK = "Sending...";
export const ACTION_REVOKE = "Revoke";
export const TITLE_REVOKE_DOWNLINK = "Revoke Downlink";
export const ACTION_REVOKING = "Revoking...";
export const ACTION_FLUSH_QUEUE = "Flush Queue";
export const TITLE_FLUSH_QUEUE = "Flush Downlink Queue";
export const ACTION_FLUSHING_QUEUE = "Flushing...";

// Downlink - Confirmations
export const CONFIRM_REVOKE_DOWNLINK = "Revoke this queued downlink message?";
export const CONFIRM_FLUSH_QUEUE =
  "Revoke every pending downlink message these filters match?";

// Downlink - Section Headers
export const SECTION_COMPOSE_DOWNLINK = "Compose Downlink";
// Downlink - DL RX status (BSSCI §3.15)
export const DL_RX_STATUS = {
  SECTION: "Downlink RX Status",
  HELPER:
    "The serving base station asks the end point at which SNR and RSSI it received its downlinks; the end point answers with a later uplink.",
  ACTION_QUERY: "Query DL RX Status",
  ACTION_QUERYING: "Querying...",
  ERR_QUERY: "The DL RX status query could not be sent",
  LABEL_RX_TIME: "Reported",
  LABEL_PACKET_CNT: "Packet Counter",
  LABEL_SNR: "DL RX SNR (dB)",
  LABEL_RSSI: "DL RX RSSI (dBm)",
  LABEL_REQUESTED_AT: "Requested",
  SECTION_PENDING: "Pending queries",
  MSG_EMPTY_STATUSES: "No DL RX status reported yet",
  MSG_EMPTY_PENDING: "No pending queries",
} as const;

// Downlink - Compose
export const LABEL_DL_ADVANCED_OPTIONS = "Advanced Options";
export const HELPER_DL_PAYLOAD_ACK =
  "Hex-encoded payload (leave empty for ACK-only)";

// Downlink - Counter-dependent mode
export const LABEL_DL_ADD_PAYLOAD_ROW = "Add Payload";
export const LABEL_DL_REMOVE_PAYLOAD_ROW = "Remove";
export const HELPER_DL_CNT_DEPEND_PAYLOADS =
  "Each payload requires a matching packet counter";

// Downlink field validation
export const VAL_FORMAT_RANGE = "Must be 0-255";
export const VAL_PRIORITY_FINITE = "Must be a number";
export const VAL_PACKET_CNT_RANGE = "Must be an integer 0-4294967295";

// Data Format Validation
export const VAL_INVALID_HEX_STRING = "Invalid hex string format";
// Downlink - Display label maps (used by formatters)
export const DOWNLINK_STATUS_LABELS: Record<string, string> = {
  pending: "Pending",
  scheduled: "Scheduled",
  reserved: "Reserved",
  queued: "Queued",
  transmitted: "Transmitted",
  delivered: "Delivered",
  failed: "Failed",
  expired: "Expired",
  revoked: "Revoked",
  acked: "Acknowledged",
};

export const DOWNLINK_PRIORITY_LABELS: Record<DownlinkPriorityPreset, string> =
  {
    low: "Low",
    normal: "Normal",
    high: "High",
  };

export const DOWNLINK_RESULT_LABELS: Record<string, string> = {
  sent: "Sent",
  expired: "Expired",
  invalid: "Invalid",
  revoked: "Revoked",
};

// ============================================================================
// Shared data tables and filters (Traffic, Logs)
// ============================================================================

export const DATA_TABLE = {
  SHOW_DETAILS: "Show details",
  HIDE_DETAILS: "Hide details",
  NO_VALUE: "-",
  UNKNOWN_VALUE: "Unknown",
  EMPTY_PAYLOAD: "(empty)",
  LOAD_FAILED: "Could not load this list",
} as const;

export const FILTER_LABELS = {
  TIME_RANGE: "Time range",
  ANY: "Any",
  IN_FLIGHT: "In flight",
  SEARCH: "Search",
  OPERATION: "Operation",
  CATEGORY: "Category",
  OUTCOME: "Outcome",
  DUPLICATES_ONLY: "duplicate only",
  DL_OPEN_ONLY: "dlOpen only",
  ERR_EUI: "Enter 16 hex digits",
  ERR_INTEGER: "Enter a whole number",
  ERR_POSITIVE_INTEGER: "Enter a positive whole number",
  ERR_NUMBER: "Enter a number",
} as const;

/** SCACI field names, shown verbatim as column headers and filter labels (units in brackets). */
export const SCACI_FIELD = {
  EP_EUI: "epEui",
  BS_EUI: "bsEui",
  OP_ID: "opId",
  QUE_ID: "queId",
  PACKET_CNT: "packetCnt",
  RX_TIME: "rxTime",
  TX_TIME: "txTime",
  USER_DATA: "userData",
  SNR: "snr (dB)",
  RSSI: "rssi (dBm)",
  EQ_SNR: "eqSnr (dB)",
  RX_DURATION: "rxDuration (ns)",
  DL_RX_SNR: "dlRxSnr (dB)",
  DL_RX_RSSI: "dlRxRssi (dBm)",
  PROFILE: "profile",
  MODE: "mode",
  BASE_STATIONS: "baseStations",
  SUBPACKETS: "subpackets",
  FREQUENCY: "frequency (Hz)",
  PHASE: "phase (°)",
  DUPLICATE: "duplicate",
  DL_OPEN: "dlOpen",
  RESPONSE_EXP: "responseExp",
  DL_ACK: "dlAck",
  PRIO: "prio",
  CNT_DEPEND: "cntDepend",
  RESPONSE_PRIO: "responsePrio",
  DL_WIND_REQ: "dlWindReq",
  EXP_ONLY: "expOnly",
  DL_RX_STAT_QRY: "dlRxStatQry",
  RESULT: "result",
  CODE: "code",
  MESSAGE: "message",
  UL_DATA: "ulData",
  FORMAT: "format",
} as const;

export const TIME_RANGE_LABELS: Record<TimeRange, string> = {
  all: "All time",
  "1h": "Last hour",
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "30d": "Last 30 days",
};

// ============================================================================
// Traffic (NAV_STRUCTURE §4); column headers keep the SCACI field names
// ============================================================================

export const TRAFFIC_PAGE = {
  TITLE: "Traffic",
  ARIA_VIEWS: "Traffic views",
} as const;

export const TRAFFIC_VIEW_LABELS: Record<TrafficView, string> = {
  uplinks: "Uplink Events",
  downlinkQueue: "Downlink Queue",
  downlinkResults: "Downlink Results",
};

export const UPLINK_TABLE = {
  EMPTY: "No uplinks match these filters",
  COL_SUBPACKET_INDEX: "#",
  NO_SUBPACKETS: "No subpacket data reported",
  NO_RECEPTIONS: "No reception data stored for this uplink",
  DECODED_PAYLOAD: "Decoded payload",
  PAYLOAD_HEX: "Payload (hex)",
  DECODE_STATUS: "Decode status",
} as const;

export const DOWNLINK_TABLE = {
  COL_QUEUED: "Queued",
  EMPTY_QUEUE: "No downlinks match these filters",
  EMPTY_RESULTS: "No downlink results match these filters",
  ACTION_EDIT: "Edit pending downlink",
} as const;

export const EDIT_DOWNLINK_DIALOG = {
  TITLE: "Edit pending downlink",
  ACTION_SAVE: "Save",
  ACTION_SAVING: "Saving...",
  MSG_UPDATED: "Downlink updated",
} as const;

export const TRAFFIC_SUMMARY = {
  TITLE: "Traffic Summary",
  TOTAL: "Uplinks",
  UNIQUE_ENDPOINTS: "End Points Heard",
  AVG_RSSI: "Average RSSI (dBm)",
  AVG_SNR: "Average SNR (dB)",
  TODAY: "Today",
  THIS_WEEK: "This Week",
  THIS_MONTH: "This Month",
  ACTIVE_DAYS: "Active Days",
  FIRST_SEEN: "First Uplink",
  LAST_SEEN: "Last Uplink",
} as const;

// ============================================================================
// Logs (NAV_STRUCTURE §5)
// ============================================================================

export const LOGS_PAGE = {
  TITLE: "Logs",
  ARIA_VIEWS: "Log views",
  ARIA_ERROR_BUCKETS: "Error buckets",
} as const;

export const LOG_VIEW_LABELS: Record<LogView, string> = {
  events: "Events Log",
  errors: "Errors Center",
  audit: "Audit Log",
};

export const EVENT_LOG_TABLE = {
  COL_TIME: "Time",
  COL_OPERATION: "Event Type",
  COL_SOURCE: "Scope",
  COL_OUTCOME: "Severity",
  COL_SUMMARY: "Summary",
  COL_ACTOR: "Actor",
  COL_ACTION: "Action",
  COL_TARGET: "Target",
  ACTOR_SERVICE: "Service Center",
  EMPTY: "No events match these filters",
  DATA_TITLE: "Event data",
  NO_DATA: "The event carries no data",
} as const;

export const ERROR_GROUP_TABLE = {
  COL_OP_ID: "Last opId",
  COL_EVENT_TYPE: "Event Type",
  COL_RELATED: "Related Object",
  COL_FIRST_SEEN: "First Seen",
  COL_LAST_SEEN: "Last Seen",
  COL_COUNT: "Count",
  EMPTY: "No failures in this time range",
} as const;

export const ERROR_BUCKET_LABELS: Record<ErrorBucket, string> = {
  control_plane: "Control Plane",
  base_station: "Base Stations",
  endpoint: "End Points",
  downlink: "Downlinks",
};

export const EVENT_OPERATION_LABELS: Record<EventOperation, string> = {
  con: "con (connection)",
  ping: "ping",
  reg: "reg (register)",
  dereg: "dereg (deregister)",
  epStat: "epStat (attach / detach)",
  propagate: "attPrp / detPrp",
  dlDataQue: "dlDataQue",
  dlDataRev: "dlDataRev",
  dlDataRes: "dlDataRes",
  dlRxStat: "dlRxStat",
  vm: "vm",
  error: "error",
};

export const EVENT_OUTCOME_LABELS: Record<EventOutcome, string> = {
  success: "Success",
  failure: "Failure",
};

export const EVENT_CATEGORY_LABELS: Record<EventCategory, string> = {
  endpoint: "Endpoint",
  basestation: "Base station",
  message: "Message",
  system: "System",
  security: "Security",
  roaming: "Roaming",
  error: "Error",
  audit: "Audit",
  protocol: "Protocol",
  scaci: "SCACI",
  bssci: "BSSCI",
  session: "Session",
};

// Device detail tabs shared by the base station and endpoint pages
export const DEVICE_DETAIL_TABS = {
  ACTIVITY: "Activity",
  TRAFFIC: "Traffic",
  ARIA_BASE_STATION_TABS: "Base station tabs",
} as const;

export const BASE_STATION_DETAIL_VIEW_LABELS: Record<
  BaseStationDetailView,
  string
> = {
  activity: DEVICE_DETAIL_TABS.ACTIVITY,
  traffic: DEVICE_DETAIL_TABS.TRAFFIC,
};

/** A device's Activity table: its events and uplinks in the Events Log columns. */
export const ACTIVITY_TABLE = {
  EMPTY: "No activity in this date range",
  UPLINK_TITLE_PREFIX: "Uplink #",
  CAPTION_SEPARATOR: " · ",
  METRIC_SEPARATOR: " ",
  PAYLOAD_TRUNCATED: "...",
  STATION_SEPARATOR: ", ",
  DECODED_PREFIX: "Decoded: ",
  DECODE_FAILED: "Decode failed",
  DECODE_CODE_OPEN: " (",
  DECODE_CODE_CLOSE: ")",
  NOT_DECODED_NO_BLUEPRINT: "Not decoded: no blueprint assigned",
} as const;

// ============================================================================
// Certificates - Errors
// ============================================================================

export const ERR_CERTIFICATE_GENERIC = "An error occurred";
export const ERR_CERT_NETWORK =
  "Cannot connect to the service. Please ensure the KC-Gateway is running on port 9090.";

// ============================================================================
// Storage Service - Messages
// ============================================================================

export const ERR_STORAGE_GET_FAILED = "Failed to get {key} from storage";
export const ERR_STORAGE_SET_FAILED = "Failed to set {key} in storage";
export const ERR_STORAGE_REMOVE_FAILED = "Failed to remove {key} from storage";
export const ERR_STORAGE_CLEAR_FAILED = "Failed to clear storage";

/**
 * Session error messages
 * Used by SessionContext for profile hydration failures
 */
export const FEEDBACK_ERRORS = {
  CONTEXT_REQUIRED: "useFeedback must be used within FeedbackProvider",
} as const;

/** Invariant failures thrown when a hook runs outside its provider or input breaks a contract. */
export const APP_ERRORS = {
  ROOT_ELEMENT_MISSING: "Failed to find the root element",
  SYSTEM_CONTEXT_REQUIRED: "useSystem must be used within SystemProvider",
  ORGANIZATION_CONTEXT_REQUIRED:
    "useOrganization must be used within OrganizationProvider",
  FILTERS_CONTEXT_REQUIRED: "useFilters must be used within a FiltersProvider",
  FEATURE_FLAGS_CONTEXT_REQUIRED:
    "useFeatureFlags must be used within a FeatureFlagProvider",
  THEME_CONTEXT_REQUIRED: "useThemeMode must be used within a KCThemeProvider",
} as const;

/** Console log lines the app writes through utils/logger. */
export const LOG_MESSAGES = {
  JWT_DECODE_FAILED: "Failed to decode JWT payload:",
  VERSION_FETCH_FAILED: "Failed to fetch version info:",
  FILTERS_LOAD_FAILED: "Failed to load filters from storage:",
  FILTERS_SAVE_FAILED: "Failed to save filters to storage:",
  ROUTE_ERROR_PREFIX: "[RouteErrorBoundary]",
  UNKNOWN_ROUTE: "Unknown route",
} as const;

export const SESSION_ERRORS = {
  PROFILE_PARSE_FAILED: "Failed to parse stored user profile, cleared storage",
  CONTEXT_REQUIRED: "useSession must be used within SessionProvider",
  PROACTIVE_REFRESH_REFUSED:
    "Proactive token refresh refused, clearing session",
  PROACTIVE_REFRESH_RETRY:
    "Proactive token refresh unavailable, retrying in ms:",
  TOKEN_REFRESH_FAILED: "Token refresh failed with outcome:",
  REFRESH_TOKEN_MISSING: "No refresh token is stored",
} as const;

// ============================================================================
// API / Network - Errors (used by query client error mapping)
// ============================================================================

export const ERR_NETWORK_ERROR =
  "Network connection failed. Please check your connection.";
export const ERR_GENERIC_ERROR =
  "An unexpected error occurred. Please try again.";
export const ERR_TIMEOUT_ERROR = "Request timed out. Please try again.";
export const ERR_SERVICE_UNREACHABLE =
  "The service is not reachable right now. Please ensure the KC-Core backend is running and try again.";
export const ERR_UNAUTHORIZED =
  "Your session has expired. Please log in again.";
export const ERR_FORBIDDEN =
  "You do not have permission to perform this action.";
// ============================================================================
// Form Labels - Base Station
// ============================================================================

export const LABEL_BS_NAME = "Base Station Name";
export const LABEL_BS_EUI = "Base Station EUI";
export const HELPER_BS_NAME = "A friendly name for this base station";
export const HELPER_BS_EUI =
  "Enter the 8-byte EUI of your base station (e.g., 0123456789ABCDEF)";
export const PLACEHOLDER_BS_EUI = "XXXXXXXXXXXXXXXX";

// Location labels and messages
export const LABEL_LATITUDE = "Latitude";
export const LABEL_LONGITUDE = "Longitude";
export const LABEL_ALTITUDE = "Altitude";
export const LABEL_LOCATION = "Location";
export const LABEL_LOCATION_OPTIONAL = "Location (optional)";
export const LABEL_LOCATION_SOURCE_GPS = "GPS";
export const LABEL_LOCATION_SOURCE_MANUAL = "Manual";
export const MSG_NO_LOCATION_SET = "No Base Station Location Set";
export const MSG_NO_GPS_FIX =
  "No GPS fix: the base station reports no position";

export const HELPER_LATITUDE = "Decimal degrees (-90 to 90)";
export const HELPER_LONGITUDE = "Decimal degrees (-180 to 180)";
export const HELPER_ALTITUDE = "Meters above sea level";

export const VAL_LATITUDE_RANGE = "Latitude must be between -90 and 90";
export const VAL_LONGITUDE_RANGE = "Longitude must be between -180 and 180";
export const VAL_LAT_LON_PAIR =
  "Both latitude and longitude are required, or leave both empty";

export const ACTION_PICK_ON_MAP = "Pick on Map";
export const ACTION_CONFIRM_LOCATION = "Confirm";
export const TITLE_MAP_PICKER = "Select Location";
export const INSTR_MAP_PICKER = "Click on the map to place a marker";

export const MSG_GPS_AUTHORITATIVE =
  "Location is reported by the base station's GPS and cannot be edited manually";

// ============================================================================
// Form Labels - Endpoint
// ============================================================================

// ============================================================================
// Form Labels - Downlink
// ============================================================================

export const LABEL_DL_PAYLOAD = "Payload (hex)";
export const LABEL_DL_PRIORITY = "Priority";
export const LABEL_DL_FORMAT = "Format";
// ============================================================================
// UI Strings - Common Actions
// ============================================================================

export const ACTION_CANCEL = "Cancel";
export const ACTION_DELETE = "Delete";
export const ACTION_COPY = "Copy";
export const ACTION_COPIED = "Copied!";
export const ACTION_EDIT = "Edit";

// ============================================================================
// UI Strings - Section Headers
// ============================================================================

export const SECTION_CONFIG = "Configuration";
export const SECTION_BS_CONFIG = "Base Station Configuration";
export const SECTION_DOWNLOAD_CERTS = "Download Certificates";
export const SECTION_NEXT_STEPS = "Next Steps";

// ============================================================================
// UI Strings - Dialog Titles
// ============================================================================

export const TITLE_ADD_BS = "Add Base Station";
export const TITLE_BS_READY = "Base Station Ready";
export const TITLE_BS_PARTIAL = "Certificate Generation Failed";

// ============================================================================
// UI Strings - Field Labels
// ============================================================================

export const LABEL_NAME = "Name:";
export const LABEL_BS_EUI_DISPLAY = "Base Station EUI:";
export const LABEL_SC_URL_DISPLAY = "Service Center URL:";
export const WARN_SC_URL_NOT_CONFIGURED =
  "No Service Center URL that base stations can reach is configured. Set it with";
export const WARN_SC_URL_ENV_ALTERNATIVE = "or the environment variable";

// ============================================================================
// UI Strings - Button Labels
// ============================================================================

export const ACTION_NEXT = "Next";
export const ACTION_CONTINUE = "Continue";
export const ACTION_CREATING = "Creating...";
export const ACTION_RETRY_CERTS = "Retry Certificate Generation";
export const ACTION_RETRYING_CERTS = "Generating Certificates...";
export const ACTION_DOWNLOAD_CA_CERT = "Download CA Certificate";
export const ACTION_DOWNLOAD_TLS_CERT = "Download TLS Certificate";
export const ACTION_DOWNLOAD_TLS_KEY = "Download TLS Key (One-time)";

// ============================================================================
// UI Strings - Commissioning Instructions
// ============================================================================

export const INSTR_STEP_1 =
  "1. Open your base station's configuration interface";
export const INSTR_STEP_2_HEADER = "2. Configure the following settings:";
export const INSTR_STEP_3 =
  "3. Save the configuration and restart your base station";
export const INSTR_TLS_NOTE =
  "The base station will establish a TLS connection to the Service Center automatically.";
export const INSTR_BS_CREATED_FALLBACK =
  "Base station created. Configure your hardware to connect to this Service Center.";

// Certificate file name labels
export const LABEL_CA_CERT_FILE =
  "CA Certificate: kilocenter-ca-certificate.crt";
export const LABEL_CLIENT_CERT_PREFIX = "Client Certificate: basestation-";
export const LABEL_CLIENT_CERT_SUFFIX = "-client-certificate.crt";
export const LABEL_PRIVATE_KEY_PREFIX = "Private Key: basestation-";
export const LABEL_PRIVATE_KEY_SUFFIX = "-private-key.key";

// ============================================================================
// UI Strings - Status & Informational
// ============================================================================

export const INFO_NOTE_PREFIX = "Note:";
export const INFO_TLS_NOTE =
  "TLS certificates will be generated automatically for secure BSSCI communication.";
export const INFO_CERT_EXPIRY =
  "Links expire in 15 minutes. Private key can only be downloaded once.";
// ============================================================================
// UI Strings - Validity Options
// ============================================================================

// ============================================================================
// Connection Status
// ============================================================================

export const STATUS_RECONNECTING = "Realtime connection reconnecting...";
export const STATUS_OFFLINE = "Realtime connection offline";
// ============================================================================
// Add Endpoint Dialog - Form Constants (BSSCI §3.8.1 / SCACI §3.6.1)
// ============================================================================

export const ENDPOINT_FORM = {
  // Dialog
  DIALOG_TITLE: "Add New End Point",
  DIALOG_SUBTITLE:
    "Configure a new MIOTY end point with network keys and propagation settings",

  // Section Headers
  SECTION_BASIC: "Basic Information",
  SECTION_NETWORK: "Network Configuration",
  SECTION_COMMUNICATION: "Communication Settings",
  SECTION_ADVANCED: "Advanced MIOTY Settings",
  SECTION_SECURITY: "Security Keys",
  SECTION_BLUEPRINT: "Blueprint Configuration",
  SECTION_METADATA: "Device Metadata",

  // Field Labels
  LABEL_EUI: "End Point EUI",
  LABEL_NAME: "Name",
  LABEL_SHORT_ADDR: "Short Address (ShAddr)",
  LABEL_CARRIER_OFFSET: "Carrier Offset (informational, Hz)",
  LABEL_TYPE_EUI: "Type EUI",
  LABEL_BIDIRECTIONAL: "Bidirectional (BiDi)",
  LABEL_PRE_ATTACH: "Pre-Attachment (PreAtt)",
  LABEL_DUAL_CHAN: "Dual Channel Mode",
  LABEL_REPETITION: "DL Repetition",
  LABEL_WIDE_CARR_OFF: "Wide Carrier Offset",
  LABEL_LONG_BLK_DIST: "Long DL Interblock Distance",
  LABEL_LAST_PACKET_CNT: "Last Packet Counter",
  LABEL_ATTACH_CNT: "Attach Counter",
  LABEL_NETWORK_KEY: "Network Key",
  LABEL_APP_KEY: "Application Key (Optional)",

  // Helper Text
  HELPER_EUI: "16 hex characters (e.g., 0123456789ABCDEF)",
  HELPER_NAME: "Friendly name for this endpoint",
  // Short address helper (always required per BSSCI §3.8.1)
  PLACEHOLDER_SHORT_ADDR: "0001-FFFF",
  HELPER_SHORT_ADDR:
    "Required. 4 hex chars, nonzero (0001-FFFF). From device provisioning.",
  HELPER_CARRIER_OFFSET:
    "Recorded with the end point only, never sent to base stations. The radio carrier offset range is the Wide Carrier Offset flag.",
  HELPER_TYPE_EUI: "16 hex characters (optional)",
  HELPER_TYPE_EUI_MODEL_OVERRIDE:
    "Type EUI will be resolved from the device model's default blueprint",
  // Packet counter helper (always required per BSSCI §3.8.1)
  HELPER_LAST_PACKET_CNT:
    "Required. Last known uplink counter (0-4294967295). Enter 0 for new devices.",
  // Attach counter helper (always required per SCACI §3.6.1)
  HELPER_ATTACH_CNT:
    "Required. Endpoint attachment counter (0-4294967295). Enter 0 for new devices.",
  HELPER_NETWORK_KEY: "16-byte network session key (32 hex chars). Required.",
  HELPER_APP_KEY: "Optional 16-byte application key (32 hex chars).",
  HELPER_DEVICE_MODEL:
    "Select a device model to enable automatic payload decoding via blueprints.",
  OPTION_NONE_MANUFACTURER: "None (clear manufacturer)",
  OPTION_NONE_MODEL: "None (clear model)",
  PLACEHOLDER_NO_MANUFACTURERS: "No manufacturers available",
  PLACEHOLDER_NO_DEVICE_MODELS: "No device models for this manufacturer",

  // Tooltips
  TOOLTIP_BIDI:
    "MIOTY Bidirectional Mode: Enables endpoint to receive downlink messages. Required for two-way communication with base stations.",
  TOOLTIP_PRE_ATTACH:
    "Immediately propagate to all base stations upon creation",
  TOOLTIP_DUAL_CHAN:
    "MIOTY Dual Channel Mode: Uses two frequency channels for improved reliability and capacity. Increases power consumption but enhances robustness.",
  TOOLTIP_REPETITION:
    "MIOTY DL Repetition: Repeats downlink messages for improved reliability in poor signal conditions. Increases latency but enhances message delivery success.",
  TOOLTIP_WIDE_CARR_OFF: "Use wide carrier offset range",
  TOOLTIP_LONG_BLK_DIST: "Use longer downlink interblock distance",

  // Validation Errors
  ERROR_EUI_REQUIRED: "EUI is required",
  ERROR_EUI_FORMAT: "EUI must be 16 hexadecimal characters",
  ERROR_NAME_REQUIRED: "Name is required",
  ERROR_NETWORK_KEY_REQUIRED: "Network key is required",
  ERROR_NETWORK_KEY_FORMAT: "Network key must be 32 hexadecimal characters",
  ERROR_APP_KEY_FORMAT:
    "Application key must be 32 hexadecimal characters if provided",
  ERROR_TYPE_EUI_FORMAT: "Type EUI must be 16 hexadecimal characters",
  // Short address validation (always required per BSSCI §3.8.1)
  ERROR_SHORT_ADDR_REQUIRED: "Short address is required (BSSCI §3.8.1)",
  ERROR_SHORT_ADDR_FORMAT: "Short address must be 4 hex characters (0001-FFFF)",
  ERROR_SHORT_ADDR_ZERO: "Short address cannot be zero",
  ERROR_SHORT_ADDR_RANGE: "Short address must be 0-65535",
  // Packet counter validation (always required per BSSCI §3.8.1)
  ERROR_LAST_PACKET_CNT_REQUIRED:
    "Last packet counter is required (BSSCI §3.8.1)",
  ERROR_LAST_PACKET_CNT_RANGE: "Packet counter must be 0-4294967295",
  // Attach counter validation (always required per SCACI §3.6.1)
  ERROR_ATTACH_CNT_REQUIRED: "Attach counter is required (SCACI §3.6.1)",
  ERROR_ATTACH_CNT_RANGE: "Attach counter must be 0-4294967295",
  ERROR_CARRIER_OFFSET_FORMAT:
    "Carrier offset must be a whole number of Hz (-2147483648 to 2147483647)",
  ERROR_CREATE_PREFIX: "Error creating endpoint: ",
  ERROR_CREATE_GENERIC: "Error creating endpoint",
  ERROR_UPDATE_PREFIX: "Error updating endpoint: ",
  ERROR_UPDATE_GENERIC: "Error updating endpoint",

  // Buttons
  BUTTON_CANCEL: "Cancel",
  BUTTON_CREATE: "Create End Point",
  BUTTON_CREATING: "Creating...",
  BUTTON_UPDATE: "Save Changes",
  BUTTON_UPDATING: "Saving...",
  BUTTON_GENERATE: "Generate",
  ACTION_SHOW_KEY: "Show key",
  ACTION_HIDE_KEY: "Hide key",
  ACTION_COPY_KEY: "Copy key",
  ACTION_REVEAL_KEY: "Reveal key (recorded in the Audit Log)",
  LABEL_KEY_COPIED: "Key copied",
  PLACEHOLDER_KEY_STORED:
    "Stored and hidden. Reveal it, or enter a new key to replace it",
  ERROR_REVEAL_KEY: "The key could not be revealed",
  ACTION_REMOVE_KEY: "Remove key",
  PLACEHOLDER_KEY_REMOVED: "Removed when you save",
  DIALOG_REMOVE_KEY_TITLE: "Remove Application Key",
  DIALOG_REMOVE_KEY_MESSAGE:
    "The stored application key is deleted when you save the End Point, and the removal is recorded in the Audit Log. A removed key cannot be recovered.",
  BUTTON_REMOVE_KEY: "Remove",

  // Edit Dialog
  EDIT_DIALOG_TITLE: "Edit End Point",
  EDIT_DIALOG_SUBTITLE: "Update MIOTY end point configuration",

  // Alerts
  ALERT_PREATTACH:
    "Pre-Attachment is enabled. The endpoint will be immediately propagated to all connected base stations upon creation, allowing it to start sending messages right away.",
  ALERT_REATTACH_PROMPT:
    "Configuration changed. Re-attach to push updates to base stations?",

  // Re-attach Dialog
  REATTACH_DIALOG_TITLE: "Re-attach Endpoint?",
  REATTACH_BUTTON_LATER: "Later",
  REATTACH_BUTTON_NOW: "Re-Attach Now",
  REATTACH_BUTTON_PENDING: "Re-attaching...",
  REATTACH_CHANGED_FIELDS: "Changed settings the base stations still hold:",
  REATTACH_LATER_HINT:
    "Choose Later to keep the end point marked Configuration pending re-attach until it is attached again.",

  // Success Messages
  MSG_ENDPOINT_UPDATED: "End point updated",
  MSG_ENDPOINT_CREATED: "End point created",
  MSG_ENDPOINT_DELETED: "End point deleted",
  MSG_ENDPOINT_ATTACHED: "End point attached",
  MSG_ENDPOINT_DETACHED: "End point detached",
  // Why Save Changes is disabled
  MSG_NO_CHANGES: "No changes to save",
} as const;

// ============================================================================
// Endpoint Details Display Constants (EndPointDetails.tsx)
// Used for read-only display labels, status text, and event rendering
// ============================================================================

/** Read-only MIOTY configuration and reception of an endpoint (endpoint detail). */
export const ENDPOINT_CONFIG = {
  SECTION_CONFIGURATION: "MIOTY Configuration",
  SECTION_RECEPTION: "Reception",
  LABEL_CLASS: "Class",
  CLASS_A: "A (bidirectional)",
  CLASS_Z: "Z (unidirectional)",
  LABEL_NETWORK_KEY: "Network Session Key",
  LABEL_APP_KEY: "Application Key",
  LABEL_SERVING_BS: "Serving Base Station",
  LABEL_LAST_RSSI: "Last RSSI",
  LABEL_LAST_SNR: "Last SNR",
  LABEL_LAST_EQ_SNR: "Last eqSNR",
  VALUE_YES: "Yes",
  VALUE_NO: "No",
  VALUE_KEY_SET: "Set",
  VALUE_KEY_NOT_SET: "Not set",
  VALUE_NOT_SET: "Not set",
  VALUE_NO_SERVING_BS: "None yet",
  VALUE_NOT_HEARD: "Not heard yet",
  UNIT_HZ: "Hz",
} as const;

export const ENDPOINT_DETAILS = {
  // Section Headers
  SECTION_DEVICE_INFO: "Device Basic Information",
  SECTION_STATUS_INFO: "Status Information",

  // Field Labels (display)
  LABEL_DEVICE_NAME: "Device Name",
  LABEL_DEVICE_EUI: "Device EUI",
  LABEL_ATTACH_STATUS: "Attach Status",
  LABEL_ROAMING_STATUS: "Roaming Status",
  LABEL_LAST_SEEN: "Last Seen",
  LABEL_OWNER_TENANT: "Owner: Tenant",

  // Status Values
  STATUS_ROAMING: "Roaming",
  STATUS_HOME_NETWORK: "Home Network",
  ATTACH_STATUS_ATTACHED: "Attached",
  ATTACH_STATUS_DETACHED: "Detached",
  ATTACH_STATUS_ATTACHING: "Attaching",
  ATTACH_STATUS_PENDING: "Pending",
  ATTACH_STATUS_UNKNOWN: "Unknown",
  BADGE_REATTACH_PENDING: "Configuration pending re-attach",
  TOOLTIP_REATTACH_PENDING:
    "Base stations still hold the earlier configuration until the end point is attached again.",
  STATUS_NEVER: "Never",

  // Action Buttons
  ACTION_ATTACH_EP: "Attach End Point",
  ACTION_DETACH_EP: "Detach End Point",
  ACTION_ATTACHING: "Attaching...",
  ACTION_DETACHING: "Detaching...",
  ACTION_DELETING: "Deleting...",
  ACTION_SAVING: "Saving...",

  // Tab Labels
  TAB_DOWNLINK: "Downlink",
  TAB_CONFIGURATION: "Configuration",

  UNIT_DBM: "dBm",
  UNIT_DB: "dB",

  // Delete Dialog
  DIALOG_DELETE_TITLE: "Delete End Point",
  DIALOG_DELETE_CONFIRM_PREFIX:
    "Are you sure you want to delete this end point",
  DIALOG_DELETE_NOTE:
    "The endpoint will be detached from all base stations before deletion.",
  DIALOG_DELETE_WARNING: "This action cannot be undone.",
  DIALOG_DELETE_NOTE_PREFIX: "Note:",

  // Edit Dialog
  DIALOG_EDIT_TITLE: "Edit End Point",
  OPTION_SELECT_MANUFACTURER: "Select Manufacturer",
  OPTION_SELECT_MODEL: "Select Model",

  // Blueprint info (read-only display in endpoint details)
  SECTION_BLUEPRINT_INFO: "Blueprint Configuration",
  BLUEPRINT_NONE: "None",
  ACTION_ASSIGN_BLUEPRINT: "Assign Blueprint",

  // Icon Button Tooltips
  TOOLTIP_EDIT: "Edit",
  TOOLTIP_DELETE: "Delete",

  // Accessibility
  ARIA_ENDPOINT_TABS: "endpoint details tabs",
} as const;

export const ENDPOINT_DETAIL_VIEW_LABELS: Record<EndpointDetailView, string> = {
  activity: DEVICE_DETAIL_TABS.ACTIVITY,
  downlink: ENDPOINT_DETAILS.TAB_DOWNLINK,
  traffic: DEVICE_DETAIL_TABS.TRAFFIC,
  configuration: ENDPOINT_DETAILS.TAB_CONFIGURATION,
};

// ============================================================================
// Connection Status - Live Indicator Labels
// ============================================================================

// ============================================================================
// System Status Widget Labels
// ============================================================================

export const SYSTEM_STATUS = {
  TITLE: "System Status",
  LABEL_CHECKING: "Checking...",
  LABEL_STATUS: "Status",
  LABEL_LATENCY: "Latency",
  LABEL_SERVICE: "Service",
  LABEL_LAST_CHECKED: "Last checked",
  NOT_CHECKED: "N/A",
  TOOLTIP_UNABLE: "Unable to fetch system status",
} as const;

// ============================================================================
// Accessibility Labels (ARIA)
// ============================================================================

export const ARIA = {
  OPEN_DRAWER: "open drawer",
  RETRY: "retry",
  USER_MENU: "User menu",
  USER_MENU_TOGGLE: "Toggle user menu",
} as const;

// ============================================================================
// Error Boundary & Error State Messages
// ============================================================================

export const ERROR_STATE = {
  TITLE: "Something went wrong",
  FALLBACK_MESSAGE: "An unexpected error occurred",
  RETRY: "Try again",
} as const;

export const ERROR_BOUNDARY = {
  TITLE: "Something went wrong",
  DESCRIPTION: "An unexpected error occurred. Please try refreshing the page.",
  ROUTE_ERROR_TITLE: "Page Error",
  RETRY_BUTTON: "Retry",
  REFRESH_BUTTON: "Refresh Page",
  STACK_TRACE_SUMMARY: "Stack trace (development only)",
} as const;

// ============================================================================
// Global Loader Messages
// ============================================================================

export const LOADER = {
  LOADING: "Loading...",
} as const;

// ============================================================================
// Organization Badge
// ============================================================================

export const ORG_BADGE = {
  LABEL_ORG: "Organization",
  LOADING: "Loading...",
  NO_ORG: "No organization",
} as const;

// ============================================================================
// Common UI Text (Page Headers, Empty States, etc.)
// ============================================================================

export const UI_COMMON = {
  // Page Headers
  TITLE_DASHBOARD: "Dashboard",

  // Confirmation
  CONFIRM_DELETE: "Are you sure you want to delete this item?",

  // Error Messages
  ERR_GENERIC: "An error occurred",
  ERR_LOAD_FAILED: "Failed to load data",

  // Time
  TIME_NOW: "just now",
  TIME_AGO: "ago",
  TIME_NEVER: "Never",
} as const;

/** Units of a relative age ("5 minutes ago"), singular and plural. */
export const RELATIVE_TIME_UNIT = {
  MINUTE: { ONE: "minute", MANY: "minutes" },
  HOUR: { ONE: "hour", MANY: "hours" },
  DAY: { ONE: "day", MANY: "days" },
} as const;

/** Unit suffixes of a compact duration ("1d 2h 3m 4s"). */
export const DURATION_UNIT_SUFFIX = {
  DAY: "d",
  HOUR: "h",
  MINUTE: "m",
  SECOND: "s",
  SEPARATOR: " ",
} as const;

// ============================================================================
// Shared Pagination Labels
// ============================================================================

export const PAGINATION_LABELS = {
  ROWS_PER_PAGE: "Rows per page:",
  OF: "of",
} as const;

// ============================================================================
// Table Column Headers (shared across data grids)
// ============================================================================

export const TABLE_HEADERS = {
  COL_STATUS: "Status",
  COL_FLAGS: "Flags",
  COL_ACTIONS: "Actions",
} as const;

// ============================================================================
// Base Stations Page Messages
// ============================================================================

export const BASE_STATIONS_PAGE = {
  // Page headers
  TITLE: "Base Stations",
  DETAILS_TITLE: "Base Station Details",
  // Map of every located base station
  MAP_TITLE: "Map",
  MAP_EMPTY: "No base station has a location yet",
  MAP_OPEN: "Open base station",
  // Actions
  ADD_BASE_STATION: "Add Base Station",
  BACK_TO_LIST: "Back to List",
  // Search
  SEARCH_PLACEHOLDER: "Search base stations...",
  // Card labels
  TOTAL_BASE_STATIONS: "Total Base Stations",
  ONLINE: "Online",
  OFFLINE: "Offline",
  EXPIRING_CERTIFICATES: "Expiring Certificates",
  // Sort labels
  SORTED_BY_ONLINE: "Sorted by online",
  SORTED_BY_OFFLINE: "Sorted by offline",
  SORTED_BY_EXPIRY: "Sorted by expiry",
  // Table headers
  COL_NAME: "Name",
  COL_EUI: "EUI",
  COL_CONNECTION: "Connection",
  COL_STATUS: "Status",
  COL_DATE_ADDED: "Date Added",
  COL_LAST_SEEN: "Last Seen",
  COL_CERTIFICATE_EXPIRY: "Certificate Expiry",
  // Status
  EXPIRED: "Expired",
  DAYS: "days",
} as const;

export const BASE_STATION_DETAILS = {
  // On-demand status request
  ACTION_REQUEST_STATUS: "Request station status",
  STATUS_REQUESTED:
    "Status requested; the metrics update when the base station answers",
  STATUS_REQUEST_FAILED: "The status request could not be sent",
  // Section headers
  BASIC_INFORMATION: "Basic Information",
  SYSTEM_INFORMATION: "System Information",
  PERFORMANCE_METRICS: "Performance Metrics",
  // Field labels
  BASE_STATION_NAME: "Base Station Name",
  BASE_STATION_EUI: "Base Station EUI",
  STATUS: "Status",
  UPTIME: "Uptime",
  NOT_SET: "Not set",
  NOT_AVAILABLE: "Not available",
  SYSTEM_TIME: "System Time",
  LAST_STATUS_UPDATE: "Last Status Update",
  TEMPERATURE: "Temperature",
  CPU_LOAD: "CPU Load",
  MEMORY_LOAD: "Memory Load",
  DUTY_CYCLE: "Duty Cycle",
  BS_CONFIG: "BS Config",
  DELETE: "Delete",
  DELETING: "Deleting...",
  // Delete dialog
  DIALOG_DELETE_TITLE: "Delete Base Station",
  DIALOG_DELETE_CONFIRM_PREFIX:
    "Are you sure you want to delete the base station",
  DIALOG_DELETE_WARNING: "This action cannot be undone.",
  // Edit dialog
  DIALOG_EDIT_TITLE: "Edit Base Station",
  LABEL_EDIT_NAME: "Base Station Name",
  LABEL_EDIT_EUI: "Base Station EUI",
  LABEL_EUI_COPIED: "EUI copied",
  ACTION_COPY_EUI: "Copy EUI",
  ACTION_SAVE: "Save",
  ACTION_SAVING: "Saving...",
  ACTION_CANCEL: "Cancel",
  // Certificate section
  CERTIFICATES_SECTION_TITLE: "Certificates",
  CERTIFICATES_HINT:
    "Regenerating issues new certificate files for this base station; they can be saved right after issuance.",
  CERTIFICATE_DOWNLOAD_ERROR:
    "Failed to download the certificate file. The links expire 15 minutes after issuance and the private key downloads once.",
  // Certificate regeneration
  ACTION_REGENERATE_CERTS: "Regenerate Certificates",
  REGENERATE_CERTS_CONFIRM_TITLE: "Regenerate Certificates?",
  REGENERATE_CERTS_CONFIRM_TEXT:
    "This issues a new certificate and key and binds the station to them, replacing its current certificate, including one issued outside the service center. The service center refuses the station until it is loaded with the new files, so install them on the station right after you download them.",
  REGENERATE_CERTS_SUCCESS: "Certificates regenerated",
  REGENERATE_CERTS_INSTALL_HINT:
    "The service center refuses the station until it has the new files: download them now and install them on the station.",
  REGENERATE_CERTS_ERROR: "Failed to regenerate certificates",
  ACTION_REGENERATE: "Regenerate",
  // Service Center URL
  LABEL_SC_URL: "Service Center URL",
  ACTION_COPY_SC_URL: "Copy Service Center URL",
  LABEL_SC_URL_COPIED: "URL copied",
} as const;

// ============================================================================
// Endpoints Page Messages
// ============================================================================

export const ENDPOINTS_PAGE = {
  // Page headers
  TITLE: "End Points",
  DETAILS_TITLE: "End Point Details",
  // Actions
  ADD_ENDPOINT: "Add End Point",
  BACK_TO_LIST: "Back to List",
  ERR_NOT_FOUND: "End point not found. It may have been deleted.",
  // Search
  SEARCH_PLACEHOLDER: "Search end points...",
  FILTERS: "Filters",
  // Card labels
  TOTAL_ENDPOINTS: "Total End Points",
  ACTIVE: "Active",
  INACTIVE: "Inactive",
  // Table headers
  COL_NAME: "Name",
  COL_EUI: "EUI",
  COL_STATUS: "Status",
  COL_LAST_SEEN: "Last Seen",
  COL_ATTACH_STATE: "Attach State",
  // Filters
  FILTER_ATTACH_STATE: "Attach state",
  FILTER_ACTIVITY: "Activity",
  SHOWING: "Showing",
  OF: "of",
  SEARCH_CHIP_PREFIX: "Search:",
  CLEAR_FILTERS: "Clear filters",
  // Empty states
  NO_MATCH: "No endpoints match your search",
  NO_ENDPOINTS: "No endpoints registered yet",
  // Misc
  UNNAMED_DEVICE: "Unnamed Device",
} as const;

// ============================================================================
// Certificates Page Messages
// ============================================================================

export const CERTIFICATES_PAGE = {
  // Page headers
  TITLE: "Certificates",
  // Empty states
  NO_CERTIFICATES: "No certificates found",
} as const;

// ============================================================================
// Dashboard Page Messages (extends existing DASHBOARD)
// ============================================================================

/** Shown to a signed-in user who holds no role yet. */
export const NO_ACCESS = {
  TITLE: "You don't have access yet",
  DESCRIPTION:
    "Your account is ready, but no roles have been granted to it. Ask an administrator to give you access in Users & Roles. This page updates on its own once they do.",
  CHANGE_PASSWORD: "Change password",
} as const;

/** Panel shown in place of a view the user's roles do not cover. */
export const NO_ACCESS_PANEL = {
  TITLE: "No access to this view",
  DESCRIPTION:
    "Your roles do not include this information. Ask an administrator for access in Users & Roles.",
} as const;

export const DASHBOARD_PAGE = {
  // Page headers
  TITLE: "Network Overview",
  SUBTITLE: "Real-time network health and status monitoring",
  // Section headers
  SERVICE_CENTER_STATUS: "Service Center Status",
  // Card labels
  BASE_STATIONS_ONLINE: "Base Stations Online",
  ALL_ONLINE: "All online",
  NO_BASE_STATIONS: "No base stations",
  ENDPOINTS_ATTACHED: "End Points Attached",
  REGISTERED_LABEL: "registered",
  NO_ENDPOINTS: "No endpoints",
  // Trend messages
  ADDED_LAST_WEEK: "added in the last 7 days",
  OFFLINE_LABEL: "offline",
  MESSAGES_RECEIVED: "Messages Received",
  LAST_24_HOURS: "Last 24 hours",
  LIVE: "Live",
  REALTIME_RECONNECTING: "Reconnecting",
  REALTIME_OFFLINE: "Offline",
} as const;

// ============================================================================
// Base Station Messages Component Messages
// ============================================================================

export const BASE_STATION_MESSAGES = {
  EXPORT_CSV: "Export Uplinks (CSV)",
  EXPORT_JSON: "Export Uplinks (JSON)",
  EXPORT_FILENAME_PREFIX: "basestation-",
  EXPORT_FILENAME_SUFFIX: "-uplinks-",
  ERR_EXPORT_NO_DATA: "No uplinks found for the selected date range.",
  ERR_EXPORT_FAILED: "Failed to export uplinks",
} as const;

// ============================================================================
// Activity Date Range Filter (base station and endpoint activity)
// ============================================================================

export const DATE_RANGE_FILTER = {
  LABEL_START_DATE: "Start Date",
  LABEL_END_DATE: "End Date",
  CLEAR_FILTERS: "Clear Filters",
  DATE_PLACEHOLDER_EU: "DD/MM/YYYY",
  ERR_DATE_INVALID: "Enter a calendar date as DD/MM/YYYY.",
} as const;

// ============================================================================
// Generate Certificate Dialog Messages
// ============================================================================

// ============================================================================
// Certificate Info Alert Messages
// ============================================================================

export const CERTIFICATE_INFO = {
  ALERT_TITLE: "About Server Certificates",
  ALERT_TEXT:
    "Server certificates are used by the BSSCI server to establish secure TLS connections with base stations. Base stations trust the Certificate Authority (CA) that signs these server certificates, not the individual server certificates themselves. This allows server certificates to be rotated without requiring updates to base stations. The CA certificate has a long validity period (20 years) while server certificates can be rotated periodically for security best practices.",
  ISSUER: "Issuer",
  SUBJECT: "Subject",
  EXPIRES: "Expires",
  DAYS_UNTIL_EXPIRY: "Days until expiry",
} as const;

// ============================================================================
// Server Certificate Actions
// ============================================================================

export const SERVER_CERTIFICATES = {
  GENERATE_BUTTON: "Generate Server Certificates",
  RENEW_BUTTON: "Renew Server Certificates",
  MSG_GENERATED: "Server certificates generated",
  MSG_RENEWED: "Server certificates renewed",
  RENEW_CONFIRM_TITLE: "Renew Server Certificates",
  RENEW_CONFIRM_TEXT:
    "A new server certificate is issued for these names, signed by the same CA. The service center serves it from the next connection on; connected stations keep their session. A station that connects by a name missing here fails hostname verification.",
  RENEW_NAMES_LABEL: "Names on the new certificate",
  CANCEL: "Cancel",
} as const;

// ============================================================================
// Users - Errors
// ============================================================================

export const ERR_LOAD_USERS = "Failed to load users. Please try again.";
// ============================================================================
// Users - Success
// ============================================================================

export const MSG_USER_CREATED = "User created";
export const MSG_USER_UPDATED = "User updated successfully";
export const MSG_USER_DELETED = "User deleted";
export const MSG_PASSWORD_CHANGED = "Password changed successfully";

// ============================================================================
// Users Page Messages
// ============================================================================

export const USERS_PAGE = {
  // Page header
  TITLE: "Users & Roles",
  // Actions
  ADD_USER: "Add User",
  BACK_TO_LIST: "Back to list",
  BACK_TO_DETAIL: "Back to details",
  // Search
  SEARCH_PLACEHOLDER: "Search users...",
  // Empty states
  NO_USERS: "No users found",
  NO_MATCH: "No users match your search",
  // Table columns
  COL_EMAIL: "Email",
  COL_CREATED: "Created",
  // Statistics
  TOTAL_USERS: "Total users",
  ACTIVE_USERS: "Active users",
  INACTIVE_USERS: "Inactive users",
  ADMIN_USERS: "Admins",
  // Details page
  DETAILS_TITLE: "User Details",
  // Errors
  ERR_NOT_FOUND: "User not found",
  // Boolean display values
  YES: "Yes",
  NO: "No",
  // Tooltips
  TOOLTIP_EDIT: "Edit user",
  TOOLTIP_DELETE: "Delete user",
  // Delete confirmation
  CONFIRM_DELETE_TITLE: "Delete User",
  CONFIRM_DELETE_MESSAGE:
    "Are you sure you want to delete this user? This action cannot be undone.",
  ACTION_CANCEL: "Cancel",
  ACTION_DELETE: "Delete",
  ERR_DELETE_FAILED: "Failed to delete user",
} as const;

/**
 * Users & Roles page messages
 * Tab labels and org-required state messages
 */
export const USERS_AND_ROLES = {
  TITLE: "Users & Roles",
  ARIA_TABS: "users and roles tabs",
  TABS: {
    SYSTEM_USERS: "System Users",
    ORGANIZATION_USERS: "Organization Users",
  },
  ORG_REQUIRED: {
    TITLE: "Organization Required",
    DESCRIPTION: "Please select an organization to view organization users.",
    ACTION: "Select Organization",
  },
} as const;

// ============================================================================
// User Form Messages
// ============================================================================

export const USER_FORM = {
  // Dialog titles
  DIALOG_TITLE_ADD: "Add User",
  DIALOG_TITLE_EDIT: "Edit User",
  // Field labels
  LABEL_EMAIL: "Email",
  LABEL_PASSWORD: "Password",
  LABEL_CONFIRM_PASSWORD: "Confirm password",
  LABEL_NOTE: "Optional notes",
  LABEL_IS_ACTIVE: "Is active",
  LABEL_IS_ADMIN: "Is admin",
  LABEL_IS_TENANT_MGR: "Is tenant manager",
  LABEL_IS_BS_MGR: "Is base station manager",
  LABEL_IS_EP_MGR: "Is endpoint manager",
  // Validation errors
  ERR_EMAIL_REQUIRED: "Email is required",
  ERR_PASSWORD_REQUIRED: "Password is required",
  ERR_CREATE_FAILED: "Failed to create user",
  ERR_UPDATE_FAILED: "Failed to update user",
  ERR_CHANGE_PASSWORD_FAILED: "Failed to change password",
  // Actions
  ACTION_SUBMIT: "Submit",
  ACTION_CANCEL: "Cancel",
  ACTION_CHANGE_PASSWORD: "Change Password",
  ACTION_DELETE: "Delete user",
  CONFIRM_DELETE: "Are you sure you want to delete this user?",
  // Info section
  INFO_TITLE: "Info",
  INFO_CREATED: "Created",
  INFO_UPDATED: "Updated",
  // Organization picker (AddUserDialog multi-org select)
  LABEL_ORGANIZATIONS: "Organizations",
  HELPER_ORGANIZATIONS:
    "Assign the user to one or more organizations after creation",
  ERR_ADD_TO_ORG_PARTIAL:
    "User created but failed to add to some organizations",
  INFO_JOINS_INSTALLATION_ORG:
    "Every user of this installation belongs to its organization: ",
  // Organization memberships (UserDetail card)
  SECTION_ORG_MEMBERSHIPS: "Organization Memberships",
  COL_ORG_NAME: "Organization",
  LABEL_ADD_TO_ORG: "Add to Organization",
  LABEL_SELECT_ORG: "Select organization",
  LABEL_SELECT_ROLE: "Select role",
  ACTION_ADD_MEMBERSHIP: "Add",
  EMPTY_NO_MEMBERSHIPS: "Not a member of any organization",
  MSG_MEMBERSHIP_ADDED: "User added to organization",
  MSG_MEMBERSHIP_REMOVED: "User removed from organization",
  ERR_MEMBERSHIP_ADD_FAILED: "Failed to add user to organization",
  ERR_MEMBERSHIP_REMOVE_FAILED: "Failed to remove user from organization",
} as const;

// ============================================================================
// API Keys Page Messages
// ============================================================================

export const API_KEYS_PAGE = {
  // Page
  TITLE: "API Keys",
  DESCRIPTION: "Manage API keys for programmatic access to the KiloCenter API.",
  // Actions
  ACTION_CREATE: "Create API Key",
  ACTION_DELETE: "Delete",
  DIALOG_DELETE_TITLE: "Delete API Key",
  ACTION_COPY: "Copy",
  ACTION_COPIED: "Copied!",
  // Table columns
  COL_NAME: "Name",
  COL_KEY_PREFIX: "Key Prefix",
  COL_TYPE: "Type",
  COL_STATUS: "Status",
  COL_LAST_USED: "Last Used",
  COL_CREATED: "Created",
  COL_EXPIRES: "Expires",
  COL_ACTIONS: "Actions",
  // Status
  STATUS_ACTIVE: "Active",
  STATUS_INACTIVE: "Inactive",
  STATUS_EXPIRED: "Expired",
  // Empty state
  NO_API_KEYS: "No API keys found",
  NO_API_KEYS_DESCRIPTION: "Create an API key to enable programmatic access.",
  // Key types
  TYPE_USER: "User",
  TYPE_SERVICE_ACCOUNT: "Service Account",
  // Misc
  NEVER: "Never",
  NO_EXPIRY: "No expiry",
  // UI states
  LOADING: "Loading...",
} as const;

export const API_KEY_FORM = {
  // Dialog titles
  DIALOG_TITLE_CREATE: "Create API Key",
  // Field labels
  LABEL_NAME: "Key Name",
  LABEL_TYPE: "Key Type",
  LABEL_EXPIRES_ON: "Expires On",
  // Placeholders
  PLACEHOLDER_NAME: "Enter a descriptive name for this key",
  PLACEHOLDER_EXPIRES_ON: DATE_RANGE_FILTER.DATE_PLACEHOLDER_EU,
  HELPER_EXPIRES_ON: "Optional. The key stops working at the end of this day.",
  // Validation
  ERR_EXPIRES_ON_INVALID: DATE_RANGE_FILTER.ERR_DATE_INVALID,
  // Actions
  ACTION_CREATE: "Create",
  ACTION_CANCEL: "Cancel",
} as const;

export const API_KEY_CREATED_DIALOG = {
  TITLE: "API Key Created",
  DESCRIPTION: "Copy your API key now. You will not be able to see it again.",
  LABEL_API_KEY: "API Key",
  ACTION_COPY: "Copy to Clipboard",
  ACTION_CLOSE: "Close",
  WARNING: "Make sure to copy this key now. It will not be shown again.",
} as const;

// API Keys - Error Messages
export const ERR_LOAD_API_KEYS = "Failed to load API keys. Please try again.";
export const ERR_CREATE_API_KEY = "Failed to create API key";
export const ERR_DELETE_API_KEY = "Failed to delete API key";
export const ERR_COPY_API_KEY = "Failed to copy the API key to the clipboard";

// API Keys - Success Messages
export const MSG_API_KEY_CREATED = "API key created successfully";
export const MSG_API_KEY_DELETED = "API key deleted successfully";
export const MSG_API_KEY_COPIED = "API key copied to clipboard";

// API Keys - Confirmation
export const CONFIRM_DELETE_API_KEY =
  "Are you sure you want to delete this API key? This action cannot be undone.";

// ============================================================================
// Organizations - Errors
// ============================================================================

export const ERR_LOAD_ORGANIZATIONS =
  "Failed to load organizations. Please try again.";
export const ERR_DELETE_ORGANIZATION = "Failed to delete organization";
export const ERR_LOAD_ORG_USERS = "Failed to load organization members";
// ============================================================================
// Organizations - Success
// ============================================================================

// ============================================================================
// Organizations Page Messages
// ============================================================================

export const ORGANIZATIONS_PAGE = {
  // Page header
  TITLE: "Organizations",
  // Accessibility
  ARIA_TABS: "organization detail tabs",
  // Actions
  ADD_ORGANIZATION: "Add Organization",
  BACK_TO_LIST: "Back to Organizations",
  // Search
  SEARCH_PLACEHOLDER: "Search organizations...",
  // Empty states
  NO_ORGANIZATIONS: "No organizations found",
  NO_MATCH: "No organizations match your search",
  // Table columns
  COL_NAME: "Name",
  COL_DESCRIPTION: "Description",
  COL_STATE: "State",
  COL_CREATED: "Created",
  COL_ACTIONS: "Actions",
  // Statistics
  TOTAL_ORGANIZATIONS: "Total organizations",
  ACTIVE_ORGANIZATIONS: "Active",
  SUSPENDED_ORGANIZATIONS: "Suspended",
  ARCHIVED_ORGANIZATIONS: "Archived",
  // Details page
  DETAILS_TITLE: "Organization Details",
  // State labels
  STATE_ACTIVE: "Active",
  STATE_SUSPENDED: "Suspended",
  STATE_ARCHIVED: "Archived",
  // Quota display
  // Errors
  ERR_NOT_FOUND: "Organization not found",
  ERR_NOT_MEMBER:
    "You are not a member of this organization. Ask an administrator to add you.",
} as const;

// ============================================================================
// Organization Form Messages
// ============================================================================

export const ORGANIZATION_FORM = {
  // Dialog titles
  DIALOG_TITLE_ADD: "Add Organization",
  DIALOG_TITLE_EDIT: "Edit Organization",
  // Field labels
  LABEL_NAME: "Name",
  LABEL_DESCRIPTION: "Description (optional)",
  LABEL_STATE: "State",
  LABEL_TAGS: "Tags",
  // Helper text
  HELPER_NAME: "Organization display name",
  HELPER_DESCRIPTION: "Optional description for the organization",
  // Validation errors
  ERR_NAME_REQUIRED: "Name is required",
  ERR_CREATE_FAILED: "Failed to create organization",
  ERR_UPDATE_FAILED: "Failed to update organization",
  // Actions
  ACTION_SUBMIT: "Submit",
  ACTION_CANCEL: "Cancel",
  ACTION_DELETE: "Delete organization",
  DIALOG_DELETE_TITLE: "Delete Organization",
  CONFIRM_DELETE:
    "Are you sure you want to delete this organization? This action cannot be undone.",
  // Success messages
  SUCCESS_CREATE: "Organization created",
  SUCCESS_UPDATE: "Organization updated successfully",
  SUCCESS_DELETE: "Organization deleted",
  // Info section
  INFO_TITLE: "Info",
  INFO_TENANT_ID: "Tenant ID",
  INFO_CREATED: "Created",
  INFO_UPDATED: "Updated",
} as const;

// ============================================================================
// Organization Users Page Messages
// ============================================================================

/** The role names of Users & Roles and the user documentation. */
export const ROLE_NAMES = {
  ADMIN: "Admin",
  TENANT_MANAGER: "Tenant Manager",
  BASE_STATION_MANAGER: "Base Station Manager",
  ENDPOINT_MANAGER: "Endpoint Manager",
} as const;

export const ORG_USERS_PAGE = {
  ORG_NAME_PREFIX: "- ",
  // Page header
  TITLE: "Organization Members",
  // Actions
  ADD_USER: "Add User",
  VIEW_MEMBERS: "View Members",
  BACK_TO_ORG: "Back to organization",
  // Statistics
  TOTAL_MEMBERS: "Total Members",
  // Search
  SEARCH_PLACEHOLDER: "Search members...",
  // Empty states
  NO_USERS: "No members found",
  NO_MATCH: "No members match your search",
  // Table columns
  COL_EMAIL: "Email",
  COL_ROLE: "Role",
  COL_STATUS: "Status",
  COL_TENANT_MANAGER: ROLE_NAMES.TENANT_MANAGER,
  COL_BASE_STATION_MANAGER: ROLE_NAMES.BASE_STATION_MANAGER,
  COL_ENDPOINT_MANAGER: ROLE_NAMES.ENDPOINT_MANAGER,
  COL_JOINED: "Joined",
  COL_ACTIONS: "Actions",
  // Role labels
  ROLE_OWNER: "Owner",
  ROLE_ADMIN: ROLE_NAMES.ADMIN,
  ROLE_MEMBER: "Member",
  // Status labels
  STATUS_ACTIVE: "Active",
  STATUS_INVITED: "Invited",
  STATUS_REMOVED: "Removed",
  // Boolean display
  YES: "Yes",
  NO: "No",
  // Tooltips
  TOOLTIP_EDIT: "Edit member",
  TOOLTIP_REMOVE: "Remove member",
  // Errors
  ERR_CANNOT_REMOVE_LAST_OWNER:
    "Cannot remove the last owner of an organization",
  ERR_CANNOT_REMOVE_SELF: "You cannot remove yourself from the organization",
} as const;

// ============================================================================
// Organization User Form Messages
// ============================================================================

export const ORG_USER_FORM = {
  // Dialog titles
  DIALOG_TITLE_ADD: "Add Member",
  DIALOG_TITLE_EDIT: "Edit Member",
  // Field labels
  LABEL_USER_ID: "User",
  LABEL_STATUS: "Status",
  LABEL_TENANT_MANAGER: ROLE_NAMES.TENANT_MANAGER,
  LABEL_BASE_STATION_MANAGER: ROLE_NAMES.BASE_STATION_MANAGER,
  LABEL_ENDPOINT_MANAGER: ROLE_NAMES.ENDPOINT_MANAGER,
  // Helper text
  HELPER_ORG_ADMIN: "Can manage organization settings and members",
  HELPER_BS_ADMIN: "Can manage base stations for this organization",
  HELPER_EP_ADMIN: "Can manage endpoints for this organization",
  // Validation errors
  ERR_USER_REQUIRED: "User is required",
  ERR_ADD_FAILED: "Failed to add member",
  ERR_UPDATE_FAILED: "Failed to update member",
  ERR_REMOVE_FAILED: "Failed to remove member",
  MSG_MEMBER_ADDED: "Member added",
  MSG_MEMBER_UPDATED: "Member updated",
  MSG_MEMBER_REMOVED: "Member removed",
  // Actions
  ACTION_SUBMIT: "Submit",
  ACTION_CANCEL: "Cancel",
  ACTION_REMOVE: "Remove member",
  DIALOG_REMOVE_TITLE: "Remove Member",
  CONFIRM_REMOVE:
    "Are you sure you want to remove this member from the organization?",
} as const;

// ============================================================================
// Tags Editor Messages
// ============================================================================

export const TAGS_EDITOR = {
  LABEL_NO_TAGS: "No tags",
  // Placeholder
  PLACEHOLDER_KEY: "Enter key",
  PLACEHOLDER_VALUE: "Enter value",
  ACTION_REMOVE: "Remove tag",
} as const;

// ============================================================================
// User Menu Messages
// ============================================================================

export const USER_MENU = {
  LOGOUT: "Logout",
  CHANGE_PASSWORD: "Change Password",
  THEME_LIGHT: "Light Mode",
  THEME_DARK: "Dark Mode",
  VERSION_TOOLTIP_BUILD: "Build",
  VERSION_TOOLTIP_COMMIT: "Commit",
  VERSION_TOOLTIP_BRANCH: "Branch",
  VERSION_TOOLTIP_SCHEMA: "Schema",
  DEVELOPMENT_BUILD_SUFFIX: " (dev)",
} as const;

/** Build and version details shown when the service does not report them. */
export const VERSION_INFO = {
  UNKNOWN: "unknown",
  LOAD_FAILED: "Failed to load version info",
} as const;

// ============================================================================
// My Password Page Messages
// ============================================================================

export const MY_PASSWORD = {
  TITLE: "Change Password",
  LABEL_CURRENT_PASSWORD: "Current Password",
  LABEL_NEW_PASSWORD: "New Password",
  LABEL_CONFIRM_PASSWORD: "Confirm Password",
  ACTION_SAVE: "Save Password",
  SIGNED_OUT: "Password changed. Sign in with your new password.",
  ERR_FAILED: "Failed to change password. Please try again.",
} as const;

/**
 * Password rule text, composed from the policy the identity service
 * publishes, and the confirmation mismatch shared by every password form.
 */
export const PASSWORD_RULES = {
  LENGTH: "{min} to {max} characters",
  MIN_PLACEHOLDER: "{min}",
  MAX_PLACEHOLDER: "{max}",
  CONTAINING: ", containing ",
  LETTER: "a letter",
  DIGIT: "a digit",
  JOIN: " and ",
  ERR_MISMATCH: "Passwords do not match",
} as const;

// ============================================================================
// Blueprint Feature - Device Catalog and Payload Decoding
// ============================================================================

/**
 * Blueprint page messages
 * Used for device catalog (manufacturers, models, blueprints) and decode preview
 */
export const BLUEPRINT_LABELS = {
  // Page titles
  PAGE_TITLE: "Blueprints",

  // Actions
  ADD_MANUFACTURER: "Add Manufacturer",
  ADD_MODEL: "Add Model",
  ADD_DECODER: "Add Decoder",
  SET_DEFAULT: "Default for new devices",
  SUBMIT_TO_REGISTRY: "Submit to Registry",
  TEST_DECODE: "Test Decode",

  // Catalog scope (System vs tenant Custom)
  SCOPE_SYSTEM: "System",
  SCOPE_CUSTOM: "Custom",
  SCOPE_TABS_ARIA: "catalog scope",
  // Joins a catalog entry's name and its scope where both catalogs are listed together.
  CATALOG_ENTRY_SEPARATOR: " — ",

  // The catalog a create form adds to (the open tab)
  CREATES_IN_SYSTEM:
    "Adds to the System catalog, shared with every organization on this service center.",
  CREATES_IN_CUSTOM: "Adds to your organization's Custom catalog.",

  // Bulk device migration
  MIGRATE_DEVICES: "Migrate Devices",
  MIGRATE_DIALOG_TITLE: "Migrate Devices to This Blueprint",
  MIGRATE_SET_AS_DEFAULT: "Also set as default for new devices",
  MIGRATE_CONFIRM: "Migrate",
  MSG_MIGRATE_SUCCESS:
    "Devices migrated: {affected} device snapshot(s) now use this blueprint",
  MSG_MANUFACTURER_CREATED: "Manufacturer created",
  MSG_MANUFACTURER_UPDATED: "Manufacturer updated",
  MSG_MANUFACTURER_DELETED: "Manufacturer deleted",
  MSG_MODEL_CREATED: "Device model created",
  MSG_MODEL_UPDATED: "Device model updated",
  MSG_MODEL_DELETED: "Device model deleted",
  MSG_BLUEPRINT_CREATED: "Blueprint created",
  MSG_BLUEPRINT_UPDATED: "Blueprint saved",
  MSG_BLUEPRINT_DEFAULT_SET: "Blueprint set as the model default",
  ERR_MIGRATE_FAILED: "Failed to migrate devices",

  // Form labels - Manufacturer
  LABEL_NAME: "Name",
  LABEL_WEBSITE: "Website",
  LABEL_DESCRIPTION: "Description",

  // Form labels - Device Model
  LABEL_MANUFACTURER: "Manufacturer",

  // Form labels - Blueprint
  LABEL_TYPE_EUI: "Type EUI",
  LABEL_VERSION: "Version",
  LABEL_SPEC_JSON: "Blueprint Specification (JSON)",
  LABEL_TEST_DATA: "Test Payload (hex)",
  LABEL_FORMAT_ID: "Format ID",
  LABEL_CONTRIBUTOR_NAME: "Contributor Name",
  LABEL_CONTRIBUTOR_EMAIL: "Contributor Email",

  HELPER_TYPE_EUI: "8-byte Type EUI in hex format (16 characters)",
  HELPER_SPEC_JSON: "Blueprint JSON per MIOTY Application Layer Specification",
  HELPER_TEST_DATA: "Hex-encoded payload to test decoding",

  // Validation errors
  ERR_INVALID_JSON: "Invalid JSON format",
  ERR_NAME_REQUIRED: "Name is required",
  ERR_MANUFACTURER_REQUIRED: "Manufacturer is required",
  ERR_VERSION_REQUIRED: "Version is required",
  ERR_SPEC_JSON_REQUIRED: "Blueprint specification is required",
  ERR_CONTRIBUTOR_NAME_REQUIRED: "Contributor name is required",
  ERR_CONTRIBUTOR_EMAIL_REQUIRED: "Contributor email is required",
  ERR_NO_MANUFACTURER_SELECTED: "No manufacturer selected",
  ERR_NO_MODEL_SELECTED: "No model selected",

  // Decode status display (human-readable)
  DECODE_SUCCESS: "Decoded successfully",
  DECODE_FAILED: "Decode failed",

  // Error messages
  ERR_LOAD_MANUFACTURERS: "Failed to load manufacturers",
  ERR_CREATE_BLUEPRINT: "Failed to create blueprint",
  ERR_BLUEPRINT_NOT_FOUND: "Blueprint not found",
  ERR_BLUEPRINT_ID_REQUIRED: "Blueprint ID is required",
  ERR_DELETE_FAILED: "Delete failed",

  // Empty states
  NO_MANUFACTURERS: "No manufacturers found",
  REGISTRY_NOT_CONFIGURED:
    "The blueprint registry is not configured on this service center (registry_provider in the KC-Core configuration). The System catalog holds only the entries a server admin adds here, and blueprints cannot be submitted to the registry.",
  REGISTRY_SUBMIT_UNAVAILABLE:
    "Submission is unavailable: the blueprint registry is not configured on this service center.",
  NO_MODELS: "No device models found",
  NO_BLUEPRINTS: "No blueprints found",
  NO_DECODE_RESULT: "No decode result",

  // Table headers
  COL_NAME: "Name",
  COL_MODELS: "Models",
  COL_BLUEPRINTS: "Blueprints",
  COL_CREATED: "Created",
  COL_ACTIONS: "Actions",

  // Misc
  BADGE_DEFAULT: "Default",
  BADGE_VERIFIED: "Verified",
  BADGE_SYSTEM: "System",

  // Bulk migration affected-count phrasing (count interpolated in component)
  MIGRATE_AFFECTED_PREFIX: "This will re-materialize ",
  MIGRATE_AFFECTED_SUFFIX: " device snapshot(s) onto this blueprint.",
  MIGRATE_NO_DEVICES:
    "No device snapshots reference this model — nothing to migrate.",

  // Navigation
  BACK_TO_BLUEPRINTS: "Back to Blueprints",

  // Page elements
  BLUEPRINT_VERSION_PREFIX: "Blueprint v",
  BLUEPRINT_INFORMATION: "Blueprint Information",

  // Actions
  ACTION_EDIT: "Edit",
  ACTION_CANCEL: "Cancel",
  ACTION_SAVE: "Save",
  ACTION_CREATE: "Create",
  ACTION_CLOSE: "Close",

  // Labels
  LABEL_CREATED: "Created",
  LABEL_UPDATED: "Updated",

  // Placeholders
  PLACEHOLDER_HEX: "0123456789ABCDEF",

  // Error display
  ERROR_CODE_PREFIX: "Code:",

  // Confirmation dialogs
  CONFIRM_DELETE_MANUFACTURER:
    "Delete this manufacturer? All device models and blueprints will also be deleted.",
  CONFIRM_DELETE_MODEL:
    "Delete this device model? All blueprints will also be deleted.",
  // Edit actions
  ACTION_DELETE: "Delete",
  DIALOG_DELETE_MANUFACTURER: "Delete Manufacturer",
  DIALOG_DELETE_MODEL: "Delete Device Model",
  DELETING: "Deleting...",

  // Dialog titles
  DIALOG_EDIT_MANUFACTURER: "Edit Manufacturer",
  DIALOG_EDIT_MODEL: "Edit Device Model",
  DIALOG_SUBMIT_TO_REGISTRY: "Submit to Registry",

  // Registry submission
  REGISTRY_DESCRIPTION_PLACEHOLDER:
    "Describe the changes or purpose of this blueprint...",
  REGISTRY_SUBMIT_SUCCESS: "Blueprint submitted to registry successfully",
  REGISTRY_PR_CREATED: "Pull request created",
  REGISTRY_VIEW_PR: "View Pull Request",
  REGISTRY_BRANCH: "Branch:",
  REGISTRY_COMMIT: "Commit:",
} as const;

// ============================================================================
// Realtime Connection Messages
// ============================================================================

export const REALTIME_MESSAGES = {
  CONNECTING_AT: "Connecting to gRPC stream at",

  // Connection status
  CONNECTED: "gRPC stream connected",
  STREAM_ENDED_NORMAL: "Stream ended normally",
  STREAM_ENDED_ERROR: "Stream ended with error",
  FAILED_ESTABLISH: "Failed to establish gRPC stream",
  RECONNECTING_IN: "Reconnecting in",
  RECONNECT_ATTEMPT: "attempt",

  // Base station stream
  BS_STREAM_CONNECTED: "Base station stream connected",
  BS_STREAM_ERROR: "Base station stream error",

  // Event stream
  EVENT_STREAM_CONNECTED: "Event stream connected",
  EVENT_STREAM_ERROR: "Event stream error",

  LEADER_TAB_CLOSED: "The tab holding the realtime streams closed",

  HANDLER_FAILED: "Realtime event handler failed",
  LISTENER_FAILED: "Realtime connection listener failed",

  // Context guards
  CONTEXT_INCOMPLETE:
    "Realtime connection deferred: missing auth, org, or user context",
} as const;

// ============================================================================
// Certificate Labels (User-Facing)
// ============================================================================

export const CERTIFICATE_LABELS = {
  SERVER: "Server Certificate",
  CA: "CA Certificate",
} as const;

// ============================================================================
// gRPC Client Error Messages (internal API layer)
// Used by KC-Web/src/services/grpc/client.ts for response validation
// ============================================================================

export const GRPC_CLIENT_ERRORS = {
  // Generic
  EMPTY_RESPONSE: "Empty response from server",

  // Auth
  INVALID_LOGIN_RESPONSE: "Invalid login response",
  INVALID_AUTH_SETTINGS_RESPONSE: "Invalid auth settings response",
  INVALID_PROFILE_RESPONSE: "Invalid profile response",
  INVALID_REFRESH_TOKENS_RESPONSE: "Invalid refresh tokens response",
  TOKEN_REFRESH_UNAVAILABLE:
    "The session could not be renewed because the service is not reachable. Please try again.",
  INVALID_OIDC_EXCHANGE_RESPONSE: "Invalid OIDC exchange response",
  INVALID_OAUTH2_EXCHANGE_RESPONSE: "Invalid OAuth2 exchange response",

  // Analytics/Dashboard
  INVALID_ANALYTICS_RESPONSE: "Invalid analytics response",

  // Users
  INVALID_CREATE_USER_RESPONSE: "Invalid create user response",
  INVALID_UPDATE_USER_RESPONSE: "Invalid update user response",

  // Organizations
  INVALID_CREATE_ORGANIZATION_RESPONSE: "Invalid create organization response",
  INVALID_UPDATE_ORGANIZATION_RESPONSE: "Invalid update organization response",
  INVALID_ADD_ORG_USER_RESPONSE: "Invalid add organization user response",
  INVALID_UPDATE_ORG_USER_RESPONSE: "Invalid update organization user response",

  // Catalog (Blueprints)
  INVALID_CREATE_MANUFACTURER_RESPONSE: "Invalid create manufacturer response",
  INVALID_CREATE_BLUEPRINT_RESPONSE: "Invalid create blueprint response",

  // API Keys
  API_KEY_CREATION_FAILED: "API key creation failed",
} as const;

// ============================================================================
// Shared value rendering
// ============================================================================

export const VALUE_FORMAT = {
  YES: "Yes",
  NO: "No",
  MILLISECONDS_SUFFIX: " ms",
  PERCENT_SUFFIX: "%",
  METERS_SUFFIX: " m",
  CELSIUS_SUFFIX: "°C",
  THOUSANDS_SUFFIX: "K",
  MILLIONS_SUFFIX: "M",
  HEX_PREFIX: "0x",
  OPID_RANGE_SEPARATOR: " / ",
  DAYS_LEFT_SUFFIX: " days left",
  DAYS_LEFT_EXPIRED: "Expired",
} as const;

// ============================================================================
// Dashboard alerts, control-plane strip and certificate expiry states (NAV 1)
// ============================================================================

export const ALERTS_PANEL = {
  TITLE: "Alerts",
  COUNTS_RULE:
    "Unresolved alerts of warning severity and above, however old; each count is the number of rows its filter lists",
  OPEN_EVENTS_LOG: "Open Events Log",
  FILTER_ALL: "All",
  EMPTY: "No open alerts",
  ERR_LOAD: "Failed to load alerts",
} as const;

export const CONTROL_PLANE_STRIP = {
  TITLE: "SC-AC Control Plane",
  SESSION_STATE: "Session",
  CONNECTED: "Connected",
  DISCONNECTED: "Disconnected",
  LISTENER_OFFLINE: "SCACI listener offline",
  VERSION: "version",
  SC_EUI: "scEui",
  AC_EUI: "acEui",
  SN_RESUME: "snResume",
  SN_AC_UUID: "snAcUuid",
  SN_SC_UUID: "snScUuid",
  LAST_CON: "Last con",
  LAST_PING: "Last ping",
  RTT: "RTT",
  MISSED_HEARTBEATS: "Missed heartbeats",
  OP_ID_RANGE: "opId max AC / min SC",
  ERR_LOAD: "Failed to load the control plane status",
} as const;

export const CERTIFICATE_EXPIRY_STATE_LABELS: Record<string, string> = {
  not_issued: "Not issued",
  expiry_not_recorded: "Expiry not recorded",
  valid: "Valid",
  expiring: "Expiring",
  critical: "Expires soon",
  expired: "Expired",
};

export const BS_CERTIFICATES_SECTION = {
  TITLE: "Base Station Certificates",
  SUBTITLE: "Client certificates issued to each base station",
  COL_NAME: "Name",
  COL_BS_EUI: "EUI",
  COL_STATUS: "Status",
  COL_EXPIRES: "Expires",
  COL_DAYS_LEFT: "Days Left",
  COL_FINGERPRINT: "Fingerprint (SHA-256)",
  COL_LAST_HANDSHAKE: "Last Handshake",
  COL_ACTIONS: "Actions",
  ACTION_OPEN: "Open base station",
  EMPTY: "No base stations",
  ERR_LOAD: "Failed to load base station certificates",
} as const;

// ============================================================================
// Base station operations (NAV 2 detail: Health, Certificates, Actions)
// ============================================================================

export const BS_OPERATIONS = {
  AVAILABILITY_TITLE: "Availability (last 24 hours)",
  AVAILABILITY_HINT: "Share of each hour the base station held a BSSCI session",
  ERR_AVAILABILITY: "Failed to load availability",
  ACTIONS_TITLE: "Actions",
  ACTION_PING: "Ping",
  ACTION_PINGING: "Pinging...",
  PING_HINT: "Sends a BSSCI ping (§5.4); the station answers with pingRsp.",
  PING_SENT: "Ping sent (opId {opId})",
  ERR_PING: "Failed to ping the base station",
  CERTIFICATE_TITLE: "Certificate",
  FINGERPRINT: "Fingerprint (SHA-256)",
  EXPIRES: "Expires",
  LAST_HANDSHAKE: "Last handshake",
  STATUS: "Status",
  NOT_ISSUED: "No certificate has been issued to this base station",
  DOWNLOAD_CA: "Download CA certificate",
  DOWNLOAD_CLIENT: "Download client certificate",
  ERR_DOWNLOAD: "Failed to download the certificate",
  ERR_DOWNLOAD_CLIENT_NOT_STORED: `This service center holds no copy of this base station's client certificate. Use "${BASE_STATION_DETAILS.ACTION_REGENERATE_CERTS}" in its Edit dialog to issue new ones.`,
} as const;

export const BRAND = {
  SOURCE: "Source Code",
  DOCUMENTATION: "Documentation",
  LICENSE: "License",
} as const;
