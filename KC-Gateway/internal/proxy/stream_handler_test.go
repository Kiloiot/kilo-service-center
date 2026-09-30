package proxy

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	bufconnSize        = 1 << 20
	bufconnTarget      = "passthrough:///bufnet"
	streamWait         = 2 * time.Second
	openHeaderKey      = "x-stream-open"
	openHeaderValue    = "yes"
	closeTrailerKey    = "x-stream-close"
	closeTrailerValue  = "done"
	upstreamVersion    = "upstream-version"
	upstreamDeniedText = "upstream denied"
)

// streamingUpstream plays KC-Core: StreamEvents runs the test's script, GetReleaseInfo a fixed version.
type streamingUpstream struct {
	pb.UnimplementedCoreServiceServer
	events func(grpc.ServerStreamingServer[pb.Event]) error
}

func (u streamingUpstream) StreamEvents(_ *pb.StreamEventsRequest, stream grpc.ServerStreamingServer[pb.Event]) error {
	return u.events(stream)
}

func (streamingUpstream) GetReleaseInfo(context.Context, *emptypb.Empty) (*pb.ReleaseInfo, error) {
	return &pb.ReleaseInfo{Version: upstreamVersion}, nil
}

func serveBufconn(t *testing.T, register func(*grpc.Server), opts ...grpc.ServerOption) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(bufconnSize)
	server := grpc.NewServer(opts...)
	register(server)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient(bufconnTarget,
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// gatewayClient returns a CoreService client that reaches the upstream only through the handler.
func gatewayClient(t *testing.T, upstream streamingUpstream) pb.CoreServiceClient {
	t.Helper()
	upstreamConn := serveBufconn(t, func(s *grpc.Server) { pb.RegisterCoreServiceServer(s, upstream) })
	director := func(ctx context.Context, _ string) (context.Context, grpc.ClientConnInterface, error) {
		return metadata.NewOutgoingContext(ctx, SanitizeAndInject(ctx, "")), upstreamConn, nil
	}
	gatewayConn := serveBufconn(t, func(*grpc.Server) {}, grpc.UnknownServiceHandler(TransparentHandler(director)))
	return pb.NewCoreServiceClient(gatewayConn)
}

func sendOpenHeader(stream grpc.ServerStreamingServer[pb.Event]) error {
	return stream.SendHeader(metadata.Pairs(openHeaderKey, openHeaderValue))
}

func TestTransparentHandler_ForwardsTheHeaderBeforeAnyMessage(t *testing.T) {
	client := gatewayClient(t, streamingUpstream{events: func(stream grpc.ServerStreamingServer[pb.Event]) error {
		if err := sendOpenHeader(stream); err != nil {
			return err
		}
		<-stream.Context().Done()
		return nil
	}})
	ctx, cancel := context.WithTimeout(testutil.TestContext(), streamWait)
	defer cancel()

	stream, err := client.StreamEvents(ctx, &pb.StreamEventsRequest{})
	require.NoError(t, err)
	header, err := stream.Header()
	require.NoError(t, err)
	assert.Equal(t, []string{openHeaderValue}, header.Get(openHeaderKey),
		"a quiet stream must reach the client as open as soon as the upstream opens it")
}

func TestTransparentHandler_ForwardsMessagesAndTrailersInOrder(t *testing.T) {
	ids := []string{"first", "second", "third"}
	client := gatewayClient(t, streamingUpstream{events: func(stream grpc.ServerStreamingServer[pb.Event]) error {
		if err := sendOpenHeader(stream); err != nil {
			return err
		}
		for _, id := range ids {
			if err := stream.Send(&pb.Event{Id: id}); err != nil {
				return err
			}
		}
		stream.SetTrailer(metadata.Pairs(closeTrailerKey, closeTrailerValue))
		return nil
	}})

	stream, err := client.StreamEvents(testutil.TestContext(), &pb.StreamEventsRequest{})
	require.NoError(t, err)
	var received []string
	for {
		event, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		require.NoError(t, recvErr)
		received = append(received, event.GetId())
	}
	assert.Equal(t, ids, received)
	assert.Equal(t, []string{closeTrailerValue}, stream.Trailer().Get(closeTrailerKey))
}

func TestTransparentHandler_PassesAnUpstreamStatusThrough(t *testing.T) {
	for name, openFirst := range map[string]bool{"after the header": true, "trailers only": false} {
		t.Run(name, func(t *testing.T) {
			client := gatewayClient(t, streamingUpstream{events: func(stream grpc.ServerStreamingServer[pb.Event]) error {
				if openFirst {
					if err := sendOpenHeader(stream); err != nil {
						return err
					}
				}
				return status.Error(codes.PermissionDenied, upstreamDeniedText)
			}})

			stream, err := client.StreamEvents(testutil.TestContext(), &pb.StreamEventsRequest{})
			require.NoError(t, err)
			_, err = stream.Recv()
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.PermissionDenied, st.Code())
			assert.Equal(t, upstreamDeniedText, st.Message())
		})
	}
}

func TestTransparentHandler_ProxiesUnaryCalls(t *testing.T) {
	client := gatewayClient(t, streamingUpstream{})

	info, err := client.GetReleaseInfo(testutil.TestContext(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, upstreamVersion, info.GetVersion())
}

func TestTransparentHandler_ClientCancelCancelsTheUpstream(t *testing.T) {
	upstreamDone := make(chan struct{})
	client := gatewayClient(t, streamingUpstream{events: func(stream grpc.ServerStreamingServer[pb.Event]) error {
		if err := sendOpenHeader(stream); err != nil {
			return err
		}
		<-stream.Context().Done()
		close(upstreamDone)
		return stream.Context().Err()
	}})
	ctx, cancel := context.WithCancel(testutil.TestContext())

	stream, err := client.StreamEvents(ctx, &pb.StreamEventsRequest{})
	require.NoError(t, err)
	_, err = stream.Header()
	require.NoError(t, err)
	cancel()

	select {
	case <-upstreamDone:
	case <-time.After(streamWait):
		t.Fatal("the upstream stream must end when the client cancels")
	}
}
