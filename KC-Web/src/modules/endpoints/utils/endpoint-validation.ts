/**
 * The one validator of the End Point form, shared by the Add and Edit dialogs
 * and by every field as it is edited (BSSCI §3.8.1, SCACI §3.6.1).
 */

import { validateEui, validateTypeEui } from "@utils/eui";
import {
  validateCarrierOffset,
  validateHexKey,
  validateShortAddr,
  validateUint32Counter,
} from "@utils/formatters";
import { ENDPOINT_FORM } from "@constants/messages";

import type { EndpointFormValues } from "./endpoint-form";

export type EndpointField = keyof EndpointFormValues;

/** Fields a registration needs, marked as required in both dialogs. */
export const ENDPOINT_REQUIRED_FIELDS: ReadonlySet<EndpointField> =
  new Set<EndpointField>([
    "epEui",
    "name",
    "shortAddr",
    "lastPacketCnt",
    "attachCnt",
    "networkKey",
  ]);

const REQUIRED_MESSAGES: Partial<Record<EndpointField, string>> = {
  name: ENDPOINT_FORM.ERROR_NAME_REQUIRED,
  networkKey: ENDPOINT_FORM.ERROR_NETWORK_KEY_REQUIRED,
  lastPacketCnt: ENDPOINT_FORM.ERROR_LAST_PACKET_CNT_REQUIRED,
  attachCnt: ENDPOINT_FORM.ERROR_ATTACH_CNT_REQUIRED,
};

/** Format checks of a filled field; a field without one accepts any value. */
const FORMAT_CHECKS: Partial<
  Record<EndpointField, (value: string) => string | null>
> = {
  epEui: validateEui,
  shortAddr: validateShortAddr,
  networkKey: (value) => validateHexKey(value, true),
  applicationKey: (value) => validateHexKey(value, false),
  lastPacketCnt: (value) => validateUint32Counter(value, "lastPacketCnt"),
  attachCnt: (value) => validateUint32Counter(value, "attachCnt"),
  typeEui: validateTypeEui,
  carrierOffset: validateCarrierOffset,
};

const NO_STORED_KEYS: ReadonlySet<EndpointField> = new Set();

/**
 * The error of one field of the form, empty when the field is valid. A key
 * field left empty keeps the stored key, so it is not missing.
 */
export function validateEndpointField(
  field: EndpointField,
  values: EndpointFormValues,
  storedKeys: ReadonlySet<EndpointField> = NO_STORED_KEYS,
): string {
  const value = values[field];
  if (typeof value !== "string") return "";
  if (value === "") {
    return ENDPOINT_REQUIRED_FIELDS.has(field) && !storedKeys.has(field)
      ? (REQUIRED_MESSAGES[field] ?? FORMAT_CHECKS[field]?.(value) ?? "")
      : "";
  }
  return FORMAT_CHECKS[field]?.(value) ?? "";
}

/** The errors of the whole form, keyed by field; empty when it is valid. */
export function validateEndpointForm(
  values: EndpointFormValues,
  storedKeys: ReadonlySet<EndpointField> = NO_STORED_KEYS,
): Record<string, string> {
  const errors: Record<string, string> = {};
  for (const field of Object.keys(values) as EndpointField[]) {
    const error = validateEndpointField(field, values, storedKeys);
    if (error) errors[field] = error;
  }
  return errors;
}
