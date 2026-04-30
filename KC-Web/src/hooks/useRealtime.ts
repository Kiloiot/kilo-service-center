/**
 * Realtime Hooks
 *
 * React hooks for realtime connection management and cache invalidation.
 */

import { useEffect, useState } from 'react';

import { useQueryClient } from '@tanstack/react-query';

import {
  type ConnectionState,
  type RealtimeEvent,
  type RealtimeEventType,
  realtimeService,
} from '@services/realtime';
import { useOrganization } from '@contexts/OrganizationContext';
import { useSession } from '@contexts/SessionContext';
import { queryKeys } from '@config/query-keys';

/**
 * Event type to query keys invalidation mapping.
 * When an event is received, these query keys are invalidated.
 */
const EVENT_INVALIDATION_MAP: Partial<Record<RealtimeEventType, readonly (readonly unknown[])[]>> =
  {
    'uplink.received': [queryKeys.endpoints.all, queryKeys.events.all, queryKeys.dashboard.all],
    'endpoint.attached': [queryKeys.endpoints.all, queryKeys.events.all, queryKeys.dashboard.all],
    'endpoint.detached': [queryKeys.endpoints.all, queryKeys.events.all, queryKeys.dashboard.all],
    'basestation.online': [
      queryKeys.baseStations.all,
      queryKeys.dashboard.all,
      queryKeys.events.all,
    ],
    'basestation.offline': [
      queryKeys.baseStations.all,
      queryKeys.dashboard.all,
      queryKeys.events.all,
    ],
    'downlink.queued': [queryKeys.endpoints.all, queryKeys.events.all, queryKeys.scaci.all],
    'downlink.sent': [queryKeys.endpoints.all, queryKeys.events.all, queryKeys.scaci.all],
    'downlink.failed': [queryKeys.endpoints.all, queryKeys.events.all, queryKeys.scaci.all],
    'downlink.revoked': [queryKeys.endpoints.all, queryKeys.events.all, queryKeys.scaci.all],
    'scaci.session.opened': [queryKeys.scaci.all, queryKeys.events.all],
    'scaci.session.closed': [queryKeys.scaci.all, queryKeys.events.all],
    'scaci.error': [queryKeys.scaci.all, queryKeys.events.all],
    'event.received': [queryKeys.events.all, queryKeys.dashboard.all],
  };

/**
 * Extract bsEui from event payload for targeted invalidation.
 * Extraction priority: sourceName → data (bytes) → details → sourceId
 *
 * @param payload Event payload containing backend event data
 * @returns bsEui string if found, undefined otherwise
 */
function extractBsEuiFromPayload(payload?: Record<string, unknown>): string | undefined {
  if (!payload?.event || typeof payload.event !== 'object') {
    return undefined;
  }

  const event = payload.event as Record<string, unknown>;

  // Primary: Use sourceName (backend sets to EUI string)
  if (typeof event.sourceName === 'string' && event.sourceName) {
    return event.sourceName;
  }

  // Secondary: Parse event.data (bytes field from proto)
  if (event.data) {
    let dataStr: string | undefined;

    if (event.data instanceof Uint8Array) {
      dataStr = new TextDecoder().decode(event.data);
    } else if (typeof event.data === 'string') {
      dataStr = event.data;
    }

    if (dataStr) {
      try {
        const parsed = JSON.parse(dataStr) as Record<string, unknown>;
        if (typeof parsed.bsEui === 'string') {
          return parsed.bsEui;
        }
      } catch {
        // Invalid JSON, continue to next fallback
      }
    }
  }

  // Tertiary: Look in details for bsEui key (legacy format)
  if (event.details) {
    let details: Record<string, unknown> | undefined;

    if (typeof event.details === 'string') {
      try {
        details = JSON.parse(event.details);
      } catch {
        // Invalid JSON, skip
      }
    } else if (typeof event.details === 'object') {
      details = event.details as Record<string, unknown>;
    }

    if (details && typeof details.bsEui === 'string') {
      return details.bsEui;
    }
  }

  // Last fallback: sourceId
  if (typeof event.sourceId === 'string' && event.sourceId) {
    return event.sourceId;
  }

  return undefined;
}

/**
 * Extract epEui from event payload for targeted endpoint invalidation.
 * Extraction priority: sourceName → data.epEui → message.epEui
 *
 * @param payload Event payload containing backend event data
 * @returns epEui string if found, undefined otherwise
 */
