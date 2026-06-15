package bssci_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// TestSCOriginatedMessageAssembly verifies BSSCI §2.5-01:
// Every mandatory field must be populated when assembling Service Center-originated
// protocol messages. Tests use real send helpers and capture actual bytes in both
// JSON and MessagePack encodings to verify the implementation.

// assemblyMockConn captures messages sent via Write() and decodes them based on encoding.
// Mimics the pattern from status_handlers_test.go.
type assemblyMockConn struct {
	net.Conn
	sentMessages []map[string]interface{}
	encoding     string // "json" or "msgpack"
}

func (m *assemblyMockConn) Write(b []byte) (n int, err error) {
	// Skip 12-byte BSSCI header (MIOTYB01 + payload size)
	if len(b) == 12 && bytes.HasPrefix(b, mioty.MIOTYFrameIdentifier[:]) {
		return len(b), nil
	}

	// Decode based on session encoding
	var msg map[string]interface{}
	if m.encoding == "json" {
		err = json.Unmarshal(b, &msg)
	} else {
		err = msgpack.Unmarshal(b, &msg)
	}

	if err == nil {
		m.sentMessages = append(m.sentMessages, msg)
	}
	return len(b), nil
}

// Reset clears sent messages between test operations
func (m *assemblyMockConn) Reset() {
	m.sentMessages = nil
}

func (m *assemblyMockConn) Close() error                       { return nil }
func (m *assemblyMockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *assemblyMockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *assemblyMockConn) SetDeadline(_ time.Time) error      { return nil }
func (m *assemblyMockConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *assemblyMockConn) SetWriteDeadline(_ time.Time) error { return nil }
func (m *assemblyMockConn) Read(_ []byte) (n int, err error)   { return 0, nil }

// TestSCOriginatedOperations tests all SC-originated operations via real send helpers
// in both JSON and MessagePack encodings per BSSCI §1 dual encoding requirement.
func TestSCOriginatedOperations(t *testing.T) {
	testLogger := logger.NewNop()

	// Test operations to exercise
	operations := []struct {
		name        string
		sendFunc    func(*bssci.Server, *bssci.Session) error
		expectedCmd string
	}{
		{
			name: "SendStatusRequest",
			sendFunc: func(s *bssci.Server, sess *bssci.Session) error {
				_, err := s.SendStatusRequest(sess)
				return err
			},
			expectedCmd: mioty.CmdStatus,
		},
		{
			name: "SendPing",
			sendFunc: func(s *bssci.Server, sess *bssci.Session) error {
				return s.SendPing(sess.ID)
			},
			expectedCmd: mioty.CmdPing,
		},
	}

	for _, op := range operations {
		for _, encoding := range []string{"json", "msgpack"} {
			testName := op.name + "_" + encoding
			t.Run(testName, func(t *testing.T) {
				// Create mock connection with encoding
				mockConn := &assemblyMockConn{encoding: encoding}
				mockConn.Reset()

				// Create test server and session
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
					queueSerializer, auditLogger, tenantResolver, mockStorage :=
					bssci.CreateTestServices(testLogger, nil)

				server := bssci.NewTestServer(testLogger, mockStorage, nil, 1,
					sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
					queueSerializer, auditLogger, tenantResolver)

				session := &bssci.Session{
					ID:                "test-session",
					BaseStationEUI:    bssci.TestBsEui04,
					Conn:              mockConn,
					Encoding:          encoding,
					LastScOpId:        -1,
					HandshakeComplete: true,
				}

				// Register session with server for operations that look up by sessionID
				server.RegisterSession(session)

				// Execute operation
				err := op.sendFunc(server, session)
				require.NoError(t, err, "Operation should succeed")

				// Verify message was sent
				require.Len(t, mockConn.sentMessages, 1,
					"Should send exactly one message")

				msg := mockConn.sentMessages[0]

				// Verify exact command constant (not just presence)
				assert.Equal(t, op.expectedCmd, msg["command"],
					"Must emit exact command constant per §2.5-01")

				// Verify opId is present and numeric
				assert.Contains(t, msg, "opId", "Message must have opId field")

				// Verify opId is negative (SC-originated)
				// JSON unmarshals as float64, MessagePack as int64
				var opId int64 //nolint:revive // opId matches MIOTY protocol field name
				switch v := msg["opId"].(type) {
				case int64:
					opId = v
				case float64:
					opId = int64(v)
				default:
					t.Fatalf("opId must be numeric type, got %T", v)
				}
				assert.Less(t, opId, int64(0), "SC-originated opId must be negative")
			})
		}
	}
}

