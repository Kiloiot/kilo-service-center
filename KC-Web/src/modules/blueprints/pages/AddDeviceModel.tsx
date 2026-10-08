/**
 * Add Device Model Page
 *
 * Single-screen editor for creating a device model with a default blueprint atomically.
 * Fields: manufacturer, model name, blueprint version, decoder spec JSON.
 * Code and type_eui are generated server-side.
 */

import React from "react";

import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Paper,
  TextField,
  Typography,
} from "@mui/material";

import { BLUEPRINT_SPEC_ROWS } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";
import { componentSpacing } from "@theme/index";

import CatalogScopeNotice from "../components/CatalogScopeNotice";
import ManufacturerSelectField from "../components/ManufacturerSelectField";
import SpecJsonField from "../components/SpecJsonField";
import { useAddDeviceModelForm, useManufacturers } from "../hooks";

/**
 * AddDeviceModel page component
 */
export const AddDeviceModel: React.FC = () => {
  const form = useAddDeviceModelForm();
  // A model belongs to its manufacturer's catalog, so the list follows the form's catalog.
  const { data: manufacturers = [], isLoading: mfrsLoading } = useManufacturers(
    form.scope,
  );

  return (
    <Box
      sx={{
        p: 3,
        pt: 4,
        maxWidth: componentSpacing.formCard.pageMaxWidth,
        mx: "auto",
      }}
    >
      <Typography variant="h4" sx={{ mb: 3 }}>
        {BLUEPRINT_LABELS.ADD_MODEL}
      </Typography>

      <Paper sx={{ p: 3 }}>
        {form.error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {form.error}
          </Alert>
        )}

        <ManufacturerSelectField
          manufacturers={manufacturers}
          value={form.fields.manufacturerId}
          onChange={form.setField("manufacturerId")}
          disabled={mfrsLoading}
          error={form.fieldErrors.manufacturerId}
        />

        <TextField
          autoFocus
          fullWidth
          required
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_NAME}
          value={form.fields.name}
          onChange={(e) => form.setField("name")(e.target.value)}
          error={!!form.fieldErrors.name}
          helperText={form.fieldErrors.name}
        />

        <TextField
          fullWidth
          required
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_VERSION}
          value={form.fields.version}
          onChange={(e) => form.setField("version")(e.target.value)}
          error={!!form.fieldErrors.version}
          helperText={form.fieldErrors.version}
        />

        <SpecJsonField
          label={BLUEPRINT_LABELS.LABEL_SPEC_JSON}
          value={form.specJsonText}
          onChange={form.setSpecJsonText}
          rows={BLUEPRINT_SPEC_ROWS.PAGE}
          error={form.fieldErrors.specJson}
        />

        <CatalogScopeNotice scope={form.scope} />

        <Box
          sx={{ display: "flex", gap: 2, mt: 3, justifyContent: "flex-end" }}
        >
          <Button onClick={form.cancel}>
            {BLUEPRINT_LABELS.ACTION_CANCEL}
          </Button>
          <Button
            variant="contained"
            onClick={form.submit}
            disabled={form.isPending}
          >
            {form.isPending ? (
              <CircularProgress size={componentSpacing.spinner.button} />
            ) : (
              BLUEPRINT_LABELS.ACTION_CREATE
            )}
          </Button>
        </Box>
      </Paper>
    </Box>
  );
};
