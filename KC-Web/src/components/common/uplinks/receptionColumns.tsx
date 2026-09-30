/**
 * Columns of an uplink's details (SCACI §3.8.1): the base station receptions
 * and, per reception, its subpackets.
 */

import type { BaseStationReceptionAPI } from "@api-types/api";

import type { DataTableColumn } from "@components/common/DataTable";
import { MonoText } from "@components/common/MonoText";
import { TimeText } from "@components/common/TimeText";
import { formatUnixNanos } from "@utils/date-format";
import { formatEui } from "@utils/eui";
import { formatMeasurement, formatOptional } from "@utils/formatters";
import type { SubpacketRow } from "@utils/ul-data";
import { SCACI_FIELD, UPLINK_TABLE } from "@constants/messages";

export const RECEPTION_COLUMNS: readonly DataTableColumn<BaseStationReceptionAPI>[] =
  [
    {
      id: "bsEui",
      header: SCACI_FIELD.BS_EUI,
      render: (r) => <MonoText>{formatEui(r.bsEui)}</MonoText>,
    },
    {
      id: "rxTime",
      header: SCACI_FIELD.RX_TIME,
      render: (r) => <TimeText>{formatUnixNanos(r.rxTime)}</TimeText>,
    },
    {
      id: "rxDuration",
      header: SCACI_FIELD.RX_DURATION,
      align: "right",
      render: (r) => formatOptional(r.rxDuration),
    },
    {
      id: "snr",
      header: SCACI_FIELD.SNR,
      align: "right",
      render: (r) => formatMeasurement(r.snr),
    },
    {
      id: "rssi",
      header: SCACI_FIELD.RSSI,
      align: "right",
      render: (r) => formatMeasurement(r.rssi),
    },
    {
      id: "eqSnr",
      header: SCACI_FIELD.EQ_SNR,
      align: "right",
      render: (r) => formatMeasurement(r.eqSnr),
    },
    {
      id: "dlRxSnr",
      header: SCACI_FIELD.DL_RX_SNR,
      align: "right",
      render: (r) => formatMeasurement(r.dlRxSnr),
    },
    {
      id: "dlRxRssi",
      header: SCACI_FIELD.DL_RX_RSSI,
      align: "right",
      render: (r) => formatMeasurement(r.dlRxRssi),
    },
    {
      id: "profile",
      header: SCACI_FIELD.PROFILE,
      render: (r) => formatOptional(r.profile),
    },
    {
      id: "mode",
      header: SCACI_FIELD.MODE,
      render: (r) => formatOptional(r.mode),
    },
  ];

export const SUBPACKET_COLUMNS: readonly DataTableColumn<SubpacketRow>[] = [
  {
    id: "index",
    header: UPLINK_TABLE.COL_SUBPACKET_INDEX,
    align: "right",
    render: (s) => s.index,
  },
  {
    id: "snr",
    header: SCACI_FIELD.SNR,
    align: "right",
    render: (s) => formatMeasurement(s.snr),
  },
  {
    id: "rssi",
    header: SCACI_FIELD.RSSI,
    align: "right",
    render: (s) => formatMeasurement(s.rssi),
  },
  {
    id: "frequency",
    header: SCACI_FIELD.FREQUENCY,
    align: "right",
    render: (s) => s.frequency,
  },
  {
    id: "phase",
    header: SCACI_FIELD.PHASE,
    align: "right",
    render: (s) => formatMeasurement(s.phase),
  },
];
