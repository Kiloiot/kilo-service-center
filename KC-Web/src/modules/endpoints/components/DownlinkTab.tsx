import { useCallback } from "react";

import { useSendDownlink } from "@hooks";
import { Box } from "@mui/material";

import { DownlinkComposer, useDownlinkForm } from "@modules/traffic";
import { useFeedback } from "@contexts/feedback";
import { getErrorMessage } from "@utils/error-message";
import { MSG_DOWNLINK_QUEUED } from "@constants/messages";

import DlRxStatusPanel from "./DlRxStatusPanel";

interface DownlinkTabProps {
  epEui: string;
  lastPacketCnt?: number;
}

export function DownlinkTab({ epEui, lastPacketCnt }: DownlinkTabProps) {
  const form = useDownlinkForm();
  const sendMutation = useSendDownlink();
  const feedback = useFeedback();

  const handleSend = useCallback(() => {
    sendMutation.mutate(
      { epEui, ...form.buildContent() },
      {
        onSuccess: () => {
          feedback.success(MSG_DOWNLINK_QUEUED);
          form.reset();
        },
      },
    );
  }, [epEui, feedback, form, sendMutation]);

  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
      <DownlinkComposer
        form={form}
        isSending={sendMutation.isPending}
        lastPacketCnt={lastPacketCnt}
        sendError={
          sendMutation.isError ? getErrorMessage(sendMutation.error) : undefined
        }
        onSend={handleSend}
      />

      <DlRxStatusPanel epEui={epEui} />
    </Box>
  );
}
