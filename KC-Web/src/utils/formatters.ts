import { UI_COMMON } from '@constants/messages';
import { formatTime24h } from '@theme/index';

// Centralized locale configuration for MIOTY
// Using 'en-GB' for day-first format and natural 24-hour time support
// Document: MIOTY spec requires 24-hour format, using European date format for consistency
const MIOTY_LOCALE = 'en-GB'; // DD/MM/YYYY format with 24-hour time

// ============================================================================
// Number Formatting
// ============================================================================

// Format numbers with thousand separators
export const formatNumber = (num: number): string => {
  if (num === null || num === undefined) return '0';
  return num.toLocaleString(MIOTY_LOCALE);
};

// ISO-style date format (YYYY-MM-DD) for technical displays
export const formatISODate = (date: Date | string | number): string => {
  const d = new Date(date);
  return d.toISOString().split('T')[0]; // YYYY-MM-DD
};

// Full date and time in 24-hour format (DD/MM/YYYY HH:mm:ss)
export const formatDateTime = (date: Date | string | number): string => {
  const d = new Date(date);
  const dateStr = d.toLocaleDateString(MIOTY_LOCALE); // DD/MM/YYYY
  return `${dateStr} ${formatTime24h(d)}`; // DD/MM/YYYY HH:mm:ss
};

// Date only (DD/MM/YYYY)
export const formatDate = (date: Date | string | number): string => {
  return new Date(date).toLocaleDateString(MIOTY_LOCALE);
};

// Relative date formatting (TODAY, YESTERDAY, or DD/MM/YYYY)
export const formatRelativeDate = (date: Date | string | number): string => {
  const d = new Date(date);
  const today = new Date().toDateString();
  const yesterday = new Date(Date.now() - 86400000).toDateString();

  if (d.toDateString() === today) return 'TODAY';
  if (d.toDateString() === yesterday) return 'YESTERDAY';
  return formatDate(d);
};

// Alternative: Technical timestamp format (ISO-style for logs/debug)
export const formatTechnicalTimestamp = (date: Date | string | number): string => {
  const d = new Date(date);
  return `${formatISODate(d)} ${formatTime24h(d)}`;
};

// Re-export formatTime24h for convenience (one-way dependency from theme)
export { formatTime24h };

// ============================================================================
// Relative Duration Formatting (for last seen, uptime, etc.)
// ============================================================================

/**
 * Format relative duration from a timestamp
 * Returns presentation-agnostic string: "5 minutes ago", "2 hours ago", etc.
 * Extracted from EndPoints.tsx:61-80
 */
