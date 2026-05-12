/**
 * RealtimeService
 *
 * Singleton service for real-time event streaming via gRPC-web.
 * Features:
 * - gRPC server-streaming (StreamMessages RPC)
 * - Single connection per app (not per page)
 * - Exponential backoff reconnection
 * - Event subscription with type filtering
 * - Connection state management
 *
 * @see src/services/grpc/client.ts for gRPC implementation
 */

import { grpc } from "@improbable-eng/grpc-web";

import * as ServiceModule from "@services/grpc/kilocenter_pb_service";
const { KiloCenterService } = ServiceModule;
import * as pb from "@services/grpc/kilocenter_pb";
import { logger } from "@utils/logger";
import { storageService } from "@utils/storage";
import {
  BACKEND_EVENT_TO_REALTIME,
  HEADER_VALUES,
  HEADERS,
  REALTIME_EVENT_TYPE,
  STORAGE_KEYS,
  TIMING_RECONNECT_BASE_DELAY,
  TIMING_RECONNECT_MAX_DELAY,
} from "@constants/app";
import { REALTIME_MESSAGES } from "@constants/messages";
import { grpcUrl } from "@config/env";

import type {
  ConnectionError,
  ConnectionEvent,
  ConnectionEventListener,
  ConnectionState,
  ErrorListener,
  EventHandler,
  RealtimeEvent,
  RealtimeEventType,
  StateChangeListener,
} from "./types";

/** Maximum number of connection events to keep in buffer */
const CONNECTION_EVENT_BUFFER_SIZE = 20;

/**
 * Validate incoming event structure
 */
function isValidRealtimeEvent(event: unknown): event is RealtimeEvent {
  return (
    typeof event === "object" &&
    event !== null &&
    "type" in event &&
    typeof (event as RealtimeEvent).type === "string" &&
    "timestamp" in event &&
    typeof (event as RealtimeEvent).timestamp === "string"
  );
}

/**
 * RealtimeService singleton class
 *
 * Uses gRPC-web server-streaming for real-time updates.
 * StreamMessages RPC provides a continuous stream of messages/events.
 */
class RealtimeService {
  private static instance: RealtimeService;
  private messageStream: grpc.Request | null = null;
  private handlers = new Map<RealtimeEventType, Set<EventHandler>>();
  private wildcardHandlers = new Set<EventHandler>();
  private reconnectAttempts = 0;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private state: ConnectionState = "disconnected";
  private stateListeners = new Set<StateChangeListener>();
  private lastError: ConnectionError | null = null;
  private errorListeners = new Set<ErrorListener>();
  private connectionEvents: ConnectionEvent[] = [];
  private connectionEventListeners = new Set<ConnectionEventListener>();
  private currentOrgId: string | null = null;
  private currentUserId: string | null = null;
  private reconnecting = false; // Guard against reconnect loops

  // Separate base station stream state (must not interfere with global stream)
  private baseStationStream: grpc.Request | null = null;
  private baseStationReconnectAttempts = 0;
  private baseStationReconnectTimeout: ReturnType<typeof setTimeout> | null =
    null;

  // Event stream state
  private eventStream: grpc.Request | null = null;
  private eventStreamReconnectAttempts = 0;
  private eventStreamReconnectTimeout: ReturnType<typeof setTimeout> | null =
    null;

  /**
   * Get the singleton instance
   */
  static getInstance(): RealtimeService {
    if (!RealtimeService.instance) {
      RealtimeService.instance = new RealtimeService();
    }
    return RealtimeService.instance;
  }

  /**
   * Private constructor enforces singleton pattern
   */
  private constructor() {
    // Initialize empty - connection is lazy
  }

