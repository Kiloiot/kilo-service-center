package scaci

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	numericTestOpID     int64  = 77
	numericTestEpEui    uint64 = 0x70B3D56770111505
	numericTestShAddr          = 0x1505
	numericTestTenantID int64  = 1
)

// numericTestKey holds byte values above the positive-fixint range, so its
// Numeric[16] form uses MessagePack's uint8 code as well.
var numericTestKey = [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

// numericValues renders bytes in the Numeric[n] shape an Application Center
// following SCACI §2.5 sends: an array of numbers.
func numericValues(data []byte) []int {
	values := make([]int, len(data))
	for i, b := range data {
		values[i] = int(b)
	}
	return values
}

func mustMsgpack(t *testing.T, msg map[string]interface{}) []byte {
	t.Helper()
	payload, err := msgpack.Marshal(msg)
	require.NoError(t, err)
	return payload
}

// sentFrame decodes the first frame the server wrote with a generic decoder,
// so the test sees the wire shape rather than a Go type's decoding of it.
func sentFrame(t *testing.T, conn *mockConn) map[string]interface{} {
	t.Helper()
	var frame map[string]interface{}
	require.NoError(t, decodeResponse(conn.written, &frame))
	return frame
}

// numericArray asserts a wire value is a Numeric array and returns its values.
func numericArray(t *testing.T, value interface{}) []int64 {
	t.Helper()
	items, ok := value.([]interface{})
	require.True(t, ok, "Numeric field must be a MessagePack array, got %T", value)
	values := make([]int64, len(items))
	for i, item := range items {
		n, isNumber := normalizeInt64(item)
		require.True(t, isNumber, "element %d is %T", i, item)
		values[i] = n
	}
	return values
}

func regMap(nwkKey interface{}) map[string]interface{} {
	return map[string]interface{}{
		"command":     CmdRegister,
		"opId":        numericTestOpID,
		"epEui":       numericTestEpEui,
		"bidi":        true,
		"preAttach":   false,
		"nwkKey":      nwkKey,
		"shAddr":      numericTestShAddr,
		"attachCnt":   0,
		"packetCnt":   0,
		"dualChan":    false,
		"repetition":  false,
		"wideCarrOff": false,
		"longBlkDist": false,
	}
}

func ulDataTxMap(nwkSnKey, userData interface{}) map[string]interface{} {
	return map[string]interface{}{
		"command":   CmdULDataTransmit,
		"opId":      numericTestOpID,
		"epEui":     numericTestEpEui,
		"nwkSnKey":  nwkSnKey,
		"shAddr":    numericTestShAddr,
		"packetCnt": 7,
		"userData":  userData,
	}
}

func newNumericTestServer(endpoints *MockEndpointService, uplinks *MockULService) *Server {
	return &Server{
		registry:    newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:       testFrameCodec,
		commands:    mustTestCommandRegistry(),
		clock:       clock.SystemClock{},
		logger:      testLogger(),
		endpointSvc: endpoints,
		ulSvc:       uplinks,
		config:      &Config{},
	}
}

// ulData carries its userData as Numeric[n] (SCACI §3.8.1), never as a
// MessagePack binary, and an empty payload as an empty array.
func TestBroadcastULData_SendsUserDataAsNumericArray(t *testing.T) {
	for name, userData := range map[string][]byte{
		"payload":       {0x01, 0x80, 0xFF},
		"empty payload": nil,
	} {
		t.Run(name, func(t *testing.T) {
			conn := &mockConn{}
			server := newBroadcastULDataServer(map[net.Conn]*Session{conn: activeBroadcastSession(broadcastULDataTestTenant)})
			data := broadcastULDataFixture()
			data.UserData = userData

			require.NoError(t, server.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, data))

			want := make([]int64, len(userData))
			for i, b := range userData {
				want[i] = int64(b)
			}
			assert.Equal(t, want, numericArray(t, sentFrame(t, conn)["userData"]))
		})
	}
}

