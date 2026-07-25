//go:build integration

package proxy

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	grpcproxy "github.com/mwitkow/grpc-proxy/proxy"
	"google.golang.org/grpc/metadata"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// startMatrixGateway starts a self-contained in-process gateway: the
// production SelectUpstream routing and metadata sanitization in front of
// fake core/identity upstreams (which answer NotFound for every routed call,
// so anything that reaches them is provably not Unimplemented), with the real
// auth interceptor enabled and no credentials supplied. The matrix therefore
// verifies routing, internal-service blocking, and auth gating without any
// external stack.
// matrixCoreStub serves the public CoreService surface the matrix asserts on.
type matrixCoreStub struct {
	pb.UnimplementedCoreServiceServer
}

func (matrixCoreStub) GetReleaseInfo(_ context.Context, _ *emptypb.Empty) (*pb.ReleaseInfo, error) {
	return &pb.ReleaseInfo{Version: "matrix-fake"}, nil
}

// matrixCompatCoreStub serves the KiloCenterService compat methods routed to core.
type matrixCompatCoreStub struct {
	pb.UnimplementedKiloCenterServiceServer
}

func (matrixCompatCoreStub) GetReleaseInfo(_ context.Context, _ *emptypb.Empty) (*pb.ReleaseInfo, error) {
	return &pb.ReleaseInfo{Version: "matrix-fake"}, nil
}

// matrixIdentityStub serves the public IdentityService surface; the other
// public methods answer a domain error (never Unimplemented).
type matrixIdentityStub struct {
	pb.UnimplementedIdentityServiceServer
}

func (matrixIdentityStub) GetAuthSettings(_ context.Context, _ *pb.GetAuthSettingsRequest) (*pb.GetAuthSettingsResponse, error) {
	return &pb.GetAuthSettingsResponse{Settings: &pb.AuthSettings{}}, nil
}

func (matrixIdentityStub) Login(_ context.Context, _ *pb.LoginRequest) (*pb.LoginResponse, error) {
	return nil, status.Error(codes.InvalidArgument, "matrix fake identity")
}

func (matrixIdentityStub) RefreshTokens(_ context.Context, _ *pb.RefreshTokensRequest) (*pb.RefreshTokensResponse, error) {
	return nil, status.Error(codes.InvalidArgument, "matrix fake identity")
}

func (matrixIdentityStub) ExchangeOIDC(_ context.Context, _ *pb.ExchangeOIDCRequest) (*pb.LoginResponse, error) {
	return nil, status.Error(codes.InvalidArgument, "matrix fake identity")
}

func (matrixIdentityStub) ExchangeOAuth2(_ context.Context, _ *pb.ExchangeOAuth2Request) (*pb.LoginResponse, error) {
	return nil, status.Error(codes.InvalidArgument, "matrix fake identity")
}

func (matrixIdentityStub) RegisterAccount(_ context.Context, _ *pb.RegisterAccountRequest) (*pb.LoginResponse, error) {
	return nil, status.Error(codes.InvalidArgument, "matrix fake identity")
}

func startMatrixGateway(t *testing.T) string {
	t.Helper()
	logger.Initialize("error", "text")
	l := logger.Get()

	fakeUpstream := func(register func(*grpc.Server)) net.Listener {
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		srv := grpc.NewServer(grpc.UnknownServiceHandler(func(_ interface{}, _ grpc.ServerStream) error {
			return status.Error(codes.NotFound, "matrix fake upstream")
		}))
		register(srv)
		go func() { _ = srv.Serve(lis) }()
		t.Cleanup(srv.GracefulStop)
		return lis
	}

	coreLis := fakeUpstream(func(srv *grpc.Server) {
		pb.RegisterCoreServiceServer(srv, &matrixCoreStub{})
		pb.RegisterKiloCenterServiceServer(srv, &matrixCompatCoreStub{})
	})
	identityLis := fakeUpstream(func(srv *grpc.Server) {
		pb.RegisterIdentityServiceServer(srv, &matrixIdentityStub{})
	})

	dial := func(addr string) *grpc.ClientConn {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })
		return conn
	}
	coreConn := dial(coreLis.Addr().String())
	identityConn := dial(identityLis.Addr().String())

	authInterceptor, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	require.NoError(t, err)
	_ = l

	director := func(ctx context.Context, fullMethod string) (context.Context, grpc.ClientConnInterface, error) {
		upstream, selErr := SelectUpstream(fullMethod, coreConn, identityConn)
		if selErr != nil {
			return nil, nil, selErr
		}
		outMD := SanitizeAndInject(ctx)
		return metadata.NewOutgoingContext(ctx, outMD), upstream, nil
	}

	gatewayServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(authInterceptor.UnaryInterceptor()),
		grpc.ChainStreamInterceptor(authInterceptor.StreamInterceptor()),
		grpc.UnknownServiceHandler(grpcproxy.TransparentHandler(director)),
	)
	gatewayLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() { _ = gatewayServer.Serve(gatewayLis) }()
	t.Cleanup(gatewayServer.GracefulStop)

	return gatewayLis.Addr().String()
}

