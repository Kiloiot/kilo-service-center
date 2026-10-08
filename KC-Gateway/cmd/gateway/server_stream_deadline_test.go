package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/rpccatalog"
)

const (
	testStreamWriteTimeout = 200 * time.Millisecond
	testStreamOutlives     = 3 * testStreamWriteTimeout
	testStreamedEvents     = 2
	testGRPCWebContentType = "application/grpc-web+proto"
	testGRPCWebFrameHeader = 5
	testGRPCWebDataFrame   = 0
	testStreamEventsMethod = "StreamEvents"
	// grpcWebOn enables the gRPC-web wrapper and its CORS, which the browser path needs.
	grpcWebOn = true
)

// slowEvents streams one event, then a second after the write timeout passed.
type slowEvents struct {
	pb.UnimplementedCoreServiceServer
}

func (slowEvents) StreamEvents(_ *pb.StreamEventsRequest, stream grpc.ServerStreamingServer[pb.Event]) error {
	if err := stream.Send(&pb.Event{}); err != nil {
		return err
	}
	time.Sleep(testStreamOutlives)
	return stream.Send(&pb.Event{})
}

// A proxied stream outlives the gateway's HTTP write timeout: the timeout
// bounds unary calls, never the event stream the web UI keeps open.
func TestMuxHandler_StreamOutlivesHTTPWriteTimeout(t *testing.T) {
	proxied := grpc.NewServer()
	pb.RegisterCoreServiceServer(proxied, slowEvents{})
	lis, err := net.Listen("tcp", testUpstreamListen)
	require.NoError(t, err)
	server := &http.Server{
		Handler:      newMuxHandler(&config.Config{}, logger.NewNop(), proxied),
		ReadTimeout:  testStreamWriteTimeout,
		WriteTimeout: testStreamWriteTimeout,
	}
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(func() { _ = server.Close() })

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	stream, err := pb.NewCoreServiceClient(conn).StreamEvents(testutil.TestContext(), &pb.StreamEventsRequest{})
	require.NoError(t, err)
	_, err = stream.Recv()
	require.NoError(t, err)
	_, err = stream.Recv()
	require.NoError(t, err, "the stream is still open after the write timeout")
}

// The browser reaches the gateway over HTTP/1.1 gRPC-web through the dev
// proxy; its event stream outlives the write timeout too.
func TestMuxHandler_GRPCWebStreamOutlivesHTTPWriteTimeout(t *testing.T) {
	proxied := grpc.NewServer()
	pb.RegisterCoreServiceServer(proxied, slowEvents{})
	cfg := &config.Config{}
	cfg.GRPC.Web.Enabled = grpcWebOn
	cfg.GRPC.Web.AllowAllOrigins = grpcWebOn
	server := httptest.NewUnstartedServer(newMuxHandler(cfg, logger.NewNop(), proxied))
	server.Config.WriteTimeout = testStreamWriteTimeout
	server.Start()
	t.Cleanup(server.Close)

	url := server.URL + rpccatalog.FullMethod(pb.CoreService_ServiceDesc, testStreamEventsMethod)
	emptyRequestFrame := bytes.NewReader(make([]byte, testGRPCWebFrameHeader))
	req, err := http.NewRequestWithContext(testutil.TestContext(), http.MethodPost, url, emptyRequestFrame)
	require.NoError(t, err)
	req.Header.Set(grpcconst.HeaderContentType, testGRPCWebContentType)
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	for range testStreamedEvents {
		header := make([]byte, testGRPCWebFrameHeader)
		_, err = io.ReadFull(resp.Body, header)
		require.NoError(t, err, "the stream is still open after the write timeout")
		require.Equal(t, byte(testGRPCWebDataFrame), header[0])
		_, err = io.ReadFull(resp.Body, make([]byte, binary.BigEndian.Uint32(header[1:])))
		require.NoError(t, err)
	}
}
