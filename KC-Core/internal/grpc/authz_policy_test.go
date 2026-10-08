package grpc

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/grpcservices"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// matrixRole is one column of the authorization matrix.
type matrixRole struct {
	name  string
	roles authz.Roles
}

var matrixRoles = []matrixRole{
	{name: "none", roles: authz.Roles{}},
	{name: "base station manager", roles: authz.Roles{BaseStationManager: true}},
	{name: "endpoint manager", roles: authz.Roles{EndpointManager: true}},
	{name: "tenant manager", roles: authz.Roles{TenantManager: true}},
	{name: "admin", roles: authz.AllRoles},
	{name: "service account", roles: authz.ServiceAccountRoles},
}

// Who may call an RPC, as the product defines it. The test states the
// expectation independently of the policy table it verifies.
const (
	whoPublic      = "public"
	whoEndpoint    = "endpoint manager"
	whoBaseStation = "base station manager"
	whoAnyRole     = "any role"
	whoAdmin       = "admin"
)

var expectedCallers = map[string][]string{
	whoPublic: {"GetReleaseInfo", "GetCEStatus", "CompleteCEOnboarding"},
	whoEndpoint: {
		"CreateEndPoint", "GetEndPoint", "UpdateEndPoint", "DeleteEndPoint", "ListEndPoints",
		"AttachEndPoint", "DetachEndPoint", "GetEndPointStats", "GetEndPointOperations",
		"ListEndpointMessages", "ListEndpointActivity", "GetMessage", "ListMessages", "StreamMessages",
		"SendDownlink", "RevokeDownlink", "ListDownlinkQueue", "GetDownlinkResults", "UpdatePendingDownlink",
		"SendULTransmit", "GetDLRXStatus", "QueryDLRXStatus", "GetDLRXStatusQueries",
		"ListScaciSessions", "GetScaciSession", "GetScaciStatistics", "ListScaciErrors", "ListScaciQueues", "GetScaciStatus",
		"CreateManufacturer", "GetManufacturer", "UpdateManufacturer", "DeleteManufacturer", "ListManufacturers",
		"CreateDeviceModel", "GetDeviceModel", "UpdateDeviceModel", "DeleteDeviceModel", "ListDeviceModels",
		"CreateBlueprint", "GetBlueprint", "UpdateBlueprint", "DeleteBlueprint", "ListBlueprints",
		"SetDefaultBlueprint", "SubmitBlueprintToRegistry", "BulkAssignBlueprint", "CreateDeviceModelWithBlueprint",
		"DecodePreview",
	},
	whoBaseStation: {
		"CreateBaseStation", "GetBaseStation", "UpdateBaseStation", "DeleteBaseStation", "ListBaseStations",
		"GetBaseStationStats", "UpdateBaseStationEui", "GetBaseStationAvailability", "GetBaseStationMessagesReceived",
		"RequestBaseStationStatus", "InitiatePing", "ListBaseStationActivity",
		"ListBaseStationMessages", "GetBaseStationMessage", "GetBaseStationMessageStats",
		"SearchBaseStationMessages", "ExportBaseStationMessages", "StreamBaseStationMessages",
		"GenerateCertificate", "DownloadCertificate", "DownloadBaseStationCertificate", "GetServerCertificateStatus",
	},
	whoAnyRole: {
		"GetSystemStatus", "GetStatistics", "GetAnalyticsOverview", "GetActivityAnalytics", "GetSignalQualityAnalytics",
		"ListCapabilities", "ListEvents", "StreamEvents", "ListErrorGroups",
	},
	whoAdmin: {
		"GetDiagnosticsBundle", "ListAlerts", "GetAlertSummary",
		"GenerateServerCertificates", "RenewServerCertificates", "ListAllBaseStationLocations",
		"ListCEInstances", "RevokeCEInstance",
		"CreateIntegration", "GetIntegration", "UpdateIntegration", "DeleteIntegration", "ListIntegrations",
	},
}

func expectedCallerOf(method string) (string, bool) {
	for who, methods := range expectedCallers {
		for _, m := range methods {
			if m == method {
				return who, true
			}
		}
	}
	return "", false
}

func allowedFor(who string, roles authz.Roles) bool {
	switch who {
	case whoPublic:
		return true
	case whoEndpoint:
		return roles.Admin || roles.EndpointManager
	case whoBaseStation:
		return roles.Admin || roles.BaseStationManager
	case whoAnyRole:
		return roles.Any()
	default:
		return roles.Admin
	}
}

