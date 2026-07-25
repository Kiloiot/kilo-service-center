package interceptors

import (
	"context"
	"testing"

	grpcconst "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/grpc"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
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
	info := &grpc.UnaryServerInfo{FullMethod: "/kilocenter.api.v1.CoreService/ListEndPoints"}
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
	info := &grpc.UnaryServerInfo{FullMethod: "/kilocenter.api.v1.CoreService/GetSystemStatus"}
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
	info := &grpc.UnaryServerInfo{FullMethod: "/kilocenter.api.v1.CoreService/ListEndPoints"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error when org header is present: %v", err)
	}
	if !handlerCalled {
		t.Error("expected handler to be called")
	}
}

func TestInternalTrust_CommunityMode_OrgOptional_NonExemptMethod(t *testing.T) {
	interceptor := NewInternalTrustInterceptor(logger.Get(), true)

	md := metadata.New(map[string]string{
		grpcconst.MetadataKeyInternalTenantID: "42",
		// no x-kc-internal-org-id — community mode makes it optional
	})
	ctx := metadata.NewIncomingContext(testutil.TestContext(), md)

	handlerCalled := false
	handler := func(_ context.Context, _ interface{}) (interface{}, error) {
		handlerCalled = true
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/kilocenter.api.v1.CoreService/ListEndPoints"}
	_, err := interceptor.UnaryInterceptor()(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error in community mode without org header: %v", err)
	}
	if !handlerCalled {
		t.Error("expected handler to be called in community mode")
	}
}