  /**
   * Build gRPC metadata with auth token, org ID, and user ID
   * Uses HEADERS constants for consistency with api.ts
   */
  private buildMetadata(): grpc.Metadata {
    const metadata = new grpc.Metadata();

    const token = storageService.getItem(STORAGE_KEYS.AUTH_TOKEN);
    if (token) {
      metadata.set(
        HEADERS.AUTHORIZATION.toLowerCase(),
        `${HEADER_VALUES.AUTH_BEARER_PREFIX}${token}`,
      );
    }
    if (this.currentOrgId) {
      metadata.set(HEADERS.X_ORGANIZATION_ID.toLowerCase(), this.currentOrgId);
    }
    if (this.currentUserId) {
      metadata.set(HEADERS.X_USER_ID.toLowerCase(), this.currentUserId);
    }
    return metadata;
  }

  /**
   * Map gRPC Message to RealtimeEvent
   * Converts protobuf Timestamp to ISO string
   */
  private mapMessageToRealtimeEvent(msg: pb.Message.AsObject): RealtimeEvent {
    // Convert protobuf Timestamp to ISO string
    let timestamp = new Date().toISOString();
    if (msg.receivedAt) {
      const seconds = msg.receivedAt.seconds || 0;
      const nanos = msg.receivedAt.nanos || 0;
      timestamp = new Date(seconds * 1000 + nanos / 1000000).toISOString();
    }

    return {
      type: REALTIME_EVENT_TYPE.UPLINK_RECEIVED,
      timestamp,
      organizationId: this.currentOrgId || undefined,
      payload: { message: msg },
    };
  }

  /**
   * Map gRPC BaseStationMessage to RealtimeEvent
   */
  private mapBaseStationMessageToRealtimeEvent(
    msg: pb.BaseStationMessage.AsObject,
  ): RealtimeEvent {
    let timestamp = new Date().toISOString();
    if (msg.receivedAt) {
      const seconds = msg.receivedAt.seconds || 0;
      const nanos = msg.receivedAt.nanos || 0;
      timestamp = new Date(seconds * 1000 + nanos / 1000000).toISOString();
    }
    return {
      type: REALTIME_EVENT_TYPE.UPLINK_RECEIVED,
      timestamp,
      organizationId: this.currentOrgId || undefined,
      payload: { message: msg },
    };
  }

  /**
   * Map gRPC Event to RealtimeEvent
   * Maps backend underscore event_type to UI dotted format
   */
  private mapEventToRealtimeEvent(
    event: pb.Event.AsObject,
  ): RealtimeEvent | null {
    // Map backend event_type to UI realtime event type, fallback to EVENT_RECEIVED
    const realtimeType =
      BACKEND_EVENT_TO_REALTIME[event.eventType] ||
      REALTIME_EVENT_TYPE.EVENT_RECEIVED;

    // Preserve backend timestamp for ordering; fallback to now if missing
    let timestamp = new Date().toISOString();
    if (event.timestamp) {
      const seconds = event.timestamp.seconds || 0;
      const nanos = event.timestamp.nanos || 0;
      timestamp = new Date(seconds * 1000 + nanos / 1000000).toISOString();
    }

    return {
      type: realtimeType,
      timestamp,
      organizationId: this.currentOrgId || undefined,
      payload: { event, originalEventType: event.eventType },
    };
  }

  /**
   * Get current connection state
   */
  getState(): ConnectionState {
    return this.state;
  }

  /**
   * Get current organization ID
   */
  getCurrentOrgId(): string | null {
    return this.currentOrgId;
  }

  /**
   * Get last connection error
   */
  getLastError(): ConnectionError | null {
    return this.lastError;
  }

  /**
   * Listen to connection error changes
   * @returns Unsubscribe function
   */
  onErrorChange(listener: ErrorListener): () => void {
    this.errorListeners.add(listener);
    // Immediately notify of current error state
    listener(this.lastError);

    return () => {
      this.errorListeners.delete(listener);
    };
  }

  /**
   * Get connection events history (most recent first)
   */
  getConnectionEvents(): ConnectionEvent[] {
    return [...this.connectionEvents];
  }

  /**
   * Listen to new connection events
   * @returns Unsubscribe function
   */
  onConnectionEvent(listener: ConnectionEventListener): () => void {
    this.connectionEventListeners.add(listener);

    return () => {
      this.connectionEventListeners.delete(listener);
    };
  }

