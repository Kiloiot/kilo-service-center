/**
 * Who performed an audited action: the user's email, the user id when the
 * backend resolved no email, or the Service Center for a service-raised event.
 */

import type { EventLogEntryUI } from "@api-types/api";
import { Tooltip } from "@mui/material";

import { MonoText } from "@components/common/MonoText";
import { EVENT_LOG_TABLE } from "@constants/messages";

// The backend resolves an email only for a user of the viewer's own tenant; any other actor stays an id.
export function ActorCell({ entry }: { entry: EventLogEntryUI }) {
  if (entry.userEmail) {
    return (
      <Tooltip title={entry.userId}>
        <span>{entry.userEmail}</span>
      </Tooltip>
    );
  }
  if (entry.userId) return <MonoText>{entry.userId}</MonoText>;
  return <>{EVENT_LOG_TABLE.ACTOR_SERVICE}</>;
}
