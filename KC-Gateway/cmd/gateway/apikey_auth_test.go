package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/config"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/adapter"
	"github.com/Kiloiot/kilo-service-center/KC-Gateway/internal/resilience"
)

const (
	testKeyActive   = "active-community-key"
	testKeyExpired  = "expired-community-key"
	testKeyRevoked  = "revoked-community-key"
	testKeyDeleted  = "deleted-community-key"
	testKeyTenantID = int64(1)
	testHMACSecret  = "gateway-api-key-test-secret-of-32-bytes"
	testCallTimeout = 5 * time.Second
	testAPIKeyBreak = 5
	testAuthEnabled = true
	testKeyInactive = false
)

var (
	testKeyOrgID  = uuid.MustParse("4c310e39-b828-4fa4-aeba-26e9a40471de")
	testKeyUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

func keyHash(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

// keyedIdentity answers ValidateAPIKey the way KC-Identity does: a known key
// with its tenant, organization, owner and state; NotFound for any other.
type keyedIdentity struct {
	pb.UnimplementedIdentityInternalServiceServer
	keys map[string]*pb.ValidateAPIKeyResponse
}

func (k keyedIdentity) ValidateAPIKey(_ context.Context, req *pb.ValidateAPIKeyRequest) (*pb.ValidateAPIKeyResponse, error) {
	if resp, ok := k.keys[req.GetKeyHash()]; ok {
		return resp, nil
	}
	return nil, status.Error(codes.NotFound, grpcconst.ResolveErrorMessage(grpcconst.ErrTokenApiKeyNotFound))
}

func (keyedIdentity) UpdateAPIKeyLastUsed(context.Context, *pb.UpdateAPIKeyLastUsedRequest) (*pb.UpdateAPIKeyLastUsedResponse, error) {
	return &pb.UpdateAPIKeyLastUsedResponse{}, nil
}

func (keyedIdentity) RecordPlatformEvent(context.Context, *pb.RecordPlatformEventRequest) (*pb.RecordPlatformEventResponse, error) {
	return &pb.RecordPlatformEventResponse{}, nil
}

// statusCore answers GetSystemStatus and keeps the identity the gateway forwarded.
type statusCore struct {
	pb.UnimplementedCoreServiceServer
	mu        sync.Mutex
	forwarded metadata.MD
}

func (c *statusCore) GetSystemStatus(ctx context.Context, _ *emptypb.Empty) (*pb.SystemStatus, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.forwarded = md
	return &pb.SystemStatus{}, nil
}

func serveOn(t *testing.T, register func(*grpc.Server)) string {
	t.Helper()
	lis, err := net.Listen("tcp", testUpstreamListen)
	require.NoError(t, err)
	server := grpc.NewServer()
	register(server)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)
	return lis.Addr().String()
}

// gatewayFor starts the gateway's proxy with its real auth interceptor and
// API key adapter in front of the given KC-Identity and KC-Core.
func gatewayFor(t *testing.T, identityAddr, coreAddr string) pb.CoreServiceClient {
	t.Helper()
	cfg := &config.Config{}
	cfg.Auth.Enabled = testAuthEnabled
	cfg.Auth.HMACSecret = testHMACSecret
	cfg.General.TenantID = testKeyTenantID
	resCfg := config.GatewayResilienceConfig{DialTimeout: testDialTimeout, MaxRetries: testMaxRetries, RetryBackoff: testRetryBackoff, RetryMaxBackoff: testRetryBackoff,
		CBFailureThreshold: testAPIKeyBreak, CBTimeout: time.Minute, CBInterval: time.Minute}
	l := logger.NewNop()

	ups, err := dialUpstreams(l, coreAddr, identityAddr, resCfg)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, ups.Close()) })
	internal := pb.NewIdentityInternalServiceClient(ups.identityInternal)
	auth, err := newAuthInterceptor(cfg, l,
		adapter.NewIdentityRPCOrgAdapter(internal, "", l, time.Minute, 1),
		adapter.NewIdentityRPCAPIKeyAdapter(internal, ""),
		adapter.NewIdentityRPCEventAdapter(internal, "", testKeyTenantID, l))
	require.NoError(t, err)
	core, identity := resilience.NewUpstreamBreaker(upstreamCore, resCfg), resilience.NewUpstreamBreaker(upstreamIdentity, resCfg)
	proxy := newProxyServer(cfg, l, ups,
		buildUnaryChain(auth.UnaryInterceptor(), nil),
		buildStreamChain(nil, auth.StreamInterceptor(), nil, breakerStreamInterceptor(core, identity, 0)))

	lis, err := net.Listen("tcp", testUpstreamListen)
	require.NoError(t, err)
	go func() { _ = proxy.Serve(lis) }()
	t.Cleanup(proxy.Stop)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return pb.NewCoreServiceClient(conn)
}

