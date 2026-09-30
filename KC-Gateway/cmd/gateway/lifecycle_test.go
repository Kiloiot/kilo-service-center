package main

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const testShutdownTimeout = 100 * time.Millisecond

// The gateway serves its gRPC server through ServeHTTP; shutting down with a
// stream still open must close it rather than panic.
func TestShutdownServers_WithAnOpenStream(t *testing.T) {
	proxyServer := grpc.NewServer()
	healthpb.RegisterHealthServer(proxyServer, health.NewServer())

	lis, err := net.Listen("tcp", testUpstreamListen)
	require.NoError(t, err)
	httpServer := &http.Server{
		Handler:           newMuxHandler(&config.Config{}, logger.NewNop(), proxyServer),
		ReadHeaderTimeout: testShutdownTimeout,
	}
	go func() { _ = httpServer.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	stream, err := healthpb.NewHealthClient(conn).Watch(testutil.TestContext(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	_, err = stream.Recv()
	require.NoError(t, err, "the watch stream is open on the server")

	shutdownServers(logger.NewNop(), httpServer, proxyServer, testShutdownTimeout)

	_, err = stream.Recv()
	require.Error(t, err, "shutdown closes the open stream")
}
