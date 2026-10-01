package bssci_test

import (
	"context"
	"encoding/binary"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bssciservices "github.com/Kiloiot/kilo-service-center/KC-Core/internal/services/bssci"
	bssci "github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
)

// Queue ids of the held-downlink scenario: one queued at the station before
// its attach propagate, one sent to it after the propagate was issued.
const (
	heldBeforePropagate uint64 = 7_300_000_000_000_201
	sentAfterPropagate  uint64 = 7_300_000_000_000_202
)

// heldRow is one downlink as the queue store keeps it.
type heldRow struct {
	message storage.DownlinkMessage
	epEUI   uint64
	status  mioty.DLQueueStatus
	holder  uint64
	sentAt  time.Time
}

// stationQueue mirrors the store's downlink lifecycle: listing and
// reservation of pending rows, queued confirmation with its send time, and
// the release of the rows a station discarded.
type stationQueue struct {
	mu   sync.Mutex
	org  uuid.UUID
	rows []*heldRow
}

// newStationQueue holds pending downlinks for the test endpoint.
// ExpireStationRevocations reports that the station was asked to drop no downlink.
func (q *stationQueue) ExpireStationRevocations(context.Context, uint64) ([]*storage.DownlinkMessage, error) {
	return nil, nil
}

// noStationExpiries reports no downlink expired at a station, ends none for a
// deleted station and caches no queue owner.
type noStationExpiries struct{}

func (noStationExpiries) ExpireRemovedStationDownlinks(context.Context, uint64) ([]*storage.DownlinkMessage, error) {
	return nil, nil
}

func (noStationExpiries) ReportExpiredAtStation(context.Context, *storage.DownlinkMessage) {}

func (noStationExpiries) UnregisterQueueTenant(int64) {}

// stationQueueReclaimer is the reclaimer over the station queue fake.
func stationQueueReclaimer(t *testing.T, queue *stationQueue, requeues *requeueLog, log logger.Logger) bssci.DownlinkReclaimer {
	t.Helper()
	reclaimer, err := bssciservices.NewDownlinkReclaimer(bssciservices.DownlinkReclaimerDeps{
		Store: queue, Revocations: queue, Removed: noStationExpiries{}, Events: requeues, Expiries: noStationExpiries{},
		Tenants: noStationExpiries{}, Logger: log,
	})
	require.NoError(t, err)
	return reclaimer
}

func newStationQueue(queueIDs ...uint64) *stationQueue {
	queue := &stationQueue{org: uuid.New()}
	for _, queueID := range queueIDs {
		queue.addPending(queueID, ulTestEpEui)
	}
	return queue
}

func (q *stationQueue) addPending(queueID, epEUI uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	id := int64(len(q.rows) + 1)
	q.rows = append(q.rows, &heldRow{epEUI: epEUI, status: mioty.DLQueueStatusPending, message: storage.DownlinkMessage{
		ID: id, QueID: int64(queueID), TenantID: strconv.FormatInt(ulTestTenantID, 10), OrganizationID: &q.org, //nolint:gosec // test queue ids fit int64
		UserData: [][]byte{{byte(id)}},
	}})
}

func (q *stationQueue) ListPendingDownlinks(context.Context) ([]storage.PendingDownlink, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var pending []storage.PendingDownlink
	for _, row := range q.rows {
		if row.status == mioty.DLQueueStatusPending {
			pending = append(pending, storage.PendingDownlink{
				QueID: uint64(row.message.QueID), TenantID: ulTestTenantID, OrganizationID: q.org, EpEUI: row.epEUI, //nolint:gosec // test queue ids are positive
			})
		}
	}
	return pending, nil
}

func (q *stationQueue) row(queueID uint64) heldRow {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, row := range q.rows {
		if uint64(row.message.QueID) == queueID { //nolint:gosec // test queue ids are positive
			return *row
		}
	}
	return heldRow{}
}

func (q *stationQueue) reserve(bsEUI uint64, match func(*heldRow) bool) (*storage.DownlinkMessage, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, row := range q.rows {
		if row.status == mioty.DLQueueStatusPending && match(row) {
			row.status, row.holder = mioty.DLQueueStatusReserved, bsEUI
			reserved := row.message
			return &reserved, nil
		}
	}
	return nil, storage.ErrNotFound
}