type fixedRoles authz.Roles

func (r fixedRoles) Roles(context.Context) (authz.Roles, error) { return authz.Roles(r), nil }

func matrixInterceptor(roles authz.Roles) *interceptors.AuthorizationInterceptor {
	return interceptors.NewAuthorizationInterceptor(fixedRoles(roles), NewMethodPolicy(), logger.NewNop())
}

// admitted runs one method through the authorization interceptor.
func admitted(t *testing.T, ai *interceptors.AuthorizationInterceptor, fullMethod string, stream bool) bool {
	t.Helper()
	ctx := testutil.TestContext()
	var err error
	if stream {
		err = ai.StreamInterceptor()(nil, matrixStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: fullMethod},
			func(interface{}, grpc.ServerStream) error { return nil })
	} else {
		_, err = ai.UnaryInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: fullMethod},
			func(context.Context, interface{}) (interface{}, error) { return nil, nil })
	}
	if err == nil {
		return true
	}
	require.Equal(t, codes.PermissionDenied, status.Code(err), "%s: %v", fullMethod, err)
	return false
}

type matrixStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s matrixStream) Context() context.Context { return s.ctx }

// TestAuthorizationMatrix admits or refuses every CoreService RPC, and every
// KiloCenterService twin KC-Core serves, for each role on its own.
func TestAuthorizationMatrix(t *testing.T) {
	core := pb.CoreService_ServiceDesc
	for _, service := range []grpc.ServiceDesc{core, pb.KiloCenterService_ServiceDesc} {
		for _, method := range serviceMethods(service) {
			who, served := expectedCallerOf(method.name)
			if service.ServiceName == core.ServiceName {
				require.True(t, served, "CoreService/%s has no expected caller in the matrix", method.name)
			}
			for _, role := range matrixRoles {
				fullMethod := fullMethodName(service, method.name)
				t.Run(fullMethod+"/"+role.name, func(t *testing.T) {
					got := admitted(t, matrixInterceptor(role.roles), fullMethod, method.stream)
					want := served && allowedFor(who, role.roles)
					if !served && grpcerrors.IsPublicMethod(fullMethod) {
						want = true
					}
					assert.Equal(t, want, got, "admitted")
				})
			}
		}
	}
}

type matrixMethod struct {
	name   string
	stream bool
}

func serviceMethods(desc grpc.ServiceDesc) []matrixMethod {
	methods := make([]matrixMethod, 0, len(desc.Methods)+len(desc.Streams))
	for _, m := range desc.Methods {
		methods = append(methods, matrixMethod{name: m.MethodName})
	}
	for _, s := range desc.Streams {
		methods = append(methods, matrixMethod{name: s.StreamName, stream: true})
	}
	return methods
}

// Key material and the cross-tenant boundary, through the interceptor and the real handler.

var revealNetworkKey = []pb.EndpointKey{pb.EndpointKey_ENDPOINT_KEY_NWK_SN_KEY}

var (
	matrixEndpointEUI = []byte{0x70, 0xB3, 0xD5, 0x67, 0x70, 0x11, 0x15, 0x05}
	matrixNwkSnKey    = []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	matrixBSKeyPEM    = []byte("station private key")
)

const (
	matrixOwnerTenant int64 = 1
	matrixOtherTenant int64 = 4
	matrixEndpointHex       = "70B3D56770111505"
	matrixStationHex        = "70B3D59CD00009E6"
	matrixKeyCertType       = "key"
	matrixKeyFilename       = "client.key"
)

func ownedEndpointSvc() *mockEndpointSvcIsolation {
	return &mockEndpointSvcIsolation{
		getByEUIFunc: func(_ context.Context, eui []byte, tenantID int64) (*models.EndPoint, error) {
			if tenantID != matrixOwnerTenant {
				return nil, storage.ErrNotFound
			}
			ep := &models.EndPoint{TenantID: tenantID, NwkSnKey: matrixNwkSnKey}
			copy(ep.EUI[:], eui)
			return ep, nil
		},
		listFunc: func(_ context.Context, tenantID int64, _, _ int) ([]*models.EndPoint, error) {
			if tenantID != matrixOwnerTenant {
				return nil, nil
			}
			ep := &models.EndPoint{TenantID: tenantID, NwkSnKey: matrixNwkSnKey}
			copy(ep.EUI[:], matrixEndpointEUI)
			return []*models.EndPoint{ep}, nil
		},
	}
}

