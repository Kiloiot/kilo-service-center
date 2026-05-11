import React, { useState } from "react";

import { ApiError } from "@api-types/api";
import {
  useCommissionBaseStationWithCerts,
  useRetryCertificateGeneration,
} from "@hooks";
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  IconButton,
  Paper,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";

import { apiService } from "@services/api";
import { formatEUIWithDashes, isValidEUI } from "@utils/formatters";
import { getMonoBody2 } from "@utils/typography";
import type { CertificateDownloadType } from "@constants/app";
import {
  CERTIFICATE_DOWNLOAD_TYPES,
  GEO_BOUNDS,
  TIMING_COPY_FEEDBACK,
} from "@constants/app";
import {
  ACTION_CANCEL,
  ACTION_CONTINUE,
  ACTION_COPIED,
  ACTION_COPY,
  ACTION_CREATING,
  ACTION_DOWNLOAD_CA_CERT,
  ACTION_DOWNLOAD_TLS_CERT,
  ACTION_DOWNLOAD_TLS_KEY,
  ACTION_NEXT,
  ACTION_PICK_ON_MAP,
  ACTION_RETRY_CERTS,
  ACTION_RETRYING_CERTS,
  ERR_BS_CREATION_AFTER_CERTS,
  ERR_CERT_GENERATION_FAILED_RETRY,
  ERR_CERT_NETWORK,
  ERR_CERT_SERVICE_UNAVAILABLE,
  ERR_CREATE_BASE_STATION,
  HELPER_ALTITUDE,
  HELPER_BS_EUI,
  HELPER_BS_NAME,
  HELPER_LATITUDE,
  HELPER_LONGITUDE,
  INFO_CERT_EXPIRY,
  INFO_NOTE_PREFIX,
  INFO_TLS_NOTE,
  INSTR_BS_CREATED_FALLBACK,
  INSTR_STEP_1,
  INSTR_STEP_2_HEADER,
  INSTR_STEP_3,
  INSTR_TLS_NOTE,
  LABEL_ALTITUDE,
  LABEL_BS_EUI,
  LABEL_BS_EUI_DISPLAY,
  LABEL_BS_NAME,
  LABEL_CA_CERT_FILE,
  LABEL_CLIENT_CERT_PREFIX,
  LABEL_CLIENT_CERT_SUFFIX,
  LABEL_LATITUDE,
  LABEL_LOCATION_OPTIONAL,
  LABEL_LONGITUDE,
  LABEL_NAME,
  LABEL_PRIVATE_KEY_PREFIX,
  LABEL_PRIVATE_KEY_SUFFIX,
  LABEL_SC_URL_DISPLAY,
  MSG_BS_ADDED_NO_CERTS,
  MSG_BS_ADDED_WITH_CERTS,
  MSG_BS_PARTIAL_SUCCESS,
  PLACEHOLDER_BS_EUI,
  SECTION_BS_CONFIG,
  SECTION_DOWNLOAD_CERTS,
  SECTION_NEXT_STEPS,
  TITLE_ADD_BS,
  TITLE_BS_PARTIAL,
  TITLE_BS_READY,
  VAL_BS_EUI_FORMAT,
  VAL_BS_NAME_REQUIRED,
  VAL_LAT_LON_PAIR,
  VAL_LATITUDE_RANGE,
  VAL_LONGITUDE_RANGE,
} from "@constants/messages";
import {
  CheckCircleIcon,
  CloseIcon,
  ContentCopyIcon,
  DownloadIcon,
  MapIcon,
  SecurityIcon,
} from "@theme/icons";

import MapPickerDialog from "./MapPickerDialog";

interface BaseStationCommissioningDialogProps {
  open: boolean;
  onClose: () => void;
}

interface CertificateData {
  bsEui: string;
  serviceCenterUrl: string;
  downloadUrls: {
    caCert: string;
    clientCert: string;
    privateKey: string;
  };
  expiresAt: string;
}

