package postgres

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

// concurrentCounterWriters is how many snapshots of one session are written at
// once, each the counters of a later frame than the one before.
const concurrentCounterWriters = 32

// Operation counters persisted out of order never move backwards: the base
// station counter only rises, the service center counter only falls.
func TestUpdateOperationIDsNeverMovesCountersBackwards(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	const tenantID = int64(141)
	orgID := uuid.New()
	createTestTenant(t, db, tenantID, "TestTenant141")
	createTestOrganization(t, db, orgID, tenantID, "TestOrg141")
	createTestBaseStation(t, db, 44, 0x0102030405060744, tenantID, "TestBS-Counters")
	cleanupSessionTestData(t, db, "TestCounters%")
	defer cleanupSessionTestData(t, db, "TestCounters%")

	repo := NewBaseStationSessionRepository(db, clock.SystemClock{}, logger.Get())
	f := sessionOwnershipFixture{repo: repo, tenantID: tenantID, orgID: orgID}
	ctx := testutil.TestContext()
	session := f.create(t, 44, "TestCounters-Conn", models.EUI{0x4B, 0x43})

	require.NoError(t, repo.UpdateOperationIDs(ctx, tenantID, session.ID, 10, -10))
	require.NoError(t, repo.UpdateOperationIDs(ctx, tenantID, session.ID, 7, -4), "a stale snapshot is not an error")
	row, err := repo.GetSessionByID(ctx, tenantID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(10), row.SnBsOpId, "a stale snapshot must not lower the base station counter")
	assert.Equal(t, int64(-10), row.SnScOpId, "a stale snapshot must not raise the service center counter")

	require.NoError(t, repo.UpdateOperationIDs(ctx, tenantID, session.ID, 12, -11))
	row, err = repo.GetSessionByID(ctx, tenantID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(12), row.SnBsOpId)
	assert.Equal(t, int64(-11), row.SnScOpId)

	require.ErrorIs(t, repo.UpdateOperationIDs(ctx, tenantID, session.ID+1000, 1, -1), storage.ErrNotFound, "a missing row is still reported")
}

// A session row its base station's deletion removed reads as not found to
// every write a closing connection makes, so the caller can tell it from a
// failure.
func TestSessionWritesAfterTheStationsDeletionAreNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	const tenantID = int64(142)
	orgID := uuid.New()
	createTestTenant(t, db, tenantID, "TestTenant142")
	createTestOrganization(t, db, orgID, tenantID, "TestOrg142")
	createTestBaseStation(t, db, 45, 0x0102030405060745, tenantID, "TestBS-Deleted")
	cleanupSessionTestData(t, db, "TestDeleted%")
	defer cleanupSessionTestData(t, db, "TestDeleted%")

	repo := NewBaseStationSessionRepository(db, clock.SystemClock{}, logger.Get())
	f := sessionOwnershipFixture{repo: repo, tenantID: tenantID, orgID: orgID}
	ctx := testutil.TestContext()
	session := f.create(t, 45, "TestDeleted-Conn", models.EUI{0x4B, 0x44})
	_, err := db.Exec(`DELETE FROM basestations WHERE id = $1`, 45)
	require.NoError(t, err)

	require.ErrorIs(t, repo.TerminateSession(ctx, tenantID, session.ID), storage.ErrNotFound)
	require.ErrorIs(t, repo.UpdateOperationIDs(ctx, tenantID, session.ID, 1, -1), storage.ErrNotFound)
}

// Concurrent counter writes of one session land in any order, yet no reader
// ever sees the stored counters move backwards and the newest snapshot is what
// remains stored.
func TestConcurrentUpdateOperationIDsNeverMoveCountersBackwards(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	checkDockerAvailable(t)

	db := setupSessionEncodingTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup

	const tenantID = int64(142)
	orgID := uuid.New()
	createTestTenant(t, db, tenantID, "TestTenant142")
	createTestOrganization(t, db, orgID, tenantID, "TestOrg142")
	createTestBaseStation(t, db, 45, 0x0102030405060745, tenantID, "TestBS-ConcurrentCounters")
	cleanupSessionTestData(t, db, "TestConcurrentCounters%")
	defer cleanupSessionTestData(t, db, "TestConcurrentCounters%")

	repo := NewBaseStationSessionRepository(db, clock.SystemClock{}, logger.Get())
	f := sessionOwnershipFixture{repo: repo, tenantID: tenantID, orgID: orgID}
	ctx := testutil.TestContext()
	session := f.create(t, 45, "TestConcurrentCounters-Conn", models.EUI{0x4B, 0x44})

	stop := make(chan struct{})
	regression := make(chan string, 1)
	var reader sync.WaitGroup
	reader.Add(1)
	go func() {
		defer reader.Done()
		var lastBs, lastSc int64
		for {
			select {
			case <-stop:
				return
			default:
			}
			row, err := repo.GetSessionByID(ctx, tenantID, session.ID)
			if err != nil {
				continue
			}
			if row.SnBsOpId < lastBs || row.SnScOpId > lastSc {
				regression <- "stored counters moved backwards"
				return
			}
			lastBs, lastSc = row.SnBsOpId, row.SnScOpId
		}
	}()

	var writers sync.WaitGroup
	for n := int64(1); n <= concurrentCounterWriters; n++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			assert.NoError(t, repo.UpdateOperationIDs(ctx, tenantID, session.ID, n, -n))
		}()
	}
	writers.Wait()
	close(stop)
	reader.Wait()
	close(regression)
	for message := range regression {
		t.Fatal(message)
	}

	row, err := repo.GetSessionByID(ctx, tenantID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(concurrentCounterWriters), row.SnBsOpId, "the newest base station counter remains stored")
	assert.Equal(t, int64(-concurrentCounterWriters), row.SnScOpId, "the newest service center counter remains stored")
}
