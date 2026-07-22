package interceptors

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// mockRoleResolver records calls and returns preconfigured results.
type mockRoleResolver struct {
	role      string
	active    bool
	err       error
	callCount atomic.Int32
}

func (m *mockRoleResolver) GetUserRole(_ context.Context, _, _ string) (string, bool, error) {
	m.callCount.Add(1)
	return m.role, m.active, m.err
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

const testMethod = "/kilocenter.api.v1.CoreService/TestMethod"

// newTestContext returns a context populated with a random org UUID and user ID.
func newTestContext() context.Context {
	ctx := testutil.TestContext()
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())
	ctx = pkgcontext.WithUserID(ctx, uuid.New().String())
	return ctx
}

// passHandler is a trivial gRPC handler that returns a sentinel value.
func passHandler(_ context.Context, _ interface{}) (interface{}, error) {
	return "ok", nil
}

func invokeInterceptor(t *testing.T, interceptor *AuthorizationInterceptor, ctx context.Context) (interface{}, error) { //nolint:revive // test helper keeps ctx as second param for readability
	t.Helper()
	unary := interceptor.UnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: testMethod}
	return unary(ctx, nil, info, passHandler)
}

func TestAdminRoleAllowed(t *testing.T) {
	resolver := &mockRoleResolver{role: "admin", active: true}
	policies := map[string][]string{testMethod: {"admin", "owner"}}
	interceptor := NewAuthorizationInterceptor(resolver, policies, noopLogger{}, 30*time.Second)

	resp, err := invokeInterceptor(t, interceptor, newTestContext())
	if err != nil {
		t.Fatalf("expected no error for admin role, got: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("expected handler response, got: %v", resp)
	}
}

func TestOwnerRoleAllowed(t *testing.T) {
	resolver := &mockRoleResolver{role: "owner", active: true}
	policies := map[string][]string{testMethod: {"admin", "owner"}}
	interceptor := NewAuthorizationInterceptor(resolver, policies, noopLogger{}, 30*time.Second)

	resp, err := invokeInterceptor(t, interceptor, newTestContext())
	if err != nil {
		t.Fatalf("expected no error for owner role, got: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("expected handler response, got: %v", resp)
	}
}

func TestMemberRoleDenied(t *testing.T) {
	resolver := &mockRoleResolver{role: "member", active: true}
	policies := map[string][]string{testMethod: {"admin", "owner"}}
	interceptor := NewAuthorizationInterceptor(resolver, policies, noopLogger{}, 30*time.Second)

	_, err := invokeInterceptor(t, interceptor, newTestContext())
	if err == nil {
		t.Fatal("expected PermissionDenied error for member role, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got: %v", st.Code())
	}
}

func TestInactiveAdminDenied(t *testing.T) {
	resolver := &mockRoleResolver{role: "admin", active: false}
	policies := map[string][]string{testMethod: {"admin", "owner"}}
	interceptor := NewAuthorizationInterceptor(resolver, policies, noopLogger{}, 30*time.Second)

	_, err := invokeInterceptor(t, interceptor, newTestContext())
	if err == nil {
		t.Fatal("expected PermissionDenied error for inactive admin, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got: %v", st.Code())
	}
}

func TestMissingUserInContext(t *testing.T) {
	resolver := &mockRoleResolver{role: "admin", active: true}
	policies := map[string][]string{testMethod: {"admin"}}
	interceptor := NewAuthorizationInterceptor(resolver, policies, noopLogger{}, 30*time.Second)

	// Context with org but no user
	ctx := testutil.TestContext()
	ctx = pkgcontext.WithOrganizationID(ctx, uuid.New())

	_, err := invokeInterceptor(t, interceptor, ctx)
	if err == nil {
		t.Fatal("expected error when user is missing from context, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got: %v", st.Code())
	}
}

func TestMissingOrgInContext(t *testing.T) {
	resolver := &mockRoleResolver{role: "admin", active: true}
	policies := map[string][]string{testMethod: {"admin"}}
	interceptor := NewAuthorizationInterceptor(resolver, policies, noopLogger{}, 30*time.Second)

	// Context with user but no org
	ctx := testutil.TestContext()
	ctx = pkgcontext.WithUserID(ctx, uuid.New().String())

	_, err := invokeInterceptor(t, interceptor, ctx)
	if err == nil {
		t.Fatal("expected error when org is missing from context, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got: %v", st.Code())
	}
}

func TestUnknownRPCPassthrough(t *testing.T) {
	resolver := &mockRoleResolver{role: "member", active: true}
	policies := map[string][]string{testMethod: {"admin"}}
	interceptor := NewAuthorizationInterceptor(resolver, policies, noopLogger{}, 30*time.Second)

	// Call a method not in the policy map
	unary := interceptor.UnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/kilocenter.api.v1.CoreService/UnknownMethod"}
	resp, err := unary(testutil.TestContext(), nil, info, passHandler)
	if err != nil {
		t.Fatalf("expected passthrough for unknown RPC, got: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("expected handler response, got: %v", resp)
	}
	if resolver.callCount.Load() != 0 {
		t.Fatal("resolver should not be called for methods outside the policy map")
	}
}

func TestCacheHitSkipsResolver(t *testing.T) {
	resolver := &mockRoleResolver{role: "admin", active: true}
	policies := map[string][]string{testMethod: {"admin"}}
	interceptor := NewAuthorizationInterceptor(resolver, policies, noopLogger{}, 30*time.Second)

	ctx := newTestContext()

	// First call populates the cache
	_, err := invokeInterceptor(t, interceptor, ctx)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if resolver.callCount.Load() != 1 {
		t.Fatalf("expected resolver to be called once, got %d", resolver.callCount.Load())
	}

	// Second call with the same context should use the cache
	_, err = invokeInterceptor(t, interceptor, ctx)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if resolver.callCount.Load() != 1 {
		t.Fatalf("expected resolver call count to remain 1 (cache hit), got %d", resolver.callCount.Load())
	}
}

func TestResolverErrorDenied(t *testing.T) {
	resolver := &mockRoleResolver{err: fmt.Errorf("database unavailable")}
	policies := map[string][]string{testMethod: {"admin"}}
	interceptor := NewAuthorizationInterceptor(resolver, policies, noopLogger{}, 30*time.Second)

	_, err := invokeInterceptor(t, interceptor, newTestContext())
	if err == nil {
		t.Fatal("expected error when resolver fails, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got: %v", st.Code())
	}
}
