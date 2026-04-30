package internal_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	kilocenterv1 "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcconst "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/grpc/interceptors"
	"github.com/kilocenter/KC-Core/pkg/logger"
	"github.com/kilocenter/KC-Gateway/internal/adapter"
	gatewayproxy "github.com/kilocenter/KC-Gateway/internal/proxy"
	pkgcontext "github.com/kilocenter/pkg/context"
	grpcproxy "github.com/mwitkow/grpc-proxy/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Shared test fixtures
var (
	testOrgUUID  = uuid.MustParse("11111111-2222-3333-4444-555555555555")
	testUserUUID = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	testTenantID = int64(42)
	rawAPIKey    = "test-api-key-for-integration"
	expectedHash string
)

func init() {
	hash := sha256.Sum256([]byte(rawAPIKey))
	expectedHash = hex.EncodeToString(hash[:])
}

// stubIdentityInternalService provides canned responses for auth/org resolution tests.
type stubIdentityInternalService struct {
	kilocenterv1.UnimplementedIdentityInternalServiceServer
}

func (s *stubIdentityInternalService) ResolveOrg(_ context.Context, req *kilocenterv1.ResolveOrgRequest) (*kilocenterv1.ResolveOrgResponse, error) {
	if req.GetOrgId() == testOrgUUID.String() {
		return &kilocenterv1.ResolveOrgResponse{TenantId: testTenantID}, nil
	}
	return nil, status.Errorf(codes.NotFound, "org not found")
}

func (s *stubIdentityInternalService) GetDefaultOrgForTenant(_ context.Context, req *kilocenterv1.GetDefaultOrgForTenantRequest) (*kilocenterv1.GetDefaultOrgForTenantResponse, error) {
	if req.GetTenantId() == testTenantID {
		return &kilocenterv1.GetDefaultOrgForTenantResponse{OrgId: testOrgUUID.String()}, nil
	}
	return nil, status.Errorf(codes.NotFound, "tenant not found")
}

func (s *stubIdentityInternalService) ValidateAPIKey(_ context.Context, req *kilocenterv1.ValidateAPIKeyRequest) (*kilocenterv1.ValidateAPIKeyResponse, error) {
	if req.GetKeyHash() != expectedHash {
		return nil, status.Errorf(codes.NotFound, "api key not found")
	}
	return &kilocenterv1.ValidateAPIKeyResponse{
		Id:             uuid.New().String(),
		TenantId:       testTenantID,
		OrganizationId: testOrgUUID.String(),
		UserId:         testUserUUID.String(),
		IsActive:       true,
		IsExpired:      false,
	}, nil
}

func (s *stubIdentityInternalService) UpdateAPIKeyLastUsed(_ context.Context, _ *kilocenterv1.UpdateAPIKeyLastUsedRequest) (*kilocenterv1.UpdateAPIKeyLastUsedResponse, error) {
	return &kilocenterv1.UpdateAPIKeyLastUsedResponse{}, nil
}

func (s *stubIdentityInternalService) RecordPlatformEvent(_ context.Context, _ *kilocenterv1.RecordPlatformEventRequest) (*kilocenterv1.RecordPlatformEventResponse, error) {
	return &kilocenterv1.RecordPlatformEventResponse{}, nil
}

// capturedMetadata stores metadata received by the upstream stub server.
type capturedMetadata struct {
	mu        sync.Mutex
	md        metadata.MD
	callCount atomic.Int64
}

func (c *capturedMetadata) capture(ctx context.Context) {
	c.callCount.Add(1)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		c.mu.Lock()
		c.md = md.Copy()
		c.mu.Unlock()
	}
}

func (c *capturedMetadata) getMD() metadata.MD {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.md.Copy()
}

// stubCoreService captures incoming metadata for assertions.
type stubCoreService struct {
	kilocenterv1.UnimplementedCoreServiceServer
	captured *capturedMetadata
}

func (s *stubCoreService) ListEndPoints(ctx context.Context, _ *kilocenterv1.ListEndPointsRequest) (*kilocenterv1.ListEndPointsResponse, error) {
	s.captured.capture(ctx)
	return &kilocenterv1.ListEndPointsResponse{}, nil
}

