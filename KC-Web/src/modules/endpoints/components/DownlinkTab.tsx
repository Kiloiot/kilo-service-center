import { useCallback, useState } from "react";

import type { SCACIDownlinkQueueDTO } from "@api-types/api";
import {
  useDownlinkQueue,
  useDownlinkResults,
  useFlushDownlinkQueue,
  useRevokeDownlink,
  useSendDownlink,
} from "@hooks";
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  IconButton,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from "@mui/material";

import { isValidHexString } from "@utils/formatters";
import {
  formatDateTime,
  formatDownlinkPriority,
  formatDownlinkQueueStatus,
  formatDownlinkResult,
} from "@utils/formatters";
import {
  DOWNLINK_PRIORITY_PRESETS,
  MIOTY_UINT32_MAX,
  PAGINATION,
  REVOCABLE_QUEUE_STATUSES,
  UI_TIMING,
} from "@constants/app";
import {
  ACTION_CANCEL,
  ACTION_FLUSH_QUEUE,
  ACTION_FLUSHING_QUEUE,
  ACTION_LOAD_MORE,
  ACTION_REVOKE,
  ACTION_REVOKING,
  ACTION_SEND_DOWNLINK,
  ACTION_SENDING_DOWNLINK,
  CONFIRM_FLUSH_QUEUE,
  CONFIRM_REVOKE_DOWNLINK,
  HELPER_DL_CNT_DEPEND,
  HELPER_DL_CNT_DEPEND_PAYLOADS,
  HELPER_DL_FORMAT,
  HELPER_DL_PACKET_CNT,
  HELPER_DL_PAYLOAD_ACK,
  LABEL_DL_ADD_PAYLOAD_ROW,
  LABEL_DL_ADVANCED_OPTIONS,
  LABEL_DL_BS_EUI,
  LABEL_DL_CNT_DEPEND,
  LABEL_DL_CREATED_AT,
  LABEL_DL_EXP_ONLY,
  LABEL_DL_FORMAT,
  LABEL_DL_PAYLOAD,
  LABEL_DL_PRIORITY,
  LABEL_DL_QUEUE_ID,
  LABEL_DL_REMOVE_PAYLOAD_ROW,
  LABEL_DL_RESPONSE_EXP,
  LABEL_DL_RESPONSE_PRIO,
  LABEL_DL_RESULT,
  LABEL_DL_RX_STAT_QRY,
  LABEL_DL_STATUS,
  LABEL_DL_TRANSMITTED_AT,
  LABEL_DL_WIND_REQ,
  LOADER,
  MSG_DOWNLINK_QUEUED,
  MSG_DOWNLINK_REVOKED,
  MSG_EMPTY_QUEUE,
  MSG_EMPTY_RESULTS,
  SECTION_COMPOSE_DOWNLINK,
  SECTION_DOWNLINK_QUEUE,
  SECTION_DOWNLINK_RESULTS,
  TABLE_HEADERS,
  VAL_FORMAT_RANGE,
  VAL_INVALID_HEX_STRING,
  VAL_PACKET_CNT_RANGE,
  VAL_PRIORITY_RANGE,
} from "@constants/messages";
import {
  AddIcon,
  DeleteIcon,
  ExpandMoreIcon,
  SendIcon,
} from "@theme/icons";

interface DownlinkTabProps {
  epEui: string;
}

interface PayloadRow {
  payload: string;
  packetCnt: string;
}

/**
 * Strict uint32 validation for packet counter values.
 * Rejects floats, negatives, and values exceeding MIOTY_UINT32_MAX (4294967295).
 */
const isValidPacketCnt = (value: string): boolean => {
  const trimmed = value.trim();
  if (trimmed === "" || !/^\d+$/.test(trimmed)) return false;
  const num = Number(trimmed);
  return Number.isInteger(num) && num >= 0 && num <= MIOTY_UINT32_MAX;
};

interface DownlinkFormState {
  payload: string;
  format: string;
  priority: string;
  cntDepend: boolean;
  payloadRows: PayloadRow[];
}

interface DownlinkFormValidation {
  payloadError: string;
  formatError: string;
  priorityError: string;
  cntDependRowsValid: boolean;
  formatNum: number;
  priorityNum: number;
}

