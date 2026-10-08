/**
 * The realtime events the streams' messages become: an uplink, or a system
 * event under its UI type, stamped with the organization whose stream
 * carried it.
 */

import type * as pb from "@services/grpc/core_pb";
import {
  BACKEND_EVENT_TO_REALTIME,
  MS_PER_SECOND,
  NANOSECONDS_PER_MILLISECOND,
  REALTIME_EVENT_TYPE,
} from "@constants/app";

import type { RealtimeEvent } from "./types";

type ProtoTimestamp = { seconds: number; nanos: number } | undefined;

function isoTime(timestamp: ProtoTimestamp): string {
  if (!timestamp) return new Date().toISOString();
  return new Date(
    timestamp.seconds * MS_PER_SECOND +
      timestamp.nanos / NANOSECONDS_PER_MILLISECOND,
  ).toISOString();
}

export function uplinkEvent(
  message: pb.Message.AsObject | pb.BaseStationMessage.AsObject,
  organizationId: string,
): RealtimeEvent {
  return {
    type: REALTIME_EVENT_TYPE.UPLINK_RECEIVED,
    timestamp: isoTime(message.receivedAt),
    organizationId,
    payload: { message },
  };
}

/** A system event under its UI type; a type without a mapping is a generic event. */
export function systemEvent(
  event: pb.Event.AsObject,
  organizationId: string,
): RealtimeEvent {
  return {
    type:
      BACKEND_EVENT_TO_REALTIME[event.eventType] ??
      REALTIME_EVENT_TYPE.EVENT_RECEIVED,
    timestamp: isoTime(event.timestamp),
    organizationId,
    payload: { event, originalEventType: event.eventType },
  };
}
