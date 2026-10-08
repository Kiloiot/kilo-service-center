package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kiloiot/kilo-service-center/KC-DB/storage/models"
	"github.com/Kiloiot/kilo-service-center/pkg/clock"
	"github.com/Kiloiot/kilo-service-center/pkg/logger"
	"github.com/Kiloiot/kilo-service-center/pkg/testutil"
)

const (
	certExpiryTestOwner   = int64(481)
	certExpiryTestForeign = int64(482)
	certExpiryTestBsEUI   = uint64(0x70B3D59CD0000481)
	certExpiryTestBsName  = "cert-expiry-backfill"
)

// TestBaseStationUpdateTLSCertExpiryIfBlank: another tenant writes nothing,
// a blank expiry is recorded once, and an existing expiry is never overwritten.
func TestBaseStationUpdateTLSCertExpiryIfBlank(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	db, cleanup := SetupPostgresContainer(t)
	t.Cleanup(cleanup)
	createTestTenant(t, db, certExpiryTestOwner, "CertExpiryOwner")
	createTestTenant(t, db, certExpiryTestForeign, "CertExpiryForeign")
	repo := NewBaseStationRepository(db, clock.SystemClock{}, logger.Get())
	ctx := testutil.TestContext()

	var eui models.EUI
	copy(eui[:], eui64Bytes(certExpiryTestBsEUI))
	require.NoError(t, repo.Create(ctx, &models.BaseStation{
		EUI:              eui,
		TenantID:         certExpiryTestOwner,
		Name:             certExpiryTestBsName,
		ConnectionType:   models.ConnectionTypeBSSCI,
		ServiceCenterURL: testServiceCenterURLPtr(),
	}))
	stored, err := repo.GetByEUI(ctx, certExpiryTestOwner, eui[:])
	require.NoError(t, err)
	require.Nil(t, stored.TLSCertExpiresAt)

	first := time.Date(2031, time.March, 4, 5, 6, 7, 0, time.UTC)
	later := first.AddDate(1, 0, 0)

	updated, err := repo.UpdateTLSCertExpiryIfBlank(ctx, certExpiryTestForeign, stored.ID, first)
	require.NoError(t, err)
	assert.False(t, updated, "another tenant's station is not written")

	updated, err = repo.UpdateTLSCertExpiryIfBlank(ctx, certExpiryTestOwner, stored.ID, first)
	require.NoError(t, err)
	assert.True(t, updated, "a blank expiry is recorded")

	updated, err = repo.UpdateTLSCertExpiryIfBlank(ctx, certExpiryTestOwner, stored.ID, later)
	require.NoError(t, err)
	assert.False(t, updated, "an existing expiry is not overwritten")

	reloaded, err := repo.GetByEUI(ctx, certExpiryTestOwner, eui[:])
	require.NoError(t, err)
	require.NotNil(t, reloaded.TLSCertExpiresAt)
	assert.True(t, first.Equal(*reloaded.TLSCertExpiresAt), "the first recorded expiry stays")
}
