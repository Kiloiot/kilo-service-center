package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
	grpcerrors "github.com/kilocenter/KC-Core/pkg/grpc"
	"github.com/kilocenter/KC-Core/pkg/grpc/interceptors"
	pkgcontext "github.com/kilocenter/pkg/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ============================================================================
// Mock Implementations
// ============================================================================

// mockInterceptorsAPIKeyAuth implements interceptors.APIKeyAuthenticator for testing.
type mockInterceptorsAPIKeyAuth struct {
	lookupByHashFunc func(ctx context.Context, hash string) (*interceptors.APIKeyRecord, error)
	updateLastUsed   func(ctx context.Context, id uuid.UUID) error
	lastUsedCalled   bool
	lastUsedKeyID    uuid.UUID
}

func (m *mockInterceptorsAPIKeyAuth) LookupByHash(ctx context.Context, hash string) (*interceptors.APIKeyRecord, error) {
	if m.lookupByHashFunc != nil {
		return m.lookupByHashFunc(ctx, hash)
	}
	return nil, nil
}

func (m *mockInterceptorsAPIKeyAuth) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	m.lastUsedCalled = true
	m.lastUsedKeyID = id
	if m.updateLastUsed != nil {
		return m.updateLastUsed(ctx, id)
	}
	return nil
}

// hashToken returns the SHA-256 hex hash of a raw token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// nonPublicInfo returns a UnaryServerInfo for a non-public method.
func nonPublicInfo() *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: "/kilocenter.api.v1.CoreService/ListEndPoints"}
}

// captureHandler returns a handler that captures the context and a pointer to read it.
func captureHandler() (grpc.UnaryHandler, *context.Context) {
	var captured context.Context
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		captured = ctx
		return "ok", nil
	}
	return handler, &captured
}

// ============================================================================
// Token Shape Branching Tests (replaces direct isJWTShaped unit tests)
// ============================================================================

func TestAuthInterceptor_OpaqueToken_CallsAPIKeyAuth(t *testing.T) {
	rawToken := "myapikey123"
	expectedHash := hashToken(rawToken)
	keyID := uuid.New()
	orgID := uuid.New()

	apiKeyCalled := false
	mock := &mockInterceptorsAPIKeyAuth{
		lookupByHashFunc: func(_ context.Context, hash string) (*interceptors.APIKeyRecord, error) {
			apiKeyCalled = true
			if hash != expectedHash {
				t.Errorf("expected hash %s, got %s", expectedHash, hash)
			}
			return &interceptors.APIKeyRecord{
				ID: keyID, TenantID: 42, OrganizationID: orgID, IsActive: true,
			}, nil
		},
	}

	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	ai.WithAPIKeyAuthenticator(mock)

	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyAuthorization: grpcerrors.BearerPrefix + rawToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler, captured := captureHandler()
	_, err = ai.UnaryInterceptor()(ctx, nil, nonPublicInfo(), handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !apiKeyCalled {
		t.Error("expected API key authenticator to be called for opaque token")
	}

	tenantID, err := pkgcontext.GetTenantID(*captured)
	if err != nil {
		t.Fatalf("expected tenant in context: %v", err)
	}
	if tenantID != 42 {
		t.Errorf("expected tenant 42, got %d", tenantID)
	}
}

func TestAuthInterceptor_JWTShaped_NotAPIKeyFallback(t *testing.T) {
	jwtToken := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.invalidsig"

	mock := &mockInterceptorsAPIKeyAuth{
		lookupByHashFunc: func(_ context.Context, _ string) (*interceptors.APIKeyRecord, error) {
			t.Error("API key authenticator should NOT be called for JWT-shaped tokens")
			return nil, nil
		},
	}

	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	ai.WithAPIKeyAuthenticator(mock)

	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyAuthorization: grpcerrors.BearerPrefix + jwtToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}
	_, err = ai.UnaryInterceptor()(ctx, nil, nonPublicInfo(), handler)
	if err == nil {
		t.Fatal("expected error for invalid JWT")
	}
}

// ============================================================================
// API Key Auth Tests
// ============================================================================

