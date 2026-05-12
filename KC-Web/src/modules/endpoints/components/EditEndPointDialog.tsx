import React, { useCallback, useEffect, useState } from "react";

import type { EndpointUI, UpdateEndpointRequest } from "@api-types/api";
import { useAttachEndpoint, useUpdateEndpoint } from "@hooks";
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Snackbar,
  TextField,
  Typography,
} from "@mui/material";
import Grid from "@mui/material/Grid";

import {
  generateRandomKey,
  validateEui,
  validateHexKey,
  validateShortAddr,
  validateTypeEui,
  validateUint32Counter,
} from "@utils/formatters";
import { MIOTY_KEY_BYTE_LENGTH } from "@constants/app";
import { ENDPOINT_FORM } from "@constants/messages";
import { InfoIcon } from "@theme/icons";

import {
  AdvancedMiotySettings,
  CommunicationSettings,
  CounterFields,
  SecurityKeyFields,
} from "./EndpointFormFields";
import DeviceModelSelector from "./DeviceModelSelector";

interface EditEndPointDialogProps {
  open: boolean;
  onClose: () => void;
  endpoint: EndpointUI | null;
}

interface EditEndpointFormData {
  epEui: string;
  name: string;
  shortAddr: string;
  bidirectional: boolean;
  preAttach: boolean;
  carrierOffset: string;
  networkKey: string;
  applicationKey: string;
  dualChan: boolean;
  repetition: boolean;
  wideCarrOff: boolean;
  longBlkDist: boolean;
  lastPacketCnt: string;
  attachCnt: string;
  typeEui: string;
  deviceModelId: string;
}

/** Builds initial EditEndpointFormData from a loaded EndpointUI. */
function buildEditEndpointFormData(endpoint: EndpointUI): EditEndpointFormData {
  return {
    epEui: endpoint.epEui || "",
    name: endpoint.name || "",
    shortAddr: endpoint.shAddr
      ? endpoint.shAddr.toString(16).toUpperCase().padStart(4, "0")
      : "",
    bidirectional: endpoint.bidi ?? false,
    preAttach: endpoint.preAttach ?? false,
    carrierOffset:
      endpoint.carrierOffset !== undefined
        ? String(endpoint.carrierOffset)
        : "",
    networkKey: endpoint.nwkSnKey || "",
    applicationKey: endpoint.appKey || "",
    dualChan: endpoint.dualChan ?? false,
    repetition: endpoint.repetition ?? false,
    wideCarrOff: endpoint.wideCarrOff ?? false,
    longBlkDist: endpoint.longBlkDist ?? false,
    lastPacketCnt:
      endpoint.lastPacketCnt !== undefined
        ? String(endpoint.lastPacketCnt)
        : "0",
    attachCnt:
      endpoint.attachCnt !== undefined ? String(endpoint.attachCnt) : "0",
    typeEui: endpoint.typeEui || "",
    deviceModelId: endpoint.deviceModelId || "",
  };
}

/** Apply epEui (cascade rename via newEpEui) + name diffs. */
function applyEndpointIdentityChanges(
  changes: UpdateEndpointRequest,
  form: EditEndpointFormData,
  original: EditEndpointFormData,
): void {
  if (form.epEui !== original.epEui && form.epEui !== "") {
    changes.newEpEui = form.epEui;
  }
  if (form.name !== original.name) changes.name = form.name;
}

/** Apply shortAddr (hex) + carrierOffset diffs. */
function applyEndpointAddressChanges(
  changes: UpdateEndpointRequest,
  form: EditEndpointFormData,
  original: EditEndpointFormData,
): void {
  if (form.shortAddr !== original.shortAddr && form.shortAddr !== "") {
    changes.shAddr = parseInt(form.shortAddr, 16);
  }
  if (form.carrierOffset !== original.carrierOffset) {
    changes.carrierOffset =
      form.carrierOffset !== ""
        ? parseInt(form.carrierOffset, 10)
        : undefined;
  }
}

/** Apply networkKey (nwkSnKey) + applicationKey (appKey) diffs. */
function applyEndpointSecurityChanges(
  changes: UpdateEndpointRequest,
  form: EditEndpointFormData,
  original: EditEndpointFormData,
): void {
  if (form.networkKey !== original.networkKey && form.networkKey !== "") {
    changes.nwkSnKey = form.networkKey;
  }
  if (form.applicationKey !== original.applicationKey) {
    changes.appKey = form.applicationKey || undefined;
  }
}

