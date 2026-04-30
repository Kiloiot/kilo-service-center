import React, { useCallback, useEffect, useState } from 'react';

import type { EndpointUI, UpdateEndpointRequest } from '@api-types/api';
import { useAttachEndpoint, useUpdateEndpoint } from '@hooks';
import {
  Alert,
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  InputAdornment,
  Snackbar,
  TextField,
  Typography,
} from '@mui/material';
import Grid from '@mui/material/Grid';
import Tooltip from '@mui/material/Tooltip';

import {
  generateRandomKey,
  validateEui,
  validateHexKey,
  validateShortAddr,
  validateTypeEui,
  validateUint32Counter,
} from '@utils/formatters';
import { MIOTY_KEY_BYTE_LENGTH, MIOTY_UINT32_MAX } from '@constants/app';
import { ENDPOINT_FORM } from '@constants/messages';
import { InfoIcon } from '@theme/icons';

import DeviceModelSelector from './DeviceModelSelector';

interface EditEndPointDialogProps {
  open: boolean;
  onClose: () => void;
  endpoint: EndpointUI | null;
}

/**
 * EditEndPointDialog - Edit existing endpoint configuration
 *
 * Uses same validation as AddEndPointDialog.
 * Only changed fields are sent in PATCH request.
 * EUI changes trigger a transactional cascade across all dependent tables.
 */
