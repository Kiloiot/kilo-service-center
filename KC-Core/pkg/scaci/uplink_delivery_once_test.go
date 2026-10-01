package scaci

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// The two Application Centers of the retry tests: A always reachable, B the
// one whose first attempt fails.
const (
	onceSessionA = int64(801)
	onceSessionB = int64(802)
	onceAcEuiA   = uint64(0x70B3D59CD0000D01)
	onceAcEuiB   = uint64(0x70B3D59CD0000D02)
)

var errOnceSocket = errors.New("connection reset by peer")

// closingConn fails every write and notes that it was closed.
type closingConn struct {
	failingMockConn
	closed bool
}

func (c *closingConn) Close() error {
	c.closed = true
	return nil
}

// twoApplicationCenters serves A on connA and B on connB, both persisted
// sessions of the tenant, over one operation ledger.
func twoApplicationCenters(connA, connB net.Conn) (*Server, *operationLedger) {
	ledger := newOperationLedger()
	server := &Server{
		registry: newTestRegistry(map[net.Conn]*Session{
			connA: onceSession(onceSessionA, onceAcEuiA),
			connB: onceSession(onceSessionB, onceAcEuiB),
		}, newHolderFake()),
		codec:              testFrameCodec,
		commands:           mustTestCommandRegistry(),
		clock:              clock.SystemClock{},
		logger:             logger.NewNop(),
		config:             &Config{},
		sessionPersistence: sessionRowsOver{repo: acceptingCounterStore{}},
		operationRecorder:  ledger,
		operationRepo:      ledger,
	}
	return server, ledger
}

func onceSession(id int64, acEui uint64) *Session {
	return withOpIDs(&Session{ID: id, TenantID: broadcastULDataTestTenant, AcEui: acEui, State: StateActive}, OpIDPair{SC: -1})
}

func deliverUplink(server *Server) error {
	return server.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, broadcastULDataFixture())
}

// A retried delivery reaches only the session that did not get the uplink:
// A keeps its one operation, B gets its first, and the telegram never goes
// out again as a new, non-duplicate operation (SCACI §3.2, §3.8.1).
func TestBroadcastULData_RetryAfterARecordFailureReachesOnlyTheMissingSession(t *testing.T) {
	connA, connB := &mockConn{}, &mockConn{}
	server, ledger := twoApplicationCenters(connA, connB)
	ledger.fail(onceSessionB, errDeliveryRecordFailed)

	firstErr := deliverUplink(server)
	ledger.heal(onceSessionB)
	require.NoError(t, deliverUplink(server))

	require.ErrorIs(t, firstErr, errDeliveryRecordFailed, "B did not get the uplink: the outbox retries it")
	opIDsA := ledger.uplinkOpIDs(onceSessionA, broadcastULDataTestMessageID)
	opIDsB := ledger.uplinkOpIDs(onceSessionB, broadcastULDataTestMessageID)
	require.Len(t, opIDsA, 1, "one operation per session and uplink")
	require.Len(t, opIDsB, 1, "one operation per session and uplink")
	assert.Equal(t, opIDsA, writtenOpIDs(t, connA.written), "A received the uplink once")
	assert.Equal(t, opIDsB, writtenOpIDs(t, connB.written), "B received it on the retry")
}

// A request that cannot be written once its operation is on record is left
// to the session's resume: the connection is closed, the delivery counts,
// a retry sends it nowhere again, and the resume reissues it under its
// original opId (SCACI §1, §3.2).
func TestBroadcastULData_SocketFailureAfterRecordingIsLeftToTheResume(t *testing.T) {
	connA, connB := &mockConn{}, &closingConn{failingMockConn: failingMockConn{writeErr: errOnceSocket}}
	server, ledger := twoApplicationCenters(connA, connB)

	require.NoError(t, deliverUplink(server), "B's operation is on record for its resume")
	require.NoError(t, deliverUplink(server))

	assert.True(t, connB.closed, "B loses its connection and stays resumable")
	opIDsA := ledger.uplinkOpIDs(onceSessionA, broadcastULDataTestMessageID)
	opIDsB := ledger.uplinkOpIDs(onceSessionB, broadcastULDataTestMessageID)
	require.Len(t, opIDsA, 1)
	require.Len(t, opIDsB, 1)
	assert.Equal(t, opIDsA, writtenOpIDs(t, connA.written), "A received the uplink once")

	resumed := &mockConn{}
	server.replayPendingOperations(resumed, onceSession(onceSessionB, onceAcEuiB))

	assert.Equal(t, opIDsB, writtenOpIDs(t, resumed.written), "B's resume reissues its one operation")
	var reissued map[string]interface{}
	require.NoError(t, decodeResponse(resumed.written, &reissued))
	assert.Equal(t, false, reissued["duplicate"], "the reissued operation is the original, not a repeat")
	assert.NotContains(t, reissued, models.OperationRequestKeySourceMessageID, "the delivery identity stays off the wire")
}

// An uplink without its stored message has no delivery identity and is
// refused rather than delivered without one.
func TestBroadcastULData_RefusesAnUnstoredUplink(t *testing.T) {
	server, _ := twoApplicationCenters(&mockConn{}, &mockConn{})
	uplink := broadcastULDataFixture()
	uplink.ID = ""

	err := server.BroadcastULData(testutil.TestContext(), broadcastULDataTestTenant, uplink)

	assert.ErrorIs(t, err, errULDataMessageNotStored)
}
