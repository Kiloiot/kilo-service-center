/**
 * Every located base station on the server as a pin, colored by its
 * online state, framed so all pins are in view.
 */

import { CircleMarker, MapContainer, Popup } from "react-leaflet";
import { Link as RouterLink } from "react-router-dom";

import type { BaseStationLocationUI } from "@api-types/api";
import { Box, Link, Typography, useTheme } from "@mui/material";
import type { FitBoundsOptions } from "leaflet";

import { DataCard } from "@components/common/DataCard";
import { BaseTileLayer } from "@components/common/map/BaseTileLayer";
import { useBaseStationLocations } from "@hooks/useBaseStationLocations";
import { formatEui } from "@utils/eui";
import { MAP_DEFAULTS } from "@constants/app";
import {
  BASE_STATIONS_PAGE,
  ERR_LOAD_BASE_STATIONS,
} from "@constants/messages";
import { baseStationDetailPath } from "@router/paths";
import { componentSpacing } from "@theme/index";

import { stationMapView, stationPinColor } from "../utils/station-map";
import { BaseStationStatusChip } from "./BaseStationStatusChip";

const FIT_OPTIONS: FitBoundsOptions = {
  padding: [MAP_DEFAULTS.FIT_PADDING, MAP_DEFAULTS.FIT_PADDING],
  maxZoom: MAP_DEFAULTS.ZOOM_LOCATION,
};

function StationPopup({ station }: { station: BaseStationLocationUI }) {
  return (
    <Popup>
      <Box
        sx={{
          display: "flex",
          flexDirection: "column",
          gap: componentSpacing.stationMap.popupGap,
        }}
      >
        {station.name && (
          <Typography variant="subtitle2">{station.name}</Typography>
        )}
        <Typography variant="caption">{formatEui(station.eui)}</Typography>
        <BaseStationStatusChip status={station.status} />
        <Link component={RouterLink} to={baseStationDetailPath(station.eui)}>
          {BASE_STATIONS_PAGE.MAP_OPEN}
        </Link>
      </Box>
    </Popup>
  );
}

function StationPin({ station }: { station: BaseStationLocationUI }) {
  const { palette } = useTheme();
  return (
    <CircleMarker
      center={[station.latitude, station.longitude]}
      radius={MAP_DEFAULTS.PIN_RADIUS}
      pathOptions={{
        color: palette.background.paper,
        weight: MAP_DEFAULTS.PIN_STROKE_WEIGHT,
        fillColor: stationPinColor(station.status, palette),
        fillOpacity: MAP_DEFAULTS.PIN_FILL_OPACITY,
      }}
    >
      <StationPopup station={station} />
    </CircleMarker>
  );
}

export function BaseStationMap() {
  const { data: stations = [], isLoading, error } = useBaseStationLocations();
  const view = stationMapView(stations);

  return (
    <DataCard
      title={BASE_STATIONS_PAGE.MAP_TITLE}
      isLoading={isLoading}
      error={error}
      errorFallback={ERR_LOAD_BASE_STATIONS}
    >
      {view ? (
        <MapContainer
          // A new framing remounts the map; a status change keeps the user's pan and zoom.
          key={JSON.stringify(view)}
          {...view}
          boundsOptions={FIT_OPTIONS}
          style={{ height: MAP_DEFAULTS.OVERVIEW_HEIGHT, width: "100%" }}
        >
          <BaseTileLayer />
          {stations.map((station) => (
            <StationPin key={station.eui} station={station} />
          ))}
        </MapContainer>
      ) : (
        <Typography color="text.secondary">
          {BASE_STATIONS_PAGE.MAP_EMPTY}
        </Typography>
      )}
    </DataCard>
  );
}
