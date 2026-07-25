/**
 * Shared form field sections used by both AddEndPointDialog and EditEndPointDialog.
 * Eliminates duplication of checkbox groups, counter fields, and key fields.
 */

import React from "react";

import {
  Box,
  Button,
  Checkbox,
  FormControlLabel,
  InputAdornment,
  TextField,
} from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";

import { MIOTY_UINT32_MAX } from "@constants/app";
import { ENDPOINT_FORM } from "@constants/messages";
import { InfoIcon } from "@theme/icons";

interface CheckboxWithTooltipProps {
  checked: boolean;
  onChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
  label: string;
  tooltip: string;
}

/** Checkbox with an inline info tooltip, used for MIOTY boolean flags. */
const CheckboxWithTooltip: React.FC<CheckboxWithTooltipProps> = ({
  checked,
  onChange,
  label,
  tooltip,
}) => (
  <Grid size={{ xs: 12, md: 6 }}>
    <FormControlLabel
      control={
        <Checkbox checked={checked} onChange={onChange} color="primary" />
      }
      label={
        <Box sx={{ display: "flex", alignItems: "center" }}>
          {label}
          <Tooltip title={tooltip}>
            <InfoIcon fontSize="small" sx={{ ml: 0.5 }} />
          </Tooltip>
        </Box>
      }
    />
  </Grid>
);

interface CommunicationSettingsProps {
  bidirectional: boolean;
  preAttach: boolean;
  onBidirectionalChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
  onPreAttachChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
}

/** Bidirectional and Pre-Attach checkbox pair (BSSCI communication flags). */
export const CommunicationSettings: React.FC<CommunicationSettingsProps> = ({
  bidirectional,
  preAttach,
  onBidirectionalChange,
  onPreAttachChange,
}) => (
  <>
    <CheckboxWithTooltip
      checked={bidirectional}
      onChange={onBidirectionalChange}
      label={ENDPOINT_FORM.LABEL_BIDIRECTIONAL}
      tooltip={ENDPOINT_FORM.TOOLTIP_BIDI}
    />
    <CheckboxWithTooltip
      checked={preAttach}
      onChange={onPreAttachChange}
      label={ENDPOINT_FORM.LABEL_PRE_ATTACH}
      tooltip={ENDPOINT_FORM.TOOLTIP_PRE_ATTACH}
    />
  </>
);

interface AdvancedMiotySettingsProps {
  dualChan: boolean;
  repetition: boolean;
  wideCarrOff: boolean;
  longBlkDist: boolean;
  onDualChanChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
  onRepetitionChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
  onWideCarrOffChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
  onLongBlkDistChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
}

/** Advanced MIOTY radio-option checkboxes (dual channel, repetition, wide carrier, long block). */
export const AdvancedMiotySettings: React.FC<AdvancedMiotySettingsProps> = ({
  dualChan,
  repetition,
  wideCarrOff,
  longBlkDist,
  onDualChanChange,
  onRepetitionChange,
  onWideCarrOffChange,
  onLongBlkDistChange,
}) => (
  <>
    <CheckboxWithTooltip
      checked={dualChan}
      onChange={onDualChanChange}
      label={ENDPOINT_FORM.LABEL_DUAL_CHAN}
      tooltip={ENDPOINT_FORM.TOOLTIP_DUAL_CHAN}
    />
    <CheckboxWithTooltip
      checked={repetition}
      onChange={onRepetitionChange}
      label={ENDPOINT_FORM.LABEL_REPETITION}
      tooltip={ENDPOINT_FORM.TOOLTIP_REPETITION}
    />
    <CheckboxWithTooltip
      checked={wideCarrOff}
      onChange={onWideCarrOffChange}
      label={ENDPOINT_FORM.LABEL_WIDE_CARR_OFF}
      tooltip={ENDPOINT_FORM.TOOLTIP_WIDE_CARR_OFF}
    />
    <CheckboxWithTooltip
      checked={longBlkDist}
      onChange={onLongBlkDistChange}
      label={ENDPOINT_FORM.LABEL_LONG_BLK_DIST}
      tooltip={ENDPOINT_FORM.TOOLTIP_LONG_BLK_DIST}
    />
  </>
);

