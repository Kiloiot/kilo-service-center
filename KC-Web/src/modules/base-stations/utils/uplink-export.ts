/**
 * Rules of the base station uplink export: when it holds no uplink and what
 * the saved file is called.
 */

import { formatFileTimestamp } from "@utils/date-format";
import { UPLINK_EXPORT_FORMAT, type UplinkExportFormat } from "@constants/app";
import { BASE_STATION_MESSAGES } from "@constants/messages";

const CSV_LINE_BREAK = /\r?\n/;

/** An export holds no uplink: a CSV of its header only, a JSON empty array. */
export function isEmptyExport(
  text: string,
  format: UplinkExportFormat,
): boolean {
  if (format === UPLINK_EXPORT_FORMAT.JSON) {
    const rows: unknown = JSON.parse(text);
    return Array.isArray(rows) && rows.length === 0;
  }
  return text.trim().split(CSV_LINE_BREAK).length <= 1;
}

/** The file name of an export, safe on every file system. */
export function uplinkExportFilename(
  bsEui: string,
  format: UplinkExportFormat,
  at: Date,
): string {
  return `${BASE_STATION_MESSAGES.EXPORT_FILENAME_PREFIX}${bsEui}${BASE_STATION_MESSAGES.EXPORT_FILENAME_SUFFIX}${formatFileTimestamp(at)}.${format}`;
}
