package bssci

import (
	"testing"
	"time"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSCInitiatedOperationsFinalizeAfterCmp verifies that the service center's
// own SC-initiated operations (status, dlDataQue, dlDataRev, dlRxStatQry) remove
// their pending operation after the SC sends its completion. A spec-compliant
// base station never returns those completions, so without finalization here the
// pending rows leak (the historical 14k-row status leak).
func TestSCInitiatedOperationsFinalizeAfterCmp(t *testing.T) {
	newServer := func(t *testing.T) (*Server, StatusService, *Session) {
		t.Helper()
		log := newRecordingLogger()
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, storage := CreateTestServices(log, nil)
		server := NewTestServer(log, storage, nil, 1,
			sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
		server.config = &Config{MessageEncoding: EncodingJSON}
		server.SetStorageForTest(storage)
		server.RegisterHandlers()
		session := &Session{
			ProtocolSessionState: ProtocolSessionState{
				ID:                "leak-test",
				BaseStationEUI:    TestBsEui01,
				ResolvedTenantID:  1,
				DbSessionID:       1,
				Encoding:          EncodingJSON,
				HandshakeComplete: true,
			},
			Conn: &bsscitest.TestConn{Encoding: EncodingJSON},
		}
		return server, statusSvc, session
	}

	cases := []struct {
		name     string
		opType   string
		respCmd  string
		endpoint []byte
		metadata map[string]interface{}
		setup    func(s *Server)
		invoke   func(s *Server, sess *Session, msg *Message, data map[string]interface{}) error
	}{
		{
			name:    "status",
			opType:  mioty.CmdStatus,
			respCmd: mioty.CmdStatusResponse,
			invoke: func(s *Server, sess *Session, msg *Message, data map[string]interface{}) error {
				return s.handleStatusResponse(s, sess, msg, data)
			},
		},
		{
			name:     "dlDataQue",
			opType:   mioty.CmdDLDataQueue,
			respCmd:  mioty.CmdDLDataQueueResponse,
			endpoint: []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77},
			metadata: map[string]interface{}{"queId": int64(42), "tenantID": "1"},
			invoke: func(s *Server, sess *Session, msg *Message, data map[string]interface{}) error {
				return s.handleDLDataQueueResponse(s, sess, msg, data)
			},
		},
		{
			name:     "dlDataRev",
			opType:   mioty.CmdDLDataRevoke,
			respCmd:  mioty.CmdDLDataRevokeResponse,
			endpoint: []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77},
			metadata: map[string]interface{}{"queId": int64(42), "tenantID": "1"},
			setup: func(s *Server) {
				s.downlinkSvc = &mqttTestDownlinkService{}
			},
			invoke: func(s *Server, sess *Session, msg *Message, data map[string]interface{}) error {
				return s.handleDLDataRevokeResponse(s, sess, msg, data)
			},
		},
		{
			name:    "dlRxStatQry",
			opType:  mioty.CmdDLRxStatusQuery,
			respCmd: mioty.CmdDLRxStatusQueryResponse,
			invoke: func(s *Server, sess *Session, msg *Message, data map[string]interface{}) error {
				return s.handleDLRXStatusQueryResponse(s, sess, msg, data)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, statusSvc, session := newServer(t)
			if tc.setup != nil {
				tc.setup(server)
			}
			const opID = int64(-7)

			metadata := tc.metadata
			if metadata == nil {
				metadata = map[string]interface{}{}
			}
			pendingOp := &PendingOperation{
				OperationType: tc.opType,
				Endpoint:      tc.endpoint,
				CreatedAt:     time.Now(),
				Metadata:      metadata,
			}
			require.NoError(t, statusSvc.RecordPendingOperation(testutil.TestContext(), session, opID, pendingOp, session.DbSessionID))
			_, err := statusSvc.GetPendingOperation(session, opID)
			require.NoError(t, err, "pending op must exist before the response")

			data := map[string]interface{}{
				"command": tc.respCmd,
				"opId":    opID,
				// status response mandatory fields (ignored by the others)
				"code":    int64(0),
				"message": "ok",
				"time":    time.Now().UnixNano(),
			}
			msg := &Message{Command: tc.respCmd, OpId: opID, Data: data}
			require.NoError(t, tc.invoke(server, session, msg, data))

			_, err = statusSvc.GetPendingOperation(session, opID)
			assert.Error(t, err, "pending op must be removed after the SC sends its completion")
		})
	}
}