  /**
   * Connect to the gRPC streaming service
   * @param organizationId Optional org ID for multi-tenant event filtering
   *                       If undefined, preserves current org (for reconnects)
   * @param userId Optional user ID
   */
  connect(organizationId?: string, userId?: string): void {
    // Don't reconnect if already connected or connecting
    if (this.messageStream && this.state === "connected") {
      return;
    }

    // Apply incoming context FIRST (before any state changes)
    if (organizationId !== undefined) {
      this.currentOrgId = organizationId || null;
    }
    if (userId !== undefined) {
      this.currentUserId = userId || null;
    }

    // Fail closed: refuse connection without required metadata
    if (!this.hasRequiredContext()) {
      this.stopAllReconnectTimers();
      logger.warn(REALTIME_MESSAGES.CONTEXT_INCOMPLETE);
      return;
    }

    this.setState("connecting");

    try {
      const host = grpcUrl || "";

      // Emit connect attempt event
      this.emitConnectionEvent({
        type: "realtime_connect",
        message: `${REALTIME_MESSAGES.CONNECTING_AT} ${host}`,
        url: host,
      });

      // Build request - empty request for general stream, org filter via metadata
      const request = new pb.StreamMessagesRequest();

      // Start gRPC server-streaming call
      this.messageStream = grpc.invoke(KiloCenterService.StreamMessages, {
        host,
        request,
        metadata: this.buildMetadata(),
        onHeaders: () => {
          // Stream connected successfully
          this.reconnectAttempts = 0;
          this.reconnecting = false;
          this.setError(null);
          this.setState("connected");

          this.emitConnectionEvent({
            type: "realtime_connected",
            message: REALTIME_MESSAGES.CONNECTED,
          });

          // Start event stream after message stream connected
          this.connectEventStream();
        },
        onMessage: (msg: pb.Message) => {
          // Convert proto message to RealtimeEvent and dispatch
          const event = this.mapMessageToRealtimeEvent(msg.toObject());
          this.dispatch(event);
        },
        onEnd: (code: grpc.Code, message: string) => {
          this.messageStream = null;

          if (code !== grpc.Code.OK) {
            const errorMessage =
              message || `${REALTIME_MESSAGES.STREAM_ENDED_ERROR} ${code}`;

            this.emitConnectionEvent({
              type: "realtime_disconnected",
              message: errorMessage,
              code,
            });

            this.setError({
              message: errorMessage,
              timestamp: new Date(),
              code,
            });
          } else {
            this.emitConnectionEvent({
              type: "realtime_disconnected",
              message: REALTIME_MESSAGES.STREAM_ENDED_NORMAL,
            });
          }

          // Schedule reconnect regardless of how stream ended
          this.scheduleReconnect();
        },
      });
    } catch (error) {
      const errorMessage =
        error instanceof Error
          ? error.message
          : REALTIME_MESSAGES.FAILED_ESTABLISH;
      logger.error(REALTIME_MESSAGES.FAILED_ESTABLISH, error);

      this.emitConnectionEvent({
        type: "realtime_error",
        message: errorMessage,
      });

      this.setError({
        message: errorMessage,
        timestamp: new Date(),
      });
      this.scheduleReconnect();
    }
  }

  /**
   * Reconnect with a new organization ID
   * Guards against rapid reconnect loops
   */
  reconnectWithOrg(organizationId: string, userId: string): void {
    // Guard: skip if already reconnecting or same org+user context is active
    if (
      this.reconnecting ||
      (this.currentOrgId === organizationId && this.currentUserId === userId)
    ) {
      return;
    }

    this.reconnecting = true;
    this.disconnect();
    this.connect(organizationId, userId);
  }

  /**
   * Disconnect from the gRPC streaming service
   */
  disconnect(): void {
    // Clear primary stream reconnect
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.messageStream) {
      this.messageStream.close();
      this.messageStream = null;
    }

