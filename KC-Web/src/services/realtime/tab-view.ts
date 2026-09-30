/**
 * What one tab sees of the leader's streams: the events, connection notices
 * and errors of the streams its demand names, and its connection state, the
 * combined state of the streams it holds.
 */

import { logger } from "@utils/logger";
import {
  CONNECTION_STATE,
  REALTIME_LIFECYCLE_EVENT,
  REALTIME_RELAY_MESSAGE,
} from "@constants/app";
import { REALTIME_MESSAGES } from "@constants/messages";

import type { StreamMessage } from "./relay";
import { combinedState } from "./stream-channel";
import {
  demandSpecs,
  heldSpecs,
  streamKey,
  type StreamSpec,
  type TabDemand,
} from "./stream-demand";
import type {
  ConnectionError,
  ConnectionEvent,
  ConnectionEventListener,
  ConnectionState,
  ErrorListener,
  EventHandler,
  RealtimeEvent,
  StateChangeListener,
} from "./types";

export class TabView {
  private consumed = new Map<string, StreamSpec>();
  private held: string[] = [];
  private states: Record<string, ConnectionState> = {};
  private state: ConnectionState = CONNECTION_STATE.DISCONNECTED;
  private lastError: ConnectionError | null = null;
  private readonly handlers = new Set<EventHandler>();
  private readonly stateListeners = new Set<StateChangeListener>();
  private readonly errorListeners = new Set<ErrorListener>();
  private readonly connectionListeners = new Set<ConnectionEventListener>();

  setDemand(demand: TabDemand | null): void {
    const consumed = demand ? demandSpecs(demand) : [];
    this.consumed = new Map(consumed.map((spec) => [streamKey(spec), spec]));
    this.held = demand ? heldSpecs(demand).map(streamKey) : [];
    this.refreshState();
  }

  receive(message: StreamMessage): void {
    if (message.type === REALTIME_RELAY_MESSAGE.STATES) {
      this.states = message.states;
      this.refreshState();
      return;
    }
    if (!this.consumed.has(message.streamKey)) return;
    switch (message.type) {
      case REALTIME_RELAY_MESSAGE.EVENT:
        return this.dispatch(message.event);
      case REALTIME_RELAY_MESSAGE.CONNECTION:
        return this.notify(message.event);
      case REALTIME_RELAY_MESSAGE.ERROR:
        return this.setError(message.error);
    }
  }

  /** The leader closed: its connected streams dropped until the next leader reopens them. */
  leaderLost(): void {
    this.consumed.forEach((spec, key) => {
      if (this.states[key] !== CONNECTION_STATE.CONNECTED) return;
      this.notify({
        type: REALTIME_LIFECYCLE_EVENT.DISCONNECTED,
        message: REALTIME_MESSAGES.LEADER_TAB_CLOSED,
        streamKind: spec.kind,
        timestamp: new Date().toISOString(),
      });
    });
    this.states = {};
    this.refreshState();
  }

  getState(): ConnectionState {
    return this.state;
  }

  getLastError(): ConnectionError | null {
    return this.lastError;
  }

  subscribeAll(handler: EventHandler): () => void {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  onStateChange(listener: StateChangeListener): () => void {
    this.stateListeners.add(listener);
    listener(this.state);
    return () => this.stateListeners.delete(listener);
  }

  onErrorChange(listener: ErrorListener): () => void {
    this.errorListeners.add(listener);
    listener(this.lastError);
    return () => this.errorListeners.delete(listener);
  }

  onConnectionEvent(listener: ConnectionEventListener): () => void {
    this.connectionListeners.add(listener);
    return () => this.connectionListeners.delete(listener);
  }

  private dispatch(event: RealtimeEvent): void {
    this.handlers.forEach((handler) => {
      try {
        handler(event);
      } catch (error) {
        logger.error(REALTIME_MESSAGES.HANDLER_FAILED, error);
      }
    });
  }

  private notify(event: ConnectionEvent): void {
    this.connectionListeners.forEach((listener) => {
      try {
        listener(event);
      } catch (error) {
        logger.error(REALTIME_MESSAGES.LISTENER_FAILED, error);
      }
    });
  }

  private refreshState(): void {
    const next = combinedState(
      this.held.map((key) => this.states[key] ?? CONNECTION_STATE.DISCONNECTED),
    );
    if (next === CONNECTION_STATE.CONNECTED) this.setError(null);
    if (next === this.state) return;
    this.state = next;
    this.stateListeners.forEach((listener) => listener(next));
  }

  private setError(error: ConnectionError | null): void {
    if (error === null && this.lastError === null) return;
    this.lastError = error;
    this.errorListeners.forEach((listener) => listener(error));
  }
}