function extractEpEuiFromPayload(payload?: Record<string, unknown>): string | undefined {
  if (!payload) {
    return undefined;
  }

  // Check event object for sourceName (for attach/detach events)
  if (payload.event && typeof payload.event === 'object') {
    const event = payload.event as Record<string, unknown>;

    if (typeof event.sourceName === 'string' && event.sourceName) {
      return event.sourceName;
    }

    // Parse event.data for epEui
    if (event.data) {
      let dataStr: string | undefined;
      if (event.data instanceof Uint8Array) {
        dataStr = new TextDecoder().decode(event.data);
      } else if (typeof event.data === 'string') {
        dataStr = event.data;
      }

      if (dataStr) {
        try {
          const parsed = JSON.parse(dataStr) as Record<string, unknown>;
          if (typeof parsed.epEui === 'string') {
            return parsed.epEui;
          }
        } catch {
          // Invalid JSON, continue
        }
      }
    }
  }

  // Check message object for epEui (for uplink events)
  if (payload.message && typeof payload.message === 'object') {
    const msg = payload.message as { epEui?: string };
    if (typeof msg.epEui === 'string') {
      return msg.epEui;
    }
  }

  return undefined;
}

/**
 * Hook for managing realtime connection state
 * Reconnects when organization changes
 * Only connects when user is authenticated (realtime endpoint requires auth)
 *
 * @returns Current connection state and isConnected flag
 */
export function useRealtimeConnection() {
  const [state, setState] = useState<ConnectionState>('disconnected');
  const { organizationId, userId } = useOrganization();
  const { isAuthenticated, isHydrated } = useSession();

  // Subscribe to state changes (runs once)
  useEffect(() => {
    const unsubscribe = realtimeService.onStateChange(setState);
    return () => {
      unsubscribe();
    };
  }, []);

  // Connect/reconnect when organization changes (including initial mount)
  // Only connect when authenticated (realtime endpoint requires auth token)
  useEffect(() => {
    // Wait for session hydration and authentication
    if (!isHydrated || !isAuthenticated) {
      return;
    }

    // Only connect when both org and user context are available
    // (required by org_resolver interceptor for authenticated principals)
    if (!organizationId || !userId) {
      return;
    }
    realtimeService.reconnectWithOrg(organizationId, userId);
    // Note: Don't disconnect on unmount - singleton persists across route changes
    // Disconnect only happens on app close via beforeunload event
  }, [organizationId, userId, isAuthenticated, isHydrated]);

  return {
    state,
    isConnected: state === 'connected',
    isReconnecting: state === 'reconnecting',
  };
}

/**
 * Hook for automatic query invalidation based on realtime events.
 * Subscribes to all realtime events and invalidates relevant React Query caches.
 */
export function useRealtimeInvalidation() {
  const queryClient = useQueryClient();

  useEffect(() => {
    return realtimeService.subscribeAll((event: RealtimeEvent) => {
      const keys = EVENT_INVALIDATION_MAP[event.type];
      if (keys) {
        keys.forEach((key) => {
          queryClient.invalidateQueries({ queryKey: key as readonly unknown[] });
        });
      }

      // Targeted invalidation for BS online/offline events
      if (event.type === 'basestation.online' || event.type === 'basestation.offline') {
        const bsEui = extractBsEuiFromPayload(event.payload);
        if (bsEui) {
          queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.detail(bsEui) });
          queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.messages(bsEui) });
          queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.activity(bsEui) });
        }
      }

      // Targeted invalidation for uplink messages (refresh BS detail/messages/activity)
      if (event.type === 'uplink.received' && event.payload?.message) {
        const msg = event.payload.message as { bsEui?: string; epEui?: string };
        if (msg.bsEui) {
          queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.detail(msg.bsEui) });
          queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.messages(msg.bsEui) });
          queryClient.invalidateQueries({ queryKey: queryKeys.baseStations.activity(msg.bsEui) });
        }
        // Also invalidate endpoint messages/activity for the specific endpoint
        if (msg.epEui) {
          queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.detail(msg.epEui) });
          queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.activity(msg.epEui) });
        }
      }

      // Targeted invalidation for endpoint attach/detach events
      if (event.type === 'endpoint.attached' || event.type === 'endpoint.detached') {
        const epEui = extractEpEuiFromPayload(event.payload);
        if (epEui) {
          queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.detail(epEui) });
          queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.activity(epEui) });
        }
      }

      // Targeted invalidation for downlink events
      if (
        event.type === 'downlink.queued' ||
        event.type === 'downlink.sent' ||
        event.type === 'downlink.failed' ||
        event.type === 'downlink.revoked'
      ) {
        const epEui = extractEpEuiFromPayload(event.payload);
        if (epEui) {
          queryClient.invalidateQueries({ queryKey: queryKeys.endpoints.detail(epEui) });
        }
      }
    });
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

  return { state, isConnected, isReconnecting };
}
