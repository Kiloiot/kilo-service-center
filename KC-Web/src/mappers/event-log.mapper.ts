/**
 * Event Log Mappers
 *
 * Turns a ListEvents record into an Events/Audit Log row; the opId column is
 * read from the event details, where BSSCI and SCACI events record it, and an
 * event without a source is scoped to the device its details name.
 */

import type { EventLogEntryUI, EventRecordAPI } from "@api-types/api";

import { isJsonObject } from "@utils/exact-json";
import { EVENT_DETAIL_KEY, EVENT_DETAIL_KEY_OP_ID } from "@constants/app";

// The end point comes first: a downlink's station is only where it was sent.
const SCOPE_DETAIL_KEYS = [
  EVENT_DETAIL_KEY.EP_EUI,
  EVENT_DETAIL_KEY.BS_EUI,
] as const;

function readOpId(
  data: Record<string, unknown> | undefined,
): string | undefined {
  const value = data?.[EVENT_DETAIL_KEY_OP_ID];
  return typeof value === "number" || typeof value === "string"
    ? String(value)
    : undefined;
}

function readScope(
  sourceName: string,
  data: Record<string, unknown> | undefined,
): string | undefined {
  if (sourceName) return sourceName;
  return SCOPE_DETAIL_KEYS.map((key) => data?.[key]).find(
    (value): value is string => typeof value === "string" && value !== "",
  );
}

export function mapEventLogEntry(api: EventRecordAPI): EventLogEntryUI {
  const data = isJsonObject(api.data) ? api.data : undefined;
  return {
    id: api.id,
    timestamp: api.timestamp.toISOString(),
    eventType: api.eventType,
    category: api.category,
    severity: api.severity,
    title: api.title,
    description: api.description,
    scope: readScope(api.sourceName, data),
    userId: api.userId || undefined,
    userEmail: api.userEmail || undefined,
    opId: readOpId(data),
    data: data && Object.keys(data).length > 0 ? data : undefined,
  };
}
