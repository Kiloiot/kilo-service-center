import type { BaseStationUI } from "@api-types/api";

import type { SortState } from "@contexts/filters";
import { certificateExpiry } from "@utils/certificate-expiry";
import { filterAndSortBaseStations } from "@utils/list-query";
import type { BaseStationStatus } from "@constants/app";
import {
  BASE_STATION_STATUS,
  CERTIFICATE_EXPIRY_STATE,
  type CertificateExpiryState,
} from "@constants/app";

export type BaseStationSortField =
  | "name"
  | "eui"
  | "status"
  | "connectionType"
  | "lastSeen";

export type StatusOverlay = BaseStationStatus | null;

/** Card-click sorts that replace the table column sort while active. */
export interface ListOverlay {
  certExpiry: boolean;
  status: StatusOverlay;
}

export interface BaseStationStats {
  total: number;
  online: number;
  offline: number;
  expiringSoon: number;
  expiringCritical: number;
}

/** Certificates inside the warning window, and those inside the critical one. */
const EXPIRING_SOON: ReadonlySet<CertificateExpiryState> = new Set([
  CERTIFICATE_EXPIRY_STATE.EXPIRING,
  CERTIFICATE_EXPIRY_STATE.CRITICAL,
  CERTIFICATE_EXPIRY_STATE.EXPIRED,
]);
const EXPIRING_CRITICAL: ReadonlySet<CertificateExpiryState> = new Set([
  CERTIFICATE_EXPIRY_STATE.CRITICAL,
  CERTIFICATE_EXPIRY_STATE.EXPIRED,
]);

function compareCertExpiry(a: BaseStationUI, b: BaseStationUI): number {
  if (!a.certificateExpiryDate) return b.certificateExpiryDate ? 1 : 0;
  if (!b.certificateExpiryDate) return -1;
  return (
    new Date(a.certificateExpiryDate).getTime() -
    new Date(b.certificateExpiryDate).getTime()
  );
}

function compareStatusFirst(status: BaseStationStatus) {
  return (a: BaseStationUI, b: BaseStationUI): number => {
    if (a.status === status && b.status !== status) return -1;
    if (a.status !== status && b.status === status) return 1;
    return 0;
  };
}

export function arrangeBaseStations(
  stations: BaseStationUI[],
  filters: { search: string; sort: SortState },
  overlay: ListOverlay,
): BaseStationUI[] {
  const overlayActive = overlay.certExpiry || overlay.status !== null;
  const rows = filterAndSortBaseStations(stations, {
    search: filters.search,
    sort: overlayActive ? undefined : filters.sort,
  });
  if (overlay.certExpiry) return [...rows].sort(compareCertExpiry);
  if (overlay.status) return [...rows].sort(compareStatusFirst(overlay.status));
  return rows;
}

export function getBaseStationStats(
  stations: BaseStationUI[],
): BaseStationStats {
  let online = 0;
  let expiringSoon = 0;
  let expiringCritical = 0;
  for (const bs of stations) {
    if (bs.status === BASE_STATION_STATUS.ONLINE) online += 1;
    const { state } = certificateExpiry(bs.certificateExpiryDate);
    if (EXPIRING_SOON.has(state)) expiringSoon += 1;
    if (EXPIRING_CRITICAL.has(state)) expiringCritical += 1;
  }
  return {
    total: stations.length,
    online,
    offline: stations.length - online,
    expiringSoon,
    expiringCritical,
  };
}