func (s *stubCoreService) GetReleaseInfo(ctx context.Context, _ *emptypb.Empty) (*kilocenterv1.ReleaseInfo, error) {
	s.captured.capture(ctx)
	return &kilocenterv1.ReleaseInfo{}, nil
}

// twoHopSetup holds the gateway and upstream servers plus the captured metadata.
type twoHopSetup struct {
	gatewayAddr string
	captured    *capturedMetadata
	cleanup     func()
}

// newTwoHopSetup creates the full two-hop test infrastructure:
// upstream gRPC server (stub CoreService) ← gateway (auth + org resolver + proxy)
// plus a stub IdentityInternalService for auth/org resolution via RPC
func newTwoHopSetup(t *testing.T) *twoHopSetup {
	t.Helper()

	logger.Initialize("error", "text")
	l := logger.Get()
	captured := &capturedMetadata{}

	// --- Stub Identity service (for auth/org resolution) ---
	identityLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "identity listener")

	identityServer := grpc.NewServer()
	kilocenterv1.RegisterIdentityInternalServiceServer(identityServer, &stubIdentityInternalService{})
	go func() { _ = identityServer.Serve(identityLis) }()

	identityConn, err := grpc.NewClient(
		identityLis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "identity conn")

	internalClient := kilocenterv1.NewIdentityInternalServiceClient(identityConn)

	// --- Upstream (hop 2): stub CoreService that captures metadata ---
	upstreamLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "upstream listener")

	upstreamServer := grpc.NewServer()
	kilocenterv1.RegisterCoreServiceServer(upstreamServer, &stubCoreService{captured: captured})

	go func() { _ = upstreamServer.Serve(upstreamLis) }()

	// --- Upstream connection for gateway ---
	upstreamConn, err := grpc.NewClient(
		upstreamLis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "upstream conn")

	// --- Gateway (hop 1): auth → org resolver → proxy ---
	apiKeyAdapter := adapter.NewIdentityRPCAPIKeyAdapter(internalClient, "")
	orgAdapter := adapter.NewIdentityRPCOrgAdapter(internalClient, "", l, 5*time.Minute, 1000)

	authInterceptor, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{
		Enabled: true,
	})
	require.NoError(t, err, "auth interceptor")
	authInterceptor.WithAPIKeyAuthenticator(apiKeyAdapter)
	authInterceptor.WithOrganizationResolver(orgAdapter)
	authInterceptor.WithTenantResolver(orgAdapter)

	skipMethods := make([]string, 0, len(grpcconst.OrgExemptMethods))
	for m := range grpcconst.OrgExemptMethods {
		skipMethods = append(skipMethods, m)
	}
	orgInterceptor, err := interceptors.NewOrgResolverInterceptor(interceptors.OrgResolverInterceptorConfig{
		Resolver:    orgAdapter,
		Logger:      l,
		SkipMethods: skipMethods,
	})
	require.NoError(t, err, "org interceptor")

	director := func(ctx context.Context, _ string) (context.Context, grpc.ClientConnInterface, error) {
		outMD := gatewayproxy.SanitizeAndInject(ctx)
		outCtx := metadata.NewOutgoingContext(ctx, outMD)
		return outCtx, upstreamConn, nil
	}

	gatewayServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			authInterceptor.UnaryInterceptor(),
			orgInterceptor.UnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			authInterceptor.StreamInterceptor(),
			orgInterceptor.StreamInterceptor(),
		),
		grpc.UnknownServiceHandler(grpcproxy.TransparentHandler(director)),
	)

	gatewayLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "gateway listener")
	go func() { _ = gatewayServer.Serve(gatewayLis) }()

	cleanup := func() {
		gatewayServer.GracefulStop()
		upstreamServer.GracefulStop()
		identityServer.GracefulStop()
		_ = upstreamConn.Close()
		_ = identityConn.Close()
	}

	return &twoHopSetup{
		gatewayAddr: gatewayLis.Addr().String(),
		captured:    captured,
		cleanup:     cleanup,
	}
}

