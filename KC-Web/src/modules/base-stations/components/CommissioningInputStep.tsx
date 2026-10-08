import { useState } from "react";

import { Alert, Box, Button, TextField, Typography } from "@mui/material";

import { formatEuiInput } from "@utils/eui";
import {
  ACTION_PICK_ON_MAP,
  HELPER_ALTITUDE,
  HELPER_BS_EUI,
  HELPER_BS_NAME,
  HELPER_LATITUDE,
  HELPER_LONGITUDE,
  INFO_NOTE_PREFIX,
  INFO_TLS_NOTE,
  LABEL_ALTITUDE,
  LABEL_BS_EUI,
  LABEL_BS_NAME,
  LABEL_LATITUDE,
  LABEL_LOCATION_OPTIONAL,
  LABEL_LONGITUDE,
  PLACEHOLDER_BS_EUI,
} from "@constants/messages";
import { MapIcon } from "@theme/icons";

import type {
  CommissioningErrors,
  CommissioningFormData,
} from "../utils/commissioning-form";
import MapPickerDialog from "./MapPickerDialog";

interface CommissioningInputStepProps {
  values: CommissioningFormData;
  errors: CommissioningErrors;
  disabled: boolean;
  onFieldChange: (field: keyof CommissioningFormData, value: string) => void;
  onLocationPick: (latitude: number, longitude: number) => void;
}

export default function CommissioningInputStep({
  values,
  errors,
  disabled,
  onFieldChange,
  onLocationPick,
}: CommissioningInputStepProps) {
  const [mapPickerOpen, setMapPickerOpen] = useState(false);

  return (
    <Box sx={{ pt: 2 }}>
      <TextField
        fullWidth
        label={LABEL_BS_NAME}
        value={values.name}
        onChange={(e) => onFieldChange("name", e.target.value)}
        margin="normal"
        required
        helperText={errors.name || HELPER_BS_NAME}
        error={!!errors.name}
        disabled={disabled}
      />
      <TextField
        fullWidth
        label={LABEL_BS_EUI}
        value={values.eui}
        onChange={(e) => onFieldChange("eui", formatEuiInput(e.target.value))}
        margin="normal"
        required
        placeholder={PLACEHOLDER_BS_EUI}
        helperText={errors.eui || HELPER_BS_EUI}
        error={!!errors.eui}
        disabled={disabled}
      />

      <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>
        {LABEL_LOCATION_OPTIONAL}
      </Typography>
      <Box sx={{ display: "flex", gap: 2 }}>
        <TextField
          label={LABEL_LATITUDE}
          value={values.latitude}
          onChange={(e) => onFieldChange("latitude", e.target.value)}
          type="number"
          helperText={errors.latitude || HELPER_LATITUDE}
          error={!!errors.latitude}
          disabled={disabled}
          sx={{ flex: 1 }}
        />
        <TextField
          label={LABEL_LONGITUDE}
          value={values.longitude}
          onChange={(e) => onFieldChange("longitude", e.target.value)}
          type="number"
          helperText={errors.longitude || HELPER_LONGITUDE}
          error={!!errors.longitude}
          disabled={disabled}
          sx={{ flex: 1 }}
        />
      </Box>
      <Box sx={{ display: "flex", gap: 2, mt: 1 }}>
        <TextField
          label={LABEL_ALTITUDE}
          value={values.altitude}
          onChange={(e) => onFieldChange("altitude", e.target.value)}
          type="number"
          helperText={HELPER_ALTITUDE}
          disabled={disabled}
          sx={{ flex: 1 }}
        />
        <Box sx={{ flex: 1, display: "flex", alignItems: "center" }}>
          <Button
            variant="outlined"
            startIcon={<MapIcon />}
            onClick={() => setMapPickerOpen(true)}
            disabled={disabled}
          >
            {ACTION_PICK_ON_MAP}
          </Button>
        </Box>
      </Box>

      <MapPickerDialog
        open={mapPickerOpen}
        onClose={() => setMapPickerOpen(false)}
        onConfirm={(lat, lng) => {
          onLocationPick(lat, lng);
          setMapPickerOpen(false);
        }}
        initialLat={values.latitude ? parseFloat(values.latitude) : undefined}
        initialLng={values.longitude ? parseFloat(values.longitude) : undefined}
      />

      {errors.general && (
        <Alert severity="error" sx={{ mt: 2 }}>
          {errors.general}
        </Alert>
      )}

      <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
        <strong>{INFO_NOTE_PREFIX}</strong> {INFO_TLS_NOTE}
      </Typography>
    </Box>
  );
}
