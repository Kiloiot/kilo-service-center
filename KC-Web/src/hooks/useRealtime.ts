/**
 * Realtime Hooks
 *
 * React hooks for realtime connection management and cache invalidation.
 */

import { useEffect, useRef, useState } from "react";

import type { QueryClient } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

import {
  type ConnectionState,
  type RealtimeEvent,
  type RealtimeEventType,
  realtimeService,
} from "@services/realtime";
import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import {
  REALTIME_STREAM_KIND,
  TIMING_REALTIME_EVENT_STREAM_CATCHUP_DEBOUNCE_MS,
  TIMING_REALTIME_EVENT_STREAM_CATCHUP_MIN_INTERVAL_MS,
  TIMING_REALTIME_INVALIDATION_WINDOW_MS,
} from "@constants/app";
import { queryKeys } from "@config/query-keys";

/**
 * Event type to query keys invalidation mapping.
 * When an event is received, these query keys are invalidated.
 */
const EVENT_INVALIDATION_MAP: Partial<
  Record<RealtimeEventType, readonly (readonly unknown[])[]>
> = {
  "uplink.received": [
    queryKeys.endpoints.all,
    queryKeys.events.all,
    queryKeys.dashboard.all,
  ],
  "endpoint.attached": [
    queryKeys.endpoints.all,
    queryKeys.events.all,
    queryKeys.dashboard.all,
  ],
  "endpoint.detached": [
    queryKeys.endpoints.all,
    queryKeys.events.all,
    queryKeys.dashboard.all,
  ],
  "basestation.online": [
    queryKeys.baseStations.all,
    queryKeys.dashboard.all,
    queryKeys.events.all,
  ],
  "basestation.offline": [
    queryKeys.baseStations.all,
    queryKeys.dashboard.all,
    queryKeys.events.all,
  ],
  "downlink.queued": [
    queryKeys.endpoints.all,
    queryKeys.events.all,
    queryKeys.scaci.all,
  ],
  "downlink.sent": [
    queryKeys.endpoints.all,
    queryKeys.events.all,
    queryKeys.scaci.all,
  ],
  "downlink.failed": [
    queryKeys.endpoints.all,
    queryKeys.events.all,
    queryKeys.scaci.all,
  ],
  "downlink.revoked": [
    queryKeys.endpoints.all,
    queryKeys.events.all,
    queryKeys.scaci.all,
  ],
  "scaci.session.opened": [queryKeys.scaci.all, queryKeys.events.all],
  "scaci.session.closed": [queryKeys.scaci.all, queryKeys.events.all],
  "scaci.error": [queryKeys.scaci.all, queryKeys.events.all],
  "event.received": [queryKeys.events.all, queryKeys.dashboard.all],
};

/** Decode a `bytes`-or-string `event.data` field to a UTF-8 string. */
function decodeEventDataField(value: unknown): string | undefined {
  if (value instanceof Uint8Array) {
    return new TextDecoder().decode(value);
  }
  if (typeof value === "string") {
    return value;
  }
  return undefined;
}

/** JSON.parse without throwing — returns undefined on any parse error. */
function safeJsonParse(text: string): Record<string, unknown> | undefined {
  try {
    return JSON.parse(text) as Record<string, unknown>;
  } catch {
    return undefined;
  }
}

/**
 * Reads `eventDataKey` (e.g. "bsEui", "epEui") from event.data, treating
 * event.data as JSON-encoded bytes or a JSON string. Returns undefined when
 * the field is absent or unparseable.
 */
function readKeyFromEventData(
  event: Record<string, unknown>,
  eventDataKey: "bsEui" | "epEui",
): string | undefined {
  const dataStr = decodeEventDataField(event.data);
  if (!dataStr) return undefined;
  const parsed = safeJsonParse(dataStr);
  if (parsed && typeof parsed[eventDataKey] === "string") {
    return parsed[eventDataKey] as string;
  }
  return undefined;
}

/**
 * Extract bsEui from event payload for targeted invalidation.
 * Priority: sourceName → event.data (bytes/JSON) → details → sourceId.
 */
function extractBsEuiFromPayload(
  payload?: Record<string, unknown>,
): string | undefined {
  if (!payload?.event || typeof payload.event !== "object") return undefined;
  const event = payload.event as Record<string, unknown>;

  if (typeof event.sourceName === "string" && event.sourceName) {
    return event.sourceName;
  }

  const fromData = readKeyFromEventData(event, "bsEui");
  if (fromData) return fromData;

  if (event.details) {
    const details =
      typeof event.details === "string"
        ? safeJsonParse(event.details)
        : (event.details as Record<string, unknown>);
    if (details && typeof details.bsEui === "string") {
      return details.bsEui;
    }
  }

  if (typeof event.sourceId === "string" && event.sourceId) {
    return event.sourceId;
  }
  return undefined;
}

/**
 * Extract epEui from event payload for targeted endpoint invalidation.
 * Priority: event.sourceName → event.data.epEui → message.epEui.
 */
