/**
 * Columns of the ulData table (SCACI §3.8.1); a row's receptions and their
 * subpackets are the shared uplink details.
 */

import type { UplinkUI } from "@api-types/api";

import type { DataTableColumn } from "@components/common/DataTable";
import { FlagChips } from "@components/common/FlagChips";
import { MonoText } from "@components/common/MonoText";
import { TimeText } from "@components/common/TimeText";
import { formatUnixNanos } from "@utils/date-format";
import { formatEui } from "@utils/eui";
import { formatMeasurement, formatOptional } from "@utils/formatters";
import { SCACI_FIELD, TABLE_HEADERS } from "@constants/messages";

import { UplinkUserDataCell } from "./UplinkUserDataCell";

export const UPLINK_COLUMNS: readonly DataTableColumn<UplinkUI>[] = [
  {
    id: "rxTime",
    header: SCACI_FIELD.RX_TIME,
    render: (u) => <TimeText>{formatUnixNanos(u.rxTime)}</TimeText>,
  },
  {
    id: "epEui",
    header: SCACI_FIELD.EP_EUI,
    render: (u) => <MonoText>{formatEui(u.epEui)}</MonoText>,
  },
  {
    id: "bsEui",
    header: SCACI_FIELD.BS_EUI,
    render: (u) => <MonoText>{formatEui(u.bsEui)}</MonoText>,
  },
  {
    id: "packetCnt",
    header: SCACI_FIELD.PACKET_CNT,
    align: "right",
    render: (u) => u.packetCnt,
  },
  {
    id: "snr",
    header: SCACI_FIELD.SNR,
    align: "right",
    render: (u) => formatMeasurement(u.snr),
  },
  {
    id: "rssi",
    header: SCACI_FIELD.RSSI,
    align: "right",
    render: (u) => formatMeasurement(u.rssi),
  },
  {
    id: "eqSnr",
    header: SCACI_FIELD.EQ_SNR,
    align: "right",
    render: (u) => formatMeasurement(u.eqSnr),
  },
  {
    id: "flags",
    header: TABLE_HEADERS.COL_FLAGS,
    render: (u) => (
      <FlagChips
        flags={[
          { label: SCACI_FIELD.DUPLICATE, set: u.duplicate },
          { label: SCACI_FIELD.DL_OPEN, set: u.dlOpen },
          { label: SCACI_FIELD.RESPONSE_EXP, set: u.responseExp },
          { label: SCACI_FIELD.DL_ACK, set: u.dlAck },
        ]}
      />
    ),
  },
  {
    id: "userData",
    header: SCACI_FIELD.USER_DATA,
    render: (u) => <UplinkUserDataCell uplink={u} />,
  },
  {
    id: "format",
    header: SCACI_FIELD.FORMAT,
    align: "right",
    render: (u) => formatOptional(u.format),
  },
  {
    id: "opId",
    header: SCACI_FIELD.OP_ID,
    render: (u) => <MonoText>{formatOptional(u.opId)}</MonoText>,
  },
];
