/**
 * Activity feed pages of the base-station and endpoint RPCs: each item is a
 * system event or a stored uplink, read by the shared event and uplink
 * decoders.
 */

import type { EventRecordAPI, UplinkAPI } from "@api-types/api";
import type { Tagged } from "@api-types/tagged";

import type * as corePb from "@services/grpc/core_pb";
import { ACTIVITY_KIND } from "@constants/app";

import { mapEventRecord } from "./events";

/** The payload of each kind of activity item (ACTIVITY_KIND). */
export interface ActivityItemPayload {
  event: { event: EventRecordAPI };
  uplink: { uplink: UplinkAPI };
}

export type ActivityItemDTO = Tagged<ActivityItemPayload>;

export interface ActivityPageDTO {
  items: ActivityItemDTO[];
  nextPageToken?: string;
  totalCount: number;
}

/** A feed item of either activity RPC; M is the uplink message the feed carries. */
interface ActivityItemProto<M> {
  getEvent(): corePb.Event | undefined;
  getMessage(): M | undefined;
}

interface ActivityResponseProto<M> {
  getItemsList(): ActivityItemProto<M>[];
  getNextPageToken(): string;
  getTotalCount(): number;
}

// A proto oneof: an item carrying neither an event nor an uplink yields no row.
function mapActivityItem<M>(
  item: ActivityItemProto<M>,
  decodeUplink: (message: M) => UplinkAPI,
): ActivityItemDTO[] {
  const event = item.getEvent();
  if (event)
    return [{ kind: ACTIVITY_KIND.EVENT, event: mapEventRecord(event) }];
  const message = item.getMessage();
  if (message) {
    return [{ kind: ACTIVITY_KIND.UPLINK, uplink: decodeUplink(message) }];
  }
  return [];
}

/** Decodes an activity response with the uplink decoder of the feed's message type. */
export function mapActivityResponse<M>(
  response: ActivityResponseProto<M>,
  decodeUplink: (message: M) => UplinkAPI,
): ActivityPageDTO {
  return {
    items: response
      .getItemsList()
      .flatMap((item) => mapActivityItem(item, decodeUplink)),
    nextPageToken: response.getNextPageToken() || undefined,
    totalCount: response.getTotalCount(),
  };
}
