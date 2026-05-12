import React, { useEffect, useState } from "react";

import type { GenerateCertificateResponse } from "@api-types/api";
import type { BaseStationUI } from "@api-types/api";
import {
  useUpdateBaseStation,
  useUpdateBaseStationEui,
} from "@hooks";
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  IconButton,
  InputAdornment,
  TextField,
  Tooltip,
} from "@mui/material";

import { apiService } from "@services/api";
import type { GrpcApiError } from "@services/grpc/client";
import {
  formatEUIWithDashes,
  isValidEUI,
} from "@utils/formatters";
import { getMonoBody1 } from "@utils/typography";
import {
  BS_DETAIL_LAYOUT,
  CERT_VALIDITY_DAYS,
  GEO_BOUNDS,
  TIMING_COPY_FEEDBACK,
} from "@constants/app";
import {
  BASE_STATION_DETAILS,
  ERR_BS_EUI_EXISTS,
  ERR_BS_NOT_FOUND,
  ERR_UPDATE_BS,
  ERR_UPDATE_BS_EUI,
  ERR_UPDATE_BS_NAME_PARTIAL,
  PLACEHOLDER_BS_EUI,
  VAL_BS_EUI_FORMAT,
  VAL_BS_EUI_REQUIRED,
  VAL_LAT_LON_PAIR,
  VAL_LATITUDE_RANGE,
  VAL_LONGITUDE_RANGE,
} from "@constants/messages";
import { CheckCircleIcon, ContentCopyIcon } from "@theme/icons";

import BaseStationCertRegenDialog from "./BaseStationCertRegenDialog";
import BaseStationCertSection from "./BaseStationCertSection";
import BaseStationLocationFields from "./BaseStationLocationFields";
import ScUrlCopyField from "./ScUrlCopyField";

interface BaseStationEditFormData {
  name: string;
  eui: string;
  latitude: string;
  longitude: string;
  altitude: string;
}

interface BaseStationDetailsSnapshot {
  locationSource?: string;
  latitude?: number | null;
  longitude?: number | null;
  altitude?: number | null;
}

/**
 * Validates the manual-location fields. GPS-sourced rows are read-only so
 * an empty errors map is returned. Returns sparse errors map keyed by
 * "latitude" / "longitude".
 */
function validateLocationForm(
  form: BaseStationEditFormData,
  isGps: boolean,
): Record<string, string> {
  if (isGps) return {};
  const errs: Record<string, string> = {};
  const hasLat = form.latitude.trim() !== "";
  const hasLng = form.longitude.trim() !== "";
  if (hasLat !== hasLng) {
    errs.latitude = VAL_LAT_LON_PAIR;
    errs.longitude = VAL_LAT_LON_PAIR;
  }
  if (hasLat) {
    const lat = parseFloat(form.latitude);
    if (
      isNaN(lat) ||
      lat < GEO_BOUNDS.LATITUDE_MIN ||
      lat > GEO_BOUNDS.LATITUDE_MAX
    ) {
      errs.latitude = VAL_LATITUDE_RANGE;
    }
  }
  if (hasLng) {
    const lng = parseFloat(form.longitude);
    if (
      isNaN(lng) ||
      lng < GEO_BOUNDS.LONGITUDE_MIN ||
      lng > GEO_BOUNDS.LONGITUDE_MAX
    ) {
      errs.longitude = VAL_LONGITUDE_RANGE;
    }
  }
  return errs;
}

/**
 * Builds the location update payload. number = set, null = clear,
 * undefined-key = omit. GPS rows always return {} (do not modify).
 */
function buildLocationUpdateData(
  form: BaseStationEditFormData,
  details: BaseStationDetailsSnapshot | null | undefined,
): {
  latitude?: number | null;
  longitude?: number | null;
  altitude?: number | null;
} {
  if (details?.locationSource === "gps") return {};
  const hasLat = form.latitude.trim() !== "";
  const hasLng = form.longitude.trim() !== "";
  const hadLat = details?.latitude != null;
  if (!hasLat && !hasLng && hadLat) {
    return { latitude: null, longitude: null, altitude: null };
  }
  if (hasLat && hasLng) {
    const data: {
      latitude: number;
      longitude: number;
      altitude?: number | null;
    } = {
      latitude: parseFloat(form.latitude),
      longitude: parseFloat(form.longitude),
    };
    if (form.altitude.trim()) {
      data.altitude = parseFloat(form.altitude);
    } else if (details?.altitude != null) {
      data.altitude = null;
    }
    return data;
  }
  return {};
}

/**
 * Returns true when the form's lat/lng/altitude differ from the fetched
 * details. GPS rows always return false (no manual edits propagate).
 */
