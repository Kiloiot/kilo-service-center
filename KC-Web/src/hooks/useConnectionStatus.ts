/**
 * Connection Status Hook
 *
 * Provides real-time connection status for UI components.
 * Maps connection states to UI-friendly values.
 */

import { useEffect, useState } from 'react';

import { type ConnectionError, type ConnectionState, realtimeService } from '@services/realtime';

/**
 * UI-friendly connection status
 */
export type UIConnectionStatus = 'connected' | 'reconnecting' | 'offline';

/**
 * Map internal connection state to UI status
 */
function mapStateToStatus(state: ConnectionState): UIConnectionStatus {
  switch (state) {
    case 'connected':
      return 'connected';
    case 'connecting':
    case 'reconnecting':
      return 'reconnecting';
    case 'disconnected':
    default:
      return 'offline';
  }
}

/**
 * Hook for connection status display in UI components
 *
 * @returns Connection status, convenience flags, and last error
 */
export function useConnectionStatus() {
  const [status, setStatus] = useState<UIConnectionStatus>(() =>
    mapStateToStatus(realtimeService.getState())
  );
  const [lastError, setLastError] = useState<ConnectionError | null>(() =>
    realtimeService.getLastError()
  );

  useEffect(() => {
    const unsubscribeState = realtimeService.onStateChange((state) => {
      setStatus(mapStateToStatus(state));
    });

    const unsubscribeError = realtimeService.onErrorChange((error) => {
      setLastError(error);
    });

    return () => {
      unsubscribeState();
      unsubscribeError();
    };
  }, []);

  return {
    status,
    isConnected: status === 'connected',
    isReconnecting: status === 'reconnecting',
    isOffline: status === 'offline',
    lastError,
  };
}
