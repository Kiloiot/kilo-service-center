package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/testsupport"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const profileChangeTestTenant = int64(722)

// An update stores the time an edit changed the station profile, a later
// update without one keeps it, and every endpoint read carries it with the
// last completed attach propagate it is compared against.
func TestEndpointUpdate_KeepsTheProfileChangeTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db := setupEndpointTestDB(t)
	defer func() { _ = db.Close() }() // #nosec G307 -- Test cleanup
	createTestTenant(t, db, profileChangeTestTenant, "TestTenant722")
	cleanupEndpointTestData(t, db, "ProfileChg%")
	defer cleanupEndpointTestData(t, db, "ProfileChg%")
	repo := NewEndPointRepository(db, testsupport.TestCipher(), clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	eui := models.EUI{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07, 0x22}
	require.NoError(t, repo.Create(ctx, &models.EndPoint{
		EUI: eui, Name: "ProfileChg-EP", TenantID: profileChangeTestTenant, EPClass: "A",
		EpStatus: "attached", Tags: make(map[string]string),
		NwkSnKey: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10},
	}))
	propagatedAt := time.Date(2026, 9, 28, 11, 0, 0, 0, time.UTC)
	_, err := db.Exec(`UPDATE endpoints SET propagated_at = $1 WHERE tenant_id = $2 AND ep_eui = $3`, propagatedAt, profileChangeTestTenant, eui[:])
	require.NoError(t, err)

	read, err := repo.GetByEUI(ctx, profileChangeTestTenant, eui[:])
	require.NoError(t, err)
	assert.Nil(t, read.ProfileChangedAt, "no edit has changed the profile")
	require.NotNil(t, read.PropagatedAt, "the lookup carries the last completed attach propagate")
	assert.True(t, propagatedAt.Equal(*read.PropagatedAt))

	changedAt := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	read.ProfileChangedAt = &changedAt
	require.NoError(t, repo.Update(ctx, read))

	read.ProfileChangedAt = nil
	read.Name = "ProfileChg-renamed"
	require.NoError(t, repo.Update(ctx, read))

	stored, err := repo.GetByEUI(ctx, profileChangeTestTenant, eui[:])
	require.NoError(t, err)
	require.NotNil(t, stored.ProfileChangedAt, "an update without a profile change keeps the recorded time")
	assert.True(t, changedAt.Equal(*stored.ProfileChangedAt))

	detail, err := repo.GetByID(ctx, stored.ID, profileChangeTestTenant)
	require.NoError(t, err)
	require.NotNil(t, detail.ProfileChangedAt)
	assert.True(t, changedAt.Equal(*detail.ProfileChangedAt))

	listed, err := repo.ListByTenantPaginated(ctx, profileChangeTestTenant, 10, 0)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.NotNil(t, listed[0].ProfileChangedAt)
	assert.True(t, changedAt.Equal(*listed[0].ProfileChangedAt))
}
