package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/mioty"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
)

// Identities of the result holder tests: the owner tenant's endpoint, two of
// its stations and a station of another tenant the endpoint roams into.
const (
	holderOwnerTenant   int64  = 100
	holderStationTenant int64  = 200
	holderEndpointEUI   uint64 = 0x70B3D59CD0000441
	holderStationA      uint64 = 0x70B3D59CD0000A41
	holderStationB      uint64 = 0x70B3D59CD0000B41
	holderRoamer        uint64 = 0x70B3D59CD0000C41
	// noHolder is the station of a row no base station holds.
	noHolder uint64 = 0
)

// resultHolderFixture is the owner tenant's downlink queue in a fresh database.
type resultHolderFixture struct {
	sqlxDB *sqlx.DB
	repo   *MIOTYDownlinkRepository
	org    uuid.UUID
}

func newResultHolderFixture(t *testing.T) *resultHolderFixture {
	t.Helper()
	sqlxDB, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	createTestTenant(t, sqlxDB, holderOwnerTenant, "HolderOwnerTenant")
	createTestTenant(t, sqlxDB, holderStationTenant, "HolderStationTenant")
	insertEndpoint(t, sqlxDB, EndpointInsertParams{EpEUI: holderEndpointEUI, Name: "TestEndpoint-ResultHolder", TenantID: holderOwnerTenant})
	logger.Initialize("error", "json")
	db := &DB{clock: clock.SystemClock{}, conn: sqlxDB.DB, sqlxDB: sqlxDB, log: logger.Get()}
	return &resultHolderFixture{sqlxDB: sqlxDB, repo: NewRepositories(db).Downlinks, org: uuid.New()}
}

// seed queues a downlink of the owner tenant held by the station in status.
func (f *resultHolderFixture) seed(t *testing.T, status mioty.DLQueueStatus, station uint64) int64 {
	t.Helper()
	org := f.org.String()
	return insertDownlink(t, &DB{conn: f.sqlxDB.DB, sqlxDB: f.sqlxDB}, DownlinkInsertParams{
		EpEUI: holderEndpointEUI, TenantID: holderOwnerTenant, Status: status, BsEUI: station, OrganizationID: &org,
	})
}

// report records a sent result for the downlink from the station.
func (f *resultHolderFixture) report(t *testing.T, station uint64, queID int64) error {
	t.Helper()
	txTime := time.Now().UnixNano()
	packetCnt := uint32(7)
	_, err := f.repo.UpdateDownlinkResult(t.Context(), holderOwnerTenant, station, &mioty.DLDataResult{
		EpEui: holderEndpointEUI, QueId: uint64(queID), Result: mioty.DLDataResultSent, TxTime: &txTime, PacketCnt: &packetCnt,
	})
	return err
}

// requireRow asserts the downlink's status, holder and recorded result.
func (f *resultHolderFixture) requireRow(t *testing.T, queID int64, status mioty.DLQueueStatus, station uint64, result sql.NullString) {
	t.Helper()
	var row struct {
		Status mioty.DLQueueStatus `db:"status"`
		BsEUI  []byte              `db:"bs_eui"`
		Result sql.NullString      `db:"result"`
	}
	require.NoError(t, f.sqlxDB.Get(&row, `SELECT status, bs_eui, result FROM downlink_queue WHERE que_id = $1`, queID))
	assert.Equal(t, status, row.Status, "status of downlink %d", queID)
	var holder []byte
	if station != noHolder {
		holder = mioty.EUI64Bytes(station)
	}
	assert.Equal(t, holder, row.BsEUI, "holder of downlink %d", queID)
	assert.Equal(t, result, row.Result, "result of downlink %d", queID)
}

var (
	noResult   = sql.NullString{}
	sentResult = sql.NullString{String: mioty.DLDataResultSent, Valid: true}
)

// TestDownlinkResultOnlyFromHolder pins that only the station holding a
// downlink, reserved or queued, records its result (BSSCI §3.14): another
// station reporting the downlink's own ids, a station the downlink was taken
// from, and a result for a downlink back in the queue leave its state and
// holder alone and are answered like an unknown queue id.
func TestDownlinkResultOnlyFromHolder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	f := newResultHolderFixture(t)

	t.Run("another station with the downlink's ids is refused", func(t *testing.T) {
		queID := f.seed(t, mioty.DLQueueStatusQueued, holderStationA)
		require.ErrorIs(t, f.report(t, holderStationB, queID), storage.ErrDownlinkNotFound)
		f.requireRow(t, queID, mioty.DLQueueStatusQueued, holderStationA, noResult)

		require.NoError(t, f.report(t, holderStationA, queID))
		f.requireRow(t, queID, mioty.DLQueueStatusTransmitted, holderStationA, sentResult)
	})

	t.Run("a late result after the downlink moved to another station is refused", func(t *testing.T) {
		queID := f.seed(t, mioty.DLQueueStatusQueued, holderStationA)
		_, err := f.repo.ReleaseStationQueue(t.Context(), holderStationA)
		require.NoError(t, err)
		_, err = f.repo.ReservePendingDownlinkByQueueID(t.Context(), holderOwnerTenant, f.org, uint64(queID), mioty.EUI64Bytes(holderEndpointEUI), holderStationB)
		require.NoError(t, err)
		require.NoError(t, f.repo.MarkReservedAsQueued(t.Context(), uint64(queID), holderOwnerTenant, holderStationB, time.Now().UnixNano(), nil, &f.org))

		require.ErrorIs(t, f.report(t, holderStationA, queID), storage.ErrDownlinkNotFound)
		f.requireRow(t, queID, mioty.DLQueueStatusQueued, holderStationB, noResult)

		require.NoError(t, f.report(t, holderStationB, queID))
		f.requireRow(t, queID, mioty.DLQueueStatusTransmitted, holderStationB, sentResult)

		require.ErrorIs(t, f.report(t, holderStationA, queID), storage.ErrDownlinkFinished,
			"a finished downlink keeps its outcome whichever station reports it again")
		f.requireRow(t, queID, mioty.DLQueueStatusTransmitted, holderStationB, sentResult)
	})

	t.Run("a result for a downlink back in the queue is refused", func(t *testing.T) {
		queID := f.seed(t, mioty.DLQueueStatusReserved, holderStationA)
		_, err := f.repo.ReleaseStationReservations(t.Context(), holderStationA, nil)
		require.NoError(t, err)

		require.ErrorIs(t, f.report(t, holderStationA, queID), storage.ErrDownlinkNotFound)
		f.requireRow(t, queID, mioty.DLQueueStatusPending, noHolder, noResult)
	})

	t.Run("a roaming holder of another tenant records the owner's result", func(t *testing.T) {
		_, err := f.sqlxDB.Exec(`INSERT INTO basestations (tenant_id, bs_eui, name) VALUES ($1, $2, $3)`,
			holderStationTenant, mioty.EUI64Bytes(holderRoamer), "HolderRoamer")
		require.NoError(t, err)
		queID := f.seed(t, mioty.DLQueueStatusQueued, holderRoamer)

		require.NoError(t, f.report(t, holderRoamer, queID))
		f.requireRow(t, queID, mioty.DLQueueStatusTransmitted, holderRoamer, sentResult)
	})
}
