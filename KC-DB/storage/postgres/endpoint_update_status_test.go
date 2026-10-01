package postgres

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	updateStatusTestTenant   = int64(720)
	updateStatusTestAttached = "attached"
	updateStatusTestDetached = "detached"
	deleteTestOtherTenant    = int64(721)
)

// An endpoint's attachment status changes only through its transition; a
// general update, and an update that also renames the EUI, carry a status
// read before the transition and must not write it back.
func TestEndpointUpdate_NeverWritesTheAttachmentStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, db, updateStatusTestTenant, "TestTenant720")
	cleanupEndpointTestData(t, db, "StatusUpd%")
	defer cleanupEndpointTestData(t, db, "StatusUpd%")
	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	for name, update := range map[string]func(read *models.EndPoint) (models.EUI, error){
		"update": func(read *models.EndPoint) (models.EUI, error) {
			return read.EUI, repo.Update(ctx, read)
		},
		"update with a new EUI": func(read *models.EndPoint) (models.EUI, error) {
			oldEUI := read.EUI
			read.EUI[7]++
			_, err := repo.UpdateWithEUI(ctx, updateStatusTestTenant, oldEUI[:], read)
			return read.EUI, err
		},
	} {
		t.Run(name, func(t *testing.T) {
			cleanupEndpointTestData(t, db, "StatusUpd%")
			eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x20}
			require.NoError(t, repo.Create(ctx, &models.EndPoint{
				EUI: eui, Name: "StatusUpd-EP", TenantID: updateStatusTestTenant, EPClass: "A",
				EpStatus: updateStatusTestDetached, Tags: make(map[string]string),
				NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
			}))
			read, err := repo.GetByEUI(ctx, updateStatusTestTenant, eui[:])
			require.NoError(t, err)
			require.Equal(t, updateStatusTestDetached, read.EpStatus)

			changed, err := repo.TransitionEndpointStatus(ctx, updateStatusTestTenant, read.ID, updateStatusTestAttached)
			require.NoError(t, err)
			require.True(t, changed)

			read.Name = "StatusUpd-renamed"
			storedEUI, err := update(read)
			require.NoError(t, err)

			stored, err := repo.GetByEUI(ctx, updateStatusTestTenant, storedEUI[:])
			require.NoError(t, err)
			assert.Equal(t, "StatusUpd-renamed", stored.Name, "the update is applied")
			assert.Equal(t, updateStatusTestAttached, stored.EpStatus, "the update does not revert the attachment")
		})
	}
}

// A delete removes only the requesting tenant's endpoint and returns its id,
// which the audit record names; another tenant's delete of the EUI finds
// nothing.
func TestEndpointDeleteByTenant_ReturnsTheRemovedID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, db, updateStatusTestTenant, "TestTenant720")
	createTestTenant(t, db, deleteTestOtherTenant, "TestTenant721")
	cleanupEndpointTestData(t, db, "StatusUpd%")
	defer cleanupEndpointTestData(t, db, "StatusUpd%")
	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()
	eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x21}
	endpoint := &models.EndPoint{
		EUI: eui, Name: "StatusUpd-delete", TenantID: updateStatusTestTenant, EPClass: "A",
		EpStatus: updateStatusTestDetached, Tags: make(map[string]string),
		NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
	}
	require.NoError(t, repo.Create(ctx, endpoint))

	_, err := repo.DeleteByTenant(ctx, deleteTestOtherTenant, eui[:])
	require.ErrorIs(t, err, storage.ErrNotFound, "another tenant's delete finds nothing")

	removedID, err := repo.DeleteByTenant(ctx, updateStatusTestTenant, eui[:])
	require.NoError(t, err)
	assert.Equal(t, endpoint.ID, removedID)
	_, err = repo.GetByEUI(ctx, updateStatusTestTenant, eui[:])
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

