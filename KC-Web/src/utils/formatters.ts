import type { PasswordPolicyAPI } from "@api-types/api";

import {
  ENCODING,
  FRACTION_DIGITS,
  MEASUREMENT_FRACTION_DIGITS,
  MS_PER_DAY,
  TRUNCATION,
} from "@constants/app";
import {
  DATA_TABLE,
  PASSWORD_RULES,
  VAL_INVALID_HEX_STRING,
} from "@constants/messages";

// ============================================================================
// Traffic Value Formatting
// ============================================================================

/** userData bytes (hex) as operators read them: 0x-prefixed, or the empty-payload label. */
export const formatUserData = (hex: string): string =>
  hex
    ? `${VALUE_FORMAT.HEX_PREFIX}${hex.toUpperCase()}`
    : DATA_TABLE.EMPTY_PAYLOAD;

/** An optional value; an absent or empty value renders the placeholder. */
export const formatOptional = (
  value?: string | number | null,
): string | number =>
  value === undefined || value === null || value === ""
    ? DATA_TABLE.NO_VALUE
    : value;

/** A radio measurement (snr, rssi, eqSnr) at display precision; absent values render the placeholder. */
export const formatMeasurement = (value: number | undefined): string =>
  value === undefined
    ? DATA_TABLE.NO_VALUE
    : value.toFixed(MEASUREMENT_FRACTION_DIGITS);

/** A MIOTY short address as its four upper-case hex digits (BSSCI §3.8.1). */
export const formatShortAddress = (shAddr: number): string =>
  shAddr
    .toString(ENCODING.HEX_RADIX)
    .toUpperCase()
    .padStart(MIOTY_SHORT_ADDR_HEX_DIGITS, "0");

// ============================================================================
// Dashboard Card Trend Formatting
// ============================================================================

/**
 * Format dashboard status card trend text as "+{count} {suffix}".
 *
 * @param count - Items added in period, clamped to >= 0
 * @param suffixLabel - Centralized suffix (e.g., DASHBOARD_PAGE.ADDED_LAST_WEEK)
 * @returns Formatted trend string for dashboard status cards
 */
