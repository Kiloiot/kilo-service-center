package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/rpccatalog"
)

const (
	testUpstreamListen = "127.0.0.1:0"
	testDialTimeout    = time.Second
	testMaxRetries     = 2
	testRetryBackoff   = 10 * time.Millisecond
)

// diagnosticsUpstream answers GetDiagnosticsBundle with an archive of the
// largest size KC-Core may produce.
type diagnosticsUpstream struct {
	pb.UnimplementedCoreServiceServer
}

func (diagnosticsUpstream) GetDiagnosticsBundle(context.Context, *pb.GetDiagnosticsBundleRequest) (*pb.GetDiagnosticsBundleResponse, error) {
	return &pb.GetDiagnosticsBundleResponse{Archive: make([]byte, config.DiagnosticsMaxBundleBytes)}, nil
}

const (
	testGatewayPeerSecret = "peer-secret-for-tests"
	testProxiedMethod     = "/kilocenter.api.v1.CoreService/ListEndPoints"
)

func TestDirector_PresentsThePeerSecretUpstream(t *testing.T) {
	lis, err := net.Listen("tcp", testUpstreamListen)
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })
	resCfg := config.GatewayResilienceConfig{DialTimeout: testDialTimeout, MaxRetries: testMaxRetries, RetryBackoff: testRetryBackoff, RetryMaxBackoff: testRetryBackoff}
	retryConfig, err := resilience.BuildRetryServiceConfig(resCfg, rpccatalog.Proxied()...)
	require.NoError(t, err)
	conn, err := dialProxied(lis.Addr().String(), resCfg, retryConfig)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	cfg := &config.Config{}
	cfg.InternalAuth.PeerSecret = testGatewayPeerSecret
	director := newDirector(cfg, logger.NewNop(), &upstreams{core: conn, identity: conn})

	outCtx, _, err := director(testutil.TestContext(), testProxiedMethod)
	require.NoError(t, err)
	md, _ := metadata.FromOutgoingContext(outCtx)
	assert.Equal(t, []string{testGatewayPeerSecret}, md.Get(grpcconst.MetadataKeyInternalPeerSecret),
		"every proxied call carries the gateway's peer secret")
}

func TestDialProxied_CarriesTheLargestDiagnosticsBundle(t *testing.T) {
	lis, err := net.Listen("tcp", testUpstreamListen)
	require.NoError(t, err)
	upstream := grpc.NewServer()
	pb.RegisterCoreServiceServer(upstream, diagnosticsUpstream{})
	go func() { _ = upstream.Serve(lis) }()
	t.Cleanup(upstream.Stop)

	resCfg := config.GatewayResilienceConfig{DialTimeout: testDialTimeout, MaxRetries: testMaxRetries, RetryBackoff: testRetryBackoff, RetryMaxBackoff: testRetryBackoff}
	retryConfig, err := resilience.BuildRetryServiceConfig(resCfg, rpccatalog.Proxied()...)
	require.NoError(t, err)
	conn, err := dialProxied(lis.Addr().String(), resCfg, retryConfig)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	resp, err := pb.NewCoreServiceClient(conn).GetDiagnosticsBundle(testutil.TestContext(), &pb.GetDiagnosticsBundleRequest{})
	require.NoError(t, err, "the gateway must accept every bundle KC-Core is allowed to build")
	assert.Len(t, resp.Archive, config.DiagnosticsMaxBundleBytes)
}
