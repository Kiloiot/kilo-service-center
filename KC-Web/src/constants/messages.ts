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
  "Invalid Base Station EUI format. Please use XX-XX-XX-XX-XX-XX-XX-XX format.";
export const VAL_BS_EUI_REQUIRED = "Base Station EUI is required.";
export const VAL_SC_URL_REQUIRED = "Service Center URL is required.";

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
export const MSG_BS_DELETED = "Base station deleted successfully";
export const MSG_BS_CREATED = "Base station created successfully";
export const MSG_BS_EUI_UPDATED = "Base station EUI updated successfully";

// ============================================================================
// Base Stations - Deletion
// ============================================================================

export const ERR_DELETE_BASE_STATION =
  "Failed to delete base station. Please try again.";
export const ERR_CREATE_BASE_STATION =
  "Failed to create base station. Please try again.";
export const ERR_BS_CREATION_AFTER_CERTS =
  "Base station could not be created. Certificates were generated - you can retry with the same EUI.";
export const ERR_CERT_GENERATION_FAILED_RETRY =
  "Certificate generation failed. The base station was created - you can retry certificate generation.";
export const MSG_BS_PARTIAL_SUCCESS =
  "Base station created, but certificate generation failed. Use the retry button to generate certificates.";

// ============================================================================
// Endpoints - Errors
// ============================================================================

export const ERR_ENDPOINT_NOT_FOUND = "Endpoint not found";
export const ERR_LOAD_ENDPOINT_DETAILS = "Failed to load endpoint details";
export const ERR_LOAD_ENDPOINTS = "Failed to load endpoints";
export const ERR_ATTACH_ENDPOINT =
  "Failed to attach endpoint. Please try again.";
export const ERR_DETACH_ENDPOINT =
  "Failed to detach endpoint. Please try again.";
export const ERR_UPDATE_ENDPOINT =
  "Failed to update endpoint. Please try again.";
export const ERR_DELETE_ENDPOINT =
  "Failed to delete endpoint. Please try again.";

// ============================================================================
// Endpoints - Validation
// ============================================================================

export const VAL_ENDPOINT_SELECT = "Please select an endpoint";

// ============================================================================
// Messages - Errors
// ============================================================================

export const ERR_LOAD_MESSAGES = "Failed to load messages. Please try again.";
export const ERR_LOAD_MESSAGES_GENERIC = "Failed to load messages";

// Endpoint Operation Log Messages (console logging)
export const LOG_ATTACH_FAILED = "Failed to attach endpoint:";
export const LOG_DETACH_FAILED = "Failed to detach endpoint:";
export const LOG_PRE_DELETE_DETACH_FAILED =
  "Pre-delete detach failed, attempting delete anyway:";
export const LOG_DELETE_FAILED = "Failed to delete endpoint:";

// ============================================================================
// Downlink - Errors
// ============================================================================

export const ERR_LOAD_DOWNLINK_QUEUE = "Failed to load downlink queue";
export const ERR_QUEUE_DOWNLINK = "Failed to queue downlink message";
export const ERR_REVOKE_DOWNLINK = "Failed to revoke downlink message";

// ============================================================================
// Downlink - Success
// ============================================================================

export const MSG_DOWNLINK_QUEUED = "Downlink message queued successfully";
export const MSG_DOWNLINK_REVOKED = "Downlink message revoked successfully";

// ============================================================================
// Downlink - Validation
// ============================================================================

export const VAL_PAYLOAD_BASE64 =
  "Invalid payload format. Must be valid base64";
export const VAL_COUNTER_REQUIRED =
  "Counter-dependent mode requires packet counter for each payload";

// Downlink - Labels
export const LABEL_DL_CNT_DEPEND = "Counter Dependent";
export const LABEL_DL_RESPONSE_EXP = "Response Expected";
export const LABEL_DL_RESPONSE_PRIO = "Response Priority";
export const LABEL_DL_WIND_REQ = "DL Window Required";
export const LABEL_DL_EXP_ONLY = "Expedited Only";
export const LABEL_DL_RX_STAT_QRY = "DL RX Status Query";
export const LABEL_DL_STATUS = "Status";
export const LABEL_DL_QUEUE_ID = "Queue ID";
export const LABEL_DL_CREATED_AT = "Created";
export const LABEL_DL_TRANSMITTED_AT = "Transmitted";
export const LABEL_DL_BS_EUI = "Base Station";
export const LABEL_DL_RESULT = "Result";
export const LABEL_DL_PACKET_CNT = "Packet Counter";

// Downlink - Helper text
export const HELPER_DL_FORMAT = "Payload format identifier (0-255)";
export const HELPER_DL_CNT_DEPEND =
  "Payloads are tied to specific packet counters";
export const HELPER_DL_PACKET_CNT = "Uplink packet counter value (uint32)";

// Downlink - Actions
export const ACTION_SEND_DOWNLINK = "Send Downlink";
export const ACTION_SENDING_DOWNLINK = "Sending...";
export const ACTION_REVOKE = "Revoke";
export const ACTION_REVOKING = "Revoking...";
export const ACTION_FLUSH_QUEUE = "Flush Queue";
export const ACTION_FLUSHING_QUEUE = "Flushing...";

// Downlink - Confirmations
export const CONFIRM_REVOKE_DOWNLINK = "Revoke this queued downlink message?";
export const CONFIRM_FLUSH_QUEUE =
  "Revoke all pending downlink messages in the queue?";

// Downlink - Section Headers
export const SECTION_COMPOSE_DOWNLINK = "Compose Downlink";
export const SECTION_DOWNLINK_QUEUE = "Downlink Queue";
export const SECTION_DOWNLINK_RESULTS = "Downlink Results";

// Downlink - Empty States
export const MSG_EMPTY_QUEUE = "No downlink messages in queue";
export const MSG_EMPTY_RESULTS = "No downlink results yet";

// Downlink - Pagination
export const ACTION_LOAD_MORE = "Load More";

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
export const VAL_PRIORITY_RANGE = "Must be 0.0-1.0";
export const VAL_PACKET_CNT_RANGE = "Must be an integer 0-4294967295";

// Data Format Validation
export const VAL_INVALID_HEX_STRING = "Invalid hex string format";
export const VAL_INVALID_TENANT_ID = "Invalid tenant ID format";

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

export const DOWNLINK_RESULT_LABELS: Record<string, string> = {
  sent: "Sent",
  expired: "Expired",
  invalid: "Invalid",
};

