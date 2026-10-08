/**
 * An uplink as one line of a device's Activity table (SCACI §3.8.1 ulData):
 * its packet counter as the title; the radio metrics, the downlink flags it
 * set and its payload as the caption; the blueprint decode of the payload as
 * a second caption; and the device on the other end of the radio link as its
 * scope.
 */

import type { UplinkUI } from "@api-types/api";

import { formatEui } from "@utils/eui";
import { formatMeasurement, formatUserData } from "@utils/formatters";
import {
  DECODE_STATUS,
  UPLINK_CAPTION_PAYLOAD_HEX_CHARS,
} from "@constants/app";
import { ACTIVITY_TABLE, SCACI_FIELD } from "@constants/messages";

/** What an uplink row names in the Scope column. */
export type UplinkScope = (uplink: UplinkUI) => string;

export const formatUplinkTitle = (uplink: UplinkUI): string =>
  `${ACTIVITY_TABLE.UPLINK_TITLE_PREFIX}${uplink.packetCnt}`;

const metric = (label: string, value: number | undefined) =>
  `${label}${ACTIVITY_TABLE.METRIC_SEPARATOR}${formatMeasurement(value)}`;

const payloadPreview = (userData: string) =>
  userData.length > UPLINK_CAPTION_PAYLOAD_HEX_CHARS
    ? `${formatUserData(userData.slice(0, UPLINK_CAPTION_PAYLOAD_HEX_CHARS))}${ACTIVITY_TABLE.PAYLOAD_TRUNCATED}`
    : formatUserData(userData);

export const formatUplinkCaption = (uplink: UplinkUI): string =>
  [
    metric(SCACI_FIELD.RSSI, uplink.rssi),
    metric(SCACI_FIELD.SNR, uplink.snr),
    metric(SCACI_FIELD.EQ_SNR, uplink.eqSnr),
    ...(uplink.dlOpen ? [SCACI_FIELD.DL_OPEN] : []),
    ...(uplink.dlAck ? [SCACI_FIELD.DL_ACK] : []),
    payloadPreview(uplink.userData),
  ].join(ACTIVITY_TABLE.CAPTION_SEPARATOR);

const decodedValue = (value: unknown): string =>
  typeof value === "object" ? JSON.stringify(value) : String(value);

/** The blueprint-decoded values as "key value · key value", when there are any. */
export const formatDecodedValues = ({
  decodedPayload,
}: UplinkUI): string | undefined => {
  const entries = Object.entries(decodedPayload ?? {});
  if (entries.length === 0) return undefined;
  return entries
    .map(
      ([key, value]) =>
        `${key}${ACTIVITY_TABLE.METRIC_SEPARATOR}${decodedValue(value)}`,
    )
    .join(ACTIVITY_TABLE.CAPTION_SEPARATOR);
};

const decodedLine = (uplink: UplinkUI): string | undefined => {
  const fields = formatDecodedValues(uplink);
  return fields && `${ACTIVITY_TABLE.DECODED_PREFIX}${fields}`;
};

/** A decode outcome followed by the blueprint error token, when there is one. */
export const withDecodeErrorCode = (text: string, code?: string): string =>
  code
    ? `${text}${ACTIVITY_TABLE.DECODE_CODE_OPEN}${code}${ACTIVITY_TABLE.DECODE_CODE_CLOSE}`
    : text;

const failedLine = ({ decodeErrorCode }: UplinkUI): string =>
  withDecodeErrorCode(ACTIVITY_TABLE.DECODE_FAILED, decodeErrorCode);

const skippedLine = ({ userData }: UplinkUI): string | undefined =>
  userData ? ACTIVITY_TABLE.NOT_DECODED_NO_BLUEPRINT : undefined;

// A pending decode has nothing to report yet.
const DECODE_LINES: Record<string, (uplink: UplinkUI) => string | undefined> = {
  [DECODE_STATUS.SUCCESS]: decodedLine,
  [DECODE_STATUS.FAILED]: failedLine,
  [DECODE_STATUS.SKIPPED]: skippedLine,
};

/** The blueprint decode of the payload as a caption line, when there is one to show. */
export const formatUplinkDecode = (uplink: UplinkUI): string | undefined =>
  DECODE_LINES[uplink.decodeStatus]?.(uplink);

/** On a base station's page: the endpoint that sent the uplink. */
export const uplinkEndpointScope: UplinkScope = (uplink) =>
  formatEui(uplink.epEui);

/** On an endpoint's page: every base station that received the uplink. */
export const uplinkStationsScope: UplinkScope = (uplink) => {
  const stations = uplink.receptions.map((reception) => reception.bsEui);
  return (stations.length > 0 ? stations : [uplink.bsEui])
    .map(formatEui)
    .join(ACTIVITY_TABLE.STATION_SEPARATOR);
};
