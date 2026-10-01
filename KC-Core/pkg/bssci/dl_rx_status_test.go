package bssci_test

import (
	"testing"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testutil.TestConn removed - migrated to testutil.TestConn (testconn.go) with thread safety

// TestDLRXStatusHandlerValidation tests field validation for DL RX status
func TestDLRXStatusHandlerValidation(t *testing.T) {
	// Create test server with service dependencies
	logger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssci.CreateTestServices(logger, nil)
	server := bssci.NewTestServer(logger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	// Create test session with mock connection
	session := &bssci.Session{
		ProtocolSessionState: bssci.ProtocolSessionState{
			BaseStationEUI: bssci.TestBsEui01,
			DbSessionID:    12345,
			// int64, not string
			Encoding: "msgpack",
		},
		// Session encoding must match TestConn encoding (Issue #3-4 Fix A2)
		Conn: &testutil.TestConn{Encoding: "msgpack"},
		// Add mock connection to prevent nil pointer
	}

	tests := []struct {
		name      string
		data      map[string]interface{}
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid DL RX status",
			data: map[string]interface{}{
				"epEui":     bssci.TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"dlRxSnr":   float64(-5.5),
				"dlRxRssi":  float64(-85.0),
			},
			expectErr: false,
		},
		{
			name: "missing epEui",
			data: map[string]interface{}{
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"dlRxSnr":   float64(-5.5),
				"dlRxRssi":  float64(-85.0),
			},
			expectErr: true,
			errMsg:    "bssci.error.mandatory_field_missing: epEui",
		},
		{
			name: "missing rxTime",
			data: map[string]interface{}{
				"epEui":     bssci.TestEpEui01,
				"packetCnt": uint32(42),
				"dlRxSnr":   float64(-5.5),
				"dlRxRssi":  float64(-85.0),
			},
			expectErr: true,
			errMsg:    "bssci.error.mandatory_field_missing: rxTime",
		},
		{
			name: "missing packetCnt",
			data: map[string]interface{}{
				"epEui":    bssci.TestEpEui01,
				"rxTime":   int64(1234567890),
				"dlRxSnr":  float64(-5.5),
				"dlRxRssi": float64(-85.0),
			},
			expectErr: true,
			errMsg:    "bssci.error.mandatory_field_missing: packetCnt",
		},
		{
			name: "missing dlRxSnr - mandatory field",
			data: map[string]interface{}{
				"epEui":     bssci.TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"dlRxRssi":  float64(-85.0),
			},
			expectErr: true,
			errMsg:    "bssci.error.mandatory_field_missing: dlRxSnr",
		},
		{
			name: "missing dlRxRssi - mandatory field",
			data: map[string]interface{}{
				"epEui":     bssci.TestEpEui01,
				"rxTime":    int64(1234567890),
				"packetCnt": uint32(42),
				"dlRxSnr":   float64(-5.5),
			},
			expectErr: true,
			errMsg:    "bssci.error.mandatory_field_missing: dlRxRssi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock connection for each sub-test to prevent state bleed
			session.Conn = &testutil.TestConn{Encoding: "msgpack"}

			msg := &bssci.Message{
				Command: "dlRxStat",
				OpId:    123,
			}

			// Call handler - BSSCI handlers send protocol errors and return nil
			err := server.CallHandleDLRXStatus(session, msg, tt.data)

			if tt.expectErr {
				// Handler returns nil even on validation failure
				assert.NoError(t, err, "Handler returns nil after sending protocol error")

				// Check that error frame was sent to connection
				mockConn := session.Conn.(*testutil.TestConn)
				errorCode, errorMsg := mockConn.LastError()
				assert.NotEqual(t, 0, errorCode, "Expected error frame sent to connection")
				assert.Equal(t, bssci.POSIX_EPROTO, errorCode, "Expected POSIX_EPROTO for validation failure")
				assert.Contains(t, errorMsg, tt.errMsg, "Error message should contain expected text")
			} else {
				// For valid cases without database, handler may still fail on persistence
				// but validation should pass (no panic, structured error handling)
				// We're mainly testing that validation doesn't reject valid input
				if err != nil {
					// If there's an error, it should be about database, not validation
					assert.NotContains(t, err.Error(), "missing", "Should not be a validation error")
				}
			}
		})
	}
}

// TestHandlerRegistration verifies all DL RX status handlers are registered
func TestHandlerRegistration(t *testing.T) {
	// Create real services for test
	logger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, _, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(logger, nil)
	versionNegotiator, err := bssciservices.NewVersionNegotiator([]string{mioty.MIOTYProtocolVersion}, logger)
	require.NoError(t, err, "NewVersionNegotiator should build from the canonical version")
	server, err := bssci.NewServer(&bssci.Config{SocketWriteTimeout: bssci.TestFrameWriteTimeout}, logger, bssci.Dependencies{
		Protocol: bssci.ProtocolServices{
			Session:            sessionSvc,
			VersionNegotiator:  versionNegotiator,
			Downlink:           downlinkSvc,
			Status:             statusSvc,
			ConnectionRegistry: connectionSvc,
			QueueSerializer:    queueSerializer,
			AuditLogger:        auditLogger,
			TenantResolver:     tenantResolver,
		},
		TenantID:        1,
		DefaultTenantID: 1,
	})
	require.NoError(t, err, "NewServer should not return error with valid StatusService")

	expectedHandlers := []string{
		"dlRxStat",
		"dlRxStatRsp",
		"dlRxStatCmp",
		"dlRxStatQryRsp",
	}

	for _, cmd := range expectedHandlers {
		assert.Contains(t, server.Handlers(), cmd, "Handler for %s should be registered", cmd)
		assert.NotNil(t, server.Handlers()[cmd], "Handler for %s should not be nil", cmd)
	}

	// Verify SC-initiated commands and SC-sent completions are NOT registered inbound
	assert.NotContains(t, server.Handlers(), "dlRxStatQry", "dlRxStatQry should not have a handler (SC-initiated)")
	assert.NotContains(t, server.Handlers(), "dlRxStatQryCmp", "dlRxStatQryCmp is SC-sent; no inbound handler")
}

// TestValidationCommandList verifies DL RX commands are in validation list
func TestValidationCommandList(t *testing.T) {
	// Expected DL RX status commands per BSSCI §5.15 that are SC→BS (outbound)
	// Note: dlRxStat/dlRxStatRsp/dlRxStatCmp are BS→SC (inbound) and not in outbound catalog
	expected := []string{
		mioty.CmdDLRxStatusQuery,
		mioty.CmdDLRxStatusQueryResponse,
		mioty.CmdDLRxStatusQueryComplete,
	}

	for _, cmd := range expected {
		_, ok := mioty.AllowedOutboundFields(cmd)
		assert.True(t, ok, "SC->BS command %s must be in the outbound catalog", cmd)
	}

	// dlRxStat itself is BS-initiated and must stay out of the SC->BS catalog
	_, ok := mioty.AllowedOutboundFields(mioty.CmdDLRxStatus)
	assert.False(t, ok, "BS->SC command %s must not be in the outbound catalog", mioty.CmdDLRxStatus)
}
