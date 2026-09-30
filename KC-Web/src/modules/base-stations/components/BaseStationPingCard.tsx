import React from "react";

import { Button, Stack, Typography } from "@mui/material";

import { DataCard } from "@components/common/DataCard";
import { useFeedback } from "@contexts/feedback";
import { useInitiatePing } from "@hooks/useBaseStationOperations";
import { getErrorMessage } from "@utils/error-message";
import { BS_OPERATIONS } from "@constants/messages";
import { NetworkCheckIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

interface BaseStationPingCardProps {
  bsEui: string;
  /** Only a connected station has a BSSCI session to ping over. */
  online: boolean;
}

/** InitiatePing: queues a BSSCI ping (§5.4) and reports the operation it was sent under. */
export const BaseStationPingCard: React.FC<BaseStationPingCardProps> = ({
  bsEui,
  online,
}) => {
  const ping = useInitiatePing();
  const feedback = useFeedback();

  const sendPing = () =>
    ping.mutate(bsEui, {
      onSuccess: (result) =>
        result.success
          ? feedback.success(
              BS_OPERATIONS.PING_SENT.replace("{opId}", String(result.opId)),
            )
          : feedback.warning(result.message),
      onError: (error) =>
        feedback.error(getErrorMessage(error, BS_OPERATIONS.ERR_PING)),
    });

  return (
    <DataCard title={BS_OPERATIONS.ACTIONS_TITLE}>
      <Stack spacing={componentSpacing.infoGrid.rowGap}>
        <Typography variant="body2" color="text.secondary">
          {BS_OPERATIONS.PING_HINT}
        </Typography>
        <div>
          <Button
            variant="outlined"
            size="small"
            startIcon={<NetworkCheckIcon />}
            disabled={!online || ping.isPending}
            onClick={sendPing}
          >
            {ping.isPending
              ? BS_OPERATIONS.ACTION_PINGING
              : BS_OPERATIONS.ACTION_PING}
          </Button>
        </div>
      </Stack>
    </DataCard>
  );
};
