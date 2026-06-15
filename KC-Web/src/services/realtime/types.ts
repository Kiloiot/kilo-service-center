/**
 * Realtime Service Types
 *
 * Type definitions for real-time gRPC streaming communication.
 * Step 0A.3 will implement gRPC streaming using StreamMessages RPC.
 */

import type { REALTIME_STREAM_KIND } from "@constants/app";

export type RealtimeStreamKind =
  (typeof REALTIME_STREAM_KIND)[keyof typeof REALTIME_STREAM_KIND];

/**
 * Connection state for realtime streaming
 */
export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting";

/**
 * Realtime event types matching backend events
 */
export type RealtimeEventType =
  // Uplink events
  | "uplink.received"
  // Downlink events
  | "downlink.queued"
  | "downlink.sent"
  | "downlink.acknowledged"
  | "downlink.failed"
  | "downlink.revoked"
  // Endpoint events
  | "endpoint.attached"
  | "endpoint.detached"
  // Base station events
  | "basestation.online"
  | "basestation.offline"
  // SCACI events
  | "scaci.session.opened"
  | "scaci.session.closed"
  | "scaci.error"
  // Generic event
  | "event.received";

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
  | "realtime_connect"
  | "realtime_connected"
  | "realtime_error"
  | "realtime_reconnect"
  | "realtime_disconnected"
  | "realtime_info";

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
  /** Which underlying stream emitted this event. Absent for service-wide
   * events that don't belong to a single stream. The catch-up hook in
   * useRealtime.ts branches on this to invalidate the right caches when a
   * particular stream reconnects after a drop. */
  streamKind?: RealtimeStreamKind;
}

/**
 * Connection event listener
 */
export type ConnectionEventListener = (event: ConnectionEvent) => void;
