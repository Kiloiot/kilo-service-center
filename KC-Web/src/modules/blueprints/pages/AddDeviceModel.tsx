/**
 * Add Device Model Page
 *
 * Single-screen editor for creating a device model with a default blueprint atomically.
 * Fields: manufacturer, model name, blueprint version, decoder spec JSON.
 * Code and type_eui are generated server-side.
 */

import React, { useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Paper,
  TextField,
  Typography,
} from "@mui/material";
import { useMutation } from "@tanstack/react-query";

import { api } from "@services/api";
import { ROUTES } from "@constants/app";
import { BLUEPRINT_LABELS } from "@constants/messages";

import { useManufacturers } from "../hooks";

/**
 * AddDeviceModel page component
 */
export const AddDeviceModel: React.FC = () => {
  const navigate = useNavigate();

  const { data: manufacturers, isLoading: mfrsLoading } = useManufacturers();

  const [manufacturerId, setManufacturerId] = useState("");
  const [name, setName] = useState("");
  const [version, setVersion] = useState("");
  const [specJsonText, setSpecJsonText] = useState("");
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: () => {
      let specJson: object;
      try {
        specJson = JSON.parse(specJsonText);
      } catch {
        throw new Error(BLUEPRINT_LABELS.ERR_INVALID_JSON);
      }
      return api.createDeviceModelWithBlueprint({
        manufacturerId,
        name,
        version,
        specJson,
      });
    },
    onSuccess: () => {
      navigate(ROUTES.BLUEPRINTS);
    },
    onError: (err: Error) => {
      setError(err.message);
    },
  });

  const handleSubmit = () => {
    setError(null);
    if (!manufacturerId) {
      setError(BLUEPRINT_LABELS.ERR_MANUFACTURER_REQUIRED);
      return;
    }
    if (!name) {
      setError(BLUEPRINT_LABELS.ERR_NAME_REQUIRED);
      return;
    }
    if (!version) {
      setError(BLUEPRINT_LABELS.ERR_VERSION_REQUIRED);
      return;
    }
    mutation.mutate();
  };

  return (
    <Box sx={{ p: 3, pt: 4, maxWidth: 800, mx: "auto" }}>
      <Typography variant="h4" sx={{ mb: 3 }}>
        {BLUEPRINT_LABELS.ADD_MODEL}
      </Typography>

      <Paper sx={{ p: 3 }}>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        <TextField
          select
          fullWidth
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_MANUFACTURER}
          value={manufacturerId}
          onChange={(e) => setManufacturerId(e.target.value)}
          disabled={mfrsLoading}
          slotProps={{
            select: { native: true },
          }}
        >
          <option value="" />
          {manufacturers?.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </TextField>

        <TextField
          autoFocus
          fullWidth
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_NAME}
          value={name}
          onChange={(e) => setName(e.target.value)}
        />

        <TextField
          fullWidth
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_VERSION}
          value={version}
          onChange={(e) => setVersion(e.target.value)}
        />

        <TextField
          fullWidth
          margin="dense"
          label={BLUEPRINT_LABELS.LABEL_SPEC_JSON}
          value={specJsonText}
          onChange={(e) => setSpecJsonText(e.target.value)}
          multiline
          rows={12}
          helperText={BLUEPRINT_LABELS.HELPER_SPEC_JSON}
          slotProps={{
            input: {
              sx: { fontFamily: "monospace", fontSize: "0.875rem" },
            },
          }}
        />

        <Box
          sx={{ display: "flex", gap: 2, mt: 3, justifyContent: "flex-end" }}
        >
          <Button onClick={() => navigate(ROUTES.BLUEPRINTS)}>
            {BLUEPRINT_LABELS.ACTION_CANCEL}
          </Button>
          <Button
            variant="contained"
            onClick={handleSubmit}
            disabled={mutation.isPending}
          >
            {mutation.isPending ? (
              <CircularProgress size={20} />
            ) : (
              BLUEPRINT_LABELS.ACTION_CREATE
            )}
          </Button>
        </Box>
      </Paper>
    </Box>
  );
};
