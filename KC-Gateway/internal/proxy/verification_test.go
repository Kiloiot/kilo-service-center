//go:build integration

package proxy

import (
	"context"
	"os"
	"testing"
	"time"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	grpcconst "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// gatewayAddr returns the gateway address from env or default.
func gatewayAddr() string {
	if addr := os.Getenv("GATEWAY_ADDR"); addr != "" {
		return addr
	}
	return "localhost:9090"
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
	conn, err := grpc.NewClient(gatewayAddr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	empty := &emptypb.Empty{}

	// 1. IdentityInternalService should be blocked (Unimplemented or Unauthenticated)
	// Auth interceptor may reject unauthenticated callers before the service
	// routing layer, so either code means the method is effectively blocked.
	t.Run("IdentityInternalService_Blocked", func(t *testing.T) {
		blocked := map[codes.Code]bool{codes.Unimplemented: true, codes.Unauthenticated: true}
		for _, method := range identityInternalServiceMethods {
			t.Run(method, func(t *testing.T) {
				var resp emptypb.Empty
				err := conn.Invoke(ctx, method, empty, &resp)
				st, ok := status.FromError(err)
				require.True(t, ok, "expected gRPC status error")
				assert.True(t, blocked[st.Code()],
					"IdentityInternalService method %s should be blocked (Unimplemented or Unauthenticated), got %s: %s",
					method, st.Code(), st.Message())
			})
		}
	})

	// 2. Public CoreService RPCs — should NOT be Unimplemented
	t.Run("Public_CoreService", func(t *testing.T) {
		for _, method := range publicCoreServiceMethods {
			t.Run(method, func(t *testing.T) {
				var resp emptypb.Empty
				err := conn.Invoke(ctx, method, empty, &resp)
				if err == nil {
					return // OK — public method succeeded
				}
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.NotEqual(t, codes.Unimplemented, st.Code(),
					"public method %s should not return Unimplemented", method)
			})
		}
	})

	// 3. Public IdentityService RPCs — should NOT be Unimplemented
	t.Run("Public_IdentityService", func(t *testing.T) {
		for _, method := range publicIdentityServiceMethods {
			t.Run(method, func(t *testing.T) {
				var resp emptypb.Empty
				err := conn.Invoke(ctx, method, empty, &resp)
				if err == nil {
					return
				}
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.NotEqual(t, codes.Unimplemented, st.Code(),
					"public method %s should not return Unimplemented", method)
			})
		}
	})

	// 4. Public KiloCenterService compat RPCs — should NOT be Unimplemented
	t.Run("Public_KiloCenterService", func(t *testing.T) {
		for _, method := range publicKiloCenterServiceMethods {
			t.Run(method, func(t *testing.T) {
				var resp emptypb.Empty
				err := conn.Invoke(ctx, method, empty, &resp)
				if err == nil {
					return
				}
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.NotEqual(t, codes.Unimplemented, st.Code(),
					"public method %s should not return Unimplemented", method)
			})
		}
	})

	// 5. Auth-required CoreService unary RPCs — should return Unauthenticated
	t.Run("AuthRequired_CoreService_Unary", func(t *testing.T) {
		for _, method := range coreServiceUnaryMethods {
			fullMethod := "/kilocenter.api.v1.CoreService/" + method
			if grpcconst.PublicMethods[fullMethod] {
				continue // Skip public methods
			}
			t.Run(method, func(t *testing.T) {
				var resp emptypb.Empty
				err := conn.Invoke(ctx, fullMethod, empty, &resp)
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.Unauthenticated, st.Code(),
					"auth-required method %s should return Unauthenticated, got %s: %s",
					fullMethod, st.Code(), st.Message())
			})
		}
	})

	// 6. Auth-required CoreService streaming RPCs — should reject stream open
	t.Run("AuthRequired_CoreService_Streaming", func(t *testing.T) {
		for _, method := range coreServiceStreamMethods {
			fullMethod := "/kilocenter.api.v1.CoreService/" + method
			t.Run(method, func(t *testing.T) {
				stream, err := conn.NewStream(ctx,
					&grpc.StreamDesc{ServerStreams: true},
					fullMethod)
				if err != nil {
					st, ok := status.FromError(err)
					require.True(t, ok)
					assert.Equal(t, codes.Unauthenticated, st.Code(),
						"streaming method %s should return Unauthenticated", fullMethod)
					return
				}
				// Stream opened — try to receive; auth rejection comes here
				err = stream.RecvMsg(empty)
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.Unauthenticated, st.Code(),
					"streaming method %s should reject with Unauthenticated, got %s",
					fullMethod, st.Code())
			})
		}
	})

	// 7. Auth-required IdentityService RPCs — should return Unauthenticated
	t.Run("AuthRequired_IdentityService", func(t *testing.T) {
		for _, method := range identityServiceUnaryMethods {
			fullMethod := "/kilocenter.api.v1.IdentityService/" + method
			if grpcconst.PublicMethods[fullMethod] {
				continue
			}
			t.Run(method, func(t *testing.T) {
				var resp emptypb.Empty
				err := conn.Invoke(ctx, fullMethod, empty, &resp)
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.Unauthenticated, st.Code(),
					"auth-required method %s should return Unauthenticated, got %s: %s",
					fullMethod, st.Code(), st.Message())
			})
		}
	})

	// 8. Auth-required KiloCenterService unary RPCs — combined core + identity
	t.Run("AuthRequired_KiloCenterService_Unary", func(t *testing.T) {
		allUnary := make([]string, 0, len(coreServiceUnaryMethods)+len(identityServiceUnaryMethods))
		allUnary = append(allUnary, coreServiceUnaryMethods...)
		allUnary = append(allUnary, identityServiceUnaryMethods...)
		for _, method := range allUnary {
			fullMethod := "/kilocenter.api.v1.KiloCenterService/" + method
			if grpcconst.PublicMethods[fullMethod] {
				continue
			}
			t.Run(method, func(t *testing.T) {
				var resp emptypb.Empty
				err := conn.Invoke(ctx, fullMethod, empty, &resp)
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.Unauthenticated, st.Code(),
					"auth-required method %s should return Unauthenticated, got %s: %s",
					fullMethod, st.Code(), st.Message())
			})
		}
	})

	// 9. Auth-required KiloCenterService streaming RPCs
	t.Run("AuthRequired_KiloCenterService_Streaming", func(t *testing.T) {
		for _, method := range coreServiceStreamMethods {
			fullMethod := "/kilocenter.api.v1.KiloCenterService/" + method
			t.Run(method, func(t *testing.T) {
				stream, err := conn.NewStream(ctx,
					&grpc.StreamDesc{ServerStreams: true},
					fullMethod)
				if err != nil {
					st, ok := status.FromError(err)
					require.True(t, ok)
					assert.Equal(t, codes.Unauthenticated, st.Code(),
						"streaming method %s should return Unauthenticated", fullMethod)
					return
				}
				err = stream.RecvMsg(empty)
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.Unauthenticated, st.Code(),
					"streaming method %s should reject with Unauthenticated, got %s",
					fullMethod, st.Code())
			})
		}
	})

	// 10. Success-path payload checks
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