func (q *stationQueue) ReserveNextPending(_ context.Context, _ int64, epEUI []byte, bsEUI uint64) (*storage.DownlinkMessage, error) {
	return q.reserve(bsEUI, func(row *heldRow) bool { return row.epEUI == binary.BigEndian.Uint64(epEUI) })
}

func (q *stationQueue) ReserveByQueueID(_ context.Context, _ int64, _ uuid.UUID, queueID uint64, _ []byte, bsEUI uint64) (*storage.DownlinkMessage, error) {
	return q.reserve(bsEUI, func(row *heldRow) bool { return uint64(row.message.QueID) == queueID }) //nolint:gosec // test queue ids are positive
}

func (q *stationQueue) UpdateDownlinkStatus(_ context.Context, id string, status mioty.DLQueueStatus, _ *uuid.UUID) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, row := range q.rows {
		if strconv.FormatInt(row.message.ID, 10) == id {
			row.status = status
		}
	}
	return nil
}

func (q *stationQueue) MarkReservedAsQueued(_ context.Context, queID uint64, _ int64, bsEUI uint64, txTime int64, _ *uint32, _ *uuid.UUID) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, row := range q.rows {
		if uint64(row.message.QueID) == queID && row.status == mioty.DLQueueStatusReserved { //nolint:gosec // test queue ids are positive
			row.status, row.holder, row.sentAt = mioty.DLQueueStatusQueued, bsEUI, time.Unix(0, txTime)
		}
	}
	return nil
}

// frameNumber reads a numeric frame field whatever width the decoder chose.
func frameNumber(t *testing.T, value interface{}) int64 {
	t.Helper()
	switch number := value.(type) {
	case int8:
		return int64(number)
	case int16:
		return int64(number)
	case int32:
		return int64(number)
	case int64:
		return number
	case uint8:
		return int64(number)
	case uint16:
		return int64(number)
	case uint32:
		return int64(number)
	case uint64:
		return int64(number) //nolint:gosec // test queue ids fit int64
	}
	require.Failf(t, "not a number", "%T %v", value, value)
	return 0
}

// sentQueueIDs lists the queue ids of the dlDataQue frames the station was sent.
func sentQueueIDs(t *testing.T, conn *testutil.TestConn) []uint64 {
	t.Helper()
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	var queueIDs []uint64
	for _, frame := range conn.SentMessages {
		if frame["command"] == mioty.CmdDLDataQueue {
			queueIDs = append(queueIDs, uint64(frameNumber(t, frame["queId"]))) //nolint:gosec // test queue ids are positive
		}
	}
	return queueIDs
}

// lastOpID is the operation id of the last frame of the command the station was sent.
func lastOpID(t *testing.T, conn *testutil.TestConn, command string) int64 {
	t.Helper()
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	for i := len(conn.SentMessages) - 1; i >= 0; i-- {
		if conn.SentMessages[i]["command"] == command {
			return frameNumber(t, conn.SentMessages[i]["opId"])
		}
	}
	require.Failf(t, "no frame", "the station was sent no %s", command)
	return 0
}

// heldDownlinkStation is a bidirectional station that holds the endpoint's
// first downlink, queued at an earlier downlink window.
type heldDownlinkStation struct {
	server   *bssci.Server
	queue    *stationQueue
	requeues *requeueLog
	session  *bssci.Session
	conn     *testutil.TestConn
}

func newHeldDownlinkStation(t *testing.T) heldDownlinkStation {
	t.Helper()
	server, _ := setupULTestServer(t, newULFakeUplinkStore(), registeredEndpoint(ulTestTenantID))
	queue := newStationQueue(heldBeforePropagate, sentAfterPropagate)
	dispatcher, err := bssciservices.NewDownlinkDispatcher(logger.NewNop(), queue, queue,
		&ulWindowClaims{claimed: map[ulWindowKey]bool{}}, server.SendDLDataQueue, clock.SystemClock{})
	require.NoError(t, err)
	requeues := &requeueLog{}
	reclaimer := stationQueueReclaimer(t, queue, requeues, logger.NewNop())
	server.SetDownlinkDispatcher(dispatcher)
	server.SetDownlinkReclaimer(reclaimer)
	session, conn := openStation(server, ulTestBsEui2, "held-downlink-station", true)
	station := heldDownlinkStation{server: server, queue: queue, requeues: requeues, session: session, conn: conn}

	station.openDownlinkWindow(t, 21)
	require.Equal(t, []uint64{heldBeforePropagate}, sentQueueIDs(t, conn))
	require.Equal(t, mioty.DLQueueStatusQueued, queue.row(heldBeforePropagate).status)
	return station
}