// publicMethods that should return OK or domain error (NOT Unimplemented) via gateway.
var publicCoreServiceMethods = []string{
	"/kilocenter.api.v1.CoreService/GetReleaseInfo",
}

var publicIdentityServiceMethods = []string{
	"/kilocenter.api.v1.IdentityService/Login",
	"/kilocenter.api.v1.IdentityService/RefreshTokens",
	"/kilocenter.api.v1.IdentityService/GetAuthSettings",
	"/kilocenter.api.v1.IdentityService/ExchangeOIDC",
	"/kilocenter.api.v1.IdentityService/ExchangeOAuth2",
	"/kilocenter.api.v1.IdentityService/RegisterAccount",
}

var publicKiloCenterServiceMethods = []string{
	"/kilocenter.api.v1.KiloCenterService/Login",
	"/kilocenter.api.v1.KiloCenterService/RefreshTokens",
	"/kilocenter.api.v1.KiloCenterService/GetAuthSettings",
	"/kilocenter.api.v1.KiloCenterService/ExchangeOIDC",
	"/kilocenter.api.v1.KiloCenterService/ExchangeOAuth2",
	"/kilocenter.api.v1.KiloCenterService/GetReleaseInfo",
	"/kilocenter.api.v1.KiloCenterService/RegisterAccount",
}

// identityInternalServiceMethods should be blocked (Unimplemented) via gateway.
var identityInternalServiceMethods = []string{
	"/kilocenter.api.v1.IdentityInternalService/ResolveOrg",
	"/kilocenter.api.v1.IdentityInternalService/GetDefaultOrgForTenant",
	"/kilocenter.api.v1.IdentityInternalService/ValidateAPIKey",
	"/kilocenter.api.v1.IdentityInternalService/UpdateAPIKeyLastUsed",
}

// coreServiceUnaryMethods lists all CoreService unary RPCs (extracted from _ServiceDesc).
var coreServiceUnaryMethods = []string{
	"CreateEndPoint", "GetEndPoint", "UpdateEndPoint", "DeleteEndPoint", "ListEndPoints",
	"AttachEndPoint", "DetachEndPoint",
	"CreateBaseStation", "GetBaseStation", "UpdateBaseStation", "DeleteBaseStation", "ListBaseStations",
	"GetBaseStationStats", "UpdateBaseStationEui",
	"GetMessage", "SendDownlink", "RevokeDownlink", "ListDownlinkQueue", "GetDownlinkResults",
	"SendULTransmit", "RequestBaseStationStatus", "InitiatePing",
	"GetDLRXStatus", "QueryDLRXStatus", "GetDLRXStatusQueries",
	"GetSystemStatus", "GetStatistics", "GetReleaseInfo",
	"CreateIntegration", "GetIntegration", "UpdateIntegration", "DeleteIntegration", "ListIntegrations",
	"GetAnalyticsOverview", "GetActivityAnalytics", "GetSignalQualityAnalytics",
	"ListEvents", "ListBaseStationActivity", "ListEndpointActivity",
	"ListAlerts", "GetAlertSummary",
	"ListScaciSessions", "GetScaciSession", "GetScaciStatistics", "ListScaciErrors", "ListScaciQueues", "GetScaciStatus",
	"GenerateCertificate", "DownloadCertificate", "DownloadBaseStationCertificate",
	"GenerateServerCertificates", "RenewServerCertificates", "GetServerCertificateStatus",
	"CreateManufacturer", "GetManufacturer", "UpdateManufacturer", "DeleteManufacturer", "ListManufacturers",
	"CreateDeviceModel", "GetDeviceModel", "UpdateDeviceModel", "DeleteDeviceModel", "ListDeviceModels",
	"CreateBlueprint", "GetBlueprint", "UpdateBlueprint", "DeleteBlueprint", "ListBlueprints",
	"SetDefaultBlueprint", "SubmitBlueprintToRegistry", "CreateDeviceModelWithBlueprint", "DecodePreview",
	"ListMessages", "ListBaseStationMessages", "GetBaseStationMessage", "GetBaseStationMessageStats",
	"SearchBaseStationMessages", "ExportBaseStationMessages",
	"ListEndpointMessages", "GetEndPointStats", "GetEndPointOperations",
}