func TestGatewayTwoHop_AuthenticatedRequest(t *testing.T) {
	setup := newTwoHopSetup(t)
	defer setup.cleanup()

	// Connect to gateway
	conn, err := grpc.NewClient(
		setup.gatewayAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	client := kilocenterv1.NewCoreServiceClient(conn)

	// Send request with valid auth + spoofed internal header
	md := metadata.Pairs(
		"authorization", "Bearer "+rawAPIKey,
		"x-organization-id", testOrgUUID.String(),
		"x-user-id", testUserUUID.String(),
		"x-kc-internal-tenant-id", "999", // spoofed — must be stripped
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	_, err = client.ListEndPoints(ctx, &kilocenterv1.ListEndPointsRequest{})
	require.NoError(t, err, "ListEndPoints should succeed through gateway")

	// Verify upstream received correct trusted headers
	assert.Equal(t, int64(1), setup.captured.callCount.Load(), "upstream should be called exactly once")

	upstreamMD := setup.captured.getMD()

	// Trusted internal headers injected by gateway
	assert.Equal(t, []string{"42"}, upstreamMD.Get(grpcconst.MetadataKeyInternalTenantID),
		"internal tenant ID should be 42 (from auth), not spoofed 999")
	assert.Equal(t, []string{testOrgUUID.String()}, upstreamMD.Get(grpcconst.MetadataKeyInternalOrgID),
		"internal org ID should match authenticated org")
	assert.Equal(t, []string{testUserUUID.String()}, upstreamMD.Get(grpcconst.MetadataKeyInternalUserID),
		"internal user ID should match authenticated user")

	// Client-facing headers must be stripped
	assert.Empty(t, upstreamMD.Get(grpcconst.MetadataKeyAuthorization),
		"authorization header must be stripped before upstream")
	assert.Empty(t, upstreamMD.Get(grpcconst.MetadataKeyOrganizationID),
		"x-organization-id header must be stripped before upstream")
	assert.Empty(t, upstreamMD.Get(grpcconst.MetadataKeyUserID),
		"x-user-id header must be stripped before upstream")
}

func TestGatewayTwoHop_UnauthenticatedRequest(t *testing.T) {
	setup := newTwoHopSetup(t)
	defer setup.cleanup()

	// Connect to gateway
	conn, err := grpc.NewClient(
		setup.gatewayAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	client := kilocenterv1.NewCoreServiceClient(conn)

	// Send request with NO authorization header
	_, err = client.ListEndPoints(context.Background(), &kilocenterv1.ListEndPointsRequest{})
	require.Error(t, err, "unauthenticated request should fail")

	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	assert.Equal(t, codes.Unauthenticated, st.Code(), "should return Unauthenticated")

	// Upstream must never be reached
	assert.Equal(t, int64(0), setup.captured.callCount.Load(),
		"upstream should not be called for unauthenticated requests")
}

func TestGatewayTwoHop_TraceContextPropagation(t *testing.T) {
	setup := newTwoHopSetup(t)
	defer setup.cleanup()

	conn, err := grpc.NewClient(
		setup.gatewayAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	client := kilocenterv1.NewCoreServiceClient(conn)

	md := metadata.Pairs(
		"authorization", "Bearer "+rawAPIKey,
		"x-organization-id", testOrgUUID.String(),
		"x-user-id", testUserUUID.String(),
		"traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	_, err = client.ListEndPoints(ctx, &kilocenterv1.ListEndPointsRequest{})
	require.NoError(t, err, "ListEndPoints should succeed with traceparent")

	upstreamMD := setup.captured.getMD()

	traceParent := upstreamMD.Get("traceparent")
	require.NotEmpty(t, traceParent, "traceparent must reach upstream")
	assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		traceParent[0], "traceparent should propagate unchanged through gateway")
}

// communityModeSetup is identical to twoHopSetup but configures the director
// with the community-mode tenant fallback (no org resolver interceptor).
type communityModeSetup struct {
	gatewayAddr      string
	captured         *capturedMetadata
	communityDefault int64
	cleanup          func()
}

// newCommunityModeSetup creates a gateway without org resolver but with the
// community-mode director fallback that mirrors main.go behavior.
func newCommunityModeSetup(t *testing.T, communityDefaultTenant int64) *communityModeSetup {
	t.Helper()

	logger.Initialize("error", "text")
	l := logger.Get()
	captured := &capturedMetadata{}

	// --- Stub Identity service ---
	identityLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "identity listener")

	identityServer := grpc.NewServer()
	kilocenterv1.RegisterIdentityInternalServiceServer(identityServer, &stubIdentityInternalService{})
	go func() { _ = identityServer.Serve(identityLis) }()

	identityConn, err := grpc.NewClient(
		identityLis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "identity conn")

	internalClient := kilocenterv1.NewIdentityInternalServiceClient(identityConn)

	// --- Upstream stub ---
	upstreamLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "upstream listener")

	upstreamServer := grpc.NewServer()
	kilocenterv1.RegisterCoreServiceServer(upstreamServer, &stubCoreService{captured: captured})
	go func() { _ = upstreamServer.Serve(upstreamLis) }()

	upstreamConn, err := grpc.NewClient(
		upstreamLis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "upstream conn")

	// --- Auth interceptor (no org resolver — community mode) ---
	apiKeyAdapter := adapter.NewIdentityRPCAPIKeyAdapter(internalClient, "")
	orgAdapter := adapter.NewIdentityRPCOrgAdapter(internalClient, "", l, 5*time.Minute, 1000)

	authInterceptor, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{
		Enabled: true,
	})
	require.NoError(t, err, "auth interceptor")
	authInterceptor.WithAPIKeyAuthenticator(apiKeyAdapter)
	authInterceptor.WithOrganizationResolver(orgAdapter)
	authInterceptor.WithTenantResolver(orgAdapter)

	// Director with community-mode fallback (mirrors main.go logic)
	director := func(ctx context.Context, _ string) (context.Context, grpc.ClientConnInterface, error) {
		if communityDefaultTenant > 0 {
			if _, err := pkgcontext.GetTenantID(ctx); err != nil {
				ctx = pkgcontext.WithTenantID(ctx, communityDefaultTenant)
			}
		}
		outMD := gatewayproxy.SanitizeAndInject(ctx)
		outCtx := metadata.NewOutgoingContext(ctx, outMD)
		return outCtx, upstreamConn, nil
	}

	gatewayServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			authInterceptor.UnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			authInterceptor.StreamInterceptor(),
		),
		grpc.UnknownServiceHandler(grpcproxy.TransparentHandler(director)),
	)

	gatewayLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "gateway listener")
	go func() { _ = gatewayServer.Serve(gatewayLis) }()

	cleanup := func() {
		gatewayServer.GracefulStop()
		upstreamServer.GracefulStop()
		identityServer.GracefulStop()
		_ = upstreamConn.Close()
		_ = identityConn.Close()
	}

	return &communityModeSetup{
		gatewayAddr:      gatewayLis.Addr().String(),
		captured:         captured,
		communityDefault: communityDefaultTenant,
		cleanup:          cleanup,
	}
}

