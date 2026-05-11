import React, { useEffect, useRef, useState } from "react";

import type { DeviceModelUI, ManufacturerUI } from "@api-types/api";
import { Autocomplete, TextField, Typography } from "@mui/material";
import Grid from "@mui/material/Grid";
import { useQuery } from "@tanstack/react-query";

import { useDeviceModels, useManufacturers } from "@modules/blueprints/hooks";
import { api } from "@services/api";
import { ENDPOINT_DETAILS, ENDPOINT_FORM } from "@constants/messages";
import { queryKeys } from "@config/query-keys";

// Discriminated union option types for clear sentinel safety
type MfgOption = { kind: "real"; data: ManufacturerUI } | { kind: "clear" };
type ModelOption = { kind: "real"; data: DeviceModelUI } | { kind: "clear" };

interface DeviceModelSelectorProps {
  value?: string;
  onChange: (id: string | undefined) => void;
  disabled?: boolean;
}

/**
 * Cascading manufacturer -> device model selector for blueprint association.
 * Optional field — clearing either dropdown returns undefined.
 *
 * Uses query-driven bootstrap: when `value` (existing deviceModelId) is provided,
 * fetches the model via React Query to resolve its manufacturerId, then hydrates
 * both dropdowns deterministically.
 */
const DeviceModelSelector: React.FC<DeviceModelSelectorProps> = ({
  value,
  onChange,
  disabled,
}) => {
  const { data: manufacturers = [], isLoading: loadingMfg } =
    useManufacturers();
  const [selectedMfg, setSelectedMfg] = useState<ManufacturerUI | null>(null);
  const { data: deviceModels = [], isLoading: loadingModels } = useDeviceModels(
    selectedMfg?.id,
  );
  const [selectedModel, setSelectedModel] = useState<DeviceModelUI | null>(
    null,
  );

  // Query-driven bootstrap: fetch the model by ID to get its manufacturerId
  const { data: resolvedModel } = useQuery({
    queryKey: queryKeys.blueprints.deviceModelDetail(value ?? ""),
    queryFn: () => api.getDeviceModel(value!),
    enabled: !!value,
  });

  // Effect 1 — Value transition reset: clear stale local state only when value prop changes
  const prevValueRef = useRef(value);
  useEffect(() => {
    if (value !== prevValueRef.current) {
      setSelectedMfg(null);
      setSelectedModel(null);
      prevValueRef.current = value;
    }
  }, [value]);

  // Effect 2 — Manufacturer resolution from resolved model
  useEffect(() => {
    if (resolvedModel && manufacturers.length > 0 && !selectedMfg) {
      const mfg = manufacturers.find(
        (m) => m.id === resolvedModel.manufacturerId,
      );
      if (mfg) setSelectedMfg(mfg);
    }
  }, [resolvedModel, manufacturers, selectedMfg]);

  // Effect 3 — Model resolution once device models load for the resolved manufacturer
  useEffect(() => {
    if (value && deviceModels.length > 0 && !selectedModel) {
      const model = deviceModels.find((m) => m.id === value);
      if (model) setSelectedModel(model);
    }
  }, [value, deviceModels, selectedModel]);

  // Clear option visibility keyed on value prop (existing bond), not just local state
  const hasBond = !!value;

  const mfgOptions: MfgOption[] = [
    ...(hasBond || selectedMfg ? [{ kind: "clear" as const }] : []),
    ...manufacturers.map((m) => ({ kind: "real" as const, data: m })),
  ];

  const modelOptions: ModelOption[] = [
    ...(hasBond || selectedModel ? [{ kind: "clear" as const }] : []),
    ...deviceModels.map((m) => ({ kind: "real" as const, data: m })),
  ];

  const handleMfgChange = (_: unknown, opt: MfgOption | null) => {
    if (!opt || opt.kind === "clear") {
      setSelectedMfg(null);
      setSelectedModel(null);
      onChange(undefined);
      return;
    }
    setSelectedMfg(opt.data);
    setSelectedModel(null);
    onChange(undefined);
  };

  const handleModelChange = (_: unknown, opt: ModelOption | null) => {
    if (!opt || opt.kind === "clear") {
      setSelectedModel(null);
      onChange(undefined);
      return;
    }
    setSelectedModel(opt.data);
    onChange(opt.data.id);
  };

  // Map local selection back to MfgOption for Autocomplete value
  const mfgValue: MfgOption | null = selectedMfg
    ? { kind: "real", data: selectedMfg }
    : null;
  const modelValue: ModelOption | null = selectedModel
    ? { kind: "real", data: selectedModel }
    : null;

  return (
    <>
      <Grid size={12}>
        <Typography variant="subtitle2" fontWeight="bold" mb={1} mt={1}>
          {ENDPOINT_FORM.SECTION_BLUEPRINT}
        </Typography>
        <Typography variant="body2" color="text.secondary" mb={1}>
          {ENDPOINT_FORM.HELPER_DEVICE_MODEL}
        </Typography>
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <Autocomplete
          options={mfgOptions}
          getOptionLabel={(opt) =>
            opt.kind === "clear"
              ? ENDPOINT_FORM.OPTION_NONE_MANUFACTURER
              : opt.data.name
          }
          isOptionEqualToValue={(opt, val) => {
            if (opt.kind === "clear" && val.kind === "clear") return true;
            if (opt.kind === "real" && val.kind === "real")
              return opt.data.id === val.data.id;
            return false;
          }}
          value={mfgValue}
          onChange={handleMfgChange}
          loading={loadingMfg}
          disabled={disabled}
          renderInput={(params) => (
            <TextField
              {...params}
              label={ENDPOINT_DETAILS.OPTION_SELECT_MANUFACTURER}
              placeholder={
                manufacturers.length === 0 && !loadingMfg
                  ? ENDPOINT_FORM.PLACEHOLDER_NO_MANUFACTURERS
                  : undefined
              }
            />
          )}
        />
      </Grid>

      <Grid size={{ xs: 12, md: 6 }}>
        <Autocomplete
          options={modelOptions}
          getOptionLabel={(opt) =>
            opt.kind === "clear"
              ? ENDPOINT_FORM.OPTION_NONE_MODEL
              : opt.data.name
          }
          isOptionEqualToValue={(opt, val) => {
            if (opt.kind === "clear" && val.kind === "clear") return true;
            if (opt.kind === "real" && val.kind === "real")
              return opt.data.id === val.data.id;
            return false;
          }}
          value={modelValue}
          onChange={handleModelChange}
          loading={loadingModels}
          disabled={disabled || !selectedMfg}
          renderInput={(params) => (
            <TextField
              {...params}
              label={ENDPOINT_DETAILS.OPTION_SELECT_MODEL}
              placeholder={
                selectedMfg && deviceModels.length === 0 && !loadingModels
                  ? ENDPOINT_FORM.PLACEHOLDER_NO_DEVICE_MODELS
                  : undefined
              }
            />
          )}
        />
      </Grid>
    </>
  );
};

export default DeviceModelSelector;