func withAPIKey(rawKey string) (context.Context, context.CancelFunc) {
	ctx, cancel := testutil.TestContextWithTimeout(testCallTimeout)
	return metadata.AppendToOutgoingContext(ctx, grpcconst.MetadataKeyAuthorization, grpcconst.BearerPrefix+rawKey), cancel
}

// A community API key authenticates a call through the gateway, which
// forwards the key's tenant, organization and owner; a revoked, an expired
// and a deleted key are refused before the call reaches KC-Core.
func TestGateway_AuthenticatesAPIKeysUntilRevokedExpiredOrDeleted(t *testing.T) {
	owner := testKeyUserID.String()
	identityAddr := serveOn(t, func(s *grpc.Server) {
		pb.RegisterIdentityInternalServiceServer(s, keyedIdentity{keys: map[string]*pb.ValidateAPIKeyResponse{
			keyHash(testKeyActive):  {Id: uuid.NewString(), TenantId: testKeyTenantID, OrganizationId: testKeyOrgID.String(), UserId: owner, IsActive: true},
			keyHash(testKeyExpired): {Id: uuid.NewString(), TenantId: testKeyTenantID, OrganizationId: testKeyOrgID.String(), UserId: owner, IsActive: true, IsExpired: true},
			keyHash(testKeyRevoked): {Id: uuid.NewString(), TenantId: testKeyTenantID, OrganizationId: testKeyOrgID.String(), UserId: owner, IsActive: testKeyInactive},
		}})
	})
	upstream := &statusCore{}
	coreAddr := serveOn(t, func(s *grpc.Server) { pb.RegisterCoreServiceServer(s, upstream) })
	gateway := gatewayFor(t, identityAddr, coreAddr)

	ctx, cancel := withAPIKey(testKeyActive)
	defer cancel()
	_, err := gateway.GetSystemStatus(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	upstream.mu.Lock()
	assert.Equal(t, []string{"1"}, upstream.forwarded.Get(grpcconst.MetadataKeyInternalTenantID))
	assert.Equal(t, []string{testKeyOrgID.String()}, upstream.forwarded.Get(grpcconst.MetadataKeyInternalOrgID))
	assert.Equal(t, []string{owner}, upstream.forwarded.Get(grpcconst.MetadataKeyInternalUserID))
	upstream.forwarded = nil
	upstream.mu.Unlock()

	ctx, cancel = withAPIKey(testKeyRevoked)
	defer cancel()
	_, err = gateway.GetSystemStatus(ctx, &emptypb.Empty{})
	assert.Equal(t, grpcconst.GetGRPCCode(grpcconst.ErrTokenApiKeyInactive), status.Code(err))
	assert.Equal(t, grpcconst.ResolveErrorMessage(grpcconst.ErrTokenApiKeyInactive), status.Convert(err).Message())

	ctx, cancel = withAPIKey(testKeyExpired)
	defer cancel()
	_, err = gateway.GetSystemStatus(ctx, &emptypb.Empty{})
	assert.Equal(t, grpcconst.ResolveErrorMessage(grpcconst.ErrTokenApiKeyExpired), status.Convert(err).Message())

	ctx, cancel = withAPIKey(testKeyDeleted)
	defer cancel()
	_, err = gateway.GetSystemStatus(ctx, &emptypb.Empty{})
	assert.Equal(t, grpcconst.ResolveErrorMessage(grpcconst.ErrTokenInvalidToken), status.Convert(err).Message())

	upstream.mu.Lock()
	defer upstream.mu.Unlock()
	assert.Nil(t, upstream.forwarded, "a refused key must not reach KC-Core")
}
