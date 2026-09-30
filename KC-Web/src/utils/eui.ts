/**
 * EUI helpers: canonical form, display form and form validation.
 */

import {
  ENCODING,
  EUI_DISPLAY_SEPARATOR,
  EUI_PLACEHOLDER,
  MIOTY_EUI_HEX_LENGTH,
  MIOTY_EUI_REGEX,
} from "@constants/app";
import { ENDPOINT_FORM } from "@constants/messages";

/**
 * Format an 8-byte EUI for display: uppercase byte pairs joined by the
 * display separator. Accepts any separator and case; anything that is not
 * 16 hex digits is returned untouched and an empty value renders the placeholder.
 */
export const formatEui = (eui: string | undefined | null): string => {
  if (!eui) return EUI_PLACEHOLDER;
  const cleaned = eui.replace(/[^0-9A-Fa-f]/g, "");
  if (cleaned.length !== MIOTY_EUI_HEX_LENGTH) return eui;
  const parts: string[] = [];
  for (let i = 0; i < cleaned.length; i += ENCODING.HEX_DIGITS_PER_BYTE) {
    parts.push(cleaned.substring(i, i + ENCODING.HEX_DIGITS_PER_BYTE));
  }
  return parts.join(EUI_DISPLAY_SEPARATOR).toUpperCase();
};

/** Shows a complete EUI typed in any form in the display form; an emptied field stays empty. */
export const formatEuiInput = (value: string): string =>
  value ? formatEui(value) : value;

/** Canonical EUI form for query keys and comparisons: no separators, lowercase. */
export const normalizeEui = (eui: string): string =>
  eui.replace(/[^0-9A-Fa-f]/g, "").toLowerCase();

/** True when the value is a 16-hex-digit EUI in any separator/case form. */
export const isValidEui = (value: string): boolean =>
  MIOTY_EUI_REGEX.test(normalizeEui(value));

/**
 * Validate an EUI hex string (16 hex chars, with or without dashes).
 * Returns the catalog error message or null if valid.
 */
export const validateEui = (
  value: string,
  formatError: string = ENDPOINT_FORM.ERROR_EUI_FORMAT,
): string | null => {
  if (!value) return ENDPOINT_FORM.ERROR_EUI_REQUIRED;
  if (!isValidEui(value)) return formatError;
  return null;
};

/**
 * Validate an optional type EUI (blueprint identifier): empty is valid,
 * anything else must be 16 hex characters.
 */
export const validateTypeEui = (value: string | undefined): string | null =>
  value ? validateEui(value, ENDPOINT_FORM.ERROR_TYPE_EUI_FORMAT) : null;