// ============================================================================
// Certificates - Errors
// ============================================================================

export const ERR_CERTIFICATE_GENERIC = "An error occurred";
export const ERR_GENERATE_CERTIFICATE = "Failed to generate certificate";
export const ERR_CERT_SERVICE_UNAVAILABLE =
  "Certificate generation service is not available. Please ensure the KC-Core backend is running.";
export const ERR_CERT_INVALID_RESPONSE =
  "Invalid response from server. The certificate generation service may not be running correctly.";
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
export const SESSION_ERRORS = {
  PROFILE_PARSE_FAILED: "Failed to parse stored user profile, cleared storage",
  CONTEXT_REQUIRED: "useSession must be used within SessionProvider",
} as const;

// ============================================================================
// API / Network - Errors (used by query client error mapping)
// ============================================================================

export const ERR_NETWORK_ERROR =
  "Network connection failed. Please check your connection.";
export const ERR_GENERIC_ERROR =
  "An unexpected error occurred. Please try again.";
export const ERR_TIMEOUT_ERROR = "Request timed out. Please try again.";
export const ERR_UNAUTHORIZED =
  "Your session has expired. Please log in again.";
export const ERR_FORBIDDEN =
  "You do not have permission to perform this action.";
export const ERR_NOT_FOUND = "The requested resource was not found.";

// ============================================================================
// Form Labels - Base Station
// ============================================================================

export const LABEL_BS_NAME = "Base Station Name";
export const LABEL_BS_EUI = "Base Station EUI";
export const LABEL_SC_URL = "Service Center URL";
export const LABEL_CERT_VALIDITY = "Certificate Validity";

export const HELPER_BS_NAME = "A friendly name for this base station";
export const HELPER_BS_EUI =
  "Enter the 8-byte EUI of your base station (e.g., 01-23-45-67-89-AB-CD-EF)";
export const HELPER_SC_URL =
  "URL where the base station will connect for BSSCI protocol";

export const PLACEHOLDER_BS_EUI = "XX-XX-XX-XX-XX-XX-XX-XX";

// Location labels and messages
export const LABEL_LATITUDE = "Latitude";
export const LABEL_LONGITUDE = "Longitude";
export const LABEL_ALTITUDE = "Altitude";
export const LABEL_LOCATION = "Location";
export const LABEL_LOCATION_OPTIONAL = "Location (optional)";
export const LABEL_LOCATION_SOURCE = "Source";
export const LABEL_LOCATION_SOURCE_GPS = "GPS";
export const LABEL_LOCATION_SOURCE_MANUAL = "Manual";
export const LABEL_LOCATION_NOT_SET = "Not set";
export const MSG_NO_LOCATION_SET = "No Base Station Location Set";

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

export const LABEL_EP_NAME = "Endpoint Name";
export const LABEL_EP_EUI = "Endpoint EUI";
export const LABEL_EP_MANUFACTURER = "Manufacturer";
export const LABEL_EP_MODEL = "Model";
export const LABEL_EP_BIDI = "Bidirectional";

export const HELPER_EP_NAME = "A friendly name for this endpoint";
export const HELPER_EP_EUI = "Enter the 8-byte EUI of your endpoint";

// ============================================================================
// Form Labels - Downlink
// ============================================================================

export const LABEL_DL_PAYLOAD = "Payload (hex)";
export const LABEL_DL_PRIORITY = "Priority";
export const LABEL_DL_FORMAT = "Format";
export const LABEL_DL_ENDPOINT = "Endpoint";

export const HELPER_DL_PAYLOAD = "Hexadecimal payload data to send";
export const HELPER_DL_PRIORITY = "Message priority (0.0 = low, 1.0 = high)";

// ============================================================================
// UI Strings - Common Actions
// ============================================================================

export const ACTION_CANCEL = "Cancel";
export const ACTION_CLOSE = "Close";
export const ACTION_SAVE = "Save";
export const ACTION_DELETE = "Delete";
export const ACTION_CREATE = "Create";
export const ACTION_DOWNLOAD = "Download";
export const ACTION_COPY = "Copy";
export const ACTION_COPIED = "Copied!";
export const ACTION_EDIT = "Edit";
export const ACTION_REFRESH = "Refresh";

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
export const LABEL_CA_CERT_FILE = "CA Certificate: ca.crt";
export const LABEL_CLIENT_CERT_PREFIX = "Client Certificate: basestation-";
export const LABEL_CLIENT_CERT_SUFFIX = "-certificate.crt";
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
export const INFO_BS_CREATED = "Base station created successfully.";

// ============================================================================
// UI Strings - Validity Options
// ============================================================================

export const VALIDITY_1_YEAR = "1 Year";
export const VALIDITY_2_YEARS = "2 Years";
export const VALIDITY_3_YEARS = "3 Years";

// ============================================================================
// Connection Status
// ============================================================================

export const STATUS_RECONNECTING = "Realtime connection reconnecting...";
export const STATUS_OFFLINE = "Realtime connection offline";
export const STATUS_CONNECTED = "Connected";

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

  // Field Labels
  LABEL_EUI: "End Point EUI",
  LABEL_NAME: "Name",
  LABEL_SHORT_ADDR: "Short Address (ShAddr)",
  LABEL_CARRIER_OFFSET: "Carrier Offset (CarrOff)",
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
  HELPER_EUI:
    "16 hex characters (e.g., 01-23-45-67-89-AB-CD-EF or 0123456789ABCDEF)",
  HELPER_NAME: "Friendly name for this endpoint",
  // Short address helper (always required per BSSCI §3.8.1)
  PLACEHOLDER_SHORT_ADDR: "0001-FFFF",
  HELPER_SHORT_ADDR:
    "Required. 4 hex chars, nonzero (0001-FFFF). From device provisioning.",
  HELPER_CARRIER_OFFSET: "Frequency offset compensation in Hz (optional).",
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

  // Edit Dialog
  EDIT_DIALOG_TITLE: "Edit End Point",
  EDIT_DIALOG_SUBTITLE: "Update MIOTY end point configuration",
  EDIT_LABEL_EUI_READ_ONLY: "End Point EUI (Read-only)",
  EDIT_HELPER_EUI_READ_ONLY: "EUI cannot be changed after creation",

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

  // Success Messages
  MSG_ENDPOINT_UPDATED: "Endpoint updated successfully",
  MSG_ENDPOINT_CREATED: "Endpoint created successfully",
} as const;

