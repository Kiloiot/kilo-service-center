/**
 * List filtering and sorting shared by the base-station and endpoint listings.
 */

import type { BaseStationUI, EndpointUI } from "@api-types/api";

import { SORT_DIRECTION, type SortDirection } from "@constants/app";

/** The other direction: a second click on the sorted column reverses it. */
export const toggleSortDirection = (direction: SortDirection): SortDirection =>
  direction === SORT_DIRECTION.ASC ? SORT_DIRECTION.DESC : SORT_DIRECTION.ASC;

export interface BaseStationListFilters {
  search?: string;
  status?: string[];
  sort?: { field: string; direction: SortDirection };
}

export interface EndpointListFilters {
  search?: string;
  attachState?: string[];
  activity?: string[];
  sort?: { field: string; direction: SortDirection };
}

export function compareNullableString(
  a: string | undefined | null,
  b: string | undefined | null,
  direction: SortDirection,
): number {
  const aMissing = a === undefined || a === null || a === "";
  const bMissing = b === undefined || b === null || b === "";
  if (aMissing && bMissing) return 0;
  if (aMissing) return 1; // null/empty sort last
  if (bMissing) return -1;
  const cmp = (a as string).localeCompare(b as string);
  return direction === SORT_DIRECTION.ASC ? cmp : -cmp;
}

export function compareNullableNumber(
  a: number | undefined | null,
  b: number | undefined | null,
  direction: SortDirection,
): number {
  const aMissing = a === undefined || a === null;
  const bMissing = b === undefined || b === null;
  if (aMissing && bMissing) return 0;
  if (aMissing) return 1;
  if (bMissing) return -1;
  const cmp = (a as number) - (b as number);
  return direction === SORT_DIRECTION.ASC ? cmp : -cmp;
}

export type Comparator<T> = (a: T, b: T, direction: SortDirection) => number;

/** Sortable base station columns; unknown fields leave the order untouched. */
export const baseStationComparators: Record<
  string,
  Comparator<BaseStationUI>
> = {
  name: (a, b, d) => compareNullableString(a.name || a.eui, b.name || b.eui, d),
  eui: (a, b, d) => compareNullableString(a.eui, b.eui, d),
  status: (a, b, d) => compareNullableString(a.status, b.status, d),
  connectionType: (a, b, d) =>
    compareNullableString(a.connectionType, b.connectionType, d),
  lastSeen: (a, b, d) => compareNullableString(a.lastSeen, b.lastSeen, d),
};

/** Sortable endpoint columns; unknown fields leave the order untouched. */
export const endpointComparators: Record<string, Comparator<EndpointUI>> = {
  name: (a, b, d) => compareNullableString(a.name, b.name, d),
  epEui: (a, b, d) => compareNullableString(a.epEui, b.epEui, d),
  attachState: (a, b, d) =>
    compareNullableString(a.attachStatus, b.attachStatus, d),
  status: (a, b, d) => compareNullableString(a.status, b.status, d),
  lastSeen: (a, b, d) => compareNullableString(a.lastSeen, b.lastSeen, d),
  packetCnt: (a, b, d) =>
    compareNullableNumber(a.lastPacketCnt, b.lastPacketCnt, d),
};

export function filterAndSortBaseStations(
  items: BaseStationUI[],
  filters?: BaseStationListFilters,
): BaseStationUI[] {
  if (!filters) return items;
  let out = items;
  if (filters.search) {
    const needle = filters.search.toLowerCase();
    out = out.filter(
      (bs) =>
        (bs.name?.toLowerCase().includes(needle) ?? false) ||
        bs.eui.toLowerCase().includes(needle),
    );
  }
  if (filters.status && filters.status.length > 0) {
    const allowed = new Set(filters.status);
    out = out.filter((bs) => allowed.has(bs.status));
  }
  if (filters.sort) {
    const compare = baseStationComparators[filters.sort.field];
    if (compare) {
      const { direction } = filters.sort;
      out = [...out].sort((a, b) => compare(a, b, direction));
    }
  }
  return out;
}

export function filterAndSortEndpoints(
  items: EndpointUI[],
  filters?: EndpointListFilters,
): EndpointUI[] {
  if (!filters) return items;
  let out = items;
  if (filters.search) {
    const needle = filters.search.toLowerCase();
    out = out.filter(
      (ep) =>
        (ep.name?.toLowerCase().includes(needle) ?? false) ||
        ep.epEui.toLowerCase().includes(needle),
    );
  }
  if (filters.attachState && filters.attachState.length > 0) {
    const allowed = new Set(filters.attachState);
    out = out.filter((ep) => allowed.has(ep.attachStatus));
  }
  if (filters.activity && filters.activity.length > 0) {
    const allowed = new Set(filters.activity);
    out = out.filter((ep) => allowed.has(ep.status));
  }
  if (filters.sort) {
    const compare = endpointComparators[filters.sort.field];
    if (compare) {
      const { direction } = filters.sort;
      out = [...out].sort((a, b) => compare(a, b, direction));
    }
  }
  return out;
}
