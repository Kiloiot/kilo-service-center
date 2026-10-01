/**
 * The messages the leader tab and the other tabs exchange, and the channel
 * that carries them. Every message is plain data, so it survives the
 * structured clone a BroadcastChannel applies.
 */

import { REALTIME_RELAY_MESSAGE } from "@constants/app";

import type { TabDemand } from "./stream-demand";
import type {
  ConnectionError,
  ConnectionEvent,
  ConnectionState,
  RealtimeEvent,
} from "./types";

/** A tab's demand; null withdraws it. */
export interface DemandMessage {
  type: typeof REALTIME_RELAY_MESSAGE.DEMAND;
  tabId: string;
  demand: TabDemand | null;
}

/** A new leader asks every tab for its demand; the previous leader is gone. */
export interface CensusMessage {
  type: typeof REALTIME_RELAY_MESSAGE.CENSUS;
}

export interface EventMessage {
  type: typeof REALTIME_RELAY_MESSAGE.EVENT;
  streamKey: string;
  event: RealtimeEvent;
}

export interface ConnectionMessage {
  type: typeof REALTIME_RELAY_MESSAGE.CONNECTION;
  streamKey: string;
  event: ConnectionEvent;
}

export interface ErrorMessage {
  type: typeof REALTIME_RELAY_MESSAGE.ERROR;
  streamKey: string;
  error: ConnectionError;
}

/** The state of every stream the leader holds, by stream key. */
export interface StatesMessage {
  type: typeof REALTIME_RELAY_MESSAGE.STATES;
  states: Record<string, ConnectionState>;
}

/** What the leader relays about its streams to every tab, itself included. */
export type StreamMessage =
  | EventMessage
  | ConnectionMessage
  | ErrorMessage
  | StatesMessage;

export type RelayMessage = DemandMessage | CensusMessage | StreamMessage;

/** The channel every tab of the origin shares; a tab does not receive its own messages. */
export interface RelayPort {
  post(message: RelayMessage): void;
  listen(handler: (message: RelayMessage) => void): void;
}

export function broadcastRelayPort(name: string): RelayPort {
  const channel = new BroadcastChannel(name);
  return {
    post: (message) => channel.postMessage(message),
    listen: (handler) => {
      channel.onmessage = (message: MessageEvent<RelayMessage>) =>
        handler(message.data);
    },
  };
}