// openDownlinkWindow delivers one uplink of a new telegram that opens a downlink window.
func (s heldDownlinkStation) openDownlinkWindow(t *testing.T, packetCnt int64) {
	t.Helper()
	data := buildTestULDataPayload(ulTestEpEui, packetCnt, time.Now().UnixNano(), 15.5, -80.0, []byte{byte(packetCnt)})
	data["dlOpen"] = true
	require.NoError(t, s.server.CallHandleULData(s.session, buildTestMessage(packetCnt, data), data))
}

// propagate sends the station the endpoint's attPrp and returns its operation id.
func (s heldDownlinkStation) propagate(t *testing.T) int64 {
	t.Helper()
	nwkSnKey := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	require.NoError(t, s.server.SendAttachPropagate(s.session.ID, ulTestEpEui, nwkSnKey, 0x1505, true, 0, false, 0, false, false))
	return lastOpID(t, s.conn, mioty.CmdAttachPropagate)
}

// answer delivers the station's attPrpRsp to the operation.
func (s heldDownlinkStation) answer(t *testing.T, opID int64, response map[string]interface{}) {
	t.Helper()
	response["command"], response["opId"] = mioty.CmdAttachPropagateResponse, opID
	require.NoError(t, s.server.CallHandleAttachPropagateResponse(s.session,
		&bssci.Message{Command: mioty.CmdAttachPropagateResponse, OpId: opID, Data: response}, response))
}

// TestAttachPropagateReturnsTheStationsHeldDownlinksToPending: a base
// station discards the downlinks it queued for an endpoint when it takes an
// attach propagate for it (BSSCI §3.8). Once the station answers the
// attPrp, the downlink it held returns to pending and goes out again at the
// endpoint's next downlink window; one sent to it after the attPrp was
// issued reaches it behind the attPrp and stays queued there.
func TestAttachPropagateReturnsTheStationsHeldDownlinksToPending(t *testing.T) {
	station := newHeldDownlinkStation(t)

	propagateOpID := station.propagate(t)
	station.openDownlinkWindow(t, 22)
	require.Equal(t, []uint64{heldBeforePropagate, sentAfterPropagate}, sentQueueIDs(t, station.conn))
	station.answer(t, propagateOpID, map[string]interface{}{})

	released := station.queue.row(heldBeforePropagate)
	assert.Equal(t, mioty.DLQueueStatusPending, released.status, "the station discarded the downlink it held")
	assert.Zero(t, released.holder, "no station holds it")
	assert.Equal(t, mioty.DLQueueStatusQueued, station.queue.row(sentAfterPropagate).status,
		"the downlink sent behind the attPrp is still held")
	assert.Equal(t, []uint64{heldBeforePropagate}, station.requeues.queueIDs(),
		"the downlink's return to the queue is announced, so every open queue view shows it pending")

	station.openDownlinkWindow(t, 23)
	assert.Equal(t, []uint64{heldBeforePropagate, sentAfterPropagate, heldBeforePropagate}, sentQueueIDs(t, station.conn),
		"the released downlink goes out at the next downlink window")
	assert.Equal(t, mioty.DLQueueStatusQueued, station.queue.row(heldBeforePropagate).status)
}

// TestRefusedAttachPropagateKeepsTheStationsHeldDownlinks: a station that
// answers the attPrp with an error did not take the attachment, so it still
// holds the downlinks it queued and nothing is sent again.
func TestRefusedAttachPropagateKeepsTheStationsHeldDownlinks(t *testing.T) {
	station := newHeldDownlinkStation(t)

	station.answer(t, station.propagate(t), map[string]interface{}{"result": int64(1)})

	held := station.queue.row(heldBeforePropagate)
	assert.Equal(t, mioty.DLQueueStatusQueued, held.status)
	assert.Equal(t, ulTestBsEui2, held.holder)
	assert.Empty(t, station.requeues.queueIDs(), "nothing returned to the queue, nothing is announced")
}