export const formatRelativeDuration = (timestamp: string | undefined): string => {
  if (!timestamp) return UI_COMMON.TIME_NEVER;

  const timestampDate = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - timestampDate.getTime();
  const diffMinutes = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffHours / 24);

  if (diffMinutes < 1) {
    return UI_COMMON.TIME_NOW;
  } else if (diffHours < 1) {
    return `${diffMinutes} minute${diffMinutes !== 1 ? 's' : ''} ${UI_COMMON.TIME_AGO}`;
  } else if (diffHours < 24) {
    return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ${UI_COMMON.TIME_AGO}`;
  } else if (diffDays < 7) {
    return `${diffDays} day${diffDays !== 1 ? 's' : ''} ${UI_COMMON.TIME_AGO}`;
  } else {
    return formatDate(timestampDate);
  }
};

// ============================================================================
// Battery Level Formatting
// ============================================================================

export interface BatteryLevelFormat {
  label: string;
  severity: 'success' | 'warning' | 'error';
}

/**
 * Format battery level with presentation-agnostic severity
 * Components map severity → theme tokens, not hex codes
 *
 * @param level - Battery percentage (0-100)
 * @returns Object with label and semantic severity
 */
export const formatBatteryLevel = (level: number | undefined): BatteryLevelFormat => {
  if (level === undefined) {
    return { label: '-', severity: 'success' };
  }

  const label = `${level}%`;

  if (level > 50) {
    return { label, severity: 'success' };
  } else if (level > 20) {
    return { label, severity: 'warning' };
  } else {
    return { label, severity: 'error' };
  }
};

// ============================================================================
// POSIX Error Formatting (SCACI §3.14)
// ============================================================================

/**
 * Map POSIX error codes to human-readable messages
 * Per SCACI v1.0.0 §3.14 error handling requirements
 *
 * @param code - POSIX error code
 * @returns Human-readable error message
 */
export const formatPosixError = (code: number): string => {
  const posixErrors: Record<number, string> = {
    1: 'Operation not permitted (EPERM)',
    2: 'No such file or directory (ENOENT)',
    3: 'No such process (ESRCH)',
    4: 'Interrupted system call (EINTR)',
    5: 'I/O error (EIO)',
    6: 'No such device or address (ENXIO)',
    7: 'Argument list too long (E2BIG)',
    8: 'Exec format error (ENOEXEC)',
    9: 'Bad file number (EBADF)',
    10: 'No child processes (ECHILD)',
    11: 'Try again (EAGAIN)',
    12: 'Out of memory (ENOMEM)',
    13: 'Permission denied (EACCES)',
    14: 'Bad address (EFAULT)',
    16: 'Device or resource busy (EBUSY)',
    17: 'File exists (EEXIST)',
    18: 'Cross-device link (EXDEV)',
    19: 'No such device (ENODEV)',
    20: 'Not a directory (ENOTDIR)',
    21: 'Is a directory (EISDIR)',
    22: 'Invalid argument (EINVAL)',
    23: 'File table overflow (ENFILE)',
    24: 'Too many open files (EMFILE)',
    28: 'No space left on device (ENOSPC)',
    32: 'Broken pipe (EPIPE)',
    61: 'No data available (ENODATA)',
    71: 'Protocol error (EPROTO)',
    110: 'Connection timed out (ETIMEDOUT)',
    111: 'Connection refused (ECONNREFUSED)',
  };

  return posixErrors[code] || `Unknown error (code ${code})`;
};

// ============================================================================
// MessagePack Payload Formatting
// ============================================================================

/**
 * Format base64 MessagePack payload for preview
 * Decodes base64 → hex preview
 *
 * @param payload - Base64-encoded payload string
 * @returns Hex string with 0x prefix, or error message
 */
export const formatMessagePackPayload = (payload: string): string => {
  if (!payload) return '0x';

  try {
    const bytes = atob(payload);
    const hex = Array.from(bytes, (c) => c.charCodeAt(0).toString(16).padStart(2, '0')).join('');
    return `0x${hex}`;
  } catch {
    return 'Invalid base64 payload';
  }
};

// ============================================================================
// Session Duration Formatting (SCACI)
// ============================================================================

/**
 * Format session duration for SCACI dashboards
 * Converts seconds to human-readable format
 *
 * @param seconds - Duration in seconds
 * @returns Formatted duration string
 */
export const formatSessionDuration = (seconds: number): string => {
  if (seconds < 60) {
    return `${seconds}s`;
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
  } else if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600);
    const remainingMinutes = Math.floor((seconds % 3600) / 60);
    return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`;
  } else {
    const days = Math.floor(seconds / 86400);
    const remainingHours = Math.floor((seconds % 86400) / 3600);
    return remainingHours > 0 ? `${days}d ${remainingHours}h` : `${days}d`;
  }
};

// ============================================================================
// Signal Quality Formatting (BSSCI)
// ============================================================================

/**
 * Format RSSI (Received Signal Strength Indicator) value
 * Per BSSCI v1.0.0 specification
 *
 * @param rssi - RSSI value in dBm
 * @returns Formatted RSSI string with unit
 */
export const formatRSSI = (rssi: number | undefined | null): string => {
  if (rssi === undefined || rssi === null) return '-';
  return `${rssi.toFixed(1)} dBm`;
};

/**
 * Format SNR (Signal-to-Noise Ratio) value
 * Per BSSCI v1.0.0 specification
 *
 * @param snr - SNR value in dB
 * @returns Formatted SNR string with unit
 */
export const formatSNR = (snr: number | undefined | null): string => {
  if (snr === undefined || snr === null) return '-';
  return `${snr.toFixed(1)} dB`;
};

