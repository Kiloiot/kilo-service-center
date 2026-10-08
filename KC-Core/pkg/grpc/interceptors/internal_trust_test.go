package interceptors

import (
	"context"
	"errors"
	"testing"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgcontext "github.com/Kiloiot/kilo-service-center/pkg/context"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
)

// Method fixtures: one org-requiring RPC and one org-exempt RPC.
const (
	testNonExemptMethod = "/kilocenter.api.v1.CoreService/ListEndPoints"
	testOrgExemptMethod = "/kilocenter.api.v1.CoreService/GetSystemStatus"
)

func TestInternalTrust_OrgRequired_NonExemptMethod(t *testing.T) {
	interceptor := NewInternalTrustInterceptor(logger.Get(), false)

	md := metadata.New(map[string]string{
		grpcconst.MetadataKeyInternalTenantID: "42",
		// no x-kc-internal-org-id
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		t.Error("handler should not be called when org header is missing for non-exempt method")
		return nil, nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: testNonExemptMethod}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)
	if err == nil {
		t.Fatal("expected error for missing org header on non-exempt method")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated, got %v", st.Code())
	}
}

func TestInternalTrust_OrgOptional_ExemptMethod(t *testing.T) {
	interceptor := NewInternalTrustInterceptor(logger.Get(), false)

	md := metadata.New(map[string]string{
		grpcconst.MetadataKeyInternalTenantID: "42",
		// no x-kc-internal-org-id
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	handlerCalled := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: testOrgExemptMethod}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error for exempt method without org header: %v", err)
	}
	if !handlerCalled {
		t.Error("expected handler to be called for exempt method")
	}
}

func TestInternalTrust_OrgPresent_NonExemptMethod(t *testing.T) {
	interceptor := NewInternalTrustInterceptor(logger.Get(), false)

	md := metadata.New(map[string]string{
		grpcconst.MetadataKeyInternalTenantID: "42",
		grpcconst.MetadataKeyInternalOrgID:    "00000000-0000-0000-0000-000000000001",
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	handlerCalled := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: testNonExemptMethod}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error when org header is present: %v", err)
	}
	if !handlerCalled {
		t.Error("expected handler to be called")
	}
}

type fakeDefaultOrgs struct {
	org   uuid.UUID
	err   error
	calls int
	last  int64
}

func (f *fakeDefaultOrgs) GetDefaultOrgForTenant(_ context.Context, tenantID int64) (uuid.UUID, error) {
	f.calls++
	f.last = tenantID
	return f.org, f.err
}

func TestInternalTrust_CommunityMode_OrgOptional_NonExemptMethod(t *testing.T) {
	defaultOrg := uuid.MustParse("6aa6b3db-ceaa-4a71-8ece-59cc2263f019")
	orgs := &fakeDefaultOrgs{org: defaultOrg}
	interceptor := NewInternalTrustInterceptor(logger.Get(), true).WithDefaultOrgResolver(orgs)

	md := metadata.New(map[string]string{
		grpcconst.MetadataKeyInternalTenantID: "42",
		// no x-kc-internal-org-id — community mode makes it optional
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	handlerCalled := false
	var seenOrg uuid.UUID
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		seenOrg, _ = pkgcontext.GetOrganizationID(ctx)
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: testNonExemptMethod}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error in community mode without org header: %v", err)
	}
	if !handlerCalled {
		t.Error("expected handler to be called in community mode")
	}
	if seenOrg != defaultOrg {
		t.Errorf("community mode must run under the tenant's default organization, got %s", seenOrg)
	}
	if orgs.calls != 1 || orgs.last != 42 {
		t.Errorf("default organization resolved %d times for tenant %d, want once for tenant 42", orgs.calls, orgs.last)
	}
}

func TestInternalTrust_CommunityMode_HeaderWinsOverDefaultOrg(t *testing.T) {
	orgs := &fakeDefaultOrgs{org: uuid.MustParse("6aa6b3db-ceaa-4a71-8ece-59cc2263f019")}
	interceptor := NewInternalTrustInterceptor(logger.Get(), true).WithDefaultOrgResolver(orgs)
	headerOrg := "fe7fe002-6880-4ea6-84ed-a69911dbdf8c"
	md := metadata.New(map[string]string{
		grpcconst.MetadataKeyInternalTenantID: "42",
		grpcconst.MetadataKeyInternalOrgID:    headerOrg,
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	var seenOrg uuid.UUID
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		seenOrg, _ = pkgcontext.GetOrganizationID(ctx)
		return "ok", nil
	}
	if _, err := interceptor.UnaryInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: testNonExemptMethod}, handler); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seenOrg.String() != headerOrg {
		t.Errorf("an explicit org header must win, got %s", seenOrg)
	}
	if orgs.calls != 0 {
		t.Errorf("the default organization must not be resolved when the header is present, resolved %d times", orgs.calls)
	}
}

func TestInternalTrust_CommunityMode_RefusesWithoutDefaultOrg(t *testing.T) {
	md := metadata.New(map[string]string{grpcconst.MetadataKeyInternalTenantID: "42"})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)
	handler := func(_ context.Context, _ interface{}) (interface{}, error) { return "ok", nil }
	info := &grpc.UnaryServerInfo{FullMethod: testNonExemptMethod}

	cases := map[string]struct {
		interceptor *InternalTrustInterceptor
		wantCode    codes.Code
	}{
		"no resolver wired": {
			interceptor: NewInternalTrustInterceptor(logger.Get(), true),
			wantCode:    grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgResolverRequired),
		},
		"resolver fails": {
			interceptor: NewInternalTrustInterceptor(logger.Get(), true).WithDefaultOrgResolver(&fakeDefaultOrgs{err: errResolverDown}),
			wantCode:    grpcconst.GetGRPCCode(grpcconst.ErrTokenOrgResolutionFailed),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := tc.interceptor.UnaryInterceptor()(ctx, nil, info, handler)
			if status.Code(err) != tc.wantCode {
				t.Fatalf("code = %v, want %v (err %v)", status.Code(err), tc.wantCode, err)
			}
		})
	}
}

var errResolverDown = errors.New("resolver down")
