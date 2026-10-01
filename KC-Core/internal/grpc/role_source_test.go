package grpc

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
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
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

const (
	roleSourceTestSecret = "test-secret"
	roleSourceTestTTL    = time.Minute
)

type mockIdentityServer struct {
	pb.UnimplementedIdentityInternalServiceServer
	roles            *pb.UserRoles
	err              error
	calls            atomic.Int32
	receivedMetadata metadata.MD
	receivedRequest  *pb.GetUserRolesRequest
}

func (m *mockIdentityServer) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	m.calls.Add(1)
	m.receivedMetadata, _ = metadata.FromIncomingContext(ctx)
	m.receivedRequest = req
	if m.err != nil {
		return nil, m.err
	}
	return &pb.GetUserRolesResponse{Roles: m.roles}, nil
}

func newIdentityClient(t *testing.T, mock pb.IdentityInternalServiceServer) pb.IdentityInternalServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	pb.RegisterIdentityInternalServiceServer(srv, mock)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.GracefulStop)

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return pb.NewIdentityInternalServiceClient(conn)
}

func callerContext(orgID uuid.UUID, userID string) context.Context {
	return pkgcontext.WithUserID(pkgcontext.WithOrganizationID(testutil.TestContext(), orgID), userID)
}

func TestIdentityRoleSource_ResolvesTheCallersRolesWithThePeerSecret(t *testing.T) {
	mock := &mockIdentityServer{roles: &pb.UserRoles{BaseStationManager: true}}
	source := NewIdentityRoleSource(newIdentityClient(t, mock), roleSourceTestSecret, roleSourceTestTTL)
	org, user := uuid.New(), uuid.New().String()

	roles, err := source.Roles(callerContext(org, user))

	require.NoError(t, err)
	assert.Equal(t, authz.Roles{BaseStationManager: true}, roles)
	assert.Equal(t, org.String(), mock.receivedRequest.GetOrgId())
	assert.Equal(t, user, mock.receivedRequest.GetUserId())
	assert.Equal(t, []string{roleSourceTestSecret}, mock.receivedMetadata[grpcconst.MetadataKeyInternalPeerSecret])
}

func TestIdentityRoleSource_CachesPerOrganizationAndUser(t *testing.T) {
	mock := &mockIdentityServer{roles: &pb.UserRoles{EndpointManager: true}}
	source := NewIdentityRoleSource(newIdentityClient(t, mock), "", roleSourceTestTTL)
	org, user := uuid.New(), uuid.New().String()

	for range 3 {
		_, err := source.Roles(callerContext(org, user))
		require.NoError(t, err)
	}
	assert.Equal(t, int32(1), mock.calls.Load(), "repeat calls within the TTL reuse the resolved roles")

	_, err := source.Roles(callerContext(uuid.New(), user))
	require.NoError(t, err)
	assert.Equal(t, int32(2), mock.calls.Load(), "another organization resolves again")
}

func TestIdentityRoleSource_ExpiredEntryResolvesAgain(t *testing.T) {
	mock := &mockIdentityServer{roles: &pb.UserRoles{}}
	source := NewIdentityRoleSource(newIdentityClient(t, mock), "", time.Nanosecond)
	ctx := callerContext(uuid.New(), uuid.New().String())

	_, err := source.Roles(ctx)
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	_, err = source.Roles(ctx)
	require.NoError(t, err)
	assert.Equal(t, int32(2), mock.calls.Load(), "a role change applies once the cached entry expires")
}

func TestIdentityRoleSource_UnknownUserHoldsNoRoles(t *testing.T) {
	mock := &mockIdentityServer{err: status.Error(codes.NotFound, grpcconst.ResolveErrorMessage(grpcconst.ErrTokenUserNotFound))}
	source := NewIdentityRoleSource(newIdentityClient(t, mock), "", roleSourceTestTTL)

	roles, err := source.Roles(callerContext(uuid.New(), uuid.New().String()))

	require.NoError(t, err)
	assert.False(t, roles.Any())
}

