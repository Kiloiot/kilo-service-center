/**
 * The streams a tab needs and the streams the leader tab holds for all tabs.
 * A stream is shared by every tab that asks for it in the same context.
 */

import { REALTIME_STREAM_KIND } from "@constants/app";

/** The organization, user and uplink grant a tab's streams are opened for. */
export interface StreamContext {
  organizationId: string;
  userId: string;
  uplinks: boolean;
}

/** What one tab asks the leader to stream: its context and the stations it watches. */
export interface TabDemand extends StreamContext {
  stations: string[];
}

/** One stream the leader opens; a station stream names its base station. */
export type StreamSpec =
  | {
      kind:
        | typeof REALTIME_STREAM_KIND.EVENT
        | typeof REALTIME_STREAM_KIND.MESSAGE;
      context: StreamContext;
    }
  | {
      kind: typeof REALTIME_STREAM_KIND.BASE_STATION;
      context: StreamContext;
      bsEui: string;
    };

/** Identity of a stream across tabs: equal specs share one stream. */
export function streamKey(spec: StreamSpec): string {
  const { organizationId, userId, uplinks } = spec.context;
  const station =
    spec.kind === REALTIME_STREAM_KIND.BASE_STATION ? spec.bsEui : null;
  return JSON.stringify([spec.kind, organizationId, userId, uplinks, station]);
}

function contextOf(demand: TabDemand): StreamContext {
  return {
    organizationId: demand.organizationId,
    userId: demand.userId,
    uplinks: demand.uplinks,
  };
}

/** The streams whose state is the tab's connection state: the event stream, and the uplink stream when granted. */
export function heldSpecs(demand: TabDemand): StreamSpec[] {
  const context = contextOf(demand);
  const events: StreamSpec = { kind: REALTIME_STREAM_KIND.EVENT, context };
  return demand.uplinks
    ? [events, { kind: REALTIME_STREAM_KIND.MESSAGE, context }]
    : [events];
}

/**
 * Every stream the tab consumes. A watched station needs its own stream only
 * without the uplink stream, which carries every uplink of the tenant.
 */
export function demandSpecs(demand: TabDemand): StreamSpec[] {
  const context = contextOf(demand);
  const stations = demand.uplinks
    ? []
    : demand.stations.map(
        (bsEui): StreamSpec => ({
          kind: REALTIME_STREAM_KIND.BASE_STATION,
          context,
          bsEui,
        }),
      );
  return [...heldSpecs(demand), ...stations];
}

/** The streams the leader holds: one per distinct spec any tab demands. */
export function unionSpecs(
  demands: Iterable<TabDemand>,
): Map<string, StreamSpec> {
  const specs = new Map<string, StreamSpec>();
  for (const demand of demands) {
    demandSpecs(demand).forEach((spec) => specs.set(streamKey(spec), spec));
  }
  return specs;
}