// stationKeyStore hands out the private key of a station of the owner tenant.
type stationKeyStore struct {
	grpcservices.CertificateService
}

func (stationKeyStore) GetStoredCertificate(_ context.Context, tenantID int64, _ []byte, _ string) ([]byte, string, error) {
	if tenantID != matrixOwnerTenant {
		return nil, "", storage.ErrNotFound
	}
	return matrixBSKeyPEM, matrixKeyFilename, nil
}

func callThrough(ctx context.Context, roles authz.Roles, fullMethod string, handler grpc.UnaryHandler) (interface{}, error) {
	return matrixInterceptor(roles).UnaryInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: fullMethod}, handler)
}

func TestKeyMaterial_ReachesOnlyTheMatchingManager(t *testing.T) {
	svc := testCoreService(coreFields{endpointSvc: ownedEndpointSvc(), certSvc: stationKeyStore{}, log: &mockLogger{}})
	ctx := testutil.TestContextWithTenant(matrixOwnerTenant)

	for _, role := range matrixRoles {
		t.Run(role.name, func(t *testing.T) {
			ep, err := callThrough(ctx, role.roles, pb.CoreService_GetEndPoint_FullMethodName, func(ctx context.Context, _ interface{}) (interface{}, error) {
				return svc.GetEndPoint(ctx, &pb.GetEndPointRequest{EpEui: matrixEndpointHex, RevealKeys: revealNetworkKey})
			})
			if role.roles.Admin || role.roles.EndpointManager {
				require.NoError(t, err)
				assert.Equal(t, matrixNwkSnKey, ep.(*pb.EndPoint).GetNwkSnKey())
			} else {
				assert.Equal(t, codes.PermissionDenied, status.Code(err))
				assert.Nil(t, ep, "no endpoint, and so no network session key, is returned")
			}

			list, err := callThrough(ctx, role.roles, pb.CoreService_ListEndPoints_FullMethodName, func(ctx context.Context, _ interface{}) (interface{}, error) {
				return svc.ListEndPoints(ctx, &pb.ListEndPointsRequest{PageSize: testListPageSize})
			})
			if role.roles.Admin || role.roles.EndpointManager {
				require.NoError(t, err)
				require.Len(t, list.(*pb.ListEndPointsResponse).GetEndpoints(), 1)
				listed := list.(*pb.ListEndPointsResponse).GetEndpoints()[0]
				assert.Empty(t, listed.GetNwkSnKey(), "a listing never carries a key")
				assert.True(t, listed.GetNwkSnKeySet())
			} else {
				assert.Equal(t, codes.PermissionDenied, status.Code(err))
				assert.Nil(t, list)
			}

			key, err := callThrough(ctx, role.roles, pb.CoreService_DownloadBaseStationCertificate_FullMethodName, func(ctx context.Context, _ interface{}) (interface{}, error) {
				return svc.DownloadBaseStationCertificate(ctx, &pb.DownloadBaseStationCertificateRequest{BsEui: matrixStationHex, CertType: matrixKeyCertType})
			})
			if role.roles.Admin || role.roles.BaseStationManager {
				require.NoError(t, err)
				assert.Equal(t, matrixBSKeyPEM, key.(*pb.DownloadCertificateResponse).GetContent())
			} else {
				assert.Equal(t, codes.PermissionDenied, status.Code(err))
				assert.Nil(t, key, "no base station private key is returned")
			}
		})
	}
}

// TestCrossTenant_ResourceOfAnotherTenantIsNotFound: holding the endpoint
// manager role in tenant A reveals nothing of tenant B.
func TestCrossTenant_ResourceOfAnotherTenantIsNotFound(t *testing.T) {
	svc := testCoreService(coreFields{endpointSvc: ownedEndpointSvc(), certSvc: stationKeyStore{}, log: &mockLogger{}})
	ctx := testutil.TestContextWithTenant(matrixOtherTenant)
	manager := authz.Roles{EndpointManager: true, BaseStationManager: true}

	ep, err := callThrough(ctx, manager, pb.CoreService_GetEndPoint_FullMethodName, func(ctx context.Context, _ interface{}) (interface{}, error) {
		return svc.GetEndPoint(ctx, &pb.GetEndPointRequest{EpEui: matrixEndpointHex})
	})
	assert.Equal(t, codes.NotFound, status.Code(err), "%v", err)
	assert.Nil(t, ep)

	list, err := callThrough(ctx, manager, pb.CoreService_ListEndPoints_FullMethodName, func(ctx context.Context, _ interface{}) (interface{}, error) {
		return svc.ListEndPoints(ctx, &pb.ListEndPointsRequest{PageSize: testListPageSize})
	})
	require.NoError(t, err)
	assert.Empty(t, list.(*pb.ListEndPointsResponse).GetEndpoints())

	key, err := callThrough(ctx, manager, pb.CoreService_DownloadBaseStationCertificate_FullMethodName, func(ctx context.Context, _ interface{}) (interface{}, error) {
		return svc.DownloadBaseStationCertificate(ctx, &pb.DownloadBaseStationCertificateRequest{BsEui: matrixStationHex, CertType: matrixKeyCertType})
	})
	assert.Error(t, err)
	assert.Nil(t, key)
}

