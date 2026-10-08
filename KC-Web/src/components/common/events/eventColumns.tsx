/**
 * The Events Log column set (Time | Event type | opId | Scope | Severity |
 * Summary) and the cells that render a system event in it. Any table that
 * reads like the Events Log builds its columns here from its own cells.
 */

import type { ReactNode } from "react";

import type { EventLogEntryUI } from "@api-types/api";

import type { DataTableColumn } from "@components/common/DataTable";
import { MonoText } from "@components/common/MonoText";
import { TimeText } from "@components/common/TimeText";
import { formatDateTime } from "@utils/date-format";
import { formatOptional } from "@utils/formatters";
import { EVENT_LOG_TABLE, SCACI_FIELD } from "@constants/messages";

import { SeverityChip, SummaryCell } from "./EventCells";

export type EventColumnId =
  | "time"
  | "eventType"
  | "opId"
  | "source"
  | "severity"
  | "summary";

const EVENT_COLUMN_ORDER: readonly EventColumnId[] = [
  "time",
  "eventType",
  "opId",
  "source",
  "severity",
  "summary",
];

const EVENT_COLUMN_HEADERS: Record<EventColumnId, string> = {
  time: EVENT_LOG_TABLE.COL_TIME,
  eventType: EVENT_LOG_TABLE.COL_OPERATION,
  opId: SCACI_FIELD.OP_ID,
  source: EVENT_LOG_TABLE.COL_SOURCE,
  severity: EVENT_LOG_TABLE.COL_OUTCOME,
  summary: EVENT_LOG_TABLE.COL_SUMMARY,
};

export type EventCells<T> = Partial<
  Record<EventColumnId, (row: T) => ReactNode>
>;

export const timeCell = (row: { timestamp?: string }) => (
  <TimeText>{formatDateTime(row.timestamp)}</TimeText>
);

export const scopeCell = (row: { scope?: string }) => (
  <MonoText>{formatOptional(row.scope)}</MonoText>
);

export const severityCell = (row: { severity: string }) => (
  <SeverityChip severity={row.severity} />
);

export const summaryCell = (row: { title: string; description?: string }) => (
  <SummaryCell title={row.title} caption={row.description} />
);

export const opIdCell = (row: { opId?: string }) => (
  <MonoText>{formatOptional(row.opId)}</MonoText>
);

/** How the Events Log renders one system event. */
export const EVENT_CELLS: Required<EventCells<EventLogEntryUI>> = {
  time: timeCell,
  eventType: (e) => <MonoText>{e.eventType}</MonoText>,
  opId: opIdCell,
  source: scopeCell,
  severity: severityCell,
  summary: summaryCell,
};

/** The Events Log columns a table has cells for, in the Events Log order. */
export function eventColumns<T>(cells: EventCells<T>): DataTableColumn<T>[] {
  return EVENT_COLUMN_ORDER.flatMap((id) => {
    const render = cells[id];
    return render ? [{ id, header: EVENT_COLUMN_HEADERS[id], render }] : [];
  });
}
