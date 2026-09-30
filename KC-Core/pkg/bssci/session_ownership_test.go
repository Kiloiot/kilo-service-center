package bssci

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// priorSessionRowID is the resumable session row the ownership tests seed.
const priorSessionRowID = int64(7)

var errOwnershipTestLoad = errors.New("pending operations unavailable")

// seedResumableRow registers a resumable prior session whose snBsUuid the
// harness connect payload carries, with one persisted SC operation.
func seedResumableRow(h *interopHarness) {
	prevUUID := make([]byte, 16)
	for i := range prevUUID {
		prevUUID[i] = byte(i + 1)
	}
	h.server.sessionSvc.(*mockSessionService).StoreSessionByUUID(&Session{ProtocolSessionState: ProtocolSessionState{
		ID:             "previous-runtime-session",
		BaseStationEUI: TestBsEui01,
		SessionUUID:    prevUUID,
		DbSessionID:    priorSessionRowID,
		LastScOpId:     -1,
	}})
	h.server.statusSvc.(*memoryStatusService).persistRows(priorSessionRowID, PersistedOperation{
		OperationID:   -1,
		OperationType: mioty.CmdStatus,
		OperationData: []byte(fmt.Sprintf(`{"command":%q,"opId":-1}`, mioty.CmdStatus)),
	})
}

// assertPriorSessionUntouched checks the prior row was neither retired nor
// stripped of its pending operations by a connection that never owned it.
func assertPriorSessionUntouched(t *testing.T, h *interopHarness) {
	t.Helper()
	terminated, disconnected := h.server.sessionSvc.(*mockSessionService).retiredRows()
	assert.NotContains(t, terminated, priorSessionRowID, "the prior session stays resumable")
	assert.NotContains(t, disconnected, priorSessionRowID, "a connection that never claimed the row does not touch it")
	statusSvc := h.server.statusSvc.(*memoryStatusService)
	assert.Empty(t, statusSvc.retiredPendingOperations(), "the prior session keeps its pending operations")
	assert.Len(t, statusSvc.persistedRowsOf(priorSessionRowID), 1, "the prior session's persisted rows survive")
}

// A resume offered in conRsp and never completed with conCmp leaves the
// prior session resumable with its pending operations.
func TestIncompleteResumeLeavesPriorSessionResumable(t *testing.T) {
	h := startInteropServer(t, EncodingJSON)
	seedResumableRow(h)

	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	conRsp := h.readFrame()
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(conRsp))
	require.Equal(t, true, conRsp["snResume"])

	require.NoError(t, h.conn.Close())
	<-h.done
	assertPriorSessionUntouched(t, h)
}

// A resume refused because the prior session's pending operations could not
// be loaded leaves that session resumable for a later attempt.
func TestRefusedResumeLeavesPriorSessionResumable(t *testing.T) {
	h := startInteropServer(t, EncodingJSON)
	seedResumableRow(h)
	statusSvc := h.server.statusSvc.(*memoryStatusService)
	statusSvc.mu.Lock()
	statusSvc.loadErr = errOwnershipTestLoad
	statusSvc.mu.Unlock()

	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	require.Equal(t, mioty.CmdError, frameCommand(h.readFrame()))
	h.writeFrame(map[string]interface{}{"command": mioty.CmdErrorAck, "opId": int64(0)})
	h.expectClosed()
	<-h.done
	assertPriorSessionUntouched(t, h)
}

// A resume that claimed its row but failed to register the live connection
// hands the row back resumable instead of retiring it.
func TestResumeFailingRegistrationKeepsRowResumable(t *testing.T) {
	h := startInteropServerWithRegistry(t, EncodingJSON, failingRegistrationConnSvc{}, nil)
	seedResumableRow(h)

	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(h.readFrame()))
	h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})
	h.expectClosed()
	<-h.done

	terminated, disconnected := h.server.sessionSvc.(*mockSessionService).retiredRows()
	assert.NotContains(t, terminated, priorSessionRowID, "the claimed row is not retired")
	assert.Contains(t, disconnected, priorSessionRowID, "the claimed row is handed back resumable")
}
