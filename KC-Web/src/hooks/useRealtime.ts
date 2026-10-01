/**
 * Realtime Hooks
 *
 * The app keeps its views current through the realtime streams: every event
 * invalidates the queries it makes stale, a reconnected stream catches up on
 * what it may have missed, and while the streams are not all connected the
 * open views are re-read on a timer instead.
 */

import { useEffect, useRef, useState } from "react";

import { useQueryClient } from "@tanstack/react-query";

import {
  catchUpKeys,
  type ConnectionState,
  invalidatedKeys,
  type RealtimeEvent,
  realtimeLifecycle,
  type RealtimeStreamKind,
  realtimeSubscriptions,
} from "@services/realtime";
import { useOrganization } from "@contexts/OrganizationContext";
import { useSession } from "@contexts/SessionContext";
import {
  CONNECTION_STATE,
  REALTIME_LIFECYCLE_EVENT,
  TIMING_FALLBACK_POLL_MS,
  TIMING_REALTIME_CATCHUP_DEBOUNCE_MS,
  TIMING_REALTIME_CATCHUP_MIN_INTERVAL_MS,
  TIMING_REALTIME_INVALIDATION_BATCH_MS,
  TIMING_REALTIME_INVALIDATION_WINDOW_MS,
} from "@constants/app";

import { useCapabilities } from "./useCapabilities";

/**
 * Opens the streams the signed-in user's roles may hold in the current
 * organization: the event stream for every role, the server narrowing its
 * categories to the roles, and the uplink stream for endpoint managers.
 */
function useStreamsForRoles(): void {
  const { organizationId, userId } = useOrganization();
  const { isAuthenticated, isHydrated } = useSession();
  const { hasAnyRole, isEndpointManager, rolesLoaded } = useCapabilities();

  useEffect(() => {
    if (!isHydrated || !isAuthenticated || !rolesLoaded) return;
    if (!organizationId || !userId) return;
    // A refused stream would only retry into the same refusal; a revoked role closes it.
    if (!hasAnyRole) {
      realtimeLifecycle.reset();
      return;
    }
    realtimeLifecycle.reconnectWithOrg(organizationId, userId, {
      uplinks: isEndpointManager,
    });
  }, [
    organizationId,
    userId,
    isAuthenticated,
    isHydrated,
    rolesLoaded,
    hasAnyRole,
    isEndpointManager,
  ]);
}

/** The combined state of the streams the user holds. */
export function useRealtimeConnection() {
  const [state, setState] = useState<ConnectionState>(() =>
    realtimeLifecycle.getState(),
  );
  useEffect(() => realtimeLifecycle.onStateChange(setState), []);
  useStreamsForRoles();

  return {
    state,
    isConnected: state === CONNECTION_STATE.CONNECTED,
    isReconnecting: state === CONNECTION_STATE.RECONNECTING,
  };
}

/**
 * Hook for automatic query invalidation based on realtime events.
 *
 * Invalidations are coalesced: each event adds its target query keys to a
 * pending set. The first event after a quiet window is flushed after
 * TIMING_REALTIME_INVALIDATION_BATCH_MS; later ones wait for the window to end,
 * so flushes happen at most once per TIMING_REALTIME_INVALIDATION_WINDOW_MS.
 * Otherwise a busy tenant's stream refetches the dashboard lists on every
 * event — a request storm.
 */
export function useRealtimeInvalidation() {
  const queryClient = useQueryClient();

  useEffect(() => {
    // Serialized key -> query key; dedups repeated keys within the window.
    const pending = new Map<string, readonly unknown[]>();
    let flushTimer: ReturnType<typeof setTimeout> | null = null;
    let lastFlushAt = Number.NEGATIVE_INFINITY;

    const flush = () => {
      flushTimer = null;
      lastFlushAt = Date.now();
      const keys = [...pending.values()];
      pending.clear();
      keys.forEach((queryKey) => queryClient.invalidateQueries({ queryKey }));
    };

    const handler = (event: RealtimeEvent) => {
      const keys = invalidatedKeys(event);
      if (keys.length === 0) return;
      keys.forEach((key) => pending.set(JSON.stringify(key), key));
      if (!flushTimer) {
        const windowEnd = lastFlushAt + TIMING_REALTIME_INVALIDATION_WINDOW_MS;
        flushTimer = setTimeout(
          flush,
          Math.max(
            TIMING_REALTIME_INVALIDATION_BATCH_MS,
            windowEnd - Date.now(),
          ),
        );
      }
    };

    const unsubscribe = realtimeSubscriptions.subscribeAll(handler);
    return () => {
      unsubscribe();
      if (flushTimer) clearTimeout(flushTimer);
    };
  }, [queryClient]);
}

