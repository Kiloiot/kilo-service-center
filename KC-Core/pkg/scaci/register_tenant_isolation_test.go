package scaci

import (
	"context"
	"encoding/binary"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/endpoint"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	isolationOwnerTenant    = int64(1)
	isolationForeignTenant  = int64(2)
	isolationEndpointID     = int64(9)
	isolationSessionID      = int64(321)
	isolationRegisterOpID   = int64(77)
	isolationDeregisterOpID = int64(78)
	isolationEndpointEpEui  = uint64(0x70B3D56770111505)
	isolationEndpointEUIHex = "70B3D56770111505"
)

// tenantEndpoints is a tenant-scoped endpoint store: an EUI belongs to one
// tenant, and only that tenant's lookups find it.
type tenantEndpoints struct {
	endpoints    map[int64]map[uint64]*models.EndPoint
	attaches     []int64
	propagations []uint64
}

func (s *tenantEndpoints) owner(epEui uint64) (int64, *models.EndPoint) {
	for tenantID, byEUI := range s.endpoints {
		if ep, ok := byEUI[epEui]; ok {
			return tenantID, ep
		}
	}
	return 0, nil
}

func (s *tenantEndpoints) Register(_ context.Context, req *Register, tenantID int64) string {
	if owner, ep := s.owner(req.EpEui); ep != nil && owner != tenantID {
		return ErrFailedCreateEndpoint
	}
	return ""
}

func (s *tenantEndpoints) Deregister(_ context.Context, epEui uint64, tenantID int64) string {
	ep, ok := s.endpoints[tenantID][epEui]
	if !ok {
		return ErrEndpointNotFound
	}
	ep.EpStatus = endpoint.EndpointStatusDetached
	return ""
}

func (s *tenantEndpoints) GetByEUI(_ context.Context, tenantID int64, eui []byte) (*models.EndPoint, string) {
	if ep, ok := s.endpoints[tenantID][binary.BigEndian.Uint64(eui)]; ok {
		clone := *ep
		return &clone, ""
	}
	return nil, ErrEndpointNotFound
}

func (s *tenantEndpoints) Attach(_ context.Context, attached *models.EndPoint) string {
	s.attaches = append(s.attaches, attached.ID)
	for _, ep := range s.endpoints[attached.TenantID] {
		if ep.ID == attached.ID {
			ep.EpStatus = endpoint.EndpointStatusAttached
			return ""
		}
	}
	return ErrEndpointNotFound
}

func (s *tenantEndpoints) PropagateDetachToAll(_ context.Context, _ int64, epEui uint64) []error {
	s.propagations = append(s.propagations, epEui)
	return nil
}

