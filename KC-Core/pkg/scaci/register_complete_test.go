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

// registerCompleteServer answers a regCmp whose operation log holds op, or
// holds nothing for that opId when op is nil.
func registerCompleteServer(t *testing.T, op *models.SCACIOperation) (*Server, *MockEndpointService, *MockSCACIOperationRepository) {
	t.Helper()
	server, endpoints, _ := preAttachServer(t)
	operations := new(MockSCACIOperationRepository)
	if op == nil {
		operations.On("GetOperationByOpID", mock.Anything, preAttachSessionID, preAttachOpID).Return(nil, storage.ErrNotFound)
	} else {
		operations.On("GetOperationByOpID", mock.Anything, preAttachSessionID, preAttachOpID).Return(op, nil)
	}
	operations.On("UpdateOperationState", mock.Anything, preAttachSessionID, preAttachOpID, mock.Anything, mock.Anything).Return(nil)
	server.operationRepo = operations
	return server, endpoints, operations
}

func loggedOperation(command string, state models.OperationState) *models.SCACIOperation {
	return &models.SCACIOperation{
		SessionID: preAttachSessionID, OpId: preAttachOpID, Command: command, State: string(state),
		RequestData: map[string]interface{}{MetadataKeyEpEui: mioty.FormatEUI64(numericTestEpEui)},
	}
}

// SCACI §3.6.3, §3.14: regCmp completes a reg the service center answered
// with regRsp. A reg answered with an error is completed by errorAck, and
// any other operation is not a reg, so a regCmp for either, or for an opId
// that names no operation, is answered with a protocol error and attaches
// nothing.
func TestRegisterCompleteCompletesOnlyAnAnsweredRegister(t *testing.T) {
	for name, op := range map[string]*models.SCACIOperation{
		"reg answered with an error": loggedOperation(CmdRegister, models.OperationStateFailed),
		"reg already completed":      loggedOperation(CmdRegister, models.OperationStateCompleted),
		"dereg":                      loggedOperation(CmdDeregister, models.OperationStateAcknowledged),
		"no operation":               nil,
	} {
		t.Run(name, func(t *testing.T) {
			server, endpoints, operations := registerCompleteServer(t, op)
			registering := &Session{ID: preAttachSessionID, TenantID: preAttachTenant, State: StateActive}
			conn := &mockConn{}

			require.NoError(t, server.handleRegisterComplete(conn, registering, preAttachOpID))
			server.wg.Wait()

			require.NotEmpty(t, conn.written, "the regCmp is answered with an error")
			reply := sentError(t, conn)
			assert.EqualValues(t, POSIX_EPROTO, reply.Code)
			assertErrorToken(t, reply, errUnexpectedRegisterComplete)
			endpoints.AssertNotCalled(t, "Attach", mock.Anything, mock.Anything)
			operations.AssertNotCalled(t, "UpdateOperationState", mock.Anything, mock.Anything, mock.Anything, models.OperationStateCompleted, mock.Anything)
		})
	}
}

// A regCmp of a reg answered with regRsp completes it and pre-attaches the
// endpoint it registered.
func TestRegisterCompleteOfAnAnsweredRegisterCompletesIt(t *testing.T) {
	server, endpoints, operations := registerCompleteServer(t, loggedOperation(CmdRegister, models.OperationStateAcknowledged))
	registering := &Session{ID: preAttachSessionID, TenantID: preAttachTenant, State: StateActive}
	conn := &mockConn{}

	require.NoError(t, server.handleRegisterComplete(conn, registering, preAttachOpID))
	server.wg.Wait()

	assert.Empty(t, conn.written, "a regCmp is not answered")
	operations.AssertCalled(t, "UpdateOperationState", mock.Anything, preAttachSessionID, preAttachOpID, models.OperationStateCompleted, mock.Anything)
	endpoints.AssertCalled(t, "Attach", mock.Anything, preAttachedEndpoint())
}