function extractEpEuiFromPayload(
  payload?: Record<string, unknown>,
): string | undefined {
  if (!payload) return undefined;

  if (payload.event && typeof payload.event === "object") {
    const event = payload.event as Record<string, unknown>;
    if (typeof event.sourceName === "string" && event.sourceName) {
      return event.sourceName;
    }
    const fromData = readKeyFromEventData(event, "epEui");
    if (fromData) return fromData;
  }

  if (payload.message && typeof payload.message === "object") {
    const msg = payload.message as { epEui?: string };
    if (typeof msg.epEui === "string") {
      return msg.epEui;
    }
  }
  return undefined;
}

function baseStationKeys(bsEui: string): (readonly unknown[])[] {
  return [
    queryKeys.baseStations.detail(bsEui),
    queryKeys.baseStations.messages(bsEui),
    queryKeys.baseStations.activity(bsEui),
  ];
}

function endpointKeys(epEui: string): (readonly unknown[])[] {
  return [
    queryKeys.endpoints.detail(epEui),
    queryKeys.endpoints.activity(epEui),
  ];
}

// Backend audit events emit the EUI as uppercase hex (`%016X`), but React
// Query keys built from the lowercase canonical form would not match without
// normalization. Strip separators and lowercase to align both forms.
function normalizeEuiForQueryKey(eui: string): string {
  return eui.replace(/-/g, "").toLowerCase();
}

/** Per-event-type targeted query keys (device detail/activity/downlink). */
function targetedInvalidationKeys(
  event: RealtimeEvent,
): (readonly unknown[])[] {
  if (
    event.type === "basestation.online" ||
    event.type === "basestation.offline"
  ) {
    const bsEui = extractBsEuiFromPayload(event.payload);
    return bsEui ? baseStationKeys(bsEui) : [];
  }

  if (event.type === "uplink.received" && event.payload?.message) {
    const msg = event.payload.message as { bsEui?: string; epEui?: string };
    const keys: (readonly unknown[])[] = [];
    if (msg.bsEui) keys.push(...baseStationKeys(msg.bsEui));
    if (msg.epEui) keys.push(...endpointKeys(msg.epEui));
    return keys;
  }

  if (
    event.type === "endpoint.attached" ||
    event.type === "endpoint.detached"
  ) {
    const epEui = extractEpEuiFromPayload(event.payload);
    return epEui ? endpointKeys(epEui) : [];
  }

  if (
    event.type === "downlink.queued" ||
    event.type === "downlink.sent" ||
    event.type === "downlink.failed" ||
    event.type === "downlink.revoked"
  ) {
    const rawEpEui = extractEpEuiFromPayload(event.payload);
    if (!rawEpEui) return [];
    const epEui = normalizeEuiForQueryKey(rawEpEui);
    return [
      queryKeys.endpoints.detail(epEui),
      queryKeys.endpoints.downlinkQueuePrefix(epEui),
      queryKeys.endpoints.downlinkResultsPrefix(epEui),
    ];
  }

  return [];
}

/**
 * Trigger realtimeService.reconnectWithOrg whenever the supplied dependencies
 * become ready (session hydrated + authenticated, organization + user set).
 * The realtime singleton manages its own backoff/disconnect lifecycle.
 */
function useReconnectBackoff(deps: {
  organizationId: string | null | undefined;
  userId: string | null | undefined;
  isAuthenticated: boolean;
  isHydrated: boolean;
}): void {
  const { organizationId, userId, isAuthenticated, isHydrated } = deps;
  useEffect(() => {
    if (!isHydrated || !isAuthenticated) return;
    if (!organizationId || !userId) return;
    realtimeService.reconnectWithOrg(organizationId, userId);
    // Singleton persists across route changes — only disconnect via beforeunload.
  }, [organizationId, userId, isAuthenticated, isHydrated]);
}

/**
 * Hook for managing realtime connection state
 * Reconnects when organization changes
 * Only connects when user is authenticated (realtime endpoint requires auth)
 *
 * @returns Current connection state and isConnected flag
 */
export function useRealtimeConnection() {
  const [state, setState] = useState<ConnectionState>("disconnected");
  const { organizationId, userId } = useOrganization();
  const { isAuthenticated, isHydrated } = useSession();

  useEffect(() => realtimeService.onStateChange(setState), []);
  useReconnectBackoff({ organizationId, userId, isAuthenticated, isHydrated });

  return {
    state,
    isConnected: state === "connected",
    isReconnecting: state === "reconnecting",
  };
}

/**
 * Hook for automatic query invalidation based on realtime events.
 *
 * Invalidations are coalesced: each event adds its target query keys to a
 * pending set, flushed at most once per TIMING_REALTIME_INVALIDATION_WINDOW_MS.
 * Otherwise a busy tenant's stream refetches the dashboard lists on every
 * event — a request storm.
 */
