package postgres

import (
	"encoding/binary"
	"testing"

	"github.com/Kiloiot/kilo-service-center/pkg/clock"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// orgIsolationFixture provisions one tenant carrying two organizations, each
// with one pending downlink for the same endpoint, and returns the queue IDs.
// Multiple organizations sharing a tenant is the configuration the dispatch
// path must never mix up.
func orgIsolationFixture(t *testing.T) (db *DB, orgA, orgB uuid.UUID, queA, queB int64, epEUI uint64) {
	t.Helper()

	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)

	const tenantID = int64(300)
	createTestTenant(t, sqlxDB, tenantID, "OrgIsolationTenant")
	orgA = uuid.New()
	orgB = uuid.New()
	createTestOrganization(t, sqlxDB, orgA, tenantID, "OrgA")
	createTestOrganization(t, sqlxDB, orgB, tenantID, "OrgB")

	logger.Initialize("error", "json")
	db = &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}

	epEUI = uint64(0x0102030405060799)
	orgAStr := orgA.String()
	orgBStr := orgB.String()
	queA = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI: epEUI, TenantID: tenantID, Priority: 1, OrganizationID: &orgAStr,
	})
	queB = insertDownlink(t, db, DownlinkInsertParams{
		EpEUI: epEUI, TenantID: tenantID, Priority: 1, OrganizationID: &orgBStr,
	})
	return db, orgA, orgB, queA, queB, epEUI
}

// TestDownlinkOrgIsolation_Reservation proves the two dispatch paths handle a
// tenant with several organizations correctly: the automatic path serves every
// organization's rows in queue order with each row carrying its own
// organization, and the exact-queue path never crosses organizations.
func TestDownlinkOrgIsolation_Reservation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, orgA, orgB, queA, queB, epEUI := orgIsolationFixture(t)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)
	const bsEUI = uint64(0x7000000000000001)

	// Exact-queue path first (while both rows are still pending): org A must
	// not be able to reserve org B's row, and the failed attempt must leave
	// org B's row untouched.
	foreign, err := NewRepositories(db).Downlinks.ReservePendingDownlinkByQueueID(
		t.Context(), 300, orgA, uint64(queB), epEUIBytes, bsEUI,
	)
	require.ErrorIs(t, err, storage.ErrNotFound)
	assert.Nil(t, foreign, "org A must not reserve org B's downlink by queue ID")

	var statusB mioty.DLQueueStatus
	require.NoError(t, db.conn.QueryRow(
		`SELECT status FROM downlink_queue WHERE que_id = $1`, queB,
	).Scan(&statusB))
	assert.Equal(t, mioty.DLQueueStatusPending, statusB)

	// Automatic path: reservation is per tenant+endpoint and returns rows in
	// queue order regardless of which organization enqueued them; the returned
	// row's organization_id names the delivery organization. Both
	// organizations' rows must be dispatchable.
	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	first, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 300, epEUIBytes, bsEUI)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	require.NotNil(t, first, "the first pending row must be reservable")
	assert.Equal(t, queA, first.QueID)
	require.NotNil(t, first.OrganizationID)
	assert.Equal(t, orgA, *first.OrganizationID, "the row must carry the organization it was enqueued under")

	tx, err = db.BeginTx(t.Context())
	require.NoError(t, err)
	second, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 300, epEUIBytes, bsEUI)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	require.NotNil(t, second, "the second organization's row must be dispatchable too, not starved")
	assert.Equal(t, queB, second.QueID)
	require.NotNil(t, second.OrganizationID)
	assert.Equal(t, orgB, *second.OrganizationID)
}

// TestDownlinkOrgIsolation_Confirmation proves the reserved->queued transition
// stays organization-scoped through both confirmation paths: the dispatcher's
// immediate confirmation and the dlDataQueRsp crash-recovery repair use the
// same repository call, so this covers both.
func TestDownlinkOrgIsolation_Confirmation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, orgA, orgB, queA, _, epEUI := orgIsolationFixture(t)
	epEUIBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(epEUIBytes, epEUI)
	const bsEUI = uint64(0x7000000000000001)

	// Reserve the first pending row (org A's, by queue order).
	tx, err := db.BeginTx(t.Context())
	require.NoError(t, err)
	dl, err := tx.MIOTYDownlinks().ReserveNextPendingDownlink(t.Context(), 300, epEUIBytes, bsEUI)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	require.NotNil(t, dl)
	require.Equal(t, queA, dl.QueID)

	// Org B must not be able to confirm org A's reservation.
	err = NewRepositories(db).Downlinks.MarkReservedAsQueued(
		t.Context(), uint64(queA), 300, bsEUI, 1, nil, &orgB,
	)
	assert.ErrorIs(t, err, ErrDownlinkAlreadyReserved,
		"a foreign organization must not confirm the reservation")

	var status mioty.DLQueueStatus
	require.NoError(t, db.conn.QueryRow(
		`SELECT status FROM downlink_queue WHERE que_id = $1`, queA,
	).Scan(&status))
	assert.Equal(t, mioty.DLQueueStatusReserved, status, "the row must stay reserved after the foreign attempt")

	// The owner confirms successfully.
	require.NoError(t, NewRepositories(db).Downlinks.MarkReservedAsQueued(
		t.Context(), uint64(queA), 300, bsEUI, 1, nil, &orgA,
	))
	require.NoError(t, db.conn.QueryRow(
		`SELECT status FROM downlink_queue WHERE que_id = $1`, queA,
	).Scan(&status))
	assert.Equal(t, mioty.DLQueueStatusQueued, status)
}

// TestDownlinkOrgOwnership_EnqueueRequiresOrganization pins the enqueue
// invariant behind the row-authoritative dispatch model: a queue row cannot be
// created without the organization it belongs to, neither through the
// repository (explicit rejection) nor through raw SQL (NOT NULL constraint).
func TestDownlinkOrgOwnership_EnqueueRequiresOrganization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)

	const tenantID = int64(303)
	createTestTenant(t, sqlxDB, tenantID, "EnqueueOrgTenant")

	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}

	_, err := NewRepositories(db).Downlinks.EnqueueDownlink(t.Context(), &storage.DownlinkMessage{
		EPEUI:    "0102030405060796",
		TenantID: "303",
		Priority: 1,
		Status:   mioty.DLQueueStatusPending,
		QueID:    900001,
	}, testDownlinkLifetime)
	require.ErrorIs(t, err, storage.ErrInvalidInput,
		"enqueue without an organization must be rejected by the repository")

	_, err = db.conn.Exec(`
		INSERT INTO downlink_queue (ep_eui, tenant_id, organization_id, payload, priority, status, que_id, created_at)
		VALUES ($1, $2, NULL, $3, 1, $4, 900002, NOW())`,
		[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x96}, tenantID, []byte{0x42}, mioty.DLQueueStatusPending)
	require.Error(t, err, "a NULL organization row must violate the schema constraint")
}
