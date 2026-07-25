package bssci

import (
	"context"
	"errors"
	"testing"

	bsscitest "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/interfaces"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pairDLRXRepo implements the DL RX correlation write used by the
// dlRxStatQry/dlDataQue pair; createErr exercises the pre-write abort path.
type pairDLRXRepo struct {
	interfaces.DLRXStatusRepository
	createErr   error
	createCalls int
}

func (r *pairDLRXRepo) CreateDLRXStatusQuery(_ context.Context, _ int64, _ *uuid.UUID, _, _ []byte, _ int64) error {
	r.createCalls++
	return r.createErr
}

// pairStorage extends the detach-test stubStorage with a working DL RX
// correlation repository.
type pairStorage struct {
	*stubStorage
	dlrx *pairDLRXRepo
}

func (s *pairStorage) DLRXStatus() interfaces.DLRXStatusRepository { return s.dlrx }

// newPairFixture builds a server + registered session for exercising the
// SendDLDataQueue dlRxStatQry pairing (SCACI §3.10.1, BSSCI rev1 §5.16 /
// classic §3.16).
func newPairFixture(t *testing.T) (*Server, *memoryStatusService, *pairDLRXRepo, *Session, *bsscitest.TestConn) {
	t.Helper()
	log := newRecordingLogger()
	sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver, _ := CreateTestServices(log, nil)
	dlrx := &pairDLRXRepo{}
	storage := &pairStorage{stubStorage: newStubStorage(), dlrx: dlrx}
	server := NewTestServer(log, storage, nil, 1,
		sessionSvc, downlinkSvc, statusSvc, connectionSvc, broadcaster, queueSerializer, auditLogger, tenantResolver)
	server.config = &Config{MessageEncoding: EncodingJSON}
	server.RegisterHandlers()

	conn := &bsscitest.TestConn{Encoding: EncodingJSON}
	session := &Session{
		ProtocolSessionState: ProtocolSessionState{
			ID:                "pair-test",
			BaseStationEUI:    TestBsEui01,
			ResolvedTenantID:  1,
			DbSessionID:       1,
			Encoding:          EncodingJSON,
			HandshakeComplete: true,
		},
		Conn:          conn,
		Bidirectional: true,
	}
	server.RegisterSession(session)
	memoryStatus, ok := statusSvc.(*memoryStatusService)
	require.True(t, ok, "CreateTestServices returns the in-memory status service")
	return server, memoryStatus, dlrx, session, conn
}

func sendPair(server *Server, session *Session, dlRxStatQry bool) error {
	return server.SendDLDataQueue(session.ID, TestEpEui01, [][]byte{{0x01, 0x02}}, 42,
		0, false, nil, 0, false, false, false, false, 1, dlRxStatQry)
}

// TestSendDLDataQueue_PairEmitsQueryBeforeQueue: with the dlRxStatQry hint the
// BSSCI dlRxStatQry frame precedes the dlDataQue frame, each with its own
// freshly allocated operation ID (query first).
func TestSendDLDataQueue_PairEmitsQueryBeforeQueue(t *testing.T) {
	server, statusSvc, dlrx, session, conn := newPairFixture(t)

	require.NoError(t, sendPair(server, session, true))

	require.Equal(t, 2, conn.MessageCount(), "query and queue frames expected")
	qryFrame := conn.GetMessage(0)
	queFrame := conn.GetMessage(1)
	assert.Equal(t, mioty.CmdDLRxStatusQuery, qryFrame["command"], "query frame precedes the queue frame")
	assert.Equal(t, mioty.CmdDLDataQueue, queFrame["command"])
	assert.Equal(t, float64(-1), qryFrame["opId"], "query ID allocated first")
	assert.Equal(t, float64(-2), queFrame["opId"], "queue ID allocated second")
	assert.Equal(t, 1, dlrx.createCalls, "one correlation row per query")

	// Both recovery records are durably recorded before the frames
	_, err := statusSvc.GetPendingOperation(session, -1)
	assert.NoError(t, err, "query pending operation recorded")
	_, err = statusSvc.GetPendingOperation(session, -2)
	assert.NoError(t, err, "queue pending operation recorded")
}

// TestSendDLDataQueue_NoHintEmitsQueueOnly: without the hint only the
// dlDataQue frame is written and no correlation row is created.
func TestSendDLDataQueue_NoHintEmitsQueueOnly(t *testing.T) {
	server, _, dlrx, session, conn := newPairFixture(t)

	require.NoError(t, sendPair(server, session, false))

	require.Equal(t, 1, conn.MessageCount())
	assert.Equal(t, mioty.CmdDLDataQueue, conn.GetMessage(0)["command"])
	assert.Zero(t, dlrx.createCalls, "no correlation row without the hint")
}

// TestSendDLDataQueue_BatchPersistFailureEmitsNeitherFrame: a failure to
// durably record the pair's recovery records aborts before any wire write.
func TestSendDLDataQueue_BatchPersistFailureEmitsNeitherFrame(t *testing.T) {
	server, statusSvc, _, session, conn := newPairFixture(t)
	statusSvc.recordErr = errors.New("insert failed")

	err := sendPair(server, session, true)

	require.Error(t, err, "batch persistence failure must abort the pair")
	assert.Zero(t, conn.MessageCount(), "neither frame may reach the wire")
}

// TestSendDLDataQueue_CorrelationFailureEmitsNeitherFrame: a failure to
// persist the DL RX correlation row is a pre-write failure for the whole pair.
func TestSendDLDataQueue_CorrelationFailureEmitsNeitherFrame(t *testing.T) {
	server, statusSvc, dlrx, session, conn := newPairFixture(t)
	dlrx.createErr = errors.New("correlation insert failed")

	err := sendPair(server, session, true)

	require.Error(t, err, "correlation persistence failure must abort the pair")
	assert.Zero(t, conn.MessageCount(), "neither frame may reach the wire")
	_, getErr := statusSvc.GetPendingOperation(session, -1)
	assert.Error(t, getErr, "no recovery record is written when the pair aborts")
}

// TestSendDLDataQueue_QueryWriteFailurePreservesBothOperations: an ambiguous
// write on the query frame aborts the pair, preserves both recovery records
// for resume reissue with their original IDs, and never writes the queue
// frame to the corrupt connection.
func TestSendDLDataQueue_QueryWriteFailurePreservesBothOperations(t *testing.T) {
	server, statusSvc, _, session, conn := newPairFixture(t)
	conn.FailWrites = true

	err := sendPair(server, session, true)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrAmbiguousWrite)
	assert.Zero(t, conn.MessageCount(), "no decodable frame was completed")

	_, getErr := statusSvc.GetPendingOperation(session, -1)
	assert.NoError(t, getErr, "query operation preserved for resume")
	_, getErr = statusSvc.GetPendingOperation(session, -2)
	assert.NoError(t, getErr, "queue operation preserved for resume")
}

// TestSendDLDataQueue_CounterNeverRolledBack: whatever fails, allocated IDs
// stay consumed (harmless gap) - never restored.
func TestSendDLDataQueue_CounterNeverRolledBack(t *testing.T) {
	server, statusSvc, _, session, _ := newPairFixture(t)
	statusSvc.recordErr = errors.New("insert failed")

	require.Error(t, sendPair(server, session, true))
	assert.Equal(t, int64(-2), session.LastScOpId, "both consumed IDs stay consumed")

	statusSvc.recordErr = nil
	require.NoError(t, sendPair(server, session, true))
	assert.Equal(t, int64(-4), session.LastScOpId, "fresh IDs continue past the gap")
}