func TestAuthInterceptor_APIKey_ValidUserKey(t *testing.T) {
	rawToken := "kc_test_user_key_abc123"
	expectedHash := hashToken(rawToken)
	keyID := uuid.New()
	orgID := uuid.New()
	userID := uuid.New()

	mock := &mockInterceptorsAPIKeyAuth{
		lookupByHashFunc: func(_ context.Context, hash string) (*interceptors.APIKeyRecord, error) {
			if hash != expectedHash {
				t.Errorf("expected hash %s, got %s", expectedHash, hash)
			}
			return &interceptors.APIKeyRecord{
				ID: keyID, TenantID: 42, OrganizationID: orgID,
				UserID: &userID, IsActive: true, IsExpired: false,
			}, nil
		},
	}

	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	ai.WithAPIKeyAuthenticator(mock)

	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyAuthorization: grpcerrors.BearerPrefix + rawToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler, captured := captureHandler()
	_, err = ai.UnaryInterceptor()(ctx, nil, nonPublicInfo(), handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tenantID, err := pkgcontext.GetTenantID(*captured)
	if err != nil {
		t.Fatalf("expected tenant ID in context: %v", err)
	}
	if tenantID != 42 {
		t.Errorf("expected tenant ID 42, got %d", tenantID)
	}

	resolvedOrgID, err := pkgcontext.GetOrganizationID(*captured)
	if err != nil {
		t.Fatalf("expected org ID in context: %v", err)
	}
	if resolvedOrgID != orgID {
		t.Errorf("expected org ID %v, got %v", orgID, resolvedOrgID)
	}

	resolvedUserID, err := pkgcontext.GetUserID(*captured)
	if err != nil {
		t.Fatalf("expected user ID in context: %v", err)
	}
	if resolvedUserID != userID.String() {
		t.Errorf("expected user ID %v, got %v", userID.String(), resolvedUserID)
	}

	if !mock.lastUsedCalled {
		t.Error("expected UpdateLastUsed to be called")
	}
	if mock.lastUsedKeyID != keyID {
		t.Errorf("expected UpdateLastUsed with key ID %v, got %v", keyID, mock.lastUsedKeyID)
	}
}

func TestAuthInterceptor_APIKey_ValidServiceAccountKey(t *testing.T) {
	rawToken := "kc_test_svc_key_def456"
	keyID := uuid.New()
	orgID := uuid.New()

	mock := &mockInterceptorsAPIKeyAuth{
		lookupByHashFunc: func(_ context.Context, _ string) (*interceptors.APIKeyRecord, error) {
			return &interceptors.APIKeyRecord{
				ID: keyID, TenantID: 42, OrganizationID: orgID,
				UserID: nil, IsActive: true, IsExpired: false,
			}, nil
		},
	}

	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	ai.WithAPIKeyAuthenticator(mock)

	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyAuthorization: grpcerrors.BearerPrefix + rawToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler, captured := captureHandler()
	_, err = ai.UnaryInterceptor()(ctx, nil, nonPublicInfo(), handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tenantID, err := pkgcontext.GetTenantID(*captured)
	if err != nil {
		t.Fatalf("expected tenant ID: %v", err)
	}
	if tenantID != 42 {
		t.Errorf("expected tenant 42, got %d", tenantID)
	}

	resolvedOrgID, err := pkgcontext.GetOrganizationID(*captured)
	if err != nil {
		t.Fatalf("expected org ID: %v", err)
	}
	if resolvedOrgID != orgID {
		t.Errorf("expected org ID %v, got %v", orgID, resolvedOrgID)
	}

	_, err = pkgcontext.GetUserID(*captured)
	if err == nil {
		t.Error("expected no user ID in context for service-account key")
	}
}

func TestAuthInterceptor_APIKey_Expired(t *testing.T) {
	rawToken := "kc_test_expired_key"

	mock := &mockInterceptorsAPIKeyAuth{
		lookupByHashFunc: func(_ context.Context, _ string) (*interceptors.APIKeyRecord, error) {
			return &interceptors.APIKeyRecord{
				ID: uuid.New(), TenantID: 42, OrganizationID: uuid.New(),
				IsActive: true, IsExpired: true,
			}, nil
		},
	}

	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	ai.WithAPIKeyAuthenticator(mock)

	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyAuthorization: grpcerrors.BearerPrefix + rawToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}
	_, err = ai.UnaryInterceptor()(ctx, nil, nonPublicInfo(), handler)
	if err == nil {
		t.Fatal("expected error for expired key")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenApiKeyExpired)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
}

func TestAuthInterceptor_APIKey_Inactive(t *testing.T) {
	rawToken := "kc_test_inactive_key"

	mock := &mockInterceptorsAPIKeyAuth{
		lookupByHashFunc: func(_ context.Context, _ string) (*interceptors.APIKeyRecord, error) {
			return &interceptors.APIKeyRecord{
				ID: uuid.New(), TenantID: 42, OrganizationID: uuid.New(),
				IsActive: false,
			}, nil
		},
	}

	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	ai.WithAPIKeyAuthenticator(mock)

	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyAuthorization: grpcerrors.BearerPrefix + rawToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}
	_, err = ai.UnaryInterceptor()(ctx, nil, nonPublicInfo(), handler)
	if err == nil {
		t.Fatal("expected error for inactive key")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenApiKeyInactive)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
}

