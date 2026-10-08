/**
 * Counts behind the dashboard status cards. Base station online/offline
 * counts come from the Base Stations list's own definition.
 */

import type { BaseStationUI, EndpointUI } from "@api-types/api";

import { getBaseStationStats } from "@modules/base-stations/utils/base-station-list";
import {
  DASHBOARD_ADDED_WINDOW_DAYS,
  ENDPOINT_ATTACH_STATUS,
} from "@constants/app";

export interface BaseStationCardCounts {
  total: number;
  online: number;
  offline: number;
  addedLastWeek: number;
}

export interface EndpointCardCounts {
  total: number;
  attached: number;
  addedLastWeek: number;
}

function countAddedSince(items: { createdAt?: string }[], now: Date): number {
  const since = new Date(now);
  since.setDate(since.getDate() - DASHBOARD_ADDED_WINDOW_DAYS);
  return items.filter(
    (item) => item.createdAt && new Date(item.createdAt) >= since,
  ).length;
}

export function baseStationCardCounts(
  stations: BaseStationUI[],
  now: Date,
): BaseStationCardCounts {
  const { total, online, offline } = getBaseStationStats(stations);
  return {
    total,
    online,
    offline,
    addedLastWeek: countAddedSince(stations, now),
  };
}

export function endpointCardCounts(
  endpoints: EndpointUI[],
  now: Date,
): EndpointCardCounts {
  return {
    total: endpoints.length,
    attached: endpoints.filter(
      (ep) => ep.attachStatus === ENDPOINT_ATTACH_STATUS.ATTACHED,
    ).length,
    addedLastWeek: countAddedSince(endpoints, now),
  };
}
