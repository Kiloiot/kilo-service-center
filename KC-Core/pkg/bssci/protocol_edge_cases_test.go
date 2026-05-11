package bssci_test

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestProtocolEdgeCases verifies BSSCI §2.4 and §2.5 edge case handling

type edgeMockConn struct {
	net.Conn
	sentMessages []map[string]interface{}
	encoding     string // "json" or "msgpack"
}

func (m *edgeMockConn) Write(b []byte) (n int, err error) {
	var msg map[string]interface{}

	// Support both JSON and MessagePack encodings
	if m.encoding == "msgpack" {
		err = msgpack.Unmarshal(b, &msg)
	} else {
		err = json.Unmarshal(b, &msg)
	}

	if err == nil {
		m.sentMessages = append(m.sentMessages, msg)
	}
	return len(b), nil
}

func (m *edgeMockConn) Reset() {
	m.sentMessages = nil
}

func (m *edgeMockConn) Close() error                       { return nil }
func (m *edgeMockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *edgeMockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *edgeMockConn) SetDeadline(_ time.Time) error      { return nil }
func (m *edgeMockConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *edgeMockConn) SetWriteDeadline(_ time.Time) error { return nil }
func (m *edgeMockConn) Read(_ []byte) (n int, err error)   { return 0, nil }

// TestBSSCI_2_4_01_forward_compat_extra_fields verifies that extra fields are ignored
// per BSSCI §2.4-01 (forward compatibility requirement)
func TestBSSCI_2_4_01_forward_compat_extra_fields(t *testing.T) {
	for _, encoding := range []string{"json", "msgpack"} {
		t.Run(encoding, func(t *testing.T) {
			testLogger := logger.NewNop()
			mockConn := &edgeMockConn{encoding: encoding}
			mockConn.Reset()

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, mockStorage :=
				bssci.CreateTestServices(testLogger, nil)

			server := bssci.NewTestServer(testLogger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)
			server.SetConfig(&bssci.Config{
				ServiceCenterEUI: bssci.TestScEui01,
				Vendor:           "TestVendor",
				Model:            "TestModel",
				Name:             "TestSC",
				SoftwareVersion:  "1.0.0",
			})
			server.SetConfig(&bssci.Config{
				ServiceCenterEUI: bssci.TestScEui01,
				Vendor:           "TestVendor",
				Model:            "TestModel",
				Name:             "TestSC",
				SoftwareVersion:  "1.0.0",
			})
			server.RegisterHandlers()
			server.RegisterHandlers()
			server.RegisterHandlers() // Register command handlers for CallHandleMessage

			session := &bssci.Session{
				ID:                "test-session",
				BaseStationEUI:    bssci.TestBsEui01,
				Conn:              mockConn,
				Encoding:          encoding,
				HandshakeComplete: true,
			}

			// statusRsp with extra unknown field "futureFeature"
			data := map[string]interface{}{
				"code":          int64(0),
				"message":       "operational",
				"time":          time.Now().UnixNano(),
				"dutyCycle":     float64(0.5),
				"futureFeature": "should be ignored",
			}

			msg := &bssci.Message{
				Command: "statusRsp",
				OpId:    -1,
				Data:    data,
			}

			err := server.CallHandleStatusResponse(session, msg, data)

			// Handler should NOT send error for extra field
			assert.NoError(t, err, "Extra fields should be ignored per §2.4-01")

			// Verify no error message was sent
			for _, sentMsg := range mockConn.sentMessages {
				if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
					t.Fatal("Should not send error for extra field (forward compatibility)")
				}
			}
		})
	}
}

