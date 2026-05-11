import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import type { GenerateCertificateResponse } from "@api-types/api";
import {
  useBaseStation,
  useDeleteBaseStation,
  useUpdateBaseStation,
  useUpdateBaseStationEui,
} from "@hooks";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Divider,
  Grid,
  IconButton,
  InputAdornment,
  Snackbar,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";

import { apiService } from "@services/api";
import type { GrpcApiError } from "@services/grpc/client";
import { realtimeService } from "@services/realtime";
import {
  calculateDaysUntilExpiry,
  formatDate,
  formatDateTime,
  formatEUIWithDashes,
  isValidEUI,
  truncateWithEllipsis,
} from "@utils/formatters";
import { getMonoBody1 } from "@utils/typography";
import {
  BS_DETAIL_LAYOUT,
  CERT_VALIDITY_DAYS,
  TIMING_COPY_FEEDBACK,
  TRUNCATION,
} from "@constants/app";
import { GEO_BOUNDS } from "@constants/app";
import {
  ACTION_CANCEL,
  ACTION_DELETE,
  ACTION_EDIT,
  ACTION_PICK_ON_MAP,
  BASE_STATION_DETAILS,
  ERR_BS_EUI_EXISTS,
  ERR_BS_NOT_FOUND,
  ERR_DELETE_BASE_STATION,
  ERR_LOAD_BS_DETAILS,
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
  LOADER,
  MSG_BS_EUI_UPDATED,
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
  DeleteIcon,
  DownloadIcon,
  EditIcon,
  GpsFixedIcon,
  MapIcon,
} from "@theme/icons";

import BaseStationLocationMap from "./BaseStationLocationMap";
import BaseStationMessages from "./BaseStationMessages";
import MapPickerDialog from "./MapPickerDialog";

interface BaseStationDetailsProps {
  baseStation: {
    id: string;
    eui: string;
    name?: string;
    status: "online" | "offline";
    connectionType: "BSSCI" | "MQTT";
    lastSeen: string;
    serviceCenterUrl: string;
    certificateExpiryDate?: string;
  };
  onDelete?: (id: string) => void;
}

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