// TestBSOriginatedResponseHandlers tests response handlers that send completion messages
// in both JSON and MessagePack encodings.
func TestBSOriginatedResponseHandlers(t *testing.T) {
	testLogger := logger.NewNop()

	// Response handlers to test
	handlers := []struct {
		name        string
		setupData   func() map[string]interface{}
		callHandler func(*bssci.Server, *bssci.Session, *bssci.Message, map[string]interface{}) error
		expectedCmd string
	}{
		{
			name: "HandleStatusResponse",
			setupData: func() map[string]interface{} {
				return map[string]interface{}{
					"code":      int64(0),
					"message":   "operational",
					"time":      time.Now().UnixNano(),
					"dutyCycle": float64(0.5),
				}
			},
			callHandler: func(s *bssci.Server, sess *bssci.Session, msg *bssci.Message, data map[string]interface{}) error {
				return s.CallHandleStatusResponse(sess, msg, data)
			},
			expectedCmd: mioty.CmdStatusComplete,
		},
		{
			name: "HandlePingResponse",
			setupData: func() map[string]interface{} {
				return map[string]interface{}{
					"code":     int64(0),
					"encoding": "json",
					"version":  "1.0.0",
				}
			},
			callHandler: func(s *bssci.Server, sess *bssci.Session, msg *bssci.Message, data map[string]interface{}) error {
				return s.CallHandlePingResponse(sess, msg, data)
			},
			expectedCmd: mioty.CmdPingComplete,
		},
		{
			name: "HandleAttachPropagateResponse",
			setupData: func() map[string]interface{} {
				return map[string]interface{}{
					"code":   int64(0),
					"epEui":  int64(bssci.TestEpEui01),
					"shAddr": int64(0x1234),
				}
			},
			callHandler: func(s *bssci.Server, sess *bssci.Session, msg *bssci.Message, data map[string]interface{}) error {
				return s.CallHandleAttachPropagateResponse(sess, msg, data)
			},
			expectedCmd: mioty.CmdAttachPropagateComplete,
		},
		{
			name: "HandleDetachPropagateResponse",
			setupData: func() map[string]interface{} {
				return map[string]interface{}{
					"code":  int64(0),
					"epEui": int64(bssci.TestEpEui01),
				}
			},
			callHandler: func(s *bssci.Server, sess *bssci.Session, msg *bssci.Message, data map[string]interface{}) error {
				return s.CallHandleDetachPropagateResponse(sess, msg, data)
			},
			expectedCmd: mioty.CmdDetachPropagateComplete,
		},
		{
			name: "HandleDLDataResultResponse",
			setupData: func() map[string]interface{} {
				return map[string]interface{}{}
			},
			callHandler: func(s *bssci.Server, sess *bssci.Session, msg *bssci.Message, data map[string]interface{}) error {
				return s.CallHandleDLDataResultResponse(sess, msg, data)
			},
			expectedCmd: mioty.CmdDLDataResultComplete,
		},
	}

	for _, handler := range handlers {
		for _, encoding := range []string{"json", "msgpack"} {
			testName := handler.name + "_" + encoding
			t.Run(testName, func(t *testing.T) {
				// Create mock connection
				mockConn := &assemblyMockConn{encoding: encoding}
				mockConn.Reset()

				// Create test server
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
					queueSerializer, auditLogger, tenantResolver, mockStorage :=
					bssci.CreateTestServices(testLogger, nil)

				server := bssci.NewTestServer(testLogger, mockStorage, nil, 1,
					sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
					queueSerializer, auditLogger, tenantResolver)

				session := &bssci.Session{
					ID:                "test-session",
					BaseStationEUI:    bssci.TestBsEui04,
					Conn:              mockConn,
					Encoding:          encoding,
					HandshakeComplete: true,
				}

				// Setup message data
				data := handler.setupData()
				msg := &bssci.Message{
					Command: "statusRsp",
					OpId:    -1,
					Data:    data,
				}

				// Execute handler
				err := handler.callHandler(server, session, msg, data)
				require.NoError(t, err, "Handler should succeed")

				// Verify completion message sent
				require.Len(t, mockConn.sentMessages, 1,
					"Should send exactly one completion message")

				sentMsg := mockConn.sentMessages[0]

				// Verify exact completion command constant
				assert.Equal(t, handler.expectedCmd, sentMsg["command"],
					"Handler must emit exact completion constant")

				// Verify opId present (per BSSCI spec)
				assert.Contains(t, sentMsg, "opId", "Completion must include opId")
			})
		}
	}
}

