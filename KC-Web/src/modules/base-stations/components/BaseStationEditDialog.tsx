import React, { useEffect, useState } from "react";

import type { BaseStationUI } from "@api-types/api";
import { useUpdateBaseStation, useUpdateBaseStationEui } from "@hooks";
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  TextField,
} from "@mui/material";

import { useClipboard } from "@hooks/useClipboard";
import {
  formatEui,
  formatEuiInput,
  normalizeEui,
  validateEui,
} from "@utils/eui";
import { getMonoBody1 } from "@utils/typography";
import {
  BS_COPY_FIELDS,
  BS_DETAIL_LAYOUT,
  STATION_LOCATION_STATE,
} from "@constants/app";
import {
  BASE_STATION_DETAILS,
  VAL_BS_EUI_FORMAT,
  VAL_BS_EUI_REQUIRED,
} from "@constants/messages";

import { useCertificateRegeneration } from "../hooks";
import {
  type BaseStationEditFormData,
  buildLocationUpdateData,
  euiUpdateFailure,
  hasLocationChanged,
  initialEditForm,
  updateFailure,
  validateLocationForm,
} from "../utils/base-station-edit-form";
import { stationLocationState } from "../utils/location-state";
import BaseStationCertRegenDialog from "./BaseStationCertRegenDialog";
import BaseStationCertSection from "./BaseStationCertSection";
import BaseStationEuiField from "./BaseStationEuiField";
import BaseStationLocationFields from "./BaseStationLocationFields";
import ScUrlCopyField from "./ScUrlCopyField";

interface BaseStationEditDialogProps {
  open: boolean;
  onClose: () => void;
  baseStation: {
    eui: string;
    name?: string;
    serviceCenterUrl: string;
  };
  baseStationDetails: BaseStationUI | null | undefined;
  onSuccess: () => void;
  onEuiChange: (newEui: string, changesSaved: boolean) => void;
}