/** Apply MIOTY radio-option diffs (bidi/preAttach/dualChan/repetition/wide/long). */
function applyEndpointMiotyOptionChanges(
  changes: UpdateEndpointRequest,
  form: EditEndpointFormData,
  original: EditEndpointFormData,
): void {
  if (form.bidirectional !== original.bidirectional) {
    changes.bidi = form.bidirectional;
  }
  if (form.preAttach !== original.preAttach) {
    changes.preAttach = form.preAttach;
  }
  if (form.dualChan !== original.dualChan) {
    changes.dualChan = form.dualChan;
  }
  if (form.repetition !== original.repetition) {
    changes.repetition = form.repetition;
  }
  if (form.wideCarrOff !== original.wideCarrOff) {
    changes.wideCarrOff = form.wideCarrOff;
  }
  if (form.longBlkDist !== original.longBlkDist) {
    changes.longBlkDist = form.longBlkDist;
  }
}

/** Apply lastPacketCnt + attachCnt counter diffs. */
function applyEndpointCounterChanges(
  changes: UpdateEndpointRequest,
  form: EditEndpointFormData,
  original: EditEndpointFormData,
): void {
  if (
    form.lastPacketCnt !== original.lastPacketCnt &&
    form.lastPacketCnt !== ""
  ) {
    changes.lastPacketCnt = parseInt(form.lastPacketCnt, 10);
  }
  if (form.attachCnt !== original.attachCnt && form.attachCnt !== "") {
    changes.attachCnt = parseInt(form.attachCnt, 10);
  }
}

/**
 * Apply typeEui + deviceModelId blueprint diffs. Empty strings on these
 * fields are surfaced as explicit `null` so the backend clears the column.
 */
function applyEndpointBlueprintChanges(
  changes: UpdateEndpointRequest,
  form: EditEndpointFormData,
  original: EditEndpointFormData,
): void {
  if (form.typeEui !== original.typeEui) {
    changes.typeEui = form.typeEui || null;
  }
  if (form.deviceModelId !== original.deviceModelId) {
    changes.deviceModelId = form.deviceModelId || null;
  }
}

/**
 * Computes the diff between form data and the original snapshot, returning
 * an UpdateEndpointRequest with only the changed fields. EUI renames are
 * surfaced via newEpEui per the cascade rename contract; clearable string
 * fields use null to request explicit clear.
 */
function buildChangedEndpointRequest(
  formData: EditEndpointFormData,
  originalData: EditEndpointFormData,
): UpdateEndpointRequest {
  const changes: UpdateEndpointRequest = {};
  applyEndpointIdentityChanges(changes, formData, originalData);
  applyEndpointAddressChanges(changes, formData, originalData);
  applyEndpointSecurityChanges(changes, formData, originalData);
  applyEndpointMiotyOptionChanges(changes, formData, originalData);
  applyEndpointCounterChanges(changes, formData, originalData);
  applyEndpointBlueprintChanges(changes, formData, originalData);
  return changes;
}

/**
 * Returns true when any profile field changed that would require the
 * endpoint to be re-attached on the base station (radio, key, or address
 * configuration).
 */
function hasEndpointProfileChanges(
  formData: EditEndpointFormData,
  originalData: EditEndpointFormData,
): boolean {
  return (
    formData.shortAddr !== originalData.shortAddr ||
    formData.networkKey !== originalData.networkKey ||
    formData.applicationKey !== originalData.applicationKey ||
    formData.bidirectional !== originalData.bidirectional ||
    formData.preAttach !== originalData.preAttach ||
    formData.dualChan !== originalData.dualChan ||
    formData.repetition !== originalData.repetition ||
    formData.wideCarrOff !== originalData.wideCarrOff ||
    formData.longBlkDist !== originalData.longBlkDist
  );
}

/** Validate epEui format + required name. */
function validateEditEndpointIdentity(
  form: EditEndpointFormData,
  errors: Record<string, string>,
): void {
  const euiErr = validateEui(form.epEui);
  if (euiErr) errors.epEui = euiErr;
  if (!form.name) errors.name = ENDPOINT_FORM.ERROR_NAME_REQUIRED;
}

/** Validate hex format of network/application keys when provided. */
function validateEditEndpointSecurity(
  form: EditEndpointFormData,
  errors: Record<string, string>,
): void {
  if (form.networkKey) {
    const nwkKeyErr = validateHexKey(form.networkKey, true);
    if (nwkKeyErr) errors.networkKey = nwkKeyErr;
  }
  if (form.applicationKey) {
    const appKeyErr = validateHexKey(form.applicationKey, false);
    if (appKeyErr) errors.applicationKey = appKeyErr;
  }
}

