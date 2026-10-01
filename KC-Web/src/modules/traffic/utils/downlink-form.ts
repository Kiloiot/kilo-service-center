/**
 * Downlink form rules: field validation, the dlDataQue content a form yields
 * and the form state an existing downlink starts from.
 */

import type { DownlinkContent, SCACIDownlinkQueueDTO } from "@api-types/api";

import { isValidHexString } from "@utils/formatters";
import { DOWNLINK_FORM_DEFAULTS, DOWNLINK_LIMITS } from "@constants/app";
import {
  VAL_FORMAT_RANGE,
  VAL_INVALID_HEX_STRING,
  VAL_PRIORITY_FINITE,
} from "@constants/messages";

export interface PayloadRow {
  payload: string;
  packetCnt: string;
}

export interface DownlinkFormValues {
  payload: string;
  format: string;
  priority: string;
  cntDepend: boolean;
  responseExp: boolean;
  responsePrio: boolean;
  dlWindReq: boolean;
  expOnly: boolean;
  dlRxStatQry: boolean;
}

export interface DownlinkFormInit {
  values: DownlinkFormValues;
  payloadRows: PayloadRow[];
}

export const emptyPayloadRow = (): PayloadRow => ({
  payload: "",
  packetCnt: DOWNLINK_FORM_DEFAULTS.PACKET_CNT,
});

export const NEW_DOWNLINK_FORM: DownlinkFormInit = {
  values: {
    payload: "",
    format: DOWNLINK_FORM_DEFAULTS.FORMAT,
    priority: DOWNLINK_FORM_DEFAULTS.PRIORITY,
    cntDepend: false,
    responseExp: false,
    responsePrio: false,
    dlWindReq: false,
    expOnly: false,
    dlRxStatQry: false,
  },
  payloadRows: [emptyPayloadRow()],
};

/**
 * Strict uint32 validation for packet counter values: rejects floats,
 * negatives and values above DOWNLINK_LIMITS.COUNTER_MAX.
 */
export const isValidPacketCnt = (value: string): boolean => {
  const trimmed = value.trim();
  if (trimmed === "" || !/^\d+$/.test(trimmed)) return false;
  const num = Number(trimmed);
  return (
    Number.isInteger(num) && num >= 0 && num <= DOWNLINK_LIMITS.COUNTER_MAX
  );
};

export interface DownlinkFormState {
  payload: string;
  format: string;
  priority: string;
  cntDepend: boolean;
  payloadRows: PayloadRow[];
}

export interface DownlinkFormValidation {
  payloadError: string;
  formatError: string;
  priorityError: string;
  cntDependRowsValid: boolean;
  formatNum: number;
  priorityNum: number;
}

/**
 * Validates the downlink form. Format must be a uint8, priority any finite
 * number (SCACI §3.10.1), payload a valid hex string when single-shot, and every
 * row a valid hex+counter pair when in counter-dependent mode.
 */
export function validateDownlinkForm(
  state: DownlinkFormState,
): DownlinkFormValidation {
  const formatNum = parseInt(state.format, 10);
  const priorityNum = parseFloat(state.priority);
  const payloadError =
    !state.cntDepend &&
    state.payload.length > 0 &&
    !isValidHexString(state.payload)
      ? VAL_INVALID_HEX_STRING
      : "";
  const formatError =
    isNaN(formatNum) || formatNum < 0 || formatNum > DOWNLINK_LIMITS.FORMAT_MAX
      ? VAL_FORMAT_RANGE
      : "";
  const priorityError = Number.isFinite(priorityNum) ? "" : VAL_PRIORITY_FINITE;
  const cntDependRowsValid = state.cntDepend
    ? state.payloadRows.length >= 1 &&
      state.payloadRows.every(
        (r) =>
          r.payload.length > 0 &&
          isValidHexString(r.payload) &&
          isValidPacketCnt(r.packetCnt),
      )
    : true;
  return {
    payloadError,
    formatError,
    priorityError,
    cntDependRowsValid,
    formatNum,
    priorityNum,
  };
}

export const isDownlinkFormValid = (validation: DownlinkFormValidation) =>
  !validation.formatError &&
  !validation.priorityError &&
  !validation.payloadError &&
  validation.cntDependRowsValid;

/**
 * The dlDataQue content of a form. In counter-dependent mode payloads are
 * per-row and packetCnt is materialized from the row counters; otherwise a
 * single-payload (or empty, ACK-only) downlink.
 */
export function buildDownlinkContent(
  values: DownlinkFormValues,
  payloadRows: PayloadRow[],
  validation: DownlinkFormValidation,
): DownlinkContent {
  const payloads = values.cntDepend
    ? payloadRows.map((r) => r.payload)
    : values.payload.length > 0
      ? [values.payload]
      : [];
  const packetCnt = values.cntDepend
    ? payloadRows.map((r) => parseInt(r.packetCnt, 10))
    : undefined;
  return {
    payloads,
    priority: validation.priorityNum,
    cntDepend: values.cntDepend,
    packetCnt,
    format: validation.formatNum,
    responseExp: values.responseExp,
    responsePrio: values.responsePrio,
    dlWindReq: values.dlWindReq,
    expOnly: values.expOnly,
    dlRxStatQry: values.dlRxStatQry,
  };
}

/** The form state a queued downlink is edited from. */
export function downlinkFormInit(
  downlink: SCACIDownlinkQueueDTO,
): DownlinkFormInit {
  const counters = downlink.packetCnt ?? [];
  return {
    values: {
      payload: downlink.cntDepend ? "" : (downlink.payloads[0] ?? ""),
      format: String(downlink.format),
      priority: String(downlink.priority),
      cntDepend: downlink.cntDepend,
      responseExp: downlink.responseExp,
      responsePrio: downlink.responsePrio,
      dlWindReq: downlink.dlWindReq,
      expOnly: downlink.expOnly,
      dlRxStatQry: downlink.dlRxStatQry ?? false,
    },
    payloadRows: downlink.cntDepend
      ? downlink.payloads.map((payload, i) => ({
          payload,
          packetCnt: String(counters[i]),
        }))
      : [emptyPayloadRow()],
  };
}
