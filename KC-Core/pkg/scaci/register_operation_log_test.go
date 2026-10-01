package scaci

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/propagation"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// storedOperationData is request data as the operation log reads it back:
// stored as JSON and decoded generically.
func storedOperationData(t *testing.T, data map[string]interface{}) map[string]interface{} {
	t.Helper()
	raw, err := json.Marshal(data)
	require.NoError(t, err)
	var stored map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &stored))
	return stored
}

// SCACI §3.6.3, BSSCI §3.8: regCmp of a pre-attached endpoint propagates the
// EUI the reg named, also one beyond the integers a float64 holds exactly.
func TestHandleRegisterComplete_PreAttachPropagatesTheRegisteredEUI(t *testing.T) {
	const (
		epEui      = uint64(0x70B3D56770111505)
		opID       = int64(42)
		endpointID = int64(7)
	)
	euiBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(euiBytes, epEui)

	var recorded map[string]interface{}
	recorder := new(MockOperationRecorder)
	recorder.On("Record", mock.Anything, mock.Anything, opID, CmdRegister, mock.Anything,
		mock.MatchedBy(func(data map[string]interface{}) bool { recorded = data; return true })).Return(nil)
	endpoints := new(MockEndpointService)
	endpoints.On("Register", mock.Anything, mock.Anything, int64(1)).Return("")
	endpoints.On("GetByEUI", mock.Anything, int64(1), euiBytes).Return(&models.EndPoint{ID: endpointID, TenantID: 1, PreAttach: true}, "")
	endpoints.On("GetByEUI", mock.Anything, int64(1), mock.Anything).Return(nil, ErrEndpointNotFound)
	endpoints.On("Attach", mock.Anything, mock.Anything).Return("")
	operations := new(MockSCACIOperationRepository)
	operations.On("UpdateOperationState", mock.Anything, int64(123), opID, mock.Anything, mock.Anything).Return(nil)
	propagations := new(mockPropagationService)
	propagations.On("TriggerEndpointPropagate", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	server := &Server{
		registry:                newTestRegistry(make(map[net.Conn]*Session), newHolderFake()),
		codec:                   testFrameCodec,
		commands:                mustTestCommandRegistry(),
		clock:                   clock.SystemClock{},
		logger:                  testLogger(),
		endpointSvc:             endpoints,
		operationRecorder:       recorder,
		operationRepo:           operations,
		propagationSvc:          propagations,
		sessionSnapshotProvider: &mockSessionSnapshotProvider{sessions: []propagation.BaseStationSession{}},
		config:                  &Config{},
	}
	conn := &mockConn{}
	session := &Session{ID: 123, TenantID: 1, State: StateActive}
	payload, err := msgpack.Marshal(&Register{
		BaseMessage: BaseMessage{Command: CmdRegister, OpId: opID},
		EpEui:       epEui,
		Bidi:        true,
		PreAttach:   true,
		NwkKey:      mioty.NetworkKey{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	})
	require.NoError(t, err)

	require.NoError(t, server.handleRegister(conn, session, opID, payload))
	operations.On("GetOperationByOpID", mock.Anything, int64(123), opID).Return(&models.SCACIOperation{
		SessionID: 123, OpId: opID, Command: CmdRegister, State: string(models.OperationStateAcknowledged),
		RequestData: storedOperationData(t, recorded),
	}, nil)
	require.NoError(t, server.handleRegisterComplete(conn, session, opID))
	server.wg.Wait()

	propagations.AssertCalled(t, "TriggerEndpointPropagate", mock.Anything, endpointID, mock.Anything)
}