// ============================================================================
// EUI Formatting (Base Station / Endpoint identifiers)
// ============================================================================

/**
 * Format EUI with dashes: XXXXXXXXXXXXXXXX → XX-XX-XX-XX-XX-XX-XX-XX
 * Used for BSSCI/SCACI protocol display and API formatting.
 *
 * @param eui - EUI string (with or without dashes, any case)
 * @returns Formatted EUI in XX-XX-XX-XX-XX-XX-XX-XX format (uppercase)
 */
export const formatEUIWithDashes = (eui: string): string => {
  const cleaned = eui.replace(/[^0-9A-Fa-f]/g, '');
  const parts: string[] = [];
  for (let i = 0; i < cleaned.length && i < 16; i += 2) {
    parts.push(cleaned.substring(i, i + 2));
  }
  return parts.join('-').toUpperCase();
};

// ============================================================================
// Dashboard Card Trend Formatting
// ============================================================================

/**
 * Format dashboard status card trend text as "+{percent}% +{count} {suffix}"
 * Dashboard-specific: always prefixes + and clamps to non-negative values.
 * Used ONLY by Dashboard.tsx for base station status card trend chips.
 *
 * @param percent - Online percentage (0-100), clamped to >= 0
 * @param count - Items added in period, clamped to >= 0
 * @param suffixLabel - Centralized suffix (e.g., DASHBOARD_PAGE.FROM_LAST_WEEK)
 * @returns Formatted trend string for dashboard status cards
 */
export const formatDashboardCardTrend = (
  percent: number,
  count: number,
  suffixLabel: string
): string => {
  // Dashboard cards always show non-negative values with + prefix
  const clampedPercent = Math.max(0, Math.round(percent));
  const clampedCount = Math.max(0, count);
  return `+${clampedPercent}% +${clampedCount} ${suffixLabel}`;
};

/**
 * Format dashboard status card trend text as "+{count} {suffix}" (count only, no percentage)
 * Dashboard-specific: used for entities where online/offline status cannot be determined.
 * Used ONLY by Dashboard.tsx for endpoint status card trend chips.
 *
 * @param count - Items added in period, clamped to >= 0
 * @param suffixLabel - Centralized suffix (e.g., DASHBOARD_PAGE.FROM_LAST_WEEK)
 * @returns Formatted trend string for dashboard status cards
 */
export const formatDashboardCountTrend = (count: number, suffixLabel: string): string => {
  const clampedCount = Math.max(0, count);
  return `+${clampedCount} ${suffixLabel}`;
};

// ============================================================================
// Pagination Helpers
// ============================================================================

/**
 * Paginate an array of items (client-side only).
 * Do NOT use for server-paged lists (e.g., EndPointDetails messages).
 *
 * @param items - Array of items to paginate
 * @param page - Zero-indexed page number
 * @param pageSize - Number of items per page
 * @returns Sliced array for the requested page
 */
export function paginate<T>(items: T[], page: number, pageSize: number): T[] {
  const start = page * pageSize;
  return items.slice(start, start + pageSize);
}

// ============================================================================
// Blueprint Decoded Payload Formatting
// ============================================================================

/**
 * Format decoded payload for terminal display
 * Converts JSON object to compact key=value format
 *
 * @param payload - Decoded payload object from blueprint decoding
 * @returns Formatted string for terminal display (e.g., "temp=23.45 humidity=65.2")
 */
export const formatDecodedPayload = (payload: Record<string, unknown>): string => {
  if (!payload || typeof payload !== 'object') return '';

  return Object.entries(payload)
    .map(([key, value]) => {
      if (typeof value === 'number') {
        // Format numbers with appropriate precision
        return `${key}=${Number.isInteger(value) ? value : value.toFixed(2)}`;
      }
      return `${key}=${value}`;
    })
    .join(' ');
};

/**
 * Format Type EUI for display
 * Converts 16-char hex string to XX:XX:XX:XX:XX:XX:XX:XX format
 *
 * @param typeEui - 16-character hex string (8 bytes)
 * @returns Formatted Type EUI with colons
 */