// TestBSSCI_2_4_02_optional_fields_default verifies optional fields get defaults
// per BSSCI §2.4-02 (optional fields must have sensible defaults)
func TestBSSCI_2_4_02_optional_fields_default(t *testing.T) {
	for _, encoding := range []string{"json", "msgpack"} {
		t.Run(encoding, func(t *testing.T) {
			testLogger := logger.NewNop()
			mockConn := &edgeMockConn{encoding: encoding}
			mockConn.Reset()

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, mockStorage :=
				bssci.CreateTestServices(testLogger, nil)

			server := bssci.NewTestServer(testLogger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)

			// Set config so handleConnect doesn't panic
			config := &bssci.Config{
				ServiceCenterEUI: bssci.TestScEui01,
				Vendor:           "test-vendor",
				Model:            "test-model",
				Name:             "test-sc",
				SoftwareVersion:  "1.0.0",
			}
			server.SetConfig(config)

			session := &bssci.Session{
				ID:       "test-session",
				Conn:     mockConn,
				Encoding: encoding,
			}

			// Connect without optional version field
			data := map[string]interface{}{
				"bsEui": bssci.TestBsEui01,
				"bidi":  true,
			}

			msg := &bssci.Message{
				Command: "con",
				OpId:    0,
				Data:    data,
			}

			_ = server.CallHandleConnect(session, msg, data)

			// VERIFY DEFAULT WAS APPLIED (not just absence of error)
			assert.Equal(t, mioty.MIOTYProtocolVersion, session.NegotiatedVersion,
				"Server must populate default version per §2.4-02")

			// Should not send error about missing version
			for _, sentMsg := range mockConn.sentMessages {
				if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
					if errMsg, hasMsg := sentMsg["message"].(string); hasMsg {
						assert.NotContains(t, errMsg, "version",
							"Optional version should default per §2.4-02")
					}
				}
			}
		})
	}
}

// TestBSSCI_2_5_02_reject_malformed_values verifies invalid field values are rejected
// per BSSCI §2.5-02 (field value validation)
func TestBSSCI_2_5_02_reject_malformed_values(t *testing.T) {
	testLogger := logger.NewNop()

	tests := []struct {
		name          string
		data          map[string]interface{}
		expectedError bool
	}{
		{
			name: "code as string instead of number",
			data: map[string]interface{}{
				"code":      "not_a_number",
				"message":   "operational",
				"time":      time.Now().UnixNano(),
				"dutyCycle": float64(0.5),
			},
			expectedError: true,
		},
		{
			name: "time as string instead of number",
			data: map[string]interface{}{
				"code":      int64(0),
				"message":   "operational",
				"time":      "not_a_timestamp",
				"dutyCycle": float64(0.5),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		for _, encoding := range []string{"json", "msgpack"} {
			testName := tt.name + "_" + encoding
			t.Run(testName, func(t *testing.T) {
				mockConn := &edgeMockConn{encoding: encoding}
				mockConn.Reset()

				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
					queueSerializer, auditLogger, tenantResolver, mockStorage :=
					bssci.CreateTestServices(testLogger, nil)

				server := bssci.NewTestServer(testLogger, mockStorage, nil, 1,
					sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
					queueSerializer, auditLogger, tenantResolver)

				session := &bssci.Session{
					ID:                "test-session",
					BaseStationEUI:    bssci.TestBsEui01,
					Conn:              mockConn,
					Encoding:          encoding,
					HandshakeComplete: true,
				}

				msg := &bssci.Message{
					Command: "statusRsp",
					OpId:    -1,
					Data:    tt.data,
				}

				_ = server.CallHandleMessage(session, msg, tt.data)

				if tt.expectedError {
					// Should send error for malformed value
					var errorMsg map[string]interface{}
					for _, sentMsg := range mockConn.sentMessages {
						if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
							errorMsg = sentMsg
							break
						}
					}

					require.NotNil(t, errorMsg, "Should send error for malformed value per §2.5-02")
				}
			})
		}
	}
}

// TestBSSCI_2_5_03_numeric_field_types verifies numeric fields are properly typed
// per BSSCI §2.5-03 (type enforcement)
func TestBSSCI_2_5_03_numeric_field_types(t *testing.T) {
	for _, encoding := range []string{"json", "msgpack"} {
		t.Run(encoding, func(t *testing.T) {
			testLogger := logger.NewNop()
			mockConn := &edgeMockConn{encoding: encoding}
			mockConn.Reset()

			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver, mockStorage :=
				bssci.CreateTestServices(testLogger, nil)

			server := bssci.NewTestServer(testLogger, mockStorage, nil, 1,
				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
				queueSerializer, auditLogger, tenantResolver)
			server.RegisterHandlers() // Register command handlers for CallHandleMessage

			session := &bssci.Session{
				ID:                "test-session",
				BaseStationEUI:    bssci.TestBsEui01,
				Conn:              mockConn,
				Encoding:          encoding,
				HandshakeComplete: true,
			}

			// All fields are properly typed
			data := map[string]interface{}{
				"code":      int64(0),
				"message":   "operational",
				"time":      int64(time.Now().UnixNano()),
				"dutyCycle": float64(0.5),
			}

			msg := &bssci.Message{
				Command: "statusRsp",
				OpId:    -1,
				Data:    data,
			}

			err := server.CallHandleStatusResponse(session, msg, data)

			// Should not return error for properly typed fields
			assert.NoError(t, err, "Properly typed fields should be accepted per §2.5-03")

			// Should send completion, not error
			var errorMsg map[string]interface{}
			for _, sentMsg := range mockConn.sentMessages {
				if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
					errorMsg = sentMsg
					break
				}
			}

			assert.Nil(t, errorMsg, "Should not send error for valid types")

			// Verify completion was sent
			require.GreaterOrEqual(t, len(mockConn.sentMessages), 1, "Should send completion")
			assert.Equal(t, mioty.CmdStatusComplete, mockConn.sentMessages[0]["command"])
		})
	}
}