const EditEndPointDialog: React.FC<EditEndPointDialogProps> = ({ open, onClose, endpoint }) => {
  const updateEndpointMutation = useUpdateEndpoint();
  const attachEndpointMutation = useAttachEndpoint();

  // Form state - initialized from endpoint when dialog opens
  const [formData, setFormData] = useState({
    epEui: '',
    name: '',
    shortAddr: '',
    bidirectional: false,
    preAttach: false,
    carrierOffset: '',
    networkKey: '',
    applicationKey: '',
    dualChan: false,
    repetition: false,
    wideCarrOff: false,
    longBlkDist: false,
    lastPacketCnt: '',
    attachCnt: '',
    typeEui: '',
    deviceModelId: '',
  });

  // Track original values to compute diff
  const [originalData, setOriginalData] = useState<typeof formData | null>(null);

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [showReattachPrompt, setShowReattachPrompt] = useState(false);
  const [successMessage, setSuccessMessage] = useState('');

  // Initialize form from endpoint data when dialog opens
  useEffect(() => {
    if (open && endpoint) {
      const initialData = {
        epEui: endpoint.epEui || '',
        name: endpoint.name || '',
        shortAddr: endpoint.shAddr
          ? endpoint.shAddr.toString(16).toUpperCase().padStart(4, '0')
          : '',
        bidirectional: endpoint.bidi ?? false,
        preAttach: endpoint.preAttach ?? false,
        carrierOffset: endpoint.carrierOffset !== undefined ? String(endpoint.carrierOffset) : '',
        networkKey: endpoint.nwkSnKey || '',
        applicationKey: endpoint.appKey || '',
        dualChan: endpoint.dualChan ?? false,
        repetition: endpoint.repetition ?? false,
        wideCarrOff: endpoint.wideCarrOff ?? false,
        longBlkDist: endpoint.longBlkDist ?? false,
        lastPacketCnt: endpoint.lastPacketCnt !== undefined ? String(endpoint.lastPacketCnt) : '0',
        attachCnt: endpoint.attachCnt !== undefined ? String(endpoint.attachCnt) : '0',
        typeEui: endpoint.typeEui || '',
        deviceModelId: endpoint.deviceModelId || '',
      };
      setFormData(initialData);
      setOriginalData(initialData);
      setErrors({});
    }
  }, [open, endpoint]);

  const handleChange = useCallback(
    (field: string) => (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      const target = event.target as HTMLInputElement;
      const value = target.type === 'checkbox' ? target.checked : target.value;
      setFormData((prev) => ({ ...prev, [field]: value }));
      setErrors((prev) => ({ ...prev, [field]: '' }));
    },
    []
  );

  const handleGenerateNetworkKey = () => {
    setFormData((prev) => ({ ...prev, networkKey: generateRandomKey(MIOTY_KEY_BYTE_LENGTH) }));
  };

  const handleGenerateApplicationKey = () => {
    setFormData((prev) => ({
      ...prev,
      applicationKey: generateRandomKey(MIOTY_KEY_BYTE_LENGTH),
    }));
  };

  // Compute which fields changed (only send changed fields in PATCH)
  const computeChangedFields = useCallback((): UpdateEndpointRequest => {
    if (!originalData) return {};

    const changes: UpdateEndpointRequest = {};

    if (formData.epEui !== originalData.epEui && formData.epEui !== '') {
      changes.newEpEui = formData.epEui;
    }
    if (formData.name !== originalData.name) {
      changes.name = formData.name;
    }
    if (formData.shortAddr !== originalData.shortAddr && formData.shortAddr !== '') {
      changes.shAddr = parseInt(formData.shortAddr, 16);
    }
    if (formData.bidirectional !== originalData.bidirectional) {
      changes.bidi = formData.bidirectional;
    }
    if (formData.preAttach !== originalData.preAttach) {
      changes.preAttach = formData.preAttach;
    }
    if (formData.carrierOffset !== originalData.carrierOffset) {
      changes.carrierOffset =
        formData.carrierOffset !== '' ? parseInt(formData.carrierOffset, 10) : undefined;
    }
    if (formData.networkKey !== originalData.networkKey && formData.networkKey !== '') {
      changes.nwkSnKey = formData.networkKey;
    }
    if (formData.applicationKey !== originalData.applicationKey) {
      changes.appKey = formData.applicationKey || undefined;
    }
    if (formData.dualChan !== originalData.dualChan) {
      changes.dualChan = formData.dualChan;
    }
    if (formData.repetition !== originalData.repetition) {
      changes.repetition = formData.repetition;
    }
    if (formData.wideCarrOff !== originalData.wideCarrOff) {
      changes.wideCarrOff = formData.wideCarrOff;
    }
    if (formData.longBlkDist !== originalData.longBlkDist) {
      changes.longBlkDist = formData.longBlkDist;
    }
    if (formData.lastPacketCnt !== originalData.lastPacketCnt && formData.lastPacketCnt !== '') {
      changes.lastPacketCnt = parseInt(formData.lastPacketCnt, 10);
    }
    if (formData.attachCnt !== originalData.attachCnt && formData.attachCnt !== '') {
      changes.attachCnt = parseInt(formData.attachCnt, 10);
    }
    if (formData.typeEui !== originalData.typeEui) {
      changes.typeEui = formData.typeEui || null;
    }
    if (formData.deviceModelId !== originalData.deviceModelId) {
      changes.deviceModelId = formData.deviceModelId || null;
    }

    return changes;
  }, [formData, originalData]);

  // Check if any profile fields changed (would require re-attach)
  const hasProfileChanges = useCallback((): boolean => {
    if (!originalData) return false;

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
  }, [formData, originalData]);

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    // EUI: 16 hex chars required
    const euiErr = validateEui(formData.epEui);
    if (euiErr) newErrors.epEui = euiErr;

    if (!formData.name) newErrors.name = ENDPOINT_FORM.ERROR_NAME_REQUIRED;

    // ShAddr: validated when provided
    if (formData.shortAddr !== '') {
      const shAddrErr = validateShortAddr(formData.shortAddr);
      if (shAddrErr) newErrors.shortAddr = shAddrErr;
    }

    // Keys: validated when provided
    if (formData.networkKey) {
      const nwkKeyErr = validateHexKey(formData.networkKey, true);
      if (nwkKeyErr) newErrors.networkKey = nwkKeyErr;
    }
    if (formData.applicationKey) {
      const appKeyErr = validateHexKey(formData.applicationKey, false);
      if (appKeyErr) newErrors.applicationKey = appKeyErr;
    }

    // Counters: validated when provided
    if (formData.lastPacketCnt !== '') {
      const pktErr = validateUint32Counter(formData.lastPacketCnt, 'lastPacketCnt');
      if (pktErr) newErrors.lastPacketCnt = pktErr;
    }
    if (formData.attachCnt !== '') {
      const attErr = validateUint32Counter(formData.attachCnt, 'attachCnt');
      if (attErr) newErrors.attachCnt = attErr;
    }

    // Type EUI: 16 hex chars if provided
    const typeEuiErr = validateTypeEui(formData.typeEui);
    if (typeEuiErr) newErrors.typeEui = typeEuiErr;

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm() || !endpoint) {
      return;
    }

    const changedFields = computeChangedFields();

    // Check if anything changed
    if (Object.keys(changedFields).length === 0) {
      handleClose();
      return;
    }

    updateEndpointMutation.mutate(
      { epEui: endpoint.epEui, data: changedFields },
      {
        onSuccess: () => {
          setSuccessMessage(ENDPOINT_FORM.MSG_ENDPOINT_UPDATED);
          // Check if profile fields changed and prompt for re-attach
          if (hasProfileChanges()) {
            setShowReattachPrompt(true);
          } else {
            handleClose();
          }
        },
        onError: (error) => {
          const message = error instanceof Error ? error.message : '';
          setErrors({
            general: message
              ? `${ENDPOINT_FORM.ERROR_UPDATE_PREFIX}${message}`
              : ENDPOINT_FORM.ERROR_UPDATE_GENERIC,
          });
        },
      }
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
      epEui: '',
      name: '',
      shortAddr: '',
      bidirectional: false,
      preAttach: false,
      carrierOffset: '',
      networkKey: '',
      applicationKey: '',
      dualChan: false,
      repetition: false,
      wideCarrOff: false,
      longBlkDist: false,
      lastPacketCnt: '',
      attachCnt: '',
      typeEui: '',
      deviceModelId: '',
    });
    setOriginalData(null);
    setErrors({});
    setShowReattachPrompt(false);
    onClose();
  };

  // Check if form has any changes
  const hasChanges = Object.keys(computeChangedFields()).length > 0;

  if (!endpoint) return null;

  return (
    <>
      <Dialog open={open && !showReattachPrompt} onClose={handleClose} maxWidth="md" fullWidth>
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
                onChange={handleChange('epEui')}
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
                onChange={handleChange('name')}
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
                onChange={handleChange('shortAddr')}
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
                onChange={handleChange('carrierOffset')}
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

            <Grid size={{ xs: 12, md: 6 }}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={formData.bidirectional}
                    onChange={handleChange('bidirectional')}
                    color="primary"
                  />
                }
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    {ENDPOINT_FORM.LABEL_BIDIRECTIONAL}
                    <Tooltip title={ENDPOINT_FORM.TOOLTIP_BIDI}>
                      <InfoIcon fontSize="small" sx={{ ml: 0.5 }} />
                    </Tooltip>
                  </Box>
                }
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={formData.preAttach}
                    onChange={handleChange('preAttach')}
                    color="primary"
                  />
                }
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    {ENDPOINT_FORM.LABEL_PRE_ATTACH}
                    <Tooltip title={ENDPOINT_FORM.TOOLTIP_PRE_ATTACH}>
                      <InfoIcon fontSize="small" sx={{ ml: 0.5 }} />
                    </Tooltip>
                  </Box>
                }
              />
            </Grid>

            {/* Advanced MIOTY Settings */}
            <Grid size={12}>
              <Typography variant="subtitle2" fontWeight="bold" mb={1} mt={1}>
                {ENDPOINT_FORM.SECTION_ADVANCED}
              </Typography>
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={formData.dualChan}
                    onChange={handleChange('dualChan')}
                    color="primary"
                  />
                }
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    {ENDPOINT_FORM.LABEL_DUAL_CHAN}
                    <Tooltip title={ENDPOINT_FORM.TOOLTIP_DUAL_CHAN}>
                      <InfoIcon fontSize="small" sx={{ ml: 0.5 }} />
                    </Tooltip>
                  </Box>
                }
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={formData.repetition}
                    onChange={handleChange('repetition')}
                    color="primary"
                  />
                }
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    {ENDPOINT_FORM.LABEL_REPETITION}
                    <Tooltip title={ENDPOINT_FORM.TOOLTIP_REPETITION}>
                      <InfoIcon fontSize="small" sx={{ ml: 0.5 }} />
                    </Tooltip>
                  </Box>
                }
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={formData.wideCarrOff}
                    onChange={handleChange('wideCarrOff')}
                    color="primary"
                  />
                }
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    {ENDPOINT_FORM.LABEL_WIDE_CARR_OFF}
                    <Tooltip title={ENDPOINT_FORM.TOOLTIP_WIDE_CARR_OFF}>
                      <InfoIcon fontSize="small" sx={{ ml: 0.5 }} />
                    </Tooltip>
                  </Box>
                }
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={formData.longBlkDist}
                    onChange={handleChange('longBlkDist')}
                    color="primary"
                  />
                }
                label={
                  <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    {ENDPOINT_FORM.LABEL_LONG_BLK_DIST}
                    <Tooltip title={ENDPOINT_FORM.TOOLTIP_LONG_BLK_DIST}>
                      <InfoIcon fontSize="small" sx={{ ml: 0.5 }} />
                    </Tooltip>
                  </Box>
                }
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <TextField
                fullWidth
                label={ENDPOINT_FORM.LABEL_LAST_PACKET_CNT}
                value={formData.lastPacketCnt}
                onChange={handleChange('lastPacketCnt')}
                error={!!errors.lastPacketCnt}
                helperText={errors.lastPacketCnt || ENDPOINT_FORM.HELPER_LAST_PACKET_CNT}
                type="number"
                inputProps={{ min: 0, max: MIOTY_UINT32_MAX }}
              />
            </Grid>

            <Grid size={{ xs: 12, md: 6 }}>
              <TextField
                fullWidth
                label={ENDPOINT_FORM.LABEL_ATTACH_CNT}
                value={formData.attachCnt}
                onChange={handleChange('attachCnt')}
                error={!!errors.attachCnt}
                helperText={errors.attachCnt || ENDPOINT_FORM.HELPER_ATTACH_CNT}
                type="number"
                inputProps={{ min: 0, max: MIOTY_UINT32_MAX }}
              />
            </Grid>

            {/* Security Keys */}
            <Grid size={12}>
              <Typography variant="subtitle2" fontWeight="bold" mb={1} mt={1}>
                {ENDPOINT_FORM.SECTION_SECURITY}
              </Typography>
            </Grid>

            <Grid size={12}>
              <TextField
                fullWidth
                label={ENDPOINT_FORM.LABEL_NETWORK_KEY}
                value={formData.networkKey}
                onChange={handleChange('networkKey')}
                error={!!errors.networkKey}
                helperText={errors.networkKey || ENDPOINT_FORM.HELPER_NETWORK_KEY}
                InputProps={{
                  endAdornment: (
                    <InputAdornment position="end">
                      <Button size="small" onClick={handleGenerateNetworkKey}>
                        {ENDPOINT_FORM.BUTTON_GENERATE}
                      </Button>
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>

            <Grid size={12}>
              <TextField
                fullWidth
                label={ENDPOINT_FORM.LABEL_APP_KEY}
                value={formData.applicationKey}
                onChange={handleChange('applicationKey')}
                error={!!errors.applicationKey}
                helperText={errors.applicationKey || ENDPOINT_FORM.HELPER_APP_KEY}
                InputProps={{
                  endAdornment: (
                    <InputAdornment position="end">
                      <Button size="small" onClick={handleGenerateApplicationKey}>
                        {ENDPOINT_FORM.BUTTON_GENERATE}
                      </Button>
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>

            {/* Type EUI */}
            <Grid size={12}>
              <TextField
                fullWidth
                label={ENDPOINT_FORM.LABEL_TYPE_EUI}
                value={formData.typeEui}
                onChange={handleChange('typeEui')}
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
                  deviceModelId: id || '',
                  typeEui: id ? '' : prev.typeEui,
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
          <Button onClick={handleSkipReattach}>{ENDPOINT_FORM.REATTACH_BUTTON_LATER}</Button>
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
        onClose={() => setSuccessMessage('')}
        message={successMessage}
      />
    </>
  );
};

export default EditEndPointDialog;