/** Validate lastPacketCnt + attachCnt uint32 range when provided. */
function validateEditEndpointCounters(
  form: EditEndpointFormData,
  errors: Record<string, string>,
): void {
  if (form.lastPacketCnt !== "") {
    const pktErr = validateUint32Counter(form.lastPacketCnt, "lastPacketCnt");
    if (pktErr) errors.lastPacketCnt = pktErr;
  }
  if (form.attachCnt !== "") {
    const attErr = validateUint32Counter(form.attachCnt, "attachCnt");
    if (attErr) errors.attachCnt = attErr;
  }
}

/** Validate shortAddr (MIOTY 4-hex-char short address) when provided. */
function validateEditEndpointMiotyConfig(
  form: EditEndpointFormData,
  errors: Record<string, string>,
): void {
  if (form.shortAddr !== "") {
    const shAddrErr = validateShortAddr(form.shortAddr);
    if (shAddrErr) errors.shortAddr = shAddrErr;
  }
}

/** Validate typeEui blueprint identifier format when provided. */
function validateEditEndpointBlueprint(
  form: EditEndpointFormData,
  errors: Record<string, string>,
): void {
  const typeEuiErr = validateTypeEui(form.typeEui);
  if (typeEuiErr) errors.typeEui = typeEuiErr;
}

/** Validates the Edit dialog form. Returns sparse errors map keyed by field. */
function validateEditEndpointForm(
  formData: EditEndpointFormData,
): Record<string, string> {
  const errors: Record<string, string> = {};
  validateEditEndpointIdentity(formData, errors);
  validateEditEndpointSecurity(formData, errors);
  validateEditEndpointCounters(formData, errors);
  validateEditEndpointMiotyConfig(formData, errors);
  validateEditEndpointBlueprint(formData, errors);
  return errors;
}

/**
 * EditEndPointDialog - Edit existing endpoint configuration
 *
 * Uses same validation as AddEndPointDialog.
 * Only changed fields are sent in PATCH request.
 * EUI changes trigger a transactional cascade across all dependent tables.
 */
