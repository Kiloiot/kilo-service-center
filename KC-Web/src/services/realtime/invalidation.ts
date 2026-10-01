/**
 * Which cached queries a realtime event makes stale: the lists of the event's
 * family, and the views of every base station and endpoint the event names.
 */

import type { SourceKind } from "@constants/app";
import {
  EVENT_SOURCE_KIND,
  REALTIME_EVENT_TYPE,
  REALTIME_STREAM_KIND,
} from "@constants/app";
import { queryKeys } from "@config/query-keys";

import { type EventScope, eventScope } from "./event-scope";
import type {
  RealtimeEvent,
  RealtimeEventType,
  RealtimeStreamKind,
} from "./types";

export type QueryKey = readonly unknown[];

interface EventHandling {
  /** Query families refreshed for every event of the type. */
  families: readonly QueryKey[];
  /** What the event's source name identifies, when it names one. */
  source?: SourceKind;
}

const FAMILIES = {
  // An uplink's reception reserves and sends the endpoint's pending downlink without an event of its own.
  uplink: [
    queryKeys.endpoints.all,
    queryKeys.baseStations.lists(),
    queryKeys.events.all,
    queryKeys.dashboard.all,
    queryKeys.traffic.uplinks,
    queryKeys.traffic.downlinkQueues,
  ],
  endpoint: [
    queryKeys.endpoints.all,
    queryKeys.events.all,
    queryKeys.dashboard.all,
    queryKeys.traffic.downlinks,
    queryKeys.blueprints.modelSnapshotCounts(),
  ],
  baseStation: [
    queryKeys.baseStations.all,
    queryKeys.events.all,
    queryKeys.dashboard.all,
  ],
  downlink: [
    queryKeys.endpoints.all,
    queryKeys.events.all,
    queryKeys.scaci.all,
    queryKeys.dashboard.all,
    queryKeys.traffic.downlinks,
  ],
  scaci: [queryKeys.scaci.all, queryKeys.events.all, queryKeys.dashboard.all],
  user: [
    queryKeys.users.all,
    queryKeys.organizations.usersAll(),
    queryKeys.userOrganizations.all,
    queryKeys.events.all,
  ],
  organization: [
    queryKeys.organizations.all,
    queryKeys.userOrganizations.all,
    queryKeys.events.all,
  ],
  // A membership change may be the viewer's own, which changes their roles.
  membership: [
    queryKeys.organizations.all,
    queryKeys.userOrganizations.all,
    queryKeys.auth.rolesAll(),
    queryKeys.events.all,
  ],
  apiKey: [queryKeys.apiKeys.all, queryKeys.events.all],
  serverCertificate: [queryKeys.certificates.all, queryKeys.events.all],
  catalog: [queryKeys.blueprints.all, queryKeys.events.all],
  events: [queryKeys.events.all, queryKeys.dashboard.all],
} as const;

const ENDPOINT: EventHandling = {
  families: FAMILIES.endpoint,
  source: EVENT_SOURCE_KIND.ENDPOINT,
};
const BASE_STATION: EventHandling = {
  families: FAMILIES.baseStation,
  source: EVENT_SOURCE_KIND.STATION,
};
const DOWNLINK: EventHandling = {
  families: FAMILIES.downlink,
  source: EVENT_SOURCE_KIND.ENDPOINT,
};
const SCACI: EventHandling = { families: FAMILIES.scaci };

/**
 * What each realtime event type refreshes. The Record is total over
 * RealtimeEventType, so adding an event type without deciding its handling
 * is a compile error.
 */
