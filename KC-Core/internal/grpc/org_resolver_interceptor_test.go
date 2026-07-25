package grpc

import (
	"context"
	"crypto/x509"
	"testing"

	grpcerrors "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// mockOrgResolver implements org.Resolver for testing
type mockOrgResolver struct {
	lookupResult int64
	lookupError  error
	lookupCalls  []uuid.UUID
}

func (m *mockOrgResolver) LookupTenant(_ context.Context, orgUUID uuid.UUID) (int64, error) {
	m.lookupCalls = append(m.lookupCalls, orgUUID)
	return m.lookupResult, m.lookupError
}

func (m *mockOrgResolver) ResolveCert(_ context.Context, _ *x509.Certificate) (uuid.UUID, int64, error) {
	return uuid.Nil, 0, nil
}

func (m *mockOrgResolver) GetDefaultOrgForTenant(_ context.Context, _ int64) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// TestNewOrgResolverInterceptor_RequiresResolver tests that nil resolver is rejected
func TestNewOrgResolverInterceptor_RequiresResolver(t *testing.T) {
	_, err := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: nil,
	})
	if err == nil {
		t.Fatal("expected error when resolver is nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("expected codes.Internal, got %v", st.Code())
	}
}

// TestNewOrgResolverInterceptor_Success tests successful creation
func TestNewOrgResolverInterceptor_Success(t *testing.T) {
	interceptor, err := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver:    &mockOrgResolver{lookupResult: 42},
		SkipMethods: []string{"/grpc.health.v1.Health/Check"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if interceptor == nil {
		t.Fatal("expected non-nil interceptor")
	}
}

// TestOrgResolverInterceptor_MissingMetadata tests fail-closed on missing metadata
func TestOrgResolverInterceptor_MissingMetadata(t *testing.T) {
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: &mockOrgResolver{lookupResult: 42},
	})

	// Create context without metadata
	ctx := testutil.TestContext()

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected codes.Unauthenticated, got %v", st.Code())
	}
}

// TestOrgResolverInterceptor_MissingOrgID tests fail-closed on missing x-organization-id
func TestOrgResolverInterceptor_MissingOrgID(t *testing.T) {
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: &mockOrgResolver{lookupResult: 42},
	})

	// Create context with metadata but no org ID
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyUserID: uuid.New().String(),
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected codes.Unauthenticated, got %v", st.Code())
	}
}

// TestOrgResolverInterceptor_InvalidOrgID tests fail-closed on invalid x-organization-id
func TestOrgResolverInterceptor_InvalidOrgID(t *testing.T) {
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: &mockOrgResolver{lookupResult: 42},
	})

	// Create context with invalid org ID
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: "not-a-uuid",
		grpcerrors.MetadataKeyUserID:         uuid.New().String(),
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected codes.InvalidArgument, got %v", st.Code())
	}
}

// TestOrgResolverInterceptor_MissingUserID tests fail-closed on missing x-user-id
func TestOrgResolverInterceptor_MissingUserID(t *testing.T) {
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: &mockOrgResolver{lookupResult: 42},
	})

	// Create context with org ID but no user ID
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: uuid.New().String(),
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected codes.Unauthenticated, got %v", st.Code())
	}
}

// TestOrgResolverInterceptor_InvalidUserID tests fail-closed on invalid x-user-id
func TestOrgResolverInterceptor_InvalidUserID(t *testing.T) {
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: &mockOrgResolver{lookupResult: 42},
	})

	// Create context with valid org ID but invalid user ID
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: uuid.New().String(),
		grpcerrors.MetadataKeyUserID:         "not-a-uuid",
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected codes.InvalidArgument, got %v", st.Code())
	}
}

// TestOrgResolverInterceptor_ResolutionFailure tests fail-closed on resolver error
func TestOrgResolverInterceptor_ResolutionFailure(t *testing.T) {
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: &mockOrgResolver{
			lookupResult: 0,
			lookupError:  status.Error(codes.NotFound, "org not found"),
		},
	})

	orgID := uuid.New()
	userID := uuid.New()
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: orgID.String(),
		grpcerrors.MetadataKeyUserID:         userID.String(),
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Errorf("expected codes.PermissionDenied, got %v", st.Code())
	}
}

