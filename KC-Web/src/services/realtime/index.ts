/**
 * Realtime Service Barrel Export
 *
 * Realtime service for event streaming.
 * Uses gRPC streaming RPCs: StreamMessages, StreamBaseStationMessages from KC-Core gRPC service.
 * @see src/services/grpc/client.ts for gRPC implementation
 */

export { realtimeService } from "./RealtimeService";
export type {
  ConnectionError,
  ConnectionEvent,
  ConnectionEventListener,
  ConnectionEventType,
  ConnectionState,
  ErrorListener,
  EventHandler,
  RealtimeEvent,
  RealtimeEventType,
  RealtimeStreamKind,
  StateChangeListener,
} from "./types";
