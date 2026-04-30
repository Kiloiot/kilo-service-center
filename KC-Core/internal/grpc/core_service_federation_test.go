package grpc

import (
	"context"
	"testing"

	pb "github.com/kilocenter/KC-Core/api/gen/kilocenter/v1"
	"github.com/kilocenter/KC-Core/pkg/testutil"
	pkgcontext "github.com/kilocenter/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockCERegistryHandler records calls and returns configured responses.
type mockCERegistryHandler struct {
	revokeCalled   bool
	revokeResponse *pb.RevokeCEInstanceResponse
	revokeErr      error
}

func (m *mockCERegistryHandler) ListCEInstances(_ context.Context, _ *pb.ListCEInstancesRequest) (*pb.ListCEInstancesResponse, error) {
	return &pb.ListCEInstancesResponse{}, nil
}

func (m *mockCERegistryHandler) RevokeCEInstance(_ context.Context, _ *pb.RevokeCEInstanceRequest) (*pb.RevokeCEInstanceResponse, error) {
	m.revokeCalled = true
	if m.revokeErr != nil {
		return nil, m.revokeErr
	}
	if m.revokeResponse != nil {
		return m.revokeResponse, nil
	}
	return &pb.RevokeCEInstanceResponse{Success: true}, nil
}

// mockAdminChecker controls IsServerAdmin results.
type mockAdminChecker struct {
	isAdmin bool
	err     error
}

func (m *mockAdminChecker) IsServerAdmin(_ context.Context, _ string) (bool, error) {
	return m.isAdmin, m.err
}

// newTestCoreServiceForFederation builds a minimal CoreService for federation RPC tests.
func newTestCoreServiceForFederation(t *testing.T) *CoreService {
	t.Helper()
	storage := &fakeLegacyStorage{}
	svc, err := NewCoreService(CoreServiceDeps{
		EndpointSvc:       &fakeEndpointSvc{},
		BasestationSvc:    &fakeBasestationSvc{},
		MessageSvc:        &fakeMessageSvc{},
		DownlinkSvc:       &fakeDownlinkSvc{},
		StatusSvc:         &fakeStatusSvc{},
		DownlinkCmd:       &fakeDownlinkCmd{},
		DownlinkScheduler: &fakeDownlinkScheduler{},
		SessionDir:        &fakeSessionDirectory{},
		ULTransmit:        &fakeULTransmit{},
		StatusReq:         &fakeStatusReq{},
		PingCmd:           &fakePingCmd{},
		StatsStore:        storage,
		DownlinkStore:     storage,
		DLRXStorage:       storage,
		SCEui:             0x0000000000000001,
		SCVendor:          "Test",
		SCModel:           "TestCenter",
		SCName:            "test-instance",
		SCSwVersion:       "test-1.0",
	})
	require.NoError(t, err)
	return svc
}

// TestCEInstanceAdmin verifies that RevokeCEInstance enforces server-admin authorization.
func TestCEInstanceAdmin(t *testing.T) {
	const userID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	const ceID = "11111111-2222-3333-4444-555555555555"

	ctx := pkgcontext.WithUserID(testutil.TestContext(), userID)

	t.Run("non-admin returns PermissionDenied", func(t *testing.T) {
		mockAdmin := &mockAdminChecker{isAdmin: false}
		mockRegistry := &mockCERegistryHandler{}

		svc := newTestCoreServiceForFederation(t).
			WithAdminChecker(mockAdmin).
			WithCERegistryHandler(mockRegistry)

		_, err := svc.RevokeCEInstance(ctx, &pb.RevokeCEInstanceRequest{CeId: ceID})

		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, st.Code())
		assert.False(t, mockRegistry.revokeCalled, "RevokeCEInstance on registry should not be called for non-admin")
	})

	t.Run("admin succeeds", func(t *testing.T) {
		mockAdmin := &mockAdminChecker{isAdmin: true}
		mockRegistry := &mockCERegistryHandler{
			revokeResponse: &pb.RevokeCEInstanceResponse{Success: true},
		}

		svc := newTestCoreServiceForFederation(t).
			WithAdminChecker(mockAdmin).
			WithCERegistryHandler(mockRegistry)

		resp, err := svc.RevokeCEInstance(ctx, &pb.RevokeCEInstanceRequest{CeId: ceID, Reason: "smoke-test"})

		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.True(t, mockRegistry.revokeCalled, "RevokeCEInstance on registry should be called for admin")
	})
}