// TestOrgResolverInterceptor_Success tests successful resolution and context injection
func TestOrgResolverInterceptor_Success(t *testing.T) {
	resolver := &mockOrgResolver{lookupResult: 42}
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: resolver,
	})

	orgID := uuid.New()
	userID := uuid.New()
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: orgID.String(),
		grpcerrors.MetadataKeyUserID:         userID.String(),
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	var handlerCtx context.Context
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		handlerCtx = ctx
		return "success", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	resp, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "success" {
		t.Errorf("expected 'success' response, got %v", resp)
	}

	// Verify resolver was called with correct org ID
	if len(resolver.lookupCalls) != 1 {
		t.Fatalf("expected 1 lookup call, got %d", len(resolver.lookupCalls))
	}
	if resolver.lookupCalls[0] != orgID {
		t.Errorf("expected lookup call with %v, got %v", orgID, resolver.lookupCalls[0])
	}

	// Verify context was enriched with org/user/tenant
	resolvedOrgID, err := pkgcontext.GetOrganizationID(handlerCtx)
	if err != nil {
		t.Errorf("expected organization ID in context, got error: %v", err)
	}
	if resolvedOrgID != orgID {
		t.Errorf("expected org ID %v, got %v", orgID, resolvedOrgID)
	}

	resolvedUserID, err := pkgcontext.GetUserID(handlerCtx)
	if err != nil {
		t.Errorf("expected user ID in context, got error: %v", err)
	}
	if resolvedUserID != userID.String() {
		t.Errorf("expected user ID %v, got %v", userID.String(), resolvedUserID)
	}

	resolvedTenantID, err := pkgcontext.GetTenantID(handlerCtx)
	if err != nil {
		t.Errorf("expected tenant ID in context, got error: %v", err)
	}
	if resolvedTenantID != 42 {
		t.Errorf("expected tenant ID 42, got %d", resolvedTenantID)
	}
}

// TestOrgResolverInterceptor_SkipMethods tests that configured methods bypass validation
func TestOrgResolverInterceptor_SkipMethods(t *testing.T) {
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver:    &mockOrgResolver{lookupResult: 42},
		SkipMethods: []string{"/grpc.health.v1.Health/Check"},
	})

	// Create context without metadata (would normally fail)
	ctx := testutil.TestContext()

	handlerCalled := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		return "success", nil
	}

	// Test skipped method
	info := &grpc.UnaryServerInfo{FullMethod: "/grpc.health.v1.Health/Check"}
	resp, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err != nil {
		t.Fatalf("unexpected error for skipped method: %v", err)
	}
	if !handlerCalled {
		t.Error("handler should have been called for skipped method")
	}
	if resp != "success" {
		t.Errorf("expected 'success' response, got %v", resp)
	}
}

// ============================================================================
// Auth-established identity tests (Steps 1 validation)
// ============================================================================

// TestOrgResolverInterceptor_AuthIdentity_TenantMatch tests that matching
// auth-established tenant passes validation.
func TestOrgResolverInterceptor_AuthIdentity_TenantMatch(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	var tenantID int64 = 42

	resolver := &mockOrgResolver{lookupResult: tenantID}
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: resolver,
	})

	// Build context with pre-established auth identity
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), tenantID)
	ctx = pkgcontext.WithOrganizationID(ctx, orgID)
	ctx = pkgcontext.WithUserID(ctx, userID.String())

	// Add matching metadata headers
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: orgID.String(),
		grpcerrors.MetadataKeyUserID:         userID.String(),
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	handlerCalled := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("handler should have been called")
	}
}

// TestOrgResolverInterceptor_AuthIdentity_TenantMismatch tests that mismatching
// tenant between auth and header is rejected with ErrTokenIdentityMismatch.
func TestOrgResolverInterceptor_AuthIdentity_TenantMismatch(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	var authTenantID int64 = 42
	var headerTenantID int64 = 99 // Different tenant from auth

	resolver := &mockOrgResolver{lookupResult: headerTenantID}
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: resolver,
	})

	// Auth established tenant 42
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), authTenantID)
	ctx = pkgcontext.WithOrganizationID(ctx, orgID)
	ctx = pkgcontext.WithUserID(ctx, userID.String())

	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: orgID.String(),
		grpcerrors.MetadataKeyUserID:         userID.String(),
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err == nil {
		t.Fatal("expected error for tenant mismatch")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Errorf("expected codes.PermissionDenied, got %v", st.Code())
	}
}

// TestOrgResolverInterceptor_AuthIdentity_OrgMismatch tests that mismatching
// org between auth and header is rejected.
func TestOrgResolverInterceptor_AuthIdentity_OrgMismatch(t *testing.T) {
	authOrgID := uuid.New()
	headerOrgID := uuid.New() // Different org
	userID := uuid.New()
	var tenantID int64 = 42

	resolver := &mockOrgResolver{lookupResult: tenantID}
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: resolver,
	})

	// Auth established with authOrgID
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), tenantID)
	ctx = pkgcontext.WithOrganizationID(ctx, authOrgID)
	ctx = pkgcontext.WithUserID(ctx, userID.String())

	// Header sends different org
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: headerOrgID.String(),
		grpcerrors.MetadataKeyUserID:         userID.String(),
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err == nil {
		t.Fatal("expected error for org mismatch")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Errorf("expected codes.PermissionDenied, got %v", st.Code())
	}
}