func TestIdentityRoleSource_FailuresCarryCatalogTokens(t *testing.T) {
	mock := &mockIdentityServer{err: status.Error(codes.Unavailable, grpcconst.ResolveErrorMessage(grpcconst.ErrTokenServiceNotConfigured))}
	source := NewIdentityRoleSource(newIdentityClient(t, mock), "", roleSourceTestTTL)

	cases := []struct {
		name  string
		ctx   context.Context
		token string
	}{
		{name: "identity unavailable", ctx: callerContext(uuid.New(), uuid.New().String()), token: grpcconst.ErrTokenInternalError},
		{name: "no user", ctx: pkgcontext.WithOrganizationID(testutil.TestContext(), uuid.New()), token: grpcconst.ErrTokenMissingUserCtx},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := source.Roles(tc.ctx)
			var te *grpcconst.TokenError
			require.True(t, errors.As(err, &te), "%v", err)
			assert.Equal(t, tc.token, te.Token)
		})
	}
}

func TestIdentityRoleSource_WithoutAnOrganizationAsksForTheOrganizationIndependentRoles(t *testing.T) {
	mock := &mockIdentityServer{roles: &pb.UserRoles{Admin: true}}
	source := NewIdentityRoleSource(newIdentityClient(t, mock), "", roleSourceTestTTL)
	user := uuid.New().String()

	roles, err := source.Roles(pkgcontext.WithUserID(testutil.TestContext(), user))

	require.NoError(t, err)
	assert.True(t, roles.Admin)
	assert.Empty(t, mock.receivedRequest.GetOrgId())
	assert.Equal(t, user, mock.receivedRequest.GetUserId())

	_, err = source.Roles(callerContext(uuid.New(), user))
	require.NoError(t, err)
	assert.Equal(t, int32(2), mock.calls.Load(), "roles without an organization are cached apart from any organization's")
}

func serviceAccountContext(orgID, keyID uuid.UUID) context.Context {
	return pkgcontext.WithServiceAccountID(pkgcontext.WithOrganizationID(testutil.TestContext(), orgID), keyID)
}

func TestIdentityRoleSource_AServiceAccountKeyIsResolvedAsItself(t *testing.T) {
	mock := &mockIdentityServer{roles: &pb.UserRoles{BaseStationManager: true, EndpointManager: true}}
	source := NewIdentityRoleSource(newIdentityClient(t, mock), "", roleSourceTestTTL)
	org, key := uuid.New(), uuid.New()

	roles, err := source.Roles(serviceAccountContext(org, key))

	require.NoError(t, err)
	assert.Equal(t, authz.ServiceAccountRoles, roles)
	assert.Equal(t, key.String(), mock.receivedRequest.GetServiceAccountId())
	assert.Empty(t, mock.receivedRequest.GetUserId())
	assert.Equal(t, org.String(), mock.receivedRequest.GetOrgId())
}

func TestIdentityRoleSource_UserAndServiceAccountAnswersAreCachedApart(t *testing.T) {
	mock := &mockIdentityServer{roles: &pb.UserRoles{EndpointManager: true}}
	source := NewIdentityRoleSource(newIdentityClient(t, mock), "", roleSourceTestTTL)
	org, id := uuid.New(), uuid.New()

	_, err := source.Roles(callerContext(org, id.String()))
	require.NoError(t, err)
	_, err = source.Roles(serviceAccountContext(org, id))
	require.NoError(t, err)

	assert.Equal(t, int32(2), mock.calls.Load(), "a key never reuses a user's cached roles")
}

// TestOrgExemptMethods_AreAuthorizedWithoutAnOrganization: every org-exempt
// method KC-Core authorizes is judged on the caller's organization-independent
// roles, so the exempt list and the role policy agree.
func TestOrgExemptMethods_AreAuthorizedWithoutAnOrganization(t *testing.T) {
	mock := &mockIdentityServer{roles: &pb.UserRoles{Admin: true}}
	source := NewIdentityRoleSource(newIdentityClient(t, mock), "", roleSourceTestTTL)
	policy := NewMethodPolicy()
	ai := interceptors.NewAuthorizationInterceptor(source, policy, logger.NewNop())
	ctx := pkgcontext.WithUserID(testutil.TestContext(), uuid.New().String())

	checked := 0
	for _, method := range grpcconst.OrgExemptMethodList() {
		if _, governed := policy.Requirement(method); !governed || grpcconst.IsPublicMethod(method) {
			continue
		}
		checked++
		_, err := ai.UnaryInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: method},
			func(context.Context, interface{}) (interface{}, error) { return nil, nil })
		require.NoError(t, err, "%s is org-exempt, so an administrator reaches it without an organization", method)
		assert.Empty(t, mock.receivedRequest.GetOrgId())
	}
	assert.Positive(t, checked, "GetSystemStatus and the admin-only coverage call are org-exempt and governed")
}
