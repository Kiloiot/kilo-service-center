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
  Typography,
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
  ACTION_PICK_ON_MAP,
  BASE_STATION_DETAILS,
  ERR_BS_EUI_EXISTS,
  ERR_BS_NOT_FOUND,
  ERR_UPDATE_BS,
  ERR_UPDATE_BS_EUI,
  ERR_UPDATE_BS_NAME_PARTIAL,
  HELPER_ALTITUDE,
  HELPER_LATITUDE,
  HELPER_LONGITUDE,
  LABEL_ALTITUDE,
  LABEL_LATITUDE,
  LABEL_LOCATION,
  LABEL_LONGITUDE,
  MSG_GPS_AUTHORITATIVE,
  PLACEHOLDER_BS_EUI,
  VAL_BS_EUI_FORMAT,
  VAL_BS_EUI_REQUIRED,
  VAL_LAT_LON_PAIR,
  VAL_LATITUDE_RANGE,
  VAL_LONGITUDE_RANGE,
} from "@constants/messages";
import {
  CheckCircleIcon,
  ContentCopyIcon,
  DownloadIcon,
  GpsFixedIcon,
  MapIcon,
} from "@theme/icons";

import BaseStationCertRegenDialog from "./BaseStationCertRegenDialog";
import MapPickerDialog from "./MapPickerDialog";
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
  const [mapPickerOpen, setMapPickerOpen] = useState(false);
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

  const handleDownloadRegenCert = async (certType: "ca" | "client" | "key") => {
    if (!regenCertData?.downloadUrls) return;

    const certIdMap: Record<"ca" | "client" | "key", string | undefined> = {
      ca: regenCertData.downloadUrls.caCert,
      client: regenCertData.downloadUrls.clientCert,
      key: regenCertData.downloadUrls.privateKey,
    };

    const certId = certIdMap[certType];
    if (!certId) return;

    try {
      const { blob, filename } = await apiService.downloadCertificate(
        certId,
        certType,
      );
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch {
      onError(BASE_STATION_DETAILS.CERTIFICATE_DOWNLOAD_ERROR);
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
            <Typography variant="subtitle2" color="text.secondary" gutterBottom>
              {LABEL_LOCATION}
            </Typography>
            {baseStationDetails?.locationSource === "gps" ? (
              <Alert severity="info" icon={<GpsFixedIcon />} sx={{ mb: 2 }}>
                {MSG_GPS_AUTHORITATIVE}
              </Alert>
            ) : null}
            <Box sx={{ display: "flex", gap: 2, mb: 2 }}>
              <TextField
                label={LABEL_LATITUDE}
                value={editFormData.latitude}
                onChange={(e) => {
                  setEditFormData((prev) => ({
                    ...prev,
                    latitude: e.target.value,
                  }));
                  if (locationErrors.latitude)
                    setLocationErrors((prev) => ({ ...prev, latitude: "" }));
                }}
                type="number"
                helperText={locationErrors.latitude || HELPER_LATITUDE}
                error={!!locationErrors.latitude}
                disabled={baseStationDetails?.locationSource === "gps"}
                sx={{ flex: 1 }}
              />
              <TextField
                label={LABEL_LONGITUDE}
                value={editFormData.longitude}
                onChange={(e) => {
                  setEditFormData((prev) => ({
                    ...prev,
                    longitude: e.target.value,
                  }));
                  if (locationErrors.longitude)
                    setLocationErrors((prev) => ({ ...prev, longitude: "" }));
                }}
                type="number"
                helperText={locationErrors.longitude || HELPER_LONGITUDE}
                error={!!locationErrors.longitude}
                disabled={baseStationDetails?.locationSource === "gps"}
                sx={{ flex: 1 }}
              />
            </Box>
            <Box sx={{ display: "flex", gap: 2, mb: 3 }}>
              <TextField
                label={LABEL_ALTITUDE}
                value={editFormData.altitude}
                onChange={(e) =>
                  setEditFormData((prev) => ({
                    ...prev,
                    altitude: e.target.value,
                  }))
                }
                type="number"
                helperText={HELPER_ALTITUDE}
                disabled={baseStationDetails?.locationSource === "gps"}
                sx={{ flex: 1 }}
              />
              <Box sx={{ flex: 1, display: "flex", alignItems: "center" }}>
                {baseStationDetails?.locationSource !== "gps" && (
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
                setEditFormData((prev) => ({
                  ...prev,
                  latitude: lat.toFixed(6),
                  longitude: lng.toFixed(6),
                }));
                setMapPickerOpen(false);
              }}
              initialLat={
                editFormData.latitude
                  ? parseFloat(editFormData.latitude)
                  : undefined
              }
              initialLng={
                editFormData.longitude
                  ? parseFloat(editFormData.longitude)
                  : undefined
              }
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
            <Divider sx={{ mb: 2 }} />
            <Typography variant="subtitle2" color="text.secondary" gutterBottom>
              {BASE_STATION_DETAILS.CERTIFICATES_SECTION_TITLE}
            </Typography>

            {regenCertData ? (
              <Box>
                <Alert severity="success" sx={{ mb: 2 }}>
                  {BASE_STATION_DETAILS.REGENERATE_CERTS_SUCCESS}
                </Alert>
                {effectiveServiceCenterUrl && (
                  <ScUrlCopyField
                    value={effectiveServiceCenterUrl}
                    copied={scUrlCopied}
                    onCopy={handleCopyScUrl}
                    inputSxGetter={getMonoBody1}
                    mb={2}
                  />
                )}
                <Box display="flex" gap={1} flexWrap="wrap">
                  <Button
                    variant="outlined"
                    size="small"
                    startIcon={<DownloadIcon />}
                    onClick={() => handleDownloadRegenCert("ca")}
                  >
                    {BASE_STATION_DETAILS.DOWNLOAD_CA}
                  </Button>
                  <Button
                    variant="outlined"
                    size="small"
                    startIcon={<DownloadIcon />}
                    onClick={() => handleDownloadRegenCert("client")}
                  >
                    {BASE_STATION_DETAILS.DOWNLOAD_CERT}
                  </Button>
                  <Button
                    variant="outlined"
                    size="small"
                    startIcon={<DownloadIcon />}
                    onClick={() => handleDownloadRegenCert("key")}
                  >
                    {BASE_STATION_DETAILS.DOWNLOAD_KEY}
                  </Button>
                </Box>
              </Box>
            ) : (
              <Box>
                <Typography
                  variant="body2"
                  color="text.secondary"
                  sx={{ mb: 2 }}
                >
                  {BASE_STATION_DETAILS.CERTIFICATES_HINT}
                </Typography>
                <Button
                  variant="outlined"
                  color="warning"
                  onClick={() => setShowRegenConfirm(true)}
                  disabled={isRegenerating}
                >
                  {BASE_STATION_DETAILS.ACTION_REGENERATE_CERTS}
                </Button>
              </Box>
            )}
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