// TestNormalizationSentinelErrors verifies normalization sentinel error paths:
// Sentinel errors enable clean errors.Is() detection and proper catalog token dispatch.
//
// Tests full code path: normalizePayload → server error mapping (lines 912-922) → sendError → catalog token emission.
// Verifies both: (a) correct sentinel triggered, (b) correct catalog token sent to base station.
func TestNormalizationSentinelErrors(t *testing.T) {
	// Setup common test infrastructure
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver, mockStorage :=
		bssci.CreateTestServices(testLogger, nil)

	server := bssci.NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.RegisterHandlers()

	t.Run("MandatoryFieldMissing", func(t *testing.T) {
		mockConn := &edgeMockConn{encoding: "json"}
		mockConn.Reset()

		session := &bssci.Session{
			ID:                "test-session",
			BaseStationEUI:    bssci.TestBsEui01,
			Conn:              mockConn,
			Encoding:          "json",
			HandshakeComplete: true,
		}

		// statusRsp missing mandatory "code" field
		data := map[string]interface{}{
			"message": "operational",
			"time":    time.Now().UnixNano(),
			// "code" field intentionally omitted - triggers ErrMandatoryFieldMissing
		}

		msg := &bssci.Message{
			Command: "statusRsp",
			OpId:    -1,
			Data:    data,
		}

		err := server.CallHandleMessage(session, msg, data)

		// handleMessage doesn't return error, sends to BS instead
		assert.NoError(t, err, "handleMessage should not return error for normalization failures")

		// Verify error frame sent to base station with correct catalog token
		require.Len(t, mockConn.sentMessages, 1, "Should send exactly one error frame")
		assert.Equal(t, "error", mockConn.sentMessages[0]["command"], "Should send error command")

		errorMessage, ok := mockConn.sentMessages[0]["message"].(string)
		require.True(t, ok, "Error message should be string")
		assert.Contains(t, errorMessage, "Mandatory field missing",
			"Error message should contain errMandatoryFieldMissing catalog token text")
	})

	t.Run("InvalidFieldType", func(t *testing.T) {
		mockConn := &edgeMockConn{encoding: "json"}
		mockConn.Reset()

		session := &bssci.Session{
			ID:                "test-session",
			BaseStationEUI:    bssci.TestBsEui01,
			Conn:              mockConn,
			Encoding:          "json",
			HandshakeComplete: true,
		}

		// statusRsp with string "code" instead of int64
		data := map[string]interface{}{
			"code":    "not_a_number", // Wrong type - triggers ErrInvalidFieldType
			"message": "operational",
			"time":    time.Now().UnixNano(),
		}

		msg := &bssci.Message{
			Command: "statusRsp",
			OpId:    -1,
			Data:    data,
		}

		err := server.CallHandleMessage(session, msg, data)
		assert.NoError(t, err, "handleMessage should not return error")

		require.Len(t, mockConn.sentMessages, 1)
		assert.Equal(t, "error", mockConn.sentMessages[0]["command"])

		errorMessage, ok := mockConn.sentMessages[0]["message"].(string)
		require.True(t, ok)
		assert.Contains(t, errorMessage, "Invalid field type",
			"Error message should contain errInvalidFieldType catalog token text")
	})

	t.Run("ResponseExpRequiresDlOpen", func(t *testing.T) {
		mockConn := &edgeMockConn{encoding: "json"}
		mockConn.Reset()

		session := &bssci.Session{
			ID:                "test-session",
			BaseStationEUI:    bssci.TestBsEui01,
			Conn:              mockConn,
			Encoding:          "json",
			HandshakeComplete: true,
		}

		// ulData with responseExp=true but dlOpen=false - conditional rule violation
		data := map[string]interface{}{
			"epEui":       bssci.TestBsEui01,
			"rxTime":      time.Now().UnixNano(),
			"packetCnt":   uint32(100),
			"userData":    []byte{0x01, 0x02, 0x03},
			"snr":         10.5,
			"rssi":        -80.0,
			"eqSnr":       9.5,
			"responseExp": true,  // Expects downlink response
			"dlOpen":      false, // But dlOpen is false - triggers ErrResponseExpRequiresDlOpen
			"dlAck":       false, // Mandatory field per BSSCI §5.11.1
		}

		msg := &bssci.Message{
			Command: "ulData",
			OpId:    1,
			Data:    data,
		}

		err := server.CallHandleMessage(session, msg, data)
		assert.NoError(t, err, "handleMessage should not return error")

		require.Len(t, mockConn.sentMessages, 1)
		assert.Equal(t, "error", mockConn.sentMessages[0]["command"])

		errorMessage, ok := mockConn.sentMessages[0]["message"].(string)
		require.True(t, ok)
		assert.Contains(t, errorMessage, "responseExp flag requires dlOpen flag",
			"Error message should contain exact errResponseExpRequiresDlOpen catalog token text from errors_catalog.go:1084")
	})

	t.Run("GenericConditionalRuleFailed", func(t *testing.T) {
		// Note: This test would verify the generic fallback case if we had other conditional rules.
		// Currently, ResponseExpRequiresDlOpen is the only specific conditional sentinel.
		// This test documents the pattern for future conditional rules.
		mockConn := &edgeMockConn{encoding: "json"}
		mockConn.Reset()

		session := &bssci.Session{
			ID:                "test-session",
			BaseStationEUI:    bssci.TestBsEui01,
			Conn:              mockConn,
			Encoding:          "json",
			HandshakeComplete: true,
		}

		// For now, use same responseExp scenario but verify it doesn't match generic token
		// In future, add other conditional rules here
		data := map[string]interface{}{
			"epEui":       bssci.TestBsEui01,
			"rxTime":      time.Now().UnixNano(),
			"packetCnt":   uint32(100),
			"userData":    []byte{0x01, 0x02, 0x03},
			"snr":         10.5,
			"rssi":        -80.0,
			"eqSnr":       9.5,
			"responseExp": true,
			"dlOpen":      false,
			"dlAck":       false, // Mandatory field per BSSCI §5.11.1
		}

		msg := &bssci.Message{
			Command: "ulData",
			OpId:    1,
			Data:    data,
		}

		err := server.CallHandleMessage(session, msg, data)
		assert.NoError(t, err)

		require.Len(t, mockConn.sentMessages, 1)
		errorMessage, ok := mockConn.sentMessages[0]["message"].(string)
		require.True(t, ok)

		// Verify it matched SPECIFIC sentinel, not generic
		assert.Contains(t, errorMessage, "responseExp flag requires dlOpen flag",
			"Should use specific sentinel errResponseExpRequiresDlOpen, not generic errConditionalRuleFailed")
	})
}

