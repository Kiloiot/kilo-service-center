/**
 * Framing and pin colors of the dashboard base station map.
 */

import type { BaseStationLocationUI } from "@api-types/api";
import type { Palette } from "@mui/material";

import { BASE_STATION_STATUS, MAP_DEFAULTS } from "@constants/app";

type Position = [number, number];

/** One position is centered at the location zoom; distinct positions are fitted. */
export type StationMapView =
  | { center: Position; zoom: number }
  | { bounds: [Position, Position] };

/** The view that shows every pin, or null when no station has a location. */
export function stationMapView(
  stations: readonly BaseStationLocationUI[],
): StationMapView | null {
  if (stations.length === 0) return null;
  const latitudes = stations.map((station) => station.latitude);
  const longitudes = stations.map((station) => station.longitude);
  const south = Math.min(...latitudes);
  const north = Math.max(...latitudes);
  const west = Math.min(...longitudes);
  const east = Math.max(...longitudes);
  if (south === north && west === east) {
    return { center: [south, west], zoom: MAP_DEFAULTS.ZOOM_LOCATION };
  }
  return {
    bounds: [
      [south, west],
      [north, east],
    ],
  };
}

export function stationPinColor(
  status: BaseStationLocationUI["status"],
  palette: Palette,
): string {
  return status === BASE_STATION_STATUS.ONLINE
    ? palette.success.main
    : palette.error.main;
}
