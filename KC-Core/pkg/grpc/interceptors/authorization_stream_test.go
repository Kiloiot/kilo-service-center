package interceptors

import (
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type recordingServerStream struct {
	stubServerStream
	sent []interface{}
}

func (s *recordingServerStream) SendMsg(m interface{}) error {
	s.sent = append(s.sent, m)
	return nil
}

type streamPayload struct{ seq int }

func runStream(t *testing.T, ai *AuthorizationInterceptor, method string, change func()) (*recordingServerStream, error) {
	t.Helper()
	inner := &recordingServerStream{stubServerStream: stubServerStream{ctx: testutil.TestContext()}}
	err := ai.StreamInterceptor()(nil, inner, &grpc.StreamServerInfo{FullMethod: method},
		func(_ interface{}, ss grpc.ServerStream) error {
			if err := ss.SendMsg(streamPayload{seq: 1}); err != nil {
				t.Fatalf("first send refused before any role change: %v", err)
			}
			change()
			return ss.SendMsg(streamPayload{seq: 2})
		})
	return inner, err
}

func requireOnlyFirstForwarded(t *testing.T, inner *recordingServerStream) {
	t.Helper()
	if len(inner.sent) != 1 || inner.sent[0] != (streamPayload{seq: 1}) {
		t.Fatalf("forwarded %v, want only the payload sent before the change", inner.sent)
	}
}

func TestAuthorizationStream_ForwardsWhileRolesAreUnchanged(t *testing.T) {
	ai := NewAuthorizationInterceptor(&stubRoleSource{roles: authz.Roles{EndpointManager: true}},
		stubPolicy{testEndpointMethod: authz.EndpointManager}, noopLogger{})

	inner, err := runStream(t, ai, testEndpointMethod, func() {})
	if err != nil {
		t.Fatalf("stream with unchanged roles refused: %v", err)
	}
	if len(inner.sent) != 2 {
		t.Fatalf("forwarded %d payloads, want 2", len(inner.sent))
	}
}

func TestAuthorizationStream_RemovedRoleStopsTheNextPayload(t *testing.T) {
	source := &stubRoleSource{roles: authz.Roles{EndpointManager: true}}
	events := &recordingEventWriter{}
	ai := NewAuthorizationInterceptor(source, stubPolicy{testEndpointMethod: authz.EndpointManager}, noopLogger{}).
		WithEventWriter(events)

	inner, err := runStream(t, ai, testEndpointMethod, func() { source.roles = authz.Roles{} })

	requireCode(t, err, codes.PermissionDenied)
	if status.Convert(err).Message() != grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInsufficientRole) {
		t.Fatalf("message = %q, want the catalog message", status.Convert(err).Message())
	}
	requireOnlyFirstForwarded(t, inner)
	if len(events.events) != 1 {
		t.Fatalf("recorded %d permission-denied events, want 1", len(events.events))
	}
}

func TestAuthorizationStream_ChangedRolesEndTheStreamEvenWhenStillGranted(t *testing.T) {
	source := &stubRoleSource{roles: authz.AllRoles}
	ai := NewAuthorizationInterceptor(source, stubPolicy{testEndpointMethod: authz.AnyRole}, noopLogger{})

	inner, err := runStream(t, ai, testEndpointMethod, func() { source.roles = authz.Roles{EndpointManager: true} })

	requireCode(t, err, codes.PermissionDenied)
	requireOnlyFirstForwarded(t, inner)
}

func TestAuthorizationStream_ResolutionFailureStopsTheNextPayload(t *testing.T) {
	source := &stubRoleSource{roles: authz.Roles{EndpointManager: true}}
	ai := NewAuthorizationInterceptor(source, stubPolicy{testEndpointMethod: authz.EndpointManager}, noopLogger{})

	inner, err := runStream(t, ai, testEndpointMethod, func() { source.err = errTestIdentityDown })

	requireCode(t, err, codes.Internal)
	requireOnlyFirstForwarded(t, inner)
}

func TestAuthorizationStream_PublicMethodIsNeverRechecked(t *testing.T) {
	source := &stubRoleSource{}
	ai := NewAuthorizationInterceptor(source, stubPolicy{}, noopLogger{})

	inner, err := runStream(t, ai, testPublicMethod, func() { source.err = errTestIdentityDown })
	if err != nil {
		t.Fatalf("public stream refused: %v", err)
	}
	if len(inner.sent) != 2 || source.calls.Load() != 0 {
		t.Fatalf("public stream forwarded %d payloads after %d role lookups, want 2 after none",
			len(inner.sent), source.calls.Load())
	}
}
