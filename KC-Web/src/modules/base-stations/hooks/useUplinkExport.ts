import type { ActivityFilter } from "@api-types/api";
import { useExportBaseStationMessages } from "@hooks";

import { downloadBlob } from "@utils/downloadBlob";
import type { UplinkExportFormat } from "@constants/app";
import { BASE_STATION_MESSAGES } from "@constants/messages";

import { isEmptyExport, uplinkExportFilename } from "../utils/uplink-export";

/**
 * Saves a base station's uplinks in the activity filter's range as a file;
 * an export without uplinks or a failed one is reported through onError.
 */
export function useUplinkExport(
  bsEui: string,
  filter: ActivityFilter,
  onError: (message: string) => void,
) {
  const exportMessages = useExportBaseStationMessages();

  const exportAs = async (format: UplinkExportFormat) => {
    try {
      const blob = await exportMessages.mutateAsync({ bsEui, filter, format });
      const text = await blob.text();
      if (isEmptyExport(text, format)) {
        onError(BASE_STATION_MESSAGES.ERR_EXPORT_NO_DATA);
        return;
      }
      downloadBlob(
        new Blob([text], { type: blob.type }),
        uplinkExportFilename(bsEui, format, new Date()),
      );
    } catch {
      onError(BASE_STATION_MESSAGES.ERR_EXPORT_FAILED);
    }
  };

  return { exportAs, isExporting: exportMessages.isPending };
}
