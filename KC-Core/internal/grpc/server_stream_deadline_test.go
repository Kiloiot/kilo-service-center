package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testStreamWriteTimeout = 200 * time.Millisecond
	testStreamOutlives     = 3 * testStreamWriteTimeout
	// withHealth serves the health service, whose Watch is the stream under test.
	withHealth = true
)

// A server-streaming RPC outlives the HTTP server's write timeout: the
// timeout bounds unary calls, never a stream the web UI keeps open.
func TestServer_StreamOutlivesHTTPWriteTimeout(t *testing.T) {
	srv, err := NewServer(Config{
		Log:          logger.NewNop(),
		Host:         testServerLoopback,
		Port:         ephemeralPort,
		EnableHealth: withHealth,
		HTTPConfig:   HTTPServerConfig{ReadTimeout: testStreamWriteTimeout, WriteTimeout: testStreamWriteTimeout},
		RoleSource:   fixedRoles(authz.Roles{}),
	})
	require.NoError(t, err)
	go func() { _ = srv.Start() }()
	t.Cleanup(srv.Stop)

	conn, err := grpclib.NewClient(srv.listener.Addr().String(), grpclib.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	service := pb.CoreService_ServiceDesc.ServiceName
	watch, err := healthpb.NewHealthClient(conn).Watch(testutil.TestContext(), &healthpb.HealthCheckRequest{Service: service})
	require.NoError(t, err)
	first, err := watch.Recv()
	require.NoError(t, err)
	require.Equal(t, healthpb.HealthCheckResponse_SERVING, first.GetStatus())

	time.Sleep(testStreamOutlives)
	srv.healthServer.SetServingStatus(service, healthpb.HealthCheckResponse_NOT_SERVING)

	next, err := watch.Recv()
	require.NoError(t, err, "the stream is still open after the write timeout")
	require.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, next.GetStatus())
}
