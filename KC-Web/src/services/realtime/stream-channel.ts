/**
 * One gRPC-web server stream that opens, reports its state and reopens with
 * exponential backoff on its own, whatever the other streams do.
 */

import { grpc } from "@improbable-eng/grpc-web";

import { exponentialBackoffMs } from "@utils/backoff";
import { logger } from "@utils/logger";
import {
  CONNECTION_STATE,
  MS_PER_SECOND,
  REALTIME_LIFECYCLE_EVENT,
  TIMING_RECONNECT_BASE_DELAY,
  TIMING_RECONNECT_MAX_DELAY,
} from "@constants/app";
import { DURATION_UNIT_SUFFIX, REALTIME_MESSAGES } from "@constants/messages";
import { grpcUrl } from "@config/env";

import type {
  ConnectionError,
  ConnectionEvent,
  ConnectionState,
  RealtimeStreamKind,
} from "./types";

/** The callbacks a channel hands the gRPC call it opens. */
export interface StreamCallbacks {
  onHeaders(): void;
  onEnd(code: grpc.Code, message: string): void;
}

/** Starts the channel's gRPC call; the caller maps and dispatches its messages. */
export type StreamOpener = (callbacks: StreamCallbacks) => grpc.Request;

/** What a channel reports to the service that owns it. */
export interface ChannelHost {
  hasContext(): boolean;
  emit(event: Omit<ConnectionEvent, "timestamp">): void;
  reportError(error: ConnectionError): void;
  stateChanged(): void;
}

/** Wording of one stream's connection events. */
export interface ChannelWording {
  connected: string;
  ended: string;
}

export class StreamChannel {
  private state: ConnectionState = CONNECTION_STATE.DISCONNECTED;
  private request: grpc.Request | null = null;
  private opener: StreamOpener | null = null;
  private attempts = 0;
  private retry: ReturnType<typeof setTimeout> | null = null;
  private readonly kind: RealtimeStreamKind;
  private readonly wording: ChannelWording;
  private readonly host: ChannelHost;

  constructor(
    kind: RealtimeStreamKind,
    wording: ChannelWording,
    host: ChannelHost,
  ) {
    this.kind = kind;
    this.wording = wording;
    this.host = host;
  }

  getState(): ConnectionState {
    return this.state;
  }

  /** Opens the stream with opener, replacing any stream the channel holds. */
  open(opener: StreamOpener): void {
    this.close();
    this.opener = opener;
    this.start();
  }

  /** Closes the stream and forgets it; nothing reopens it. */
  close(): void {
    this.clearRetry();
    this.opener = null;
    this.request?.close();
    this.request = null;
    this.attempts = 0;
    this.setState(CONNECTION_STATE.DISCONNECTED);
  }

  private start(): void {
    const opener = this.opener;
    if (!opener || !this.host.hasContext()) {
      this.close();
      logger.warn(REALTIME_MESSAGES.CONTEXT_INCOMPLETE);
      return;
    }
    this.setState(
      this.attempts === 0
        ? CONNECTION_STATE.CONNECTING
        : CONNECTION_STATE.RECONNECTING,
    );
    this.host.emit({
      type: REALTIME_LIFECYCLE_EVENT.CONNECT,
      message: `${REALTIME_MESSAGES.CONNECTING_AT} ${grpcUrl}`,
      url: grpcUrl,
      streamKind: this.kind,
    });
    try {
      this.request = opener({
        onHeaders: () => this.connected(),
        onEnd: (code, message) => this.ended(code, message),
      });
    } catch (error) {
      logger.error(REALTIME_MESSAGES.FAILED_ESTABLISH, error);
      this.failed(
        error instanceof Error
          ? error.message
          : REALTIME_MESSAGES.FAILED_ESTABLISH,
      );
    }
  }

  /** Repeats the connected notice of a connected stream, for listeners that joined after it. */
  announceConnected(): void {
    if (this.state !== CONNECTION_STATE.CONNECTED) return;
    this.host.emit({
      type: REALTIME_LIFECYCLE_EVENT.CONNECTED,
      message: this.wording.connected,
      streamKind: this.kind,
    });
  }

  private connected(): void {
    this.attempts = 0;
    this.setState(CONNECTION_STATE.CONNECTED);
    this.announceConnected();
  }

  private ended(code: grpc.Code, message: string): void {
    this.request = null;
    if (code === grpc.Code.OK) {
      this.host.emit({
        type: REALTIME_LIFECYCLE_EVENT.DISCONNECTED,
        message: REALTIME_MESSAGES.STREAM_ENDED_NORMAL,
        streamKind: this.kind,
      });
    } else {
      const reason = message || `${this.wording.ended} ${code}`;
      this.host.reportError({ message: reason, timestamp: new Date(), code });
      this.host.emit({
        type: REALTIME_LIFECYCLE_EVENT.DISCONNECTED,
        message: reason,
        code,
        streamKind: this.kind,
      });
    }
    this.scheduleRetry();
  }

  private failed(reason: string): void {
    this.host.reportError({ message: reason, timestamp: new Date() });
    this.host.emit({
      type: REALTIME_LIFECYCLE_EVENT.ERROR,
      message: reason,
      streamKind: this.kind,
    });
    this.scheduleRetry();
  }

  private scheduleRetry(): void {
    if (this.retry || !this.opener) return;
    if (!this.host.hasContext()) {
      this.close();
      return;
    }
    const delay = exponentialBackoffMs(
      this.attempts,
      TIMING_RECONNECT_BASE_DELAY,
      TIMING_RECONNECT_MAX_DELAY,
    );
    this.attempts++;
    this.setState(CONNECTION_STATE.RECONNECTING);
    this.host.emit({
      type: REALTIME_LIFECYCLE_EVENT.RECONNECT,
      message: `${REALTIME_MESSAGES.RECONNECTING_IN} ${Math.round(delay / MS_PER_SECOND)}${DURATION_UNIT_SUFFIX.SECOND} (${REALTIME_MESSAGES.RECONNECT_ATTEMPT} ${this.attempts})`,
      attempt: this.attempts,
      delayMs: delay,
      streamKind: this.kind,
    });
    this.retry = setTimeout(() => {
      this.retry = null;
      this.start();
    }, delay);
  }

  private clearRetry(): void {
    if (this.retry) clearTimeout(this.retry);
    this.retry = null;
  }

  private setState(state: ConnectionState): void {
    if (this.state === state) return;
    this.state = state;
    this.host.stateChanged();
  }
}

/** The state of the streams a user holds: connected only when all of them are. */
export function combinedState(states: ConnectionState[]): ConnectionState {
  if (states.length === 0) return CONNECTION_STATE.DISCONNECTED;
  if (states.every((state) => state === CONNECTION_STATE.CONNECTED))
    return CONNECTION_STATE.CONNECTED;
  if (states.includes(CONNECTION_STATE.RECONNECTING))
    return CONNECTION_STATE.RECONNECTING;
  if (states.includes(CONNECTION_STATE.CONNECTING))
    return CONNECTION_STATE.CONNECTING;
  return CONNECTION_STATE.DISCONNECTED;
}