function hasLocationChanged(
  form: BaseStationEditFormData,
  details: BaseStationDetailsSnapshot | null | undefined,
): boolean {
  if (details?.locationSource === "gps") return false;
  const detailLat = details?.latitude != null ? String(details.latitude) : "";
  const detailLng = details?.longitude != null ? String(details.longitude) : "";
  const detailAlt = details?.altitude != null ? String(details.altitude) : "";
  return (
    form.latitude.trim() !== detailLat ||
    form.longitude.trim() !== detailLng ||
    form.altitude.trim() !== detailAlt
  );
}

interface BaseStationEditDialogProps {
  open: boolean;
  onClose: () => void;
  baseStation: {
    eui: string;
    name?: string;
    serviceCenterUrl: string;
  };
  baseStationDetails: BaseStationUI | null | undefined;
  onSuccess: (newEui?: string) => void;
  onError: (message: string) => void;
}

/** Edit dialog for base station properties, location, SC URL, and certificates. */
const BaseStationEditDialog: React.FC<BaseStationEditDialogProps> = ({
  open,
  onClose,
  baseStation,
  baseStationDetails,
  onSuccess,
  onError,
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
  const [euiCopied, setEuiCopied] = useState(false);
  const [scUrlCopied, setScUrlCopied] = useState(false);

  // Certificate regeneration state
  const [showRegenConfirm, setShowRegenConfirm] = useState(false);
  const [regenCertData, setRegenCertData] =
    useState<GenerateCertificateResponse | null>(null);
  const [isRegenerating, setIsRegenerating] = useState(false);

  const updateBaseStationMutation = useUpdateBaseStation();
  const updateEuiMutation = useUpdateBaseStationEui();

  // Prefer regen-provided URL, then detail-fetched, then list-sourced
  const effectiveServiceCenterUrl =
    regenCertData?.serviceCenterUrl ||
    baseStationDetails?.serviceCenterUrl ||
    baseStation.serviceCenterUrl;

  // Initialize form data when dialog opens
  useEffect(() => {
    if (open) {
      setEditFormData({
        name: baseStation.name || "",
        eui: formatEUIWithDashes(baseStation.eui),
        latitude:
          baseStationDetails?.latitude != null
            ? String(baseStationDetails.latitude)
            : "",
        longitude:
          baseStationDetails?.longitude != null
            ? String(baseStationDetails.longitude)
            : "",
        altitude:
          baseStationDetails?.altitude != null
            ? String(baseStationDetails.altitude)
            : "",
      });
      setEuiError(null);
      setLocationErrors({});
      setEditError(null);
      setRegenCertData(null);
      setShowRegenConfirm(false);
      setIsRegenerating(false);
    }
  }, [open, baseStation.eui, baseStation.name, baseStationDetails]);

  const handleClose = () => {
    setRegenCertData(null);
    setShowRegenConfirm(false);
    setIsRegenerating(false);
    onClose();
  };

  const handleEuiChange = (value: string) => {
    const formatted = formatEUIWithDashes(value);
    setEditFormData((prev) => ({ ...prev, eui: formatted }));
    if (euiError) setEuiError(null);
  };

  const handleCopyEui = async () => {
    const formattedEui = formatEUIWithDashes(baseStation.eui);
    await navigator.clipboard.writeText(formattedEui);
    setEuiCopied(true);
    setTimeout(() => setEuiCopied(false), TIMING_COPY_FEEDBACK);
  };

  const handleCopyScUrl = async () => {
    if (!effectiveServiceCenterUrl) return;
    await navigator.clipboard.writeText(effectiveServiceCenterUrl);
    setScUrlCopied(true);
    setTimeout(() => setScUrlCopied(false), TIMING_COPY_FEEDBACK);
  };

  const validateLocation = (): boolean => {
    const errs = validateLocationForm(
      editFormData,
      baseStationDetails?.locationSource === "gps",
    );
    setLocationErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleUpdateError = (error: unknown) => {
    if (error instanceof Error && error.name === "GrpcApiError") {
      const grpcError = error as GrpcApiError;
      if (grpcError.isNotFound()) {
        setEditError(ERR_BS_NOT_FOUND);
      } else {
        setEditError(ERR_UPDATE_BS);
      }
    } else {
      setEditError(ERR_UPDATE_BS);
    }
  };

  const handleEditSave = async () => {
    setEditError(null);
    const cleanedNewEui = editFormData.eui.replace(/-/g, "").toLowerCase();
    const cleanedOldEui = baseStation.eui.replace(/-/g, "").toLowerCase();
    const euiChanged = cleanedNewEui !== cleanedOldEui;
    const nameChanged = editFormData.name !== (baseStation.name || "");

    if (euiChanged) {
      if (!editFormData.eui.trim()) {
        setEuiError(VAL_BS_EUI_REQUIRED);
        return;
      }
      if (!isValidEUI(editFormData.eui)) {
        setEuiError(VAL_BS_EUI_FORMAT);
        return;
      }
    }

    if (!validateLocation()) return;

    const locationData = buildLocationUpdateData(editFormData, baseStationDetails);
    const locationChanged = hasLocationChanged(editFormData, baseStationDetails);

    if (euiChanged) {
      updateEuiMutation.mutate(
        { eui: baseStation.eui, newEui: cleanedNewEui },
        {
          onSuccess: () => {
            if (nameChanged || locationChanged) {
              updateBaseStationMutation.mutate(
                {
                  eui: cleanedNewEui,
                  data: {
                    name: editFormData.name || undefined,
                    ...locationData,
                  },
                },
                {
                  onSuccess: () => onSuccess(cleanedNewEui),
                  onError: () => {
                    setEditError(ERR_UPDATE_BS_NAME_PARTIAL);
                  },
                },
              );
            } else {
              onSuccess(cleanedNewEui);
            }
          },
          onError: (error) => {
            if (error instanceof Error && error.name === "GrpcApiError") {
              const grpcError = error as GrpcApiError;
              if (grpcError.isAlreadyExists()) {
                setEuiError(ERR_BS_EUI_EXISTS);
              } else if (grpcError.isInvalidArgument()) {
                setEuiError(VAL_BS_EUI_FORMAT);
              } else if (grpcError.isNotFound()) {
                setEditError(ERR_BS_NOT_FOUND);
              } else {
                setEditError(ERR_UPDATE_BS_EUI);
              }
            } else {
              setEditError(ERR_UPDATE_BS_EUI);
            }
          },
        },
      );
    } else if (nameChanged || locationChanged) {
      updateBaseStationMutation.mutate(
        {
          eui: baseStation.eui,
          data: {
            name: editFormData.name || undefined,
            ...locationData,
          },
        },
        {
          onSuccess: () => onSuccess(),
          onError: handleUpdateError,
        },
      );
    } else {
      handleClose();
    }
  };

  const handleRegenerateCerts = async () => {
    setIsRegenerating(true);
    try {
      const response = await apiService.generateCertificate({
        bsEui: formatEUIWithDashes(baseStation.eui),
        validityDays: CERT_VALIDITY_DAYS.THREE_YEARS,
      });
      setRegenCertData(response);
      setShowRegenConfirm(false);
      onError(""); // Clear any previous error
      onSuccess(); // Notify parent of cert regen success implicitly
    } catch {
      onError(BASE_STATION_DETAILS.REGENERATE_CERTS_ERROR);
    } finally {
      setIsRegenerating(false);
    }
  };

  return (
    <>
      <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
        <DialogTitle>{BASE_STATION_DETAILS.DIALOG_EDIT_TITLE}</DialogTitle>
        <DialogContent>
          <Box sx={{ pt: 2 }}>
            {/* Editable EUI field with copy button */}
            <TextField
              label={BASE_STATION_DETAILS.LABEL_EDIT_EUI}
              value={editFormData.eui}
              onChange={(e) => handleEuiChange(e.target.value)}
              error={!!euiError}
              helperText={euiError}
              fullWidth
              placeholder={PLACEHOLDER_BS_EUI}
              slotProps={{
                input: {
                  endAdornment: (
                    <InputAdornment position="end">
                      <Tooltip
                        title={
                          euiCopied
                            ? BASE_STATION_DETAILS.LABEL_EUI_COPIED
                            : BASE_STATION_DETAILS.ACTION_COPY_EUI
                        }
                      >
                        <IconButton
                          size="small"
                          onClick={handleCopyEui}
                          edge="end"
                        >
                          {euiCopied ? (
                            <CheckCircleIcon fontSize="small" color="success" />
                          ) : (
                            <ContentCopyIcon fontSize="small" />
                          )}
                        </IconButton>
                      </Tooltip>
                    </InputAdornment>
                  ),
                  sx: (theme) => getMonoBody1(theme),
                },
              }}
              sx={{ mb: 3 }}
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
              isGps={baseStationDetails?.locationSource === "gps"}
            />

            {/* Service Center URL (read-only with copy) */}
            {effectiveServiceCenterUrl && (
              <ScUrlCopyField
                value={effectiveServiceCenterUrl}
                copied={scUrlCopied}
                onCopy={handleCopyScUrl}
                inputSxGetter={getMonoBody1}
              />
            )}

            {/* Certificates section */}
            <BaseStationCertSection
              regenCertData={regenCertData}
              effectiveServiceCenterUrl={effectiveServiceCenterUrl}
              scUrlCopied={scUrlCopied}
              onCopyScUrl={handleCopyScUrl}
              isRegenerating={isRegenerating}
              onRegenerate={() => setShowRegenConfirm(true)}
              onError={onError}
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
          <Button onClick={handleClose}>
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
        open={showRegenConfirm}
        onClose={() => {
          setShowRegenConfirm(false);
          setIsRegenerating(false);
        }}
        onConfirm={handleRegenerateCerts}
        isRegenerating={isRegenerating}
      />
    </>
  );
};

export default BaseStationEditDialog;
