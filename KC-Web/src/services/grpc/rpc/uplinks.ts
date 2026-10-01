/**
 * Decoders of a stored ulData (SCACI §3.8.1) shared by every RPC that
 * returns uplinks: the Traffic listings and the activity feeds.
 */

import type { BaseStationReceptionAPI, UplinkAPI } from "@api-types/api";

import type * as corePb from "@services/grpc/core_pb";
import { decodeExactJsonObject } from "@utils/exact-json";
import { bytesToHex } from "@utils/formatters";
import { DECODE_STATUS } from "@constants/app";

import { optionalInt64String } from "../codec";

/** The getters a BaseStationMessage and an endpoint Message share. */
interface UplinkProto {
  getId(): string;
  getOpId(): string;
  getPacketCounter(): number;
  getSnr(): number;
  getRssi(): number;
  getDlOpen(): boolean;
  getResExp(): boolean;
  getDlAck(): boolean;
  getDuplicate(): boolean;
  getPayload_asU8(): Uint8Array;
  hasFormat(): boolean;
  getFormat(): number;
  getBaseStationsList(): corePb.BaseStationReceptionInfo[];
  getDecodeStatus(): string;
  getDecodeErrorCode(): string;
  getDecodedPayload_asU8(): Uint8Array;
}

// The blueprint output of a successful decode; any other status carries none to show.
function decodedPayloadOf(m: UplinkProto): Record<string, unknown> | undefined {
  if (m.getDecodeStatus() !== DECODE_STATUS.SUCCESS) return undefined;
  // One unreadable stored blob must not fail the whole listing.
  return decodeExactJsonObject(m.getDecodedPayload_asU8());
}

function mapUplink(
  m: UplinkProto,
  euis: { epEui: string; bsEui: string },
): UplinkAPI {
  return {
    id: m.getId(),
    opId: optionalInt64String(m.getOpId()),
    ...euis,
    packetCnt: m.getPacketCounter(),
    snr: m.getSnr(),
    rssi: m.getRssi(),
    dlOpen: m.getDlOpen(),
    responseExp: m.getResExp(),
    dlAck: m.getDlAck(),
    duplicate: m.getDuplicate(),
    userData: bytesToHex(m.getPayload_asU8()),
    format: m.hasFormat() ? m.getFormat() : undefined,
    receptions: mapBaseStationReceptions(m.getBaseStationsList()),
    decodeStatus: m.getDecodeStatus(),
    decodeErrorCode: m.getDecodeErrorCode() || undefined,
    decodedPayload: decodedPayloadOf(m),
  };
}

/** A stored ulData as ListBaseStationMessages and the station activity feed carry it. */
export const mapStationUplinkMessage = (
  m: corePb.BaseStationMessage,
): UplinkAPI => mapUplink(m, { epEui: m.getEpEui(), bsEui: m.getBsEui() });

/** A stored ulData as ListMessages and the endpoint activity feed carry it. */
export const mapUplinkMessage = (m: corePb.Message): UplinkAPI =>
  mapUplink(m, { epEui: m.getEpeui(), bsEui: m.getBseui() });

/** The receptions of an uplink, one per base station that heard it (SCACI §3.8.1 baseStations). */
function mapBaseStationReceptions(
  bsList: corePb.BaseStationReceptionInfo[],
): BaseStationReceptionAPI[] {
  return bsList.map((bs) => {
    const sp = bs.getSubpackets();
    return {
      bsEui: bs.getBsEui(),
      rxTime: bs.getRxTime(),
      snr: bs.getSnr(),
      rssi: bs.getRssi(),
      eqSnr: bs.getEqSnr()?.getValue(),
      rxDuration: bs.getRxDuration()?.getValue(),
      profile: bs.getProfile()?.getValue(),
      mode: bs.getMode()?.getValue(),
      dlRxSnr: bs.getDlRxSnr()?.getValue(),
      dlRxRssi: bs.getDlRxRssi()?.getValue(),
      subpackets: sp
        ? {
            snr: sp.getSnrList(),
            rssi: sp.getRssiList(),
            frequency: sp.getFrequencyList(),
            phase: sp.getPhaseList().length > 0 ? sp.getPhaseList() : undefined,
          }
        : undefined,
    };
  });
}
