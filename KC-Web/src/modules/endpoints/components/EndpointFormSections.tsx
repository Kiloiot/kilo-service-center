/**
 * The End Point form both dialogs render: the same sections and fields in
 * the same order, each field required and validated from one source.
 */

import React from "react";

import { Alert, InputAdornment, Typography } from "@mui/material";
import Grid from "@mui/material/Grid";

import { isValidEui } from "@utils/eui";
import { EUI_INPUT_MAX_LENGTH, MIOTY_UINT32_MAX } from "@constants/app";
import { ENDPOINT_FORM } from "@constants/messages";
import { CheckCircleIcon, InfoIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

import type { EndpointFormState } from "../hooks";
import DeviceModelSelector from "./DeviceModelSelector";
import {
  AdvancedMiotySettings,
  CommunicationSettings,
  SecurityKeyFields,
} from "./EndpointFormFields";
import { EndpointTextField } from "./EndpointTextField";

interface SectionHeadingProps {
  title: string;
}

const SectionHeading: React.FC<SectionHeadingProps> = ({ title }) => (
  <Grid size={componentSpacing.gridSpan.full}>
    <Typography variant="subtitle2" fontWeight="bold" mb={1} mt={1}>
      {title}
    </Typography>
  </Grid>
);

interface SectionProps {
  form: EndpointFormState;
}

const IdentitySection: React.FC<SectionProps> = ({ form }) => (
  <>
    <SectionHeading title={ENDPOINT_FORM.SECTION_BASIC} />
    <EndpointTextField
      form={form}
      field="epEui"
      label={ENDPOINT_FORM.LABEL_EUI}
      helper={ENDPOINT_FORM.HELPER_EUI}
      inputProps={{ maxLength: EUI_INPUT_MAX_LENGTH }}
      InputProps={{
        endAdornment: isValidEui(form.values.epEui) && (
          <InputAdornment position="end">
            <CheckCircleIcon color="success" />
          </InputAdornment>
        ),
      }}
    />
    <EndpointTextField
      form={form}
      field="name"
      label={ENDPOINT_FORM.LABEL_NAME}
      helper={ENDPOINT_FORM.HELPER_NAME}
    />
  </>
);

const NetworkSection: React.FC<SectionProps> = ({ form }) => (
  <>
    <SectionHeading title={ENDPOINT_FORM.SECTION_NETWORK} />
    <EndpointTextField
      form={form}
      field="shortAddr"
      label={ENDPOINT_FORM.LABEL_SHORT_ADDR}
      helper={ENDPOINT_FORM.HELPER_SHORT_ADDR}
      placeholder={ENDPOINT_FORM.PLACEHOLDER_SHORT_ADDR}
      inputProps={{ maxLength: ENDPOINT_FORM.PLACEHOLDER_SHORT_ADDR.length }}
    />
    <EndpointTextField
      form={form}
      field="typeEui"
      label={ENDPOINT_FORM.LABEL_TYPE_EUI}
      helper={
        form.values.deviceModelId
          ? ENDPOINT_FORM.HELPER_TYPE_EUI_MODEL_OVERRIDE
          : ENDPOINT_FORM.HELPER_TYPE_EUI
      }
      disabled={!!form.values.deviceModelId}
    />
  </>
);

const RadioSection: React.FC<SectionProps> = ({ form }) => (
  <>
    <SectionHeading title={ENDPOINT_FORM.SECTION_COMMUNICATION} />
    <CommunicationSettings
      bidirectional={form.values.bidirectional}
      preAttach={form.values.preAttach}
      onBidirectionalChange={form.handleInput("bidirectional")}
      onPreAttachChange={form.handleInput("preAttach")}
    />
    <SectionHeading title={ENDPOINT_FORM.SECTION_ADVANCED} />
    <AdvancedMiotySettings
      dualChan={form.values.dualChan}
      repetition={form.values.repetition}
      wideCarrOff={form.values.wideCarrOff}
      longBlkDist={form.values.longBlkDist}
      onDualChanChange={form.handleInput("dualChan")}
      onRepetitionChange={form.handleInput("repetition")}
      onWideCarrOffChange={form.handleInput("wideCarrOff")}
      onLongBlkDistChange={form.handleInput("longBlkDist")}
    />
    {(["lastPacketCnt", "attachCnt"] as const).map((field) => (
      <EndpointTextField
        key={field}
        form={form}
        field={field}
        label={COUNTER_LABELS[field]}
        helper={COUNTER_HELPERS[field]}
        type="number"
        inputProps={{ min: 0, max: MIOTY_UINT32_MAX }}
      />
    ))}
  </>
);

const COUNTER_LABELS = {
  lastPacketCnt: ENDPOINT_FORM.LABEL_LAST_PACKET_CNT,
  attachCnt: ENDPOINT_FORM.LABEL_ATTACH_CNT,
} as const;

const COUNTER_HELPERS = {
  lastPacketCnt: ENDPOINT_FORM.HELPER_LAST_PACKET_CNT,
  attachCnt: ENDPOINT_FORM.HELPER_ATTACH_CNT,
} as const;

const SecuritySection: React.FC<SectionProps> = ({ form }) => (
  <>
    <SectionHeading title={ENDPOINT_FORM.SECTION_SECURITY} />
    <SecurityKeyFields form={form} />
  </>
);

/** Every section of the End Point form, in the order both dialogs show them. */
export const EndpointFormSections: React.FC<SectionProps> = ({ form }) => (
  <Grid container spacing={2} sx={{ mt: 1 }}>
    <IdentitySection form={form} />
    <NetworkSection form={form} />
    <RadioSection form={form} />
    <SecuritySection form={form} />
    <DeviceModelSelector
      value={form.values.deviceModelId || undefined}
      onChange={form.setDeviceModel}
    />
    <SectionHeading title={ENDPOINT_FORM.SECTION_METADATA} />
    <EndpointTextField
      form={form}
      field="carrierOffset"
      label={ENDPOINT_FORM.LABEL_CARRIER_OFFSET}
      helper={ENDPOINT_FORM.HELPER_CARRIER_OFFSET}
    />
    {form.values.preAttach && (
      <Grid size={componentSpacing.gridSpan.full}>
        <Alert severity="info" icon={<InfoIcon />}>
          {ENDPOINT_FORM.ALERT_PREATTACH}
        </Alert>
      </Grid>
    )}
  </Grid>
);
