/**
 * How each kind of Activity row renders in the Events Log columns, its row
 * key and its expanded details. Every cell looks its renderer up by the row's
 * kind, so a new kind of row is one more entry per table below.
 */

import type { ReactNode } from "react";

import type {
  ActivityKind,
  ActivityRow,
  ActivityRowPayload,
} from "@api-types/api";
import type { KindHandlers } from "@api-types/tagged";

import type { DataTableColumn } from "@components/common/DataTable";
import { SummaryCell } from "@components/common/events/EventCells";
import {
  EVENT_CELLS,
  type EventColumnId,
  eventColumns,
  opIdCell,
} from "@components/common/events/eventColumns";
import { EventData } from "@components/common/events/EventData";
import { MonoText } from "@components/common/MonoText";
import { TimeText } from "@components/common/TimeText";
import { UplinkDetails } from "@components/common/uplinks/UplinkDetails";
import { byKind } from "@utils/by-kind";
import { formatUnixNanos } from "@utils/date-format";
import {
  formatUplinkCaption,
  formatUplinkDecode,
  formatUplinkTitle,
  type UplinkScope,
} from "@utils/uplink-summary";
import { MIOTY_COMMAND, ROW_KEY_SEPARATOR } from "@constants/app";

type RowCells<K extends ActivityKind> = Record<
  EventColumnId,
  (row: ActivityRow<K>) => ReactNode
>;

// An event row reads exactly like the Events Log.
const EVENT_ROW_CELLS: RowCells<"event"> = {
  time: ({ entry }) => EVENT_CELLS.time(entry),
  eventType: ({ entry }) => EVENT_CELLS.eventType(entry),
  opId: ({ entry }) => EVENT_CELLS.opId(entry),
  source: ({ entry }) => EVENT_CELLS.source(entry),
  severity: ({ entry }) => EVENT_CELLS.severity(entry),
  summary: ({ entry }) => EVENT_CELLS.summary(entry),
};

// An uplink carries no severity.
const uplinkRowCells = (scope: UplinkScope): RowCells<"uplink"> => ({
  time: ({ uplink }) => <TimeText>{formatUnixNanos(uplink.rxTime)}</TimeText>,
  eventType: () => <MonoText>{MIOTY_COMMAND.UL_DATA}</MonoText>,
  opId: ({ uplink }) => opIdCell(uplink),
  source: ({ uplink }) => <MonoText>{scope(uplink)}</MonoText>,
  severity: () => null,
  summary: ({ uplink }) => (
    <SummaryCell
      title={formatUplinkTitle(uplink)}
      caption={formatUplinkCaption(uplink)}
      detail={formatUplinkDecode(uplink)}
    />
  ),
});

/** The Events Log columns over events and uplinks; the scope names an uplink's far end. */
export function activityColumns(
  scope: UplinkScope,
): DataTableColumn<ActivityRow>[] {
  const uplinkCells = uplinkRowCells(scope);
  const cell = (id: EventColumnId) => {
    const byRowKind: KindHandlers<ActivityRowPayload, ReactNode> = {
      event: EVENT_ROW_CELLS[id],
      uplink: uplinkCells[id],
    };
    return (row: ActivityRow): ReactNode => byKind(row, byRowKind);
  };
  return eventColumns<ActivityRow>({
    time: cell("time"),
    eventType: cell("eventType"),
    opId: cell("opId"),
    source: cell("source"),
    severity: cell("severity"),
    summary: cell("summary"),
  });
}

const ROW_KEYS: KindHandlers<ActivityRowPayload, string> = {
  event: ({ kind, entry }) => `${kind}${ROW_KEY_SEPARATOR}${entry.id}`,
  uplink: ({ kind, uplink }) => `${kind}${ROW_KEY_SEPARATOR}${uplink.id}`,
};

const ROW_DETAILS: KindHandlers<ActivityRowPayload, ReactNode> = {
  event: ({ entry }) => <EventData entry={entry} />,
  uplink: ({ uplink }) => <UplinkDetails uplink={uplink} />,
};

export const activityRowKey = (row: ActivityRow): string =>
  byKind(row, ROW_KEYS);

export const activityRowDetails = (row: ActivityRow): ReactNode =>
  byKind(row, ROW_DETAILS);
