/**
 * The tile layer every KC-Web map draws: the deployment's tile provider and
 * its attribution.
 */

import { TileLayer } from "react-leaflet";

import { env } from "@config/env";

export function BaseTileLayer() {
  return (
    <TileLayer attribution={env.mapTileAttribution} url={env.mapTileUrl} />
  );
}