/**
 * BaseStationCommissioningDialog - 3-Step Commissioning Wizard
 *
 * Step 1 (Input): User enters Base Station Name and EUI
 * Step 2 (Partial): BS created but cert generation failed - offers retry
 * Step 3 (Success): Displays Service Center address and certificate downloads
 *
 * Flow (BS-first for cert persistence):
 * 1. User enters Name + EUI
 * 2. Click "Next" → validates input
 * 3. Create BS record FIRST (certs must be persisted to existing BS)
 * 4. Generate TLS certificates SECOND (persisted server-side to BS record)
 * 5. Display SC address + certificate download links
 * 6. User downloads certs, copies SC address to configure base station hardware
 * 7. Click "Continue" → closes dialog, list auto-refreshes via cache invalidation
 *
 * Error handling:
 * - BS creation failure: User can retry with same inputs
 * - Cert generation failure after BS: Shows partial success with retry button
 *   (retrying only regenerates certs, doesn't create duplicate BS)
 */
export default function BaseStationCommissioningDialog({
  open,
  onClose,
}: BaseStationCommissioningDialogProps) {
  // Form state
  const [formData, setFormData] = useState({
    name: "",
    eui: "",
    latitude: "",
    longitude: "",
    altitude: "",
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [mapPickerOpen, setMapPickerOpen] = useState(false);

  // Wizard state: input → partial (BS created, certs failed) → success
  const [phase, setPhase] = useState<"input" | "partial" | "success">("input");
  const [loading, setLoading] = useState(false);
  const [certificateData, setCertificateData] =
    useState<CertificateData | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);
  const [retryToken, setRetryToken] = useState<string | null>(null);

  // Commission hook with BS-first flow for cert persistence
  const commissionMutation = useCommissionBaseStationWithCerts();
  const retryCertsMutation = useRetryCertificateGeneration();

  const handleEUIChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const formatted = formatEUIWithDashes(e.target.value);
    setFormData({ ...formData, eui: formatted });
    if (errors.eui) {
      setErrors({ ...errors, eui: "" });
    }
  };

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, name: e.target.value });
    if (errors.name) {
      setErrors({ ...errors, name: "" });
    }
  };

  /**
   * Validate form fields
   */
  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = VAL_BS_NAME_REQUIRED;
    }

    // Validate EUI format (8-byte, 16 hex chars)
    if (!isValidEUI(formData.eui)) {
      newErrors.eui = VAL_BS_EUI_FORMAT;
    }

    // Validate lat/lon pair: both present or both absent
    const hasLat = formData.latitude.trim() !== "";
    const hasLng = formData.longitude.trim() !== "";
    if (hasLat !== hasLng) {
      newErrors.latitude = VAL_LAT_LON_PAIR;
      newErrors.longitude = VAL_LAT_LON_PAIR;
    }

    if (hasLat) {
      const lat = parseFloat(formData.latitude);
      if (
        isNaN(lat) ||
        lat < GEO_BOUNDS.LATITUDE_MIN ||
        lat > GEO_BOUNDS.LATITUDE_MAX
      ) {
        newErrors.latitude = VAL_LATITUDE_RANGE;
      }
    }

    if (hasLng) {
      const lng = parseFloat(formData.longitude);
      if (
        isNaN(lng) ||
        lng < GEO_BOUNDS.LONGITUDE_MIN ||
        lng > GEO_BOUNDS.LONGITUDE_MAX
      ) {
        newErrors.longitude = VAL_LONGITUDE_RANGE;
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  /**
   * Handle "Next" button - creates BS first, then generates certs (for cert persistence)
   */
  const handleNext = async () => {
    if (!validateForm()) return;

    setLoading(true);
    setErrors({});

    try {
      const mutationData: {
        eui: string;
        name: string;
        latitude?: number;
        longitude?: number;
        altitude?: number;
      } = {
        eui: formData.eui,
        name: formData.name.trim(),
      };

      if (formData.latitude.trim() && formData.longitude.trim()) {
        mutationData.latitude = parseFloat(formData.latitude);
        mutationData.longitude = parseFloat(formData.longitude);
        if (formData.altitude.trim()) {
          mutationData.altitude = parseFloat(formData.altitude);
        }
      }

      const result = await commissionMutation.mutateAsync(mutationData);

      if (result.status === "complete" && result.certData) {
        // Full success - BS created and certs generated
        setCertificateData({
          bsEui: result.bsEui,
          serviceCenterUrl: result.certData.serviceCenterUrl,
          downloadUrls: result.certData.downloadUrls,
          expiresAt: result.certData.expiryDate || "",
        });
        setPhase("success");
      } else if (result.status === "partial") {
        // Partial success - BS created but cert generation failed
        setRetryToken(result.retryToken || result.bsEui);
        setPhase("partial");
      }
    } catch (err) {
      // Map errors using status codes (robust) instead of message fragments (brittle)
      if (err instanceof TypeError && err.message === "Failed to fetch") {
        setErrors({ general: ERR_CERT_NETWORK });
      } else if (err instanceof ApiError) {
        // Use HTTP status codes for error classification
        if (err.status === 409) {
          // 409 Conflict = duplicate EUI (BS already exists)
          setErrors({ general: ERR_BS_CREATION_AFTER_CERTS });
        } else if (err.status === 503 || err.isServerError()) {
          // 503 Service Unavailable or 5xx = cert service unavailable
          setErrors({ general: ERR_CERT_SERVICE_UNAVAILABLE });
        } else if (err.status === 400) {
          // 400 Bad Request = validation error
          setErrors({ general: err.message || ERR_CREATE_BASE_STATION });
        } else {
          setErrors({ general: err.message || ERR_CREATE_BASE_STATION });
        }
      } else if (err instanceof Error) {
        // Catch-all for non-HTTP errors (network, parsing)
        setErrors({ general: err.message || ERR_CREATE_BASE_STATION });
      } else {
        setErrors({ general: ERR_CREATE_BASE_STATION });
      }
    } finally {
      setLoading(false);
    }
  };

  /**
   * Handle retry certificate generation for partial success scenario
   */
  const handleRetryCerts = async () => {
    if (!retryToken) return;

    setLoading(true);
    setErrors({});

    try {
      const result = await retryCertsMutation.mutateAsync({
        bsEui: retryToken,
      });

      setCertificateData({
        bsEui: result.bsEui,
        serviceCenterUrl: result.serviceCenterUrl,
        downloadUrls: result.downloadUrls,
        expiresAt: result.expiryDate || "",
      });
      setRetryToken(null);
      setPhase("success");
    } catch (err) {
      if (err instanceof TypeError && err.message === "Failed to fetch") {
        setErrors({ general: ERR_CERT_NETWORK });
      } else if (err instanceof ApiError) {
        if (err.status === 503 || err.isServerError()) {
          setErrors({ general: ERR_CERT_SERVICE_UNAVAILABLE });
        } else {
          setErrors({ general: ERR_CERT_GENERATION_FAILED_RETRY });
        }
      } else {
        setErrors({ general: ERR_CERT_GENERATION_FAILED_RETRY });
      }
    } finally {
      setLoading(false);
    }
  };

  /**
   * Copy text to clipboard with visual feedback
   */
  const handleCopy = (text: string, field: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), TIMING_COPY_FEEDBACK);
  };

  /**
   * Reset form and close dialog
   * Note: List refresh is handled by hook's cache invalidation on success
   */
  const handleClose = () => {
    // Reset state
    setFormData({
      name: "",
      eui: "",
      latitude: "",
      longitude: "",
      altitude: "",
    });
    setErrors({});
    setPhase("input");
    setLoading(false);
    setCertificateData(null);
    setCopiedField(null);
    setRetryToken(null);

    onClose();
  };

  /**
   * Handle certificate download via gRPC
   */
  const handleDownloadCertificate = async (
    certType: CertificateDownloadType,
  ) => {
    try {
      // Use the caCert ID for all downloads (they share the same cert bundle ID)
      const certId = certificateData?.downloadUrls.caCert;
      if (!certId) return;

      const { blob, filename } = await apiService.downloadCertificate(
        certId,
        certType,
      );

      // Trigger browser download
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch {
      setErrors({ general: ERR_CERT_NETWORK });
    }
  };

  /**
   * Render Step 1 - Input Form
   */
  const renderInputPhase = () => (
    <Box sx={{ pt: 2 }}>
      <TextField
        fullWidth
        label={LABEL_BS_NAME}
        value={formData.name}
        onChange={handleNameChange}
        margin="normal"
        required
        helperText={errors.name || HELPER_BS_NAME}
        error={!!errors.name}
        disabled={loading}
      />

      <TextField
        fullWidth
        label={LABEL_BS_EUI}
        value={formData.eui}
        onChange={handleEUIChange}
        margin="normal"
        required
        placeholder={PLACEHOLDER_BS_EUI}
        helperText={errors.eui || HELPER_BS_EUI}
        error={!!errors.eui}
        disabled={loading}
      />

      {/* Location (optional) */}
      <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>
        {LABEL_LOCATION_OPTIONAL}
      </Typography>
      <Box sx={{ display: "flex", gap: 2 }}>
        <TextField
          label={LABEL_LATITUDE}
          value={formData.latitude}
          onChange={(e) => {
            setFormData({ ...formData, latitude: e.target.value });
            if (errors.latitude) setErrors({ ...errors, latitude: "" });
          }}
          type="number"
          helperText={errors.latitude || HELPER_LATITUDE}
          error={!!errors.latitude}
          disabled={loading}
          sx={{ flex: 1 }}
        />
        <TextField
          label={LABEL_LONGITUDE}
          value={formData.longitude}
          onChange={(e) => {
            setFormData({ ...formData, longitude: e.target.value });
            if (errors.longitude) setErrors({ ...errors, longitude: "" });
          }}
          type="number"
          helperText={errors.longitude || HELPER_LONGITUDE}
          error={!!errors.longitude}
          disabled={loading}
          sx={{ flex: 1 }}
        />
      </Box>
      <Box sx={{ display: "flex", gap: 2, mt: 1 }}>
        <TextField
          label={LABEL_ALTITUDE}
          value={formData.altitude}
          onChange={(e) =>
            setFormData({ ...formData, altitude: e.target.value })
          }
          type="number"
          helperText={HELPER_ALTITUDE}
          disabled={loading}
          sx={{ flex: 1 }}
        />
        <Box sx={{ flex: 1, display: "flex", alignItems: "center" }}>
          <Button
            variant="outlined"
            startIcon={<MapIcon />}
            onClick={() => setMapPickerOpen(true)}
            disabled={loading}
          >
            {ACTION_PICK_ON_MAP}
          </Button>
        </Box>
      </Box>

      <MapPickerDialog
        open={mapPickerOpen}
        onClose={() => setMapPickerOpen(false)}
        onConfirm={(lat, lng) => {
          setFormData({
            ...formData,
            latitude: lat.toFixed(6),
            longitude: lng.toFixed(6),
          });
          setMapPickerOpen(false);
        }}
        initialLat={
          formData.latitude ? parseFloat(formData.latitude) : undefined
        }
        initialLng={
          formData.longitude ? parseFloat(formData.longitude) : undefined
        }
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

  /**
   * Render Step 2 - Partial Success (BS created, certs failed)
   */
  const renderPartialPhase = () => (
    <Box sx={{ pt: 2 }}>
      <Alert severity="warning" sx={{ mb: 3 }}>
        {MSG_BS_PARTIAL_SUCCESS}
      </Alert>

      {/* Configuration Summary */}
      <Paper variant="outlined" sx={{ p: 2, mb: 3 }}>
        <Typography variant="subtitle2" gutterBottom>
          {SECTION_BS_CONFIG}
        </Typography>

        <Box sx={{ mt: 2 }}>
          {/* Name */}
          <Box
            display="flex"
            alignItems="center"
            justifyContent="space-between"
            sx={{ mb: 1 }}
          >
            <Typography variant="body2" color="text.secondary">
              {LABEL_NAME}
            </Typography>
            <Typography variant="body2">{formData.name}</Typography>
          </Box>

          {/* EUI with copy */}
          <Box
            display="flex"
            alignItems="center"
            justifyContent="space-between"
          >
            <Typography variant="body2" color="text.secondary">
              {LABEL_BS_EUI_DISPLAY}
            </Typography>
            <Box display="flex" alignItems="center" gap={1}>
              <Typography variant="body2" sx={(theme) => getMonoBody2(theme)}>
                {formData.eui}
              </Typography>
              <Tooltip
                title={copiedField === "eui" ? ACTION_COPIED : ACTION_COPY}
              >
                <IconButton
                  size="small"
                  onClick={() => handleCopy(formData.eui, "eui")}
                >
                  {copiedField === "eui" ? (
                    <CheckCircleIcon fontSize="small" color="success" />
                  ) : (
                    <ContentCopyIcon fontSize="small" />
                  )}
                </IconButton>
              </Tooltip>
            </Box>
          </Box>
        </Box>
      </Paper>

      {errors.general && (
        <Alert severity="error" sx={{ mt: 2 }}>
          {errors.general}
        </Alert>
      )}
    </Box>
  );

  /**
   * Render Step 3 - Success / Handoff
   */
  const renderSuccessPhase = () => (
    <Box sx={{ pt: 2 }}>
      <Alert severity="success" sx={{ mb: 3 }}>
        {certificateData ? MSG_BS_ADDED_WITH_CERTS : MSG_BS_ADDED_NO_CERTS}
      </Alert>

      {/* Configuration Summary */}
      <Paper variant="outlined" sx={{ p: 2, mb: 3 }}>
        <Typography variant="subtitle2" gutterBottom>
          {SECTION_BS_CONFIG}
        </Typography>

        <Box sx={{ mt: 2 }}>
          {/* Name */}
          <Box
            display="flex"
            alignItems="center"
            justifyContent="space-between"
            sx={{ mb: 1 }}
          >
            <Typography variant="body2" color="text.secondary">
              {LABEL_NAME}
            </Typography>
            <Typography variant="body2">{formData.name}</Typography>
          </Box>

          {/* EUI with copy */}
          <Box
            display="flex"
            alignItems="center"
            justifyContent="space-between"
            sx={{ mb: 1 }}
          >
            <Typography variant="body2" color="text.secondary">
              {LABEL_BS_EUI_DISPLAY}
            </Typography>
            <Box display="flex" alignItems="center" gap={1}>
              <Typography variant="body2" sx={(theme) => getMonoBody2(theme)}>
                {formData.eui}
              </Typography>
              <Tooltip
                title={copiedField === "eui" ? ACTION_COPIED : ACTION_COPY}
              >
                <IconButton
                  size="small"
                  onClick={() => handleCopy(formData.eui, "eui")}
                >
                  {copiedField === "eui" ? (
                    <CheckCircleIcon fontSize="small" color="success" />
                  ) : (
                    <ContentCopyIcon fontSize="small" />
                  )}
                </IconButton>
              </Tooltip>
            </Box>
          </Box>

          {/* Service Center URL with copy */}
          <Box
            display="flex"
            alignItems="center"
            justifyContent="space-between"
          >
            <Typography variant="body2" color="text.secondary">
              {LABEL_SC_URL_DISPLAY}
            </Typography>
            <Box display="flex" alignItems="center" gap={1}>
              <Typography variant="body2" sx={(theme) => getMonoBody2(theme)}>
                {certificateData?.serviceCenterUrl || "—"}
              </Typography>
              {certificateData?.serviceCenterUrl && (
                <Tooltip
                  title={copiedField === "url" ? ACTION_COPIED : ACTION_COPY}
                >
                  <IconButton
                    size="small"
                    onClick={() =>
                      handleCopy(certificateData.serviceCenterUrl, "url")
                    }
                  >
                    {copiedField === "url" ? (
                      <CheckCircleIcon fontSize="small" color="success" />
                    ) : (
                      <ContentCopyIcon fontSize="small" />
                    )}
                  </IconButton>
                </Tooltip>
              )}
            </Box>
          </Box>
        </Box>
      </Paper>

      {/* Certificate Downloads */}
      {certificateData && (
        <Paper variant="outlined" sx={{ p: 2, mb: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            {SECTION_DOWNLOAD_CERTS}
          </Typography>
          <Typography variant="caption" color="text.secondary" gutterBottom>
            {INFO_CERT_EXPIRY}
          </Typography>

          <Box sx={{ mt: 2, display: "flex", flexDirection: "column", gap: 1 }}>
            <Button
              onClick={() =>
                handleDownloadCertificate(CERTIFICATE_DOWNLOAD_TYPES.CA)
              }
              startIcon={<DownloadIcon />}
              variant="outlined"
              size="small"
            >
              {ACTION_DOWNLOAD_CA_CERT}
            </Button>
            <Button
              onClick={() =>
                handleDownloadCertificate(CERTIFICATE_DOWNLOAD_TYPES.CLIENT)
              }
              startIcon={<DownloadIcon />}
              variant="outlined"
              size="small"
            >
              {ACTION_DOWNLOAD_TLS_CERT}
            </Button>
            <Button
              onClick={() =>
                handleDownloadCertificate(CERTIFICATE_DOWNLOAD_TYPES.KEY)
              }
              startIcon={<DownloadIcon />}
              variant="outlined"
              size="small"
              color="warning"
            >
              {ACTION_DOWNLOAD_TLS_KEY}
            </Button>
          </Box>
        </Paper>
      )}

      <Divider sx={{ my: 3 }} />

      {/* Next Steps */}
      <Typography variant="subtitle2" gutterBottom>
        {SECTION_NEXT_STEPS}
      </Typography>
      <Box sx={{ mt: 1 }}>
        {certificateData ? (
          <>
            <Typography variant="body2" paragraph>
              {INSTR_STEP_1}
            </Typography>
            <Typography variant="body2" paragraph>
              {INSTR_STEP_2_HEADER}
              <br />
              &nbsp;&nbsp;• {LABEL_SC_URL_DISPLAY}{" "}
              {certificateData.serviceCenterUrl}
              <br />
              &nbsp;&nbsp;• {LABEL_CA_CERT_FILE}
              <br />
              &nbsp;&nbsp;• {LABEL_CLIENT_CERT_PREFIX}
              {certificateData.bsEui}
              {LABEL_CLIENT_CERT_SUFFIX}
              <br />
              &nbsp;&nbsp;• {LABEL_PRIVATE_KEY_PREFIX}
              {certificateData.bsEui}
              {LABEL_PRIVATE_KEY_SUFFIX}
            </Typography>
            <Typography variant="body2" paragraph>
              {INSTR_STEP_3}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {INSTR_TLS_NOTE}
            </Typography>
          </>
        ) : (
          <Typography variant="body2" color="text.secondary">
            {INSTR_BS_CREATED_FALLBACK}
          </Typography>
        )}
      </Box>
    </Box>
  );

  return (
    <Dialog
      open={open}
      onClose={handleClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: { bgcolor: "background.paper" },
      }}
    >
      <DialogTitle>
        <Box display="flex" alignItems="center" justifyContent="space-between">
          <Box display="flex" alignItems="center" gap={1}>
            <SecurityIcon color={phase === "partial" ? "warning" : "primary"} />
            <Typography variant="h6">
              {phase === "input" && TITLE_ADD_BS}
              {phase === "partial" && TITLE_BS_PARTIAL}
              {phase === "success" && TITLE_BS_READY}
            </Typography>
          </Box>
          <IconButton onClick={handleClose} size="small">
            <CloseIcon />
          </IconButton>
        </Box>
      </DialogTitle>

      <DialogContent>
        {phase === "input" && renderInputPhase()}
        {phase === "partial" && renderPartialPhase()}
        {phase === "success" && renderSuccessPhase()}
      </DialogContent>

      <DialogActions>
        {phase === "input" && (
          <>
            <Button onClick={handleClose} disabled={loading}>
              {ACTION_CANCEL}
            </Button>
            <Button
              onClick={handleNext}
              variant="contained"
              disabled={loading || !formData.eui || !formData.name.trim()}
              startIcon={
                loading ? <CircularProgress size={20} /> : <SecurityIcon />
              }
            >
              {loading ? ACTION_CREATING : ACTION_NEXT}
            </Button>
          </>
        )}
        {phase === "partial" && (
          <>
            <Button onClick={handleClose} disabled={loading}>
              {ACTION_CONTINUE}
            </Button>
            <Button
              onClick={handleRetryCerts}
              variant="contained"
              color="warning"
              disabled={loading}
              startIcon={
                loading ? <CircularProgress size={20} /> : <SecurityIcon />
              }
            >
              {loading ? ACTION_RETRYING_CERTS : ACTION_RETRY_CERTS}
            </Button>
          </>
        )}
        {phase === "success" && (
          <Button onClick={handleClose} variant="contained">
            {ACTION_CONTINUE}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}
