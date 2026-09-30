/**
 * Revocability of downlink queue entries (SCACI §3.11 dlDataRev).
 */

import type {
  DownlinkQueueFilter,
  SCACIDownlinkQueueDTO,
} from "@api-types/api";

import { REVOCABLE_QUEUE_STATUSES } from "@constants/app";

export const isRevocableDownlink = (downlink: SCACIDownlinkQueueDTO) =>
  !!downlink.status && REVOCABLE_QUEUE_STATUSES.has(downlink.status);

/** One payload of a downlink, with the packet counter it is valid for when counter-dependent. */
export interface DownlinkPayloadLine {
  payload: string;
  packetCnt?: number;
}

/**
 * The payloads of a downlink: every payload of a counter-dependent downlink
 * paired with its packet counter (BSSCI §3.12.1), else the one payload.
 */
export const downlinkPayloadLines = (
  downlink: Pick<SCACIDownlinkQueueDTO, "payloads" | "packetCnt" | "cntDepend">,
): DownlinkPayloadLine[] =>
  (downlink.payloads.length > 0 ? downlink.payloads : [""]).map(
    (payload, i) => ({
      payload,
      packetCnt: downlink.cntDepend ? downlink.packetCnt?.[i] : undefined,
    }),
  );

// Unfiltered, the queue lists only the in-flight states, which are the revocable ones.
export const filterMatchesRevocable = (filter: DownlinkQueueFilter) =>
  !filter.status || REVOCABLE_QUEUE_STATUSES.has(filter.status);
