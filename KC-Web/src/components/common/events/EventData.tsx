/**
 * Expanded event row: the event's JSON data, or a note that it carries none.
 */

import type { EventLogEntryUI } from "@api-types/api";
import { Typography } from "@mui/material";

import { JsonPreview } from "@components/common/JsonPreview";
import { EVENT_LOG_TABLE } from "@constants/messages";

export function EventData({ entry }: { entry: EventLogEntryUI }) {
  if (!entry.data) {
    return (
      <Typography variant="body2" color="text.secondary">
        {EVENT_LOG_TABLE.NO_DATA}
      </Typography>
    );
  }
  return <JsonPreview title={EVENT_LOG_TABLE.DATA_TITLE} value={entry.data} />;
}