/** Edit dialog for base station properties, location, SC URL, and certificates. */
const BaseStationEditDialog: React.FC<BaseStationEditDialogProps> = ({
  open,
  onClose,
  baseStation,
  baseStationDetails,
  onSuccess,
  onEuiChange,
}) => {
  const [editFormData, setEditFormData] = useState<BaseStationEditFormData>({
    name: "",
    eui: "",
    latitude: "",
    longitude: "",
    altitude: "",
  });
  const [euiError, setEuiError] = useState<string | null>(null);
  const [locationErrors, setLocationErrors] = useState<Record<string, string>>(
    {},
  );
  const [editError, setEditError] = useState<string | null>(null);
  const clipboard = useClipboard();

  const regeneration = useCertificateRegeneration(
    baseStation.eui,
    open,
    setEditError,
  );

  const updateBaseStationMutation = useUpdateBaseStation();
  const updateEuiMutation = useUpdateBaseStationEui();

  // Prefer regen-provided URL, then detail-fetched, then list-sourced
  const effectiveServiceCenterUrl =
    regeneration.issued?.serviceCenterUrl ||
    baseStationDetails?.serviceCenterUrl ||
    baseStation.serviceCenterUrl;

  // Initialize form data when dialog opens
  useEffect(() => {
    if (open) {
      setEditFormData(
        initialEditForm(baseStation.name, baseStation.eui, baseStationDetails),
      );
      setEuiError(null);
      setLocationErrors({});
      setEditError(null);
    }
  }, [open, baseStation.eui, baseStation.name, baseStationDetails]);

  const handleEuiChange = (value: string) => {
    setEditFormData((prev) => ({ ...prev, eui: formatEuiInput(value) }));
    if (euiError) setEuiError(null);
  };

  const handleCopyEui = () =>
    clipboard.copy(formatEui(baseStation.eui), BS_COPY_FIELDS.EUI);
  const handleCopyScUrl = () =>
    clipboard.copy(effectiveServiceCenterUrl, BS_COPY_FIELDS.SC_URL);

  const validateLocation = (): boolean => {
    const errs = validateLocationForm(
      editFormData,
      stationLocationState(baseStationDetails) ===
        STATION_LOCATION_STATE.GPS_FIX,
    );
    setLocationErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleEditSave = async () => {
    setEditError(null);
    const cleanedNewEui = normalizeEui(editFormData.eui);
    const euiChanged = cleanedNewEui !== normalizeEui(baseStation.eui);
    const nameChanged = editFormData.name !== (baseStation.name || "");

    if (euiChanged) {
      if (!editFormData.eui.trim()) {
        setEuiError(VAL_BS_EUI_REQUIRED);
        return;
      }
      if (validateEui(editFormData.eui) !== null) {
        setEuiError(VAL_BS_EUI_FORMAT);
        return;
      }
    }

    if (!validateLocation()) return;

    const locationData = buildLocationUpdateData(
      editFormData,
      baseStationDetails,
    );
    const locationChanged = hasLocationChanged(
      editFormData,
      baseStationDetails,
    );
    const changes =
      nameChanged || locationChanged
        ? { name: editFormData.name || undefined, ...locationData }
        : undefined;

    if (euiChanged) {
      // Awaited rather than mutate callbacks: the list reloads under the old
      // EUI's route and unmounts this dialog before the move completes.
      try {
        const { changesSaved } = await updateEuiMutation.mutateAsync({
          eui: baseStation.eui,
          newEui: cleanedNewEui,
          changes,
        });
        onEuiChange(cleanedNewEui, changesSaved);
      } catch (error) {
        const failure = euiUpdateFailure(error);
        (failure.onEui ? setEuiError : setEditError)(failure.message);
      }
    } else if (changes) {
      updateBaseStationMutation.mutate(
        { eui: baseStation.eui, data: changes },
        {
          onSuccess: () => onSuccess(),
          onError: (error) => setEditError(updateFailure(error)),
        },
      );
    } else {
      onClose();
    }
  };

  return (
    <>
      <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
        <DialogTitle>{BASE_STATION_DETAILS.DIALOG_EDIT_TITLE}</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2 }}>
            <BaseStationEuiField
              value={editFormData.eui}
              error={euiError}
              copied={clipboard.copiedField === BS_COPY_FIELDS.EUI}
              onChange={handleEuiChange}
              onCopy={handleCopyEui}
            />

            {/* Basic fields section */}
            <TextField
              autoFocus
              fullWidth
              label={BASE_STATION_DETAILS.LABEL_EDIT_NAME}
              value={editFormData.name}
              onChange={(e) =>
                setEditFormData((prev) => ({ ...prev, name: e.target.value }))
              }
              sx={{ mb: 3 }}
            />

            {/* Location fields */}
            <Divider sx={{ mb: 2 }} />
            <BaseStationLocationFields
              values={{
                latitude: editFormData.latitude,
                longitude: editFormData.longitude,
                altitude: editFormData.altitude,
              }}
              onChange={(field, value) =>
                setEditFormData((prev) => ({ ...prev, [field]: value }))
              }
              errors={locationErrors}
              onClearError={(field) =>
                setLocationErrors((prev) => ({ ...prev, [field]: "" }))
              }
              isGps={
                stationLocationState(baseStationDetails) ===
                STATION_LOCATION_STATE.GPS_FIX
              }
            />

            {/* Service Center URL (read-only with copy) */}
            {effectiveServiceCenterUrl && (
              <ScUrlCopyField
                value={effectiveServiceCenterUrl}
                copied={clipboard.copiedField === BS_COPY_FIELDS.SC_URL}
                onCopy={handleCopyScUrl}
                inputSxGetter={getMonoBody1}
              />
            )}

            {/* Certificates section */}
            <BaseStationCertSection
              regenCertData={regeneration.issued}
              effectiveServiceCenterUrl={effectiveServiceCenterUrl}
              scUrlCopied={clipboard.copiedField === BS_COPY_FIELDS.SC_URL}
              onCopyScUrl={handleCopyScUrl}
              isRegenerating={regeneration.isRegenerating}
              onRegenerate={regeneration.askToConfirm}
              onError={setEditError}
            />
          </Box>
        </DialogContent>
        {editError && (
          <Alert
            severity="error"
            sx={{
              mx: BS_DETAIL_LAYOUT.DIALOG_ALERT_MX,
              mb: BS_DETAIL_LAYOUT.DIALOG_ALERT_MB,
            }}
            onClose={() => setEditError(null)}
          >
            {editError}
          </Alert>
        )}
        <DialogActions>
          <Button onClick={onClose}>
            {BASE_STATION_DETAILS.ACTION_CANCEL}
          </Button>
          <Button
            onClick={handleEditSave}
            variant="contained"
            disabled={
              updateBaseStationMutation.isPending || updateEuiMutation.isPending
            }
          >
            {updateBaseStationMutation.isPending || updateEuiMutation.isPending
              ? BASE_STATION_DETAILS.ACTION_SAVING
              : BASE_STATION_DETAILS.ACTION_SAVE}
          </Button>
        </DialogActions>
      </Dialog>

      <BaseStationCertRegenDialog
        open={regeneration.confirming}
        onClose={regeneration.cancel}
        onConfirm={regeneration.regenerate}
        isRegenerating={regeneration.isRegenerating}
      />
    </>
  );
};

export default BaseStationEditDialog;