// mockEventStore is a simple event store for testing that discards all events
type mockEventStore struct{}

func (m *mockEventStore) CreateEvent(_ context.Context, _ *models.SystemEvent) error {
	return nil
}

func (m *mockEventStore) GetEvents(_ context.Context, _ interfaces.SystemEventFilter) ([]*models.SystemEvent, error) {
	return []*models.SystemEvent{}, nil
}

func (m *mockEventStore) GetActiveAlerts(_ context.Context, _ interfaces.AlertFilter) ([]*models.SystemEvent, error) {
	return []*models.SystemEvent{}, nil
}

func (m *mockEventStore) GetEventStats(_ context.Context, _ string, _ time.Time) (*models.SystemEventStats, error) {
	return &models.SystemEventStats{}, nil
}

func (m *mockEventStore) RecordSCACIError(_ context.Context, _ int64, _ int64, _ string, _ int64, _ int, _ string) error {
	return nil
}
func (m *mockEventStore) CountEvents(_ context.Context, _ interfaces.SystemEventFilter) (int64, error) {
	return 0, nil
}
func (m *mockEventStore) CountActiveAlerts(_ context.Context, _ interfaces.AlertFilter) (int64, error) {
	return 0, nil
}

// TestComplexSCOperations tests complex SC-originated operations that require repository seeding.
// Per BSSCI §2.5-01: Every mandatory field must be populated when Service Center originates messages.
// This test inspects actual serializer output to verify field presence.
// newAssemblySession builds a registered, handshake-complete bidirectional
// session and its server/connection for SC-originated assembly tests in the
// given encoding ("json" or "msgpack").
func newAssemblySession(t *testing.T, encoding, sessionID string) (*bssci.Server, *assemblyMockConn, *bssci.Session) {
	t.Helper()

	mockConn := &assemblyMockConn{encoding: encoding}
	mockConn.Reset()

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver, mockStorage :=
		bssci.CreateTestServices(logger.NewNop(), nil)

	server := bssci.NewTestServer(logger.NewNop(), mockStorage, &mockEventStore{}, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver)

	session := &bssci.Session{
		ID:                sessionID,
		BaseStationEUI:    bssci.TestBsEui04,
		Conn:              mockConn,
		Encoding:          encoding,
		LastScOpId:        -1,
		HandshakeComplete: true,
		Bidirectional:     true,
	}

	server.RegisterSession(session)
	return server, mockConn, session
}

// assertNegativeOpID asserts the message carries a negative (SC-originated) opId.
// JSON decodes the field to float64, MessagePack to int64.
func assertNegativeOpID(t *testing.T, msg map[string]interface{}) {
	t.Helper()

	var opID int64
	switch v := msg["opId"].(type) {
	case int64:
		opID = v
	case float64:
		opID = int64(v)
	}
	assert.Less(t, opID, int64(0), "SC-originated opId must be negative")
}

// TestComplexSCOperations drives every SC-originated send helper in both JSON
// and MessagePack encodings, verifying the assembled wire message carries the
// mandatory fields per BSSCI §2.5-01.
func TestComplexSCOperations(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T, encoding string)
	}{
		{"SendAttachPropagate", testSendAttachPropagate},
		{"SendDetachPropagate", testSendDetachPropagate},
		{"SendDLDataQueue", testSendDLDataQueue},
		{"SendDLDataQueue_CounterDependent", testSendDLDataQueueCounterDependent},
		{"SendDLDataRevoke", testSendDLDataRevoke},
		{"SendDLRXStatusQuery", testSendDLRXStatusQuery},
		{"SendULDataTransmit", testSendULDataTransmit},
		{"SendVMActivate", testSendVMActivate},
		{"SendVMDeactivate", testSendVMDeactivate},
		{"SendVMStatus", testSendVMStatus},
		{"SendVMDownlinkData", testSendVMDownlinkData},
	}

	for _, encoding := range []string{"json", "msgpack"} {
		for _, tc := range cases {
			t.Run(tc.name+"_"+encoding, func(t *testing.T) {
				tc.run(t, encoding)
			})
		}
	}
}