// serviceAccountSource answers roles the way KC-Identity does for a service-account key.
type serviceAccountSource struct{ key *models.APIKey }

func (s serviceAccountSource) Roles(ctx context.Context) (authz.Roles, error) {
	org, _ := pkgcontext.GetOrganizationID(ctx)
	return authz.ServiceAccount(s.key, org), nil
}

func TestServiceAccount_ActsOnlyInItsOwnOrganization(t *testing.T) {
	ownOrg, otherOrg := uuid.New(), uuid.New()
	key := &models.APIKey{ID: uuid.New(), OrgID: ownOrg, KeyType: models.KeyTypeServiceAccount, IsActive: true}
	ai := interceptors.NewAuthorizationInterceptor(serviceAccountSource{key: key}, NewMethodPolicy(), logger.NewNop())
	svc := testCoreService(coreFields{endpointSvc: ownedEndpointSvc(), certSvc: stationKeyStore{}, log: &mockLogger{}})
	inOrg := func(org uuid.UUID) context.Context {
		return pkgcontext.WithServiceAccountID(testutil.TestContextWithTenantAndOrg(matrixOwnerTenant, org), key.ID)
	}
	call := func(ctx context.Context, fullMethod string, handler grpc.UnaryHandler) (interface{}, error) {
		return ai.UnaryInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: fullMethod}, handler)
	}
	getEndpoint := func(ctx context.Context, _ interface{}) (interface{}, error) {
		return svc.GetEndPoint(ctx, &pb.GetEndPointRequest{EpEui: matrixEndpointHex, RevealKeys: revealNetworkKey})
	}
	pass := func(context.Context, interface{}) (interface{}, error) { return nil, nil }

	ep, err := call(inOrg(ownOrg), pb.CoreService_GetEndPoint_FullMethodName, getEndpoint)
	require.NoError(t, err, "the key reads its own organization's endpoints")
	assert.Equal(t, matrixNwkSnKey, ep.(*pb.EndPoint).GetNwkSnKey())

	for _, write := range []string{
		pb.CoreService_CreateEndPoint_FullMethodName, pb.CoreService_UpdateEndPoint_FullMethodName,
		pb.CoreService_CreateBaseStation_FullMethodName, pb.CoreService_UpdateBaseStation_FullMethodName,
		pb.CoreService_ListBaseStations_FullMethodName,
	} {
		_, err := call(inOrg(ownOrg), write, pass)
		assert.NoError(t, err, "%s in its own organization", write)
	}

	for _, admin := range []string{
		pb.CoreService_RenewServerCertificates_FullMethodName, pb.CoreService_ListIntegrations_FullMethodName,
		pb.CoreService_ListAllBaseStationLocations_FullMethodName, pb.CoreService_GetDiagnosticsBundle_FullMethodName,
	} {
		_, err := call(inOrg(ownOrg), admin, pass)
		assert.Equal(t, codes.PermissionDenied, status.Code(err), "%s is an administrator operation", admin)
	}

	ep, err = call(inOrg(otherOrg), pb.CoreService_GetEndPoint_FullMethodName, getEndpoint)
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "the key holds nothing in another organization")
	assert.Nil(t, ep)
	_, err = call(inOrg(otherOrg), pb.CoreService_CreateBaseStation_FullMethodName, pass)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

// stationUplinkStore holds one uplink heard by one station of the owner tenant.
type stationUplinkStore struct {
	mockMessageListingSvcIsolation
	askedTenant  int64
	askedStation []byte
	askedFilters *grpcservices.MessageFilters
}

