package bssci

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// startActiveInteropSession completes con/conCmp on a harness whose status
// polling stays out of the way of the frames a test exchanges.
func startActiveInteropSession(t *testing.T) *interopHarness {
	t.Helper()
	h := startInteropServerWithConfig(t, EncodingJSON, func(cfg *Config) {
		cfg.StatusRequestInitialDelay = interopIODeadline * interopQuietStatusFactor
	})
	h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(h.readFrame()))
	h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})
	return h
}

// interopQuietStatusFactor keeps the first status poll beyond a test's lifetime.
const interopQuietStatusFactor = 30

func (h *interopHarness) ping(opID int64) {
	h.t.Helper()
	h.writeFrame(map[string]interface{}{"command": mioty.CmdPing, "opId": opID})
	response := h.readFrame()
	require.Equal(h.t, mioty.CmdPingResponse, frameCommand(response), "ping %d is answered", opID)
	respOpID, err := coerceInt64(response["opId"])
	require.NoError(h.t, err)
	require.Equal(h.t, opID, respOpID)
}

// Base station operations overlap: completing an older operation after a
// newer one started is legal, only new operations need a higher opId
// (rev1 §5.2, classic §3.2).
func TestOverlappingBaseStationOperationsComplete(t *testing.T) {
	h := startActiveInteropSession(t)

	h.ping(1)
	h.ping(2)
	h.writeFrame(map[string]interface{}{"command": mioty.CmdPingComplete, "opId": int64(1)})
	h.writeFrame(map[string]interface{}{"command": mioty.CmdPingComplete, "opId": int64(2)})

	h.ping(3)
}

// uplink sends a ulData and checks it is answered with its own opId.
func (h *interopHarness) uplink(opID int64, packetCnt uint32) {
	h.t.Helper()
	h.writeFrame(map[string]interface{}{
		"command":     mioty.CmdULData,
		"opId":        opID,
		"epEui":       uint64(TestEpEui01),
		"rxTime":      time.Now().UnixNano(),
		"packetCnt":   packetCnt,
		"userData":    []byte{0x42},
		"snr":         12.5,
		"rssi":        -80.5,
		"dlOpen":      false,
		"responseExp": false,
		"dlAck":       false,
	})
	response := h.readFrame()
	require.Equal(h.t, mioty.CmdULDataResponse, frameCommand(response), "ulData %d is answered", opID)
	respOpID, err := coerceInt64(response["opId"])
	require.NoError(h.t, err)
	require.Equal(h.t, opID, respOpID)
}

// Field report: a station announcing BSSCI 1.1.0 resumes its msgpack session
// (negotiated to 1.0.0), sends ulData 101951 before completing ulData 101950,
// then completes 101950. The late completion names an open operation and is
// accepted without an error, and the session stays open (rev1 §5.2).
func TestResumedStationCompletesAnUplinkAfterTheNextOneStarted(t *testing.T) {
	const (
		resumedBsOpID = int64(101949)
		resumedScOpID = int64(-33805)
	)
	h := startInteropServerWithSetup(t, EncodingMessagePack, func(server *Server) {
		server.uplinkIngestSvc = NewMockUplinkIngestSvc()
		server.config.StatusRequestInitialDelay = interopIODeadline * interopQuietStatusFactor
	})
	priorUUID := make([]byte, 16)
	for i := range priorUUID {
		priorUUID[i] = byte(i + 1)
	}
	h.server.sessionSvc.(*mockSessionService).StoreSessionByUUID(&Session{ProtocolSessionState: ProtocolSessionState{
		ID:             "previous-runtime-session",
		BaseStationEUI: TestBsEui01,
		SessionUUID:    priorUUID,
		DbSessionID:    priorSessionRowID,
		LastBsOpId:     resumedBsOpID,
		LastScOpId:     resumedScOpID,
	}})

	con := connectPayload("1.1.0", uint64(TestBsEui01))
	con["snBsOpId"] = resumedBsOpID
	con["snScOpId"] = resumedScOpID
	h.writeFrame(con)
	conRsp := h.readFrame()
	require.Equal(t, mioty.CmdConnectResponse, frameCommand(conRsp))
	require.Equal(t, true, conRsp["snResume"], "the prior session is resumed")
	require.Equal(t, mioty.MIOTYProtocolVersion, conRsp["version"])
	h.writeFrame(map[string]interface{}{"command": mioty.CmdConnectComplete, "opId": int64(0)})

	h.uplink(101950, 1)
	h.uplink(101951, 2)
	// Completions are not answered: an error frame would stall the next write
	// on the synchronous pipe or arrive in place of the pingRsp.
	h.writeFrame(map[string]interface{}{"command": mioty.CmdULDataComplete, "opId": int64(101950)})
	h.ping(101952)
	h.writeFrame(map[string]interface{}{"command": mioty.CmdULDataComplete, "opId": int64(101951)})
	h.ping(101953)
}

// A resumed base station may complete or reissue the operations it had open
// with their original IDs (rev1 §5.2: assignment continues from the previous
// state), while a new operation still needs a higher ID.
func TestResumedSessionReopensBaseStationOperations(t *testing.T) {
	server := newResumeReissueServer(t)
	persistedAttach := &PendingOperation{OperationID: 5, OperationType: mioty.CmdAttach}
	session := newResumeSession(&countingConn{}, []*PendingOperation{persistedAttach})
	session.LastBsOpId = 7
	t.Cleanup(func() { stopSessionStatus(session) })

	require.NoError(t, server.handleConnectComplete(session, &Message{OpId: 0, Command: mioty.CmdConnectComplete}, nil))

	assert.Empty(t, server.sequenceOperation(session, mioty.CmdAttachComplete, 5), "the persisted attach completes")
	assert.Empty(t, server.sequenceOperation(session, mioty.CmdULData, 7), "the newest operation may be reissued")
	assert.Equal(t, errOperationIDBackwards, server.sequenceOperation(session, mioty.CmdPing, 6), "an unopened older ID is not a new operation")
	assert.Empty(t, server.sequenceOperation(session, mioty.CmdPing, 8))
}

// A second con on an active session is an unexpected message: it is refused
// with an error, the errorAck closes only that exchange, and the session stays
// active (rev1 §5.17).
func TestConnectMessagesOnActiveSessionAreRefusedWithoutEndingIt(t *testing.T) {
	for _, command := range []string{mioty.CmdConnect, mioty.CmdConnectComplete} {
		t.Run(command, func(t *testing.T) {
			h := startActiveInteropSession(t)
			h.ping(1)

			if command == mioty.CmdConnect {
				h.writeFrame(connectPayload(mioty.MIOTYProtocolVersion, uint64(TestBsEui01)))
			} else {
				h.writeFrame(map[string]interface{}{"command": command, "opId": int64(0)})
			}
			refusal := h.readFrame()
			require.Equal(t, mioty.CmdError, frameCommand(refusal))
			h.writeFrame(map[string]interface{}{"command": mioty.CmdErrorAck, "opId": int64(0)})

			h.ping(2)
			assert.Equal(t, ConnectStateComplete, h.server.sessions.byEUI(TestBsEui01, nil).ConnectState,
				"the active session keeps its connect state")
		})
	}
}