func testSendAttachPropagate(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session")

	epEui := bssci.TestEpEui01
	nwkSnKey := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	shAddr := uint16(0x1234)

	err := server.SendAttachPropagate(session.ID, epEui, nwkSnKey, shAddr,
		true, 100, false, 0, false, false)
	require.NoError(t, err, "SendAttachPropagate should succeed")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	// Verify mandatory fields per BSSCI §3.6
	assert.Equal(t, mioty.CmdAttachPropagate, msg["command"],
		"Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assertNegativeOpID(t, msg)

	// Verify ALL 11 mandatory fields per BSSCI §2.5-01
	assert.Contains(t, msg, "epEui", "Must include epEui")
	assert.Contains(t, msg, "nwkSnKey", "Must include nwkSnKey (16 bytes)")
	assert.Contains(t, msg, "shAddr", "Must include shAddr")
	assert.Contains(t, msg, "bidi", "Must include bidi")
	assert.Contains(t, msg, "lastPacketCnt", "Must include lastPacketCnt")
	assert.Contains(t, msg, "dualChan", "Must include dualChan")
	assert.Contains(t, msg, "repetition", "Must include repetition")
	assert.Contains(t, msg, "wideCarrOff", "Must include wideCarrOff")
	assert.Contains(t, msg, "longBlkDist", "Must include longBlkDist")

	t.Logf("SendAttachPropagate (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

func testSendDetachPropagate(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session")

	epEui := bssci.TestEpEui01

	err := server.SendDetachPropagate(session.ID, epEui)
	require.NoError(t, err, "SendDetachPropagate should succeed")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	// Verify mandatory fields per BSSCI §3.7
	assert.Equal(t, mioty.CmdDetachPropagate, msg["command"],
		"Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assertNegativeOpID(t, msg)

	assert.Contains(t, msg, "epEui", "Must include epEui")

	t.Logf("SendDetachPropagate (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

func testSendDLDataQueue(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session")

	queId := int64(1000) //nolint:revive // queId matches MIOTY protocol field
	epEui := bssci.TestEpEui01
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	payloads := [][]byte{payload}
	packetCnt := []int64{}

	err := server.SendDLDataQueue(session.ID, epEui, payloads, queId,
		0, false, packetCnt, 0, false, false, false, false, 1)
	require.NoError(t, err, "SendDLDataQueue should succeed")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	// Verify mandatory fields per BSSCI §3.9
	assert.Equal(t, mioty.CmdDLDataQueue, msg["command"],
		"Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assertNegativeOpID(t, msg)

	assert.Contains(t, msg, "queId", "Must include queId")
	assert.Contains(t, msg, "epEui", "Must include epEui")
	assert.Contains(t, msg, "userData", "Must include userData (payload data)")

	// Verify cntDepend is present and false (BSSCI §5.12.1: required field)
	assert.Contains(t, msg, "cntDepend", "cntDepend must be present (required field)")
	assert.Equal(t, false, msg["cntDepend"], "cntDepend should be false in non-counter-dependent mode")
	assert.NotContains(t, msg, "packetCnt", "packetCnt should not be present when cntDepend=false")

	// BSSCI §5.12.1 declares userData as Numeric[m][n] — a 2D array
	// of m packet-counter entries by n payload bytes. For cntDepend=false
	// m=1 but the outer array MUST still be present (the "single user
	// data entry if cntDepend is false" line in the spec describes
	// cardinality, not dimensionality). The base station rejects the
	// message with BSSCI error 22 ("DL data queue message malformed")
	// when userData arrives as the 1D inner array directly. This
	// assertion pins the wrapping fix from b10daee6 AND verifies the
	// inner bytes still match the original payload — a shape-only
	// check would pass even if the payload were truncated or wrong.
	outer, ok := msg["userData"].([]interface{})
	require.True(t, ok,
		"userData must serialize to an outer slice (got %T) for cntDepend=false", msg["userData"])
	require.Len(t, outer, 1,
		"non-counter-dependent userData must contain exactly one inner payload entry (m=1)")
	assertInnerPayloadMatches(t, outer[0], payload)

	t.Logf("SendDLDataQueue (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

// assertInnerPayloadMatches verifies a single userData inner entry matches the
// expected payload bytes across the byte-slice / decoded-array shapes the JSON
// and MessagePack decoders produce.
func assertInnerPayloadMatches(t *testing.T, entry interface{}, expected []byte) {
	t.Helper()

	switch inner := entry.(type) {
	case []byte:
		require.Equal(t, expected, inner,
			"inner userData bytes must match the original payload")
	case []interface{}:
		require.Lenf(t, inner, len(expected),
			"inner userData length must match original payload (got %d, want %d)",
			len(inner), len(expected))
		for i, want := range expected {
			switch v := inner[i].(type) {
			case uint8:
				require.Equalf(t, want, v, "byte %d mismatch", i)
			case int64:
				require.Equalf(t, int64(want), v, "byte %d mismatch", i)
			case float64:
				require.Equalf(t, float64(want), v, "byte %d mismatch", i)
			default:
				t.Fatalf("inner userData[%d] has unexpected element type %T", i, v)
			}
		}
	default:
		t.Fatalf("inner userData entry has unexpected type %T (want []byte or []interface{})", entry)
	}
}

func testSendDLDataQueueCounterDependent(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session-cntdepend")

	queId := int64(2000) //nolint:revive // queId matches MIOTY protocol field
	epEui := bssci.TestEpEui01
	// Counter-dependent mode: must provide same number of payloads and packet counts
	payloads := [][]byte{
		{0x05, 0x06, 0x07, 0x08},
		{0x09, 0x0A, 0x0B, 0x0C},
		{0x0D, 0x0E, 0x0F, 0x10},
	}
	packetCnt := []int64{100, 200, 300} // Counter-dependent packet counts

	err := server.SendDLDataQueue(session.ID, epEui, payloads, queId,
		0, true, packetCnt, 0, false, false, false, false, 1)
	require.NoError(t, err, "SendDLDataQueue with cntDepend should succeed")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	assert.Equal(t, mioty.CmdDLDataQueue, msg["command"], "Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assert.Contains(t, msg, "queId", "Must include queId")
	assert.Contains(t, msg, "epEui", "Must include epEui")
	assert.Contains(t, msg, "userData", "Must include userData")

	// Verify counter-dependent fields ARE present when cntDepend=true (BSSCI §5.12)
	assert.Contains(t, msg, "cntDepend", "cntDepend should be present when true")
	assert.Equal(t, true, msg["cntDepend"], "cntDepend value should be true")
	assert.Contains(t, msg, "packetCnt", "packetCnt should be present when cntDepend=true")

	packetCntField, ok := msg["packetCnt"].([]interface{})
	require.True(t, ok, "packetCnt should be an array")
	require.Len(t, packetCntField, 3, "packetCnt should have 3 elements")
	// JSON unmarshals to float64, MessagePack to int64
	for i, expected := range []int64{100, 200, 300} {
		switch v := packetCntField[i].(type) {
		case int64:
			assert.Equal(t, expected, v)
		case float64:
			assert.Equal(t, float64(expected), v)
		default:
			t.Fatalf("packetCnt[%d] has unexpected type %T", i, v)
		}
	}

	t.Logf("SendDLDataQueue counter-dependent (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

func testSendDLDataRevoke(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session")

	queId := uint64(1000) //nolint:revive // queId matches MIOTY protocol field
	epEui := bssci.TestEpEui01

	err := server.SendDLDataRevoke(session.ID, epEui, queId)
	require.NoError(t, err, "SendDLDataRevoke should succeed")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	// Verify mandatory fields per BSSCI §3.10
	assert.Equal(t, mioty.CmdDLDataRevoke, msg["command"],
		"Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assertNegativeOpID(t, msg)

	assert.Contains(t, msg, "queId", "Must include queId")
	assert.Contains(t, msg, "epEui", "Must include epEui")

	t.Logf("SendDLDataRevoke (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

func testSendDLRXStatusQuery(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session")

	epEui := bssci.TestEpEui01

	err := server.SendDLRXStatusQuery(session.ID, epEui)
	require.NoError(t, err, "SendDLRXStatusQuery should succeed")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	// Verify mandatory fields per BSSCI §3.11
	assert.Equal(t, mioty.CmdDLRxStatusQuery, msg["command"],
		"Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assertNegativeOpID(t, msg)

	assert.Contains(t, msg, "epEui", "Must include epEui")

	t.Logf("SendDLRXStatusQuery (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

func testSendULDataTransmit(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session")

	epEui := bssci.TestEpEui01
	nwkSnKey := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	shAddr := uint16(0x1234)
	packetCnt := uint32(100)
	userData := []byte{0x01, 0x02, 0x03}

	opId, err := server.SendULDataTransmit(session.ID, epEui, nwkSnKey, shAddr, packetCnt, userData, "", 0) //nolint:revive // opId matches protocol

	require.NoError(t, err, "SendULDataTransmit should succeed")
	require.NotZero(t, opId, "SendULDataTransmit should return non-zero opId")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	// Verify mandatory fields per BSSCI §3.8
	assert.Equal(t, mioty.CmdULDataTransmit, msg["command"],
		"Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assertNegativeOpID(t, msg)

	assert.Contains(t, msg, "epEui", "Must include epEui")
	assert.Contains(t, msg, "nwkSnKey", "Must include nwkSnKey")
	assert.Contains(t, msg, "shAddr", "Must include shAddr")
	assert.Contains(t, msg, "packetCnt", "Must include packetCnt")
	assert.Contains(t, msg, "userData", "Must include userData")

	t.Logf("SendULDataTransmit (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

func testSendVMActivate(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session")

	epEui := bssci.TestEpEui01
	macType := uint8(1)

	err := server.SendVMActivate(session.ID, epEui, macType)
	require.NoError(t, err, "SendVMActivate should succeed")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	// Verify mandatory fields per BSSCI VM operations
	assert.Equal(t, mioty.CmdVMActivate, msg["command"],
		"Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assertNegativeOpID(t, msg)

	assert.Contains(t, msg, "epEui", "Must include epEui")
	assert.Contains(t, msg, "macType", "Must include macType")

	t.Logf("SendVMActivate (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

func testSendVMDeactivate(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session")

	epEui := bssci.TestEpEui01
	macType := uint8(1)

	err := server.SendVMDeactivate(session.ID, epEui, macType)
	require.NoError(t, err, "SendVMDeactivate should succeed")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	assert.Equal(t, mioty.CmdVMDeactivate, msg["command"],
		"Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assertNegativeOpID(t, msg)

	assert.Contains(t, msg, "epEui", "Must include epEui")
	assert.Contains(t, msg, "macType", "Must include macType")

	t.Logf("SendVMDeactivate (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

func testSendVMStatus(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session")

	epEui := bssci.TestEpEui01

	err := server.SendVMStatus(session.ID, epEui)
	require.NoError(t, err, "SendVMStatus should succeed")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	assert.Equal(t, mioty.CmdVMStatus, msg["command"],
		"Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assertNegativeOpID(t, msg)

	assert.Contains(t, msg, "epEui", "Must include epEui")

	t.Logf("SendVMStatus (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

func testSendVMDownlinkData(t *testing.T, encoding string) {
	server, mockConn, session := newAssemblySession(t, encoding, "test-session")

	epEui := bssci.TestEpEui01
	macType := uint8(1)
	userData := []byte{0x01, 0x02, 0x03}

	err := server.SendVMDownlinkData(session.ID, epEui, macType, userData)
	require.NoError(t, err, "SendVMDownlinkData should succeed")

	require.Len(t, mockConn.sentMessages, 1, "Should send exactly one message")
	msg := mockConn.sentMessages[0]

	assert.Equal(t, mioty.CmdVMDLData, msg["command"],
		"Must use exact command constant")
	assert.Contains(t, msg, "opId", "Message must have opId")
	assertNegativeOpID(t, msg)

	assert.Contains(t, msg, "epEui", "Must include epEui")
	assert.Contains(t, msg, "macType", "Must include macType")
	assert.Contains(t, msg, "userData", "Must include userData")

	t.Logf("SendVMDownlinkData (%s) emitted fields: %v", encoding, getFieldNames(msg))
}

// getFieldNames returns a sorted list of field names from a message for logging
func getFieldNames(msg map[string]interface{}) []string {
	names := make([]string, 0, len(msg))
	for k := range msg {
		names = append(names, k)
	}
	return names
}
