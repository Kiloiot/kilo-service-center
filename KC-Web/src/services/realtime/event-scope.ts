/**
 * The base stations and endpoints a realtime event concerns: the stations
 * that heard an uplink and its endpoint, or the EUIs a system event's details
 * and source name carry.
 */

import { normalizeEui } from "@utils/eui";
import { decodeExactJsonObject } from "@utils/exact-json";
import type { SourceKind } from "@constants/app";
import { EVENT_DETAIL_KEY, EVENT_SOURCE_KIND } from "@constants/app";

import type { RealtimeEvent } from "./types";

/** What a system event's source name identifies. */

export interface EventScope {
  stations: string[];
  endpoints: string[];
}

interface UplinkMessage {
  bsEui?: string;
  epEui?: string;
  baseStationsList?: { bsEui?: string }[];
}

interface StreamedEvent {
  sourceName?: string;
  data?: string | Uint8Array;
}

/** Reads a streamed event's JSON details; gRPC-web renders bytes as base64. */
function eventDetails(data: StreamedEvent["data"]): Record<string, unknown> {
  if (!data) return {};
  try {
    const bytes =
      typeof data === "string"
        ? Uint8Array.from(atob(data), (c) => c.charCodeAt(0))
        : data;
    return decodeExactJsonObject(bytes) ?? {};
  } catch {
    // atob rejects a malformed base64 rendering.
    return {};
  }
}

function euis(...values: unknown[]): string[] {
  const found = values.filter(
    (value): value is string => typeof value === "string" && value !== "",
  );
  return [...new Set(found.map(normalizeEui))];
}

function uplinkScope(message: UplinkMessage): EventScope {
  const receivers = (message.baseStationsList ?? []).map((bs) => bs.bsEui);
  return {
    stations: euis(message.bsEui, ...receivers),
    endpoints: euis(message.epEui),
  };
}

function systemEventScope(
  event: StreamedEvent,
  source?: SourceKind,
): EventScope {
  const details = eventDetails(event.data);
  const sourceName = event.sourceName;
  return {
    stations: euis(
      details[EVENT_DETAIL_KEY.BS_EUI],
      source === EVENT_SOURCE_KIND.STATION ? sourceName : undefined,
    ),
    endpoints: euis(
      details[EVENT_DETAIL_KEY.EP_EUI],
      source === EVENT_SOURCE_KIND.ENDPOINT ? sourceName : undefined,
    ),
  };
}

/** The stations and endpoints an event names, as canonical EUIs. */
export function eventScope(
  event: RealtimeEvent,
  source?: SourceKind,
): EventScope {
  const payload = event.payload ?? {};
  if (payload.message && typeof payload.message === "object") {
    return uplinkScope(payload.message as UplinkMessage);
  }
  if (payload.event && typeof payload.event === "object") {
    return systemEventScope(payload.event as StreamedEvent, source);
  }
  return { stations: [], endpoints: [] };
}