func TestAuthInterceptor_APIKey_UnknownHash(t *testing.T) {
	rawToken := "kc_test_unknown_key"

	mock := &mockInterceptorsAPIKeyAuth{
		lookupByHashFunc: func(_ context.Context, _ string) (*interceptors.APIKeyRecord, error) {
			return nil, status.Error(codes.NotFound, "key not found")
		},
	}

	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	ai.WithAPIKeyAuthenticator(mock)

	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyAuthorization: grpcerrors.BearerPrefix + rawToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}
	_, err = ai.UnaryInterceptor()(ctx, nil, nonPublicInfo(), handler)
	if err == nil {
		t.Fatal("expected error for unknown key")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidToken)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
}

// ============================================================================
// General Auth Tests
// ============================================================================

func TestAuthInterceptor_OpaqueToken_NoAPIKeyAuth(t *testing.T) {
	opaqueToken := "kc_test_no_auth_configured"

	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	// No API key auth configured

	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyAuthorization: grpcerrors.BearerPrefix + opaqueToken,
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}
	_, err = ai.UnaryInterceptor()(ctx, nil, nonPublicInfo(), handler)
	if err == nil {
		t.Fatal("expected error for opaque token without API key auth")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidToken)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
}

func TestAuthInterceptor_AuthDisabled(t *testing.T) {
	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: false})
	if err != nil {
		t.Fatal(err)
	}

	handlerCalled := false
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		if ctx == nil {
			t.Error("expected non-nil context")
		}
		return "ok", nil
	}
	_, err = ai.UnaryInterceptor()(context.Background(), nil, nonPublicInfo(), handler)
	if err != nil {
		t.Fatalf("unexpected error when auth disabled: %v", err)
	}
	if !handlerCalled {
		t.Error("expected handler to be called when auth disabled")
	}
}

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}
	_, err = ai.UnaryInterceptor()(context.Background(), nil, nonPublicInfo(), handler)
	if err == nil {
		t.Fatal("expected error for missing metadata")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenMissingMetadata)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
}

func TestAuthInterceptor_MissingBearerPrefix(t *testing.T) {
	ai, err := interceptors.NewAuthInterceptor(interceptors.AuthConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	md := metadata.New(map[string]string{
		grpcerrors.MetadataKeyAuthorization: "NotBearer sometoken",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}
	_, err = ai.UnaryInterceptor()(ctx, nil, nonPublicInfo(), handler)
	if err == nil {
		t.Fatal("expected error for missing Bearer prefix")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	expectedCode := grpcerrors.GetGRPCCode(grpcerrors.ErrTokenInvalidAuthFormat)
	if st.Code() != expectedCode {
		t.Errorf("expected code %v, got %v", expectedCode, st.Code())
	}
}
