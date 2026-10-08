package bssci

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	pkgmioty "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/mioty"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
)

// resumedMetadata is metadata as a resumed session reads it back: stored as
// JSON and decoded by the pending-operation loader.
func resumedMetadata(t *testing.T, metadata map[string]interface{}) map[string]interface{} {
	t.Helper()
	stored, err := json.Marshal(metadata)
	require.NoError(t, err)
	decoded, err := decodeJSONFrame(stored)
	require.NoError(t, err)
	return normalizeStrictDecodedMap(decoded)
}

// SCACI §3.13.1: the epStat of an attach completed after a session resume
// carries the att's nonce and signature exactly as received.
func TestHandleAttachComplete_AfterResumeForwardsNonceAndSign(t *testing.T) {
	const (
		opID     = int64(5101)
		epEUI    = uint64(0x70B3D56770111505)
		tenantID = int64(51)
	)
	testLogger := logger.NewNop()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, _, queueSerializer, auditLogger, tenantResolver, mockStorage := CreateTestServices(testLogger, nil)
	server := NewTestServer(testLogger, mockStorage, nil, tenantID,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, nil, queueSerializer, auditLogger, tenantResolver)
	broadcaster := NewSyncMockSCACIEPStatusBroadcaster()
	server.SetSCACIEPStatusBroadcaster(broadcaster)
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:               "test-attach-resumed",
			BaseStationEUI:   TestBsEui01,
			ResolvedTenantID: tenantID,
			DbSessionID:      51,
			Encoding:         EncodingJSON,
		},
	}
	metadata := resumedMetadata(t, map[string]interface{}{
		"epEui":               mioty.FormatEUI64(epEUI),
		metadataKeyEndpointID: int64(51),
		"attachCnt":           int64(0),
		"snr":                 15.5,
		"rssi":                -80.0,
		"nonce":               []byte{0x01, 0x02, 0x03, 0x04},
		"sign":                []byte{0xAA, 0xBB, 0xCC, 0xDD},
	})
	pendingOp := &PendingOperation{OperationID: opID, OperationType: mioty.CmdAttach, Metadata: metadata}
	require.NoError(t, statusSvc.RecordPendingOperation(testutil.TestContext(), session, opID, pendingOp, tenantID))

	broadcaster.ExpectCall(1)
	require.NoError(t, server.CallHandleAttachComplete(session, &Message{Command: mioty.CmdAttachComplete, OpId: opID}, nil))
	broadcaster.Wait()

	call := broadcaster.LastCall()
	require.NotNil(t, call)
	assert.Equal(t, pkgmioty.EPStatusAttached, call.Data.EpStatus)
	require.NotNil(t, call.Data.Nonce, "the resumed attach's nonce is forwarded")
	assert.Equal(t, mioty.Numeric4{0x01, 0x02, 0x03, 0x04}, *call.Data.Nonce)
	require.NotNil(t, call.Data.Sign, "the resumed attach's signature is forwarded")
	assert.Equal(t, mioty.Numeric4{0xAA, 0xBB, 0xCC, 0xDD}, *call.Data.Sign)
}
