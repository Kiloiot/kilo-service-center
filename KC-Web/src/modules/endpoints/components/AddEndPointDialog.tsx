import React, { useEffect, useState } from "react";

import type { CreateEndpointRequest } from "@api-types/api";
import { useCreateEndpoint } from "@hooks";
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  InputAdornment,
  TextField,
  Typography,
} from "@mui/material";
import Grid from "@mui/material/Grid";

import {
  formatEUIWithDashes,
  generateRandomKey,
  validateEui,
  validateHexKey,
  validateShortAddr,
  validateTypeEui,
  validateUint32Counter,
} from "@utils/formatters";
import {
  MIOTY_EUI_REGEX,
  MIOTY_KEY_BYTE_LENGTH,
  MIOTY_UINT32_MAX,
} from "@constants/app";
import { ENDPOINT_FORM } from "@constants/messages";
import { CheckCircleIcon, InfoIcon } from "@theme/icons";

import {
  AdvancedMiotySettings,
  CommunicationSettings,
  CounterFields,
  SecurityKeyFields,
} from "./EndpointFormFields";
import DeviceModelSelector from "./DeviceModelSelector";

interface AddEndPointDialogProps {
  open: boolean;
  onClose: () => void;
}

interface AddEndpointFormData {
  eui: string;
  name: string;
  shortAddr: string;
  bidirectional: boolean;
  preAttach: boolean;
  carrierOffset: string;
  networkKey: string;
  applicationKey: string;
  typeEui: string;
  dualChan: boolean;
  repetition: boolean;
  wideCarrOff: boolean;
  longBlkDist: boolean;
  lastPacketCnt: string;
  attachCnt: string;
  deviceModelId: string;
}

/**
 * Validates the AddEndPointDialog form per BSSCI §3.8.1 / SCACI §3.6.1.
 * Returns a sparse errors map keyed by form field name.
 */
function validateAddEndpointForm(
  formData: AddEndpointFormData,
): Record<string, string> {
  const errors: Record<string, string> = {};

  const euiErr = validateEui(formData.eui);
  if (euiErr) errors.eui = euiErr;

  if (!formData.name) errors.name = ENDPOINT_FORM.ERROR_NAME_REQUIRED;

  if (!formData.networkKey) {
    errors.networkKey = ENDPOINT_FORM.ERROR_NETWORK_KEY_REQUIRED;
  } else {
    const keyErr = validateHexKey(formData.networkKey, true);
    if (keyErr) errors.networkKey = keyErr;
  }

  const shAddrErr = validateShortAddr(formData.shortAddr);
  if (shAddrErr) errors.shortAddr = shAddrErr;

  if (formData.lastPacketCnt === "") {
    errors.lastPacketCnt = ENDPOINT_FORM.ERROR_LAST_PACKET_CNT_REQUIRED;
  } else {
    const pktErr = validateUint32Counter(
      formData.lastPacketCnt,
      "lastPacketCnt",
    );
    if (pktErr) errors.lastPacketCnt = pktErr;
  }

  if (formData.attachCnt === "") {
    errors.attachCnt = ENDPOINT_FORM.ERROR_ATTACH_CNT_REQUIRED;
  } else {
    const attErr = validateUint32Counter(formData.attachCnt, "attachCnt");
    if (attErr) errors.attachCnt = attErr;
  }

  if (formData.applicationKey) {
    const appKeyErr = validateHexKey(formData.applicationKey, false);
    if (appKeyErr) errors.applicationKey = appKeyErr;
  }

  const typeEuiErr = validateTypeEui(formData.typeEui);
  if (typeEuiErr) errors.typeEui = typeEuiErr;

  return errors;
}

/**
 * Builds the CreateEndpointRequest from validated form data. All MIOTY
 * protocol fields are required (BSSCI §3.8.1 / SCACI §3.6.1) and surfaced
 * with explicit boolean values for radio flags; optional fields are
 * omitted when blank.
 */
function buildCreateEndpointPayload(
  formData: AddEndpointFormData,
): CreateEndpointRequest {
  return {
    epEui: formData.eui.replace(/-/g, ""),
    name: formData.name,
    shAddr: parseInt(formData.shortAddr, 16),
    nwkSnKey: formData.networkKey,
    bidi: formData.bidirectional,
    preAttach: formData.preAttach,
    dualChan: formData.dualChan,
    repetition: formData.repetition,
    wideCarrOff: formData.wideCarrOff,
    longBlkDist: formData.longBlkDist,
    lastPacketCnt: parseInt(formData.lastPacketCnt, 10),
    attachCnt: parseInt(formData.attachCnt, 10),
    appKey: formData.applicationKey || undefined,
    typeEui: formData.typeEui || undefined,
    carrierOffset:
      formData.carrierOffset !== ""
        ? parseInt(formData.carrierOffset, 10)
        : undefined,
    deviceModelId: formData.deviceModelId || undefined,
  };
}

