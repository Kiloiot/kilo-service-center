package proxy

import (
	"context"
	"testing"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func TestMetadataSanitization(t *testing.T) {
	// Simulate client sending spoofed internal headers
	inMD := metadata.Pairs(
		grpcconst.MetadataKeyInternalTenantID, "999",
		grpcconst.MetadataKeyInternalOrgID, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		grpcconst.MetadataKeyInternalUserID, "spoofed-user",
		grpcconst.MetadataKeyAuthorization, "Bearer spoofed-token",
		"x-custom-header", "preserved",
	)
	ctx := metadata.NewIncomingContext(context.Background(), inMD)

	// Simulate interceptor populating context with real identity
	orgUUID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	ctx = pkgcontext.WithTenantID(ctx, 42)
	ctx = pkgcontext.WithOrganizationID(ctx, orgUUID)
	ctx = pkgcontext.WithUserID(ctx, "66666666-7777-8888-9999-aaaaaaaaaaaa")

	outMD := SanitizeAndInject(ctx)

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
