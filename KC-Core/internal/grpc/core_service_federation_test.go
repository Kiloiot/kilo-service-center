package grpc

import (
	"context"
	"testing"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"

	pb "github.com/Kiloiot/kilo-service-center/KC-Core/api/gen/kilocenter/v1"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// newTestCoreServiceForFederation builds a minimal CoreService for federation RPC tests.
func newTestCoreServiceForFederation(t *testing.T) *CoreService {
	t.Helper()
	storage := &fakeLegacyStorage{}
	svc, err := NewCoreService(CoreServiceDeps{
		Log:          logger.NewNop(),
		Audit:        &captureAuditRecorder{},
		Endpoints:    EndpointHandlerDeps{Endpoints: &fakeEndpointSvc{}, Attachment: &mockEndpointAttachmentSvc{}, Clock: clock.SystemClock{}, KeyReveals: &captureAuditRecorder{}},
		BaseStations: BaseStationHandlerDeps{BaseStations: &fakeBasestationSvc{}, Stats: storage, StatusReq: &fakeStatusReq{}, Ping: &fakePingCmd{}, Sessions: &fakeSessionDirectory{}},
		Downlinks:    downlinkHandlerDeps(t, downlinkFakes{endpoints: &fakeEndpointSvc{}, messages: &fakeMessageSvc{}}),
		ULTransmit:   ULTransmitHandlerDeps{Sessions: &fakeSessionDirectory{}, Transmitter: &fakeULTransmit{}, BaseStations: &fakeBasestationSvc{}},
		DLRX:         DLRXHandlerDeps{Queries: storage, Statuses: &fakeMessageSvc{}, Stations: &fakeMessageSvc{}, Commander: &fakeDownlinkCmd{}, Sessions: &fakeSessionDirectory{}},
		System:       SystemHandlerDeps{StartedAt: testServiceStart, SCEui: 0x0000000000000001, SCVendor: "Test", SCModel: "TestCenter", SCName: "test-instance", SCSwVersion: "test-1.0"},
	})
	require.NoError(t, err)
	return svc
}

// TestRevokeCEInstance_ReachesRegistry verifies that an admitted revocation
// reaches the CE registry; who may call it is the method policy's concern.
func TestRevokeCEInstance_ReachesRegistry(t *testing.T) {
	const ceID = "11111111-2222-3333-4444-555555555555"

	mockRegistry := &mockCERegistryHandler{
		revokeResponse: &pb.RevokeCEInstanceResponse{Success: true},
	}
	svc := newTestCoreServiceForFederation(t)
	svc.ceRegistrySvc = mockRegistry

	resp, err := svc.RevokeCEInstance(testutil.TestContext(), &pb.RevokeCEInstanceRequest{CeId: ceID, Reason: "smoke-test"})

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.True(t, mockRegistry.revokeCalled, "RevokeCEInstance on registry should be called")
}
