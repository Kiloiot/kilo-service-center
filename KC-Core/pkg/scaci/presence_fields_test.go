package scaci

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
)

// requestTestServer answers every service call, so a refusal can only come
// from the decoding under test, and records what reached the services.
func requestTestServer() (*Server, *MockDLService, *MockEndpointService, *MockULService) {
	server, downlinks := downlinkTestServer()
	endpoints := new(MockEndpointService)
	endpoints.On("Register", mock.Anything, mock.Anything, mock.Anything).Return("").Maybe()
	endpoints.On("Deregister", mock.Anything, mock.Anything, mock.Anything).Return("").Maybe()
	endpoints.On("GetByEUI", mock.Anything, mock.Anything, mock.Anything).Return(&models.EndPoint{Bidi: true}, "").Maybe()
	uplinks := new(MockULService)
	uplinks.On("ScheduleULTransmit", mock.Anything, mock.Anything, mock.Anything).Return(int64(1), uint64(1), "").Maybe()
	stations := new(MockStatusService)
	stations.On("GetBaseStation", mock.Anything, mock.Anything, mock.Anything).Return(&models.BaseStation{}, nil).Maybe()
	server.endpointSvc = endpoints
	server.ulSvc = uplinks
	server.statusSvc = stations
	return server, downlinks, endpoints, uplinks
}

// handleRequest routes an Application Center request frame to its handler.
func handleRequest(t *testing.T, server *Server, conn *mockConn, wire map[string]interface{}) {
	t.Helper()
	payload := mustMsgpack(t, wire)
	session := downlinkTestSession()
	switch wire["command"] {
	case CmdRegister:
		require.NoError(t, server.handleRegister(conn, session, numericTestOpID, payload))
	case CmdDeregister:
		require.NoError(t, server.handleDeregister(conn, session, numericTestOpID, payload))
	case CmdULDataTransmit:
		require.NoError(t, server.handleULDataTransmit(conn, session, numericTestOpID, payload))
	case CmdDLDataQueue:
		require.NoError(t, server.handleDLDataQueue(conn, session, numericTestOpID, payload))
	case CmdDLDataRevoke:
		require.NoError(t, server.handleDLDataRevoke(conn, session, numericTestOpID, payload))
	default:
		t.Fatalf("no handler for %v", wire["command"])
	}
}

func deregMap() map[string]interface{} {
	return map[string]interface{}{"command": CmdDeregister, "opId": numericTestOpID, "epEui": numericTestEpEui}
}

func dlDataRevMap() map[string]interface{} {
	return map[string]interface{}{"command": CmdDLDataRevoke, "opId": numericTestOpID, "epEui": numericTestEpEui, "packetCnt": 1}
}

// mandatoryRequest is an Application Center request with the mandatory rows
// of its spec table past the core fields command and opId (§3.2).
type mandatoryRequest struct {
	wire      func() map[string]interface{}
	mandatory []string
	decoded   func() interface{}
}

// applicationCenterRequests are the requests of SCACI §3.6.1, §3.7.1, §3.9.1,
// §3.10.1 and §3.11.1 with their mandatory rows.
func applicationCenterRequests() map[string]mandatoryRequest {
	key := numericValues(numericTestKey[:])
	return map[string]mandatoryRequest{
		CmdRegister: {
			wire:      func() map[string]interface{} { return regMap(key) },
			mandatory: []string{"epEui", "bidi", "preAttach", "nwkKey", "shAddr", "attachCnt", "packetCnt", "dualChan", "repetition", "wideCarrOff", "longBlkDist"},
			decoded:   func() interface{} { return new(Register) },
		},
		CmdDeregister:     {wire: deregMap, mandatory: []string{"epEui"}, decoded: func() interface{} { return new(Deregister) }},
		CmdULDataTransmit: {wire: func() map[string]interface{} { return ulDataTxMap(key, []int{1}) }, mandatory: []string{"epEui", "nwkSnKey", "shAddr", "packetCnt", "userData"}, decoded: func() interface{} { return new(ULDataTransmit) }},
		CmdDLDataQueue:    {wire: func() map[string]interface{} { return dlDataQueMap(nil) }, mandatory: []string{"epEui", "queId", "cntDepend", "userData"}, decoded: func() interface{} { return new(DLDataQueue) }},
		CmdDLDataRevoke:   {wire: dlDataRevMap, mandatory: []string{"epEui", "packetCnt"}, decoded: func() interface{} { return new(DLDataRevoke) }},
	}
}

