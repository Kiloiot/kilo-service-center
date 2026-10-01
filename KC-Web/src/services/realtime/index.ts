/**
 * Realtime service barrel: the gRPC streams (StreamEvents, StreamMessages,
 * StreamBaseStationMessages) one tab holds for every tab, behind three narrow
 * views, and the cached queries each streamed event makes stale.
 */

export { catchUpKeys, invalidatedKeys } from "./invalidation";
export {
  realtimeLifecycle,
  realtimeStreams,
  realtimeSubscriptions,
} from "./RealtimeService";
export type {
  ConnectionError,
  ConnectionState,
  RealtimeEvent,
  RealtimeStreamKind,
} from "./types";
