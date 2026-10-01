package proxy

import (
	"testing"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

const (
	testPeerSecret    = "peer-secret-for-tests"
	testSpoofedSecret = "client-supplied-secret"
	testNoPeerSecret  = ""
)

func TestSanitizeAndInject_PresentsTheConfiguredPeerSecret(t *testing.T) {
	inMD := metadata.Pairs(grpcconst.MetadataKeyInternalPeerSecret, testSpoofedSecret)
	ctx := metadata.NewIncomingContext(testutil.TestContext(), inMD)

	outMD := SanitizeAndInject(ctx, testPeerSecret)
	if v := outMD.Get(grpcconst.MetadataKeyInternalPeerSecret); len(v) != 1 || v[0] != testPeerSecret {
		t.Errorf("expected the gateway's own peer secret, got %v", v)
	}

	outMD = SanitizeAndInject(ctx, testNoPeerSecret)
	if v := outMD.Get(grpcconst.MetadataKeyInternalPeerSecret); len(v) != 0 {
		t.Errorf("a client-supplied secret must never be forwarded, got %v", v)
	}
}

func TestMetadataSanitization(t *testing.T) {
	// Simulate client sending spoofed internal headers
	inMD := metadata.Pairs(
		grpcconst.MetadataKeyInternalTenantID, "999",
		grpcconst.MetadataKeyInternalOrgID, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		grpcconst.MetadataKeyInternalUserID, "spoofed-user",
		grpcconst.MetadataKeyAuthorization, "Bearer spoofed-token",
		"x-custom-header", "preserved",
	)
	ctx := metadata.NewIncomingContext(testutil.TestContext(), inMD)

	// Simulate interceptor populating context with real identity
	orgUUID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	ctx = pkgcontext.WithTenantID(ctx, 42)
	ctx = pkgcontext.WithOrganizationID(ctx, orgUUID)
	ctx = pkgcontext.WithUserID(ctx, "66666666-7777-8888-9999-aaaaaaaaaaaa")

	outMD := SanitizeAndInject(ctx, testNoPeerSecret)

	// Spoofed internal headers must be replaced with context values
	if v := outMD.Get(grpcconst.MetadataKeyInternalTenantID); len(v) != 1 || v[0] != "42" {
		t.Errorf("expected tenant=42, got %v", v)
	}
	if v := outMD.Get(grpcconst.MetadataKeyInternalOrgID); len(v) != 1 || v[0] != orgUUID.String() {
		t.Errorf("expected org=%s, got %v", orgUUID.String(), v)
	}
	if v := outMD.Get(grpcconst.MetadataKeyInternalUserID); len(v) != 1 || v[0] != "66666666-7777-8888-9999-aaaaaaaaaaaa" {
		t.Errorf("expected user=66666666-7777-8888-9999-aaaaaaaaaaaa, got %v", v)
	}

	// Authorization must be stripped
	if v := outMD.Get(grpcconst.MetadataKeyAuthorization); len(v) != 0 {
		t.Errorf("authorization header should be stripped, got %v", v)
	}

	// Non-spoofable headers must be preserved
	if v := outMD.Get("x-custom-header"); len(v) != 1 || v[0] != "preserved" {
		t.Errorf("custom header should be preserved, got %v", v)
	}
}

func TestSanitizeAndInject_ServiceAccountHeaderComesOnlyFromTheValidatedKey(t *testing.T) {
	spoofed := metadata.Pairs(grpcconst.MetadataKeyInternalServiceAccountID, "11111111-1111-1111-1111-111111111111")
	ctx := metadata.NewIncomingContext(testutil.TestContext(), spoofed)

	if v := SanitizeAndInject(ctx, testNoPeerSecret).Get(grpcconst.MetadataKeyInternalServiceAccountID); len(v) != 0 {
		t.Fatalf("a client-supplied service account must never be forwarded, got %v", v)
	}

	keyID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	outMD := SanitizeAndInject(pkgcontext.WithServiceAccountID(ctx, keyID), testNoPeerSecret)
	if v := outMD.Get(grpcconst.MetadataKeyInternalServiceAccountID); len(v) != 1 || v[0] != keyID.String() {
		t.Fatalf("expected the validated key %s, got %v", keyID, v)
	}
	if v := outMD.Get(grpcconst.MetadataKeyInternalUserID); len(v) != 0 {
		t.Fatalf("a service account carries no user, got %v", v)
	}
}
