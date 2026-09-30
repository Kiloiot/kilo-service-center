/**
 * Columns of the Audit Log and Errors Center tables (NAV_STRUCTURE §5); the
 * Events Log columns are the shared event columns.
 */

import type { ErrorGroupUI, EventLogEntryUI } from "@api-types/api";

import type { DataTableColumn } from "@components/common/DataTable";
import { ActorCell } from "@components/common/events/ActorCell";
import { EVENT_CELLS } from "@components/common/events/eventColumns";
import { MonoText } from "@components/common/MonoText";
import { TimeText } from "@components/common/TimeText";
import { formatDateTime } from "@utils/date-format";
import { formatOptional } from "@utils/formatters";
import {
  ERROR_GROUP_TABLE,
  EVENT_LOG_TABLE,
  SCACI_FIELD,
} from "@constants/messages";

type Entry = EventLogEntryUI;

export const AUDIT_COLUMNS: readonly DataTableColumn<Entry>[] = [
  { id: "time", header: EVENT_LOG_TABLE.COL_TIME, render: EVENT_CELLS.time },
  {
    id: "actor",
    header: EVENT_LOG_TABLE.COL_ACTOR,
    render: (e) => <ActorCell entry={e} />,
  },
  {
    id: "action",
    header: EVENT_LOG_TABLE.COL_ACTION,
    render: EVENT_CELLS.eventType,
  },
  {
    id: "target",
    header: EVENT_LOG_TABLE.COL_TARGET,
    render: EVENT_CELLS.source,
  },
  {
    id: "summary",
    header: EVENT_LOG_TABLE.COL_SUMMARY,
    render: EVENT_CELLS.summary,
  },
];

export const ERROR_GROUP_COLUMNS: readonly DataTableColumn<ErrorGroupUI>[] = [
  {
    id: "lastOpId",
    header: ERROR_GROUP_TABLE.COL_OP_ID,
    render: (g) => <MonoText>{formatOptional(g.lastOpId)}</MonoText>,
  },
  {
    id: "code",
    header: SCACI_FIELD.CODE,
    render: (g) => <MonoText>{formatOptional(g.code)}</MonoText>,
  },
  {
    id: "message",
    header: SCACI_FIELD.MESSAGE,
    render: (g) => formatOptional(g.message),
  },
  {
    id: "eventType",
    header: ERROR_GROUP_TABLE.COL_EVENT_TYPE,
    render: (g) => <MonoText>{g.eventType}</MonoText>,
  },
  {
    id: "related",
    header: ERROR_GROUP_TABLE.COL_RELATED,
    render: (g) => <MonoText>{formatOptional(g.sourceName)}</MonoText>,
  },
  {
    id: "firstSeen",
    header: ERROR_GROUP_TABLE.COL_FIRST_SEEN,
    render: (g) => <TimeText>{formatDateTime(g.firstSeen)}</TimeText>,
  },
  {
    id: "lastSeen",
    header: ERROR_GROUP_TABLE.COL_LAST_SEEN,
    render: (g) => <TimeText>{formatDateTime(g.lastSeen)}</TimeText>,
  },
  {
    id: "count",
    header: ERROR_GROUP_TABLE.COL_COUNT,
    align: "right",
    render: (g) => g.count,
  },
];
