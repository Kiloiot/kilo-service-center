package scaci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

const (
	deregCmpTenant    = int64(1)
	deregCmpSessionID = int64(456)
	deregCmpOpID      = int64(51)
)

// deregisterCompleteServer answers a deregCmp whose operation log holds op,
// or holds nothing for that opId when op is nil.
func deregisterCompleteServer(t *testing.T, op *models.SCACIOperation) (*Server, *MockEndpointService, *MockDLService, *MockSCACIOperationRepository) {
	t.Helper()
	operations := new(MockSCACIOperationRepository)
	if op == nil {
		operations.On("GetOperationByOpID", mock.Anything, deregCmpSessionID, deregCmpOpID).Return(nil, storage.ErrNotFound)
	} else {
		operations.On("GetOperationByOpID", mock.Anything, deregCmpSessionID, deregCmpOpID).Return(op, nil)
	}
	operations.On("UpdateOperationState", mock.Anything, deregCmpSessionID, deregCmpOpID, mock.Anything, mock.Anything).Return(nil)
	endpoints := new(MockEndpointService)
	endpoints.On("PropagateDetachToAll", mock.Anything, deregCmpTenant, numericTestEpEui).Return([]error{})
	downlinks := new(MockDLService)
	downlinks.On("GetDownlinkQueue", mock.Anything, mock.Anything, mock.Anything).Return([]*storage.DownlinkMessage{}, nil)

	server := newBroadcastULDataServer(nil)
	server.operationRepo = operations
	server.endpointSvc = endpoints
	server.dlSvc = downlinks
	return server, endpoints, downlinks, operations
}

func loggedDeregisterOperation(command string, state models.OperationState) *models.SCACIOperation {
	return &models.SCACIOperation{
		SessionID: deregCmpSessionID, TenantID: deregCmpTenant, OpId: deregCmpOpID, Command: command, State: string(state),
		RequestData: map[string]interface{}{MetadataKeyEpEui: mioty.FormatEUI64(numericTestEpEui)},
	}
}

func deregisteringSession() *Session {
	return &Session{ID: deregCmpSessionID, TenantID: deregCmpTenant, State: StateActive}
}

// SCACI §3.7.3, §3.14: deregCmp completes a dereg the service center
// answered with deregRsp. A dereg answered with an error is completed by
// errorAck, one not answered yet or already completed has nothing to
// complete, and any other operation is not a dereg, so a deregCmp for any of
// them, or for an opId that names no operation, is answered with a protocol
// error and neither revokes downlinks nor sends detPrp.
func TestDeregisterCompleteCompletesOnlyAnAnsweredDeregister(t *testing.T) {
	for name, op := range map[string]*models.SCACIOperation{
		"dereg answered with an error":  loggedDeregisterOperation(CmdDeregister, models.OperationStateFailed),
		"dereg not answered yet":        loggedDeregisterOperation(CmdDeregister, models.OperationStatePending),
		"dereg already completed":       loggedDeregisterOperation(CmdDeregister, models.OperationStateCompleted),
		"dlDataQue naming the endpoint": loggedDeregisterOperation(CmdDLDataQueue, models.OperationStateAcknowledged),
		"reg":                           loggedDeregisterOperation(CmdRegister, models.OperationStateAcknowledged),
		"dereg logged without its epEui": {
			SessionID: deregCmpSessionID, OpId: deregCmpOpID, Command: CmdDeregister,
			State: string(models.OperationStateAcknowledged), RequestData: map[string]interface{}{},
		},
		"no operation": nil,
	} {
		t.Run(name, func(t *testing.T) {
			server, endpoints, downlinks, operations := deregisterCompleteServer(t, op)
			conn := &mockConn{}

			require.NoError(t, server.handleDeregisterComplete(conn, deregisteringSession(), deregCmpOpID))
			server.wg.Wait()

			require.NotEmpty(t, conn.written, "the deregCmp is answered with an error")
			reply := sentError(t, conn)
			assert.EqualValues(t, POSIX_EPROTO, reply.Code)
			assertErrorToken(t, reply, errUnexpectedDeregisterComplete)
			endpoints.AssertNotCalled(t, "PropagateDetachToAll", mock.Anything, mock.Anything, mock.Anything)
			downlinks.AssertNotCalled(t, "GetDownlinkQueue", mock.Anything, mock.Anything, mock.Anything)
			operations.AssertNotCalled(t, "UpdateOperationState", mock.Anything, mock.Anything, mock.Anything, models.OperationStateCompleted, mock.Anything)
		})
	}
}

// A deregCmp of a dereg answered with deregRsp completes it and sends the
// tenant's endpoint detPrp, also when the dereg was answered on the
// connection the session had before it resumed.
func TestDeregisterCompleteOfAnAnsweredDeregisterDetachesTheEndpoint(t *testing.T) {
	server, endpoints, _, operations := deregisterCompleteServer(t, loggedDeregisterOperation(CmdDeregister, models.OperationStateAcknowledged))
	conn := &mockConn{}

	require.NoError(t, server.handleDeregisterComplete(conn, deregisteringSession(), deregCmpOpID))
	server.wg.Wait()

	assert.Empty(t, conn.written, "a deregCmp is not answered")
	endpoints.AssertCalled(t, "PropagateDetachToAll", mock.Anything, deregCmpTenant, numericTestEpEui)
	operations.AssertCalled(t, "UpdateOperationState", mock.Anything, deregCmpSessionID, deregCmpOpID, models.OperationStateCompleted, mock.Anything)
}
