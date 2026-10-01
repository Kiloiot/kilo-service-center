/**
 * Activity feed pages as Activity table rows, shared by the base-station and
 * endpoint activity calls: events through the Events Log mapper, uplinks
 * through the Traffic uplink mapper.
 */

import type { ActivityPage, ActivityRow } from "@api-types/api";
import type { KindHandlers } from "@api-types/tagged";
import { mapEventLogEntry, mapUplink } from "@mappers";

import type {
  ActivityItemPayload,
  ActivityPageDTO,
} from "@services/grpc/rpc/activity";
import { byKind } from "@utils/by-kind";
import { ACTIVITY_KIND } from "@constants/app";

const toRow = (
  bsEui: string | undefined,
): KindHandlers<ActivityItemPayload, ActivityRow> => ({
  event: ({ event }) => ({
    kind: ACTIVITY_KIND.EVENT,
    entry: mapEventLogEntry(event),
  }),
  uplink: ({ uplink }) => ({
    kind: ACTIVITY_KIND.UPLINK,
    uplink: mapUplink(uplink, bsEui),
  }),
});

/** An uplink row reports the reception of the station in view (bsEui), else its first one. */
export function mapActivityPage(
  response: ActivityPageDTO,
  bsEui?: string,
): ActivityPage {
  const handlers = toRow(bsEui);
  return {
    items: response.items.map((item) => byKind(item, handlers)),
    nextPageToken: response.nextPageToken,
    totalCount: response.totalCount,
  };
}
