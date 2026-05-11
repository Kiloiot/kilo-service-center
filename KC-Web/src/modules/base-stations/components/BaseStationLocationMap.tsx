/**
 * Read-only map card for base station detail page.
 * Displays location on an interactive-disabled Leaflet map with coordinate footer,
 * or a placeholder when no location is set.
 */

import { MapContainer, Marker, TileLayer } from "react-leaflet";

import { Box, Card, Chip, Typography } from "@mui/material";
import L from "leaflet";
import markerIcon from "leaflet/dist/images/marker-icon.png";
// Fix Leaflet default marker icon paths (broken by bundlers)
import markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import markerShadow from "leaflet/dist/images/marker-shadow.png";

import { formatAltitude } from "@utils/formatters";
import { MAP_DEFAULTS } from "@constants/app";
import {
  LABEL_ALTITUDE,
  LABEL_LATITUDE,
  LABEL_LOCATION_SOURCE_GPS,
  LABEL_LOCATION_SOURCE_MANUAL,
  LABEL_LONGITUDE,
  MSG_NO_LOCATION_SET,
} from "@constants/messages";
import { env } from "@config/env";
import { GpsFixedIcon, MapIcon } from "@theme/icons";

delete (L.Icon.Default.prototype as unknown as Record<string, unknown>)
  ._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

interface BaseStationLocationMapProps {
  latitude?: number | null;
  longitude?: number | null;
  altitude?: number | null;
  locationSource?: string | null;
}

export default function BaseStationLocationMap({
  latitude,
  longitude,
  altitude,
  locationSource,
}: BaseStationLocationMapProps) {
  const hasLocation = latitude != null && longitude != null;

  const cardSx = {
    width: "100%",
    aspectRatio: "1 / 1",
    overflow: "hidden",
    position: "relative" as const,
    display: "flex",
    flexDirection: "column" as const,
  };

  if (!hasLocation) {
    return (
      <Card sx={{ ...cardSx, justifyContent: "center", alignItems: "center" }}>
        <MapIcon sx={{ fontSize: 48, color: "text.disabled", mb: 1 }} />
        <Typography variant="body2" color="text.secondary">
          {MSG_NO_LOCATION_SET}
        </Typography>
      </Card>
    );
  }

  const isGps = locationSource === "gps";
  const tileUrl = env.mapTileUrl || MAP_DEFAULTS.TILE_URL;
  const tileAttribution =
    env.mapTileAttribution || MAP_DEFAULTS.TILE_ATTRIBUTION;

  return (
    <Card sx={cardSx}>
      <Box sx={{ flex: 1, position: "relative", minHeight: 0 }}>
        <MapContainer
          center={[latitude, longitude]}
          zoom={MAP_DEFAULTS.ZOOM_LOCATION}
          style={{ height: "100%", width: "100%" }}
          scrollWheelZoom={true}
          doubleClickZoom={true}
          touchZoom={true}
          zoomControl={true}
        >
          <TileLayer attribution={tileAttribution} url={tileUrl} />
          <Marker position={[latitude, longitude]} />
        </MapContainer>
      </Box>
      <Box
        sx={{
          px: 2,
          py: 1.5,
          bgcolor: "background.paper",
          borderTop: 1,
          borderColor: "divider",
          display: "flex",
          flexWrap: "wrap",
          gap: 2,
          alignItems: "center",
        }}
      >
        <Box>
          <Typography variant="caption" color="text.secondary">
            {LABEL_LATITUDE}
          </Typography>
          <Typography variant="body2">{latitude.toFixed(6)}</Typography>
        </Box>
        <Box>
          <Typography variant="caption" color="text.secondary">
            {LABEL_LONGITUDE}
          </Typography>
          <Typography variant="body2">{longitude.toFixed(6)}</Typography>
        </Box>
        {altitude != null && (
          <Box>
            <Typography variant="caption" color="text.secondary">
              {LABEL_ALTITUDE}
            </Typography>
            <Typography variant="body2">{formatAltitude(altitude)}</Typography>
          </Box>
        )}
        <Chip
          icon={isGps ? <GpsFixedIcon /> : undefined}
          label={
            isGps ? LABEL_LOCATION_SOURCE_GPS : LABEL_LOCATION_SOURCE_MANUAL
          }
          color="success"
          size="small"
          variant="outlined"
        />
      </Box>
    </Card>
  );
}