interface CounterFieldsProps {
  lastPacketCnt: string;
  attachCnt: string;
  onLastPacketCntChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
  onAttachCntChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
  errors: Record<string, string>;
  /** Whether the fields are required (true for Add, false for Edit). */
  required?: boolean;
}

/** lastPacketCnt and attachCnt counter text fields. */
export const CounterFields: React.FC<CounterFieldsProps> = ({
  lastPacketCnt,
  attachCnt,
  onLastPacketCntChange,
  onAttachCntChange,
  errors,
  required,
}) => (
  <>
    <Grid size={{ xs: 12, md: 6 }}>
      <TextField
        fullWidth
        label={
          required
            ? `${ENDPOINT_FORM.LABEL_LAST_PACKET_CNT} *`
            : ENDPOINT_FORM.LABEL_LAST_PACKET_CNT
        }
        value={lastPacketCnt}
        onChange={onLastPacketCntChange}
        error={!!errors.lastPacketCnt}
        helperText={
          errors.lastPacketCnt || ENDPOINT_FORM.HELPER_LAST_PACKET_CNT
        }
        type="number"
        required={required}
        inputProps={{ min: 0, max: MIOTY_UINT32_MAX }}
      />
    </Grid>
    <Grid size={{ xs: 12, md: 6 }}>
      <TextField
        fullWidth
        label={
          required
            ? `${ENDPOINT_FORM.LABEL_ATTACH_CNT} *`
            : ENDPOINT_FORM.LABEL_ATTACH_CNT
        }
        value={attachCnt}
        onChange={onAttachCntChange}
        error={!!errors.attachCnt}
        helperText={errors.attachCnt || ENDPOINT_FORM.HELPER_ATTACH_CNT}
        type="number"
        required={required}
        inputProps={{ min: 0, max: MIOTY_UINT32_MAX }}
      />
    </Grid>
  </>
);

interface SecurityKeyFieldsProps {
  networkKey: string;
  applicationKey: string;
  onNetworkKeyChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
  onApplicationKeyChange: (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => void;
  onGenerateNetworkKey: () => void;
  onGenerateApplicationKey: () => void;
  errors: Record<string, string>;
  /** Whether network key is required (true for Add). */
  networkKeyRequired?: boolean;
}

/** Network key and application key fields with generate buttons. */
export const SecurityKeyFields: React.FC<SecurityKeyFieldsProps> = ({
  networkKey,
  applicationKey,
  onNetworkKeyChange,
  onApplicationKeyChange,
  onGenerateNetworkKey,
  onGenerateApplicationKey,
  errors,
  networkKeyRequired,
}) => (
  <>
    <Grid size={12}>
      <TextField
        fullWidth
        label={ENDPOINT_FORM.LABEL_NETWORK_KEY}
        value={networkKey}
        onChange={onNetworkKeyChange}
        error={!!errors.networkKey}
        helperText={errors.networkKey || ENDPOINT_FORM.HELPER_NETWORK_KEY}
        required={networkKeyRequired}
        InputProps={{
          endAdornment: (
            <InputAdornment position="end">
              <Button size="small" onClick={onGenerateNetworkKey}>
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
        value={applicationKey}
        onChange={onApplicationKeyChange}
        error={!!errors.applicationKey}
        helperText={errors.applicationKey || ENDPOINT_FORM.HELPER_APP_KEY}
        InputProps={{
          endAdornment: (
            <InputAdornment position="end">
              <Button size="small" onClick={onGenerateApplicationKey}>
                {ENDPOINT_FORM.BUTTON_GENERATE}
              </Button>
            </InputAdornment>
          ),
        }}
      />
    </Grid>
  </>
);
