package interceptors

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/authz"
	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubRoleSource struct {
	roles authz.Roles
	err   error
	calls atomic.Int32
}

func (s *stubRoleSource) Roles(context.Context) (authz.Roles, error) {
	s.calls.Add(1)
	return s.roles, s.err
}

type stubPolicy map[string]authz.Requirement

func (p stubPolicy) Requirement(method string) (authz.Requirement, bool) {
	requirement, ok := p[method]
	return requirement, ok
}

// noopLogger satisfies the logger.Logger interface without producing output.
type noopLogger struct{}

func (noopLogger) Debug(_ string, _ ...interface{})                           {}
func (noopLogger) Info(_ string, _ ...interface{})                            {}
func (noopLogger) Warn(_ string, _ ...interface{})                            {}
func (noopLogger) Error(_ string, _ ...interface{})                           {}
func (noopLogger) Fatal(_ string, _ ...interface{})                           {}
func (noopLogger) DebugContext(_ context.Context, _ string, _ ...interface{}) {}
func (noopLogger) InfoContext(_ context.Context, _ string, _ ...interface{})  {}
func (noopLogger) WarnContext(_ context.Context, _ string, _ ...interface{})  {}
func (noopLogger) ErrorContext(_ context.Context, _ string, _ ...interface{}) {}
func (noopLogger) FatalContext(_ context.Context, _ string, _ ...interface{}) {}
func (l noopLogger) WithField(_ string, _ interface{}) logger.Logger          { return l }
func (l noopLogger) WithFields(_ map[string]interface{}) logger.Logger        { return l }

const (
	testEndpointMethod = "/kilocenter.api.v1.CoreService/ListEndPoints"
	testUnknownMethod  = "/kilocenter.api.v1.CoreService/UnknownMethod"
	testPublicMethod   = "/kilocenter.api.v1.CoreService/GetReleaseInfo"
)

var errTestIdentityDown = errors.New("identity unavailable")

func invokeUnary(t *testing.T, ai *AuthorizationInterceptor, method string) (authz.Roles, error) {
	t.Helper()
	var seen authz.Roles
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		seen = authz.FromContext(ctx)
		return nil, nil
	}
	_, err := ai.UnaryInterceptor()(testutil.TestContext(), nil, &grpc.UnaryServerInfo{FullMethod: method}, handler)
	return seen, err
}

func requireCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if status.Code(err) != want {
		t.Fatalf("code = %v (%v), want %v", status.Code(err), err, want)
	}
}

func TestAuthorization_GrantedCallCarriesRoles(t *testing.T) {
	roles := authz.Roles{EndpointManager: true}
	ai := NewAuthorizationInterceptor(&stubRoleSource{roles: roles}, stubPolicy{testEndpointMethod: authz.EndpointManager}, noopLogger{})

	seen, err := invokeUnary(t, ai, testEndpointMethod)
	if err != nil {
		t.Fatalf("granted call refused: %v", err)
	}
	if seen != roles {
		t.Fatalf("handler saw roles %+v, want %+v", seen, roles)
	}
}

func TestAuthorization_InsufficientRoleDeniedAndRecorded(t *testing.T) {
	events := &recordingEventWriter{}
	ai := NewAuthorizationInterceptor(&stubRoleSource{roles: authz.Roles{BaseStationManager: true}},
		stubPolicy{testEndpointMethod: authz.EndpointManager}, noopLogger{}).WithEventWriter(events)

	_, err := invokeUnary(t, ai, testEndpointMethod)
	requireCode(t, err, codes.PermissionDenied)
	if status.Convert(err).Message() != grpcerrors.ResolveErrorMessage(grpcerrors.ErrTokenInsufficientRole) {
		t.Fatalf("message = %q, want the catalog message", status.Convert(err).Message())
	}
	if got := len(events.events); got != 1 {
		t.Fatalf("recorded %d permission-denied events, want 1", got)
	}
}

func TestAuthorization_UnknownMethodRefusedWithoutResolvingRoles(t *testing.T) {
	source := &stubRoleSource{roles: authz.AllRoles}
	ai := NewAuthorizationInterceptor(source, stubPolicy{}, noopLogger{})

	_, err := invokeUnary(t, ai, testUnknownMethod)
	requireCode(t, err, codes.PermissionDenied)
	if source.calls.Load() != 0 {
		t.Fatal("roles were resolved for a method the policy does not know")
	}
}

func TestAuthorization_PublicMethodSkipsRoleResolution(t *testing.T) {
	source := &stubRoleSource{}
	ai := NewAuthorizationInterceptor(source, stubPolicy{}, noopLogger{})

	if _, err := invokeUnary(t, ai, testPublicMethod); err != nil {
		t.Fatalf("public method refused: %v", err)
	}
	if source.calls.Load() != 0 {
		t.Fatal("roles were resolved for a public method")
	}
}

func TestAuthorization_ResolutionFailureFailsClosed(t *testing.T) {
	ai := NewAuthorizationInterceptor(&stubRoleSource{err: errTestIdentityDown},
		stubPolicy{testEndpointMethod: authz.AnyRole}, noopLogger{})

	_, err := invokeUnary(t, ai, testEndpointMethod)
	requireCode(t, err, codes.Internal)
}

func TestAuthorization_ResolutionTokenErrorKeepsItsCode(t *testing.T) {
	missingUser := grpcerrors.NewTokenError(grpcerrors.ErrTokenMissingUserCtx, nil)
	ai := NewAuthorizationInterceptor(&stubRoleSource{err: missingUser},
		stubPolicy{testEndpointMethod: authz.AnyRole}, noopLogger{})

	_, err := invokeUnary(t, ai, testEndpointMethod)
	requireCode(t, err, grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingUserCtx))
}

type stubServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s stubServerStream) Context() context.Context { return s.ctx }

func TestAuthorization_StreamCarriesRoles(t *testing.T) {
	roles := authz.Roles{BaseStationManager: true}
	ai := NewAuthorizationInterceptor(&stubRoleSource{roles: roles}, stubPolicy{testEndpointMethod: authz.AnyManager}, noopLogger{})

	var seen authz.Roles
	err := ai.StreamInterceptor()(nil, stubServerStream{ctx: testutil.TestContext()},
		&grpc.StreamServerInfo{FullMethod: testEndpointMethod},
		func(_ interface{}, ss grpc.ServerStream) error {
			seen = authz.FromContext(ss.Context())
			return nil
		})
	if err != nil {
		t.Fatalf("granted stream refused: %v", err)
	}
	if seen != roles {
		t.Fatalf("stream handler saw roles %+v, want %+v", seen, roles)
	}
}
