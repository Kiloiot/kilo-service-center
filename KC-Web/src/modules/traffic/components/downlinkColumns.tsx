/**
 * Columns of the Downlink Queue (SCACI §3.10.1 dlDataQue) and Downlink
 * Results (SCACI §3.12.1 dlDataRes) tables.
 */

import type { SCACIDownlinkQueueDTO } from "@api-types/api";
import { Box, Chip, IconButton, Tooltip, Typography } from "@mui/material";

import type { DataTableColumn } from "@components/common/DataTable";
import { FlagChips } from "@components/common/FlagChips";
import { HexText } from "@components/common/HexText";
import { MonoText } from "@components/common/MonoText";
import { TimeText } from "@components/common/TimeText";
import { formatDateTime, formatUnixNanos } from "@utils/date-format";
import {
  downlinkPayloadLines,
  isRevocableDownlink,
} from "@utils/downlink-queue";
import { formatEui } from "@utils/eui";
import {
  formatDownlinkPriority,
  formatDownlinkQueueStatus,
  formatDownlinkResult,
  formatUserData,
} from "@utils/formatters";
import { DOWNLINK_QUEUE_STATUS, DOWNLINK_RESULT } from "@constants/app";
import {
  ACTION_REVOKE,
  DATA_TABLE,
  DOWNLINK_TABLE,
  LABEL_DL_ACCEPTED,
  LABEL_DL_ENDPOINT_ACK,
  LABEL_DL_PAYLOAD_COUNTER,
  MSG_DL_AWAITING_ACCEPTANCE,
  SCACI_FIELD,
  TABLE_HEADERS,
} from "@constants/messages";
import { CheckCircleIcon, DeleteIcon, EditIcon } from "@theme/icons";

type Downlink = SCACIDownlinkQueueDTO;

interface DownlinkRowActions {
  onEdit: (downlink: Downlink) => void;
  onRevoke: (downlink: Downlink) => void;
}

// Only a downlink no base station has taken yet can be rewritten (UpdatePendingDownlink).
const isEditable = (d: Downlink) => d.status === DOWNLINK_QUEUE_STATUS.PENDING;

const euiCell = (eui: string | undefined) =>
  eui ? <MonoText>{formatEui(eui)}</MonoText> : DATA_TABLE.NO_VALUE;

const userDataCell = (d: Downlink) => (
  <Box sx={{ display: "flex", flexDirection: "column" }}>
    {downlinkPayloadLines(d).map((line, i) => (
      <Box key={i} sx={{ display: "flex", gap: 1, alignItems: "baseline" }}>
        <HexText>{formatUserData(line.payload)}</HexText>
        {line.packetCnt !== undefined && (
          <Typography variant="caption" color="text.secondary" noWrap>
            {`${LABEL_DL_PAYLOAD_COUNTER} ${line.packetCnt}`}
          </Typography>
        )}
      </Box>
    ))}
  </Box>
);

// The station holding a queued downlink and whether it accepted it (dlDataQueRsp, BSSCI §3.12).
const holdingStationCell = (d: Downlink) =>
  d.bsEui ? (
    <Box sx={{ display: "flex", flexDirection: "column" }}>
      {euiCell(d.bsEui)}
      <Typography variant="caption" color="text.secondary">
        {d.acceptedAt
          ? `${LABEL_DL_ACCEPTED} ${formatDateTime(d.acceptedAt)}`
          : MSG_DL_AWAITING_ACCEPTANCE}
      </Typography>
    </Box>
  ) : (
    DATA_TABLE.NO_VALUE
  );

const QUE_ID: DataTableColumn<Downlink> = {
  id: "queId",
  header: SCACI_FIELD.QUE_ID,
  render: (d) => <MonoText>{d.queId}</MonoText>,
};

const EP_EUI: DataTableColumn<Downlink> = {
  id: "epEui",
  header: SCACI_FIELD.EP_EUI,
  render: (d) => euiCell(d.epEui),
};

const BS_EUI: DataTableColumn<Downlink> = {
  id: "bsEui",
  header: SCACI_FIELD.BS_EUI,
  render: (d) => euiCell(d.bsEui),
};

const HOLDING_STATION: DataTableColumn<Downlink> = {
  id: "bsEui",
  header: SCACI_FIELD.BS_EUI,
  render: holdingStationCell,
};

const USER_DATA: DataTableColumn<Downlink> = {
  id: "userData",
  header: SCACI_FIELD.USER_DATA,
  render: userDataCell,
};