// The status change time is the repository clock's time of a transition that
// changed the status; a transition that changed nothing leaves it. The tenant's
// endpoints detached after a moment are the ones whose change came later, and
// without a moment every recorded detachment.
func TestEndpointStatusChangeIsRecordedOnlyWhenTheStatusChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, db, updateStatusTestTenant, "TestTenant720")
	cleanupEndpointTestData(t, db, "StatusUpd%")
	defer cleanupEndpointTestData(t, db, "StatusUpd%")
	decided := time.Date(2027, time.January, 15, 9, 0, 0, 0, time.UTC)
	clk := testutil.NewFakeClock(decided)
	repo := NewEndPointRepository(db, testsupport.TestCipher(), clk, logger.Get())
	ctx := testutil.TestContext()
	create := func(eui models.EUI, name string) int64 {
		endpoint := &models.EndPoint{
			EUI: eui, Name: name, TenantID: updateStatusTestTenant, EPClass: "A", Tags: make(map[string]string),
			NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
		}
		require.NoError(t, repo.Create(ctx, endpoint))
		return endpoint.ID
	}
	detachedID := create(models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x22}, "StatusUpd-detached")
	create(models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x23}, "StatusUpd-never-attached")

	for _, epStatus := range []string{updateStatusTestAttached, updateStatusTestDetached} {
		changed, err := repo.TransitionEndpointStatus(ctx, updateStatusTestTenant, detachedID, epStatus)
		require.NoError(t, err)
		require.True(t, changed)
	}
	clk.Advance(time.Hour)
	changed, err := repo.TransitionEndpointStatus(ctx, updateStatusTestTenant, detachedID, updateStatusTestDetached)
	require.NoError(t, err)
	require.False(t, changed, "the endpoint is already detached")

	all, err := repo.GetByAttachmentChangedSince(ctx, updateStatusTestTenant, updateStatusTestDetached, nil)
	require.NoError(t, err)
	require.Len(t, all, 1, "every recorded detachment, never an endpoint with no recorded change")
	assert.Equal(t, detachedID, all[0].ID)

	before := decided.Add(-time.Minute)
	later, err := repo.GetByAttachmentChangedSince(ctx, updateStatusTestTenant, updateStatusTestDetached, &before)
	require.NoError(t, err)
	assert.Len(t, later, 1, "detached after a moment before the decision")

	after := decided.Add(time.Minute)
	later, err = repo.GetByAttachmentChangedSince(ctx, updateStatusTestTenant, updateStatusTestDetached, &after)
	require.NoError(t, err)
	assert.Empty(t, later, "the transition that changed nothing did not move the decision time")
}

// A restated status records the repository clock's time even when the status
// stays, and reports a change only when it moved: of concurrent identical
// restates exactly one reports it.
func TestEndpointRestateRecordsTheTimeEvenWhenTheStatusStays(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, db, updateStatusTestTenant, "TestTenant720")
	cleanupEndpointTestData(t, db, "StatusUpd%")
	defer cleanupEndpointTestData(t, db, "StatusUpd%")
	decided := time.Date(2027, time.February, 1, 9, 0, 0, 0, time.UTC)
	clk := testutil.NewFakeClock(decided)
	repo := NewEndPointRepository(db, testsupport.TestCipher(), clk, logger.Get())
	ctx := testutil.TestContext()
	endpoint := &models.EndPoint{
		EUI: models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x24}, Name: "StatusUpd-restated",
		TenantID: updateStatusTestTenant, EPClass: "A", Tags: make(map[string]string),
		NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
	}
	require.NoError(t, repo.Create(ctx, endpoint))

	const concurrentRestates = 4
	changes := make(chan bool, concurrentRestates)
	var wg sync.WaitGroup
	for range concurrentRestates {
		wg.Add(1)
		go func() {
			defer wg.Done()
			changed, err := repo.RestateEndpointStatus(ctx, updateStatusTestTenant, endpoint.ID, updateStatusTestAttached)
			assert.NoError(t, err)
			changes <- changed
		}()
	}
	wg.Wait()
	close(changes)
	var reported int
	for changed := range changes {
		if changed {
			reported++
		}
	}
	assert.Equal(t, 1, reported, "one of the identical restates changed the status")

	clk.Advance(time.Hour)
	changed, err := repo.RestateEndpointStatus(ctx, updateStatusTestTenant, endpoint.ID, updateStatusTestAttached)
	require.NoError(t, err)
	assert.False(t, changed, "the endpoint is attached already")

	afterFirst := decided.Add(time.Minute)
	restated, err := repo.GetByAttachmentChangedSince(ctx, updateStatusTestTenant, updateStatusTestAttached, &afterFirst)
	require.NoError(t, err)
	require.Len(t, restated, 1, "the restate that kept the status recorded its time")
	assert.Equal(t, endpoint.ID, restated[0].ID)

	missing, err := repo.RestateEndpointStatus(ctx, updateStatusTestTenant, endpoint.ID+1_000_000, updateStatusTestAttached)
	require.NoError(t, err)
	assert.False(t, missing, "no endpoint, no change")
}
