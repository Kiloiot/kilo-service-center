package scaci

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	preAttachTenant     = int64(1)
	preAttachOpID       = int64(42)
	preAttachEndpointID = int64(7)
	preAttachSessionID  = int64(123)
)

// preAttachServer answers a regCmp of a pre-attached endpoint.
func preAttachServer(t *testing.T) (*Server, *MockEndpointService, *mockPropagationService) {
	t.Helper()
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, numericTestEpEui)
	endpoints := new(MockEndpointService)
	endpoints.On("GetByEUI", mock.Anything, preAttachTenant, euiBytes).Return(preAttachedEndpoint(), "")
	endpoints.On("Attach", mock.Anything, preAttachedEndpoint()).Return("").Maybe()
	operations := new(MockSCACIOperationRepository)
	operations.On("GetOperationByOpID", mock.Anything, preAttachSessionID, preAttachOpID).Return(&models.SCACIOperation{
		SessionID: preAttachSessionID, OpId: preAttachOpID, Command: CmdRegister, State: string(models.OperationStateAcknowledged),
		RequestData: map[string]interface{}{MetadataKeyEpEui: mioty.FormatEUI64(numericTestEpEui)},
	}, nil)
	operations.On("UpdateOperationState", mock.Anything, preAttachSessionID, preAttachOpID, mock.Anything, mock.Anything).Return(nil)
	propagations := new(mockPropagationService)
	propagations.On("TriggerEndpointPropagate", mock.Anything, preAttachEndpointID, mock.Anything).Return(nil)
	server := newBroadcastULDataServer(map[net.Conn]*Session{})
	server.endpointSvc = endpoints
	server.operationRepo = operations
	server.propagationSvc = propagations
	server.sessionSnapshotProvider = &mockSessionSnapshotProvider{sessions: []propagation.BaseStationSession{}}
	return server, endpoints, propagations
}

func preAttachedEndpoint() *models.EndPoint {
	return &models.EndPoint{ID: preAttachEndpointID, TenantID: preAttachTenant, PreAttach: true}
}

// SCACI §3.6: regCmp of a pre-attached endpoint has the endpoint service
// attach it when the registration completes, which announces the change
// (SCACI §3.13), and the propagation follows (BSSCI §3.8).
func TestRegisterCompleteAttachesAPreAttachedEndpoint(t *testing.T) {
	server, endpoints, propagations := preAttachServer(t)
	registering := &Session{ID: preAttachSessionID, TenantID: preAttachTenant, State: StateActive}

	require.NoError(t, server.handleRegisterComplete(&mockConn{}, registering, preAttachOpID))
	server.wg.Wait()

	endpoints.AssertCalled(t, "Attach", mock.Anything, preAttachedEndpoint())
	propagations.AssertCalled(t, "TriggerEndpointPropagate", mock.Anything, preAttachEndpointID, mock.Anything)
}