export const formatTypeEUI = (typeEui: string | undefined): string => {
  if (!typeEui) return '-';

  const cleaned = typeEui.replace(/[^0-9A-Fa-f]/g, '');
  if (cleaned.length !== 16) return typeEui; // Return as-is if invalid

  const parts: string[] = [];
  for (let i = 0; i < 16; i += 2) {
    parts.push(cleaned.substring(i, i + 2));
  }
  return parts.join(':').toUpperCase();
};

// ============================================================================
// EUI Validation (Shared Internal Helper)
// ============================================================================

/**
 * Internal: shared hex-length EUI validation
 * Used by isValidEUI and isValidTypeEUI to avoid duplicate logic.
 *
 * @param eui - EUI string (with or without dashes/colons)
 * @param expectedLength - Expected hex character count (16 for 8-byte EUI)
 * @returns true if valid, false otherwise
 */
const isValidHexEUI = (eui: string, expectedLength: number): boolean => {
  if (!eui) return false;
  const cleaned = eui.replace(/[^0-9A-Fa-f]/g, '');
  return cleaned.length === expectedLength;
};

/**
 * Validate EUI format (8-byte, 16 hex chars)
 * Accepts dashed (XX-XX-XX-XX-XX-XX-XX-XX) or non-dashed format.
 *
 * @param eui - EUI string to validate
 * @returns true if valid, false otherwise
 */
export const isValidEUI = (eui: string): boolean => isValidHexEUI(eui, 16);

/**
 * Validate Type EUI format
 * Must be exactly 16 hex characters (8 bytes)
 *
 * @param typeEui - Type EUI string to validate
 * @returns true if valid, false otherwise
 */
export const isValidTypeEUI = (typeEui: string): boolean => isValidHexEUI(typeEui, 16);

/**
 * Validate hex string format
 *
 * @param value - String to validate
 * @returns true if valid hex with even length, false otherwise
 */
export const isValidHexString = (value: string): boolean => {
  if (!value) return false;
  const hexRegex = /^[0-9A-Fa-f]*$/;
  return hexRegex.test(value) && value.length % 2 === 0;
};

/**
 * Format hex string with 0x prefix
 *
 * @param hex - Hex string (without prefix)
 * @returns Formatted hex string with 0x prefix (uppercase)
 */
export const formatHexWithPrefix = (hex: string | undefined): string => {
  if (!hex) return '0x';
  return `0x${hex.toUpperCase()}`;
};

/**
 * Convert hex string to Uint8Array for gRPC bytes fields
 *
 * @param hex - Hex string to convert (must have even length)
 * @throws Error with VAL_INVALID_HEX_STRING message if invalid hex
 * @returns Uint8Array of bytes
 */
export const bytesToHex = (bytes: Uint8Array | string): string => {
  if (typeof bytes === 'string') return bytes;
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
};

export const hexToBytes = (hex: string): Uint8Array => {
  if (!hex || hex.length % 2 !== 0 || !isValidHexString(hex)) {
    // Import would create circular dependency, use constant directly
    throw new Error('Invalid hex string format');
  }
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substring(i, i + 2), 16);
  }
  return bytes;
};

/**
 * Parse tenant ID from string (proto) to number (UI types)
 *
 * @param tenantId - Tenant ID as string from gRPC response
 * @throws Error with VAL_INVALID_TENANT_ID message if parsing fails
 * @returns Numeric tenant ID
 */
export const parseTenantId = (tenantId: string): number => {
  if (!tenantId) {
    throw new Error('Invalid tenant ID format');
  }
  const parsed = parseInt(tenantId, 10);
  if (isNaN(parsed)) {
    throw new Error('Invalid tenant ID format');
  }
  return parsed;
};

// ============================================================================
// Date Input Parsing (EU Format)
// ============================================================================

/**
 * Parse EU date input (DD/MM/YYYY) to ISO date string (YYYY-MM-DD)
 * Companion to formatDate() for controlled text inputs.
 * Returns undefined if format is invalid or incomplete.
 *
 * @param text - User-typed date string in DD/MM/YYYY format
 * @returns ISO date string (YYYY-MM-DD) or undefined if invalid
 */