/**
 * Validates the downlink form. Format must be an integer 0-255, priority
 * a float 0.0-1.0, payload a valid hex string when single-shot, and every
 * row a valid hex+counter pair when in counter-dependent mode.
 */
function validateDownlinkForm(
  state: DownlinkFormState,
): DownlinkFormValidation {
  const formatNum = parseInt(state.format, 10);
  const priorityNum = parseFloat(state.priority);
  const payloadError =
    !state.cntDepend &&
    state.payload.length > 0 &&
    !isValidHexString(state.payload)
      ? VAL_INVALID_HEX_STRING
      : "";
  const formatError =
    isNaN(formatNum) || formatNum < 0 || formatNum > 255
      ? VAL_FORMAT_RANGE
      : "";
  const priorityError =
    isNaN(priorityNum) || priorityNum < 0.0 || priorityNum > 1.0
      ? VAL_PRIORITY_RANGE
      : "";
  const cntDependRowsValid = state.cntDepend
    ? state.payloadRows.length >= 1 &&
      state.payloadRows.every(
        (r) =>
          r.payload.length > 0 &&
          isValidHexString(r.payload) &&
          isValidPacketCnt(r.packetCnt),
      )
    : true;
  return {
    payloadError,
    formatError,
    priorityError,
    cntDependRowsValid,
    formatNum,
    priorityNum,
  };
}

interface BuildSendDownlinkInput {
  epEui: string;
  state: DownlinkFormState;
  formatNum: number;
  priorityNum: number;
  responseExp: boolean;
  responsePrio: boolean;
  dlWindReq: boolean;
  expOnly: boolean;
  dlRxStatQry: boolean;
}

/**
 * Builds the send-downlink mutation argument. In counter-dependent mode
 * payloads are per-row + packetCnt is materialized from the row counters;
 * otherwise a single-payload (or empty) call.
 */
function buildSendDownlinkPayload(
  input: BuildSendDownlinkInput,
): Parameters<ReturnType<typeof useSendDownlink>["mutate"]>[0] {
  const { state } = input;
  const payloads = state.cntDepend
    ? state.payloadRows.map((r) => r.payload)
    : state.payload.length > 0
      ? [state.payload]
      : [];
  const packetCnt = state.cntDepend
    ? state.payloadRows.map((r) => parseInt(r.packetCnt, 10))
    : undefined;
  return {
    epEui: input.epEui,
    payloads,
    priority: input.priorityNum,
    cntDepend: state.cntDepend,
    packetCnt,
    format: input.formatNum,
    responseExp: input.responseExp,
    responsePrio: input.responsePrio,
    dlWindReq: input.dlWindReq,
    expOnly: input.expOnly,
    dlRxStatQry: input.dlRxStatQry,
  };
}

