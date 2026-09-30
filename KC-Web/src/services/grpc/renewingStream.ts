/**
 * A server stream that carries an access token the server accepts: an
 * expiring token is renewed before the stream is sent, and a stream the
 * server refuses as unauthenticated is renewed once and sent again, as a
 * unary call is. A renewal the server refuses ends the stream with the
 * refusal; one the service cannot answer ends it as unavailable, so the
 * owner reconnects with its backoff. Only headers that open the stream are
 * reported.
 */

import { grpc } from "@improbable-eng/grpc-web";

import { HEADERS, TOKEN_REFRESH_OUTCOME } from "@constants/app";
import { ERR_UNAUTHORIZED, GRPC_CLIENT_ERRORS } from "@constants/messages";

import { toGrpcApiError } from "./errors";
import type { RefreshOutcome } from "./refreshCoordinator";

/** The single-flight token renewal every call and stream shares. */
export interface TokenRenewal {
  isAccessTokenExpiring(): boolean;
  refresh(): Promise<RefreshOutcome>;
}

/** What the stream reports to the one who opened it. */
export interface StreamHandlers {
  onHeaders?: (headers: grpc.Metadata) => void;
  onEnd: (code: grpc.Code, message: string, trailers: grpc.Metadata) => void;
}

/** Sends the stream once, with the token stored at the time of the send. */
export type StreamSender = (handlers: Required<StreamHandlers>) => grpc.Request;

export function openRenewingStream(
  renewal: TokenRenewal,
  send: StreamSender,
  handlers: StreamHandlers,
): grpc.Request {
  const stream = new RenewingStream(renewal, send, handlers);
  stream.start();
  return stream;
}

class RenewingStream implements grpc.Request {
  private request: grpc.Request | null = null;
  private closed = false;
  private renewedAfterRefusal = false;
  private readonly renewal: TokenRenewal;
  private readonly send: StreamSender;
  private readonly handlers: StreamHandlers;

  constructor(
    renewal: TokenRenewal,
    send: StreamSender,
    handlers: StreamHandlers,
  ) {
    this.renewal = renewal;
    this.send = send;
    this.handlers = handlers;
  }

  start(): void {
    if (!this.renewal.isAccessTokenExpiring()) {
      this.sendNow();
      return;
    }
    void this.renewThenSend(
      grpc.Code.Unauthenticated,
      ERR_UNAUTHORIZED,
      new grpc.Metadata(),
    );
  }

  close(): void {
    this.closed = true;
    this.request?.close();
    this.request = null;
  }

  private sendNow(): void {
    this.request = this.send({
      onHeaders: (headers) => this.headers(headers),
      onEnd: (code, message, trailers) => this.ended(code, message, trailers),
    });
  }

  private headers(headers: grpc.Metadata): void {
    // A grpc-status header marks a response that ended the call before any message.
    if (headers.has(HEADERS.GRPC_STATUS)) return;
    this.handlers.onHeaders?.(headers);
  }

  private ended(
    code: grpc.Code,
    message: string,
    trailers: grpc.Metadata,
  ): void {
    this.request = null;
    const refused = toGrpcApiError({ code, message, metadata: trailers });
    if (this.renewedAfterRefusal || !refused.isUnauthorized()) {
      this.handlers.onEnd(code, message, trailers);
      return;
    }
    this.renewedAfterRefusal = true;
    void this.renewThenSend(code, message, trailers);
  }

  /** Sends with a renewed token, or ends with the refusal or as unavailable. */
  private async renewThenSend(
    code: grpc.Code,
    message: string,
    trailers: grpc.Metadata,
  ): Promise<void> {
    const outcome = await this.renewal.refresh();
    if (this.closed) return;
    switch (outcome) {
      case TOKEN_REFRESH_OUTCOME.RENEWED:
        this.sendNow();
        return;
      case TOKEN_REFRESH_OUTCOME.REFUSED:
        this.handlers.onEnd(code, message, trailers);
        return;
      case TOKEN_REFRESH_OUTCOME.UNAVAILABLE:
        this.handlers.onEnd(
          grpc.Code.Unavailable,
          GRPC_CLIENT_ERRORS.TOKEN_REFRESH_UNAVAILABLE,
          trailers,
        );
    }
  }
}