const EditEndPointDialog: React.FC<EditEndPointDialogProps> = ({
  open,
  onClose,
  endpoint,
}) => {
  const updateEndpointMutation = useUpdateEndpoint();
  const attachEndpointMutation = useAttachEndpoint();

  // Form state - initialized from endpoint when dialog opens
  const [formData, setFormData] = useState({
    epEui: "",
    name: "",
    shortAddr: "",
    bidirectional: false,
    preAttach: false,
    carrierOffset: "",
    networkKey: "",
    applicationKey: "",
    dualChan: false,
    repetition: false,
    wideCarrOff: false,
    longBlkDist: false,
    lastPacketCnt: "",
    attachCnt: "",
    typeEui: "",
    deviceModelId: "",
  });

  // Track original values to compute diff
  const [originalData, setOriginalData] = useState<typeof formData | null>(
    null,
  );

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [showReattachPrompt, setShowReattachPrompt] = useState(false);
  const [successMessage, setSuccessMessage] = useState("");

  // Initialize form from endpoint data when dialog opens
  useEffect(() => {
    if (open && endpoint) {
      const initialData = buildEditEndpointFormData(endpoint);
      setFormData(initialData);
      setOriginalData(initialData);
      setErrors({});
    }
  }, [open, endpoint]);

  const handleChange = useCallback(
    (field: string) =>
      (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        const target = event.target as HTMLInputElement;
        const value =
          target.type === "checkbox" ? target.checked : target.value;
        setFormData((prev) => ({ ...prev, [field]: value }));
        setErrors((prev) => ({ ...prev, [field]: "" }));
      },
    [],
  );

  const handleGenerateNetworkKey = () => {
    setFormData((prev) => ({
      ...prev,
      networkKey: generateRandomKey(MIOTY_KEY_BYTE_LENGTH),
    }));
  };

  const handleGenerateApplicationKey = () => {
    setFormData((prev) => ({
      ...prev,
      applicationKey: generateRandomKey(MIOTY_KEY_BYTE_LENGTH),
    }));
  };

  const handleSubmit = async () => {
    if (!endpoint || !originalData) return;
    const newErrors = validateEditEndpointForm(formData);
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    const changedFields = buildChangedEndpointRequest(formData, originalData);
    if (Object.keys(changedFields).length === 0) {
      handleClose();
      return;
    }

    updateEndpointMutation.mutate(
      { epEui: endpoint.epEui, data: changedFields },
      {
        onSuccess: () => {
          setSuccessMessage(ENDPOINT_FORM.MSG_ENDPOINT_UPDATED);
          if (hasEndpointProfileChanges(formData, originalData)) {
            setShowReattachPrompt(true);
          } else {
            handleClose();
          }
        },
        onError: (error) => {
          const message = error instanceof Error ? error.message : "";
          setErrors({
            general: message
              ? `${ENDPOINT_FORM.ERROR_UPDATE_PREFIX}${message}`
              : ENDPOINT_FORM.ERROR_UPDATE_GENERIC,
          });
        },
      },
    );
  };

  const handleReattach = () => {
    if (endpoint) {
      attachEndpointMutation.mutate(formData.epEui, {
        onSuccess: () => {
          setShowReattachPrompt(false);
          handleClose();
        },
        onError: () => {
          setShowReattachPrompt(false);
          handleClose();
        },
      });
    }
  };

  const handleSkipReattach = () => {
    setShowReattachPrompt(false);
    handleClose();
  };

  const handleClose = () => {
    setFormData({
      epEui: "",
      name: "",
      shortAddr: "",
      bidirectional: false,
      preAttach: false,
      carrierOffset: "",
      networkKey: "",
      applicationKey: "",
      dualChan: false,
      repetition: false,
      wideCarrOff: false,
      longBlkDist: false,
      lastPacketCnt: "",
      attachCnt: "",
      typeEui: "",
      deviceModelId: "",
    });
    setOriginalData(null);
    setErrors({});
    setShowReattachPrompt(false);
    onClose();
  };

  // Check if form has any changes
  const hasChanges = originalData
    ? Object.keys(buildChangedEndpointRequest(formData, originalData)).length >
      0
    : false;

  if (!endpoint) return null;

  return (
    <>
      <Dialog
        open={open && !showReattachPrompt}
        onClose={handleClose}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          {ENDPOINT_FORM.EDIT_DIALOG_TITLE}
          <Typography variant="body2" color="text.secondary">
            {ENDPOINT_FORM.EDIT_DIALOG_SUBTITLE}
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
                value={formData.epEui}
                onChange={handleChange("epEui")}
                error={!!errors.epEui}
                helperText={errors.epEui || ENDPOINT_FORM.HELPER_EUI}
                inputProps={{ maxLength: 16 }}
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
                label={ENDPOINT_FORM.LABEL_SHORT_ADDR}
                value={formData.shortAddr}
                onChange={handleChange("shortAddr")}
                error={!!errors.shortAddr}
                helperText={errors.shortAddr || ENDPOINT_FORM.HELPER_SHORT_ADDR}
                placeholder={ENDPOINT_FORM.PLACEHOLDER_SHORT_ADDR}
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
            />

            {/* Type EUI */}
            <Grid size={12}>
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

            {/* Blueprint Configuration (Optional) */}
            <DeviceModelSelector
              value={formData.deviceModelId || undefined}
              onChange={(id) => {
                setFormData((prev) => ({
                  ...prev,
                  deviceModelId: id || "",
                  typeEui: id ? "" : prev.typeEui,
                }));
              }}
            />
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>{ENDPOINT_FORM.BUTTON_CANCEL}</Button>
          <Button
            onClick={handleSubmit}
            variant="contained"
            disabled={updateEndpointMutation.isPending || !hasChanges}
          >
            {updateEndpointMutation.isPending
              ? ENDPOINT_FORM.BUTTON_UPDATING
              : ENDPOINT_FORM.BUTTON_UPDATE}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Re-attach Prompt Dialog */}
      <Dialog open={showReattachPrompt} onClose={handleSkipReattach}>
        <DialogTitle>{ENDPOINT_FORM.REATTACH_DIALOG_TITLE}</DialogTitle>
        <DialogContent>
          <Alert severity="info" sx={{ mt: 1 }}>
            {ENDPOINT_FORM.ALERT_REATTACH_PROMPT}
          </Alert>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleSkipReattach}>
            {ENDPOINT_FORM.REATTACH_BUTTON_LATER}
          </Button>
          <Button
            onClick={handleReattach}
            variant="contained"
            disabled={attachEndpointMutation.isPending}
          >
            {attachEndpointMutation.isPending
              ? ENDPOINT_FORM.REATTACH_BUTTON_PENDING
              : ENDPOINT_FORM.REATTACH_BUTTON_NOW}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Success Snackbar */}
      <Snackbar
        open={!!successMessage}
        autoHideDuration={3000}
        onClose={() => setSuccessMessage("")}
        message={successMessage}
      />
    </>
  );
};

export default EditEndPointDialog;
