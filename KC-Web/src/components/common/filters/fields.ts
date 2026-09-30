/**
 * Filter fields shared by the Traffic and Logs listings. Each is typed on the
 * smallest filter shape it edits, so any listing filter that includes that
 * shape can use it.
 */

import type { DeviceScope } from "@api-types/api";

import { isValidEui } from "@utils/eui";
import { type TimeRange } from "@constants/app";
import {
  FILTER_LABELS,
  SCACI_FIELD,
  TIME_RANGE_LABELS,
} from "@constants/messages";

import type { SelectFilterField, TextFilterField } from "./types";

const SIGNED_INTEGER = /^-?\d+$/;
const POSITIVE_INTEGER = /^[1-9]\d*$/;

const validateEui = (value: string) =>
  isValidEui(value) ? null : FILTER_LABELS.ERR_EUI;

export const validateNumber = (value: string) =>
  Number.isFinite(Number(value)) ? null : FILTER_LABELS.ERR_NUMBER;

export function timeRangeField(
  ranges: readonly TimeRange[],
): SelectFilterField<{ timeRange: TimeRange }, TimeRange> {
  return {
    id: "timeRange",
    kind: "select",
    label: FILTER_LABELS.TIME_RANGE,
    options: ranges.map((range) => ({
      value: range,
      label: TIME_RANGE_LABELS[range],
    })),
    get: (filter) => filter.timeRange,
    set: (timeRange) => ({ timeRange: timeRange ?? ranges[0] }),
  };
}

export const EP_EUI_FIELD: TextFilterField<DeviceScope> = {
  id: "epEui",
  kind: "text",
  label: SCACI_FIELD.EP_EUI,
  validate: validateEui,
  get: (filter) => filter.epEui,
  set: (epEui) => ({ epEui }),
};

export const BS_EUI_FIELD: TextFilterField<DeviceScope> = {
  id: "bsEui",
  kind: "text",
  label: SCACI_FIELD.BS_EUI,
  validate: validateEui,
  get: (filter) => filter.bsEui,
  set: (bsEui) => ({ bsEui }),
};

export const OP_ID_FIELD: TextFilterField<{ opId?: string }> = {
  id: "opId",
  kind: "text",
  label: SCACI_FIELD.OP_ID,
  validate: (value) =>
    SIGNED_INTEGER.test(value) ? null : FILTER_LABELS.ERR_INTEGER,
  get: (filter) => filter.opId,
  set: (opId) => ({ opId }),
};

export const QUE_ID_FIELD: TextFilterField<{ queId?: string }> = {
  id: "queId",
  kind: "text",
  label: SCACI_FIELD.QUE_ID,
  validate: (value) =>
    POSITIVE_INTEGER.test(value) ? null : FILTER_LABELS.ERR_POSITIVE_INTEGER,
  get: (filter) => filter.queId,
  set: (queId) => ({ queId }),
};
