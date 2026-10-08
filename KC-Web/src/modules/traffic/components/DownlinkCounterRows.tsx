/** Counter-dependent payloads (SCACI §3.10.1): one payload per packet counter. */

import { Box, Button, IconButton, TextField, Typography } from "@mui/material";

import { isValidHexString } from "@utils/formatters";
import {
  HELPER_DL_CNT_DEPEND_PAYLOADS,
  HELPER_DL_PACKET_CNT,
  LABEL_DL_ADD_PAYLOAD_ROW,
  LABEL_DL_PAYLOAD,
  LABEL_DL_REMOVE_PAYLOAD_ROW,
  VAL_PACKET_CNT_RANGE,
} from "@constants/messages";
import { AddIcon, DeleteIcon } from "@theme/icons";

import type { DownlinkForm } from "../hooks";
import { isValidPacketCnt, type PayloadRow } from "../utils/downlink-form";

interface CounterRowProps {
  form: DownlinkForm;
  row: PayloadRow;
  index: number;
  removable: boolean;
}

function CounterRow({ form, row, index, removable }: CounterRowProps) {
  const packetCntInvalid =
    row.packetCnt.length > 0 && !isValidPacketCnt(row.packetCnt);
  return (
    <Box sx={{ display: "flex", gap: 1, alignItems: "flex-start" }}>
      <TextField
        label={LABEL_DL_PAYLOAD}
        value={row.payload}
        onChange={(e) => form.updateRow(index, { payload: e.target.value })}
        error={row.payload.length > 0 && !isValidHexString(row.payload)}
        size="small"
        sx={{ flex: 2 }}
      />
      <TextField
        label={HELPER_DL_PACKET_CNT}
        value={row.packetCnt}
        onChange={(e) => form.updateRow(index, { packetCnt: e.target.value })}
        error={packetCntInvalid}
        helperText={packetCntInvalid ? VAL_PACKET_CNT_RANGE : undefined}
        type="number"
        size="small"
        sx={{ flex: 1 }}
      />
      {removable && (
        <IconButton
          size="small"
          onClick={() => form.removeRow(index)}
          aria-label={LABEL_DL_REMOVE_PAYLOAD_ROW}
        >
          <DeleteIcon fontSize="small" />
        </IconButton>
      )}
    </Box>
  );
}

export function DownlinkCounterRows({ form }: { form: DownlinkForm }) {
  const { payloadRows } = form;
  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
      <Typography variant="caption" color="text.secondary">
        {HELPER_DL_CNT_DEPEND_PAYLOADS}
      </Typography>
      {payloadRows.map((row, idx) => (
        <CounterRow
          key={idx}
          form={form}
          row={row}
          index={idx}
          removable={payloadRows.length > 1}
        />
      ))}
      <Button
        size="small"
        startIcon={<AddIcon />}
        onClick={form.addRow}
        sx={{ alignSelf: "flex-start" }}
      >
        {LABEL_DL_ADD_PAYLOAD_ROW}
      </Button>
    </Box>
  );
}
