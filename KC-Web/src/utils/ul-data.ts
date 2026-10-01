/**
 * Views of a stored uplink in SCACI §3.8.1 terms: the ulData message as an
 * Application Center receives it, and one row per reported subpacket.
 */

import type { BaseStationReceptionAPI, UplinkUI } from "@api-types/api";

import { hexToBytes } from "@utils/formatters";
import { MIOTY_COMMAND } from "@constants/app";

export interface SubpacketRow {
  index: number;
  snr: number;
  rssi: number;
  frequency: number;
  phase?: number;
}

/** The ulData message of an uplink, field for field (userData as its byte values). */
export function toUlDataMessage(uplink: UplinkUI) {
  return {
    command: MIOTY_COMMAND.UL_DATA,
    opId: uplink.opId,
    epEui: uplink.epEui,
    baseStations: uplink.receptions,
    packetCnt: uplink.packetCnt,
    userData: uplink.userData ? Array.from(hexToBytes(uplink.userData)) : [],
    format: uplink.format,
    dlOpen: uplink.dlOpen,
    responseExp: uplink.responseExp,
    dlAck: uplink.dlAck,
    duplicate: uplink.duplicate,
  };
}

/** The per-subpacket arrays of a reception, turned into rows. */
export function subpacketRows(
  reception: BaseStationReceptionAPI,
): SubpacketRow[] {
  const subpackets = reception.subpackets;
  if (!subpackets) return [];
  return subpackets.snr.map((snr, index) => ({
    index,
    snr,
    rssi: subpackets.rssi[index],
    frequency: subpackets.frequency[index],
    phase: subpackets.phase?.[index],
  }));
}