    // Close base station stream with its own state
    this.disconnectBaseStationStream();

    // Close event stream
    this.disconnectEventStream();

    this.setState("disconnected");
    this.reconnectAttempts = 0;
  }

  /**
   * Subscribe to a specific event type
   * @returns Unsubscribe function
   */
  subscribe(type: RealtimeEventType, handler: EventHandler): () => void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set());
    }
    this.handlers.get(type)!.add(handler);

    return () => {
      this.handlers.get(type)?.delete(handler);
    };
  }

  /**
   * Subscribe to all events
   * @returns Unsubscribe function
   */
  subscribeAll(handler: EventHandler): () => void {
    this.wildcardHandlers.add(handler);

    return () => {
      this.wildcardHandlers.delete(handler);
    };
  }

  /**
   * Listen to connection state changes
   * @returns Unsubscribe function
   */
  onStateChange(listener: StateChangeListener): () => void {
    this.stateListeners.add(listener);
    // Immediately notify of current state
    listener(this.state);

    return () => {
      this.stateListeners.delete(listener);
    };
  }

  /**
   * Dispatch event to handlers
   * Includes client-side org filtering as defense-in-depth (supplements server isolation)
   */
  private dispatch(event: unknown): void {
    if (!isValidRealtimeEvent(event)) {
      logger.warn("Invalid realtime event received:", event);
      return;
    }

    // Defense-in-depth: client-side filter supplements server-side tenant isolation
    // This is best-effort and does NOT replace server-side isolation
    const eventOrgId = (event as RealtimeEvent & { organizationId?: string })
      .organizationId;
    if (eventOrgId && this.currentOrgId && eventOrgId !== this.currentOrgId) {
      logger.warn(
        "Received event for different org, dropping (defense-in-depth)",
      );
      return;
    }

    // Dispatch to type-specific handlers
    const typeHandlers = this.handlers.get(event.type);
    typeHandlers?.forEach((handler) => {
      try {
        handler(event);
      } catch (error) {
        logger.error(`Error in event handler for ${event.type}:`, error);
      }
    });

    // Dispatch to wildcard handlers
    this.wildcardHandlers.forEach((handler) => {
      try {
        handler(event);
      } catch (error) {
        logger.error("Error in wildcard event handler:", error);
      }
    });
  }

  /**
   * Update connection state and notify listeners
   */
  private setState(newState: ConnectionState): void {
    if (this.state === newState) return;
    this.state = newState;
    this.stateListeners.forEach((listener) => listener(newState));
  }

  /**
   * Update last error and notify listeners
   */
  private setError(error: ConnectionError | null): void {
    this.lastError = error;
    this.errorListeners.forEach((listener) => listener(error));
  }

  /**
   * Emit a connection event (for activity feed)
   * Maintains a ring buffer of recent events
   */
  private emitConnectionEvent(event: Omit<ConnectionEvent, "timestamp">): void {
    const fullEvent: ConnectionEvent = {
      ...event,
      timestamp: new Date().toISOString(),
    };

    // Add to buffer (most recent first)
    this.connectionEvents.unshift(fullEvent);

    // Trim to max size
    if (this.connectionEvents.length > CONNECTION_EVENT_BUFFER_SIZE) {
      this.connectionEvents = this.connectionEvents.slice(
        0,
        CONNECTION_EVENT_BUFFER_SIZE,
      );
    }

    // Notify listeners
    this.connectionEventListeners.forEach((listener) => {
      try {
        listener(fullEvent);
      } catch (error) {
        logger.error("Error in connection event listener:", error);
      }
    });
  }

  /**
   * Check if all required metadata context is available for streaming connections.
   * The org resolver interceptor requires auth token, x-organization-id, and x-user-id.
   */
  private hasRequiredContext(): boolean {
    const token = storageService.getItem(STORAGE_KEYS.AUTH_TOKEN);
    return !!(token && this.currentOrgId && this.currentUserId);
  }

  /**
   * Stop all reconnect timers across all stream types.
   * Used when context is no longer available (logout, auth failure).
   */
  private stopAllReconnectTimers(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    if (this.eventStreamReconnectTimeout) {
      clearTimeout(this.eventStreamReconnectTimeout);
      this.eventStreamReconnectTimeout = null;
    }
    if (this.baseStationReconnectTimeout) {
      clearTimeout(this.baseStationReconnectTimeout);
      this.baseStationReconnectTimeout = null;
    }
  }

  /**
   * Reset the realtime service state completely.
   * Disconnects all streams, clears timers and cached context.
   * Called on logout and auth failure.
   */
  reset(): void {
    this.disconnect();
    this.currentOrgId = null;
    this.currentUserId = null;
    this.reconnecting = false;
    this.reconnectAttempts = 0;
  }

  /**
   * Schedule reconnection with exponential backoff
   */
  private scheduleReconnect(): void {
    if (this.reconnectTimeout) {
      return; // Already scheduled
    }

    if (!this.hasRequiredContext()) {
      this.stopAllReconnectTimers();
      return;
    }

    this.setState("reconnecting");

    // Exponential backoff: base * 2^attempts, capped at max
    const delay = Math.min(
      TIMING_RECONNECT_BASE_DELAY * Math.pow(2, this.reconnectAttempts),
      TIMING_RECONNECT_MAX_DELAY,
    );

    this.reconnectAttempts++;

    // Emit reconnect event with timing info
    this.emitConnectionEvent({
      type: "realtime_reconnect",
      message: `${REALTIME_MESSAGES.RECONNECTING_IN} ${Math.round(delay / 1000)}s (attempt ${this.reconnectAttempts})`,
      attempt: this.reconnectAttempts,
      delayMs: delay,
    });

    this.reconnectTimeout = setTimeout(() => {
      this.reconnectTimeout = null;
      this.connect();
    }, delay);
  }

  // ============================================================================
  // BASE STATION STREAM METHODS
  // ============================================================================

  /**
   * Connect to base station-specific message stream
   * @param bsEui Base station EUI (required)
   */
  connectBaseStationStream(bsEui: string): void {
    if (!bsEui) {
      logger.error(REALTIME_MESSAGES.BS_EUI_REQUIRED);
      return;
    }

    if (!this.hasRequiredContext()) {
      this.stopAllReconnectTimers();
      logger.warn(REALTIME_MESSAGES.CONTEXT_INCOMPLETE);
      return;
    }

    // Close existing BS stream if any
    if (this.baseStationStream) {
      this.baseStationStream.close();
    }

    const request = new pb.StreamBaseStationMessagesRequest();
    request.setBsEui(bsEui);

    this.baseStationStream = grpc.invoke(
      KiloCenterService.StreamBaseStationMessages,
      {
        host: grpcUrl || "",
        request,
        metadata: this.buildMetadata(),
        onHeaders: () => {
          this.baseStationReconnectAttempts = 0; // Reset on successful connect
          this.emitConnectionEvent({
            type: "realtime_connected",
            message: REALTIME_MESSAGES.BS_STREAM_CONNECTED,
          });
        },
        onMessage: (msg: pb.BaseStationMessage) => {
          const event = this.mapBaseStationMessageToRealtimeEvent(
            msg.toObject(),
          );
          this.dispatch(event);
        },
        onEnd: (code: grpc.Code, message: string) => {
          this.baseStationStream = null;
          if (code !== grpc.Code.OK) {
            this.setError({
              message: message || REALTIME_MESSAGES.BS_STREAM_ERROR,
              timestamp: new Date(),
              code,
            });
            // Use SEPARATE reconnect state
            this.scheduleBaseStationReconnect(bsEui);
          }
        },
      },
    );
  }

  /**
   * Schedule reconnection for base station stream with SEPARATE backoff state
   */
  private scheduleBaseStationReconnect(bsEui: string): void {
    if (!this.hasRequiredContext()) {
      this.stopAllReconnectTimers();
      return;
    }

    // Clear any existing BS reconnect timeout
    if (this.baseStationReconnectTimeout) {
      clearTimeout(this.baseStationReconnectTimeout);
    }

    // Use SEPARATE reconnect attempts counter
    const delay = Math.min(
      TIMING_RECONNECT_BASE_DELAY *
        Math.pow(2, this.baseStationReconnectAttempts),
      TIMING_RECONNECT_MAX_DELAY,
    );
    this.baseStationReconnectAttempts++;

    this.baseStationReconnectTimeout = setTimeout(() => {
      this.baseStationReconnectTimeout = null;
      this.connectBaseStationStream(bsEui);
    }, delay);
  }

  /**
   * Disconnect base station stream only
   */
  disconnectBaseStationStream(): void {
    if (this.baseStationReconnectTimeout) {
      clearTimeout(this.baseStationReconnectTimeout);
      this.baseStationReconnectTimeout = null;
    }

    if (this.baseStationStream) {
      this.baseStationStream.close();
      this.baseStationStream = null;
    }

    this.baseStationReconnectAttempts = 0;
  }

  // ============================================================================
  // EVENT STREAM METHODS
  // ============================================================================

  /**
   * Connect to event stream for realtime system events
   * Streams endpoint attach/detach, basestation online/offline, etc.
   */
  connectEventStream(): void {
    if (!this.hasRequiredContext()) {
      this.stopAllReconnectTimers();
      logger.warn(REALTIME_MESSAGES.CONTEXT_INCOMPLETE);
      return;
    }

    // Close existing event stream if any
    if (this.eventStream) {
      this.eventStream.close();
    }

    const request = new pb.StreamEventsRequest();
    // No filters - stream all events for the tenant

    this.eventStream = grpc.invoke(KiloCenterService.StreamEvents, {
      host: grpcUrl || "",
      request,
      metadata: this.buildMetadata(),
      onHeaders: () => {
        this.eventStreamReconnectAttempts = 0; // Reset on successful connect
        this.emitConnectionEvent({
          type: "realtime_connected",
          message: REALTIME_MESSAGES.EVENT_STREAM_CONNECTED,
        });
      },
      onMessage: (msg: pb.Event) => {
        const event = this.mapEventToRealtimeEvent(msg.toObject());
        if (event) {
          this.dispatch(event);
        }
      },
      onEnd: (code: grpc.Code, message: string) => {
        this.eventStream = null;
        if (code !== grpc.Code.OK) {
          this.setError({
            message: message || REALTIME_MESSAGES.EVENT_STREAM_ERROR,
            timestamp: new Date(),
            code,
          });
        }
        // Schedule reconnect regardless of how stream ended (mirrors message stream)
        this.scheduleEventStreamReconnect();
      },
    });
  }

  /**
   * Schedule reconnection for event stream with separate backoff state
   */
  private scheduleEventStreamReconnect(): void {
    if (!this.hasRequiredContext()) {
      this.stopAllReconnectTimers();
      return;
    }

    if (this.eventStreamReconnectTimeout) {
      clearTimeout(this.eventStreamReconnectTimeout);
    }

    const delay = Math.min(
      TIMING_RECONNECT_BASE_DELAY *
        Math.pow(2, this.eventStreamReconnectAttempts),
      TIMING_RECONNECT_MAX_DELAY,
    );
    this.eventStreamReconnectAttempts++;

    this.eventStreamReconnectTimeout = setTimeout(() => {
      this.eventStreamReconnectTimeout = null;
      this.connectEventStream();
    }, delay);
  }

  /**
   * Disconnect event stream only
   */
  disconnectEventStream(): void {
    if (this.eventStreamReconnectTimeout) {
      clearTimeout(this.eventStreamReconnectTimeout);
      this.eventStreamReconnectTimeout = null;
    }

    if (this.eventStream) {
      this.eventStream.close();
      this.eventStream = null;
    }

    this.eventStreamReconnectAttempts = 0;
  }
}

/**
 * Singleton instance export
 */
export const realtimeService = RealtimeService.getInstance();