/** Formats uptime seconds, including days when greater than 24h. */
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;
  if (days > 0) return `${days}d ${hours}h ${minutes}m ${secs}s`;
  return `${hours}h ${minutes}m ${secs}s`;
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
  details: BaseStationDetailsSnapshot | undefined,
): { latitude?: number | null; longitude?: number | null; altitude?: number | null } {
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
  details: BaseStationDetailsSnapshot | undefined,
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

const BaseStationDetails: React.FC<BaseStationDetailsProps> = ({
  baseStation,
  onDelete,
}) => {
  const navigate = useNavigate();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editFormData, setEditFormData] = useState({
    name: baseStation.name || "",
    eui: formatEUIWithDashes(baseStation.eui),
    latitude: "",
    longitude: "",
    altitude: "",
  });
  const [euiError, setEuiError] = useState<string | null>(null);
  const [locationErrors, setLocationErrors] = useState<Record<string, string>>(
    {},
  );
  const [mapPickerOpen, setMapPickerOpen] = useState(false);
  const [euiCopied, setEuiCopied] = useState(false);
  const [scUrlCopied, setScUrlCopied] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [editError, setEditError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Certificate regeneration state
  const [showRegenConfirm, setShowRegenConfirm] = useState(false);
  const [regenCertData, setRegenCertData] =
    useState<GenerateCertificateResponse | null>(null);
  const [isRegenerating, setIsRegenerating] = useState(false);

  // Handle EUI copy with feedback
  const handleCopyEui = async () => {
    const formattedEui = formatEUIWithDashes(baseStation.eui);
    await navigator.clipboard.writeText(formattedEui);
    setEuiCopied(true);
    setTimeout(() => setEuiCopied(false), TIMING_COPY_FEEDBACK);
  };

  // Handle SC URL copy with feedback
  const handleCopyScUrl = async () => {
    if (!effectiveServiceCenterUrl) return;
    await navigator.clipboard.writeText(effectiveServiceCenterUrl);
    setScUrlCopied(true);
    setTimeout(() => setScUrlCopied(false), TIMING_COPY_FEEDBACK);
  };

  // Delete mutation with cache invalidation
  const deleteBaseStationMutation = useDeleteBaseStation();

  // Update mutation for edit dialog
  const updateBaseStationMutation = useUpdateBaseStation();

  // EUI update mutation with navigation to new route
  const updateEuiMutation = useUpdateBaseStationEui();

  // Use React Query hooks for data fetching with smart refetch
  const {
    data: baseStationDetails,
    isLoading: loadingDetails,
    error: detailsError,
  } = useBaseStation(baseStation.eui);

  // Connect to realtime stream for this base station
  useEffect(() => {
    realtimeService.connectBaseStationStream(baseStation.eui);
    return () => realtimeService.disconnectBaseStationStream();
  }, [baseStation.eui]);

  // Reset certificate regeneration state when base station changes
  useEffect(() => {
    setRegenCertData(null);
    setShowRegenConfirm(false);
    setIsRegenerating(false);
  }, [baseStation.eui]);

  // Prefer regen-provided URL, then detail-fetched, then list-sourced
  const effectiveServiceCenterUrl =
    regenCertData?.serviceCenterUrl ||
    baseStationDetails?.serviceCenterUrl ||
    baseStation.serviceCenterUrl;

  // Certificate expiry calculation using centralized helper
  const daysUntilExpiry = calculateDaysUntilExpiry(
    baseStation.certificateExpiryDate,
  );
  const isExpiringSoon = daysUntilExpiry !== null && daysUntilExpiry <= 30;
  const isExpired = daysUntilExpiry !== null && daysUntilExpiry < 0;

  // Handle edit dialog open - initialize form data
  const handleEditDialogOpen = () => {
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
    setEditDialogOpen(true);
  };

  // Handle edit dialog close - reset all state
  const handleEditDialogClose = () => {
    setEditDialogOpen(false);
    setEuiError(null);
    setRegenCertData(null);
    setShowRegenConfirm(false);
    setIsRegenerating(false);
  };

  // Handle EUI field change with auto-formatting
  const handleEuiChange = (value: string) => {
    const formatted = formatEUIWithDashes(value);
    setEditFormData((prev) => ({ ...prev, eui: formatted }));
    if (euiError) setEuiError(null);
  };

  // Handle edit form save - update name and/or EUI
  /** Validate location fields; returns true if valid */
  const validateLocation = (): boolean => {
    const errs = validateLocationForm(
      editFormData,
      baseStationDetails?.locationSource === "gps",
    );
    setLocationErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const buildLocationData = () =>
    buildLocationUpdateData(editFormData, baseStationDetails);

  const locationHasChanges = () =>
    hasLocationChanged(editFormData, baseStationDetails);

  const handleUpdateSuccess = (newEui?: string) => {
    if (newEui) {
      setSuccessMessage(MSG_BS_EUI_UPDATED);
      handleEditDialogClose();
      navigate(`/base-stations/${formatEUIWithDashes(newEui)}`);
    } else {
      handleEditDialogClose();
    }
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

    // Validate EUI if changed
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

    // Validate location
    if (!validateLocation()) return;

    const locationData = buildLocationData();
    const locationChanged = locationHasChanges();

    // If EUI changed, update it first
    if (euiChanged) {
      updateEuiMutation.mutate(
        { eui: baseStation.eui, newEui: cleanedNewEui },
        {
          onSuccess: () => {
            // After EUI update, update name/location if changed
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
                  onSuccess: () => handleUpdateSuccess(cleanedNewEui),
                  onError: () => {
                    setEditError(ERR_UPDATE_BS_NAME_PARTIAL);
                  },
                },
              );
            } else {
              handleUpdateSuccess(cleanedNewEui);
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
          onSuccess: () => handleUpdateSuccess(),
          onError: handleUpdateError,
        },
      );
    } else {
      // No changes
      handleEditDialogClose();
    }
  };

  // Handle certificate regeneration confirmation cancel
  const handleRegenCancel = () => {
    setShowRegenConfirm(false);
    setIsRegenerating(false);
  };

  // Handle certificate regeneration
  const handleRegenerateCerts = async () => {
    setIsRegenerating(true);
    try {
      const response = await apiService.generateCertificate({
        bsEui: formatEUIWithDashes(baseStation.eui),
        validityDays: CERT_VALIDITY_DAYS.THREE_YEARS,
      });
      setRegenCertData(response);
      setShowRegenConfirm(false);
      setSuccessMessage(BASE_STATION_DETAILS.REGENERATE_CERTS_SUCCESS);
    } catch {
      setErrorMessage(BASE_STATION_DETAILS.REGENERATE_CERTS_ERROR);
    } finally {
      setIsRegenerating(false);
    }
  };

  // Handle regenerated certificate download
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
      setErrorMessage(BASE_STATION_DETAILS.CERTIFICATE_DOWNLOAD_ERROR);
    }
  };

  return (
    <Card sx={{ mt: 2, mb: 2 }}>
      <CardContent>
        <Box
          display="flex"
          justifyContent="flex-end"
          alignItems="center"
          mb={2}
        >
          <Box>
            <Tooltip title={ACTION_EDIT}>
              <IconButton size="small" onClick={handleEditDialogOpen}>
                <EditIcon />
              </IconButton>
            </Tooltip>
            <Tooltip title={ACTION_DELETE}>
              <IconButton
                size="small"
                color="error"
                onClick={() => setDeleteDialogOpen(true)}
              >
                <DeleteIcon />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>

        <Grid container spacing={BS_DETAIL_LAYOUT.GRID_SPACING}>
          {/* Left column: Map card */}
          <Grid size={{ xs: 12, md: 3 }}>
            <BaseStationLocationMap
              latitude={baseStationDetails?.latitude}
              longitude={baseStationDetails?.longitude}
              altitude={baseStationDetails?.altitude}
              locationSource={baseStationDetails?.locationSource}
            />
          </Grid>

          {/* Right column: Info sections */}
          <Grid size={{ xs: 12, md: 9 }}>
            <Box
              sx={{
                display: "flex",
                gap: 32,
                flexWrap: { xs: "wrap", md: "nowrap" },
              }}
            >
              <Box sx={{ minWidth: 0 }}>
                <Typography
                  variant="subtitle2"
                  color="text.secondary"
                  gutterBottom
                  sx={{ mb: 2 }}
                >
                  {BASE_STATION_DETAILS.BASIC_INFORMATION}
                </Typography>
                <Box mb={1}>
                  <Typography variant="body2" color="text.secondary">
                    {BASE_STATION_DETAILS.BASE_STATION_NAME}
                  </Typography>
                  <Typography variant="body1">
                    {baseStation.name || BASE_STATION_DETAILS.NOT_SET}
                  </Typography>
                </Box>
                <Box mb={1}>
                  <Typography variant="body2" color="text.secondary">
                    {BASE_STATION_DETAILS.BASE_STATION_EUI}
                  </Typography>
                  <Typography
                    variant="body1"
                    sx={(theme) => getMonoBody1(theme)}
                  >
                    {formatEUIWithDashes(baseStation.eui)}
                  </Typography>
                </Box>
                <Box mb={1}>
                  <Typography variant="body2" color="text.secondary">
                    {BASE_STATION_DETAILS.STATUS}
                  </Typography>
                  <Chip
                    label={baseStation.status}
                    color={
                      baseStation.status === "online" ? "success" : "default"
                    }
                    size="small"
                  />
                </Box>
              </Box>

              <Box sx={{ minWidth: 0 }}>
                <Typography
                  variant="subtitle2"
                  color="text.secondary"
                  gutterBottom
                  sx={{ mb: 2 }}
                >
                  {BASE_STATION_DETAILS.PERFORMANCE_METRICS}
                </Typography>
                {baseStationDetails ? (
                  <>
                    <Box mb={1}>
                      <Typography variant="body2" color="text.secondary">
                        {BASE_STATION_DETAILS.TEMPERATURE}
                      </Typography>
                      <Typography variant="body1">
                        {baseStationDetails.temperatureCelsius != null
                          ? `${baseStationDetails.temperatureCelsius.toFixed(1)}°C`
                          : BASE_STATION_DETAILS.NOT_AVAILABLE}
                      </Typography>
                    </Box>
                    <Box mb={1}>
                      <Typography variant="body2" color="text.secondary">
                        {BASE_STATION_DETAILS.CPU_LOAD}
                      </Typography>
                      <Typography variant="body1">
                        {baseStationDetails.cpuLoad != null
                          ? `${(baseStationDetails.cpuLoad * 100).toFixed(1)}%`
                          : BASE_STATION_DETAILS.NOT_AVAILABLE}
                      </Typography>
                    </Box>
                    <Box mb={1}>
                      <Typography variant="body2" color="text.secondary">
                        {BASE_STATION_DETAILS.MEMORY_LOAD}
                      </Typography>
                      <Typography variant="body1">
                        {baseStationDetails.memoryLoad != null
                          ? `${(baseStationDetails.memoryLoad * 100).toFixed(1)}%`
                          : BASE_STATION_DETAILS.NOT_AVAILABLE}
                      </Typography>
                    </Box>
                    <Box mb={1}>
                      <Typography variant="body2" color="text.secondary">
                        {BASE_STATION_DETAILS.DUTY_CYCLE}
                      </Typography>
                      <Typography variant="body1">
                        {baseStationDetails.dutyCycle != null
                          ? `${(baseStationDetails.dutyCycle * 100).toFixed(2)}%`
                          : BASE_STATION_DETAILS.NOT_AVAILABLE}
                      </Typography>
                    </Box>
                    <Box mb={1}>
                      <Typography variant="body2" color="text.secondary">
                        {BASE_STATION_DETAILS.BS_CONFIG}
                      </Typography>
                      <Typography
                        variant="body1"
                        sx={(theme) => getMonoBody1(theme)}
                      >
                        {baseStationDetails.bsConfig
                          ? truncateWithEllipsis(
                              JSON.stringify(
                                baseStationDetails.bsConfig,
                                null,
                                2,
                              ),
                              TRUNCATION.CONFIG_PREVIEW_LENGTH,
                              TRUNCATION.ELLIPSIS,
                            )
                          : BASE_STATION_DETAILS.NOT_AVAILABLE}
                      </Typography>
                    </Box>
                  </>
                ) : loadingDetails ? (
                  <Box display="flex" alignItems="center" gap={1}>
                    <CircularProgress size={16} />
                    <Typography variant="body2" color="text.secondary">
                      {LOADER.LOADING}
                    </Typography>
                  </Box>
                ) : detailsError ? (
                  <Alert severity="error" variant="outlined">
                    {detailsError instanceof Error
                      ? detailsError.message
                      : ERR_LOAD_BS_DETAILS}
                  </Alert>
                ) : null}
              </Box>

              <Box sx={{ minWidth: 0 }}>
                <Typography
                  variant="subtitle2"
                  color="text.secondary"
                  gutterBottom
                  sx={{ mb: 2 }}
                >
                  {BASE_STATION_DETAILS.SYSTEM_INFORMATION}
                </Typography>
                {baseStationDetails ? (
                  <>
                    <Box mb={1}>
                      <Typography variant="body2" color="text.secondary">
                        {BASE_STATION_DETAILS.SYSTEM_TIME}
                      </Typography>
                      <Typography variant="body1">
                        {baseStationDetails.systemTime
                          ? formatDateTime(
                              baseStationDetails.systemTime / 1000000,
                            )
                          : BASE_STATION_DETAILS.NOT_AVAILABLE}
                      </Typography>
                    </Box>
                    <Box mb={1}>
                      <Typography variant="body2" color="text.secondary">
                        {BASE_STATION_DETAILS.UPTIME}
                      </Typography>
                      <Typography variant="body1">
                        {baseStationDetails.uptimeSeconds
                          ? formatUptime(baseStationDetails.uptimeSeconds)
                          : BASE_STATION_DETAILS.NOT_AVAILABLE}
                      </Typography>
                    </Box>
                    <Box mb={1}>
                      <Typography variant="body2" color="text.secondary">
                        {BASE_STATION_DETAILS.LAST_STATUS_UPDATE}
                      </Typography>
                      <Typography variant="body1">
                        {baseStationDetails.lastStatusAt
                          ? formatDateTime(baseStationDetails.lastStatusAt)
                          : BASE_STATION_DETAILS.NOT_AVAILABLE}
                      </Typography>
                    </Box>
                    {baseStation.certificateExpiryDate && (
                      <Box mb={1}>
                        <Typography variant="body2" color="text.secondary">
                          {BASE_STATION_DETAILS.CERTIFICATE_EXPIRY}
                        </Typography>
                        <Typography
                          variant="body1"
                          color={
                            isExpired
                              ? "error.main"
                              : isExpiringSoon
                                ? "warning.main"
                                : "text.primary"
                          }
                        >
                          {formatDate(baseStation.certificateExpiryDate)}
                        </Typography>
                      </Box>
                    )}
                  </>
                ) : loadingDetails ? (
                  <Box display="flex" alignItems="center" gap={1}>
                    <CircularProgress size={16} />
                    <Typography variant="body2" color="text.secondary">
                      {LOADER.LOADING}
                    </Typography>
                  </Box>
                ) : detailsError ? (
                  <Alert severity="error" variant="outlined">
                    {detailsError instanceof Error
                      ? detailsError.message
                      : ERR_LOAD_BS_DETAILS}
                  </Alert>
                ) : null}
              </Box>
            </Box>
          </Grid>

          <Grid size={12}>
            <Divider sx={{ my: 1 }} />
          </Grid>

          {/* Base Station Messages Section */}
          <Grid size={12}>
            <BaseStationMessages
              bsEui={baseStation.eui}
              basestationName={
                baseStation.name || formatEUIWithDashes(baseStation.eui)
              }
            />
          </Grid>
        </Grid>
      </CardContent>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
      >
        <DialogTitle>{BASE_STATION_DETAILS.DIALOG_DELETE_TITLE}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {BASE_STATION_DETAILS.DIALOG_DELETE_CONFIRM_PREFIX} &quot;
            {baseStation.name ||
              formatEUIWithDashes(baseStation.eui)}&quot;?{" "}
            {BASE_STATION_DETAILS.DIALOG_DELETE_WARNING}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>
            {ACTION_CANCEL}
          </Button>
          <Button
            onClick={() => {
              deleteBaseStationMutation.mutate(baseStation.eui, {
                onSuccess: () => {
                  setDeleteDialogOpen(false);
                  if (onDelete) {
                    onDelete(baseStation.id);
                  }
                },
                onError: () => {
                  setErrorMessage(ERR_DELETE_BASE_STATION);
                },
              });
            }}
            color="error"
            variant="contained"
            disabled={deleteBaseStationMutation.isPending}
          >
            {deleteBaseStationMutation.isPending
              ? BASE_STATION_DETAILS.DELETING
              : BASE_STATION_DETAILS.DELETE}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Base Station Dialog */}
      <Dialog
        open={editDialogOpen}
        onClose={handleEditDialogClose}
        maxWidth="sm"
        fullWidth
      >
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
              <TextField
                label={BASE_STATION_DETAILS.LABEL_SC_URL}
                value={effectiveServiceCenterUrl}
                fullWidth
                slotProps={{
                  input: {
                    readOnly: true,
                    endAdornment: (
                      <InputAdornment position="end">
                        <Tooltip
                          title={
                            scUrlCopied
                              ? BASE_STATION_DETAILS.LABEL_SC_URL_COPIED
                              : BASE_STATION_DETAILS.ACTION_COPY_SC_URL
                          }
                        >
                          <IconButton
                            size="small"
                            onClick={handleCopyScUrl}
                            edge="end"
                          >
                            {scUrlCopied ? (
                              <CheckCircleIcon
                                fontSize="small"
                                color="success"
                              />
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
            )}

            {/* Certificates section */}
            <Divider sx={{ mb: 2 }} />
            <Typography variant="subtitle2" color="text.secondary" gutterBottom>
              {BASE_STATION_DETAILS.CERTIFICATES_SECTION_TITLE}
            </Typography>

            {regenCertData ? (
              // After regeneration: show download buttons
              <Box>
                <Alert severity="success" sx={{ mb: 2 }}>
                  {BASE_STATION_DETAILS.REGENERATE_CERTS_SUCCESS}
                </Alert>
                {effectiveServiceCenterUrl && (
                  <TextField
                    label={BASE_STATION_DETAILS.LABEL_SC_URL}
                    value={effectiveServiceCenterUrl}
                    fullWidth
                    slotProps={{
                      input: {
                        readOnly: true,
                        endAdornment: (
                          <InputAdornment position="end">
                            <Tooltip
                              title={
                                scUrlCopied
                                  ? BASE_STATION_DETAILS.LABEL_SC_URL_COPIED
                                  : BASE_STATION_DETAILS.ACTION_COPY_SC_URL
                              }
                            >
                              <IconButton
                                size="small"
                                onClick={handleCopyScUrl}
                                edge="end"
                              >
                                {scUrlCopied ? (
                                  <CheckCircleIcon
                                    fontSize="small"
                                    color="success"
                                  />
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
                    sx={{ mb: 2 }}
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
              // Before regeneration: show regenerate button
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
          <Button onClick={handleEditDialogClose}>
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

      {/* Certificate Regeneration Confirmation Dialog */}
      <Dialog open={showRegenConfirm} onClose={handleRegenCancel}>
        <DialogTitle>
          {BASE_STATION_DETAILS.REGENERATE_CERTS_CONFIRM_TITLE}
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {BASE_STATION_DETAILS.REGENERATE_CERTS_CONFIRM_TEXT}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleRegenCancel}>{ACTION_CANCEL}</Button>
          <Button
            onClick={handleRegenerateCerts}
            color="warning"
            disabled={isRegenerating}
          >
            {BASE_STATION_DETAILS.ACTION_REGENERATE}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Error Snackbar */}
      <Snackbar
        open={!!errorMessage}
        autoHideDuration={6000}
        onClose={() => setErrorMessage(null)}
        anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
      >
        <Alert severity="error" onClose={() => setErrorMessage(null)}>
          {errorMessage}
        </Alert>
      </Snackbar>

      {/* Success Snackbar */}
      <Snackbar
        open={!!successMessage}
        autoHideDuration={4000}
        onClose={() => setSuccessMessage(null)}
        anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
      >
        <Alert severity="success" onClose={() => setSuccessMessage(null)}>
          {successMessage}
        </Alert>
      </Snackbar>
    </Card>
  );
};

export default BaseStationDetails;
