/**
 * Location form fields for base station edit dialog.
 * Handles lat/lng/altitude inputs, GPS read-only state, and map picker.
 */

import React, { useState } from "react";

import { Alert, Box, Button, TextField, Typography } from "@mui/material";

import {
  ACTION_PICK_ON_MAP,
  HELPER_ALTITUDE,
  HELPER_LATITUDE,
  HELPER_LONGITUDE,
  LABEL_ALTITUDE,
  LABEL_LATITUDE,
  LABEL_LOCATION,
  LABEL_LONGITUDE,
  MSG_GPS_AUTHORITATIVE,
} from "@constants/messages";
import { GpsFixedIcon, MapIcon } from "@theme/icons";

import MapPickerDialog from "./MapPickerDialog";

interface LocationFieldValues {
  latitude: string;
  longitude: string;
  altitude: string;
}

interface BaseStationLocationFieldsProps {
  values: LocationFieldValues;
  onChange: (field: keyof LocationFieldValues, value: string) => void;
  errors: Record<string, string>;
  onClearError: (field: string) => void;
  isGps: boolean;
}

const BaseStationLocationFields: React.FC<BaseStationLocationFieldsProps> = ({
  values,
  onChange,
  errors,
  onClearError,
  isGps,
}) => {
  const [mapPickerOpen, setMapPickerOpen] = useState(false);

  return (
    <>
      <Typography variant="subtitle2" color="text.secondary" gutterBottom>
        {LABEL_LOCATION}
      </Typography>
      {isGps ? (
        <Alert severity="info" icon={<GpsFixedIcon />} sx={{ mb: 2 }}>
          {MSG_GPS_AUTHORITATIVE}
        </Alert>
      ) : null}
      <Box sx={{ display: "flex", gap: 2, mb: 2 }}>
        <TextField
          label={LABEL_LATITUDE}
          value={values.latitude}
          onChange={(e) => {
            onChange("latitude", e.target.value);
            if (errors.latitude) onClearError("latitude");
          }}
          type="number"
          helperText={errors.latitude || HELPER_LATITUDE}
          error={!!errors.latitude}
          disabled={isGps}
          sx={{ flex: 1 }}
        />
        <TextField
          label={LABEL_LONGITUDE}
          value={values.longitude}
          onChange={(e) => {
            onChange("longitude", e.target.value);
            if (errors.longitude) onClearError("longitude");
          }}
          type="number"
          helperText={errors.longitude || HELPER_LONGITUDE}
          error={!!errors.longitude}
          disabled={isGps}
          sx={{ flex: 1 }}
        />
      </Box>
      <Box sx={{ display: "flex", gap: 2, mb: 3 }}>
        <TextField
          label={LABEL_ALTITUDE}
          value={values.altitude}
          onChange={(e) => onChange("altitude", e.target.value)}
          type="number"
          helperText={HELPER_ALTITUDE}
          disabled={isGps}
          sx={{ flex: 1 }}
        />
        <Box sx={{ flex: 1, display: "flex", alignItems: "center" }}>
          {!isGps && (
            <Button
              variant="outlined"
              startIcon={<MapIcon />}
              onClick={() => setMapPickerOpen(true)}
            >
              {ACTION_PICK_ON_MAP}
            </Button>
          )}
        </Box>
      </Box>

      <MapPickerDialog
        open={mapPickerOpen}
        onClose={() => setMapPickerOpen(false)}
        onConfirm={(lat, lng) => {
          onChange("latitude", lat.toFixed(6));
          onChange("longitude", lng.toFixed(6));
          setMapPickerOpen(false);
        }}
        initialLat={values.latitude ? parseFloat(values.latitude) : undefined}
        initialLng={
          values.longitude ? parseFloat(values.longitude) : undefined
        }
      />
    </>
  );
};

export default BaseStationLocationFields;