const AddEndPointDialog: React.FC<AddEndPointDialogProps> = ({
  open,
  onClose,
}) => {
  // Use the mutation hook which properly invalidates the cache on success
  const createEndpointMutation = useCreateEndpoint();
  const [formData, setFormData] = useState({
    eui: "",
    name: "",
    shortAddr: "",
    bidirectional: false,
    preAttach: false, // Pre-Attachment for immediate propagation
    carrierOffset: "", // Optional - empty = omit from request
    networkKey: "",
    applicationKey: "",
    typeEui: "",
    // Additional MIOTY fields per BSSCI v1.0.0
    dualChan: false, // Dual channel mode
    repetition: false, // True if End Point uses DL repetition per BSSCI v1.0.0 Section 3.8.1
    wideCarrOff: false, // Wide carrier offset
    longBlkDist: false, // Long DL interblock distance
    lastPacketCnt: "", // Last known packet counter (BSSCI §3.8.1) - empty = not set
    attachCnt: "", // Attachment counter (SCACI §3.6.1) - empty = not set
    deviceModelId: "",
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [euiValid, setEuiValid] = useState(false);

  useEffect(() => {
    const euiWithoutDashes = formData.eui.replace(/-/g, "");
    setEuiValid(!!euiWithoutDashes && MIOTY_EUI_REGEX.test(euiWithoutDashes));
  }, [formData.eui]);

  const handleChange =
    (field: string) =>
    (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      const target = event.target as HTMLInputElement;
      const value = target.type === "checkbox" ? target.checked : target.value;
      setFormData({ ...formData, [field]: value });
      setErrors({ ...errors, [field]: "" });
    };

  const handleEuiChange = (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => {
    const formatted = formatEUIWithDashes(event.target.value);
    setFormData({ ...formData, eui: formatted });
    setErrors({ ...errors, eui: "" });
  };

  const handleGenerateNetworkKey = () => {
    setFormData({
      ...formData,
      networkKey: generateRandomKey(MIOTY_KEY_BYTE_LENGTH),
    });
  };

  const handleGenerateApplicationKey = () => {
    setFormData({
      ...formData,
      applicationKey: generateRandomKey(MIOTY_KEY_BYTE_LENGTH),
    });
  };

  const handleSubmit = async () => {
    const newErrors = validateAddEndpointForm(formData);
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    const createRequest = buildCreateEndpointPayload(formData);

    createEndpointMutation.mutate(createRequest, {
      onSuccess: () => {
        // Cache is automatically invalidated by the mutation hook
        handleClose();
      },
      onError: (error) => {
        if (error instanceof Error) {
          setErrors({
            general: ENDPOINT_FORM.ERROR_CREATE_PREFIX + error.message,
          });
        } else {
          setErrors({ general: ENDPOINT_FORM.ERROR_CREATE_GENERIC });
        }
      },
    });
  };

  const handleClose = () => {
    setFormData({
      eui: "",
      name: "",
      shortAddr: "",
      bidirectional: false,
      preAttach: false,
      carrierOffset: "", // Optional - empty = omit from request
      networkKey: "",
      applicationKey: "",
      typeEui: "",
      dualChan: false,
      repetition: false,
      wideCarrOff: false,
      longBlkDist: false,
      lastPacketCnt: "", // Empty = not set (per BSSCI §3.8.1)
      attachCnt: "", // Empty = not set (per SCACI §3.6.1)
      deviceModelId: "",
    });
    setErrors({});
    setEuiValid(false);
    onClose();
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>
        {ENDPOINT_FORM.DIALOG_TITLE}
        <Typography variant="body2" color="text.secondary">
          {ENDPOINT_FORM.DIALOG_SUBTITLE}
        </Typography>
      </DialogTitle>
      <DialogContent>
        {errors.general && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {errors.general}
          </Alert>
        )}

        <Grid container spacing={2} sx={{ mt: 1 }}>
          {/* Basic Information */}
          <Grid size={12}>
            <Typography variant="subtitle2" fontWeight="bold" mb={1}>
              {ENDPOINT_FORM.SECTION_BASIC}
            </Typography>
          </Grid>

          <Grid size={{ xs: 12, md: 6 }}>
            <TextField
              fullWidth
              label={ENDPOINT_FORM.LABEL_EUI}
              value={formData.eui}
              onChange={handleEuiChange}
              error={!!errors.eui || (!!formData.eui && !euiValid)}
              helperText={
                errors.eui ||
                (formData.eui && !euiValid
                  ? ENDPOINT_FORM.ERROR_EUI_FORMAT
                  : ENDPOINT_FORM.HELPER_EUI)
              }
              required
              inputProps={{ maxLength: 23 }}
              InputProps={{
                endAdornment: euiValid && (
                  <InputAdornment position="end">
                    <CheckCircleIcon color="success" />
                  </InputAdornment>
                ),
              }}
            />
          </Grid>

          <Grid size={{ xs: 12, md: 6 }}>
            <TextField
              fullWidth
              label={ENDPOINT_FORM.LABEL_NAME}
              value={formData.name}
              onChange={handleChange("name")}
              error={!!errors.name}
              helperText={errors.name || ENDPOINT_FORM.HELPER_NAME}
              required
            />
          </Grid>

          {/* Network Configuration */}
          <Grid size={12}>
            <Typography variant="subtitle2" fontWeight="bold" mb={1} mt={1}>
              {ENDPOINT_FORM.SECTION_NETWORK}
            </Typography>
          </Grid>

          <Grid size={{ xs: 12, md: 4 }}>
            <TextField
              fullWidth
              label={`${ENDPOINT_FORM.LABEL_SHORT_ADDR} *`}
              value={formData.shortAddr}
              onChange={handleChange("shortAddr")}
              error={!!errors.shortAddr}
              helperText={errors.shortAddr || ENDPOINT_FORM.HELPER_SHORT_ADDR}
              placeholder={ENDPOINT_FORM.PLACEHOLDER_SHORT_ADDR}
              required
              inputProps={{ maxLength: 4 }}
            />
          </Grid>

          <Grid size={{ xs: 12, md: 4 }}>
            <TextField
              fullWidth
              label={ENDPOINT_FORM.LABEL_CARRIER_OFFSET}
              value={formData.carrierOffset}
              onChange={handleChange("carrierOffset")}
              helperText={ENDPOINT_FORM.HELPER_CARRIER_OFFSET}
              type="number"
            />
          </Grid>

          <Grid size={{ xs: 12, md: 4 }}>
            <TextField
              fullWidth
              label={ENDPOINT_FORM.LABEL_TYPE_EUI}
              value={formData.typeEui}
              onChange={handleChange("typeEui")}
              error={!!errors.typeEui}
              helperText={
                formData.deviceModelId
                  ? ENDPOINT_FORM.HELPER_TYPE_EUI_MODEL_OVERRIDE
                  : errors.typeEui || ENDPOINT_FORM.HELPER_TYPE_EUI
              }
              disabled={!!formData.deviceModelId}
            />
          </Grid>

          {/* Communication Settings */}
          <Grid size={12}>
            <Typography variant="subtitle2" fontWeight="bold" mb={1} mt={1}>
              {ENDPOINT_FORM.SECTION_COMMUNICATION}
            </Typography>
          </Grid>

          <CommunicationSettings
            bidirectional={formData.bidirectional}
            preAttach={formData.preAttach}
            onBidirectionalChange={handleChange("bidirectional")}
            onPreAttachChange={handleChange("preAttach")}
          />

          {/* Advanced MIOTY Settings */}
          <Grid size={12}>
            <Typography variant="subtitle2" fontWeight="bold" mb={1} mt={1}>
              {ENDPOINT_FORM.SECTION_ADVANCED}
            </Typography>
          </Grid>

          <AdvancedMiotySettings
            dualChan={formData.dualChan}
            repetition={formData.repetition}
            wideCarrOff={formData.wideCarrOff}
            longBlkDist={formData.longBlkDist}
            onDualChanChange={handleChange("dualChan")}
            onRepetitionChange={handleChange("repetition")}
            onWideCarrOffChange={handleChange("wideCarrOff")}
            onLongBlkDistChange={handleChange("longBlkDist")}
          />

          <CounterFields
            lastPacketCnt={formData.lastPacketCnt}
            attachCnt={formData.attachCnt}
            onLastPacketCntChange={handleChange("lastPacketCnt")}
            onAttachCntChange={handleChange("attachCnt")}
            errors={errors}
            required
          />

          {/* Security Keys */}
          <Grid size={12}>
            <Typography variant="subtitle2" fontWeight="bold" mb={1} mt={1}>
              {ENDPOINT_FORM.SECTION_SECURITY}
            </Typography>
          </Grid>

          <SecurityKeyFields
            networkKey={formData.networkKey}
            applicationKey={formData.applicationKey}
            onNetworkKeyChange={handleChange("networkKey")}
            onApplicationKeyChange={handleChange("applicationKey")}
            onGenerateNetworkKey={handleGenerateNetworkKey}
            onGenerateApplicationKey={handleGenerateApplicationKey}
            errors={errors}
            networkKeyRequired
          />

          {/* Blueprint Configuration (Optional) */}
          <DeviceModelSelector
            value={formData.deviceModelId || undefined}
            onChange={(id) =>
              setFormData({
                ...formData,
                deviceModelId: id || "",
                typeEui: id ? "" : formData.typeEui,
              })
            }
          />

          {formData.preAttach && (
            <Grid size={12}>
              <Alert severity="info" icon={<InfoIcon />}>
                {ENDPOINT_FORM.ALERT_PREATTACH}
              </Alert>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>{ENDPOINT_FORM.BUTTON_CANCEL}</Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={
            // Disable while mutation is pending
            createEndpointMutation.isPending ||
            // All required fields per BSSCI §3.8.1, SCACI §3.6.1
            !euiValid ||
            !formData.name ||
            !formData.networkKey ||
            formData.shortAddr === "" ||
            formData.lastPacketCnt === "" ||
            formData.attachCnt === ""
          }
        >
          {createEndpointMutation.isPending
            ? ENDPOINT_FORM.BUTTON_CREATING
            : ENDPOINT_FORM.BUTTON_CREATE}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default AddEndPointDialog;
