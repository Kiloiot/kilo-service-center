/**
 * Read-only MIOTY configuration of an endpoint (BSSCI §3.8.1) and how it is
 * heard: the base station serving it and its latest reception.
 */

import type { ReactNode } from "react";

import type { EndpointUI } from "@api-types/api";
import { Box, Typography } from "@mui/material";
import Grid from "@mui/material/Grid";

import { formatEui } from "@utils/eui";
import { formatMeasurement, formatShortAddress } from "@utils/formatters";
import { getMonoBody1 } from "@utils/typography";
import {
  ENDPOINT_CONFIG,
  ENDPOINT_DETAILS,
  ENDPOINT_FORM,
} from "@constants/messages";
import { componentSpacing } from "@theme/index";

interface ConfigRow {
  label: string;
  value: ReactNode;
  mono?: boolean;
}

const yesNo = (flag?: boolean): string =>
  flag ? ENDPOINT_CONFIG.VALUE_YES : ENDPOINT_CONFIG.VALUE_NO;

const keyState = (isSet?: boolean): string =>
  isSet ? ENDPOINT_CONFIG.VALUE_KEY_SET : ENDPOINT_CONFIG.VALUE_KEY_NOT_SET;

const orNotSet = (value?: string | number): string | number =>
  value ?? ENDPOINT_CONFIG.VALUE_NOT_SET;

const measured = (value: number | undefined, unit: string): string =>
  value === undefined
    ? ENDPOINT_CONFIG.VALUE_NOT_HEARD
    : `${formatMeasurement(value)} ${unit}`;

function configurationRows(endpoint: EndpointUI): ConfigRow[] {
  return [
    {
      label: ENDPOINT_FORM.LABEL_SHORT_ADDR,
      value:
        endpoint.shAddr === undefined
          ? ENDPOINT_CONFIG.VALUE_NOT_SET
          : formatShortAddress(endpoint.shAddr),
      mono: true,
    },
    {
      label: ENDPOINT_CONFIG.LABEL_CLASS,
      value: endpoint.bidi ? ENDPOINT_CONFIG.CLASS_A : ENDPOINT_CONFIG.CLASS_Z,
    },
    {
      label: ENDPOINT_FORM.LABEL_LAST_PACKET_CNT,
      value: orNotSet(endpoint.lastPacketCnt),
    },
    {
      label: ENDPOINT_FORM.LABEL_ATTACH_CNT,
      value: orNotSet(endpoint.attachCnt),
    },
    {
      label: ENDPOINT_FORM.LABEL_TYPE_EUI,
      value: endpoint.typeEui
        ? formatEui(endpoint.typeEui)
        : ENDPOINT_CONFIG.VALUE_NOT_SET,
      mono: true,
    },
    {
      label: ENDPOINT_FORM.LABEL_CARRIER_OFFSET,
      value: `${endpoint.carrierOffset ?? 0} ${ENDPOINT_CONFIG.UNIT_HZ}`,
    },
    { label: ENDPOINT_FORM.LABEL_PRE_ATTACH, value: yesNo(endpoint.preAttach) },
    { label: ENDPOINT_FORM.LABEL_DUAL_CHAN, value: yesNo(endpoint.dualChan) },
    {
      label: ENDPOINT_FORM.LABEL_REPETITION,
      value: yesNo(endpoint.repetition),
    },
    {
      label: ENDPOINT_FORM.LABEL_WIDE_CARR_OFF,
      value: yesNo(endpoint.wideCarrOff),
    },
    {
      label: ENDPOINT_FORM.LABEL_LONG_BLK_DIST,
      value: yesNo(endpoint.longBlkDist),
    },
    {
      label: ENDPOINT_CONFIG.LABEL_NETWORK_KEY,
      value: keyState(endpoint.nwkSnKeySet),
    },
    {
      label: ENDPOINT_CONFIG.LABEL_APP_KEY,
      value: keyState(endpoint.appKeySet),
    },
  ];
}

function receptionRows(endpoint: EndpointUI): ConfigRow[] {
  return [
    {
      label: ENDPOINT_CONFIG.LABEL_SERVING_BS,
      value: endpoint.servingBsEui
        ? formatEui(endpoint.servingBsEui)
        : ENDPOINT_CONFIG.VALUE_NO_SERVING_BS,
      mono: !!endpoint.servingBsEui,
    },
    {
      label: ENDPOINT_CONFIG.LABEL_LAST_RSSI,
      value: measured(endpoint.lastRssi, ENDPOINT_DETAILS.UNIT_DBM),
    },
    {
      label: ENDPOINT_CONFIG.LABEL_LAST_SNR,
      value: measured(endpoint.lastSnr, ENDPOINT_DETAILS.UNIT_DB),
    },
    {
      label: ENDPOINT_CONFIG.LABEL_LAST_EQ_SNR,
      value: measured(endpoint.lastEqSnr, ENDPOINT_DETAILS.UNIT_DB),
    },
  ];
}

function ConfigSection({ title, rows }: { title: string; rows: ConfigRow[] }) {
  return (
    <Box>
      <Typography variant="subtitle2" color="text.secondary" sx={{ mb: 2 }}>
        {title}
      </Typography>
      <Grid container spacing={1}>
        {rows.map((row) => (
          <Grid key={row.label} size={componentSpacing.gridSpan.halfFromSm}>
            <Typography variant="body2" color="text.secondary">
              {row.label}
            </Typography>
            <Typography
              variant="body1"
              sx={row.mono ? (theme) => getMonoBody1(theme) : undefined}
            >
              {row.value}
            </Typography>
          </Grid>
        ))}
      </Grid>
    </Box>
  );
}

export function EndpointConfigurationPanel({
  endpoint,
}: {
  endpoint: EndpointUI;
}) {
  return (
    <Grid container spacing={3}>
      <Grid size={componentSpacing.gridSpan.twoThirds}>
        <ConfigSection
          title={ENDPOINT_CONFIG.SECTION_CONFIGURATION}
          rows={configurationRows(endpoint)}
        />
      </Grid>
      <Grid size={componentSpacing.gridSpan.third}>
        <ConfigSection
          title={ENDPOINT_CONFIG.SECTION_RECEPTION}
          rows={receptionRows(endpoint)}
        />
      </Grid>
    </Grid>
  );
}
