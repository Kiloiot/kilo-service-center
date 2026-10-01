package grpc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc/interceptors"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
)

// streamRoleCacheTTL is short enough to wait out, long enough for the sends right after a change to hit the cache.
const streamRoleCacheTTL = 200 * time.Millisecond

type changingIdentityServer struct {
	pb.UnimplementedIdentityInternalServiceServer
	mu      sync.Mutex
	answers map[string]identityAnswer
}

type identityAnswer struct {
	roles *pb.UserRoles
	err   error
}

func principalKey(org, principal string) string { return org + roleCacheKeySeparator + principal }

func (m *changingIdentityServer) set(org, principal string, answer identityAnswer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.answers[principalKey(org, principal)] = answer
}

func (m *changingIdentityServer) GetUserRoles(_ context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	principal := req.GetUserId()
	if principal == "" {
		principal = req.GetServiceAccountId()
	}
	m.mu.Lock()
	answer := m.answers[principalKey(req.GetOrgId(), principal)]
	m.mu.Unlock()
	if answer.err != nil {
		return nil, answer.err
	}
	return &pb.GetUserRolesResponse{Roles: answer.roles}, nil
}

type capturingServerStream struct {
	grpc.ServerStream
	ctx  context.Context
	sent int
}

func (s *capturingServerStream) Context() context.Context { return s.ctx }

func (s *capturingServerStream) SendMsg(interface{}) error {
	s.sent++
	return nil
}

func openAuthorizedStream(ctx context.Context, ai *interceptors.AuthorizationInterceptor, method string) (sends chan<- struct{}, results <-chan error, inner *capturingServerStream) {
	inner = &capturingServerStream{ctx: ctx}
	sendCh, resultCh, ended := make(chan struct{}), make(chan error), make(chan error, 1)
	go func() {
		ended <- ai.StreamInterceptor()(nil, inner, &grpc.StreamServerInfo{FullMethod: method},
			func(_ interface{}, ss grpc.ServerStream) error {
				for range sendCh {
					err := ss.SendMsg(&pb.Event{})
					resultCh <- err
					if err != nil {
						return err
					}
				}
				return nil
			})
		close(resultCh)
	}()
	return sendCh, resultCh, inner
}

func sendOne(sends chan<- struct{}, results <-chan error) error {
	sends <- struct{}{}
	return <-results
}

type streamRevocationFixture struct {
	identity *changingIdentityServer
	ai       *interceptors.AuthorizationInterceptor
}

func newStreamRevocationFixture(t *testing.T) streamRevocationFixture {
	t.Helper()
	identity := &changingIdentityServer{answers: map[string]identityAnswer{}}
	source := NewIdentityRoleSource(newIdentityClient(t, identity), "", streamRoleCacheTTL)
	return streamRevocationFixture{
		identity: identity,
		ai:       interceptors.NewAuthorizationInterceptor(source, NewMethodPolicy(), logger.NewNop()),
	}
}

func TestStreamRevocation_TheNextPayloadAfterTheCacheExpiresIsNotForwarded(t *testing.T) {
	endpointManager := &pb.UserRoles{EndpointManager: true}
	notFound := status.Error(codes.NotFound, grpcconst.ResolveErrorMessage(grpcconst.ErrTokenApiKeyNotFound))
	unavailable := status.Error(codes.Unavailable, grpcconst.ResolveErrorMessage(grpcconst.ErrTokenServiceNotConfigured))
	cases := []struct {
		name        string
		method      string
		initial     *pb.UserRoles
		after       identityAnswer
		serviceAcct bool
		wantCode    codes.Code
	}{
		{name: "administrator becomes endpoint manager", method: pb.CoreService_StreamEvents_FullMethodName,
			initial: &pb.UserRoles{Admin: true, TenantManager: true, BaseStationManager: true, EndpointManager: true},
			after:   identityAnswer{roles: endpointManager}, wantCode: codes.PermissionDenied},
		{name: "endpoint manager role removed", method: pb.CoreService_StreamMessages_FullMethodName,
			initial: endpointManager, after: identityAnswer{roles: &pb.UserRoles{}}, wantCode: codes.PermissionDenied},
		{name: "user deactivated", method: pb.CoreService_StreamMessages_FullMethodName,
			initial: endpointManager, after: identityAnswer{roles: &pb.UserRoles{}}, wantCode: codes.PermissionDenied},
		{name: "service-account key revoked", method: pb.CoreService_StreamMessages_FullMethodName, serviceAcct: true,
			initial: &pb.UserRoles{BaseStationManager: true, EndpointManager: true},
			after:   identityAnswer{err: notFound}, wantCode: codes.PermissionDenied},
		{name: "role resolution fails", method: pb.CoreService_StreamMessages_FullMethodName,
			initial: endpointManager, after: identityAnswer{err: unavailable}, wantCode: codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newStreamRevocationFixture(t)
			org, principal := uuid.New(), uuid.New()
			ctx := callerContext(org, principal.String())
			if tc.serviceAcct {
				ctx = serviceAccountContext(org, principal)
			}
			f.identity.set(org.String(), principal.String(), identityAnswer{roles: tc.initial})

			sends, results, inner := openAuthorizedStream(ctx, f.ai, tc.method)
			defer close(sends)
			require.NoError(t, sendOne(sends, results))

			f.identity.set(org.String(), principal.String(), tc.after)
			require.NoError(t, sendOne(sends, results), "within the cache TTL the stream keeps its roles")
			time.Sleep(streamRoleCacheTTL + streamRoleCacheTTL/2)

			err := sendOne(sends, results)
			assert.Equal(t, tc.wantCode, status.Code(err), "%v", err)
			assert.Equal(t, 2, inner.sent, "the payload after the change applied must not be forwarded")
		})
	}
}

func TestStreamRevocation_APrincipalInAnotherOrganizationKeepsItsStream(t *testing.T) {
	f := newStreamRevocationFixture(t)
	revokedOrg, revoked := uuid.New(), uuid.New().String()
	otherOrg, other := uuid.New(), uuid.New().String()
	f.identity.set(revokedOrg.String(), revoked, identityAnswer{roles: &pb.UserRoles{EndpointManager: true}})
	f.identity.set(otherOrg.String(), other, identityAnswer{roles: &pb.UserRoles{EndpointManager: true}})

	revokedSends, revokedResults, _ := openAuthorizedStream(callerContext(revokedOrg, revoked), f.ai, pb.CoreService_StreamMessages_FullMethodName)
	defer close(revokedSends)
	otherSends, otherResults, otherInner := openAuthorizedStream(callerContext(otherOrg, other), f.ai, pb.CoreService_StreamMessages_FullMethodName)
	defer close(otherSends)
	require.NoError(t, sendOne(revokedSends, revokedResults))
	require.NoError(t, sendOne(otherSends, otherResults))

	f.identity.set(revokedOrg.String(), revoked, identityAnswer{roles: &pb.UserRoles{}})
	time.Sleep(streamRoleCacheTTL + streamRoleCacheTTL/2)

	assert.Equal(t, codes.PermissionDenied, status.Code(sendOne(revokedSends, revokedResults)))
	require.NoError(t, sendOne(otherSends, otherResults))
	assert.Equal(t, 2, otherInner.sent, "the unaffected principal's stream keeps forwarding")
}
