package bssci

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

const queueErrorOpID = int64(-9)

// rejectionRecorder records every rejected dlDataQue the service is asked to fail.
type rejectionRecorder struct {
	mqttTestDownlinkService
	rejections []QueueRejection
}

func (r *rejectionRecorder) ProcessQueueError(_ context.Context, _ *Session, rejection QueueRejection) error {
	r.rejections = append(r.rejections, rejection)
	return nil
}

// answerWithError sends the base station's error answer to a pending
// service-center operation of the given type.
func answerWithError(t *testing.T, operation string) (*rejectionRecorder, *bsscitest.TestConn) {
	t.Helper()
	server := NewTestServerWithMemoryStatusService(logger.NewNop(), nil, nil, 1)
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{ID: "rejecting-station", BaseStationEUI: TestBsEui04, Encoding: EncodingJSON, DbSessionID: 1},
		Conn:                 &bsscitest.TestConn{Encoding: "json"},
	}
	server.statusSvc.RestorePendingOperation(session, queueErrorOpID, &PendingOperation{OperationID: queueErrorOpID, OperationType: operation})
	server.statusSvc = queueMetadata{
		StatusService: server.statusSvc, endpointEUI: queueOwnerEndpoint, queueID: queueOwnerQueueID, tenant: "3",
	}
	recorder := &rejectionRecorder{}
	server.downlinkSvc = recorder

	require.NoError(t, server.handleError(session, &Message{Command: mioty.CmdError, OpId: queueErrorOpID},
		map[string]interface{}{"code": int64(POSIX_EAGAIN), "message": "queue full"}))
	return recorder, session.Conn.(*bsscitest.TestConn)
}

// TestHandleError_RejectedDLDataQueueFailsTheDownlink: a dlDataQue the base
// station answered with error will never be transmitted (BSSCI §3.17), so
// its downlink is failed with the station's error.
func TestHandleError_RejectedDLDataQueueFailsTheDownlink(t *testing.T) {
	recorder, conn := answerWithError(t, mioty.CmdDLDataQueue)

	assert.Equal(t, []QueueRejection{{
		QueueID: queueOwnerQueueID, EndpointEUI: queueOwnerEndpoint, OwnerTenant: "3", Code: POSIX_EAGAIN, Message: "queue full",
	}}, recorder.rejections)
	assert.True(t, conn.SeenCommand(mioty.CmdErrorAck))
}

// TestHandleError_OtherOperationsFailNoDownlink: an error to any other
// service-center operation leaves the downlink queue alone.
func TestHandleError_OtherOperationsFailNoDownlink(t *testing.T) {
	recorder, conn := answerWithError(t, mioty.CmdDLDataRevoke)

	assert.Empty(t, recorder.rejections)
	assert.True(t, conn.SeenCommand(mioty.CmdErrorAck))
}
