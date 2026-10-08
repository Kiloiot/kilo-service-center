// Adapted from github.com/mwitkow/grpc-proxy proxy/handler.go,
// Copyright 2017 Michal Witkowski, licensed under the Apache License 2.0.

package proxy

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
)

// StreamDirector picks the outgoing context and upstream connection for a proxied call.
type StreamDirector func(ctx context.Context, fullMethod string) (context.Context, grpc.ClientConnInterface, error)

// proxiedStreamDesc opens every upstream call bidirectionally, which carries unary and streaming RPCs alike.
var proxiedStreamDesc = &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}

// TransparentHandler proxies unregistered services and forwards the upstream header as soon as it arrives.
func TransparentHandler(director StreamDirector) grpc.StreamHandler {
	return func(_ any, downstream grpc.ServerStream) error {
		return proxyCall(director, downstream)
	}
}

func proxyCall(director StreamDirector, downstream grpc.ServerStream) error {
	fullMethod, ok := grpc.MethodFromServerStream(downstream)
	if !ok {
		return grpcconst.ToStatusError(grpcconst.NewTokenError(grpcconst.ErrTokenInternalError, nil))
	}
	outCtx, conn, err := director(downstream.Context(), fullMethod)
	if err != nil {
		return err
	}
	upstreamCtx, cancelUpstream := context.WithCancel(outCtx)
	defer cancelUpstream()
	upstream, err := conn.NewStream(upstreamCtx, proxiedStreamDesc, fullMethod)
	if err != nil {
		return err
	}
	return pump(downstream, upstream, cancelUpstream)
}

// pump runs both directions until the upstream ends; a failed client side cancels the upstream.
func pump(downstream grpc.ServerStream, upstream grpc.ClientStream, cancelUpstream context.CancelFunc) error {
	requests := goForward(func() error { return relayMessages(downstream.RecvMsg, upstream.SendMsg) })
	responses := goForward(func() error { return relayResponses(upstream, downstream) })
	for {
		select {
		case err := <-requests:
			if !errors.Is(err, io.EOF) {
				cancelUpstream()
				return grpcconst.ToStatusError(err)
			}
			if err := upstream.CloseSend(); err != nil {
				cancelUpstream()
				return grpcconst.ToStatusError(err)
			}
			requests = nil
		case err := <-responses:
			downstream.SetTrailer(upstream.Trailer())
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func goForward(forward func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- forward() }()
	return done
}

func relayResponses(upstream grpc.ClientStream, downstream grpc.ServerStream) error {
	if err := forwardHeader(upstream, downstream); err != nil {
		return err
	}
	return relayMessages(upstream.RecvMsg, downstream.SendMsg)
}

// forwardHeader waits for the upstream header; a trailers-only upstream has none and RecvMsg surfaces its status.
func forwardHeader(upstream grpc.ClientStream, downstream grpc.ServerStream) error {
	header, err := upstream.Header()
	if err == nil && header != nil {
		return downstream.SendHeader(header)
	}
	return nil
}

// relayMessages copies frames verbatim: an Empty keeps unknown fields as bytes and writes them back.
func relayMessages(recv, send func(any) error) error {
	frame := &emptypb.Empty{}
	for {
		if err := recv(frame); err != nil {
			return err
		}
		if err := send(frame); err != nil {
			return err
		}
	}
}