export const parseDateInputEU = (text: string): string | undefined => {
  const match = text.match(/^(\d{2})\/(\d{2})\/(\d{4})$/);
  if (!match) return undefined;
  const [, day, month, year] = match;
  const date = new Date(`${year}-${month}-${day}`);
  if (isNaN(date.getTime())) return undefined;
  return `${year}-${month}-${day}`;
};

// ============================================================================
// Text Truncation
// ============================================================================

/**
 * Truncate string with ellipsis if exceeds maxLength
 *
 * @param str - String to truncate
 * @param maxLength - Maximum length before truncation
 * @param ellipsis - Ellipsis string (default '...')
 * @returns Truncated string with ellipsis, or original if within limit
 */
export const truncateWithEllipsis = (str: string, maxLength: number, ellipsis = '...'): string => {
  if (str.length <= maxLength) return str;
  return str.substring(0, maxLength) + ellipsis;
};

// ============================================================================
// Certificate Expiry Calculation
// ============================================================================

/**
 * Calculate days until certificate expiration
 *
 * @param expiryDate - ISO date string of certificate expiry
 * @returns Days until expiry (negative if expired), or null if no date
 */
export const calculateDaysUntilExpiry = (expiryDate: string | undefined): number | null => {
  if (!expiryDate) return null;
  const expiry = new Date(expiryDate);
  const now = new Date();
  const diffTime = expiry.getTime() - now.getTime();
  return Math.ceil(diffTime / (1000 * 60 * 60 * 24));
};

// ============================================================================
// MIOTY Endpoint Form Validators (Shared by Add/Edit Dialogs)
// ============================================================================

import {
  MIOTY_EUI_REGEX,
  MIOTY_KEY_BYTE_LENGTH,
  MIOTY_KEY_REGEX,
  MIOTY_SHORT_ADDR_MAX,
  MIOTY_SHORT_ADDR_REGEX,
  MIOTY_UINT32_MAX,
} from '@constants/app';
import { ENDPOINT_FORM } from '@constants/messages';

/**
 * Generate a cryptographically random hex key.
 * @param byteLength - Number of random bytes (e.g. 16 for a 128-bit key)
 */
export const generateRandomKey = (byteLength: number = MIOTY_KEY_BYTE_LENGTH): string => {
  const bytes = new Uint8Array(byteLength);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
};

/**
 * Validate an EUI hex string (16 hex chars, with or without dashes).
 * Returns error message or null if valid.
 */
export const validateEui = (value: string): string | null => {
  const cleaned = value.replace(/-/g, '');
  if (!value) return ENDPOINT_FORM.ERROR_EUI_REQUIRED;
  if (!MIOTY_EUI_REGEX.test(cleaned)) return ENDPOINT_FORM.ERROR_EUI_FORMAT;
  return null;
};

/**
 * Validate a hex key string of expected hex length.
 * Returns error message or null if valid.
 */
export const validateHexKey = (value: string, isNetwork: boolean): string | null => {
  if (!MIOTY_KEY_REGEX.test(value)) {
    return isNetwork ? ENDPOINT_FORM.ERROR_NETWORK_KEY_FORMAT : ENDPOINT_FORM.ERROR_APP_KEY_FORMAT;
  }
  return null;
};

/**
 * Validate a MIOTY short address (1-4 hex chars, nonzero, max 0xFFFF).
 * Returns error message or null if valid.
 */
export const validateShortAddr = (value: string): string | null => {
  if (value === '') return ENDPOINT_FORM.ERROR_SHORT_ADDR_REQUIRED;
  if (!MIOTY_SHORT_ADDR_REGEX.test(value)) return ENDPOINT_FORM.ERROR_SHORT_ADDR_FORMAT;
  const num = parseInt(value, 16);
  if (num === 0) return ENDPOINT_FORM.ERROR_SHORT_ADDR_ZERO;
  if (num > MIOTY_SHORT_ADDR_MAX) return ENDPOINT_FORM.ERROR_SHORT_ADDR_RANGE;
  return null;
};