const EVENT_HANDLING: Record<RealtimeEventType, EventHandling> = {
  [REALTIME_EVENT_TYPE.UPLINK_RECEIVED]: { families: FAMILIES.uplink },
  [REALTIME_EVENT_TYPE.ENDPOINT_ATTACHED]: ENDPOINT,
  [REALTIME_EVENT_TYPE.ENDPOINT_DETACHED]: ENDPOINT,
  [REALTIME_EVENT_TYPE.ENDPOINT_CHANGED]: ENDPOINT,
  [REALTIME_EVENT_TYPE.BASESTATION_ONLINE]: BASE_STATION,
  [REALTIME_EVENT_TYPE.BASESTATION_OFFLINE]: BASE_STATION,
  [REALTIME_EVENT_TYPE.BASESTATION_CHANGED]: BASE_STATION,
  [REALTIME_EVENT_TYPE.DOWNLINK_QUEUED]: DOWNLINK,
  [REALTIME_EVENT_TYPE.DOWNLINK_UPDATED]: DOWNLINK,
  [REALTIME_EVENT_TYPE.DOWNLINK_SENT]: DOWNLINK,
  [REALTIME_EVENT_TYPE.DOWNLINK_ACKNOWLEDGED]: DOWNLINK,
  [REALTIME_EVENT_TYPE.DOWNLINK_FAILED]: DOWNLINK,
  [REALTIME_EVENT_TYPE.DOWNLINK_REVOKED]: DOWNLINK,
  [REALTIME_EVENT_TYPE.SCACI_SESSION_OPENED]: SCACI,
  [REALTIME_EVENT_TYPE.SCACI_SESSION_CLOSED]: SCACI,
  [REALTIME_EVENT_TYPE.SCACI_ERROR]: SCACI,
  [REALTIME_EVENT_TYPE.USER_CHANGED]: { families: FAMILIES.user },
  [REALTIME_EVENT_TYPE.ORGANIZATION_CHANGED]: {
    families: FAMILIES.organization,
  },
  [REALTIME_EVENT_TYPE.MEMBERSHIP_CHANGED]: { families: FAMILIES.membership },
  [REALTIME_EVENT_TYPE.API_KEY_CHANGED]: { families: FAMILIES.apiKey },
  [REALTIME_EVENT_TYPE.SERVER_CERTIFICATE_CHANGED]: {
    families: FAMILIES.serverCertificate,
  },
  [REALTIME_EVENT_TYPE.CATALOG_CHANGED]: { families: FAMILIES.catalog },
  [REALTIME_EVENT_TYPE.EVENT_RECEIVED]: { families: FAMILIES.events },
};

const UPLINK_TYPES: readonly RealtimeEventType[] = [
  REALTIME_EVENT_TYPE.UPLINK_RECEIVED,
];

/** The event types each stream carries: uplinks, or every system event. */
const STREAM_EVENT_TYPES: Record<
  RealtimeStreamKind,
  readonly RealtimeEventType[]
> = {
  [REALTIME_STREAM_KIND.MESSAGE]: UPLINK_TYPES,
  [REALTIME_STREAM_KIND.BASE_STATION]: UPLINK_TYPES,
  [REALTIME_STREAM_KIND.EVENT]: Object.values(REALTIME_EVENT_TYPE).filter(
    (type) => !UPLINK_TYPES.includes(type),
  ),
};

function stationViews(eui: string): QueryKey[] {
  return [
    queryKeys.baseStations.detail(eui),
    queryKeys.baseStations.activityPrefix(eui),
  ];
}

function endpointViews(eui: string): QueryKey[] {
  return [
    queryKeys.endpoints.detail(eui),
    queryKeys.endpoints.activityPrefix(eui),
  ];
}

function scopedViews(scope: EventScope): QueryKey[] {
  return [
    ...scope.stations.flatMap(stationViews),
    ...scope.endpoints.flatMap(endpointViews),
  ];
}

/** The query keys an event makes stale; empty for an unknown event type. */
export function invalidatedKeys(event: RealtimeEvent): QueryKey[] {
  const handling = EVENT_HANDLING[event.type];
  if (!handling) return [];
  return [
    ...handling.families,
    ...scopedViews(eventScope(event, handling.source)),
  ];
}

/** Every query family a stream's events refresh, re-read once it reconnects after a drop. */
export function catchUpKeys(kind: RealtimeStreamKind): QueryKey[] {
  const keys = new Map<string, QueryKey>();
  STREAM_EVENT_TYPES[kind]
    .flatMap((type) => EVENT_HANDLING[type].families)
    .forEach((key) => keys.set(JSON.stringify(key), key));
  return [...keys.values()];
}