export function DownlinkTab({ epEui }: DownlinkTabProps) {
  // Compose form state
  const [payload, setPayload] = useState("");
  const [format, setFormat] = useState("0");
  const [priority, setPriority] = useState("0.5");
  const [cntDepend, setCntDepend] = useState(false);
  const [responseExp, setResponseExp] = useState(false);
  const [responsePrio, setResponsePrio] = useState(false);
  const [dlWindReq, setDlWindReq] = useState(false);
  const [expOnly, setExpOnly] = useState(false);
  const [dlRxStatQry, setDlRxStatQry] = useState(false);
  const [payloadRows, setPayloadRows] = useState<PayloadRow[]>([
    { payload: "", packetCnt: "0" },
  ]);
  const [successMsg, setSuccessMsg] = useState("");

  // Confirmation dialogs
  const [revokeTarget, setRevokeTarget] =
    useState<SCACIDownlinkQueueDTO | null>(null);
  const [flushDialogOpen, setFlushDialogOpen] = useState(false);

  // Mutations
  const sendMutation = useSendDownlink();
  const revokeMutation = useRevokeDownlink();
  const flushMutation = useFlushDownlinkQueue();

  // Queries
  const queueQuery = useDownlinkQueue(
    epEui,
    PAGINATION.DOWNLINK_QUEUE_PAGE_SIZE,
  );
  const resultsQuery = useDownlinkResults(
    epEui,
    PAGINATION.DOWNLINK_RESULTS_PAGE_SIZE,
  );

  // Flatten infinite query pages
  const queueMessages = queueQuery.data?.pages.flatMap((p) => p.messages) ?? [];
  const resultMessages =
    resultsQuery.data?.pages.flatMap((p) => p.results) ?? [];
  const queueTotalCount = queueQuery.data?.pages[0]?.totalCount ?? 0;
  const resultsTotalCount = resultsQuery.data?.pages[0]?.totalCount ?? 0;

  // Validation
  const {
    payloadError,
    formatError,
    priorityError,
    cntDependRowsValid,
    formatNum,
    priorityNum,
  } = validateDownlinkForm({
    payload,
    format,
    priority,
    cntDepend,
    payloadRows,
  });

  const canSend =
    !sendMutation.isPending &&
    !formatError &&
    !priorityError &&
    !payloadError &&
    cntDependRowsValid;

  const handleSend = useCallback(() => {
    sendMutation.mutate(
      buildSendDownlinkPayload({
        epEui,
        state: { payload, format, priority, cntDepend, payloadRows },
        formatNum,
        priorityNum,
        responseExp,
        responsePrio,
        dlWindReq,
        expOnly,
        dlRxStatQry,
      }),
      {
        onSuccess: () => {
          setSuccessMsg(MSG_DOWNLINK_QUEUED);
          setTimeout(
            () => setSuccessMsg(""),
            UI_TIMING.SUCCESS_MESSAGE_DISMISS_MS,
          );
          setPayload("");
          setPayloadRows([{ payload: "", packetCnt: "0" }]);
        },
      },
    );
  }, [
    cntDepend,
    dlRxStatQry,
    dlWindReq,
    epEui,
    expOnly,
    format,
    formatNum,
    payload,
    payloadRows,
    priority,
    priorityNum,
    responseExp,
    responsePrio,
    sendMutation,
  ]);

  const handleRevoke = useCallback(() => {
    if (!revokeTarget) return;
    revokeMutation.mutate(
      { epEui, queueId: String(revokeTarget.queId) },
      {
        onSuccess: () => {
          setRevokeTarget(null);
          setSuccessMsg(MSG_DOWNLINK_REVOKED);
          setTimeout(
            () => setSuccessMsg(""),
            UI_TIMING.SUCCESS_MESSAGE_DISMISS_MS,
          );
        },
      },
    );
  }, [epEui, revokeTarget, revokeMutation]);

  const handleFlush = useCallback(() => {
    flushMutation.mutate(
      { epEui },
      {
        onSuccess: () => {
          setFlushDialogOpen(false);
        },
      },
    );
  }, [epEui, flushMutation]);

  const handleCntDependToggle = useCallback((checked: boolean) => {
    setCntDepend(checked);
    if (checked) {
      setPayloadRows([{ payload: "", packetCnt: "0" }]);
    } else {
      setPayload("");
    }
  }, []);

  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
      {/* Success message */}
      {successMsg && (
        <Typography color="success.main" variant="body2">
          {successMsg}
        </Typography>
      )}

      {/* ================================================================ */}
      {/* Compose Downlink Section */}
      {/* ================================================================ */}
      <Box>
        <Typography variant="subtitle2" gutterBottom>
          {SECTION_COMPOSE_DOWNLINK}
        </Typography>
        <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
          {/* Payload input */}
          {!cntDepend ? (
            <TextField
              label={LABEL_DL_PAYLOAD}
              value={payload}
              onChange={(e) => setPayload(e.target.value)}
              error={!!payloadError}
              helperText={payloadError || HELPER_DL_PAYLOAD_ACK}
              size="small"
              fullWidth
            />
          ) : (
            <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
              <Typography variant="caption" color="text.secondary">
                {HELPER_DL_CNT_DEPEND_PAYLOADS}
              </Typography>
              {payloadRows.map((row, idx) => (
                <Box
                  key={idx}
                  sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}
                >
                  <TextField
                    label={LABEL_DL_PAYLOAD}
                    value={row.payload}
                    onChange={(e) => {
                      const updated = [...payloadRows];
                      updated[idx] = {
                        ...updated[idx],
                        payload: e.target.value,
                      };
                      setPayloadRows(updated);
                    }}
                    error={
                      row.payload.length > 0 && !isValidHexString(row.payload)
                    }
                    size="small"
                    sx={{ flex: 2 }}
                  />
                  <TextField
                    label={HELPER_DL_PACKET_CNT}
                    value={row.packetCnt}
                    onChange={(e) => {
                      const updated = [...payloadRows];
                      updated[idx] = {
                        ...updated[idx],
                        packetCnt: e.target.value,
                      };
                      setPayloadRows(updated);
                    }}
                    error={
                      row.packetCnt.length > 0 &&
                      !isValidPacketCnt(row.packetCnt)
                    }
                    helperText={
                      row.packetCnt.length > 0 &&
                      !isValidPacketCnt(row.packetCnt)
                        ? VAL_PACKET_CNT_RANGE
                        : undefined
                    }
                    type="number"
                    size="small"
                    sx={{ flex: 1 }}
                  />
                  {payloadRows.length > 1 && (
                    <IconButton
                      size="small"
                      onClick={() =>
                        setPayloadRows(payloadRows.filter((_, i) => i !== idx))
                      }
                      aria-label={LABEL_DL_REMOVE_PAYLOAD_ROW}
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  )}
                </Box>
              ))}
              <Button
                size="small"
                startIcon={<AddIcon />}
                onClick={() =>
                  setPayloadRows([
                    ...payloadRows,
                    { payload: "", packetCnt: "0" },
                  ])
                }
                sx={{ alignSelf: "flex-start" }}
              >
                {LABEL_DL_ADD_PAYLOAD_ROW}
              </Button>
            </Box>
          )}

          {/* Common fields row */}
          <Box sx={{ display: "flex", gap: 2, flexWrap: "wrap" }}>
            <TextField
              label={LABEL_DL_FORMAT}
              value={format}
              onChange={(e) => setFormat(e.target.value)}
              error={!!formatError}
              helperText={formatError || HELPER_DL_FORMAT}
              type="number"
              size="small"
              sx={{ width: 150 }}
            />
            <TextField
              label={LABEL_DL_PRIORITY}
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
              error={!!priorityError}
              helperText={
                priorityError ||
                DOWNLINK_PRIORITY_PRESETS.map(
                  (p) => `${p.value}=${p.label}`,
                ).join(", ")
              }
              type="number"
              inputProps={{ step: 0.1, min: 0, max: 1 }}
              size="small"
              sx={{ width: 200 }}
            />
          </Box>

          {/* Toggle switches */}
          <Box
            sx={{
              display: "flex",
              gap: 3,
              flexWrap: "wrap",
              alignItems: "center",
            }}
          >
            <Box sx={{ display: "flex", alignItems: "center" }}>
              <Switch
                checked={cntDepend}
                onChange={(_, checked) => handleCntDependToggle(checked)}
                size="small"
              />
              <Tooltip title={HELPER_DL_CNT_DEPEND}>
                <Typography variant="body2">{LABEL_DL_CNT_DEPEND}</Typography>
              </Tooltip>
            </Box>
          </Box>

          {/* Advanced options */}
          <Accordion disableGutters sx={{ "&:before": { display: "none" } }}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="body2">
                {LABEL_DL_ADVANCED_OPTIONS}
              </Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Box sx={{ display: "flex", gap: 3, flexWrap: "wrap" }}>
                {[
                  {
                    label: LABEL_DL_RESPONSE_EXP,
                    value: responseExp,
                    setter: setResponseExp,
                  },
                  {
                    label: LABEL_DL_RESPONSE_PRIO,
                    value: responsePrio,
                    setter: setResponsePrio,
                  },
                  {
                    label: LABEL_DL_WIND_REQ,
                    value: dlWindReq,
                    setter: setDlWindReq,
                  },
                  {
                    label: LABEL_DL_EXP_ONLY,
                    value: expOnly,
                    setter: setExpOnly,
                  },
                  {
                    label: LABEL_DL_RX_STAT_QRY,
                    value: dlRxStatQry,
                    setter: setDlRxStatQry,
                  },
                ].map(({ label, value, setter }) => (
                  <Box
                    key={label}
                    sx={{ display: "flex", alignItems: "center" }}
                  >
                    <Switch
                      checked={value}
                      onChange={(_, checked) => setter(checked)}
                      size="small"
                    />
                    <Typography variant="body2">{label}</Typography>
                  </Box>
                ))}
              </Box>
            </AccordionDetails>
          </Accordion>

          {/* Send button */}
          <Box>
            <Button
              variant="contained"
              startIcon={<SendIcon />}
              onClick={handleSend}
              disabled={!canSend}
              size="small"
            >
              {sendMutation.isPending
                ? ACTION_SENDING_DOWNLINK
                : ACTION_SEND_DOWNLINK}
            </Button>
            {sendMutation.isError && (
              <Typography color="error" variant="caption" sx={{ ml: 2 }}>
                {sendMutation.error?.message}
              </Typography>
            )}
          </Box>
        </Box>
      </Box>

      {/* ================================================================ */}
      {/* Queue Table Section */}
      {/* ================================================================ */}
      <Box>
        <Box sx={{ display: "flex", alignItems: "center", gap: 1, mb: 1 }}>
          <Typography variant="subtitle2">{SECTION_DOWNLINK_QUEUE}</Typography>
          <Typography variant="caption" color="text.secondary">
            ({queueTotalCount})
          </Typography>
          <Box sx={{ flex: 1 }} />
          {queueMessages.some(
            (m) => m.status && REVOCABLE_QUEUE_STATUSES.has(m.status),
          ) && (
            <Button
              size="small"
              color="warning"
              variant="outlined"
              onClick={() => setFlushDialogOpen(true)}
              disabled={flushMutation.isPending}
            >
              {flushMutation.isPending
                ? ACTION_FLUSHING_QUEUE
                : ACTION_FLUSH_QUEUE}
            </Button>
          )}
        </Box>

        {queueQuery.isLoading ? (
          <Box display="flex" justifyContent="center" py={2}>
            <CircularProgress size={24} />
          </Box>
        ) : queueMessages.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>
            {MSG_EMPTY_QUEUE}
          </Typography>
        ) : (
          <>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>{LABEL_DL_QUEUE_ID}</TableCell>
                    <TableCell>{LABEL_DL_STATUS}</TableCell>
                    <TableCell>{LABEL_DL_PRIORITY}</TableCell>
                    <TableCell>{LABEL_DL_PAYLOAD}</TableCell>
                    <TableCell>{LABEL_DL_CREATED_AT}</TableCell>
                    <TableCell align="right">
                      {TABLE_HEADERS.COL_ACTIONS}
                    </TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {queueMessages.map((msg) => {
                    const statusDisplay = formatDownlinkQueueStatus(
                      msg.status ?? "",
                    );
                    const isRevocable = msg.status
                      ? REVOCABLE_QUEUE_STATUSES.has(msg.status)
                      : false;
                    return (
                      <TableRow key={msg.queId}>
                        <TableCell>{msg.queId}</TableCell>
                        <TableCell>
                          <Chip
                            label={statusDisplay.label}
                            color={statusDisplay.color}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>
                          {formatDownlinkPriority(msg.priority)}
                        </TableCell>
                        <TableCell
                          sx={{
                            maxWidth: 200,
                            overflow: "hidden",
                            textOverflow: "ellipsis",
                          }}
                        >
                          <Typography
                            variant="body2"
                            noWrap
                            sx={{
                              fontFamily: "monospace",
                              fontSize: "0.75rem",
                            }}
                          >
                            {msg.payload || "(empty)"}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          {msg.createdAt ? formatDateTime(msg.createdAt) : ""}
                        </TableCell>
                        <TableCell align="right">
                          {isRevocable && (
                            <Tooltip title={ACTION_REVOKE}>
                              <IconButton
                                size="small"
                                color="warning"
                                onClick={() => setRevokeTarget(msg)}
                              >
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          )}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </TableContainer>
            {queueQuery.hasNextPage && (
              <Box sx={{ display: "flex", justifyContent: "center", mt: 1 }}>
                <Button
                  size="small"
                  onClick={() => queueQuery.fetchNextPage()}
                  disabled={queueQuery.isFetchingNextPage}
                >
                  {queueQuery.isFetchingNextPage
                    ? LOADER.LOADING
                    : ACTION_LOAD_MORE}
                </Button>
              </Box>
            )}
          </>
        )}
      </Box>

      {/* ================================================================ */}
      {/* Results Table Section */}
      {/* ================================================================ */}
      <Box>
        <Box sx={{ display: "flex", alignItems: "center", gap: 1, mb: 1 }}>
          <Typography variant="subtitle2">
            {SECTION_DOWNLINK_RESULTS}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            ({resultsTotalCount})
          </Typography>
          <Box sx={{ flex: 1 }} />
        </Box>

        {resultsQuery.isLoading ? (
          <Box display="flex" justifyContent="center" py={2}>
            <CircularProgress size={24} />
          </Box>
        ) : resultMessages.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>
            {MSG_EMPTY_RESULTS}
          </Typography>
        ) : (
          <>
            <TableContainer>
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>{LABEL_DL_QUEUE_ID}</TableCell>
                    <TableCell>{LABEL_DL_RESULT}</TableCell>
                    <TableCell>{LABEL_DL_STATUS}</TableCell>
                    <TableCell>{LABEL_DL_BS_EUI}</TableCell>
                    <TableCell>{LABEL_DL_TRANSMITTED_AT}</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {resultMessages.map((msg, idx) => {
                    const resultDisplay = formatDownlinkResult(
                      msg.result ?? "",
                    );
                    const statusDisplay = formatDownlinkQueueStatus(
                      msg.status ?? "",
                    );
                    return (
                      <TableRow key={`${msg.queId}-${idx}`}>
                        <TableCell>{msg.queId}</TableCell>
                        <TableCell>
                          <Chip
                            label={resultDisplay.label}
                            color={resultDisplay.color}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={statusDisplay.label}
                            color={statusDisplay.color}
                            size="small"
                            variant="outlined"
                          />
                        </TableCell>
                        <TableCell
                          sx={{ fontFamily: "monospace", fontSize: "0.75rem" }}
                        >
                          {msg.bsEui || "-"}
                        </TableCell>
                        <TableCell>
                          {msg.transmittedAt
                            ? formatDateTime(msg.transmittedAt)
                            : "-"}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </TableContainer>
            {resultsQuery.hasNextPage && (
              <Box sx={{ display: "flex", justifyContent: "center", mt: 1 }}>
                <Button
                  size="small"
                  onClick={() => resultsQuery.fetchNextPage()}
                  disabled={resultsQuery.isFetchingNextPage}
                >
                  {resultsQuery.isFetchingNextPage
                    ? LOADER.LOADING
                    : ACTION_LOAD_MORE}
                </Button>
              </Box>
            )}
          </>
        )}
      </Box>

      {/* Revoke confirmation dialog */}
      <Dialog open={!!revokeTarget} onClose={() => setRevokeTarget(null)}>
        <DialogTitle>{ACTION_REVOKE}</DialogTitle>
        <DialogContent>
          <DialogContentText>{CONFIRM_REVOKE_DOWNLINK}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRevokeTarget(null)}>{ACTION_CANCEL}</Button>
          <Button
            color="warning"
            onClick={handleRevoke}
            disabled={revokeMutation.isPending}
          >
            {revokeMutation.isPending ? ACTION_REVOKING : ACTION_REVOKE}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Flush confirmation dialog */}
      <Dialog open={flushDialogOpen} onClose={() => setFlushDialogOpen(false)}>
        <DialogTitle>{ACTION_FLUSH_QUEUE}</DialogTitle>
        <DialogContent>
          <DialogContentText>{CONFIRM_FLUSH_QUEUE}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setFlushDialogOpen(false)}>
            {ACTION_CANCEL}
          </Button>
          <Button
            color="warning"
            onClick={handleFlush}
            disabled={flushMutation.isPending}
          >
            {flushMutation.isPending
              ? ACTION_FLUSHING_QUEUE
              : ACTION_FLUSH_QUEUE}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