// Every mandatory field of an Application Center request (SCACI §3.6.1,
// §3.7.1, §3.9.1, §3.10.1, §3.11.1) must be present: its absence is a
// protocol error (§2.4), never a zero, false or empty value.
func TestApplicationCenterRequests_RefuseMissingMandatoryFields(t *testing.T) {
	for command, request := range applicationCenterRequests() {
		for _, field := range request.mandatory {
			t.Run(command+" "+field, func(t *testing.T) {
				server, downlinks, endpoints, uplinks := requestTestServer()
				conn := &mockConn{}
				wire := request.wire()
				delete(wire, field)

				handleRequest(t, server, conn, wire)

				assertErrorToken(t, sentError(t, conn), errMissingMandatoryField)
				assert.Empty(t, endpoints.Calls, "no endpoint is registered or deregistered")
				assert.Empty(t, uplinks.Calls, "no uplink is transmitted")
				downlinks.AssertNotCalled(t, "EnqueueDownlink", mock.Anything, mock.Anything)
				downlinks.AssertNotCalled(t, "GetDownlinksByPacketCnt", mock.Anything, mock.Anything)
			})
		}
	}
}

// version, acEui and snAcUuid are mandatory in con (SCACI §3.3.1).
func TestConnect_RefusesMissingMandatoryFields(t *testing.T) {
	for _, field := range []string{"version", "acEui", "snAcUuid"} {
		t.Run(field, func(t *testing.T) {
			wire := map[string]interface{}{
				"command": CmdConnect, "opId": OpIDConnect, "version": ProtocolVersionString,
				"acEui": numericTestEpEui, "snAcUuid": numericValues(numericTestKey[:]),
			}
			delete(wire, field)

			var req Connect
			err := decodePayload(mustMsgpack(t, wire), &req)

			require.ErrorIs(t, err, mioty.ErrMissingMandatoryField)
			assert.Equal(t, errMissingMandatoryField, decodeFailureToken(err, errInvalidConnectFormat, errInvalidConnectFormat))
		})
	}
}

// A request decodes from exactly its mandatory rows, in MessagePack and in
// JSON: the message demands nothing else, the core fields command and opId
// included, which the frame envelope requires before routing (§3.2).
func TestApplicationCenterRequests_DecodeFromExactlyTheirMandatoryFields(t *testing.T) {
	requests := applicationCenterRequests()
	requests[CmdConnect] = mandatoryRequest{
		wire: func() map[string]interface{} {
			return map[string]interface{}{"version": ProtocolVersionString, "acEui": numericTestEpEui, "snAcUuid": numericValues(numericTestKey[:])}
		},
		mandatory: []string{"version", "acEui", "snAcUuid"},
		decoded:   func() interface{} { return new(Connect) },
	}
	for command, request := range requests {
		t.Run(command, func(t *testing.T) {
			full := request.wire()
			wire := make(map[string]interface{}, len(request.mandatory))
			for _, field := range request.mandatory {
				wire[field] = full[field]
			}
			encodedJSON, err := json.Marshal(wire)
			require.NoError(t, err)

			require.NoError(t, decodePayload(mustMsgpack(t, wire), request.decoded()), "MessagePack")
			require.NoError(t, decodePayload(encodedJSON, request.decoded()), "JSON")
		})
	}
}

// A present mandatory field is accepted even when it holds its zero value:
// an empty userData queues a pure acknowledgement (SCACI §3.10), and a reg of
// a unidirectional endpoint carries bidi false.
func TestApplicationCenterRequests_ZeroValuedMandatoryFieldsArePresent(t *testing.T) {
	key := numericValues(numericTestKey[:])
	for name, tc := range map[string]struct {
		wire     map[string]interface{}
		response string
	}{
		"dlDataQue without entries":      {wire: dlDataQueMap(map[string]interface{}{"userData": []int{}}), response: CmdDLDataQueueResponse},
		"dlDataQue with one empty entry": {wire: dlDataQueMap(map[string]interface{}{"userData": [][]int{{}}}), response: CmdDLDataQueueResponse},
		"reg of a unidirectional endpoint": {
			wire:     withFields(regMap(key), map[string]interface{}{"bidi": false, "attachCnt": 0, "packetCnt": 0}),
			response: CmdRegisterResponse,
		},
	} {
		t.Run(name, func(t *testing.T) {
			server, _, _, _ := requestTestServer()
			conn := &mockConn{}

			handleRequest(t, server, conn, tc.wire)

			assert.Equal(t, tc.response, sentFrame(t, conn)["command"])
		})
	}
}
