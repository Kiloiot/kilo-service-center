/**
 * DL RX status of an end point (BSSCI §3.15): the statuses it reported through
 * a base station, the queries still waiting for an answer, and a new query.
 */

import React from "react";

import type { DlRxStatusDTO, DlRxStatusQueryDTO } from "@api-types/api";
import {
  useDlRxStatuses,
  useDlRxStatusQueries,
  useQueryDlRxStatus,
} from "@hooks";
import {
  Box,
  Button,
  List,
  ListItem,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import type { Theme } from "@mui/material/styles";

import { useFeedback } from "@contexts/feedback";
import { formatDateTime, formatUnixNanos } from "@utils/date-format";
import { getErrorMessage } from "@utils/error-message";
import { formatEui } from "@utils/eui";
import { formatMeasurement } from "@utils/formatters";
import { DL_RX_QUERY_STATUS } from "@constants/app";
import { DL_RX_STATUS, SCACI_FIELD } from "@constants/messages";

const MONO_CELL = (theme: Theme) => ({
  fontFamily: theme.typography.monoFontFamily,
  fontSize: theme.typography.caption.fontSize,
});

function StatusTable({ statuses }: { statuses: DlRxStatusDTO[] }) {
  if (statuses.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ py: 1 }}>
        {DL_RX_STATUS.MSG_EMPTY_STATUSES}
      </Typography>
    );
  }
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>{DL_RX_STATUS.LABEL_RX_TIME}</TableCell>
            <TableCell>{DL_RX_STATUS.LABEL_PACKET_CNT}</TableCell>
            <TableCell>{DL_RX_STATUS.LABEL_SNR}</TableCell>
            <TableCell>{DL_RX_STATUS.LABEL_RSSI}</TableCell>
            <TableCell>{SCACI_FIELD.BS_EUI}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {statuses.map((status, idx) => (
            <TableRow key={`${status.rxTime}-${status.bsEui}-${idx}`}>
              <TableCell>{formatUnixNanos(status.rxTime)}</TableCell>
              <TableCell>{status.packetCnt}</TableCell>
              <TableCell>{formatMeasurement(status.dlRxSnr)}</TableCell>
              <TableCell>{formatMeasurement(status.dlRxRssi)}</TableCell>
              <TableCell sx={MONO_CELL}>{formatEui(status.bsEui)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}

function PendingQueries({ queries }: { queries: DlRxStatusQueryDTO[] }) {
  const pending = queries.filter(
    (query) => query.status === DL_RX_QUERY_STATUS.PENDING,
  );
  return (
    <Box>
      <Typography variant="subtitle2">
        {DL_RX_STATUS.SECTION_PENDING}
      </Typography>
      {pending.length === 0 ? (
        <Typography variant="body2" color="text.secondary" sx={{ py: 1 }}>
          {DL_RX_STATUS.MSG_EMPTY_PENDING}
        </Typography>
      ) : (
        <List dense aria-label={DL_RX_STATUS.SECTION_PENDING}>
          {pending.map((query, idx) => (
            <ListItem
              key={`${query.opId}-${idx}`}
              disableGutters
              sx={{ gap: 2, flexWrap: "wrap" }}
            >
              <Typography variant="body2" color="text.secondary">
                {DL_RX_STATUS.LABEL_REQUESTED_AT}
              </Typography>
              <Typography variant="body2">
                {formatDateTime(query.requestedAt)}
              </Typography>
              <Typography variant="body2" color="text.secondary">
                {SCACI_FIELD.BS_EUI}
              </Typography>
              <Typography variant="body2" sx={MONO_CELL}>
                {formatEui(query.bsEui)}
              </Typography>
            </ListItem>
          ))}
        </List>
      )}
    </Box>
  );
}

interface DlRxStatusPanelProps {
  epEui: string;
}

const DlRxStatusPanel: React.FC<DlRxStatusPanelProps> = ({ epEui }) => {
  const statusesQuery = useDlRxStatuses(epEui);
  const queriesQuery = useDlRxStatusQueries(epEui);
  const queryMutation = useQueryDlRxStatus(epEui);
  const feedback = useFeedback();

  const sendQuery = () =>
    queryMutation.mutate(undefined, {
      onSuccess: (answer) =>
        answer.queryInitiated
          ? feedback.success(answer.message)
          : feedback.warning(answer.message),
      onError: (error) =>
        feedback.error(getErrorMessage(error, DL_RX_STATUS.ERR_QUERY)),
    });

  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
      <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
        <Typography variant="subtitle2">{DL_RX_STATUS.SECTION}</Typography>
        <Box sx={{ flex: 1 }} />
        <Button
          size="small"
          variant="outlined"
          onClick={sendQuery}
          disabled={queryMutation.isPending}
        >
          {queryMutation.isPending
            ? DL_RX_STATUS.ACTION_QUERYING
            : DL_RX_STATUS.ACTION_QUERY}
        </Button>
      </Box>
      <Typography variant="body2" color="text.secondary">
        {DL_RX_STATUS.HELPER}
      </Typography>
      <StatusTable statuses={statusesQuery.data?.statuses ?? []} />
      <PendingQueries queries={queriesQuery.data?.queries ?? []} />
    </Box>
  );
};

export default DlRxStatusPanel;