// ============================================================================
// Endpoint Details Display Constants (EndPointDetails.tsx)
// Used for read-only display labels, status text, and event rendering
// ============================================================================

export const ENDPOINT_DETAILS = {
  // Section Headers
  SECTION_DEVICE_INFO: "Device Basic Information",
  SECTION_STATUS_INFO: "Status Information",

  // Field Labels (display)
  LABEL_DEVICE_NAME: "Device Name",
  LABEL_DEVICE_EUI: "Device EUI",
  LABEL_FREQUENCY: "Frequency",
  LABEL_ATTACH_STATUS: "Attach Status",
  LABEL_ROAMING_STATUS: "Roaming Status",
  LABEL_LAST_SEEN: "Last Seen",
  LABEL_OWNER_TENANT: "Owner: Tenant",

  // Status Values
  STATUS_ROAMING: "Roaming",
  STATUS_HOME_NETWORK: "Home Network",
  STATUS_ATTACHED: "Attached",
  STATUS_ATTACHING: "Attaching",
  STATUS_DETACHED: "Detached",
  STATUS_DETACHING: "Detaching",
  STATUS_PROPAGATED: "(Propagated)",
  STATUS_NEVER: "Never",

  // Action Buttons
  ACTION_ATTACH_EP: "Attach EP",
  ACTION_DETACH_EP: "Detach EP",
  ACTION_ATTACHING: "Attaching...",
  ACTION_DETACHING: "Detaching...",
  ACTION_DELETING: "Deleting...",
  ACTION_SAVING: "Saving...",

  // Tab Labels
  TAB_MESSAGES: "Activity",
  TAB_DOWNLINK: "Downlink",

  // Message Display
  PLACEHOLDER_DATE_FILTER: "Filter by date (YYYY-MM-DD)",
  LABEL_TOTAL_ITEMS: "Total:",
  LABEL_ITEMS_SUFFIX: "items",
  MSG_NO_MESSAGES: "[NO MESSAGES OR EVENTS FOUND]",
  MSG_ENDPOINT_PREFIX: "Endpoint",
  MSG_PACKET_PREFIX: ": Packet#",
  MSG_RSSI_PREFIX: ", RSSI: ",
  MSG_SNR_PREFIX: ", SNR: ",
  MSG_DATA_PREFIX: " Data:",
  MSG_TO_BS: "to base station",
  MSG_FROM_BS: "from base station",
  MSG_OPERATION_COMPLETE: " (complete)",
  MSG_SYSTEM_EVENT_FALLBACK: "System event occurred",
  MSG_UPLINK_RECEIVED: "Uplink received",
  MSG_DECODED_PREFIX: "Decoded:",

  // Date Labels
  LABEL_TODAY: "TODAY",
  LABEL_YESTERDAY: "YESTERDAY",

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
  LABEL_EDIT_DEVICE_NAME: "Device Name",
  LABEL_EDIT_DEVICE_EUI: "Device EUI",
  HELPER_EUI_FORMAT: "8-byte EUI in hexadecimal format",
  OPTION_SELECT_MANUFACTURER: "Select Manufacturer",
  OPTION_SELECT_MODEL: "Select Model",
  OPTION_SELECT_FREQUENCY: "Select Frequency",

  // Blueprint info (read-only display in endpoint details)
  SECTION_BLUEPRINT_INFO: "Blueprint Configuration",
  LABEL_DECODED_PAYLOAD: "Decoded Payload",
  LABEL_DECODE_ERROR: "Decode Error",

  // Icon Button Tooltips
  TOOLTIP_EDIT: "Edit",
  TOOLTIP_DELETE: "Delete",

  // Accessibility
  ARIA_ENDPOINT_TABS: "endpoint details tabs",

  // Terminal Message Components (for event display)
  TERM_UPLINK_RECEIVED: "Uplink received",
  TERM_HEARD_BY_BS: ", heard by base station ",
  TERM_HEARD_BY_N_BS: ", heard by ",
  TERM_BASE_STATIONS_SUFFIX: " base stations",
  TERM_DUPLICATE: " [duplicate]",
  TERM_PACKET_PREFIX: ": Packet#",
  TERM_RSSI_PREFIX: ", RSSI:",
  TERM_SNR_PREFIX: ", SNR:",
  TERM_DATA_PREFIX: " Data:",
  TERM_ENDPOINT: "Endpoint ",
  TERM_TO_BS: "to base station ",
  TERM_FROM_BS: "from base station ",
  TERM_COMPLETE_SUFFIX: " (complete)",
  TERM_UNKNOWN: "unknown",
} as const;

// ============================================================================
// Connection Status - Live Indicator Labels
// ============================================================================

export const CONNECTION_STATUS = {
  LABEL_LIVE: "Live",
  LABEL_RECONNECTING: "Reconnecting...",
} as const;

// ============================================================================
// System Status Widget Labels
// ============================================================================

export const SYSTEM_STATUS = {
  TITLE: "System Status",
  LABEL_CHECKING: "Checking...",
  LABEL_ERROR: "Error",
  LABEL_OK: "OK",
  LABEL_HEALTHY: "Healthy",
  LABEL_UNHEALTHY: "Unhealthy",
  LABEL_STATUS: "Status",
  LABEL_LATENCY: "Latency",
  LABEL_SERVICE: "Service",
  LABEL_LAST_CHECKED: "Last checked",
  TOOLTIP_UNABLE: "Unable to fetch system status",
} as const;

// ============================================================================
// Accessibility Labels (ARIA)
// ============================================================================

export const ARIA = {
  OPEN_DRAWER: "open drawer",
  RETRY: "retry",
  DISMISS_ERROR: "dismiss error",
  CLOSE: "close",
  PAGINATION_FIRST: "first page",
  PAGINATION_PREV: "previous page",
  PAGINATION_NEXT: "next page",
  PAGINATION_LAST: "last page",
  DETAILS_TABS: "details tabs",
  DATA_GRID: "data grid",
  SEARCH_FIELD: "search",
  FILTER_FIELD: "filter",
  USER_MENU: "User menu",
  USER_MENU_TOGGLE: "Toggle user menu",
} as const;

// ============================================================================
// Error Boundary & Error State Messages
// ============================================================================

export const ERROR_BOUNDARY = {
  TITLE: "Something went wrong",
  DESCRIPTION: "An unexpected error occurred. Please try refreshing the page.",
  ROUTE_ERROR_TITLE: "Page Error",
  ROUTE_ERROR_DESCRIPTION: "This page encountered an error.",
  RETRY_BUTTON: "Retry",
  REFRESH_BUTTON: "Refresh Page",
  GO_HOME_BUTTON: "Go to Home",
  STACK_TRACE_SUMMARY: "Stack trace (development only)",
} as const;