// TestOrgResolverInterceptor_AuthIdentity_UserMismatch tests that mismatching
// user between auth and header is rejected.
func TestOrgResolverInterceptor_AuthIdentity_UserMismatch(t *testing.T) {
	orgID := uuid.New()
	authUserID := uuid.New()
	headerUserID := uuid.New() // Different user
	var tenantID int64 = 42

	resolver := &mockOrgResolver{lookupResult: tenantID}
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: resolver,
	})

	// Auth established with authUserID
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), tenantID)
	ctx = pkgcontext.WithOrganizationID(ctx, orgID)
	ctx = pkgcontext.WithUserID(ctx, authUserID.String())

	// Header sends different user
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: orgID.String(),
		grpcerrors.MetadataKeyUserID:         headerUserID.String(),
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err == nil {
		t.Fatal("expected error for user mismatch")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Errorf("expected codes.PermissionDenied, got %v", st.Code())
	}
}

// TestOrgResolverInterceptor_ServiceAccountPrincipal_NoUserHeader tests that
// service-account principals (no user in auth context) pass without x-user-id header.
func TestOrgResolverInterceptor_ServiceAccountPrincipal_NoUserHeader(t *testing.T) {
	orgID := uuid.New()
	var tenantID int64 = 42

	resolver := &mockOrgResolver{lookupResult: tenantID}
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: resolver,
	})

	// Auth established tenant+org but NO user (service-account principal)
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), tenantID)
	ctx = pkgcontext.WithOrganizationID(ctx, orgID)

	// Header only has org, no user
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: orgID.String(),
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	handlerCalled := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err != nil {
		t.Fatalf("unexpected error for service-account principal: %v", err)
	}
	if !handlerCalled {
		t.Error("handler should have been called for service-account principal")
	}
}

// TestOrgResolverInterceptor_ServiceAccountPrincipal_UserHeaderRejected tests that
// service-account principals with x-user-id header present are rejected (prevents spoofing).
func TestOrgResolverInterceptor_ServiceAccountPrincipal_UserHeaderRejected(t *testing.T) {
	orgID := uuid.New()
	spoofedUserID := uuid.New()
	var tenantID int64 = 42

	resolver := &mockOrgResolver{lookupResult: tenantID}
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: resolver,
	})

	// Auth established tenant+org but NO user (service-account principal)
	ctx := pkgcontext.WithTenantID(testutil.TestContext(), tenantID)
	ctx = pkgcontext.WithOrganizationID(ctx, orgID)

	// Header has org AND user (spoofing attempt)
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: orgID.String(),
		grpcerrors.MetadataKeyUserID:         spoofedUserID.String(),
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err == nil {
		t.Fatal("expected error for service-account with x-user-id header")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Errorf("expected codes.PermissionDenied, got %v", st.Code())
	}
}

// TestOrgResolverInterceptor_NoAuthIdentity_SetsFromHeaders tests that when
// auth has not established identity, values are set from headers (original behavior).
func TestOrgResolverInterceptor_NoAuthIdentity_SetsFromHeaders(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	var tenantID int64 = 42

	resolver := &mockOrgResolver{lookupResult: tenantID}
	interceptor, _ := NewOrgResolverInterceptor(OrgResolverInterceptorConfig{
		Resolver: resolver,
	})

	// No pre-established auth identity — just metadata
	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyOrganizationID: orgID.String(),
		grpcerrors.MetadataKeyUserID:         userID.String(),
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	var handlerCtx context.Context
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		handlerCtx = ctx
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test/Method"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify context was populated from headers
	resolvedOrgID, err := pkgcontext.GetOrganizationID(handlerCtx)
	if err != nil {
		t.Errorf("expected organization ID in context, got error: %v", err)
	}
	if resolvedOrgID != orgID {
		t.Errorf("expected org ID %v, got %v", orgID, resolvedOrgID)
	}

	resolvedUserID, err := pkgcontext.GetUserID(handlerCtx)
	if err != nil {
		t.Errorf("expected user ID in context, got error: %v", err)
	}
	if resolvedUserID != userID.String() {
		t.Errorf("expected user ID %v, got %v", userID.String(), resolvedUserID)
	}

	resolvedTenantID, err := pkgcontext.GetTenantID(handlerCtx)
	if err != nil {
		t.Errorf("expected tenant ID in context, got error: %v", err)
	}
	if resolvedTenantID != tenantID {
		t.Errorf("expected tenant ID %d, got %d", tenantID, resolvedTenantID)
	}
}

// TestMetadataKeyConstants verifies header constants are set correctly
func TestMetadataKeyConstants(t *testing.T) {
	if grpcerrors.MetadataKeyOrganizationID != "x-organization-id" {
		t.Errorf("expected MetadataKeyOrganizationID to be 'x-organization-id', got %q", grpcerrors.MetadataKeyOrganizationID)
	}
	if grpcerrors.MetadataKeyUserID != "x-user-id" {
		t.Errorf("expected MetadataKeyUserID to be 'x-user-id', got %q", grpcerrors.MetadataKeyUserID)
	}
}