const QUEUE_STATUS: DataTableColumn<Downlink> = {
  id: "status",
  header: TABLE_HEADERS.COL_STATUS,
  render: (d) => {
    const status = formatDownlinkQueueStatus(d.status ?? "");
    return <Chip label={status.label} color={status.color} size="small" />;
  },
};

const QUEUED_AT: DataTableColumn<Downlink> = {
  id: "queued",
  header: DOWNLINK_TABLE.COL_QUEUED,
  render: (d) => <TimeText>{formatDateTime(d.createdAt)}</TimeText>,
};

const PRIO: DataTableColumn<Downlink> = {
  id: "prio",
  header: SCACI_FIELD.PRIO,
  align: "right",
  render: (d) => formatDownlinkPriority(d.priority),
};

const CNT_DEPEND: DataTableColumn<Downlink> = {
  id: "cntDepend",
  header: SCACI_FIELD.CNT_DEPEND,
  render: (d) => (
    <FlagChips flags={[{ label: SCACI_FIELD.CNT_DEPEND, set: d.cntDepend }]} />
  ),
};

const FLAGS: DataTableColumn<Downlink> = {
  id: "flags",
  header: TABLE_HEADERS.COL_FLAGS,
  render: (d) => (
    <FlagChips
      flags={[
        { label: SCACI_FIELD.RESPONSE_EXP, set: d.responseExp },
        { label: SCACI_FIELD.RESPONSE_PRIO, set: d.responsePrio },
        { label: SCACI_FIELD.DL_WIND_REQ, set: d.dlWindReq },
        { label: SCACI_FIELD.EXP_ONLY, set: d.expOnly },
        { label: SCACI_FIELD.DL_RX_STAT_QRY, set: d.dlRxStatQry ?? false },
      ]}
    />
  ),
};

const rowActionsCell = (d: Downlink, actions: DownlinkRowActions) => (
  <Box sx={{ display: "flex", justifyContent: "flex-end" }}>
    {isEditable(d) && (
      <Tooltip title={DOWNLINK_TABLE.ACTION_EDIT}>
        <IconButton
          size="small"
          aria-label={DOWNLINK_TABLE.ACTION_EDIT}
          onClick={() => actions.onEdit(d)}
        >
          <EditIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    )}
    {isRevocableDownlink(d) && (
      <Tooltip title={ACTION_REVOKE}>
        <IconButton
          size="small"
          color="error"
          aria-label={ACTION_REVOKE}
          onClick={() => actions.onRevoke(d)}
        >
          <DeleteIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    )}
  </Box>
);

export function downlinkQueueColumns(
  actions: DownlinkRowActions,
): DataTableColumn<Downlink>[] {
  return [
    QUE_ID,
    EP_EUI,
    QUEUE_STATUS,
    QUEUED_AT,
    PRIO,
    CNT_DEPEND,
    FLAGS,
    HOLDING_STATION,
    USER_DATA,
    {
      id: "actions",
      header: TABLE_HEADERS.COL_ACTIONS,
      align: "right",
      render: (d) => rowActionsCell(d, actions),
    },
  ];
}

export const DOWNLINK_RESULT_COLUMNS: readonly DataTableColumn<Downlink>[] = [
  {
    id: "txTime",
    header: SCACI_FIELD.TX_TIME,
    render: (d) =>
      d.result === DOWNLINK_RESULT.SENT ? (
        <TimeText>{formatUnixNanos(d.txTime)}</TimeText>
      ) : (
        DATA_TABLE.NO_VALUE
      ),
  },
  EP_EUI,
  QUE_ID,
  {
    id: "result",
    header: SCACI_FIELD.RESULT,
    render: (d) => {
      const result = formatDownlinkResult(d.result ?? "");
      return <Chip label={result.label} color={result.color} size="small" />;
    },
  },
  BS_EUI,
  {
    id: "packetCnt",
    header: SCACI_FIELD.PACKET_CNT,
    align: "right",
    render: (d) =>
      d.result === DOWNLINK_RESULT.SENT
        ? d.transmissionPacketCnt
        : DATA_TABLE.NO_VALUE,
  },
  USER_DATA,
  {
    id: "dlAck",
    header: SCACI_FIELD.DL_ACK,
    render: (d) =>
      d.endpointAckedAt ? (
        <Tooltip title={LABEL_DL_ENDPOINT_ACK}>
          <Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
            <CheckCircleIcon color="success" fontSize="small" />
            <TimeText>{formatDateTime(d.endpointAckedAt)}</TimeText>
          </Box>
        </Tooltip>
      ) : (
        DATA_TABLE.NO_VALUE
      ),
  },
];
