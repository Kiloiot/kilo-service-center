/**
 * Detail action that asks the base station for its status now and reports
 * whether the request reached it.
 */

import { useRequestBaseStationStatus } from "@hooks";
import { CircularProgress, IconButton, Tooltip } from "@mui/material";

import { useFeedback } from "@contexts/feedback";
import { BASE_STATION_DETAILS } from "@constants/messages";
import { RequestStatusIcon } from "@theme/icons";
import { componentSpacing } from "@theme/index";

export function BaseStationStatusRequestButton({ bsEui }: { bsEui: string }) {
  const request = useRequestBaseStationStatus();
  const feedback = useFeedback();

  const handleClick = () =>
    request.mutate(bsEui, {
      onSuccess: (result) =>
        result.success
          ? feedback.success(BASE_STATION_DETAILS.STATUS_REQUESTED)
          : feedback.warning(result.message),
      onError: () => feedback.error(BASE_STATION_DETAILS.STATUS_REQUEST_FAILED),
    });

  return (
    <Tooltip title={BASE_STATION_DETAILS.ACTION_REQUEST_STATUS}>
      <span>
        <IconButton
          size="small"
          aria-label={BASE_STATION_DETAILS.ACTION_REQUEST_STATUS}
          onClick={handleClick}
          disabled={request.isPending}
        >
          {request.isPending ? (
            <CircularProgress size={componentSpacing.spinner.buttonSmall} />
          ) : (
            <RequestStatusIcon />
          )}
        </IconButton>
      </span>
    </Tooltip>
  );
}
