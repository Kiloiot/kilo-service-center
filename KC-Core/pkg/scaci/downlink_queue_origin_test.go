package scaci

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// queuedByTheSession matches the row of a downlink the core test session's
// Application Center queued: its queue id and its EUI (SCACI §3.10.1, §3.12).
func queuedByTheSession() interface{} {
	return mock.MatchedBy(func(dl *storage.DownlinkMessage) bool {
		return dl != nil && dl.ACQueID != nil && *dl.ACQueID == coreACQueID && dl.QueID == 0 &&
			dl.ACEUI != nil && *dl.ACEUI == coreQueuerAcEui && dl.Ref == ""
	})
}

// TestProcessDLDataQueueCore_AcceptedWhenTheQueueRecordFails pins that the
// dlDataQue record is an audit trail only: when it cannot be written after
// the row was stored, the downlink is accepted and its row already names the
// Application Center its result goes to (SCACI §3.12).
func TestProcessDLDataQueueCore_AcceptedWhenTheQueueRecordFails(t *testing.T) {
	orgID := uuid.New()
	mockDL := new(MockDLService)
	mockDL.On("EnqueueDownlink", mock.Anything, queuedByTheSession()).
		Return(&storage.DownlinkMessage{ID: 44, QueID: coreInternalQueID}, nil)
	mockDL.On("QueueDownlink", mock.Anything, dispatchedUnder(coreInternalQueID), coreTenantID, orgID).
		Return(DownlinkQueueOutcome{QueID: uint64(coreInternalQueID), Deferred: true}, "")
	recorder := new(MockOperationRecorder)
	recorder.On("Record", mock.Anything, mock.Anything, int64(104), CmdDLDataQueue,
		models.OperationDirectionInbound, mock.Anything).Return(errors.New("operation log unavailable"))
	server := coreTestServer(mockDL)
	server.operationRecorder = recorder

	result, errToken, posixCode := server.processDLDataQueueCore(testutil.TestContext(),
		&Session{ID: 9, TenantID: coreTenantID, OrganizationID: orgID, AcEui: coreQueuerAcEui}, 104, &DLDataQueue{
			EpEui:    coreEpEUI,
			QueId:    coreACQueID,
			UserData: [][]byte{{0xAB}},
		}, applicationQueueIDOf(coreACQueID), "")
	require.Empty(t, errToken)
	require.Zero(t, posixCode)
	require.NotNil(t, result)
	assert.Equal(t, uint64(coreInternalQueID), result.QueID)
	mockDL.AssertExpectations(t)
	recorder.AssertExpectations(t)
}