// TestCommunityMode_AuthenticatedTenantPreserved verifies that when auth resolves
// a tenant (e.g., tenant 42), the community-mode director does NOT overwrite it
// with the default tenant.
func TestCommunityMode_AuthenticatedTenantPreserved(t *testing.T) {
	communityDefault := int64(1)
	setup := newCommunityModeSetup(t, communityDefault)
	defer setup.cleanup()

	conn, err := grpc.NewClient(
		setup.gatewayAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	client := kilocenterv1.NewCoreServiceClient(conn)

	md := metadata.Pairs(
		"authorization", "Bearer "+rawAPIKey,
		"x-organization-id", testOrgUUID.String(),
		"x-user-id", testUserUUID.String(),
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	_, err = client.ListEndPoints(ctx, &kilocenterv1.ListEndPointsRequest{})
	require.NoError(t, err, "ListEndPoints should succeed")

	assert.Equal(t, int64(1), setup.captured.callCount.Load(), "upstream called once")

	upstreamMD := setup.captured.getMD()
	tenantValues := upstreamMD.Get(grpcconst.MetadataKeyInternalTenantID)
	require.NotEmpty(t, tenantValues, "tenant header must be present")

	tenantID, err := strconv.ParseInt(tenantValues[0], 10, 64)
	require.NoError(t, err, "tenant header must be numeric")
	assert.Equal(t, testTenantID, tenantID,
		"authenticated tenant (42) must NOT be overwritten by community default (1)")
}

// TestCommunityMode_FallbackOnlyWhenTenantMissing verifies that the director
// injects the community default tenant only when auth did not set one
// (e.g., for public/exempt methods that bypass auth).
func TestCommunityMode_FallbackOnlyWhenTenantMissing(t *testing.T) {
	communityDefault := int64(1)
	setup := newCommunityModeSetup(t, communityDefault)
	defer setup.cleanup()

	conn, err := grpc.NewClient(
		setup.gatewayAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// GetReleaseInfo is a public method — skips auth entirely.
	// Without auth, no tenant is set in context, so director should inject default.
	client := kilocenterv1.NewCoreServiceClient(conn)
	_, err = client.GetReleaseInfo(context.Background(), &emptypb.Empty{})
	require.NoError(t, err, "GetReleaseInfo (public) should succeed without auth")

	assert.Equal(t, int64(1), setup.captured.callCount.Load(), "upstream called once")

	upstreamMD := setup.captured.getMD()
	tenantValues := upstreamMD.Get(grpcconst.MetadataKeyInternalTenantID)
	require.NotEmpty(t, tenantValues, "community default tenant should be injected")

	tenantID, err := strconv.ParseInt(tenantValues[0], 10, 64)
	require.NoError(t, err, "tenant header must be numeric")
	assert.Equal(t, communityDefault, tenantID,
		"public method without auth should receive community default tenant")
}

// TestStrictMode_MissingOrgHeaderRejected verifies that with org enforcement enabled,
// an authenticated request missing x-organization-id is rejected before reaching upstream.
func TestStrictMode_MissingOrgHeaderRejected(t *testing.T) {
	setup := newTwoHopSetup(t) // strict mode (org resolver enabled)
	defer setup.cleanup()

	conn, err := grpc.NewClient(
		setup.gatewayAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	client := kilocenterv1.NewCoreServiceClient(conn)

	// Send request with valid API key auth but NO x-organization-id header
	md := metadata.Pairs(
		"authorization", "Bearer "+rawAPIKey,
		"x-user-id", testUserUUID.String(),
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	_, err = client.ListEndPoints(ctx, &kilocenterv1.ListEndPointsRequest{})
	require.Error(t, err, "missing org header should fail in strict mode")

	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status")
	assert.Contains(t, []codes.Code{codes.Unauthenticated, codes.PermissionDenied}, st.Code(),
		"should reject with auth/permission error")

	assert.Equal(t, int64(0), setup.captured.callCount.Load(),
		"upstream must not be reached when org header is missing in strict mode")
}

// TestPublicMethodWorksRegardless verifies that public methods (e.g., GetReleaseInfo)
// succeed without auth in both strict and community modes.
func TestPublicMethodWorksRegardless(t *testing.T) {
	t.Run("strict mode", func(t *testing.T) {
		setup := newTwoHopSetup(t)
		defer setup.cleanup()

		conn, err := grpc.NewClient(
			setup.gatewayAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		client := kilocenterv1.NewCoreServiceClient(conn)
		_, err = client.GetReleaseInfo(context.Background(), &emptypb.Empty{})
		require.NoError(t, err, "GetReleaseInfo (public) should succeed without auth in strict mode")

		assert.Equal(t, int64(1), setup.captured.callCount.Load(), "upstream should be reached")
	})

	t.Run("community mode", func(t *testing.T) {
		setup := newCommunityModeSetup(t, 1)
		defer setup.cleanup()

		conn, err := grpc.NewClient(
			setup.gatewayAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()

		client := kilocenterv1.NewCoreServiceClient(conn)
		_, err = client.GetReleaseInfo(context.Background(), &emptypb.Empty{})
		require.NoError(t, err, "GetReleaseInfo (public) should succeed without auth in community mode")

		assert.Equal(t, int64(1), setup.captured.callCount.Load(), "upstream should be reached")
	})
}