/**
 * Validate a uint32 counter value (0 – 4294967295).
 * Returns error message or null if valid.
 */
export const validateUint32Counter = (
  value: string,
  fieldName: 'lastPacketCnt' | 'attachCnt'
): string | null => {
  const num = parseInt(value, 10);
  if (isNaN(num) || num < 0 || num > MIOTY_UINT32_MAX) {
    return fieldName === 'lastPacketCnt'
      ? ENDPOINT_FORM.ERROR_LAST_PACKET_CNT_RANGE
      : ENDPOINT_FORM.ERROR_ATTACH_CNT_RANGE;
  }
  return null;
};

/**
 * Validate a Type EUI (optional 16 hex chars).
 * Returns error message or null if valid/empty.
 */
export const validateTypeEui = (value: string): string | null => {
  if (!value) return null;
  if (!MIOTY_EUI_REGEX.test(value)) return ENDPOINT_FORM.ERROR_TYPE_EUI_FORMAT;
  return null;
};

// ============================================================================
// Altitude Formatting
// ============================================================================

/** Format altitude value with unit suffix */
export const formatAltitude = (meters: number): string => {
  return `${meters.toFixed(1)} m`;
};

// ============================================================================
// Downlink Formatters
// ============================================================================

import type { ChipProps } from '@mui/material';

import { DOWNLINK_PRIORITY_PRESETS, DOWNLINK_QUEUE_STATUS, DOWNLINK_RESULT } from '@constants/app';
import { DOWNLINK_RESULT_LABELS, DOWNLINK_STATUS_LABELS } from '@constants/messages';

/**
 * Map downlink priority value to display label.
 * Derives labels from DOWNLINK_PRIORITY_PRESETS; falls back to numeric display.
 */
export const formatDownlinkPriority = (priority: number): string => {
  const preset = DOWNLINK_PRIORITY_PRESETS.find((p) => p.value === priority);
  return preset ? preset.label : priority.toFixed(2);
};

/**
 * Map downlink queue status to display label and MUI Chip color.
 * Labels are sourced from DOWNLINK_STATUS_LABELS; colors map to MUI theme palette.
 */
export const formatDownlinkQueueStatus = (
  status: string
): { label: string; color: ChipProps['color'] } => {
  const label = DOWNLINK_STATUS_LABELS[status] || status || 'Unknown';
  switch (status) {
    case DOWNLINK_QUEUE_STATUS.PENDING:
      return { label, color: 'default' };
    case DOWNLINK_QUEUE_STATUS.SCHEDULED:
    case DOWNLINK_QUEUE_STATUS.RESERVED:
    case DOWNLINK_QUEUE_STATUS.QUEUED:
      return { label, color: 'info' };
    case DOWNLINK_QUEUE_STATUS.TRANSMITTED:
    case DOWNLINK_QUEUE_STATUS.DELIVERED:
    case DOWNLINK_QUEUE_STATUS.ACKED:
      return { label, color: 'success' };
    case DOWNLINK_QUEUE_STATUS.FAILED:
      return { label, color: 'error' };
    case DOWNLINK_QUEUE_STATUS.EXPIRED:
      return { label, color: 'warning' };
    case DOWNLINK_QUEUE_STATUS.REVOKED:
      return { label, color: 'default' };
    default:
      return { label, color: 'default' };
  }
};

/**
 * Map downlink transmission result to display label and MUI Chip color.
 * Labels are sourced from DOWNLINK_RESULT_LABELS; colors map to MUI theme palette.
 */
export const formatDownlinkResult = (
  result: string
): { label: string; color: ChipProps['color'] } => {
  const label = DOWNLINK_RESULT_LABELS[result] || result || 'Unknown';
  switch (result) {
    case DOWNLINK_RESULT.SENT:
      return { label, color: 'success' };
    case DOWNLINK_RESULT.EXPIRED:
      return { label, color: 'warning' };
    case DOWNLINK_RESULT.INVALID:
      return { label, color: 'error' };
    default:
      return { label, color: 'default' };
  }
};
