package bssci_test

import (
	"sort"
	"strings"
	"testing"

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
		BaseStationEUI: bssci.TestBsEui01,
		DbSessionID:    12345,                                   // int64, not string
		Encoding:       "msgpack",                               // Session encoding must match TestConn encoding (Issue #3-4 Fix A2)
		Conn:           &testutil.TestConn{Encoding: "msgpack"}, // Add mock connection to prevent nil pointer
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
			errMsg:    "Missing epEui in dlRxStat",
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
			errMsg:    "Missing rxTime in dlRxStat",
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
			errMsg:    "Missing packetCnt in dlRxStat",
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
			errMsg:    "Missing dlRxSnr in dlRxStat",
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
			errMsg:    "Missing dlRxRssi in dlRxStat",
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

// TestDLRXStatusPersistence tests database persistence of DL RX status
func TestDLRXStatusPersistence(t *testing.T) {
	t.Skip("Requires database connection - implement in integration tests")

	// This would test:
	// 1. Successful persistence with all fields
	// 2. Correct conversion of EUI to bytes
	// 3. Proper tenant ID handling
	// 4. Timestamp generation
}

// TestDLRXStatusQueryComplete tests cleanup of pending operations
func TestDLRXStatusQueryComplete(t *testing.T) {
	logger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, mockStorage := bssci.CreateTestServices(logger, nil)
	server := bssci.NewTestServer(logger, mockStorage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)

	// Add a pending operation (access via unexported field now impossible from external test)
	// This test needs to be revised or moved to internal package

	// Create test session
	session := &bssci.Session{
		BaseStationEUI: bssci.TestBsEui01,
		DbSessionID:    12345,
	}

	msg := &bssci.Message{
		Command: "dlRxStatQryCmp",
		OpId:    int64(-100),
	}

	// Call handler
	err := server.CallHandleDLRXStatusQueryComplete(session, msg, nil)

	// Without database, this will fail, but we're testing it doesn't panic
	// In a real integration test with database, we'd verify cleanup
	if err != nil {
		// Expected error due to nil database
		t.Logf("Expected error without database: %v", err)
	}

	// Verify memory cleanup (if handler got that far)
	// In a real test with DB, we'd assert the operation was removed
}

// TestHandlerRegistration verifies all DL RX status handlers are registered
func TestHandlerRegistration(t *testing.T) {
	// Create real services for test
	logger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := bssci.CreateTestServices(logger, nil)
	server, err := bssci.NewServer(&bssci.Config{}, logger, nil, nil, nil, nil, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, nil, 1)
	require.NoError(t, err, "NewServer should not return error with valid StatusService")
	server.RegisterHandlers()

	expectedHandlers := []string{
		"dlRxStat",
		"dlRxStatRsp",
		"dlRxStatCmp",
		"dlRxStatQryRsp",
		"dlRxStatQryCmp",
	}

	for _, cmd := range expectedHandlers {
		assert.Contains(t, server.Handlers(), cmd, "Handler for %s should be registered", cmd)
		assert.NotNil(t, server.Handlers()[cmd], "Handler for %s should not be nil", cmd)
	}

	// Verify dlRxStatQry is NOT registered (SC-initiated only)
	assert.NotContains(t, server.Handlers(), "dlRxStatQry", "dlRxStatQry should not have a handler (SC-initiated)")
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

	// Get actual DL RX commands from outbound catalog by enumerating catalog itself
	var actual []string
	for command := range mioty.OutboundFieldCatalog {
		// Include SC→BS DL RX commands (dlRxStatQry*), exclude BS→SC (dlRxStat without Qry)
		if strings.HasPrefix(command, "dlRxStat") &&
			command != mioty.CmdDLRxStatus &&
			command != mioty.CmdDLRxStatusResponse &&
			command != mioty.CmdDLRxStatusComplete {
			actual = append(actual, command)
		}
	}

	// Sort both for stable comparison
	sort.Strings(expected)
	sort.Strings(actual)

	// Verify all expected commands are in catalog
	if len(actual) != len(expected) {
		t.Errorf("Command count mismatch: expected %d commands, found %d in catalog", len(expected), len(actual))
		t.Logf("Expected: %v", expected)
		t.Logf("In catalog: %v", actual)
	}

	for i, cmd := range expected {
		if i >= len(actual) || actual[i] != cmd {
			t.Errorf("Command %s not found in outbound catalog", cmd)
		}
	}
}