// TestConConditionalRules verifies MIOTY radio spec §3.6.5.3 session resume mutual presence validation
// Tests the ConditionalRules added to con command in message_metadata.go:669-688
func TestConConditionalRules(t *testing.T) {
	testLogger := logger.NewNop()

	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver, mockStorage :=
		bssci.CreateTestServices(testLogger, nil)

	server := bssci.NewTestServer(testLogger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
		queueSerializer, auditLogger, tenantResolver)
	server.SetConfig(&bssci.Config{
		ServiceCenterEUI: bssci.TestScEui01,
		Vendor:           "TestVendor",
		Model:            "TestModel",
		Name:             "TestSC",
		SoftwareVersion:  "1.0.0",
	})

	t.Run("NewConnect_NoResumeFields_Success", func(t *testing.T) {
		mockConn := &edgeMockConn{encoding: "json"}
		mockConn.Reset()
		server.RegisterHandlers()

		session := &bssci.Session{
			ID:                "test-session",
			BaseStationEUI:    bssci.TestBsEui01,
			Conn:              mockConn,
			Encoding:          "json",
			HandshakeComplete: false, // Fresh connection
		}

		// New connection without resume fields - should succeed
		data := map[string]interface{}{
			"version":  "1.0.0",
			"bsEui":    bssci.TestBsEui01,
			"bidi":     true,
			"snBsUuid": []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		}

		msg := &bssci.Message{
			Command: "con",
			OpId:    0,
			Data:    data,
		}

		err := server.CallHandleMessage(session, msg, data)
		assert.NoError(t, err, "handleMessage should not return error for new connect without resume fields")

		// Should send conRsp (not error)
		require.GreaterOrEqual(t, len(mockConn.sentMessages), 1)
		if mockConn.sentMessages[0]["command"] == "error" {
			t.Logf("connect error payload: %+v", mockConn.sentMessages[0])
		}
		assert.Equal(t, "conRsp", mockConn.sentMessages[0]["command"])
	})

	t.Run("Resume_BothFields_Success", func(t *testing.T) {
		mockConn := &edgeMockConn{encoding: "json"}
		mockConn.Reset()
		server.RegisterHandlers()

		session := &bssci.Session{
			ID:                "test-session",
			BaseStationEUI:    bssci.TestBsEui01,
			Conn:              mockConn,
			Encoding:          "json",
			HandshakeComplete: false,
		}

		// Resume with BOTH snBsOpId and snScOpId - mutual presence satisfied
		data := map[string]interface{}{
			"version":  "1.0.0",
			"bsEui":    bssci.TestBsEui01,
			"bidi":     true,
			"snBsUuid": []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			"snBsOpId": int64(100), // BS counter present
			"snScOpId": int64(-50), // SC counter present
		}

		msg := &bssci.Message{
			Command: "con",
			OpId:    0,
			Data:    data,
		}

		err := server.CallHandleMessage(session, msg, data)
		assert.NoError(t, err, "handleMessage should not return error when both resume fields present")

		// Should send conRsp (not error)
		require.GreaterOrEqual(t, len(mockConn.sentMessages), 1)
		if mockConn.sentMessages[0]["command"] == "error" {
			t.Logf("connect error payload: %+v", mockConn.sentMessages[0])
		}
		assert.Equal(t, "conRsp", mockConn.sentMessages[0]["command"])
	})

	t.Run("Resume_OnlySnBsOpId_FailsConditionalRule", func(t *testing.T) {
		mockConn := &edgeMockConn{encoding: "json"}
		mockConn.Reset()
		server.RegisterHandlers()

		session := &bssci.Session{
			ID:                "test-session",
			BaseStationEUI:    bssci.TestBsEui01,
			Conn:              mockConn,
			Encoding:          "json",
			HandshakeComplete: false,
		}

		// Resume with ONLY snBsOpId - violates mutual presence rule
		data := map[string]interface{}{
			"version":  "1.0.0",
			"bsEui":    bssci.TestBsEui01,
			"bidi":     true,
			"snBsUuid": []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			"snBsOpId": int64(100), // BS counter present
			// snScOpId MISSING - triggers conditional rule failure
		}

		msg := &bssci.Message{
			Command: "con",
			OpId:    0,
			Data:    data,
		}

		err := server.CallHandleMessage(session, msg, data)
		assert.NoError(t, err, "handleMessage should not return error (sends error frame instead)")

		// Should send error frame (not conRsp)
		require.Len(t, mockConn.sentMessages, 1)
		assert.Equal(t, "error", mockConn.sentMessages[0]["command"])

		errorMessage, ok := mockConn.sentMessages[0]["message"].(string)
		require.True(t, ok)
		assert.Equal(t, "Conditional field requirement failed", errorMessage,
			"Error response should use catalog message without leaking rule text")
	})

	t.Run("Resume_OnlySnScOpId_FailsConditionalRule", func(t *testing.T) {
		mockConn := &edgeMockConn{encoding: "json"}
		mockConn.Reset()
		server.RegisterHandlers()

		session := &bssci.Session{
			ID:                "test-session",
			BaseStationEUI:    bssci.TestBsEui01,
			Conn:              mockConn,
			Encoding:          "json",
			HandshakeComplete: false,
		}

		// Resume with ONLY snScOpId - violates mutual presence rule
		data := map[string]interface{}{
			"version":  "1.0.0",
			"bsEui":    bssci.TestBsEui01,
			"bidi":     true,
			"snBsUuid": []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			"snScOpId": int64(-50), // SC counter present
			// snBsOpId MISSING - triggers conditional rule failure
		}

		msg := &bssci.Message{
			Command: "con",
			OpId:    0,
			Data:    data,
		}

		err := server.CallHandleMessage(session, msg, data)
		assert.NoError(t, err, "handleMessage should not return error (sends error frame instead)")

		// Should send error frame (not conRsp)
		require.Len(t, mockConn.sentMessages, 1)
		assert.Equal(t, "error", mockConn.sentMessages[0]["command"])

		errorMessage, ok := mockConn.sentMessages[0]["message"].(string)
		require.True(t, ok)
		assert.Equal(t, "Conditional field requirement failed", errorMessage,
			"Error response should use catalog message without leaking rule text")
	})
}

