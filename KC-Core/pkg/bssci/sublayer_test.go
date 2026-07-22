package bssci_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sublayerMockConn implements net.Conn for testing sublayer error handling
type sublayerMockConn struct {
	net.Conn
	sentMessages []map[string]interface{}
}

func (m *sublayerMockConn) Write(b []byte) (n int, err error) {
	var msg map[string]interface{}
	if jsonErr := json.Unmarshal(b, &msg); jsonErr == nil {
		m.sentMessages = append(m.sentMessages, msg)
	}
	return len(b), nil
}

func (m *sublayerMockConn) Reset() {
	m.sentMessages = nil
}

func (m *sublayerMockConn) Close() error                       { return nil }
func (m *sublayerMockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *sublayerMockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *sublayerMockConn) SetDeadline(_ time.Time) error      { return nil }
func (m *sublayerMockConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *sublayerMockConn) SetWriteDeadline(_ time.Time) error { return nil }
func (m *sublayerMockConn) Read(_ []byte) (n int, err error)   { return 0, nil }

// TestBSSCI_4_01_unsupported_sublayer verifies BSSCI §4-01:
// When an unsupported sublayer message is received, the server sends an error
// but keeps the session alive (does not close the connection)
func TestBSSCI_4_01_unsupported_sublayer(t *testing.T) {
	testLogger := logger.NewNop()

	mockConn := &sublayerMockConn{}

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver, mockStorage :=
		bssci.CreateTestServices(testLogger, nil)

	server := bssci.NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver)

	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			ID:                "test-session",
			BaseStationEUI:    bssci.TestBsEui01,
			Encoding:          "json",
			HandshakeComplete: true,
			DbSessionID:       1,
		},
		Conn: mockConn,
	}

	server.RegisterSession(session)

	// Send unsupported sublayer command
	invalidMsg := &bssci.Message{
		Command: "unknownSublayerCommand",
		OpId:    1,
		Data:    map[string]interface{}{},
	}

	_ = server.CallHandleMessage(session, invalidMsg, invalidMsg.Data.(map[string]interface{}))

	// Verify error frame sent
	require.GreaterOrEqual(t, len(mockConn.sentMessages), 1,
		"Should send error for unsupported sublayer per §4-01")

	errorMsg := mockConn.sentMessages[0]
	assert.Equal(t, "error", errorMsg["command"],
		"Server must send error frame for unsupported sublayer")

	// Clear messages and prove session stays alive by sending valid message
	mockConn.sentMessages = nil

	_, err := server.SendStatusRequest(session)
	require.NoError(t, err, "Subsequent valid message must succeed per §4-01")
	require.Len(t, mockConn.sentMessages, 1, "Valid message should send")
	assert.Equal(t, mioty.CmdStatus, mockConn.sentMessages[0]["command"],
		"Session must remain operational after sublayer error")
}