// ============================================================================
// Global Loader Messages
// ============================================================================

export const LOADER = {
  LOADING: "Loading...",
  LOADING_DATA: "Loading data...",
  PLEASE_WAIT: "Please wait",
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
  TITLE_BASE_STATIONS: "Base Stations",
  TITLE_ENDPOINTS: "End Points",
  TITLE_CERTIFICATES: "Certificates",

  // Empty States
  EMPTY_NO_DATA: "No data available",
  EMPTY_NO_RESULTS: "No results found",
  EMPTY_NO_MESSAGES: "No messages",
  EMPTY_NO_ITEMS: "No items",

  // Actions
  ACTION_RETRY: "Retry",
  ACTION_RELOAD: "Reload",
  ACTION_GO_BACK: "Go Back",
  ACTION_VIEW_DETAILS: "View Details",
  ACTION_EXPAND: "Expand",
  ACTION_COLLAPSE: "Collapse",

  // Status
  STATUS_LOADING: "Loading...",
  STATUS_ERROR: "Error",
  STATUS_SUCCESS: "Success",
  STATUS_PENDING: "Pending",
  STATUS_COMPLETED: "Completed",
  STATUS_FAILED: "Failed",

  // Confirmation
  CONFIRM_DELETE: "Are you sure you want to delete this item?",
  CONFIRM_UNSAVED: "You have unsaved changes. Are you sure you want to leave?",

  // Search/Filter
  PLACEHOLDER_SEARCH: "Search...",
  PLACEHOLDER_FILTER: "Filter...",
  LABEL_SHOWING: "Showing",

  // Error Messages
  ERR_GENERIC: "An error occurred",
  ERR_LOAD_FAILED: "Failed to load data",
  ERR_SAVE_FAILED: "Failed to save changes",

  // Time
  TIME_NOW: "just now",
  TIME_AGO: "ago",
  TIME_NEVER: "Never",
} as const;

// ============================================================================
// Shared Pagination Labels
// ============================================================================

export const PAGINATION_LABELS = {
  ROWS_PER_PAGE: "Rows per page:",
  OF: "of",
} as const;

// ============================================================================
// Dashboard-specific Messages
// ============================================================================

export const DASHBOARD = {
  SECTION_OVERVIEW: "Overview",
  SECTION_ACTIVITY: "Recent Activity",
  SECTION_STATISTICS: "Statistics",
  CARD_TOTAL_BASE_STATIONS: "Total Base Stations",
  CARD_CONNECTED_BASE_STATIONS: "Connected Base Stations",
  CARD_TOTAL_ENDPOINTS: "Total Endpoints",
  CARD_ACTIVE_ENDPOINTS: "Active Endpoints",
  CARD_MESSAGES_TODAY: "Messages Today",
  CARD_UPLINK_MESSAGES: "Uplink Messages",
  CARD_DOWNLINK_MESSAGES: "Downlink Messages",
  EMPTY_NO_ACTIVITY: "No recent activity",
  LABEL_LAST_UPDATED: "Last Updated",
} as const;

// ============================================================================
// Table Column Headers (shared across data grids)
// ============================================================================

export const TABLE_HEADERS = {
  // Common
  COL_NAME: "Name",
  COL_EUI: "EUI",
  COL_STATUS: "Status",
  COL_CREATED: "Created",
  COL_UPDATED: "Updated",
  COL_ACTIONS: "Actions",
  COL_TYPE: "Type",
  COL_TIMESTAMP: "Timestamp",
  COL_DETAILS: "Details",

  // Base Stations
  COL_BS_NAME: "Base Station",
  COL_BS_EUI: "BS EUI",
  COL_CONNECTION: "Connection",
  COL_LAST_SEEN: "Last Seen",
  COL_MESSAGES: "Messages",

  // Endpoints
  COL_EP_NAME: "Endpoint",
  COL_EP_EUI: "EP EUI",
  COL_ATTACH_STATE: "Attach State",
  COL_BIDIRECTIONAL: "BiDi",

  // Messages
  COL_MESSAGE_ID: "Message ID",
  COL_DIRECTION: "Direction",
  COL_PAYLOAD: "Payload",
  COL_RSSI: "RSSI",
  COL_SNR: "SNR",
  COL_RX_TIME: "RX Time",

  // Downlink
  COL_QUEUE_ID: "Queue ID",
  COL_PRIORITY: "Priority",
  COL_PORT: "Port",
  COL_RESULT: "Result",
  COL_DELIVERED: "Delivered",
} as const;

// ============================================================================
// Base Stations Page Messages
// ============================================================================

export const BASE_STATIONS_PAGE = {
  // Page headers
  TITLE: "Base Stations",
  DETAILS_TITLE: "Base Station Details",
  // Actions
  REFRESH: "Refresh",
  ADD_BASESTATION: "Add Basestation",
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
  COL_ACTIONS: "Actions",
  // Status
  EXPIRED: "Expired",
  DAYS: "days",
} as const;

