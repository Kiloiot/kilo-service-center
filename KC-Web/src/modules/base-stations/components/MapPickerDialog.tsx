/**
 * Map Picker Dialog
 *
 * Leaflet-based map dialog for selecting geographic coordinates.
 * Uses OpenStreetMap tiles — no API key required.
 */

import { useCallback, useEffect, useState } from "react";
import { MapContainer, Marker, TileLayer, useMapEvents } from "react-leaflet";

import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from "@mui/material";
import L from "leaflet";
import markerIcon from "leaflet/dist/images/marker-icon.png";
// Fix Leaflet default marker icon paths (broken by bundlers)
import markerIcon2x from "leaflet/dist/images/marker-icon-2x.png";
import markerShadow from "leaflet/dist/images/marker-shadow.png";

import { MAP_DEFAULTS } from "@constants/app";
import {
  ACTION_CANCEL,
  ACTION_CONFIRM_LOCATION,
  INSTR_MAP_PICKER,
  TITLE_MAP_PICKER,
} from "@constants/messages";
import { env } from "@config/env";

delete (L.Icon.Default.prototype as unknown as Record<string, unknown>)
  ._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

interface MapPickerDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (lat: number, lng: number) => void;
  initialLat?: number;
  initialLng?: number;
}

/** Inner component that listens for map click events */
function MapClickHandler({
  onLocationSelect,
}: {
  onLocationSelect: (lat: number, lng: number) => void;
}) {
  useMapEvents({
    click(e) {
      onLocationSelect(e.latlng.lat, e.latlng.lng);
    },
  });
  return null;
}

export default function MapPickerDialog({
  open,
  onClose,
  onConfirm,
  initialLat,
  initialLng,
}: MapPickerDialogProps) {
  const [selectedPosition, setSelectedPosition] = useState<
    [number, number] | null
  >(null);

  useEffect(() => {
    if (open) {
      setSelectedPosition(
        initialLat !== undefined && initialLng !== undefined
          ? [initialLat, initialLng]
          : null,
      );
    }
  }, [open, initialLat, initialLng]);

  const handleLocationSelect = useCallback((lat: number, lng: number) => {
    setSelectedPosition([lat, lng]);
  }, []);

  const handleConfirm = () => {
    if (selectedPosition) {
      onConfirm(selectedPosition[0], selectedPosition[1]);
    }
    onClose();
  };

  const hasInitial = initialLat !== undefined && initialLng !== undefined;
  const center: [number, number] = hasInitial
    ? [initialLat, initialLng]
    : [MAP_DEFAULTS.CENTER_LAT, MAP_DEFAULTS.CENTER_LNG];
  const zoom = hasInitial
    ? MAP_DEFAULTS.ZOOM_LOCATION
    : MAP_DEFAULTS.ZOOM_DEFAULT;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{TITLE_MAP_PICKER}</DialogTitle>
      <DialogContent>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
          {INSTR_MAP_PICKER}
        </Typography>
        <MapContainer
          center={center}
          zoom={zoom}
          style={{ height: MAP_DEFAULTS.PICKER_HEIGHT, width: "100%" }}
        >
          <TileLayer
            attribution={
              env.mapTileAttribution || MAP_DEFAULTS.TILE_ATTRIBUTION
            }
            url={env.mapTileUrl || MAP_DEFAULTS.TILE_URL}
          />
          <MapClickHandler onLocationSelect={handleLocationSelect} />
          {selectedPosition && <Marker position={selectedPosition} />}
        </MapContainer>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{ACTION_CANCEL}</Button>
        <Button
          onClick={handleConfirm}
          variant="contained"
          disabled={!selectedPosition}
        >
          {ACTION_CONFIRM_LOCATION}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
