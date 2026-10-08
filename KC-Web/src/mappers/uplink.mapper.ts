/**
 * Uplink Mappers
 *
 * Turns a stored ulData into a table row. A message heard by several base
 * stations carries one reception per station (SCACI §3.8.1 baseStations); the
 * row reports the reception of the station in view, else the first one.
 */

import type {
  BaseStationReceptionAPI,
  UplinkAPI,
  UplinkUI,
} from "@api-types/api";

import { normalizeEui } from "@utils/eui";

function receptionInView(
  receptions: BaseStationReceptionAPI[],
  bsEui: string | undefined,
): BaseStationReceptionAPI | undefined {
  if (!bsEui) return receptions[0];
  const wanted = normalizeEui(bsEui);
  return receptions.find((r) => normalizeEui(r.bsEui) === wanted);
}

type RadioView = Pick<UplinkUI, "bsEui" | "snr" | "rssi">;

function uplinkRow(
  api: UplinkAPI,
  reception: BaseStationReceptionAPI | undefined,
  radio: RadioView,
): UplinkUI {
  return {
    id: api.id,
    opId: api.opId,
    epEui: api.epEui,
    ...radio,
    rxTime: reception?.rxTime,
    packetCnt: api.packetCnt,
    eqSnr: reception?.eqSnr,
    dlOpen: api.dlOpen,
    responseExp: api.responseExp,
    dlAck: api.dlAck,
    duplicate: api.duplicate,
    userData: api.userData,
    format: api.format,
    receptions: api.receptions,
    decodeStatus: api.decodeStatus,
    decodeErrorCode: api.decodeErrorCode,
    decodedPayload: api.decodedPayload,
  };
}

/** A tenant-wide listing row: the radio values of the reception in view. */
export function mapUplink(api: UplinkAPI, bsEui?: string): UplinkUI {
  const reception = receptionInView(api.receptions, bsEui);
  return uplinkRow(api, reception, {
    bsEui: reception?.bsEui ?? api.bsEui,
    snr: reception?.snr ?? api.snr,
    rssi: reception?.rssi ?? api.rssi,
  });
}

/**
 * A station listing row: the service center already reports that station's
 * bsEui, snr and rssi; the message carries no rxTime and cannot tell an
 * absent eqSnr from zero, so both come from the station's reception.
 */
export function mapStationUplink(api: UplinkAPI): UplinkUI {
  const reception = receptionInView(api.receptions, api.bsEui);
  return uplinkRow(api, reception, {
    bsEui: api.bsEui,
    snr: api.snr,
    rssi: api.rssi,
  });
}