// coreServiceStreamMethods lists CoreService streaming RPCs.
var coreServiceStreamMethods = []string{
	"StreamEvents",
	"StreamMessages", "StreamBaseStationMessages",
}

// identityServiceUnaryMethods lists all IdentityService unary RPCs.
var identityServiceUnaryMethods = []string{
	"Login", "RefreshTokens", "GetProfile", "GetAuthSettings", "Logout", "ChangePassword",
	"ExchangeOIDC", "ExchangeOAuth2", "RegisterAccount",
	"CreateUser", "GetUser", "UpdateUser", "DeleteUser", "ListUsers", "UpdateUserPassword",
	"CreateOrganization", "GetOrganization", "UpdateOrganization", "DeleteOrganization", "ListOrganizations",
	"AddOrganizationUser", "GetOrganizationUser", "UpdateOrganizationUser", "RemoveOrganizationUser", "ListOrganizationUsers", "ListUserOrganizations",
	"CreateApiKey", "GetApiKey", "DeleteApiKey", "ListApiKeys",
}

func TestVerificationMatrix(t *testing.T) {
	runVerificationMatrix(t, startMatrixGateway(t))
}

// TestVerificationMatrixExternal runs the same matrix against a running
// gateway (full-stack smoke); it skips unless GATEWAY_ADDR is set.
func TestVerificationMatrixExternal(t *testing.T) {
	addr := os.Getenv("GATEWAY_ADDR")
	if addr == "" {
		t.Skip("GATEWAY_ADDR not set; external gateway matrix skipped (in-process matrix covers routing)")
	}
	runVerificationMatrix(t, addr)
}

// matrixClient bundles the shared connection and context the matrix assertion
// helpers invoke methods with.
type matrixClient struct {
	conn *grpc.ClientConn
	ctx  context.Context
}

// mustStatus asserts err carries a gRPC status and returns it.
func mustStatus(t *testing.T, err error) *status.Status {
	t.Helper()
	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error")
	return st
}

// assertBlocked verifies each fully qualified method is effectively blocked.
// The auth interceptor may reject unauthenticated callers before the service
// routing layer, so Unimplemented and Unauthenticated both count as blocked.
func (c matrixClient) assertBlocked(t *testing.T, label string, methods []string) {
	blocked := map[codes.Code]bool{codes.Unimplemented: true, codes.Unauthenticated: true}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var resp emptypb.Empty
			err := c.conn.Invoke(c.ctx, method, &emptypb.Empty{}, &resp)
			st := mustStatus(t, err)
			assert.True(t, blocked[st.Code()],
				"%s method %s should be blocked (Unimplemented or Unauthenticated), got %s: %s",
				label, method, st.Code(), st.Message())
		})
	}
}

// assertNotUnimplemented verifies each fully qualified public method reaches a
// real handler: success or any domain error passes, Unimplemented fails.
func (c matrixClient) assertNotUnimplemented(t *testing.T, methods []string) {
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			var resp emptypb.Empty
			err := c.conn.Invoke(c.ctx, method, &emptypb.Empty{}, &resp)
			if err == nil {
				return // OK — public method succeeded
			}
			st := mustStatus(t, err)
			assert.NotEqual(t, codes.Unimplemented, st.Code(),
				"public method %s should not return Unimplemented", method)
		})
	}
}

// assertUnauthenticatedUnary verifies each non-public unary method under the
// service prefix rejects anonymous callers with Unauthenticated.
func (c matrixClient) assertUnauthenticatedUnary(t *testing.T, servicePrefix string, methods []string) {
	for _, method := range methods {
		fullMethod := servicePrefix + method
		if grpcconst.PublicMethods[fullMethod] {
			continue // Skip public methods
		}
		t.Run(method, func(t *testing.T) {
			var resp emptypb.Empty
			err := c.conn.Invoke(c.ctx, fullMethod, &emptypb.Empty{}, &resp)
			require.Error(t, err)
			st := mustStatus(t, err)
			assert.Equal(t, codes.Unauthenticated, st.Code(),
				"auth-required method %s should return Unauthenticated, got %s: %s",
				fullMethod, st.Code(), st.Message())
		})
	}
}

