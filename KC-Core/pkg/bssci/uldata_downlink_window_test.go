package bssci_test

import (
	"context"
	"sync"
	"testing"
	"time"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ulWindowQueueDepth leaves more pending downlinks than one window can carry,
// so only the window claim can hold a second dlDataQue back.
const ulWindowQueueDepth = 2

// ulPendingQueue hands out the endpoint's pending downlinks one reservation at a time.
type ulPendingQueue struct {
	mu      sync.Mutex
	pending []*storage.DownlinkMessage
}

func newULPendingQueue(depth int) *ulPendingQueue {
	queue := &ulPendingQueue{}
	org := uuid.New()
	for i := range depth {
		queID := int64(i + 1)
		queue.pending = append(queue.pending, &storage.DownlinkMessage{
			ID: queID, QueID: queID, OrganizationID: &org, UserData: [][]byte{{byte(queID)}},
		})
	}
	return queue
}

func (q *ulPendingQueue) ReserveNextPending(context.Context, int64, []byte, uint64) (*storage.DownlinkMessage, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) == 0 {
		return nil, storage.ErrNotFound
	}
	next := q.pending[0]
	q.pending = q.pending[1:]
	return next, nil
}

func (q *ulPendingQueue) ReserveByQueueID(context.Context, int64, uuid.UUID, uint64, []byte, uint64) (*storage.DownlinkMessage, error) {
	return nil, storage.ErrNotFound
}

func (q *ulPendingQueue) UpdateDownlinkStatus(context.Context, string, mioty.DLQueueStatus, *uuid.UUID) error {
	return nil
}

func (q *ulPendingQueue) MarkReservedAsQueued(context.Context, uint64, int64, uint64, int64, *uint32, *uuid.UUID) error {
	return nil
}

type ulWindowKey struct {
	tenantID  int64
	messageID string
}

// ulWindowClaims mirrors the message store's atomic dl_window_claimed flag.
type ulWindowClaims struct {
	mu      sync.Mutex
	claimed map[ulWindowKey]bool
}

func (w *ulWindowClaims) ClaimDownlinkWindow(_ context.Context, tenantID int64, messageID string) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	key := ulWindowKey{tenantID: tenantID, messageID: messageID}
	if w.claimed[key] {
		return false, nil
	}
	w.claimed[key] = true
	return true, nil
}

func (w *ulWindowClaims) ReleaseDownlinkWindow(_ context.Context, tenantID int64, messageID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.claimed, ulWindowKey{tenantID: tenantID, messageID: messageID})
	return nil
}

// useRealDispatcher wires the production dispatcher to the server's own
// SendDLDataQueue, so the frames that reach each station are the real ones.
func useRealDispatcher(t *testing.T, server *bssci.Server) {
	t.Helper()
	queue := newULPendingQueue(ulWindowQueueDepth)
	dispatcher, err := bssciservices.NewDownlinkDispatcher(logger.NewNop(), queue, queue,
		&ulWindowClaims{claimed: map[ulWindowKey]bool{}}, server.SendDLDataQueue, clock.SystemClock{})
	require.NoError(t, err)
	server.SetDownlinkDispatcher(dispatcher)
}

func openStation(server *bssci.Server, bsEui uint64, id string, bidirectional bool) (*bssci.Session, *testutil.TestConn) {
	conn := &testutil.TestConn{Encoding: mioty.EncodingMessagePack}
	session := createTestSession(bsEui, ulTestTenantID, conn)
	session.ID = id
	session.Bidirectional = bidirectional
	server.RegisterSession(session)
	return session, conn
}

func downlinkQueuesSent(conn *testutil.TestConn) int {
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	var count int
	for _, frame := range conn.SentMessages {
		if frame["command"] == mioty.CmdDLDataQueue {
			count++
		}
	}
	return count
}

func receiveOpenWindow(t *testing.T, server *bssci.Server, rxTime int64, receptions ...*bssci.Session) {
	t.Helper()
	for i, session := range receptions {
		data := buildTestULDataPayload(ulTestEpEui, 12, rxTime+int64(i), 15.5, -80.0, []byte{0x0C})
		data["dlOpen"] = true
		require.NoError(t, server.CallHandleULData(session, buildTestMessage(int64(1105+i), data), data))
	}
}

func TestULDataIngest_ReceiveOnlyFirstStationLeavesTheWindowToTheBidirectionalOne(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	useRealDispatcher(t, server)
	receiveOnly, receiveOnlyConn := openStation(server, ulTestBsEui1, "ul-window-rx-only", false)
	bidirectional, bidirectionalConn := openStation(server, ulTestBsEui2, "ul-window-bidi", true)

	receiveOpenWindow(t, server, time.Now().UnixNano(), receiveOnly, bidirectional)

	require.Len(t, store.StoredMessages(), 1, "both receptions are one telegram")
	assert.Zero(t, downlinkQueuesSent(receiveOnlyConn), "a receive-only station is never asked to transmit")
	assert.Equal(t, 1, downlinkQueuesSent(bidirectionalConn),
		"the bidirectional station that also heard the telegram fills its downlink window")
}

func TestULDataIngest_OneDownlinkWindowIsFilledOncePerTelegram(t *testing.T) {
	t.Parallel()
	store := newULFakeUplinkStore()
	server, _ := setupULTestServer(t, store, registeredEndpoint(ulTestTenantID))
	useRealDispatcher(t, server)
	first, firstConn := openStation(server, ulTestBsEui1, "ul-window-first", true)
	second, secondConn := openStation(server, ulTestBsEui2, "ul-window-second", true)

	receiveOpenWindow(t, server, time.Now().UnixNano(), first, second)

	assert.Equal(t, 1, downlinkQueuesSent(firstConn), "the first bidirectional reception fills the window")
	assert.Zero(t, downlinkQueuesSent(secondConn), "one downlink per window, however many stations heard it")
}
