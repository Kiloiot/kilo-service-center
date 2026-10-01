/**
 * Connection Status Hook
 *
 * Provides real-time connection status for UI components.
 * Maps connection states to UI-friendly values.
 */

import { useEffect, useState } from "react";

import {
  type ConnectionError,
  type ConnectionState,
  realtimeLifecycle,
} from "@services/realtime";
import { CONNECTION_STATE, UI_CONNECTION_STATUS } from "@constants/app";

/**
 * UI-friendly connection status
 */
export type UIConnectionStatus =
  (typeof UI_CONNECTION_STATUS)[keyof typeof UI_CONNECTION_STATUS];

/**
 * Map internal connection state to UI status
 */
function mapStateToStatus(state: ConnectionState): UIConnectionStatus {
  switch (state) {
    case CONNECTION_STATE.CONNECTED:
      return UI_CONNECTION_STATUS.CONNECTED;
    case CONNECTION_STATE.CONNECTING:
    case CONNECTION_STATE.RECONNECTING:
      return UI_CONNECTION_STATUS.RECONNECTING;
    case CONNECTION_STATE.DISCONNECTED:
    default:
      return UI_CONNECTION_STATUS.OFFLINE;
  }
}

/**
 * Hook for connection status display in UI components
 *
 * @returns Connection status, convenience flags, and last error
 */
export function useConnectionStatus() {
  const [status, setStatus] = useState<UIConnectionStatus>(() =>
    mapStateToStatus(realtimeLifecycle.getState()),
  );
  const [lastError, setLastError] = useState<ConnectionError | null>(() =>
    realtimeLifecycle.getLastError(),
  );

  useEffect(() => {
    const unsubscribeState = realtimeLifecycle.onStateChange((state) => {
      setStatus(mapStateToStatus(state));
    });

    const unsubscribeError = realtimeLifecycle.onErrorChange((error) => {
      setLastError(error);
    });

    return () => {
      unsubscribeState();
      unsubscribeError();
    };
  }, []);

  return {
    status,
    isConnected: status === UI_CONNECTION_STATUS.CONNECTED,
    isReconnecting: status === UI_CONNECTION_STATUS.RECONNECTING,
    isOffline: status === UI_CONNECTION_STATUS.OFFLINE,
    lastError,
  };
}