type StreamFlight = {
  hasConnected: boolean;
  connected: boolean;
  pendingInvalidate: ReturnType<typeof setTimeout> | null;
  lastInvalidateAt: number;
};

const idleFlight = (): StreamFlight => ({
  hasConnected: false,
  connected: false,
  pendingInvalidate: null,
  lastInvalidateAt: 0,
});

/**
 * Catch-up invalidation when a stream reconnects after a drop: events emitted
 * during the gap never reach the browser, so the queries that stream's events
 * refresh are re-read. A first connect (page load) catches up nothing.
 *
 * Debounced and rate-limited per stream, so a burst of reconnects collapses
 * into a single invalidation and no more than one fires per
 * TIMING_REALTIME_CATCHUP_MIN_INTERVAL_MS.
 */
function useCatchUpOnStreamReconnect(): void {
  const queryClient = useQueryClient();
  const flightsRef = useRef(new Map<RealtimeStreamKind, StreamFlight>());

  useEffect(() => {
    const flights = flightsRef.current;
    const flightOf = (kind: RealtimeStreamKind): StreamFlight => {
      const flight = flights.get(kind) ?? idleFlight();
      flights.set(kind, flight);
      return flight;
    };

    const scheduleCatchUp = (
      kind: RealtimeStreamKind,
      flight: StreamFlight,
    ) => {
      if (flight.pendingInvalidate) clearTimeout(flight.pendingInvalidate);
      flight.pendingInvalidate = setTimeout(() => {
        flight.pendingInvalidate = null;
        const now = Date.now();
        if (
          now - flight.lastInvalidateAt <
          TIMING_REALTIME_CATCHUP_MIN_INTERVAL_MS
        ) {
          return;
        }
        flight.lastInvalidateAt = now;
        catchUpKeys(kind).forEach((queryKey) =>
          queryClient.invalidateQueries({ queryKey }),
        );
      }, TIMING_REALTIME_CATCHUP_DEBOUNCE_MS);
    };

    const unsubscribe = realtimeLifecycle.onConnectionEvent((evt) => {
      const flight = flightOf(evt.streamKind);
      if (evt.type === REALTIME_LIFECYCLE_EVENT.CONNECTED) {
        const isReconnect = flight.hasConnected && !flight.connected;
        flight.hasConnected = true;
        flight.connected = true;
        if (isReconnect) scheduleCatchUp(evt.streamKind, flight);
      } else if (
        evt.type === REALTIME_LIFECYCLE_EVENT.DISCONNECTED ||
        evt.type === REALTIME_LIFECYCLE_EVENT.ERROR
      ) {
        flight.connected = false;
      }
    });

    return () => {
      unsubscribe();
      flights.forEach((flight) => {
        if (flight.pendingInvalidate) clearTimeout(flight.pendingInvalidate);
        flight.pendingInvalidate = null;
      });
    };
  }, [queryClient]);
}

/**
 * Fallback polling: while the signed-in user's streams are not all connected,
 * no event announces a change, so every open view is re-read at
 * TIMING_FALLBACK_POLL_MS until they are.
 */
function useFallbackPolling(state: ConnectionState): void {
  const queryClient = useQueryClient();
  const { isAuthenticated } = useSession();

  useEffect(() => {
    if (!isAuthenticated || state === CONNECTION_STATE.CONNECTED) return;
    const timer = setInterval(
      () => queryClient.invalidateQueries(),
      TIMING_FALLBACK_POLL_MS,
    );
    return () => clearInterval(timer);
  }, [queryClient, isAuthenticated, state]);
}

/**
 * Combined hook for realtime connection + cache invalidation
 *
 * Use this in App.tsx or at the top level to enable realtime updates
 */
export function useRealtimeUpdates() {
  const connection = useRealtimeConnection();
  useRealtimeInvalidation();
  useCatchUpOnStreamReconnect();
  useFallbackPolling(connection.state);

  return connection;
}