export const BASE_STATION_DETAILS = {
  // Section headers
  BASIC_INFORMATION: "Basic Information",
  CERTIFICATE_INFORMATION: "Certificate Information",
  TECHNICAL_DETAILS: "Technical Details",
  CONNECTION_STATUS: "Connection Status",
  SYSTEM_INFORMATION: "System Information",
  PERFORMANCE_METRICS: "Performance Metrics",
  // Field labels
  BASE_STATION_NAME: "Base Station Name",
  BASE_STATION_EUI: "Base Station EUI",
  CONNECTION_TYPE: "Connection Type",
  STATUS: "Status",
  FIRMWARE_VERSION: "Firmware Version",
  LAST_SEEN: "Last Seen",
  DATE_ADDED: "Date Added",
  CERTIFICATE_EXPIRY: "Certificate Expiry",
  CERTIFICATE_CN: "Certificate CN",
  IP_ADDRESS: "IP Address",
  PORT: "Port",
  PROTOCOL: "Protocol",
  UPTIME: "Uptime",
  MESSAGES_RECEIVED: "Messages Received",
  ENDPOINTS_ATTACHED: "Endpoints Attached",
  NOT_SET: "Not set",
  NOT_AVAILABLE: "Not available",
  SYSTEM_TIME: "System Time",
  LAST_STATUS_UPDATE: "Last Status Update",
  TEMPERATURE: "Temperature",
  CPU_LOAD: "CPU Load",
  MEMORY_LOAD: "Memory Load",
  DUTY_CYCLE: "Duty Cycle",
  BS_CONFIG: "BS Config",
  // Actions
  EDIT: "Edit",
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
    "Download the original certificates generated for this base station.",
  DOWNLOAD_CA: "Download CA",
  DOWNLOAD_CERT: "Download Certificate",
  DOWNLOAD_KEY: "Download Private Key",
  CERTIFICATE_DOWNLOAD_ERROR: "Failed to download certificate",
  NO_CERTIFICATES: "No certificates available",
  // Certificate regeneration
  ACTION_REGENERATE_CERTS: "Regenerate Certificates",
  REGENERATE_CERTS_CONFIRM_TITLE: "Regenerate Certificates?",
  REGENERATE_CERTS_CONFIRM_TEXT:
    "This will create new TLS certificates. You will need to upload them to the base station.",
  REGENERATE_CERTS_SUCCESS: "Certificates regenerated successfully",
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
  COL_MANUFACTURER: "Manufacturer",
  COL_MODEL: "Model",
  COL_STATUS: "Status",
  COL_LAST_SEEN: "Last Seen",
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
  // Actions
  GENERATE: "Generate Certificate",
  // Table headers
  COL_TYPE: "Type",
  // Status
  VALID: "Valid",
  EXPIRED: "Expired",
  // Empty states
  NO_CERTIFICATES: "No certificates found",
} as const;

// ============================================================================
// Dashboard Page Messages (extends existing DASHBOARD)
// ============================================================================

export const DASHBOARD_PAGE = {
  // Page headers
  TITLE: "Network Overview",
  SUBTITLE: "Real-time network health and status monitoring",
  // Section headers
  RECENT_ACTIVITY: "Platform Activity",
  ALERT_SUMMARY: "Alert Summary",
  SERVICE_CENTER_STATUS: "Service Center Status",
  // Card labels
  ACTIVE_BASE_STATIONS: "Active Base Stations",
  ALL_ONLINE: "All online",
  CONNECTED_ENDPOINTS: "Connected End Points",
  TOTAL_REGISTERED: "Total registered",
  NO_ENDPOINTS: "No endpoints",
  // Trend messages
  FROM_LAST_WEEK: "from last week",
  OFFLINE_LABEL: "offline",
  MESSAGE_TRAFFIC: "Message Traffic",
  LAST_24_HOURS: "Last 24 hours",
  // Realtime
  REALTIME_CONNECTION: "Realtime Connection",
  LIVE: "Live",
  // Alerts
  CRITICAL_ALERTS: "Critical Alerts",
  WARNINGS: "Warnings",
  INFO_MESSAGES: "Info Messages",
  ALL_SYSTEMS_OPERATIONAL: "All systems operational",
} as const;

// ============================================================================
// Base Station Messages Component Messages
// ============================================================================

export const BASE_STATION_MESSAGES = {
  // Title
  TITLE_PREFIX: "Activity for ",
  TITLE_FALLBACK_PREFIX: "Base Station ",
  // Actions
  REFRESH: "Refresh",
  EXPORT_CSV: "Export CSV",
  EXPORT_JSON: "Export JSON",
  // Filter labels
  LABEL_START_DATE: "Start Date",
  LABEL_END_DATE: "End Date",
  // Loading states
  LOADING: "Loading...",
  // Stats labels
  TOTAL_MESSAGES: "Total Messages",
  MESSAGES_TODAY: "Messages Today",
  UNIQUE_ENDPOINTS: "Unique Endpoints",
  AVG_SIGNAL_QUALITY: "Avg Signal Quality",
  RSSI_PREFIX: "RSSI: ",
  SNR_PREFIX: " / SNR: ",
  // Filter labels
  SEARCH: "Search",
  DIRECTION: "Direction",
  DIRECTION_ALL: "All",
  DIRECTION_UPLINK: "Uplink",
  DIRECTION_DOWNLINK: "Downlink",
  ENDPOINT_EUI: "Endpoint EUI",
  CLEAR_FILTERS: "Clear Filters",
  // Date input (EU format: DD/MM/YYYY)
  DATE_PLACEHOLDER_EU: "DD/MM/YYYY",
  DATE_HELPER_EU: "Format: DD/MM/YYYY",
  ERR_DATE_INVALID: "Invalid date format. Use DD/MM/YYYY.",
  // Table headers
  COL_TIME: "Time",
  COL_TYPE: "Type",
  COL_DIRECTION: "Direction",
  COL_ENDPOINT: "Endpoint",
  COL_PACKET_NUM: "Packet #",
  COL_RSSI: "RSSI",
  COL_SNR: "SNR",
  COL_DATA: "Data",
  COL_FLAGS: "Flags",
  // Chip labels
  DL_OPEN: "DL Open",
  RESPONSE: "Response",
  ROAMING: "Roaming",
  // Empty state
  NO_MESSAGES: "No messages found",
  NO_ACTIVITY: "No activity found",
  // Export
  EXPORT_FILENAME_PREFIX: "basestation-",
  EXPORT_FILENAME_SUFFIX: "-messages-",
  EXPORT_CSV_HEADER: "id,ep_eui,bs_eui,rx_time,packet_cnt,snr,rssi,user_data",
  EXPORT_JSON_EMPTY: "[]",
  ERR_EXPORT_NO_DATA: "No messages found for the selected date range.",
  // Error messages
  ERR_SEARCH_FAILED: "Failed to search messages",
  ERR_EXPORT_FAILED: "Failed to export messages",
  // Activity feed labels
  ACTIVITY_TYPE_EVENT: "Event",
  ACTIVITY_TYPE_MESSAGE: "Message",
} as const;

// ============================================================================
// Generate Certificate Dialog Messages
// ============================================================================

export const GENERATE_CERTIFICATE_DIALOG = {
  // Dialog
  TITLE: "Generate New Server Certificate",
  // Warning message
  WARNING_TEXT:
    "Generating a new server certificate will replace the existing one. Base stations that trust the Certificate Authority (CA) will automatically accept the new certificate. Ensure certificate rotation is performed before the current certificate expires to avoid service interruption.",
  // Form labels
  VALIDITY_PERIOD: "Validity Period",
  // Validity options
  ONE_YEAR: "1 Year",
  TWO_YEARS: "2 Years",
  THREE_YEARS: "3 Years (Maximum)",
  // Actions
  CANCEL: "Cancel",
  GENERATE: "Generate Certificate",
} as const;

