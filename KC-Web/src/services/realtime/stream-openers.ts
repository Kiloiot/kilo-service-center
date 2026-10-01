/**
 * Opens the gRPC-web server stream of a spec through the transport, which
 * renews an expiring token first, and maps each streamed message to the
 * realtime event every tab consumes.
 */

import { grpc } from "@improbable-eng/grpc-web";

import * as pb from "@services/grpc/core_pb";
import * as CoreServiceModule from "@services/grpc/core_pb_service";
import { grpcTransport } from "@services/grpc/transport";
import { REALTIME_STREAM_KIND } from "@constants/app";
import { REALTIME_MESSAGES } from "@constants/messages";
import { grpcUrl } from "@config/env";

import type { ChannelWording, StreamCallbacks } from "./stream-channel";
import type { StreamSpec } from "./stream-demand";
import { systemEvent, uplinkEvent } from "./stream-events";
import type { RealtimeEvent, RealtimeStreamKind } from "./types";

const { CoreService } = CoreServiceModule;

/** Receives the events of one stream. */
export type StreamSink = (event: RealtimeEvent) => void;

/** Starts the stream of a spec and hands its events to the sink. */
export type SpecOpener = (
  spec: StreamSpec,
  callbacks: StreamCallbacks,
  sink: StreamSink,
) => grpc.Request;

/** Wording of each stream kind's connection events. */
export const STREAM_WORDING: Record<RealtimeStreamKind, ChannelWording> = {
  [REALTIME_STREAM_KIND.EVENT]: {
    connected: REALTIME_MESSAGES.EVENT_STREAM_CONNECTED,
    ended: REALTIME_MESSAGES.EVENT_STREAM_ERROR,
  },
  [REALTIME_STREAM_KIND.MESSAGE]: {
    connected: REALTIME_MESSAGES.CONNECTED,
    ended: REALTIME_MESSAGES.STREAM_ENDED_ERROR,
  },
  [REALTIME_STREAM_KIND.BASE_STATION]: {
    connected: REALTIME_MESSAGES.BS_STREAM_CONNECTED,
    ended: REALTIME_MESSAGES.BS_STREAM_ERROR,
  },
};

function openStream<
  TRequest extends grpc.ProtobufMessage,
  TResponse extends grpc.ProtobufMessage,
  M extends grpc.MethodDefinition<TRequest, TResponse>,
>(
  method: M,
  spec: StreamSpec,
  callbacks: StreamCallbacks,
  call: { request: TRequest; onMessage: (message: TResponse) => void },
): grpc.Request {
  return grpcTransport.openStream(
    method,
    {
      ...call,
      host: grpcUrl,
      onHeaders: callbacks.onHeaders,
      onEnd: callbacks.onEnd,
    },
    {
      organizationId: spec.context.organizationId,
      userId: spec.context.userId,
    },
  );
}

function stationRequest(bsEui: string): pb.StreamBaseStationMessagesRequest {
  const request = new pb.StreamBaseStationMessagesRequest();
  request.setBsEui(bsEui);
  return request;
}

export const grpcSpecOpener: SpecOpener = (spec, callbacks, sink) => {
  const { organizationId } = spec.context;
  switch (spec.kind) {
    case REALTIME_STREAM_KIND.EVENT:
      return openStream(CoreService.StreamEvents, spec, callbacks, {
        request: new pb.StreamEventsRequest(),
        onMessage: (msg: pb.Event) =>
          sink(systemEvent(msg.toObject(), organizationId)),
      });
    case REALTIME_STREAM_KIND.MESSAGE:
      return openStream(CoreService.StreamMessages, spec, callbacks, {
        request: new pb.StreamMessagesRequest(),
        onMessage: (msg: pb.Message) =>
          sink(uplinkEvent(msg.toObject(), organizationId)),
      });
    case REALTIME_STREAM_KIND.BASE_STATION:
      return openStream(
        CoreService.StreamBaseStationMessages,
        spec,
        callbacks,
        {
          request: stationRequest(spec.bsEui),
          onMessage: (msg: pb.BaseStationMessage) =>
            sink(uplinkEvent(msg.toObject(), organizationId)),
        },
      );
  }
};