// TestTypeBytesIntegration verifies TypeBytes field conversion in normalizePayload.
//
// BSSCI §5.6.1 requires 'nonce' and 'sign' fields to be 4-byte arrays (TypeBytes).
// MessagePack decoding can produce any of the 12 numeric types, so normalizePayload
// must coerce them to []byte with range validation [0, 255].
//
// This test validates:
// - All 12 MessagePack numeric types convert correctly when in range
// - Out-of-range values (>255, <0) are rejected
// - Fractional floats are rejected per spec
// - Native []byte passes through with length validation
// - Wrong types (strings, etc.) are rejected
//
// Tests normalizePayload directly via CallNormalizePayload helper, scoped to
// just field conversion logic without attach handler provisioning dependencies.
func TestTypeBytesIntegration(t *testing.T) {
	testLogger := logger.NewNop()
	ctx := context.Background()

	tests := []struct {
		name         string
		nonce        interface{}
		expectError  bool
		errorPattern string // Expected substring in error message
	}{
		// Valid conversions (all 12 MessagePack numeric types in range [0, 255])
		{"uint8_array", []interface{}{uint8(1), uint8(2), uint8(3), uint8(4)}, false, ""},
		{"int8_array", []interface{}{int8(100), int8(50), int8(25), int8(10)}, false, ""},
		{"uint16_array", []interface{}{uint16(200), uint16(150), uint16(100), uint16(50)}, false, ""},
		{"int16_array", []interface{}{int16(150), int16(100), int16(50), int16(25)}, false, ""},
		{"uint32_array", []interface{}{uint32(99), uint32(88), uint32(77), uint32(66)}, false, ""},
		{"int32_array", []interface{}{int32(77), int32(66), int32(55), int32(44)}, false, ""},
		{"uint64_array", []interface{}{uint64(255), uint64(200), uint64(150), uint64(100)}, false, ""},
		{"int64_array", []interface{}{int64(1), int64(2), int64(3), int64(4)}, false, ""},
		{"int_array", []interface{}{int(128), int(64), int(32), int(16)}, false, ""},
		{"uint_array", []interface{}{uint(64), uint(32), uint(16), uint(8)}, false, ""},
		{"float32_array", []interface{}{float32(33.0), float32(44.0), float32(55.0), float32(66.0)}, false, ""},
		{"float64_array", []interface{}{float64(211.0), float64(100.0), float64(50.0), float64(25.0)}, false, ""},
		{"native_bytes", []byte{0x01, 0x02, 0x03, 0x04}, false, ""},

		// Length mismatch errors (need exactly 4 bytes per message_metadata.go:208)
		{"wrong_length_too_short", []byte{0x01, 0x02}, true, "expected 4 bytes"},
		{"wrong_length_too_long", []interface{}{1, 2, 3, 4, 5, 6}, true, "expected 4 bytes"},

		// Numeric overflow errors (values outside [0, 255] range)
		{"uint16_overflow", []interface{}{1, 2, uint16(300), 4}, true, "out of byte range"},
		{"int32_overflow", []interface{}{int32(1), int32(2), int32(500), int32(4)}, true, "out of byte range"},
		{"int64_negative", []interface{}{int64(1), int64(-100), int64(3), int64(4)}, true, "out of byte range"},

		// Type errors (non-numeric, non-byte values)
		{"string_type", "not_bytes", true, "expected byte array"},
		{"float_array_fractional", []interface{}{1.5, 2.7, 3.9, 4.2}, true, "out of byte range"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Minimal attach payload for normalization
			data := map[string]interface{}{
				"epEui":       bssci.TestBsEui01,
				"rxTime":      int64(1700000000000000000),
				"attachCnt":   uint32(1),
				"snr":         10.5,
				"rssi":        -80.0,
				"nonce":       tt.nonce, // ← Field under test
				"sign":        []byte{0xAA, 0xBB, 0xCC, 0xDD},
				"dualChan":    false,
				"repetition":  false,
				"wideCarrOff": false,
				"longBlkDist": false,
			}

			normalized, err := bssci.CallNormalizePayload(ctx, testLogger, "att", data)

			if tt.expectError {
				require.Error(t, err, "Expected error for %s", tt.name)
				if tt.errorPattern != "" {
					assert.Contains(t, err.Error(), tt.errorPattern,
						"Error for %s should contain pattern '%s'", tt.name, tt.errorPattern)
				}
			} else {
				require.NoError(t, err, "Should not error for valid %s", tt.name)
				// Verify nonce was converted to []byte correctly
				nonceBytes, ok := normalized["nonce"].([]byte)
				require.True(t, ok, "nonce should be []byte after normalization")
				assert.Len(t, nonceBytes, 4, "nonce should be 4 bytes")
			}
		})
	}
}