// ============================================================================
// Certificate Info Alert Messages
// ============================================================================

export const CERTIFICATE_INFO = {
  ALERT_TITLE: "About Server Certificates",
  ALERT_TEXT:
    "Server certificates are used by the BSSCI server to establish secure TLS connections with base stations. Base stations trust the Certificate Authority (CA) that signs these server certificates, not the individual server certificates themselves. This allows server certificates to be rotated without requiring updates to base stations. The CA certificate has a long validity period (20 years) while server certificates can be rotated periodically for security best practices.",
  // Field labels
  PATH: "Path",
  ISSUER: "Issuer",
  SUBJECT: "Subject",
  SERIAL: "Serial",
  EXPIRES: "Expires",
  DAYS_UNTIL_EXPIRY: "Days until expiry",
  EXPIRES_IN_DAYS: "Expires in {days} days",
  EXPIRES_IN_PREFIX: "Expires in ",
  EXPIRES_IN_SUFFIX: " days",
} as const;

// ============================================================================
// Server Certificate Actions
// ============================================================================

export const SERVER_CERTIFICATES = {
  GENERATE_BUTTON: "Generate Server Certificates",
  RENEW_BUTTON: "Renew Server Certificates",
  RENEW_CONFIRM_TITLE: "Renew Server Certificates",
  RENEW_CONFIRM_TEXT:
    "Renewing server certificates will require a service restart. Are you sure you want to proceed?",
  GENERATE_SUCCESS:
    "Server certificates generated successfully. Restart the service to apply.",
  RENEW_SUCCESS:
    "Server certificates renewed successfully. Restart the service to apply.",
  RESTART_REMINDER:
    "A service restart is required for the new certificates to take effect.",
  CANCEL: "Cancel",
} as const;

// ============================================================================
// Users - Errors
// ============================================================================

export const ERR_LOAD_USERS = "Failed to load users. Please try again.";
export const ERR_CREATE_USER = "Failed to create user";
export const ERR_UPDATE_USER = "Failed to update user";
export const ERR_DELETE_USER = "Failed to delete user";
export const ERR_CHANGE_PASSWORD = "Failed to change password";

// ============================================================================
// Users - Success
// ============================================================================

export const MSG_USER_CREATED = "User created successfully";
export const MSG_USER_UPDATED = "User updated successfully";
export const MSG_USER_DELETED = "User deleted successfully";
export const MSG_PASSWORD_CHANGED = "Password changed successfully";

// ============================================================================
// Users Page Messages
// ============================================================================

export const USERS_PAGE = {
  // Page header
  TITLE: "Users & Roles",
  // Breadcrumb constants (no inline literals in UI)
  BREADCRUMB_ROOT: "Network Server",
  BREADCRUMB_SEPARATOR: " > ",
  BREADCRUMB_USERS: "Users & Roles",
  // Actions
  ADD_USER: "Add user",
  BACK_TO_LIST: "Back to list",
  BACK_TO_DETAIL: "Back to details",
  // Search
  SEARCH_PLACEHOLDER: "Search users...",
  // Empty states
  NO_USERS: "No users found",
  NO_MATCH: "No users match your search",
  // Table columns
  COL_EMAIL: "Email",
  COL_ADMIN: "Is admin",
  COL_ACTIVE: "Is active",
  COL_TENANT_MGR: "Is tenant manager",
  COL_BS_MGR: "Is base station manager",
  COL_EP_MGR: "Is endpoint manager",
  COL_NOTE: "Note",
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
  DIALOG_TITLE_ADD: "Add user",
  DIALOG_TITLE_EDIT: "Edit user",
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
  ERR_PASSWORD_MISMATCH: "The password does not match!",
  ERR_CREATE_FAILED: "Failed to create user",
  ERR_UPDATE_FAILED: "Failed to update user",
  ERR_CHANGE_PASSWORD_FAILED: "Failed to change password",
  // Actions
  ACTION_SUBMIT: "Submit",
  ACTION_CANCEL: "Cancel",
  ACTION_CHANGE_PASSWORD: "Change password",
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
  TOOLTIP_REFRESH: "Refresh",
} as const;