// SCACI §3.6: an application center registers its own tenant's endpoints. A
// reg naming another tenant's EUI is refused, and the regCmp that follows
// neither attaches nor propagates that tenant's endpoint.
func TestRegisterCompleteNeverReachesAnotherTenantsEndpoint(t *testing.T) {
	ownerEndpoint := &models.EndPoint{
		ID: isolationEndpointID, TenantID: isolationOwnerTenant,
		PreAttach: true, EpStatus: endpoint.EndpointStatusDetached,
	}
	binary.BigEndian.PutUint64(ownerEndpoint.EUI[:], isolationEndpointEpEui)
	store := &tenantEndpoints{endpoints: map[int64]map[uint64]*models.EndPoint{
		isolationOwnerTenant: {isolationEndpointEpEui: ownerEndpoint},
	}}

	var recorded map[string]interface{}
	recorder := new(MockOperationRecorder)
	recorder.On("Record", mock.Anything, mock.Anything, isolationRegisterOpID, CmdRegister, mock.Anything,
		mock.MatchedBy(func(data map[string]interface{}) bool { recorded = data; return true })).Return(nil)
	operations := new(MockSCACIOperationRepository)
	operations.On("UpdateOperationState", mock.Anything, isolationSessionID, isolationRegisterOpID, mock.Anything, mock.Anything).Return(nil)
	propagations := new(mockPropagationService)
	propagations.On("TriggerEndpointPropagate", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	ownerAC := &mockConn{}
	foreignAC := &mockConn{}
	foreignSession := withOpIDs(&Session{ID: isolationSessionID, TenantID: isolationForeignTenant, State: StateActive}, OpIDPair{SC: -1})
	server := newBroadcastULDataServer(map[net.Conn]*Session{
		ownerAC:   activeBroadcastSession(isolationOwnerTenant),
		foreignAC: foreignSession,
	})
	server.endpointSvc = store
	server.operationRecorder = recorder
	server.operationRepo = operations
	server.propagationSvc = propagations
	server.sessionSnapshotProvider = &mockSessionSnapshotProvider{sessions: []propagation.BaseStationSession{}}

	reg := withFields(regMap(numericValues(numericTestKey[:])), map[string]interface{}{
		"opId": isolationRegisterOpID, "epEui": isolationEndpointEpEui, "preAttach": true,
	})
	require.NoError(t, server.handleRegister(foreignAC, foreignSession, isolationRegisterOpID, mustMsgpack(t, reg)))
	assertErrorToken(t, sentError(t, foreignAC), ErrFailedCreateEndpoint)
	require.Equal(t, isolationEndpointEUIHex, recorded[MetadataKeyEpEui], "the refused reg is still recorded")
	operations.On("GetOperationByOpID", mock.Anything, isolationSessionID, isolationRegisterOpID).Return(&models.SCACIOperation{
		SessionID: isolationSessionID, OpId: isolationRegisterOpID, Command: CmdRegister,
		State: string(models.OperationStateFailed), RequestData: recorded,
	}, nil)
	foreignAC.written = nil

	require.NoError(t, server.handleRegisterComplete(foreignAC, foreignSession, isolationRegisterOpID))
	server.wg.Wait()

	assert.Empty(t, store.attaches, "the other tenant's endpoint is not attached")
	assert.Equal(t, endpoint.EndpointStatusDetached, ownerEndpoint.EpStatus, "the other tenant's endpoint stays detached")
	propagations.AssertNotCalled(t, "TriggerEndpointPropagate", mock.Anything, mock.Anything, mock.Anything)
}

// SCACI §3.7: an application center deregisters its own tenant's endpoints.
// A dereg naming another tenant's EUI is refused, and the deregCmp that
// follows neither detaches that tenant's endpoint from the base stations nor
// revokes its downlinks.
func TestDeregisterCompleteNeverDetachesAnotherTenantsEndpoint(t *testing.T) {
	ownerEndpoint := &models.EndPoint{ID: isolationEndpointID, TenantID: isolationOwnerTenant, EpStatus: endpoint.EndpointStatusAttached}
	binary.BigEndian.PutUint64(ownerEndpoint.EUI[:], isolationEndpointEpEui)
	store := &tenantEndpoints{endpoints: map[int64]map[uint64]*models.EndPoint{
		isolationOwnerTenant: {isolationEndpointEpEui: ownerEndpoint},
	}}

	var recorded map[string]interface{}
	recorder := new(MockOperationRecorder)
	recorder.On("Record", mock.Anything, mock.Anything, isolationDeregisterOpID, CmdDeregister, mock.Anything,
		mock.MatchedBy(func(data map[string]interface{}) bool { recorded = data; return true })).Return(nil)
	operations := new(MockSCACIOperationRepository)
	operations.On("UpdateOperationState", mock.Anything, isolationSessionID, isolationDeregisterOpID, mock.Anything, mock.Anything).Return(nil)
	downlinks := new(MockDLService)
	downlinks.On("GetDownlinkQueue", mock.Anything, mock.Anything, mock.Anything).Return([]*storage.DownlinkMessage{}, nil)

	foreignAC := &mockConn{}
	foreignSession := &Session{ID: isolationSessionID, TenantID: isolationForeignTenant, State: StateActive}
	server := newBroadcastULDataServer(map[net.Conn]*Session{foreignAC: foreignSession})
	server.endpointSvc = store
	server.operationRecorder = recorder
	server.operationRepo = operations
	server.dlSvc = downlinks

	dereg := map[string]interface{}{"command": CmdDeregister, "opId": isolationDeregisterOpID, "epEui": isolationEndpointEpEui}
	require.NoError(t, server.handleDeregister(foreignAC, foreignSession, isolationDeregisterOpID, mustMsgpack(t, dereg)))
	assertErrorToken(t, sentError(t, foreignAC), ErrEndpointNotFound)
	require.Equal(t, isolationEndpointEUIHex, recorded[MetadataKeyEpEui], "the refused dereg is still recorded")
	operations.On("GetOperationByOpID", mock.Anything, isolationSessionID, isolationDeregisterOpID).Return(&models.SCACIOperation{
		SessionID: isolationSessionID, TenantID: isolationForeignTenant, OpId: isolationDeregisterOpID, Command: CmdDeregister,
		State: string(models.OperationStateFailed), RequestData: recorded,
	}, nil)
	foreignAC.written = nil

	require.NoError(t, server.handleDeregisterComplete(foreignAC, foreignSession, isolationDeregisterOpID))
	server.wg.Wait()

	assert.Empty(t, store.propagations, "the other tenant's endpoint is not detached from the base stations")
	assert.Equal(t, endpoint.EndpointStatusAttached, ownerEndpoint.EpStatus, "the other tenant's endpoint stays attached")
	downlinks.AssertNotCalled(t, "GetDownlinkQueue", mock.Anything, mock.Anything, mock.Anything)
	require.NotEmpty(t, foreignAC.written, "the deregCmp is answered with an error")
	assertErrorToken(t, sentError(t, foreignAC), errUnexpectedDeregisterComplete)
}
