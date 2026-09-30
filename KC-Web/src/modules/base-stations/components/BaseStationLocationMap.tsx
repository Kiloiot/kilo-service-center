/**
 * Read-only map card for base station detail page.
 * Displays location on an interactive-disabled Leaflet map with coordinate footer,
 * or a placeholder when no location is set.
 */

import { MapContainer } from "react-leaflet";

import { Box, Card, Chip, Typography } from "@mui/material";

import { BaseTileLayer } from "@components/common/map/BaseTileLayer";
import { PositionMarker } from "@components/common/map/PositionMarker";
import { formatAltitude } from "@utils/formatters";
import {
  FRACTION_DIGITS,
  MAP_DEFAULTS,
  STATION_LOCATION_STATE,
} from "@constants/app";
import {
  LABEL_ALTITUDE,
  LABEL_LATITUDE,
  LABEL_LOCATION_SOURCE_GPS,
  LABEL_LOCATION_SOURCE_MANUAL,
  LABEL_LONGITUDE,
  MSG_NO_GPS_FIX,
  MSG_NO_LOCATION_SET,
} from "@constants/messages";
import { GpsFixedIcon, MapIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import { stationLocationState } from "../utils/location-state";

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
  const state = stationLocationState({ locationSource, latitude, longitude });

  const cardSx = {
    width: "100%",
    aspectRatio: "1 / 1",
    overflow: "hidden",
    position: "relative" as const,
    display: "flex",
    flexDirection: "column" as const,
  };

  if (latitude == null || longitude == null) {
    return (
      <Card sx={{ ...cardSx, justifyContent: "center", alignItems: "center" }}>
        <MapIcon
          sx={{
            fontSize: componentSpacing.resultIcon.size,
            color: "text.disabled",
            mb: 1,
          }}
        />
        <Typography variant="body2" color="text.secondary">
          {state === STATION_LOCATION_STATE.GPS_NO_FIX
            ? MSG_NO_GPS_FIX
            : MSG_NO_LOCATION_SET}
        </Typography>
      </Card>
    );
  }

  const isGps = state === STATION_LOCATION_STATE.GPS_FIX;

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
          <BaseTileLayer />
          <PositionMarker position={[latitude, longitude]} />
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
          <Typography variant="body2">
            {latitude.toFixed(FRACTION_DIGITS.COORDINATE)}
          </Typography>
        </Box>
        <Box>
          <Typography variant="caption" color="text.secondary">
            {LABEL_LONGITUDE}
          </Typography>
          <Typography variant="body2">
            {longitude.toFixed(FRACTION_DIGITS.COORDINATE)}
          </Typography>
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
