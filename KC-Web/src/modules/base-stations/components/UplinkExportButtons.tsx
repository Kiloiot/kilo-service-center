/**
 * Saves the base station's uplinks in the Activity date range as CSV or
 * JSON; an export without uplinks or a failed one is reported as an error.
 */

import type { ActivityFilter } from "@api-types/api";
import { Box, Button } from "@mui/material";

import { useFeedback } from "@contexts/feedback";
import { UPLINK_EXPORT_FORMAT } from "@constants/app";
import { BASE_STATION_MESSAGES } from "@constants/messages";
import { DownloadIcon } from "@theme/icons";

import { useUplinkExport } from "../hooks";

interface UplinkExportButtonsProps {
  bsEui: string;
  filter: ActivityFilter;
}

export function UplinkExportButtons({
  bsEui,
  filter,
}: UplinkExportButtonsProps) {
  const feedback = useFeedback();
  const uplinkExport = useUplinkExport(bsEui, filter, feedback.error);

  return (
    <Box>
      <Button
        size="small"
        startIcon={<DownloadIcon />}
        onClick={() => uplinkExport.exportAs(UPLINK_EXPORT_FORMAT.CSV)}
        disabled={uplinkExport.isExporting}
      >
        {BASE_STATION_MESSAGES.EXPORT_CSV}
      </Button>
      <Button
        size="small"
        startIcon={<DownloadIcon />}
        onClick={() => uplinkExport.exportAs(UPLINK_EXPORT_FORMAT.JSON)}
        disabled={uplinkExport.isExporting}
      >
        {BASE_STATION_MESSAGES.EXPORT_JSON}
      </Button>
    </Box>
  );
}
