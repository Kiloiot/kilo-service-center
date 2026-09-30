package interceptors

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
)

// Header fixtures for the identity-header rules.
const (
	testHeaderTenant   = "42"
	testHeaderOrg      = "00000000-0000-0000-0000-000000000001"
	testHeaderUser     = "0b0d7c1e-2f5c-4c3a-9c2e-1f8a6f0a1d2e"
	testHeaderKey      = "5b1f0c1e-3f4e-4d1a-9f2b-7c0d6e8a9b10"
	testMalformedValue = "not-an-id"
	testZeroTenant     = "0"
	testTenantID       = int64(42)
)

// recordingEventWriter keeps the security events the interceptor writes and
// the state of the context each was written with.
type recordingEventWriter struct {
	events    []*models.SystemEvent
	writeErrs []error
}

func (r *recordingEventWriter) CreateEvent(ctx context.Context, event *models.SystemEvent) error {
	r.events = append(r.events, event)
	r.writeErrs = append(r.writeErrs, ctx.Err())
	return nil
}

// TestInternalTrust_HeaderRules walks every identity-header rule: a refused
// request gets Unauthenticated and one security event, an admitted one runs
// with the tenant, organization and user it carried.
func TestInternalTrust_HeaderRules(t *testing.T) {
	org := uuid.MustParse(testHeaderOrg)
	cases := []struct {
		name     string
		metadata map[string]string
		method   string
		refused  bool
		wantOrg  uuid.UUID
		wantUser string
		wantKey  uuid.UUID
	}{
		{name: "missing tenant", metadata: map[string]string{}, method: testNonExemptMethod, refused: true},
		{name: "malformed tenant", metadata: map[string]string{grpcconst.MetadataKeyInternalTenantID: testMalformedValue}, method: testNonExemptMethod, refused: true},
		{name: "zero tenant", metadata: map[string]string{grpcconst.MetadataKeyInternalTenantID: testZeroTenant}, method: testNonExemptMethod, refused: true},
		{name: "missing org on a non-exempt method", metadata: map[string]string{grpcconst.MetadataKeyInternalTenantID: testHeaderTenant}, method: testNonExemptMethod, refused: true},
		{name: "malformed org on an exempt method", metadata: map[string]string{
			grpcconst.MetadataKeyInternalTenantID: testHeaderTenant, grpcconst.MetadataKeyInternalOrgID: testMalformedValue,
		}, method: testOrgExemptMethod, refused: true},
		{name: "malformed user", metadata: map[string]string{
			grpcconst.MetadataKeyInternalTenantID: testHeaderTenant, grpcconst.MetadataKeyInternalOrgID: testHeaderOrg, grpcconst.MetadataKeyInternalUserID: testMalformedValue,
		}, method: testNonExemptMethod, refused: true},
		{name: "malformed service account", metadata: map[string]string{
			grpcconst.MetadataKeyInternalTenantID: testHeaderTenant, grpcconst.MetadataKeyInternalOrgID: testHeaderOrg, grpcconst.MetadataKeyInternalServiceAccountID: testMalformedValue,
		}, method: testNonExemptMethod, refused: true},
		{name: "user and service account together", metadata: map[string]string{
			grpcconst.MetadataKeyInternalTenantID: testHeaderTenant, grpcconst.MetadataKeyInternalOrgID: testHeaderOrg,
			grpcconst.MetadataKeyInternalUserID: testHeaderUser, grpcconst.MetadataKeyInternalServiceAccountID: testHeaderKey,
		}, method: testNonExemptMethod, refused: true},
		{name: "service account identity", metadata: map[string]string{
			grpcconst.MetadataKeyInternalTenantID: testHeaderTenant, grpcconst.MetadataKeyInternalOrgID: testHeaderOrg, grpcconst.MetadataKeyInternalServiceAccountID: testHeaderKey,
		}, method: testNonExemptMethod, wantOrg: org, wantKey: uuid.MustParse(testHeaderKey)},
		{name: "exempt method without org", metadata: map[string]string{grpcconst.MetadataKeyInternalTenantID: testHeaderTenant}, method: testOrgExemptMethod},
		{name: "full identity", metadata: map[string]string{
			grpcconst.MetadataKeyInternalTenantID: testHeaderTenant, grpcconst.MetadataKeyInternalOrgID: testHeaderOrg, grpcconst.MetadataKeyInternalUserID: testHeaderUser,
		}, method: testNonExemptMethod, wantOrg: org, wantUser: testHeaderUser},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := &recordingEventWriter{}
			interceptor := NewInternalTrustInterceptor(logger.NewNop(), false).WithEventWriter(events)
			ctx := metadata.NewIncomingContext(testutil.TestContext(), metadata.New(tc.metadata))

			var admitted context.Context
			handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
				admitted = ctx
				return nil, nil
			}
			_, err := interceptor.UnaryInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: tc.method}, handler)

			if tc.refused {
				if status.Code(err) != codes.Unauthenticated || admitted != nil || len(events.events) != 1 {
					t.Fatalf("want a refusal with one security event: err=%v admitted=%v events=%d", err, admitted != nil, len(events.events))
				}
				return
			}
			if err != nil || admitted == nil {
				t.Fatalf("want the request admitted: err=%v", err)
			}
			if tenant, _ := pkgcontext.GetTenantID(admitted); tenant != testTenantID {
				t.Fatalf("tenant = %d, want %d", tenant, testTenantID)
			}
			if got, _ := pkgcontext.GetOrganizationID(admitted); got != tc.wantOrg {
				t.Fatalf("organization = %v, want %v", got, tc.wantOrg)
			}
			if got, _ := pkgcontext.GetUserID(admitted); got != tc.wantUser {
				t.Fatalf("user = %q, want %q", got, tc.wantUser)
			}
			if got, _ := pkgcontext.GetServiceAccountID(admitted); got != tc.wantKey {
				t.Fatalf("service account = %v, want %v", got, tc.wantKey)
			}
		})
	}
}

func TestInternalTrust_RequestWithoutMetadataIsRefused(t *testing.T) {
	events := &recordingEventWriter{}
	interceptor := NewInternalTrustInterceptor(logger.NewNop(), false).WithEventWriter(events)
	handler := func(context.Context, interface{}) (interface{}, error) {
		t.Fatal("a request without metadata reached the handler")
		return nil, nil
	}
	_, err := interceptor.UnaryInterceptor()(testutil.TestContext(), nil, &grpc.UnaryServerInfo{FullMethod: testNonExemptMethod}, handler)
	if status.Code(err) != codes.Unauthenticated || len(events.events) != 1 {
		t.Fatalf("err=%v events=%d", err, len(events.events))
	}
}
