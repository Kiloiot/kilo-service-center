/**
 * Field groups of the End Point form: the MIOTY flag checkboxes and the key
 * fields, composed by EndpointFormSections.
 */

import React from "react";

import { useRevealEndpointKey } from "@hooks";
import { Box, Checkbox, FormControlLabel } from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";

import { ENDPOINT_KEY } from "@constants/app";
import { ENDPOINT_FORM } from "@constants/messages";
import { InfoIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import type { EndpointFormState } from "../hooks";
import { SecretKeyField } from "./SecretKeyField";

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
  <Grid size={componentSpacing.gridSpan.half}>
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

interface SecurityKeyFieldsProps {
  form: EndpointFormState;
}

interface KeyRemoval {
  removed: boolean;
  remove: () => void;
}

const KEY_FIELDS = [
  {
    field: "networkKey",
    key: ENDPOINT_KEY.NETWORK,
    label: ENDPOINT_FORM.LABEL_NETWORK_KEY,
    helper: ENDPOINT_FORM.HELPER_NETWORK_KEY,
  },
  {
    field: "applicationKey",
    key: ENDPOINT_KEY.APPLICATION,
    label: ENDPOINT_FORM.LABEL_APP_KEY,
    helper: ENDPOINT_FORM.HELPER_APP_KEY,
  },
] as const;

/**
 * Network key and application key fields, masked and generatable; a key the
 * endpoint has stored is revealed on request and only then copyable.
 */
export const SecurityKeyFields: React.FC<SecurityKeyFieldsProps> = ({
  form,
}) => {
  const reveal = useRevealEndpointKey();
  const generators = {
    networkKey: form.generateNetworkKey,
    applicationKey: form.generateApplicationKey,
  };
  // Only the application key can be removed: the network session key is mandatory.
  const removals: Partial<
    Record<(typeof KEY_FIELDS)[number]["field"], KeyRemoval>
  > = {
    applicationKey: {
      removed: form.values.removeApplicationKey,
      remove: form.removeApplicationKey,
    },
  };
  const epEui = form.storedEpEui;
  return (
    <>
      {KEY_FIELDS.map(({ field, key, label, helper }) => {
        const removal = removals[field];
        const stored = form.storedKeys.has(field) && !removal?.removed;
        return (
          <SecretKeyField
            key={field}
            form={form}
            field={field}
            label={label}
            helper={helper}
            onGenerate={generators[field]}
            onReveal={
              epEui && stored
                ? () => reveal.mutateAsync({ epEui, key })
                : undefined
            }
            onRemove={stored ? removal?.remove : undefined}
            removed={removal?.removed}
          />
        );
      })}
    </>
  );
};