// assertUnauthenticatedStream verifies each streaming method under the service
// prefix rejects anonymous callers with Unauthenticated, whether the rejection
// arrives at stream open or on the first receive.
func (c matrixClient) assertUnauthenticatedStream(t *testing.T, servicePrefix string, methods []string) {
	for _, method := range methods {
		fullMethod := servicePrefix + method
		t.Run(method, func(t *testing.T) {
			stream, err := c.conn.NewStream(c.ctx,
				&grpc.StreamDesc{ServerStreams: true},
				fullMethod)
			if err != nil {
				st := mustStatus(t, err)
				assert.Equal(t, codes.Unauthenticated, st.Code(),
					"streaming method %s should return Unauthenticated", fullMethod)
				return
			}
			// Stream opened — try to receive; auth rejection comes here
			err = stream.RecvMsg(&emptypb.Empty{})
			require.Error(t, err)
			st := mustStatus(t, err)
			assert.Equal(t, codes.Unauthenticated, st.Code(),
				"streaming method %s should reject with Unauthenticated, got %s",
				fullMethod, st.Code())
		})
	}
}

func runVerificationMatrix(t *testing.T, addr string) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := matrixClient{conn: conn, ctx: ctx}
	empty := &emptypb.Empty{}

	t.Run("IdentityInternalService_Blocked", func(t *testing.T) {
		client.assertBlocked(t, "IdentityInternalService", identityInternalServiceMethods)
	})

	t.Run("Public_CoreService", func(t *testing.T) {
		client.assertNotUnimplemented(t, publicCoreServiceMethods)
	})

	t.Run("Public_IdentityService", func(t *testing.T) {
		client.assertNotUnimplemented(t, publicIdentityServiceMethods)
	})

	t.Run("Public_KiloCenterService", func(t *testing.T) {
		client.assertNotUnimplemented(t, publicKiloCenterServiceMethods)
	})

	t.Run("AuthRequired_CoreService_Unary", func(t *testing.T) {
		client.assertUnauthenticatedUnary(t, "/kilocenter.api.v1.CoreService/", coreServiceUnaryMethods)
	})

	t.Run("AuthRequired_CoreService_Streaming", func(t *testing.T) {
		client.assertUnauthenticatedStream(t, "/kilocenter.api.v1.CoreService/", coreServiceStreamMethods)
	})

	t.Run("AuthRequired_IdentityService", func(t *testing.T) {
		client.assertUnauthenticatedUnary(t, "/kilocenter.api.v1.IdentityService/", identityServiceUnaryMethods)
	})

	// KiloCenterService compat surface combines the core and identity unary sets.
	t.Run("AuthRequired_KiloCenterService_Unary", func(t *testing.T) {
		allUnary := make([]string, 0, len(coreServiceUnaryMethods)+len(identityServiceUnaryMethods))
		allUnary = append(allUnary, coreServiceUnaryMethods...)
		allUnary = append(allUnary, identityServiceUnaryMethods...)
		client.assertUnauthenticatedUnary(t, "/kilocenter.api.v1.KiloCenterService/", allUnary)
	})

	t.Run("AuthRequired_KiloCenterService_Streaming", func(t *testing.T) {
		client.assertUnauthenticatedStream(t, "/kilocenter.api.v1.KiloCenterService/", coreServiceStreamMethods)
	})

	// Success-path payload checks
	t.Run("SuccessPath_CoreService_GetReleaseInfo", func(t *testing.T) {
		var resp pb.ReleaseInfo
		err := conn.Invoke(ctx, "/kilocenter.api.v1.CoreService/GetReleaseInfo", empty, &resp)
		require.NoError(t, err, "GetReleaseInfo should succeed")
		assert.NotEmpty(t, resp.GetVersion(), "ReleaseInfo.Version should be non-empty")
	})

	t.Run("SuccessPath_IdentityService_GetAuthSettings", func(t *testing.T) {
		var resp pb.GetAuthSettingsResponse
		err := conn.Invoke(ctx, "/kilocenter.api.v1.IdentityService/GetAuthSettings",
			&pb.GetAuthSettingsRequest{}, &resp)
		require.NoError(t, err, "GetAuthSettings should succeed")
		assert.NotNil(t, resp.GetSettings(), "AuthSettings should be non-nil")
	})

	t.Run("SuccessPath_KiloCenterService_GetReleaseInfo", func(t *testing.T) {
		var resp pb.ReleaseInfo
		err := conn.Invoke(ctx, "/kilocenter.api.v1.KiloCenterService/GetReleaseInfo", empty, &resp)
		require.NoError(t, err, "GetReleaseInfo (compat) should succeed")
		assert.NotEmpty(t, resp.GetVersion(), "ReleaseInfo.Version should be non-empty")
	})
}