// TestNormalizeResponseCommandsWithUnknownFields verifies BSSCI §2.4 forward compatibility.
// Only BS->SC and bidirectional commands should be normalized.
// SC->BS responses are our own outbound messages and should NOT be normalized.
func TestNormalizeResponseCommandsWithUnknownFields(t *testing.T) {
	tests := []struct {
		command string
		payload map[string]interface{}
		desc    string
	}{
		{
			command: mioty.CmdPingResponse,
			payload: map[string]interface{}{
				"unknownField": "value",
				"extraField":   123,
			},
			desc: "Ping response with unknown fields",
		},
		// Note: CmdDLDataResultResponse and CmdDLRxStatusResponse are DirectionSCtoBS
		// (SC's own responses), so they should NOT be normalized per BSSCI §2.4.
		// Only BS→SC and bidirectional inbound messages should be normalized.
	}

	for _, tt := range tests {
		for _, encoding := range []string{"json", "msgpack"} {
			t.Run(tt.desc+"_"+encoding, func(t *testing.T) {
				// Use observing logger to verify WARN logs for unknown fields
				observedCore, observedLogs := observer.New(zap.WarnLevel)
				testLogger := logger.FromZap(zap.New(observedCore))
				mockConn := &edgeMockConn{encoding: encoding}
				mockConn.Reset()

				sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
					queueSerializer, auditLogger, tenantResolver, mockStorage :=
					bssci.CreateTestServices(testLogger, nil)

				server := bssci.NewTestServer(testLogger, mockStorage, nil, 1,
					sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster,
					queueSerializer, auditLogger, tenantResolver)
				server.RegisterHandlers() // Register command handlers for CallHandleMessage

				session := &bssci.Session{
					ID:                "test-session",
					BaseStationEUI:    bssci.TestBsEui01,
					Conn:              mockConn,
					Encoding:          encoding,
					HandshakeComplete: true,
				}

				msg := &bssci.Message{
					Command: tt.command,
					OpId:    1,
					Data:    tt.payload,
				}
				// Route through handleMessage to trigger normalization
				err := server.CallHandleMessage(session, msg, tt.payload)

				// Handler should NOT return error for unknown fields (forward compatibility)
				assert.NoError(t, err, "%s should not error on unknown fields per §2.4-01", tt.desc)

				// Verify no error message was sent
				for _, sentMsg := range mockConn.sentMessages {
					if cmd, ok := sentMsg["command"].(string); ok && cmd == "error" {
						t.Fatalf("%s should not send error for unknown fields (forward compatibility)", tt.desc)
					}
				}

				// Verify unknown field warnings were logged
				allLogs := observedLogs.All()
				warnLogs := observedLogs.FilterMessage("Unknown field in message - dropping for forward compatibility").All()
				require.GreaterOrEqual(t, len(warnLogs), 1,
					"%s should log WARN for unknown fields (found %d WARN logs total, %d 'unknown field' logs)",
					tt.desc, len(allLogs), len(warnLogs))

				// Verify log contains command name in context
				foundCommandLog := false
				for _, log := range warnLogs {
					fields := log.ContextMap()
					if cmd, ok := fields["command"].(string); ok && cmd == tt.command {
						foundCommandLog = true
						break
					}
				}
				assert.True(t, foundCommandLog,
					"%s WARN log should contain command=%s in context", tt.desc, tt.command)
			})
		}
	}
}
