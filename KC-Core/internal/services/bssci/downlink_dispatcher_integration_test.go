package bssciservices

import (
	"context"
	"testing"
	"time"

	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/bssci"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/KC-Core/pkg/testutil"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/postgres"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport/teststore"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bypassReserver simulates a reservation invariant bypass by returning a fixed
// row regardless of what the reservation should have produced.
type bypassReserver struct{ row *storage.DownlinkMessage }

func (r *bypassReserver) ReserveNextPending(context.Context, int64, []byte, uint64) (*storage.DownlinkMessage, error) {
	return r.row, nil
}

func (r *bypassReserver) ReserveByQueueID(context.Context, int64, uuid.UUID, uint64, []byte, uint64) (*storage.DownlinkMessage, error) {
	return r.row, nil
}

// reservationSnapshot holds every reservation-mutated downlink_queue column.
type reservationSnapshot struct {
	Status         string
	OrganizationID *uuid.UUID
	BsEui          *int64
	Attempts       int32
	TxTime         *int64
	PacketCnt      *int64
	UpdatedAt      time.Time
}

func snapshotReservation(t *testing.T, db *postgres.DB, queID int64) reservationSnapshot {
	t.Helper()
	var snap reservationSnapshot
	row := db.QueryRow(testutil.TestContext(), `
		SELECT status, organization_id, bs_eui, attempts, tx_time, transmission_packet_cnt, updated_at
		FROM downlink_queue WHERE que_id = $1`, queID)
	require.NoError(t, row.Scan(&snap.Status, &snap.OrganizationID, &snap.BsEui,
		&snap.Attempts, &snap.TxTime, &snap.PacketCnt, &snap.UpdatedAt))
	return snap
}

// TestDispatchReserved_OwnerlessRowZeroWrites proves a reserved row that
// reaches the dispatcher without an organization (an invariant bypass: enqueue
// and the schema both forbid ownerless rows) survives the dispatch attempt
// with no wire send and no database mutation: every reservation-mutated
// column is identical before and after.
func TestDispatchReserved_OwnerlessRowZeroWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()

	ctx := testutil.TestContext()
	rowOrg := uuid.New()
	queID := int64(2000001)
	epEUIBytes := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11}

	_, err := db.Query(ctx, `
		INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, payload, status, priority,
			organization_id, attempts, created_at, updated_at, earliest_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW(), NULL)`,
		queID, epEUIBytes, 1, []byte("payload"), bssci.DLQueueStatusReserved, 5, rowOrg, 0)
	require.NoError(t, err, "seed reserved downlink row")

	before := snapshotReservation(t, db, queID)
	require.NotNil(t, before.OrganizationID)
	require.Equal(t, rowOrg, *before.OrganizationID)

	// The bypass reserver hands the dispatcher an in-memory copy whose
	// organization was lost, simulating an invariant bypass.
	var reserved storage.DownlinkMessage
	reserved.ID = fetchDownlinkID(t, db, queID)
	reserved.QueID = queID
	reserved.OrganizationID = nil
	reserved.Payload = []byte("payload")
	sends := 0
	sendFn := func(string, uint64, [][]byte, int64, float32, bool, []int64, uint8,
		bool, bool, bool, bool, int64, *uuid.UUID, bool) error {
		sends++
		return nil
	}
	dispatcher := mustDispatcher(t, logger.NewNop(), &bypassReserver{row: &reserved},
		postgres.NewRepositories(db).Downlinks, sendFn)

	session := &bssci.Session{ProtocolSessionState: bssci.ProtocolSessionState{BaseStationEUI: 0x70B3D59CD00009E6}, Bidirectional: true}
	dispatched, err := dispatcher.DispatchIfAvailable(
		ctx, 1, session, 0xAABBCCDDEEFF0011, dispatchTestMessageID, false)

	assert.ErrorIs(t, err, bssci.ErrDispatchOrgMismatch)
	assert.False(t, dispatched)
	assert.Zero(t, sends, "an ownerless row must never reach the wire")

	after := snapshotReservation(t, db, queID)
	assert.Equal(t, before, after, "the rejected dispatch must leave the row untouched")
}

func fetchDownlinkID(t *testing.T, db *postgres.DB, queID int64) int64 {
	t.Helper()
	var id int64
	require.NoError(t, db.QueryRow(testutil.TestContext(),
		`SELECT id FROM downlink_queue WHERE que_id = $1`, queID).Scan(&id))
	return id
}

// TestReclaimReservations_ReturnsOrphanedRowToPending runs the reclaim over
// PostgreSQL: a row left reserved by an ambiguous write on a session that is
// not resumed becomes dispatchable again, and its return to the queue is
// stored as an event the endpoint managers of its owner read.
func TestReclaimReservations_ReturnsOrphanedRowToPending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	db, cleanup := teststore.Setup(t)
	defer cleanup()

	ctx := testutil.TestContext()
	queID := int64(2000002)
	station := uint64(0x70B3D59CD00009E6)
	_, err := db.Query(ctx, `
		INSERT INTO downlink_queue (que_id, ep_eui, tenant_id, payload, status, priority,
			organization_id, bs_eui, attempts, created_at, updated_at, earliest_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW(), NULL)`,
		queID, []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x12}, 1, []byte("payload"),
		bssci.DLQueueStatusReserved, 5, uuid.New(), []byte{0x70, 0xB3, 0xD5, 0x9C, 0xD0, 0x00, 0x09, 0xE6}, 0)
	require.NoError(t, err, "seed reserved downlink row")

	repos := postgres.NewRepositories(db)
	events := mustAuditLogger(t, repos.SystemEvents, repos.Downlinks, repos.BaseStations)
	released, err := mustReclaimer(t, repos.Downlinks, events).ReclaimReservations(ctx, station, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), released)
	var status string
	require.NoError(t, db.QueryRow(ctx, `SELECT status FROM downlink_queue WHERE que_id = $1`, queID).Scan(&status))
	assert.Equal(t, string(bssci.DLQueueStatusPending), status)

	var tenant int64
	var category, sourceName, epEui string
	require.NoError(t, db.QueryRow(ctx, `SELECT tenant_id, event_category, source_name, data->>$2 FROM system_events
		WHERE event_type = $1`, models.EventTypeDLDataRequeued, models.EventDetailKeyEpEui).Scan(&tenant, &category, &sourceName, &epEui))
	assert.Equal(t, int64(1), tenant, "filed under the downlink's owner")
	assert.Equal(t, models.EventCategoryMessage, category)
	assert.Equal(t, "AABBCCDDEEFF0012", sourceName)
	assert.Equal(t, "AABBCCDDEEFF0012", epEui)
}