// reg.nwkKey is Numeric[16] (SCACI §3.6.1); the binary form existing clients
// send stays accepted.
func TestHandleRegister_AcceptsNumericNetworkKey(t *testing.T) {
	for name, nwkKey := range map[string]interface{}{
		"spec numeric array": numericValues(numericTestKey[:]),
		"binary":             numericTestKey[:],
	} {
		t.Run(name, func(t *testing.T) {
			endpoints := new(MockEndpointService)
			endpoints.On("Register", mock.Anything, mock.MatchedBy(func(req *Register) bool {
				return [16]byte(req.NwkKey) == numericTestKey && req.EpEui == numericTestEpEui
			}), numericTestTenantID).Return("")
			conn := &mockConn{}

			err := newNumericTestServer(endpoints, nil).handleRegister(conn, createTestSession(t), numericTestOpID, mustMsgpack(t, regMap(nwkKey)))

			require.NoError(t, err)
			assert.Equal(t, CmdRegisterResponse, sentFrame(t, conn)["command"])
			endpoints.AssertExpectations(t)
		})
	}
}

// A Numeric[16] key of another length, or with a value outside 0-255, is a
// protocol error (SCACI §2.4), never a truncated or padded key.
func TestHandleRegister_RefusesMalformedNumericNetworkKey(t *testing.T) {
	tooShort := numericValues(numericTestKey[:15])
	outOfRange := numericValues(numericTestKey[:])
	outOfRange[3] = 256
	for name, nwkKey := range map[string]interface{}{"15 values": tooShort, "value above 255": outOfRange} {
		t.Run(name, func(t *testing.T) {
			endpoints := new(MockEndpointService)
			conn := &mockConn{}

			require.NoError(t, newNumericTestServer(endpoints, nil).handleRegister(conn, createTestSession(t), numericTestOpID, mustMsgpack(t, regMap(nwkKey))))

			assert.Equal(t, CmdError, sentFrame(t, conn)["command"])
			endpoints.AssertNotCalled(t, "Register")
		})
	}
}

// ulDataTx.nwkSnKey and userData are Numeric[16] and Numeric[n] (SCACI
// §3.9.1); binary forms stay accepted.
func TestHandleULDataTransmit_AcceptsNumericKeyAndUserData(t *testing.T) {
	userData := []byte{0xDE, 0xAD, 0x00, 0x7F}
	for name, wire := range map[string]map[string]interface{}{
		"spec numeric arrays": ulDataTxMap(numericValues(numericTestKey[:]), numericValues(userData)),
		"binary":              ulDataTxMap(numericTestKey[:], userData),
	} {
		t.Run(name, func(t *testing.T) {
			uplinks := new(MockULService)
			uplinks.On("ScheduleULTransmit", mock.Anything, mock.MatchedBy(func(req *mioty.ULDataTransmit) bool {
				return [16]byte(req.NwkSnKey) == numericTestKey && assert.ObjectsAreEqual(userData, []byte(req.UserData))
			}), numericTestTenantID).Return(int64(1), uint64(0x70B3D59CD00009E6), "")
			conn := &mockConn{}

			err := newNumericTestServer(nil, uplinks).handleULDataTransmit(conn, createTestSession(t), numericTestOpID, mustMsgpack(t, wire))

			require.NoError(t, err)
			assert.Equal(t, CmdULDataTransmitResponse, sentFrame(t, conn)["command"])
			uplinks.AssertExpectations(t)
		})
	}
}

func TestHandleULDataTransmit_RefusesMalformedNumericFields(t *testing.T) {
	outOfRange := []int{1, 2, 300}
	for name, wire := range map[string]map[string]interface{}{
		"nwkSnKey of 17 values":    ulDataTxMap(append(numericValues(numericTestKey[:]), 1), numericValues([]byte{1})),
		"userData value above 255": ulDataTxMap(numericValues(numericTestKey[:]), outOfRange),
	} {
		t.Run(name, func(t *testing.T) {
			uplinks := new(MockULService)
			conn := &mockConn{}

			require.NoError(t, newNumericTestServer(nil, uplinks).handleULDataTransmit(conn, createTestSession(t), numericTestOpID, mustMsgpack(t, wire)))

			assert.Equal(t, CmdError, sentFrame(t, conn)["command"])
			uplinks.AssertNotCalled(t, "ScheduleULTransmit")
		})
	}
}
