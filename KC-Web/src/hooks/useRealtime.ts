/**
 * Realtime Hooks
 *
 * React hooks for realtime connection management and cache invalidation.
 */

import { useCallback, useEffect, useState } from "react";

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

function invalidateBaseStation(qc: QueryClient, bsEui: string): void {
  qc.invalidateQueries({ queryKey: queryKeys.baseStations.detail(bsEui) });
  qc.invalidateQueries({ queryKey: queryKeys.baseStations.messages(bsEui) });
  qc.invalidateQueries({ queryKey: queryKeys.baseStations.activity(bsEui) });
}

function invalidateEndpoint(qc: QueryClient, epEui: string): void {
  qc.invalidateQueries({ queryKey: queryKeys.endpoints.detail(epEui) });
  qc.invalidateQueries({ queryKey: queryKeys.endpoints.activity(epEui) });
}

/** Per-event-type targeted invalidations (run after the generic map). */
function applyTargetedInvalidations(
  qc: QueryClient,
  event: RealtimeEvent,
): void {
  if (
    event.type === "basestation.online" ||
    event.type === "basestation.offline"
  ) {
    const bsEui = extractBsEuiFromPayload(event.payload);
    if (bsEui) invalidateBaseStation(qc, bsEui);
    return;
  }

  if (event.type === "uplink.received" && event.payload?.message) {
    const msg = event.payload.message as { bsEui?: string; epEui?: string };
    if (msg.bsEui) invalidateBaseStation(qc, msg.bsEui);
    if (msg.epEui) invalidateEndpoint(qc, msg.epEui);
    return;
  }

  if (
    event.type === "endpoint.attached" ||
    event.type === "endpoint.detached"
  ) {
    const epEui = extractEpEuiFromPayload(event.payload);
    if (epEui) invalidateEndpoint(qc, epEui);
    return;
  }

  if (
    event.type === "downlink.queued" ||
    event.type === "downlink.sent" ||
    event.type === "downlink.failed" ||
    event.type === "downlink.revoked"
  ) {
    const epEui = extractEpEuiFromPayload(event.payload);
    if (epEui) {
      qc.invalidateQueries({ queryKey: queryKeys.endpoints.detail(epEui) });
    }
  }
}

/**
 * Subscribe to every realtime event for the lifetime of the calling component
 * and dispatch each to the handler. Cleans up on unmount or handler change.
 */
function useSubscriptionLifecycle(
  handler: (event: RealtimeEvent) => void,
): void {
  useEffect(() => realtimeService.subscribeAll(handler), [handler]);
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
 * Subscribes to all realtime events and invalidates relevant React Query caches.
 */
export function useRealtimeInvalidation() {
  const queryClient = useQueryClient();

  const handler = useCallback(
    (event: RealtimeEvent) => {
      const keys = EVENT_INVALIDATION_MAP[event.type];
      if (keys) {
        keys.forEach((key) => {
          queryClient.invalidateQueries({
            queryKey: key as readonly unknown[],
          });
        });
      }
      applyTargetedInvalidations(queryClient, event);
    },
    [queryClient],
  );

  useSubscriptionLifecycle(handler);
}

/**
 * Combined hook for realtime connection + cache invalidation
 *
 * Use this in App.tsx or at the top level to enable realtime updates
 */
export function useRealtimeUpdates() {
  const { state, isConnected, isReconnecting } = useRealtimeConnection();
  useRealtimeInvalidation();

  return { state, isConnected, isReconnecting };
}
