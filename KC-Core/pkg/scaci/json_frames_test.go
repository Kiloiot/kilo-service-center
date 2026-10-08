package scaci

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// SCACI frames carry JSON or MessagePack objects (§1, §3): a handler decodes
// the payload in the codec the frame arrived in.
func mustJSON(t *testing.T, msg map[string]interface{}) []byte {
	t.Helper()
	payload, err := json.Marshal(msg)
	require.NoError(t, err)
	return payload
}

func TestHandleRegister_DecodesJSONFrames(t *testing.T) {
	endpoints := new(MockEndpointService)
	endpoints.On("Register", mock.Anything, mock.MatchedBy(func(req *Register) bool {
		return [16]byte(req.NwkKey) == numericTestKey && req.ShAddr == numericTestShAddr && req.Bidi
	}), numericTestTenantID).Return("")
	conn := &mockConn{}

	err := newNumericTestServer(endpoints, nil).handleRegister(conn, createTestSession(t), numericTestOpID,
		mustJSON(t, regMap(numericValues(numericTestKey[:]))))

	require.NoError(t, err)
	assert.Equal(t, CmdRegisterResponse, sentFrame(t, conn)["command"])
	endpoints.AssertExpectations(t)
}

// The JSON codec refuses the same values the MessagePack codec refuses, with
// the same tokens (SCACI §2.4).
func TestHandleRegister_JSONFramesKeepTheFieldChecks(t *testing.T) {
	for name, tc := range map[string]struct {
		fields map[string]interface{}
		token  string
	}{
		"shAddr above 16 bits": {fields: map[string]interface{}{"shAddr": rangeTestAbove16}, token: errFieldOutOfRange},
		"nwkKey of 15 values":  {fields: map[string]interface{}{"nwkKey": numericValues(numericTestKey[:15])}, token: errInvalidNwkKeyLength},
	} {
		t.Run(name, func(t *testing.T) {
			endpoints := new(MockEndpointService)
			conn := &mockConn{}
			wire := withFields(regMap(numericValues(numericTestKey[:])), tc.fields)

			require.NoError(t, newNumericTestServer(endpoints, nil).handleRegister(conn, createTestSession(t), numericTestOpID, mustJSON(t, wire)))

			assertErrorToken(t, sentError(t, conn), tc.token)
			endpoints.AssertNotCalled(t, "Register", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestHandleULDataTransmit_DecodesJSONFrames(t *testing.T) {
	userData := []byte{0xDE, 0xAD}
	uplinks := new(MockULService)
	uplinks.On("ScheduleULTransmit", mock.Anything, mock.MatchedBy(func(req *mioty.ULDataTransmit) bool {
		return [16]byte(req.NwkSnKey) == numericTestKey && assert.ObjectsAreEqual(userData, []byte(req.UserData)) && req.ShAddr == numericTestShAddr
	}), numericTestTenantID).Return(int64(1), uint64(0x70B3D59CD00009E6), "")
	conn := &mockConn{}

	err := newNumericTestServer(nil, uplinks).handleULDataTransmit(conn, createTestSession(t), numericTestOpID,
		mustJSON(t, ulDataTxMap(numericValues(numericTestKey[:]), numericValues(userData))))

	require.NoError(t, err)
	assert.Equal(t, CmdULDataTransmitResponse, sentFrame(t, conn)["command"])
	uplinks.AssertExpectations(t)
}

func TestHandleDLDataQueue_DecodesJSONFrames(t *testing.T) {
	server, downlinks := downlinkTestServer()
	conn := &mockConn{}
	wire := dlDataQueMap(map[string]interface{}{"userData": [][]int{{1, 255}}, "prio": 1.5, "format": 7})

	require.NoError(t, server.handleDLDataQueue(conn, downlinkTestSession(), numericTestOpID, mustJSON(t, wire)))

	require.Equal(t, CmdDLDataQueueResponse, sentFrame(t, conn)["command"])
	assert.Equal(t, float32(1.5), enqueuedPriority(t, downlinks))
	for _, call := range downlinks.Calls {
		if call.Method == "QueueDownlink" {
			queued := call.Arguments.Get(1).(*DLDataQueue)
			assert.Equal(t, [][]byte{{1, 255}}, [][]byte(queued.UserData))
			require.NotNil(t, queued.Format)
			assert.Equal(t, uint8(7), *queued.Format)
		}
	}
}

func TestHandleDLDataQueue_JSONFramesKeepTheFieldChecks(t *testing.T) {
	for name, tc := range map[string]struct {
		wire  map[string]interface{}
		token string
	}{
		"missing cntDepend":    {wire: func() map[string]interface{} { w := dlDataQueMap(nil); delete(w, "cntDepend"); return w }(), token: errMissingMandatoryField},
		"format above 8 bits":  {wire: dlDataQueMap(map[string]interface{}{"format": rangeTestAbove8}), token: errFieldOutOfRange},
		"userData value > 255": {wire: dlDataQueMap(map[string]interface{}{"userData": [][]int{{256}}}), token: errFieldOutOfRange},
	} {
		t.Run(name, func(t *testing.T) {
			server, downlinks := downlinkTestServer()
			conn := &mockConn{}

			require.NoError(t, server.handleDLDataQueue(conn, downlinkTestSession(), numericTestOpID, mustJSON(t, tc.wire)))

			assertErrorToken(t, sentError(t, conn), tc.token)
			downlinks.AssertNotCalled(t, "EnqueueDownlink", mock.Anything, mock.Anything)
		})
	}
}

func TestHandleDLDataRevoke_DecodesJSONFrames(t *testing.T) {
	const packetCnt = uint32(41)
	server, downlinks := downlinkTestServer()
	conn := &mockConn{}
	wire := map[string]interface{}{"command": CmdDLDataRevoke, "opId": numericTestOpID, "epEui": numericTestEpEui, "packetCnt": packetCnt}

	require.NoError(t, server.handleDLDataRevoke(conn, downlinkTestSession(), numericTestOpID, mustJSON(t, wire)))

	downlinks.AssertCalled(t, "GetDownlinksByPacketCnt", mock.Anything, mock.MatchedBy(func(query storage.PacketCounterDownlinks) bool {
		return query.EpEUI == numericTestEpEui && query.PacketCnt == packetCnt
	}))
}