export const API_KEY_FORM = {
  // Dialog titles
  DIALOG_TITLE_CREATE: "Create API Key",
  // Field labels
  LABEL_NAME: "Key Name",
  LABEL_TYPE: "Key Type",
  LABEL_EXPIRES_AT: "Expires At",
  // Placeholders
  PLACEHOLDER_NAME: "Enter a descriptive name for this key",
  // Validation
  ERR_NAME_REQUIRED: "Key name is required",
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
export const ERR_CREATE_ORGANIZATION = "Failed to create organization";
export const ERR_UPDATE_ORGANIZATION = "Failed to update organization";
export const ERR_DELETE_ORGANIZATION = "Failed to delete organization";
export const ERR_LOAD_ORG_USERS = "Failed to load organization members";
export const ERR_ADD_ORG_USER = "Failed to add user to organization";
export const ERR_UPDATE_ORG_USER = "Failed to update organization member";
export const ERR_REMOVE_ORG_USER = "Failed to remove user from organization";

// ============================================================================
// Organizations - Success
// ============================================================================

export const MSG_ORGANIZATION_CREATED = "Organization created successfully";
export const MSG_ORGANIZATION_UPDATED = "Organization updated successfully";
export const MSG_ORGANIZATION_DELETED = "Organization archived successfully";
export const MSG_ORG_USER_ADDED = "User added to organization successfully";
export const MSG_ORG_USER_UPDATED = "Organization member updated successfully";
export const MSG_ORG_USER_REMOVED =
  "User removed from organization successfully";

// ============================================================================
// Organizations Page Messages
// ============================================================================

export const ORGANIZATIONS_PAGE = {
  // Page header
  TITLE: "Organizations",
  // Breadcrumb
  BREADCRUMB_ROOT: "Network Server",
  BREADCRUMB_SEPARATOR: " > ",
  BREADCRUMB_ORGANIZATIONS: "Organizations",
  // Accessibility
  ARIA_TABS: "organization detail tabs",
  // Actions
  ADD_ORGANIZATION: "Add organization",
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
  COL_CAN_HAVE_BS: "Base Stations",
  COL_BS_QUOTA: "BS Quota",
  COL_EP_QUOTA: "EP Quota",
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
  QUOTA_UNLIMITED: "Unlimited",
  QUOTA_ALLOWED: "Allowed",
  QUOTA_NOT_ALLOWED: "Not allowed",
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
  DIALOG_TITLE_ADD: "Add organization",
  DIALOG_TITLE_EDIT: "Edit organization",
  // Field labels
  LABEL_NAME: "Name",
  LABEL_DESCRIPTION: "Description (optional)",
  LABEL_STATE: "State",
  LABEL_CAN_HAVE_BS: "Can have base stations",
  LABEL_MAX_BS_COUNT: "Max base station count (optional)",
  LABEL_MAX_EP_COUNT: "Max endpoint count (optional)",
  LABEL_TAGS: "Tags",
  // Helper text
  HELPER_NAME: "Organization display name",
  HELPER_DESCRIPTION: "Optional description for the organization",
  HELPER_CAN_HAVE_BS: "Whether this organization can own base stations",
  HELPER_MAX_BS_COUNT: "Leave empty for unlimited",
  HELPER_MAX_EP_COUNT: "Leave empty for unlimited",
  // Validation errors
  ERR_NAME_REQUIRED: "Name is required",
  ERR_CREATE_FAILED: "Failed to create organization",
  ERR_UPDATE_FAILED: "Failed to update organization",
  // Actions
  ACTION_SUBMIT: "Submit",
  ACTION_CANCEL: "Cancel",
  ACTION_DELETE: "Delete organization",
  ACTION_RESTORE: "Restore organization",
  CONFIRM_DELETE:
    "Are you sure you want to delete this organization? This action cannot be undone.",
  // Success messages
  SUCCESS_UPDATE: "Organization updated successfully",
  // Info section
  INFO_TITLE: "Info",
  INFO_TENANT_ID: "Tenant ID",
  INFO_CREATED: "Created",
  INFO_UPDATED: "Updated",
} as const;

// ============================================================================
// Organization Users Page Messages
// ============================================================================

export const ORG_USERS_PAGE = {
  // Page header
  TITLE: "Organization Members",
  // Actions
  ADD_USER: "Add user",
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
  COL_ORG_ADMIN: "Org Admin",
  COL_BS_ADMIN: "BS Admin",
  COL_EP_ADMIN: "EP Admin",
  COL_JOINED: "Joined",
  COL_ACTIONS: "Actions",
  // Role labels
  ROLE_OWNER: "Owner",
  ROLE_ADMIN: "Admin",
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
  DIALOG_TITLE_ADD: "Add member",
  DIALOG_TITLE_EDIT: "Edit member",
  // Field labels
  LABEL_USER_ID: "User",
  LABEL_ROLE: "Role",
  LABEL_STATUS: "Status",
  LABEL_IS_ORG_ADMIN: "Organization admin",
  LABEL_IS_BS_ADMIN: "Base station admin",
  LABEL_IS_EP_ADMIN: "Endpoint admin",
  // Helper text
  HELPER_ORG_ADMIN: "Can manage organization settings and members",
  HELPER_BS_ADMIN: "Can manage base stations for this organization",
  HELPER_EP_ADMIN: "Can manage endpoints for this organization",
  // Validation errors
  ERR_USER_REQUIRED: "User is required",
  ERR_ROLE_REQUIRED: "Role is required",
  ERR_ADD_FAILED: "Failed to add member",
  ERR_UPDATE_FAILED: "Failed to update member",
  ERR_REMOVE_FAILED: "Failed to remove member",
  // Actions
  ACTION_SUBMIT: "Submit",
  ACTION_CANCEL: "Cancel",
  ACTION_REMOVE: "Remove member",
  CONFIRM_REMOVE:
    "Are you sure you want to remove this member from the organization?",
} as const;

// ============================================================================
// Tags Editor Messages
// ============================================================================

export const TAGS_EDITOR = {
  // Labels
  LABEL_KEY: "Key",
  LABEL_VALUE: "Value",
  LABEL_ADD_TAG: "Add tag",
  LABEL_NO_TAGS: "No tags",
  // Placeholder
  PLACEHOLDER_KEY: "Enter key",
  PLACEHOLDER_VALUE: "Enter value",
  // Actions
  ACTION_ADD: "Add",
  ACTION_REMOVE: "Remove tag",
  // Validation
  ERR_KEY_REQUIRED: "Key is required",
  ERR_DUPLICATE_KEY: "Duplicate key",
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
  SUCCESS: "Password changed successfully",
  ERR_MISMATCH: "Passwords do not match",
  ERR_WEAK: "Password does not meet minimum requirements",
  ERR_FAILED: "Failed to change password. Please try again.",
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
  DEVICE_CATALOG: "Device Catalog",

  // Actions
  ADD_MANUFACTURER: "Add Manufacturer",
  ADD_MODEL: "Add Model",
  ADD_BLUEPRINT: "Add Blueprint",
  ADD_DECODER: "Add Decoder",
  SET_DEFAULT: "Set as Default",
  SUBMIT_TO_REGISTRY: "Submit to Registry",
  TEST_DECODE: "Test Decode",

  // Form labels - Manufacturer
  LABEL_NAME: "Name",
  LABEL_CODE: "Code",
  LABEL_WEBSITE: "Website",
  LABEL_DESCRIPTION: "Description",
  LABEL_LOGO_URL: "Logo URL",

  // Form labels - Device Model
  LABEL_MANUFACTURER: "Manufacturer",
  LABEL_DATASHEET_URL: "Datasheet URL",

  // Form labels - Blueprint
  LABEL_TYPE_EUI: "Type EUI",
  LABEL_VERSION: "Version",
  LABEL_SPEC_JSON: "Blueprint Specification (JSON)",
  LABEL_TEST_DATA: "Test Payload (hex)",
  LABEL_FORMAT_ID: "Format ID",
  LABEL_CONTRIBUTOR_NAME: "Contributor Name",
  LABEL_CONTRIBUTOR_EMAIL: "Contributor Email",

  // Helper text
  HELPER_CODE: "URL-friendly identifier (lowercase, no spaces)",
  HELPER_TYPE_EUI: "8-byte Type EUI in hex format (16 characters)",
  HELPER_SPEC_JSON: "Blueprint JSON per MIOTY Application Layer Specification",
  HELPER_TEST_DATA: "Hex-encoded payload to test decoding",

  // Validation errors
  ERR_INVALID_JSON: "Invalid JSON format",
  ERR_MISSING_VERSION: "Missing required field: version",
  ERR_MISSING_TYPE_EUI: "Missing required field: typeEui",
  ERR_MISSING_UPLINK: "At least one uplink definition required",
  ERR_NAME_REQUIRED: "Name is required",
  ERR_MANUFACTURER_REQUIRED: "Manufacturer is required",
  ERR_CODE_REQUIRED: "Code is required",
  ERR_VERSION_REQUIRED: "Version is required",
  ERR_TYPE_EUI_REQUIRED: "Type EUI is required",
  ERR_TYPE_EUI_FORMAT: "Type EUI must be 16 hex characters (8 bytes)",
  ERR_SPEC_JSON_REQUIRED: "Blueprint specification is required",
  ERR_CONTRIBUTOR_NAME_REQUIRED: "Contributor name is required",
  ERR_CONTRIBUTOR_EMAIL_REQUIRED: "Contributor email is required",
  ERR_NO_MANUFACTURER_SELECTED: "No manufacturer selected",
  ERR_NO_MODEL_SELECTED: "No model selected",

  // Decode status display (human-readable)
  DECODE_SUCCESS: "Decoded successfully",
  DECODE_FAILED: "Decode failed",
  DECODE_PENDING: "Pending decode",
  DECODE_SKIPPED: "No blueprint available",

  // Success messages
  MSG_MANUFACTURER_CREATED: "Manufacturer created successfully",
  MSG_MANUFACTURER_UPDATED: "Manufacturer updated successfully",
  MSG_MANUFACTURER_DELETED: "Manufacturer deleted successfully",
  MSG_MODEL_CREATED: "Device model created successfully",
  MSG_MODEL_UPDATED: "Device model updated successfully",
  MSG_MODEL_DELETED: "Device model deleted successfully",
  MSG_BLUEPRINT_CREATED: "Blueprint created successfully",
  MSG_BLUEPRINT_UPDATED: "Blueprint updated successfully",
  MSG_BLUEPRINT_DELETED: "Blueprint deleted successfully",
  MSG_BLUEPRINT_SET_DEFAULT: "Blueprint set as default",

  // Error messages
  ERR_LOAD_MANUFACTURERS: "Failed to load manufacturers",
  ERR_LOAD_MODELS: "Failed to load device models",
  ERR_LOAD_BLUEPRINTS: "Failed to load blueprints",
  ERR_CREATE_MANUFACTURER: "Failed to create manufacturer",
  ERR_CREATE_MODEL: "Failed to create device model",
  ERR_CREATE_BLUEPRINT: "Failed to create blueprint",
  ERR_DECODE_PREVIEW: "Failed to decode payload",
  ERR_BLUEPRINT_NOT_FOUND: "Blueprint not found",
  ERR_BLUEPRINT_ID_REQUIRED: "Blueprint ID is required",
  ERR_DELETE_FAILED: "Delete failed",
  ERR_MODEL_WITH_BLUEPRINT: "Failed to create device model with blueprint",

  // Empty states
  NO_MANUFACTURERS: "No manufacturers found",
  NO_MODELS: "No device models found",
  NO_BLUEPRINTS: "No blueprints found",
  NO_DECODE_RESULT: "No decode result",

  // Table headers
  COL_NAME: "Name",
  COL_CODE: "Code",
  COL_MODELS: "Models",
  COL_BLUEPRINTS: "Blueprints",
  COL_VERSION: "Version",
  COL_TYPE_EUI: "Type EUI",
  COL_DEFAULT: "Default",
  COL_CREATED: "Created",
  COL_ACTIONS: "Actions",

  // Misc
  BADGE_DEFAULT: "Default",
  BADGE_VERIFIED: "Verified",

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
  CONFIRM_DELETE_BLUEPRINT:
    "Delete this blueprint? This action cannot be undone.",

  // Edit actions
  ACTION_DELETE: "Delete",
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
  ERR_REGISTRY_SUBMIT: "Failed to submit blueprint to registry",
  ERR_REGISTRY_NOT_CONFIGURED: "Registry service is not configured",
} as const;

// ============================================================================
// Realtime Connection Messages
// ============================================================================

export const REALTIME_MESSAGES = {
  // Connection attempt
  CONNECTING: "Connecting to gRPC stream",
  CONNECTING_AT: "Connecting to gRPC stream at",

  // Connection status
  CONNECTED: "gRPC stream connected",
  DISCONNECTED: "gRPC stream disconnected",
  STREAM_ENDED_NORMAL: "Stream ended normally",
  STREAM_ENDED_ERROR: "Stream ended with error",
  FAILED_ESTABLISH: "Failed to establish gRPC stream",
  RECONNECTING: "Reconnecting to gRPC stream",
  RECONNECTING_IN: "Reconnecting in",

  // Base station stream
  BS_STREAM_CONNECTED: "Base station stream connected",
  BS_STREAM_ERROR: "Base station stream error",
  BS_EUI_REQUIRED: "bsEui required for base station stream",

  // Event stream
  EVENT_STREAM_CONNECTED: "Event stream connected",
  EVENT_STREAM_ERROR: "Event stream error",

  // Context guards
  CONTEXT_INCOMPLETE:
    "Realtime connection deferred: missing auth, org, or user context",
  CONTEXT_RECONNECT_DEFERRED: "Reconnect deferred: context not available",
} as const;

// ============================================================================
// Certificate Labels (User-Facing)
// ============================================================================

export const CERTIFICATE_LABELS = {
  SERVER: "Server Certificate",
  CA: "CA Certificate",
  UNKNOWN: "Unknown Certificate",
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
  INVALID_OIDC_EXCHANGE_RESPONSE: "Invalid OIDC exchange response",
  INVALID_OAUTH2_EXCHANGE_RESPONSE: "Invalid OAuth2 exchange response",

  // Analytics/Dashboard
  INVALID_ANALYTICS_RESPONSE: "Invalid analytics response",
  INVALID_ALERT_SUMMARY_RESPONSE: "Invalid alert summary response",

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
  INVALID_CREATE_DEVICE_MODEL_RESPONSE: "Invalid create device model response",
  INVALID_CREATE_BLUEPRINT_RESPONSE: "Invalid create blueprint response",

  // API Keys
  API_KEY_CREATION_FAILED: "API key creation failed",
} as const;

export const BRAND = {
  SOURCE: "Source Code",
  DOCUMENTATION: "Documentation",
  LICENSE: "License",
} as const;