export const formatDashboardCountTrend = (
  count: number,
  suffixLabel: string,
): string => {
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
 * Format a decoded payload as one compact line
 * Converts JSON object to compact key=value format
 *
 * @param payload - Decoded payload object from blueprint decoding
 * @returns key=value pairs (e.g., "temp=23.45 humidity=65.2")
 */
export const formatDecodedPayload = (
  payload: Record<string, unknown>,
): string => {
  if (!payload || typeof payload !== "object") return "";

  return Object.entries(payload)
    .map(([key, value]) => {
      if (typeof value === "number") {
        // Format numbers with appropriate precision
        return `${key}=${Number.isInteger(value) ? value : value.toFixed(FRACTION_DIGITS.DECODED_VALUE)}`;
      }
      return `${key}=${value}`;
    })
    .join(" ");
};

/**
 * Validate hex string format
 *
 * @param value - String to validate
 * @returns true if valid hex with even length, false otherwise
 */
export const isValidHexString = (value: string): boolean => {
  if (!value) return false;
  const hexRegex = /^[0-9A-Fa-f]*$/;
  return (
    hexRegex.test(value) && value.length % ENCODING.HEX_DIGITS_PER_BYTE === 0
  );
};

export const bytesToHex = (bytes: Uint8Array | string): string => {
  if (typeof bytes === "string") return bytes;
  return Array.from(bytes, (b) =>
    b.toString(ENCODING.HEX_RADIX).padStart(ENCODING.HEX_DIGITS_PER_BYTE, "0"),
  ).join("");
};

/**
 * Convert hex string to Uint8Array for gRPC bytes fields
 *
 * @param hex - Hex string to convert (must have even length)
 * @throws Error with VAL_INVALID_HEX_STRING message if invalid hex
 * @returns Uint8Array of bytes
 */
export const hexToBytes = (hex: string): Uint8Array => {
  if (!isValidHexString(hex)) {
    throw new Error(VAL_INVALID_HEX_STRING);
  }
  const step = ENCODING.HEX_DIGITS_PER_BYTE;
  const bytes = new Uint8Array(hex.length / step);
  for (let i = 0; i < hex.length; i += step) {
    bytes[i / step] = parseInt(hex.substring(i, i + step), ENCODING.HEX_RADIX);
  }
  return bytes;
};

// ============================================================================
// Date Input Parsing (EU Format)
// ============================================================================

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
export const truncateWithEllipsis = (
  str: string,
  maxLength: number,
  ellipsis: string = TRUNCATION.ELLIPSIS,
): string => {
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
export const calculateDaysUntilExpiry = (
  expiryDate: string | undefined,
): number | null => {
  if (!expiryDate) return null;
  const expiry = new Date(expiryDate);
  const now = new Date();
  const diffTime = expiry.getTime() - now.getTime();
  return Math.ceil(diffTime / MS_PER_DAY);
};

// ============================================================================
// MIOTY Endpoint Form Validators (Shared by Add/Edit Dialogs)
// ============================================================================

import {
  ENDPOINT_CARRIER_OFFSET_RANGE,
  ENDPOINT_CARRIER_OFFSET_REGEX,
  MIOTY_KEY_BYTE_LENGTH,
  MIOTY_KEY_REGEX,
  MIOTY_SHORT_ADDR_HEX_DIGITS,
  MIOTY_SHORT_ADDR_MAX,
  MIOTY_SHORT_ADDR_REGEX,
  MIOTY_UINT32_MAX,
} from "@constants/app";
import { ENDPOINT_FORM } from "@constants/messages";

/**
 * Generate a cryptographically random hex key.
 * @param byteLength - Number of random bytes (e.g. 16 for a 128-bit key)
 */
export const generateRandomKey = (
  byteLength: number = MIOTY_KEY_BYTE_LENGTH,
): string => {
  const bytes = new Uint8Array(byteLength);
  crypto.getRandomValues(bytes);
  return bytesToHex(bytes);
};

/**
 * Validate a hex key string of expected hex length.
 * Returns error message or null if valid.
 */
export const validateHexKey = (
  value: string,
  isNetwork: boolean,
): string | null => {
  if (!MIOTY_KEY_REGEX.test(value)) {
    return isNetwork
      ? ENDPOINT_FORM.ERROR_NETWORK_KEY_FORMAT
      : ENDPOINT_FORM.ERROR_APP_KEY_FORMAT;
  }
  return null;
};

/**
 * Validate a MIOTY short address (1-4 hex chars, nonzero, max 0xFFFF).
 * Returns error message or null if valid.
 */
export const validateShortAddr = (value: string): string | null => {
  if (value === "") return ENDPOINT_FORM.ERROR_SHORT_ADDR_REQUIRED;
  if (!MIOTY_SHORT_ADDR_REGEX.test(value))
    return ENDPOINT_FORM.ERROR_SHORT_ADDR_FORMAT;
  const num = parseInt(value, 16);
  if (num === 0) return ENDPOINT_FORM.ERROR_SHORT_ADDR_ZERO;
  if (num > MIOTY_SHORT_ADDR_MAX) return ENDPOINT_FORM.ERROR_SHORT_ADDR_RANGE;
  return null;
};

/**
 * Validate an end point carrier offset: an integer (Hz) within the int32
 * range of EndPoint.carrier_offset. Returns error message or null if valid.
 */
export const validateCarrierOffset = (value: string): string | null => {
  const num = Number(value);
  return ENDPOINT_CARRIER_OFFSET_REGEX.test(value) &&
    num >= ENDPOINT_CARRIER_OFFSET_RANGE.MIN &&
    num <= ENDPOINT_CARRIER_OFFSET_RANGE.MAX
    ? null
    : ENDPOINT_FORM.ERROR_CARRIER_OFFSET_FORMAT;
};

/**
 * Validate a uint32 counter value (0 – 4294967295).
 * Returns error message or null if valid.
 */
export const validateUint32Counter = (
  value: string,
  fieldName: "lastPacketCnt" | "attachCnt",
): string | null => {
  const num = parseInt(value, 10);
  if (isNaN(num) || num < 0 || num > MIOTY_UINT32_MAX) {
    return fieldName === "lastPacketCnt"
      ? ENDPOINT_FORM.ERROR_LAST_PACKET_CNT_RANGE
      : ENDPOINT_FORM.ERROR_ATTACH_CNT_RANGE;
  }
  return null;
};

// ============================================================================
// Altitude Formatting
// ============================================================================

/**
 * The password rule a policy states, e.g. "8 to 128 characters, containing a
 * letter and a digit".
 */
export const formatPasswordRule = (policy: PasswordPolicyAPI): string => {
  const length = PASSWORD_RULES.LENGTH.replace(
    PASSWORD_RULES.MIN_PLACEHOLDER,
    String(policy.min_length),
  ).replace(PASSWORD_RULES.MAX_PLACEHOLDER, String(policy.max_length));
  const needs = [
    policy.requires_letter ? PASSWORD_RULES.LETTER : "",
    policy.requires_digit ? PASSWORD_RULES.DIGIT : "",
  ].filter(Boolean);
  return needs.length === 0
    ? length
    : length + PASSWORD_RULES.CONTAINING + needs.join(PASSWORD_RULES.JOIN);
};

/** Format altitude value with unit suffix */
export const formatAltitude = (meters: number): string => {
  return `${meters.toFixed(FRACTION_DIGITS.ALTITUDE)}${VALUE_FORMAT.METERS_SUFFIX}`;
};

// ============================================================================
// Downlink Formatters
// ============================================================================

import type { ChipProps } from "@mui/material";

import {
  DOWNLINK_PRIORITY_PRESETS,
  DOWNLINK_QUEUE_STATUS,
  DOWNLINK_RESULT,
} from "@constants/app";
import {
  DOWNLINK_PRIORITY_LABELS,
  DOWNLINK_RESULT_LABELS,
  DOWNLINK_STATUS_LABELS,
} from "@constants/messages";

/**
 * Map downlink priority value to display label.
 * Derives labels from DOWNLINK_PRIORITY_PRESETS; falls back to numeric display.
 */
export const formatDownlinkPriority = (priority: number): string => {
  const preset = DOWNLINK_PRIORITY_PRESETS.find((p) => p.value === priority);
  return preset
    ? DOWNLINK_PRIORITY_LABELS[preset.preset]
    : priority.toFixed(FRACTION_DIGITS.DOWNLINK_PRIORITY);
};

/**
 * Map downlink queue status to display label and MUI Chip color.
 * Labels are sourced from DOWNLINK_STATUS_LABELS; colors map to MUI theme palette.
 */
export const formatDownlinkQueueStatus = (
  status: string,
): { label: string; color: ChipProps["color"] } => {
  const label =
    DOWNLINK_STATUS_LABELS[status] || status || DATA_TABLE.UNKNOWN_VALUE;
  switch (status) {
    case DOWNLINK_QUEUE_STATUS.PENDING:
      return { label, color: "default" };
    case DOWNLINK_QUEUE_STATUS.SCHEDULED:
    case DOWNLINK_QUEUE_STATUS.RESERVED:
    case DOWNLINK_QUEUE_STATUS.QUEUED:
      return { label, color: "info" };
    case DOWNLINK_QUEUE_STATUS.TRANSMITTED:
    case DOWNLINK_QUEUE_STATUS.DELIVERED:
    case DOWNLINK_QUEUE_STATUS.ACKED:
      return { label, color: "success" };
    case DOWNLINK_QUEUE_STATUS.FAILED:
      return { label, color: "error" };
    case DOWNLINK_QUEUE_STATUS.EXPIRED:
      return { label, color: "warning" };
    case DOWNLINK_QUEUE_STATUS.REVOKED:
      return { label, color: "default" };
    default:
      return { label, color: "default" };
  }
};

/**
 * Map downlink transmission result to display label and MUI Chip color.
 * Labels are sourced from DOWNLINK_RESULT_LABELS; colors map to MUI theme palette.
 */
export const formatDownlinkResult = (
  result: string,
): { label: string; color: ChipProps["color"] } => {
  const label =
    DOWNLINK_RESULT_LABELS[result] || result || DATA_TABLE.UNKNOWN_VALUE;
  switch (result) {
    case DOWNLINK_RESULT.SENT:
      return { label, color: "success" };
    case DOWNLINK_RESULT.EXPIRED:
      return { label, color: "warning" };
    case DOWNLINK_RESULT.INVALID:
      return { label, color: "error" };
    default:
      return { label, color: "default" };
  }
};

// ============================================================================
// Control-plane and certificate formatters
// ============================================================================

import { CERTIFICATE_EXPIRY_STATE, PERCENT_SCALE } from "@constants/app";
import {
  CERTIFICATE_EXPIRY_STATE_LABELS,
  VALUE_FORMAT,
} from "@constants/messages";

type ChipColor = ChipProps["color"];

export const formatYesNo = (value: boolean): string =>
  value ? VALUE_FORMAT.YES : VALUE_FORMAT.NO;

export const formatMilliseconds = (ms?: number): string =>
  ms === undefined
    ? DATA_TABLE.NO_VALUE
    : `${ms}${VALUE_FORMAT.MILLISECONDS_SUFFIX}`;

/** A value already expressed in percent. */
const formatPercent = (
  percent: number,
  decimals: number = FRACTION_DIGITS.PERCENT,
): string => `${percent.toFixed(decimals)}${VALUE_FORMAT.PERCENT_SUFFIX}`;

/** A 0..1 fraction as a percentage. */
export const formatFraction = (
  fraction: number,
  decimals: number = FRACTION_DIGITS.PERCENT,
): string => formatPercent(fraction * PERCENT_SCALE, decimals);

/** SCACI opId range: highest AC-issued (positive) and lowest SC-issued (negative). */
export const formatOpIdRange = (maxAcOpId: string, minScOpId: string): string =>
  `${maxAcOpId}${VALUE_FORMAT.OPID_RANGE_SEPARATOR}${minScOpId}`;

export const formatDaysLeft = (days: number | null): string => {
  if (days === null) return DATA_TABLE.NO_VALUE;
  if (days < 0) return VALUE_FORMAT.DAYS_LEFT_EXPIRED;
  return `${days}${VALUE_FORMAT.DAYS_LEFT_SUFFIX}`;
};

const labelled = (labels: Record<string, string>, value: string) =>
  labels[value] || value;

export const formatCertificateExpiryState = (
  state: string,
): { label: string; color: ChipColor } => {
  const label = labelled(CERTIFICATE_EXPIRY_STATE_LABELS, state);
  switch (state) {
    case CERTIFICATE_EXPIRY_STATE.EXPIRED:
    case CERTIFICATE_EXPIRY_STATE.CRITICAL:
      return { label, color: "error" };
    case CERTIFICATE_EXPIRY_STATE.EXPIRING:
      return { label, color: "warning" };
    case CERTIFICATE_EXPIRY_STATE.VALID:
      return { label, color: "success" };
    default:
      return { label, color: "default" };
  }
};