func (s *stationUplinkStore) ListBaseStationMessages(_ context.Context, tenantID int64, bsEui []byte, filters *grpcservices.MessageFilters, _, _ int) ([]*mioty.ULDataMessage, int64, error) {
	s.askedTenant, s.askedStation, s.askedFilters = tenantID, bsEui, filters
	station, _ := hex.DecodeString(matrixStationHex)
	if tenantID != matrixOwnerTenant || !bytes.Equal(bsEui, station) {
		return nil, 0, nil
	}
	format := uint8(7)
	msg := &mioty.ULDataMessage{ID: "ul-1", TenantID: tenantID, BsEui: binary.BigEndian.Uint64(station)}
	msg.OpId, msg.DlOpen, msg.Format = 42, true, &format
	return []*mioty.ULDataMessage{msg}, 1, nil
}

// TestStationUplinks_StayWithTheCallersTenantAndStation: a base station
// manager reads a station's uplinks through ListBaseStationMessages, which
// lists only that station of the caller's own tenant and carries the SCACI
// ulData flags the Traffic uplink view shows.
func TestStationUplinks_StayWithTheCallersTenantAndStation(t *testing.T) {
	store := &stationUplinkStore{}
	svc := testCoreService(coreFields{msgListingSvc: store, log: &mockLogger{}})
	dlOpen := true
	list := func(ctx context.Context, roles authz.Roles, station string) (*pb.ListBaseStationMessagesResponse, error) {
		resp, err := callThrough(ctx, roles, pb.CoreService_ListBaseStationMessages_FullMethodName, func(ctx context.Context, _ interface{}) (interface{}, error) {
			return svc.ListBaseStationMessages(ctx, &pb.ListBaseStationMessagesRequest{BsEui: station, DlOpen: &dlOpen, PageSize: testListPageSize})
		})
		if err != nil {
			return nil, err
		}
		return resp.(*pb.ListBaseStationMessagesResponse), nil
	}
	manager := authz.Roles{BaseStationManager: true}

	resp, err := list(testutil.TestContextWithTenant(matrixOwnerTenant), manager, matrixStationHex)
	require.NoError(t, err)
	require.Len(t, resp.GetMessages(), 1)
	got := resp.GetMessages()[0]
	assert.True(t, got.GetDlOpen())
	assert.Equal(t, int64(42), got.GetOpId())
	assert.Equal(t, uint32(7), got.GetFormat())
	assert.Equal(t, matrixOwnerTenant, store.askedTenant)
	assert.Equal(t, matrixStationHex, strings.ToUpper(hex.EncodeToString(store.askedStation)))
	require.NotNil(t, store.askedFilters.DlOpen)
	assert.True(t, *store.askedFilters.DlOpen, "the uplink view's filters reach the station listing")

	resp, err = list(testutil.TestContextWithTenant(matrixOwnerTenant), manager, "70B3D59CD00009BB")
	require.NoError(t, err)
	assert.Empty(t, resp.GetMessages(), "another station's uplinks are not listed")

	resp, err = list(testutil.TestContextWithTenant(matrixOtherTenant), manager, matrixStationHex)
	require.NoError(t, err)
	assert.Empty(t, resp.GetMessages(), "the station of another tenant lists nothing")
	assert.Equal(t, matrixOtherTenant, store.askedTenant, "the tenant comes from the caller, not the request")

	_, err = list(testutil.TestContextWithTenant(matrixOwnerTenant), authz.Roles{EndpointManager: true}, matrixStationHex)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

// TestStationScopedTraffic_StaysWithEndpointManagers: narrowing the downlink
// queue or the uplinks to one base station does not open them to a
// base-station-only manager, since both carry endpoint payloads.
func TestStationScopedTraffic_StaysWithEndpointManagers(t *testing.T) {
	requests := map[string]interface{}{
		pb.CoreService_ListDownlinkQueue_FullMethodName: &pb.ListDownlinkQueueRequest{BsEui: matrixStationHex},
		pb.CoreService_ListMessages_FullMethodName:      &pb.ListMessagesRequest{BsEui: matrixStationHex},
	}
	for fullMethod, req := range requests {
		for _, role := range matrixRoles {
			t.Run(fullMethod+"/"+role.name, func(t *testing.T) {
				_, err := matrixInterceptor(role.roles).UnaryInterceptor()(testutil.TestContext(), req,
					&grpc.UnaryServerInfo{FullMethod: fullMethod},
					func(context.Context, interface{}) (interface{}, error) { return nil, nil })
				if role.roles.Admin || role.roles.EndpointManager {
					assert.NoError(t, err)
					return
				}
				assert.Equal(t, codes.PermissionDenied, status.Code(err))
			})
		}
	}
}
