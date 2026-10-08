/**
 * A Leaflet default-icon marker at one position.
 */

import { Marker } from "react-leaflet";

import L from "leaflet";
import markerIcon from "leaflet/dist/images/marker-icon.png";
import markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import markerShadow from "leaflet/dist/images/marker-shadow.png";

// Bundlers rename the icon images, so Leaflet cannot find them by its CSS path.
L.Icon.Default.imagePath = "";
L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

interface PositionMarkerProps {
  position: [number, number];
}

export function PositionMarker({ position }: PositionMarkerProps) {
  return <Marker position={position} />;
}