export function useRealtimeInvalidation() {
  const queryClient = useQueryClient();

  useEffect(() => {
    // Serialized key -> query key; dedups repeated keys within the window.
    const pending = new Map<string, readonly unknown[]>();
    let flushTimer: ReturnType<typeof setTimeout> | null = null;

    const flush = () => {
      flushTimer = null;
      const keys = [...pending.values()];
      pending.clear();
      keys.forEach((queryKey) => queryClient.invalidateQueries({ queryKey }));
    };

    const handler = (event: RealtimeEvent) => {
      const keys = [
        ...(EVENT_INVALIDATION_MAP[event.type] ?? []),
        ...targetedInvalidationKeys(event),
      ];
      if (keys.length === 0) return;
      keys.forEach((key) => pending.set(JSON.stringify(key), key));
      if (!flushTimer) {
        flushTimer = setTimeout(flush, TIMING_REALTIME_INVALIDATION_WINDOW_MS);
      }
    };

    const unsubscribe = realtimeService.subscribeAll(handler);
    return () => {
      unsubscribe();
      if (flushTimer) clearTimeout(flushTimer);
    };
  }, [queryClient]);
}

type EventStreamFlight = {
  hasConnected: boolean;
  connected: boolean;
  pendingInvalidate: ReturnType<typeof setTimeout> | null;
  lastInvalidateAt: number;
};

/**
 * Invalidate the EVENT-stream-backed caches: anything that depends on
 * system_events. Includes endpoints (downlink lifecycle), base stations
 * (online/offline events), events feed, SCACI, and dashboard.
 */
function invalidateForEventStream(qc: QueryClient): void {
  qc.invalidateQueries({ queryKey: queryKeys.endpoints.all });
  qc.invalidateQueries({ queryKey: queryKeys.baseStations.all });
  qc.invalidateQueries({ queryKey: queryKeys.events.all });
  qc.invalidateQueries({ queryKey: queryKeys.scaci.all });
  qc.invalidateQueries({ queryKey: queryKeys.dashboard.all });
}

/**
 * Catch-up invalidation when the EVENT stream reconnects after a drop. The
 * EVENT gRPC stream idle-times out every few minutes; events emitted during
 * the gap never reach the browser. Without this hook, the UI stays stale
 * until something else (window focus, manual refresh) refetches.
 *
 * Scoped narrowly to the EVENT stream only — the MESSAGE stream's reconnect
 * cycles `connectEventStream()` (closing+reopening the event stream) every
 * time it itself reconnects, so handling MESSAGE here would double-count and
 * could cascade into a refetch storm. The EVENT stream's own
 * realtime_connected event captures every relevant reconnect anyway.
 *
 * Debounced + rate-limited so any cluster of rapid reconnect events
 * collapses into a single invalidation, and no more than one fires per
 * `TIMING_REALTIME_EVENT_STREAM_CATCHUP_MIN_INTERVAL_MS`. This prevents the kind of
 * cascade that would happen when the message stream reconnects and triggers
 * an event-stream close+reopen in quick succession.
 */
function useCatchUpOnStreamReconnect(): void {
  const queryClient = useQueryClient();
  const flightRef = useRef<EventStreamFlight>({
    hasConnected: false,
    connected: false,
    pendingInvalidate: null,
    lastInvalidateAt: 0,
  });

  useEffect(() => {
    // Snapshot the ref once: the flight state object is stable for the effect's
    // lifetime, and the cleanup must not re-read the ref (react-hooks/exhaustive-deps).
    const flight = flightRef.current;

    const scheduleInvalidate = () => {
      if (flight.pendingInvalidate) {
        clearTimeout(flight.pendingInvalidate);
      }
      flight.pendingInvalidate = setTimeout(() => {
        flight.pendingInvalidate = null;
        const now = Date.now();
        if (
          now - flight.lastInvalidateAt <
          TIMING_REALTIME_EVENT_STREAM_CATCHUP_MIN_INTERVAL_MS
        ) {
          return;
        }
        flight.lastInvalidateAt = now;
        invalidateForEventStream(queryClient);
      }, TIMING_REALTIME_EVENT_STREAM_CATCHUP_DEBOUNCE_MS);
    };

    const unsubscribe = realtimeService.onConnectionEvent((evt) => {
      if (evt.streamKind !== REALTIME_STREAM_KIND.EVENT) return;

      if (evt.type === "realtime_connected") {
        const isReconnect = flight.hasConnected && !flight.connected;
        flight.hasConnected = true;
        flight.connected = true;
        if (isReconnect) {
          scheduleInvalidate();
        }
      } else if (
        evt.type === "realtime_disconnected" ||
        evt.type === "realtime_error"
      ) {
        flight.connected = false;
      }
    });

    return () => {
      unsubscribe();
      if (flight.pendingInvalidate) {
        clearTimeout(flight.pendingInvalidate);
        flight.pendingInvalidate = null;
      }
    };
  }, [queryClient]);
}

/**
 * Combined hook for realtime connection + cache invalidation
 *
 * Use this in App.tsx or at the top level to enable realtime updates
 */
export function useRealtimeUpdates() {
  const { state, isConnected, isReconnecting } = useRealtimeConnection();
  useRealtimeInvalidation();
  useCatchUpOnStreamReconnect();

  return { state, isConnected, isReconnecting };
}
