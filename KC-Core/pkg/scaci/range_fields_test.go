package scaci

import (
	"math"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

const (
	rangeTestQueID     uint64 = 7100002
	rangeTestStoredID  int64  = 9
	rangeTestAbove32   uint64 = 1 << 32
	rangeTestAbove16          = 70000
	rangeTestAbove8           = 300
	rangeTestMax16            = math.MaxUint16
	rangeTestMax32     uint64 = math.MaxUint32
	rangeTestMaxFormat        = math.MaxUint8
)

// sentError decodes the error frame the server answered with.
func sentError(t *testing.T, conn *mockConn) Error {
	t.Helper()
	var reply Error
	require.NoError(t, decodeResponse(conn.written, &reply))
	require.Equal(t, CmdError, reply.Command, "expected an error frame")
	return reply
}

func withFields(wire map[string]interface{}, fields map[string]interface{}) map[string]interface{} {
	for key, value := range fields {
		wire[key] = value
	}
	return wire
}

func dlDataQueMap(fields map[string]interface{}) map[string]interface{} {
	return withFields(map[string]interface{}{
		"command":   CmdDLDataQueue,
		"opId":      numericTestOpID,
		"epEui":     numericTestEpEui,
		"queId":     rangeTestQueID,
		"cntDepend": false,
		"userData":  [][]int{{1}},
	}, fields)
}

// downlinkTestServer accepts every downlink it is handed, so a refusal can only
// come from the decoding under test.
func downlinkTestServer() (*Server, *MockDLService) {
	downlinks := new(MockDLService)
	endpoints := new(MockEndpointService)
	endpoints.On("GetByEUI", mock.Anything, numericTestTenantID, mock.Anything).Return(&models.EndPoint{Bidi: true}, "").Maybe()
	downlinks.On("EnqueueDownlink", mock.Anything, mock.Anything).
		Return(&storage.DownlinkMessage{ID: rangeTestStoredID, QueID: int64(rangeTestQueID)}, nil).Maybe()
	downlinks.On("QueueDownlink", mock.Anything, mock.Anything, numericTestTenantID, mock.Anything).
		Return(DownlinkQueueOutcome{QueID: rangeTestQueID, Deferred: true}, "").Maybe()
	downlinks.On("GetDownlinksByPacketCnt", mock.Anything, mock.Anything).
		Return(nil, storage.ErrNotFound).Maybe()
	return &Server{
		registry:    newTestRegistry(make(map[net.Conn]*Session), nil),
		codec:       testFrameCodec,
		commands:    mustTestCommandRegistry(),
		clock:       clock.SystemClock{},
		logger:      testLogger(),
		dlSvc:       downlinks,
		endpointSvc: endpoints,
		config:      &Config{},
	}, downlinks
}

func downlinkTestSession() *Session {
	return &Session{TenantID: numericTestTenantID, OrganizationID: uuid.New(), State: StateActive}
}

func enqueuedPriority(t *testing.T, downlinks *MockDLService) float32 {
	t.Helper()
	for _, call := range downlinks.Calls {
		if call.Method == "EnqueueDownlink" {
			return call.Arguments.Get(1).(*storage.DownlinkMessage).Priority
		}
	}
	t.Fatal("the downlink was not enqueued")
	return 0
}

// dlDataQue numbers wider than their field and a non-finite priority are a
// protocol error (SCACI §2.4, §3.10.1), never truncated into another value.
func TestHandleDLDataQueue_RefusesNumbersOutsideTheirField(t *testing.T) {
	for name, fields := range map[string]map[string]interface{}{
		"format above 8 bits":             {"format": rangeTestAbove8},
		"packetCnt above 32 bits":         {"cntDepend": true, "packetCnt": []uint64{rangeTestAbove32}},
		"negative packetCnt":              {"cntDepend": true, "packetCnt": []int64{-1}},
		"NaN prio":                        {"prio": math.NaN()},
		"infinite prio":                   {"prio": math.Inf(-1)},
		"prio beyond single precision":    {"prio": math.MaxFloat64},
		"single precision infinite prio":  {"prio": float32(math.Inf(1))},
		"format as a non-integral number": {"format": 1.5},
	} {
		t.Run(name, func(t *testing.T) {
			server, downlinks := downlinkTestServer()
			conn := &mockConn{}

			require.NoError(t, server.handleDLDataQueue(conn, downlinkTestSession(), numericTestOpID, mustMsgpack(t, dlDataQueMap(fields))))

			reply := sentError(t, conn)
			assertErrorToken(t, reply, errFieldOutOfRange)
			assert.Equal(t, POSIX_ERANGE, reply.Code)
			downlinks.AssertNotCalled(t, "EnqueueDownlink", mock.Anything, mock.Anything)
		})
	}
}

// prio is single precision on every numeric wire width an encoder picks
// (SCACI §3.10.1); counters and formats at their field's limit are kept.
func TestHandleDLDataQueue_AcceptsNumbersWithinTheirField(t *testing.T) {
	for name, tc := range map[string]struct {
		fields map[string]interface{}
		prio   float32
	}{
		"float64 prio":         {fields: map[string]interface{}{"prio": 1.5}, prio: 1.5},
		"float32 prio":         {fields: map[string]interface{}{"prio": float32(2.25)}, prio: 2.25},
		"integer prio":         {fields: map[string]interface{}{"prio": 3}, prio: 3},
		"negative prio":        {fields: map[string]interface{}{"prio": -0.5}, prio: -0.5},
		"format at 8 bits":     {fields: map[string]interface{}{"format": rangeTestMaxFormat}},
		"packetCnt at 32 bits": {fields: map[string]interface{}{"cntDepend": true, "packetCnt": []uint64{rangeTestMax32}}},
	} {
		t.Run(name, func(t *testing.T) {
			server, downlinks := downlinkTestServer()
			conn := &mockConn{}

			require.NoError(t, server.handleDLDataQueue(conn, downlinkTestSession(), numericTestOpID, mustMsgpack(t, dlDataQueMap(tc.fields))))

			require.Equal(t, CmdDLDataQueueResponse, sentFrame(t, conn)["command"])
			assert.Equal(t, tc.prio, enqueuedPriority(t, downlinks))
		})
	}
}

// reg counters wider than their field are a protocol error (SCACI §2.4,
// §3.6.1): shAddr is 16 bits, attachCnt and packetCnt 32 bits.
func TestHandleRegister_RefusesCountersOutsideTheirField(t *testing.T) {
	for name, fields := range map[string]map[string]interface{}{
		"shAddr above 16 bits":    {"shAddr": rangeTestAbove16},
		"attachCnt above 32 bits": {"attachCnt": rangeTestAbove32},
		"packetCnt above 32 bits": {"packetCnt": rangeTestAbove32},
		"negative packetCnt":      {"packetCnt": -1},
	} {
		t.Run(name, func(t *testing.T) {
			endpoints := new(MockEndpointService)
			endpoints.On("Register", mock.Anything, mock.Anything, numericTestTenantID).Return("").Maybe()
			conn := &mockConn{}
			wire := withFields(regMap(numericValues(numericTestKey[:])), fields)

			require.NoError(t, newNumericTestServer(endpoints, nil).handleRegister(conn, createTestSession(t), numericTestOpID, mustMsgpack(t, wire)))

			assertErrorToken(t, sentError(t, conn), errFieldOutOfRange)
			endpoints.AssertNotCalled(t, "Register", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestHandleRegister_AcceptsCountersAtTheirWidth(t *testing.T) {
	endpoints := new(MockEndpointService)
	endpoints.On("Register", mock.Anything, mock.MatchedBy(func(req *Register) bool {
		return req.ShAddr == rangeTestMax16 && uint64(req.AttachCnt) == rangeTestMax32 && uint64(req.PacketCnt) == rangeTestMax32
	}), numericTestTenantID).Return("")
	conn := &mockConn{}
	wire := withFields(regMap(numericValues(numericTestKey[:])), map[string]interface{}{
		"shAddr": rangeTestMax16, "attachCnt": rangeTestMax32, "packetCnt": rangeTestMax32,
	})

	require.NoError(t, newNumericTestServer(endpoints, nil).handleRegister(conn, createTestSession(t), numericTestOpID, mustMsgpack(t, wire)))

	assert.Equal(t, CmdRegisterResponse, sentFrame(t, conn)["command"])
	endpoints.AssertExpectations(t)
}

// A key of another length names the key, a byte value outside 0-255 the range.
func TestHandleRegister_MalformedNetworkKeyNamesTheFault(t *testing.T) {
	outOfRange := numericValues(numericTestKey[:])
	outOfRange[0] = -1
	for name, tc := range map[string]struct {
		nwkKey interface{}
		token  string
	}{
		"15 values":      {nwkKey: numericValues(numericTestKey[:15]), token: errInvalidNwkKeyLength},
		"negative value": {nwkKey: outOfRange, token: errFieldOutOfRange},
	} {
		t.Run(name, func(t *testing.T) {
			conn := &mockConn{}

			require.NoError(t, newNumericTestServer(new(MockEndpointService), nil).handleRegister(conn, createTestSession(t), numericTestOpID, mustMsgpack(t, regMap(tc.nwkKey))))

			assertErrorToken(t, sentError(t, conn), tc.token)
		})
	}
}

// ulDataTx: shAddr is 16 bits, packetCnt 32 bits, format 8 bits (SCACI §3.9.1).
func TestHandleULDataTransmit_RefusesNumbersOutsideTheirField(t *testing.T) {
	key := numericValues(numericTestKey[:])
	for name, tc := range map[string]struct {
		wire  map[string]interface{}
		token string
	}{
		"shAddr above 16 bits":    {wire: withFields(ulDataTxMap(key, []int{1}), map[string]interface{}{"shAddr": rangeTestAbove16}), token: errFieldOutOfRange},
		"packetCnt above 32 bits": {wire: withFields(ulDataTxMap(key, []int{1}), map[string]interface{}{"packetCnt": rangeTestAbove32}), token: errFieldOutOfRange},
		"format above 8 bits":     {wire: withFields(ulDataTxMap(key, []int{1}), map[string]interface{}{"format": rangeTestAbove8}), token: errFieldOutOfRange},
		"nwkSnKey of 17 values":   {wire: ulDataTxMap(append(key, 1), []int{1}), token: errInvalidNwkSnKeyLength},
	} {
		t.Run(name, func(t *testing.T) {
			uplinks := new(MockULService)
			uplinks.On("ScheduleULTransmit", mock.Anything, mock.Anything, numericTestTenantID).Return(int64(1), uint64(1), "").Maybe()
			conn := &mockConn{}

			require.NoError(t, newNumericTestServer(nil, uplinks).handleULDataTransmit(conn, createTestSession(t), numericTestOpID, mustMsgpack(t, tc.wire)))

			assertErrorToken(t, sentError(t, conn), tc.token)
			uplinks.AssertNotCalled(t, "ScheduleULTransmit", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// dlDataRev.packetCnt is 32 bits (SCACI §3.11.1): a wider counter must not
// revoke the downlink its truncation happens to name.
func TestHandleDLDataRevoke_RefusesPacketCounterAbove32Bits(t *testing.T) {
	server, downlinks := downlinkTestServer()
	conn := &mockConn{}
	wire := map[string]interface{}{
		"command":   CmdDLDataRevoke,
		"opId":      numericTestOpID,
		"epEui":     numericTestEpEui,
		"packetCnt": rangeTestAbove32 << 1,
	}

	require.NoError(t, server.handleDLDataRevoke(conn, downlinkTestSession(), numericTestOpID, mustMsgpack(t, wire)))

	assertErrorToken(t, sentError(t, conn), errFieldOutOfRange)
	downlinks.AssertNotCalled(t, "GetDownlinksByPacketCnt", mock.Anything, mock.Anything)
}
