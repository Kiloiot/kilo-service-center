/**
 * Common Mappers
 *
 * Shared transformation utilities for MIOTY data types.
 * Used by domain-specific mappers.
 */

/**
 * Parse ISO date string safely
 * Returns null for invalid/empty dates
 */
export function parseISODate(dateStr: string | null | undefined): Date | null {
  if (!dateStr) return null;
  const date = new Date(dateStr);
  return isNaN(date.getTime()) ? null : date;
}

/**
 * Normalize status string to known values with fallback
 */
export function normalizeStatus<T extends string>(
  value: string | undefined,
  validValues: readonly T[],
  fallback: T
): T {
  if (!value) return fallback;
  const normalized = value.toLowerCase() as T;
  return validValues.includes(normalized) ? normalized : fallback;
}

/**
 * Format EUI (Extended Unique Identifier) for display
 * Handles both string and number formats
 */
export function formatEUI(eui: string | number | undefined): string {
  if (eui === undefined || eui === null) return '';
  if (typeof eui === 'number') {
    // Convert to hex string, pad to 16 chars (8 bytes)
    return eui.toString(16).padStart(16, '0').toUpperCase();
  }
  // Already a string - ensure uppercase
  return eui.toUpperCase();
}

/**
 * Convert base64 encoded data to hex string for display
 * Returns '0x' prefix format
 */
export function base64ToHex(base64: string | undefined): string {
  if (!base64) return '0x';
  try {
    const bytes = atob(base64);
    const hex = Array.from(bytes, (c) => c.charCodeAt(0).toString(16).padStart(2, '0')).join('');
    return `0x${hex}`;
  } catch {
    return '0x';
  }
}

/**
 * Convert Unix nanoseconds to Date
 */
export function unixNsToDate(ns: number | undefined): Date | null {
  if (ns === undefined || ns === null) return null;
  return new Date(ns / 1000000); // Convert ns to ms
}

/**
 * Format Unix nanoseconds as locale string
 */
export function formatUnixNs(ns: number | undefined): string | undefined {
  if (ns === undefined || ns === null) return undefined;
  const date = new Date(ns / 1000000);
  return date.toLocaleString();
}
