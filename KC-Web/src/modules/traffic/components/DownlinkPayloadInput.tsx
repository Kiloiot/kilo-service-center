/** The single-shot dlDataQue payload; empty sends an ACK-only downlink. */

import { TextField } from "@mui/material";

import { HELPER_DL_PAYLOAD_ACK, LABEL_DL_PAYLOAD } from "@constants/messages";

import type { DownlinkForm } from "../hooks";

export function DownlinkPayloadInput({ form }: { form: DownlinkForm }) {
  const { values, setField, validation } = form;
  return (
    <TextField
      label={LABEL_DL_PAYLOAD}
      value={values.payload}
      onChange={(e) => setField("payload", e.target.value)}
      error={!!validation.payloadError}
      helperText={validation.payloadError || HELPER_DL_PAYLOAD_ACK}
      size="small"
      fullWidth
    />
  );
}
