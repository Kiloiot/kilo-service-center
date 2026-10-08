/**
 * Create Downlink (dlDataQue): the downlink fields and the send button.
 */

import { Box, Button, Typography } from "@mui/material";

import {
  ACTION_SEND_DOWNLINK,
  ACTION_SENDING_DOWNLINK,
  LABEL_DL_LAST_PACKET_CNT,
  MSG_DL_NO_PACKET_CNT,
  SECTION_COMPOSE_DOWNLINK,
} from "@constants/messages";
import { SendIcon } from "@theme/icons";

import type { DownlinkForm } from "../hooks";
import { DownlinkFormFields } from "./DownlinkFormFields";

interface DownlinkComposerProps {
  form: DownlinkForm;
  isSending: boolean;
  lastPacketCnt?: number;
  sendError?: string;
  onSend: () => void;
}

export function DownlinkComposer({
  form,
  isSending,
  lastPacketCnt,
  sendError,
  onSend,
}: DownlinkComposerProps) {
  return (
    <Box>
      <Typography variant="subtitle2" gutterBottom>
        {SECTION_COMPOSE_DOWNLINK}
      </Typography>
      <Typography
        variant="caption"
        color="text.secondary"
        component="p"
        sx={{ mb: 1 }}
      >
        {`${LABEL_DL_LAST_PACKET_CNT} ${lastPacketCnt ?? MSG_DL_NO_PACKET_CNT}`}
      </Typography>
      <DownlinkFormFields form={form} />
      <Box sx={{ mt: 2 }}>
        <Button
          variant="contained"
          startIcon={<SendIcon />}
          onClick={onSend}
          disabled={isSending || !form.isValid}
          size="small"
        >
          {isSending ? ACTION_SENDING_DOWNLINK : ACTION_SEND_DOWNLINK}
        </Button>
        {sendError && (
          <Typography color="error" variant="caption" sx={{ ml: 2 }}>
            {sendError}
          </Typography>
        )}
      </Box>
    </Box>
  );
}
