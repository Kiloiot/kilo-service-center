/**
 * Realtime Service Types
 *
 * Type definitions for real-time gRPC streaming communication.
 */

import type {
  CONNECTION_STATE,
  REALTIME_EVENT_TYPE,
  REALTIME_LIFECYCLE_EVENT,
  REALTIME_STREAM_KIND,
} from "@constants/app";

export type RealtimeStreamKind =
  (typeof REALTIME_STREAM_KIND)[keyof typeof REALTIME_STREAM_KIND];

/**
 * Connection state for realtime streaming
 */
export type ConnectionState =
  (typeof CONNECTION_STATE)[keyof typeof CONNECTION_STATE];

/** Every realtime event type, derived from the REALTIME_EVENT_TYPE catalog. */
export type RealtimeEventType =
  (typeof REALTIME_EVENT_TYPE)[keyof typeof REALTIME_EVENT_TYPE];

/**
 * Base realtime event structure
 */
export interface RealtimeEvent {
  /** Event type identifier */
  type: RealtimeEventType;
  /** Event timestamp (ISO string) */
  timestamp: string;
  /** Organization ID the event belongs to */
  organizationId?: string;
  /** Event-specific payload */
  payload?: Record<string, unknown>;
}

/**
 * Event handler function type
 */
export type EventHandler = (event: RealtimeEvent) => void;

/**
 * Connection state change listener
 */
export type StateChangeListener = (state: ConnectionState) => void;

/**
 * Connection error information
 */
export interface ConnectionError {
  /** Error message */
  message: string;
  /** Timestamp when error occurred */
  timestamp: Date;
  /** Optional error code (e.g., gRPC status code) */
  code?: number;
}

/**
 * Connection error listener
 */
export type ErrorListener = (error: ConnectionError | null) => void;

/**
 * Connection event types for activity feed
 */
export type ConnectionEventType =
  (typeof REALTIME_LIFECYCLE_EVENT)[keyof typeof REALTIME_LIFECYCLE_EVENT];

/**
 * Connection event for activity timeline
 */
export interface ConnectionEvent {
  /** Event type */
  type: ConnectionEventType;
  /** ISO timestamp string */
  timestamp: string;
  /** Human-readable message */
  message: string;
  /** Connection URL (for connect events) */
  url?: string;
  /** Error/close code */
  code?: number;
  /** Reconnect attempt number */
  attempt?: number;
  /** Delay until next reconnect (ms) */
  delayMs?: number;
  /** The stream that emitted the event; the catch-up hook in useRealtime.ts
   * refreshes after the event stream reconnects. */
  streamKind: RealtimeStreamKind;
}

/**
 * Connection event listener
 */
export type ConnectionEventListener = (event: ConnectionEvent) => void;

/** The streams a user's roles open besides the event stream every role opens. */
export interface RealtimeStreamSet {
  /** The uplink stream, for roles that may read uplinks. */
  uplinks: boolean;
}

/** Connection lifecycle and state observation of the realtime singleton. */
export interface RealtimeLifecycle {
  reconnectWithOrg(
    organizationId: string,
    userId: string,
    streams: RealtimeStreamSet,
  ): void;
  reset(): void;
  getState(): ConnectionState;
  getLastError(): ConnectionError | null;
  onStateChange(listener: StateChangeListener): () => void;
  onErrorChange(listener: ErrorListener): () => void;
  onConnectionEvent(listener: ConnectionEventListener): () => void;
}

/** Event fan-out to application handlers. */
export interface RealtimeSubscriptions {
  subscribeAll(handler: EventHandler): () => void;
}

/** The live updates of the base station pages open. */
export interface RealtimeStreams {
  /** Keeps the station's uplinks live until the returned stop is called. */
  watchBaseStation(bsEui: string): () => void;
}
