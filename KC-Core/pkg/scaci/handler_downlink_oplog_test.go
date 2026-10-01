package scaci

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// operationLogStore keeps response_data per operation with the repository's
// update semantics: an acknowledgement replaces it, a completion keeps it
// unless the completion carries data of its own.
type operationLogStore struct {
	mockOperationRepo
	mu       sync.Mutex
	response map[int64]map[string]interface{}
	states   map[int64]models.OperationState
}

func newOperationLogStore() *operationLogStore {
	return &operationLogStore{
		response: map[int64]map[string]interface{}{},
		states:   map[int64]models.OperationState{},
	}
}

func (s *operationLogStore) UpdateOperationState(_ context.Context, _ int64, opId int64, state models.OperationState, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[opId] = state
	if state == models.OperationStateAcknowledged || data != nil {
		s.response[opId] = data
	}
	return nil
}

func (s *operationLogStore) responseOf(opId int64) map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.response[opId]
}

// TestDLDataQueueComplete_KeepsAcknowledgedStatus pins that the
// dlDataQueCmp completion leaves the status recorded at acknowledgement: a
// deferred downlink stays logged as pending, not queued.
func TestDLDataQueueComplete_KeepsAcknowledgedStatus(t *testing.T) {
	const opID int64 = 330
	orgID := uuid.New()
	mockDL := new(MockDLService)
	mockDL.On("EnqueueDownlink", mock.Anything, mock.Anything).
		Return(&storage.DownlinkMessage{ID: 7, QueID: coreInternalQueID}, nil)
	mockDL.On("QueueDownlink", mock.Anything, mock.Anything, coreTenantID, orgID).
		Return(DownlinkQueueOutcome{QueID: uint64(coreInternalQueID), Deferred: true}, "")
	recorder := new(MockOperationRecorder)
	recorder.On("Record", mock.Anything, mock.Anything, opID, CmdDLDataQueue, models.OperationDirectionInbound, mock.Anything).Return(nil)
	oplog := newOperationLogStore()
	server := coreTestServer(mockDL)
	server.operationRepo = oplog
	server.operationRecorder = recorder
	session := &Session{ID: 5, TenantID: coreTenantID, OrganizationID: orgID, State: StateActive}

	payload, err := msgpack.Marshal(&DLDataQueue{
		BaseMessage: mioty.BaseMessage{CommandType: CmdDLDataQueue, OpId: opID},
		EpEui:       coreEpEUI,
		QueId:       coreACQueID,
		UserData:    [][]byte{{0x01}},
	})
	require.NoError(t, err)
	require.NoError(t, server.handleDLDataQueue(&mockConn{}, session, opID, payload))
	require.NoError(t, server.handleDLDataQueueComplete(&mockConn{}, session, opID))

	assert.Equal(t, models.OperationStateCompleted, oplog.states[opID])
	assert.Equal(t, bssci.DLQueueStatusPending, oplog.responseOf(opID)["status"])
}

// TestHandleDLDataRevoke_PacketCounterZeroIsValid pins SCACI §3.11.1: the
// packet counter identifies the scheduled data and zero is a valid counter.
func TestHandleDLDataRevoke_PacketCounterZeroIsValid(t *testing.T) {
	mockDL := new(MockDLService)
	mockDL.On("GetDownlinksByPacketCnt", mock.Anything, tenantCounterDownlinks(coreTenantID, coreEpEUI, 0)).
		Return([]*storage.DownlinkMessage{{QueID: coreInternalQueID}}, nil)
	mockDL.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(coreTenantID, coreEpEUI, uint64(coreInternalQueID))).Return(uint64(0), "")
	server := coreTestServer(mockDL)
	conn := &mockConn{}

	payload, err := msgpack.Marshal(map[string]interface{}{
		"command": CmdDLDataRevoke, "opId": int64(331), "epEui": coreEpEUI, "packetCnt": uint32(0),
	})
	require.NoError(t, err)
	require.NoError(t, server.handleDLDataRevoke(conn, &Session{TenantID: coreTenantID, State: StateActive}, 331, payload))

	var resp mioty.BaseMessage
	require.NoError(t, decodeResponse(conn.written, &resp))
	assert.Equal(t, CmdDLDataRevokeResponse, resp.CommandType)
	mockDL.AssertExpectations(t)
}

// TestHandleDLDataRevoke_MissingPacketCounterIsRejected pins that the packet
// counter is a mandatory field of dlDataRev (SCACI §3.11.1).
func TestHandleDLDataRevoke_MissingPacketCounterIsRejected(t *testing.T) {
	mockDL := new(MockDLService)
	server := coreTestServer(mockDL)
	conn := &mockConn{}

	payload, err := msgpack.Marshal(map[string]interface{}{
		"command": CmdDLDataRevoke, "opId": int64(332), "epEui": coreEpEUI,
	})
	require.NoError(t, err)
	require.NoError(t, server.handleDLDataRevoke(conn, &Session{TenantID: coreTenantID, State: StateActive}, 332, payload))

	var errorResp Error
	require.NoError(t, decodeResponse(conn.written, &errorResp))
	assert.Equal(t, POSIX_EINVAL, errorResp.Code)
	assertErrorToken(t, errorResp, errMissingMandatoryField)
	mockDL.AssertNotCalled(t, "GetDownlinksByPacketCnt", mock.Anything, mock.Anything)
}

// TestHandleDLDataRevoke_QueueRevokeLogsNoBaseStation: a downlink revoked in
// the queue was held by no base station, so the acknowledgement names none.
func TestHandleDLDataRevoke_QueueRevokeLogsNoBaseStation(t *testing.T) {
	const opID int64 = 333
	mockDL := new(MockDLService)
	mockDL.On("GetDownlinksByPacketCnt", mock.Anything, tenantCounterDownlinks(coreTenantID, coreEpEUI, 4)).
		Return([]*storage.DownlinkMessage{{QueID: coreInternalQueID}}, nil)
	mockDL.On("RevokeDownlink", mock.Anything, tenantDownlinkRef(coreTenantID, coreEpEUI, uint64(coreInternalQueID))).Return(uint64(0), "")
	oplog := newOperationLogStore()
	server := coreTestServer(mockDL)
	server.operationRepo = oplog

	payload, err := msgpack.Marshal(map[string]interface{}{
		"command": CmdDLDataRevoke, "opId": opID, "epEui": coreEpEUI, "packetCnt": uint32(4),
	})
	require.NoError(t, err)
	require.NoError(t, server.handleDLDataRevoke(&mockConn{}, &Session{ID: 5, TenantID: coreTenantID, State: StateActive}, opID, payload))

	require.Equal(t, models.OperationStateAcknowledged, oplog.states[opID])
	assert.NotContains(t, oplog.responseOf(opID), "bsEuis")
}
